package db

import (
	"time"

	"dodevops-api/api/system/model"
	"dodevops-api/common/util"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// migrateManagementNavigation keeps database-backed menus aligned with the
// frontend routes. Existing IDs, button permissions and role links are retained.
func migrateManagementNavigation(database *gorm.DB) error {
	if !database.Migrator().HasTable(&model.SysMenu{}) {
		return nil
	}
	return database.Transaction(func(tx *gorm.DB) error {
		var menus []model.SysMenu
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Order("id ASC").Find(&menus).Error; err != nil {
			return err
		}
		var systemID, containerID, databaseID uint
		for _, menu := range menus {
			if menu.ParentId != 0 || menu.MenuType != 1 {
				continue
			}
			switch menu.MenuName {
			case "系统管理", "基础管理":
				systemID = menu.ID
			case "容器管理":
				containerID = menu.ID
			case "数据库管理":
				databaseID = menu.ID
			}
		}
		if systemID != 0 {
			if err := tx.Model(&model.SysMenu{}).Where("id = ?", systemID).Update("menu_name", "基础管理").Error; err != nil {
				return err
			}
		}
		if databaseID == 0 {
			root := model.SysMenu{MenuName: "数据库管理", Url: "database", Icon: "Coin", MenuType: 1, MenuStatus: 2, Sort: 11, CreateTime: util.HTime{Time: time.Now()}}
			if err := tx.Create(&root).Error; err != nil {
				return err
			}
			databaseID = root.ID
		}
		// Retain historical role links when retiring pages now embedded in cluster details.
		if err := tx.Model(&model.SysMenu{}).Where("url IN ? AND menu_type = ?", []string{"database/diagnostics", "/database/diagnostics", "database/configurations", "/database/configurations"}, 2).Update("menu_status", 1).Error; err != nil {
			return err
		}
		changes := []struct {
			parent          uint
			name, url, icon string
			old             []string
		}{
			{databaseID, "数据源管理", "database/sources", "Coin", []string{"cmdb/db"}},
			{databaseID, "集群管理", "database/clusters", "Connection", nil},
			{databaseID, "存储管理", "database/storage", "Files", nil},
			{databaseID, "备份恢复", "database/backups", "RefreshLeft", nil},
			{databaseID, "操作审计", "database/audit", "Document", nil},

			{containerID, "Docker管理", "k8s/docker", "Box", []string{"cmdb/docker"}},
			{systemID, "API Key", "system/api-keys", "Key", []string{"system/role-permission"}},
			{systemID, "授权管理", "system/authorization", "Lock", []string{"cmdb/asset-permission", "system/asset-permission", "system/k8s-permission", "k8s/permission", "cmdb/user-group", "cmdb/credential-permission"}},
		}
		for changeIndex, change := range changes {
			if change.parent == 0 {
				continue
			}
			urls := []string{change.url, "/" + change.url}
			for _, url := range change.old {
				urls = append(urls, url, "/"+url)
			}
			var candidates []model.SysMenu
			if err := tx.Where("url IN ? AND menu_type = ?", urls, 2).Order("id ASC").Find(&candidates).Error; err != nil {
				return err
			}
			var target model.SysMenu
			var oldIDs []uint
			for _, candidate := range candidates {
				oldIDs = append(oldIDs, candidate.ID)
				if target.ID == 0 || candidate.Url == change.url || candidate.Url == "/"+change.url {
					target = candidate
				}
			}
			if target.ID == 0 {
				target = model.SysMenu{ParentId: change.parent, MenuName: change.name, Url: change.url, Icon: change.icon, MenuType: 2, MenuStatus: 2, Sort: uint(changeIndex + 1), CreateTime: util.HTime{Time: time.Now()}}
				if change.url == "k8s/docker" {
					target.Value = "docker:read"
				}
				if err := tx.Create(&target).Error; err != nil {
					return err
				}
			} else if err := tx.Model(&model.SysMenu{}).Where("id = ?", target.ID).Updates(map[string]interface{}{"parent_id": change.parent, "menu_name": change.name, "url": change.url, "icon": change.icon}).Error; err != nil {
				return err
			}
			if len(oldIDs) > 0 {
				if err := tx.Model(&model.SysMenu{}).Where("id IN ? AND id <> ?", oldIDs, target.ID).Update("menu_status", 1).Error; err != nil {
					return err
				}
			}
			if !tx.Migrator().HasTable(&model.SysRoleMenu{}) {
				continue
			}
			var roleIDs []uint
			if len(oldIDs) > 0 {
				if err := tx.Model(&model.SysRoleMenu{}).Where("menu_id IN ?", oldIDs).Distinct().Pluck("role_id", &roleIDs).Error; err != nil {
					return err
				}
			}
			if tx.Migrator().HasTable("sys_role") {
				var admins []uint
				if err := tx.Table("sys_role").Where("role_key = ? AND status = ?", "admin", 1).Pluck("id", &admins).Error; err != nil {
					return err
				}
				roleIDs = append(roleIDs, admins...)
			}
			for _, roleID := range roleIDs {
				for _, menuID := range []uint{target.ID, change.parent} {
					link := model.SysRoleMenu{RoleId: roleID, MenuId: menuID}
					if err := tx.Where("role_id = ? AND menu_id = ?", roleID, menuID).FirstOrCreate(&link).Error; err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
}
