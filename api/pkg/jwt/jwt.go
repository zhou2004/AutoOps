// jwt工具类（生产token/解析token/获取用户信息）
package jwt

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"time"

	"dodevops-api/api/system/model"
	"dodevops-api/common"
	"dodevops-api/common/constant"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

type userStdClaims struct {
	model.JwtAdmin
	jwt.StandardClaims
}

// token过期时间
const TokenExpireDuration = time.Hour * 24

// token密钥
var Secret = signingSecret()

func signingSecret() []byte {
	if configured := os.Getenv("AUTOOPS_JWT_SECRET"); configured != "" {
		if len(configured) < 32 {
			panic("AUTOOPS_JWT_SECRET must contain at least 32 bytes")
		}
		return []byte(configured)
	}
	// Development fallback: unpredictable per-process secret. Production replicas share an environment secret.
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		panic("cannot initialize signing secret")
	}
	return secret
}

var (
	ErrAbsent  = "token absent"  // 令牌不存在
	ErrInvalid = "token invalid" //令牌无效
)

// 根据用户信息生成token
func GenerateTokenByAdmin(admin model.SysAdmin) (string, error) {
	var jwtAdmin = model.JwtAdmin{
		ID:       admin.ID,
		Username: admin.Username,
		Nickname: admin.Nickname,
		Icon:     admin.Icon,
		Email:    admin.Email,
		Phone:    admin.Phone,
		Note:     admin.Note,
	}
	c := userStdClaims{
		jwtAdmin,
		jwt.StandardClaims{
			ExpiresAt: time.Now().Add(TokenExpireDuration).Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    "admin", //签发人
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, c)
	tokenString, err := token.SignedString(Secret)

	if err == nil {
		// 将token保存到redis，设置过期时间
		rc := common.GetRedisClient()
		if rc != nil {
			rc.Set(context.Background(), "token:"+tokenString, jwtAdmin.Username, TokenExpireDuration)
		}
	}

	return tokenString, err
}

// ValidateToken 解析JWT
func ValidateToken(tokenString string) (*model.JwtAdmin, error) {
	if tokenString == "" {
		return nil, errors.New(ErrAbsent)
	}
	claims := userStdClaims{}
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return Secret, nil
	})
	if err != nil || token == nil || !token.Valid || claims.ExpiresAt == 0 || claims.ID == 0 {
		return nil, errors.New(ErrInvalid)
	}

	// 验证Redis中的Token是否过期
	rc := common.GetRedisClient()
	if rc != nil {
		err = rc.Get(context.Background(), "token:"+tokenString).Err()
		if err != nil {
			return nil, errors.New("token已在redis中过期")
		}
	}

	return &claims.JwtAdmin, err
}

// 返回id
func GetAdminId(c *gin.Context) (uint, error) {
	u, exist := c.Get(constant.ContextKeyUserObj)
	if !exist {
		return 0, errors.New("无法获取用户id")
	}
	admin, ok := u.(*model.JwtAdmin)
	if ok {
		return admin.ID, nil
	}
	return 0, errors.New("无法转换为id结构")
}

// 返回用户名
func GetAdminName(c *gin.Context) (string, error) {
	u, exist := c.Get(constant.ContextKeyUserObj)
	if !exist {
		return "", errors.New("无法获取用户名")
	}
	admin, ok := u.(*model.JwtAdmin)
	if ok {
		return admin.Username, nil
	}
	return "", errors.New("无法转换为api名称")
}

// 返回admin信息
func GetAdmin(c *gin.Context) (*model.JwtAdmin, error) {
	u, exist := c.Get(constant.ContextKeyUserObj)
	if !exist {
		return nil, errors.New("can't get api")
	}
	admin, ok := u.(*model.JwtAdmin)
	if ok {
		return admin, nil
	}
	return nil, errors.New("无法转换为api结构")
}
