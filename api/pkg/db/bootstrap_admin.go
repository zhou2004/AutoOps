package db

import (
	"dodevops-api/common/util"
	"fmt"
	"gorm.io/gorm"
	"os"
)

// Apply only to the untouched bootstrap account; restarts never reset user passwords.
func initializeAdminPassword(database *gorm.DB) error {
	password := os.Getenv("AUTOOPS_INITIAL_ADMIN_PASSWORD")
	if password == "" {
		return nil
	}
	if len(password) < 10 || len(password) > 72 {
		return fmt.Errorf("AUTOOPS_INITIAL_ADMIN_PASSWORD must be 10–72 bytes")
	}
	hash, err := util.HashPassword(password)
	if err != nil {
		return err
	}
	return database.Table("sys_admin").Where("username = ? AND password = ?", "admin", util.EncryptionMd5("123456")).Update("password", hash).Error
}
