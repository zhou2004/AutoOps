package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

var keyMu sync.Mutex

func encryptionKey() ([]byte, error) {
	keyMu.Lock()
	defer keyMu.Unlock()
	if value := os.Getenv("AIOPS_ENCRYPTION_KEY"); value != "" {
		key, err := base64.StdEncoding.DecodeString(value)
		if err != nil || len(key) != 32 {
			return nil, errors.New("AIOPS_ENCRYPTION_KEY must be a base64-encoded 32-byte key")
		}
		return key, nil
	}
	path := os.Getenv("AIOPS_KEY_FILE")
	if path == "" {
		path = "data/aiops.key"
	}
	key, err := os.ReadFile(path)
	if err == nil {
		if len(key) != 32 {
			return nil, errors.New("invalid AI encryption key file")
		}
		return key, nil
	}
	if !os.IsNotExist(err) {
		return nil, errors.New("cannot read AI encryption key file")
	}
	if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, errors.New("cannot create AI key directory")
	}
	key = make([]byte, 32)
	if _, err = rand.Read(key); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if os.IsExist(err) {
		key, err = os.ReadFile(path)
		if err == nil && len(key) == 32 {
			return key, nil
		}
		return nil, errors.New("cannot read shared AI encryption key")
	}
	if err != nil {
		return nil, errors.New("cannot create AI encryption key file")
	}
	_, err = file.Write(key)
	closeErr := file.Close()
	if err != nil || closeErr != nil {
		return nil, errors.New("cannot save AI encryption key")
	}
	return key, nil
}
func sealSecret(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	key, err := encryptionKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(value), []byte("autoops-aiops-v1"))), nil
}
func openSecret(value string) (string, error) {
	if value == "" {
		return "", nil
	}
	key, err := encryptionKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(value)
	if err != nil || len(raw) < gcm.NonceSize() {
		return "", errors.New("invalid encrypted credential")
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], []byte("autoops-aiops-v1"))
	if err != nil {
		return "", errors.New("cannot decrypt credential; check AI encryption key")
	}
	return string(plain), nil
}
