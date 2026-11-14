package db

import (
	"path/filepath"
	"testing"

	"dodevops-api/common/util"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestBootstrapPasswordDoesNotOverwriteAnExistingPassword(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "bootstrap.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := database.DB()
	t.Cleanup(func() { raw.Close() })
	if err := database.Exec("CREATE TABLE sys_admin (username TEXT, password TEXT)").Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Exec("INSERT INTO sys_admin VALUES (?, ?)", "admin", util.EncryptionMd5("123456")).Error; err != nil {
		t.Fatal(err)
	}
	t.Setenv("AUTOOPS_INITIAL_ADMIN_PASSWORD", "fresh-install-secret")
	if err := initializeAdminPassword(database); err != nil {
		t.Fatal(err)
	}
	var first string
	database.Table("sys_admin").Select("password").Where("username = ?", "admin").Scan(&first)
	if !util.VerifyPassword(first, "fresh-install-secret") {
		t.Fatal("initial administrator must use the supplied password")
	}
	t.Setenv("AUTOOPS_INITIAL_ADMIN_PASSWORD", "another-install-secret")
	if err := initializeAdminPassword(database); err != nil {
		t.Fatal(err)
	}
	var second string
	database.Table("sys_admin").Select("password").Where("username = ?", "admin").Scan(&second)
	if first != second {
		t.Fatal("restarting with a different initial password must not reset an existing account")
	}
}
