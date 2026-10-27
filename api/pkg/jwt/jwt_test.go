package jwt

import (
	"dodevops-api/api/system/model"
	"encoding/json"
	jwtlib "github.com/dgrijalva/jwt-go"
	"strings"
	"testing"
	"time"
)

func TestSignedSessionsRequireExpiryAndExpectedAlgorithm(t *testing.T) {
	if string(Secret) == "dodevops-api" || len(Secret) < 32 {
		t.Fatal("predictable signing key")
	}
	token, err := GenerateTokenByAdmin(model.SysAdmin{ID: 42, Username: "operator"})
	if err != nil {
		t.Fatal(err)
	}
	if user, err := ValidateToken(token); err != nil || user.ID != 42 {
		t.Fatalf("valid token rejected: %v", err)
	}
	for _, tc := range []struct {
		expiry int64
		method jwtlib.SigningMethod
		key    []byte
	}{{0, jwtlib.SigningMethodHS256, Secret}, {time.Now().Add(-time.Minute).Unix(), jwtlib.SigningMethodHS256, Secret}, {time.Now().Add(time.Hour).Unix(), jwtlib.SigningMethodHS512, Secret}, {time.Now().Add(time.Hour).Unix(), jwtlib.SigningMethodHS256, []byte("dodevops-api")}} {
		c := userStdClaims{JwtAdmin: model.JwtAdmin{ID: 42}, StandardClaims: jwtlib.StandardClaims{ExpiresAt: tc.expiry}}
		s, err := jwtlib.NewWithClaims(tc.method, c).SignedString(tc.key)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = ValidateToken(s); err == nil {
			t.Fatal("invalid session accepted")
		}
	}
	// API hashes and signing material are never part of the identity response.
	b, _ := json.Marshal(model.JwtAdmin{ID: 42})
	if strings.Contains(string(b), string(Secret)) {
		t.Fatal("signing secret exposed")
	}
}
