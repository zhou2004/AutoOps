package util

import (
	"crypto/subtle"
	"golang.org/x/crypto/bcrypt"
	"strings"
)

func HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(b), err
}
func VerifyPassword(hash, password string) bool {
	if strings.HasPrefix(hash, "$2") {
		return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
	}
	// Existing installations keep working until the next password change.
	return hash != "" && subtle.ConstantTimeCompare([]byte(hash), []byte(EncryptionMd5(password))) == 1
}
