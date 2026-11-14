package websecret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
)

func secretKey() ([]byte, error) {
	if raw := os.Getenv("WEB_ENCRYPTION_KEY"); raw != "" {
		k, e := base64.StdEncoding.DecodeString(raw)
		if e == nil && len(k) == 32 {
			return k, nil
		}
		return nil, errors.New("WEB_ENCRYPTION_KEY must be a base64 encoded 32 byte key")
	}
	path := os.Getenv("WEB_KEY_FILE")
	if path == "" {
		path = "data/web.key"
	}
	k, e := os.ReadFile(path)
	if e != nil || len(k) != 32 {
		return nil, errors.New("Web secret key unavailable; configure WEB_KEY_FILE or WEB_ENCRYPTION_KEY")
	}
	return k, nil
}

// Initialize is an explicit first-install command, never an API startup fallback.
// A complete key is linked into place atomically and an existing key is preserved.
func Initialize() (string, bool, error) {
	if os.Getenv("WEB_ENCRYPTION_KEY") != "" {
		return "WEB_ENCRYPTION_KEY", false, Ready()
	}
	path := os.Getenv("WEB_KEY_FILE")
	if path == "" {
		path = "data/web.key"
	}
	if _, e := os.Stat(path); e == nil {
		return path, false, Ready()
	} else if !os.IsNotExist(e) {
		return path, false, e
	}
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return path, false, e
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".web-key-*")
	if e != nil {
		return path, false, e
	}
	defer os.Remove(f.Name())
	defer f.Close()
	key := make([]byte, 32)
	if _, e = rand.Read(key); e != nil {
		return path, false, e
	}
	if _, e = f.Write(key); e != nil {
		return path, false, e
	}
	if e = f.Sync(); e != nil {
		return path, false, e
	}
	if e = f.Close(); e != nil {
		return path, false, e
	}
	if e = os.Link(f.Name(), path); e != nil {
		if os.IsExist(e) {
			return path, false, Ready()
		}
		return path, false, e
	}
	return path, true, nil
}
func Seal(value string) (string, error) {
	k, e := secretKey()
	if e != nil {
		return "", e
	}
	b, e := aes.NewCipher(k)
	if e != nil {
		return "", e
	}
	g, e := cipher.NewGCM(b)
	if e != nil {
		return "", e
	}
	n := make([]byte, g.NonceSize())
	if _, e = rand.Read(n); e != nil {
		return "", e
	}
	return base64.StdEncoding.EncodeToString(g.Seal(n, n, []byte(value), []byte("autoops-web-v1"))), nil
}
func Unseal(value string) (string, error) {
	k, e := secretKey()
	if e != nil {
		return "", e
	}
	b, e := aes.NewCipher(k)
	if e != nil {
		return "", e
	}
	g, e := cipher.NewGCM(b)
	if e != nil {
		return "", e
	}
	v, e := base64.StdEncoding.DecodeString(value)
	if e != nil || len(v) < g.NonceSize() {
		return "", errors.New("invalid encrypted Web secret")
	}
	v, e = g.Open(nil, v[:g.NonceSize()], v[g.NonceSize():], []byte("autoops-web-v1"))
	return string(v), e
}

// Ready checks configuration without exposing the master key.
func Ready() error { _, e := secretKey(); return e }
