package websecret

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestInitializePreservesExistingKeyAndEncryptedData(t *testing.T) {
	t.Setenv("WEB_ENCRYPTION_KEY", "")
	path := filepath.Join(t.TempDir(), "state", "web.key")
	t.Setenv("WEB_KEY_FILE", path)
	if _, created, err := Initialize(); err != nil || !created {
		t.Fatal(created, err)
	}
	key, _ := os.ReadFile(path)
	info, _ := os.Stat(path)
	if len(key) != 32 || info.Mode().Perm() != 0600 {
		t.Fatal("invalid key length or permissions")
	}
	ciphertext, err := Seal("test private material")
	if err != nil {
		t.Fatal(err)
	}
	if _, created, err := Initialize(); err != nil || created {
		t.Fatal(created, err)
	}
	again, _ := os.ReadFile(path)
	if !bytes.Equal(key, again) {
		t.Fatal("existing master key changed")
	}
	if value, err := Unseal(ciphertext); err != nil || value != "test private material" {
		t.Fatal("existing ciphertext became unreadable", err)
	}
}

func TestInitializeNeverReplacesInvalidKey(t *testing.T) {
	t.Setenv("WEB_ENCRYPTION_KEY", "")
	path := filepath.Join(t.TempDir(), "web.key")
	t.Setenv("WEB_KEY_FILE", path)
	if err := os.WriteFile(path, []byte("damaged"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, created, err := Initialize(); err == nil || created {
		t.Fatal("damaged key was silently replaced")
	}
	value, _ := os.ReadFile(path)
	if string(value) != "damaged" {
		t.Fatal("key changed")
	}
	t.Setenv("WEB_ENCRYPTION_KEY", "invalid")
	if _, _, err := Initialize(); err == nil {
		t.Fatal("invalid environment key accepted")
	}
}
