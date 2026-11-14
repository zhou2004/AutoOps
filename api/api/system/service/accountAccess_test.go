package service

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"dodevops-api/api/system/model"
	"dodevops-api/common/util"
	dbpkg "dodevops-api/pkg/db"
	"dodevops-api/pkg/jwt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func accountTestService(t *testing.T) (*AccountAccessService, *string) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "account.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&model.SysAdmin{}, &model.AccountChallenge{}, &model.AccountRegistration{}, &model.AccountRateLimit{}, &model.AccountSessionVersion{}); err != nil {
		t.Fatal(err)
	}
	sent := ""
	now := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)
	return &AccountAccessService{DB: db, Now: func() time.Time { return now }, Send: func(_ context.Context, email, purpose, code string) error { sent = code; return nil }}, &sent
}
func TestAccountRegistrationAndPasswordRecovery(t *testing.T) {
	s, code := accountTestService(t)
	ctx := context.Background()
	id, err := s.SendCode(ctx, "new@example.com", "register", "192.0.2.1")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Complete(ctx, id, "new@example.com", "reset", *code, "operator", "a-long-password"); !errors.Is(err, ErrAccountCode) {
		t.Fatal("registration code accepted for reset")
	}
	if err = s.Complete(ctx, id, "other@example.com", "register", *code, "operator", "a-long-password"); !errors.Is(err, ErrAccountCode) {
		t.Fatal("code accepted for another mailbox")
	}
	if err = s.Complete(ctx, id, "new@example.com", "register", *code, "operator", "a-long-password"); err != nil {
		t.Fatal(err)
	}
	var user model.SysAdmin
	s.DB.First(&user, "username = ?", "operator")
	if user.Status != 2 || !strings.HasPrefix(user.Password, "$2") || !util.VerifyPassword(user.Password, "a-long-password") {
		t.Fatal("registered user must be pending with a hashed password")
	}
	if err = s.Complete(ctx, id, "new@example.com", "register", *code, "someone", "a-long-password"); !errors.Is(err, ErrAccountCode) {
		t.Fatal("code replay accepted")
	}
	s.Now = func() time.Time { return time.Date(2026, 11, 14, 1, 0, 0, 0, time.UTC) }
	oldDB := dbpkg.Db
	dbpkg.Db = s.DB
	defer func() { dbpkg.Db = oldDB }()
	oldToken, err := jwt.GenerateTokenByAdmin(user)
	if err != nil {
		t.Fatal(err)
	}
	id, err = s.SendCode(ctx, "new@example.com", "reset", "192.0.2.2")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Complete(ctx, id, "new@example.com", "reset", *code, "", "a-different-password"); err != nil {
		t.Fatal(err)
	}
	if _, err = jwt.ValidateToken(oldToken); err == nil {
		t.Fatal("password reset did not revoke old session")
	}
	s.DB.First(&user, user.ID)
	if user.Status != 2 || !util.VerifyPassword(user.Password, "a-different-password") {
		t.Fatal("reset enabled user or failed to update password")
	}
}
func TestAccountCodeLimitsExpiryAndSendFailure(t *testing.T) {
	s, code := accountTestService(t)
	ctx := context.Background()
	id, err := s.SendCode(ctx, "x@example.com", "register", "192.0.2.1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.SendCode(ctx, "x@example.com", "register", "192.0.2.1"); !errors.Is(err, ErrAccountLimit) {
		t.Fatal("resend not limited")
	}
	bad := "000000"
	if bad == *code {
		bad = "111111"
	}
	for i := 0; i < 5; i++ {
		if err = s.Complete(ctx, id, "x@example.com", "register", bad, "operator", "a-long-password"); !errors.Is(err, ErrAccountCode) {
			t.Fatal(err)
		}
	}
	if err = s.Complete(ctx, id, "x@example.com", "register", *code, "operator", "a-long-password"); !errors.Is(err, ErrAccountCode) {
		t.Fatal("attempt cap bypassed")
	}
	id, err = s.SendCode(ctx, "y@example.com", "register", "192.0.2.2")
	if err != nil {
		t.Fatal(err)
	}
	s.Now = func() time.Time { return time.Date(2026, 11, 14, 0, 11, 0, 0, time.UTC) }
	if err = s.Complete(ctx, id, "y@example.com", "register", *code, "operator", "a-long-password"); !errors.Is(err, ErrAccountCode) {
		t.Fatal("expired code accepted")
	}
	s.Send = func(context.Context, string, string, string) error { return errors.New("transport unavailable") }
	if _, err = s.SendCode(ctx, "z@example.com", "register", "192.0.2.3"); err == nil {
		t.Fatal("failed mail accepted")
	}
	var n int64
	s.DB.Model(&model.AccountChallenge{}).Where("email = ?", "z@example.com").Count(&n)
	if n != 0 {
		t.Fatal("failed mail left a usable code")
	}
}
func TestAccountUnknownMailboxAndDuplicateIdentity(t *testing.T) {
	s, code := accountTestService(t)
	ctx := context.Background()
	id, err := s.SendCode(ctx, "missing@example.com", "reset", "192.0.2.1")
	if err != nil || len(id) != 64 || *code != "" {
		t.Fatal("unknown mailbox leaked or was emailed")
	}
	if err := s.DB.Create(&model.SysAdmin{Username: "exists", Email: "existing@example.com", Password: "hash", CreateTime: util.HTime{Time: s.Now()}}).Error; err != nil {
		t.Fatal(err)
	}
	id, err = s.SendCode(ctx, "new@example.com", "register", "192.0.2.2")
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Complete(ctx, id, "new@example.com", "register", *code, "exists", "a-long-password"); err == nil {
		t.Fatal("duplicate username accepted")
	}
}
