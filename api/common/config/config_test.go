package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeploymentEnvironmentOverridesFile(t *testing.T) {
	previous := Config
	t.Cleanup(func() { Config = previous })
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte("db:\n  host: old-host\n  port: 3306\n  password: old-password\nredis:\n  password: old-password\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DB_HOST", "mysql")
	t.Setenv("DB_PORT", "3307")
	t.Setenv("DB_PASSWORD", "deployment-password")
	t.Setenv("REDIS_PASSWORD", "")
	if err := LoadConfig(path); err != nil {
		t.Fatal(err)
	}
	if Config.Db.Host != "mysql" || Config.Db.Port != 3307 || Config.Db.Password != "deployment-password" || Config.Redis.Password != "" {
		t.Fatal("deployment values, including an explicitly empty Redis password, must override the file")
	}
	valid := Config
	t.Setenv("DB_PORT", "65536")
	if err := LoadConfig(path); err == nil || Config != valid {
		t.Fatal("invalid port must fail without replacing the last valid configuration")
	}
}
