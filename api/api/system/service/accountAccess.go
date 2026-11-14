package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"net/mail"
	"os"
	"regexp"
	"strings"
	"time"

	"dodevops-api/api/system/model"
	"dodevops-api/common/util"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrAccountCode = errors.New("验证码无效或已过期，请重新获取")
var ErrAccountLimit = errors.New("请求过于频繁，请稍后重试")
var ErrAccountUnavailable = errors.New("暂时无法完成请求，请联系管理员")
var accountName = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{2,31}$`)

type AccountAccessService struct {
	DB   *gorm.DB
	Send func(context.Context, string, string, string) error
	Now  func() time.Time
}

func NewAccountAccessService(db *gorm.DB) *AccountAccessService {
	return &AccountAccessService{DB: db, Send: AccountMailer(db).Send, Now: time.Now}
}
func RegistrationEnabled() bool { return os.Getenv("AUTOOPS_REGISTRATION_ENABLED") != "false" }
func NormalizeAccountEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	a, err := mail.ParseAddress(email)
	if err != nil || a.Address != email || len(email) > 254 {
		return "", errors.New("请输入有效邮箱")
	}
	return email, nil
}
func digest(s string) string { h := sha256.Sum256([]byte(s)); return hex.EncodeToString(h[:]) }
func (s *AccountAccessService) rate(ctx context.Context, key string, window time.Duration, max int) error {
	now := s.Now()
	start := now.Truncate(window)
	row := model.AccountRateLimit{ID: digest(key + start.String()), Hits: 1, ExpiresAt: start.Add(window)}
	q := s.DB.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
	if q.Error != nil {
		return ErrAccountUnavailable
	}
	if q.RowsAffected == 1 {
		return nil
	}
	q = s.DB.WithContext(ctx).Model(&model.AccountRateLimit{}).Where("id = ? AND hits < ?", row.ID, max).Update("hits", gorm.Expr("hits + 1"))
	if q.Error != nil {
		return ErrAccountUnavailable
	}
	if q.RowsAffected == 0 {
		return ErrAccountLimit
	}
	return nil
}

func (s *AccountAccessService) SendCode(ctx context.Context, email, purpose, ip string) (string, error) {
	var err error
	if email, err = NormalizeAccountEmail(email); err != nil {
		return "", err
	}
	if purpose != "register" && purpose != "reset" {
		return "", errors.New("不支持的验证用途")
	}
	if purpose == "register" && !RegistrationEnabled() {
		return "", errors.New("注册暂未开放，请联系管理员")
	}
	for _, v := range []struct {
		key    string
		window time.Duration
		max    int
	}{{"ip:" + ip, time.Hour, 20}, {"email:" + email, time.Hour, 5}, {"cooldown:" + email, time.Minute, 1}} {
		if err = s.rate(ctx, v.key, v.window, v.max); err != nil {
			return "", err
		}
	}
	var users []model.SysAdmin
	if err = s.DB.WithContext(ctx).Where("LOWER(TRIM(email)) = ?", email).Limit(2).Find(&users).Error; err != nil {
		return "", ErrAccountUnavailable
	}
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", ErrAccountUnavailable
	}
	id := hex.EncodeToString(buf)
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", ErrAccountUnavailable
	}
	code := fmt.Sprintf("%06d", n.Int64())
	// Same response for absent/ambiguous reset addresses and existing registrations.
	eligible := purpose == "register" && len(users) == 0 || purpose == "reset" && len(users) == 1
	if !eligible {
		return id, nil
	}
	var userID uint
	if purpose == "reset" {
		userID = users[0].ID
	}
	row := model.AccountChallenge{ID: id, Email: email, Purpose: purpose, UserID: userID, CodeHash: digest(id + code), ExpiresAt: s.Now().Add(10 * time.Minute)}
	if err = s.DB.WithContext(ctx).Create(&row).Error; err != nil {
		return "", ErrAccountUnavailable
	}
	if err = s.Send(ctx, email, purpose, code); err != nil {
		s.DB.WithContext(ctx).Delete(&row)
		return "", errors.New("邮件发送失败，请稍后重试或联系管理员")
	}
	s.DB.WithContext(ctx).Where("expires_at < ?", s.Now().Add(-24*time.Hour)).Delete(&model.AccountChallenge{})
	s.DB.WithContext(ctx).Where("expires_at < ?", s.Now().Add(-time.Hour)).Delete(&model.AccountRateLimit{})
	return id, nil
}

// Wrong-code attempts commit, while account mutations and successful consumption are atomic.
func (s *AccountAccessService) Complete(ctx context.Context, id, email, purpose, code, username, password string) error {
	email, err := NormalizeAccountEmail(email)
	if err != nil {
		return err
	}
	if len(password) < 10 || len(password) > 72 {
		return errors.New("密码长度需为 10–72 字节")
	}
	if len(id) != 64 || len(code) != 6 {
		return ErrAccountCode
	}
	username = strings.ToLower(strings.TrimSpace(username))
	if purpose == "register" && (!RegistrationEnabled() || !accountName.MatchString(username) || username == "admin" || username == "root") {
		return errors.New("账号需为 3–32 位字母、数字、下划线或短横线，并以字母开头")
	}
	if purpose != "register" && purpose != "reset" {
		return ErrAccountCode
	}

	invalid := false
	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row model.AccountChallenge
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, "id = ?", id).Error; err != nil {
			return ErrAccountCode
		}
		if row.Email != email || row.Purpose != purpose || row.Consumed || row.Attempts >= 5 || !s.Now().Before(row.ExpiresAt) {
			return ErrAccountCode
		}
		if subtle.ConstantTimeCompare([]byte(row.CodeHash), []byte(digest(id+code))) != 1 {
			invalid = true
			return tx.Model(&row).Update("attempts", gorm.Expr("attempts + 1")).Error
		}
		hash, err := util.HashPassword(password)
		if err != nil {
			return ErrAccountUnavailable
		}
		if purpose == "register" {
			var n int64
			if err := tx.Model(&model.SysAdmin{}).Where("LOWER(username) = ? OR LOWER(TRIM(email)) = ?", username, email).Count(&n).Error; err != nil {
				return err
			}
			if n > 0 {
				return errors.New("账号或邮箱已被使用，请登录或找回密码")
			}
			reservation := model.AccountRegistration{Username: username, Email: email}
			if err := tx.Create(&reservation).Error; err != nil {
				return errors.New("账号或邮箱已被使用，请登录或找回密码")
			}
			user := model.SysAdmin{Username: username, Nickname: username, Email: email, Password: hash, Status: 2, Note: "邮箱验证注册，等待管理员启用并分配权限", CreateTime: util.HTime{Time: s.Now()}}
			if err := tx.Create(&user).Error; err != nil {
				return err
			}
			if err := tx.Model(&reservation).Update("user_id", user.ID).Error; err != nil {
				return err
			}
		} else {
			// Never enable a disabled account and never select an account by a mutable name.
			q := tx.Model(&model.SysAdmin{}).Where("id = ? AND LOWER(TRIM(email)) = ?", row.UserID, email).Update("password", hash)
			if q.Error != nil {
				return q.Error
			}
			if q.RowsAffected != 1 {
				return ErrAccountCode
			}
			version := model.AccountSessionVersion{UserID: row.UserID, Version: 1}
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}}, DoUpdates: clause.Assignments(map[string]interface{}{"version": gorm.Expr("version + 1")})}).Create(&version).Error; err != nil {
				return err
			}
		}
		return tx.Model(&row).Updates(map[string]interface{}{"consumed": true, "code_hash": ""}).Error
	})
	if invalid {
		return ErrAccountCode
	}
	return err
}
