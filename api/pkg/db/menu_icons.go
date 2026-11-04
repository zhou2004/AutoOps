package db

import (
	"dodevops-api/api/system/model"
	"gorm.io/gorm"
	"strings"
)

// Default icon names are local Element Plus icons, matching the web registry.
var navigationIconNames = map[string]string{
	"/":                     "HomeFilled",
	"/dashboard":            "DataBoard",
	"/aiops":                "ChatDotRound",
	"/aiops/diagnose":       "FirstAidKit",
	"/aiops/models":         "Cpu",
	"/aiops/skills":         "MagicStick",
	"/aiops/workflows":      "Share",
	"/aiops/history":        "DocumentChecked",
	"/cmdb":                 "Monitor",
	"/cmdb/ecs":             "Monitor",
	"/cmdb/group":           "FolderOpened",
	"/cmdb/physical":        "Cpu",
	"/cmdb/network":         "Connection",
	"/cmdb/my-assets":       "Key",
	"/database":             "Coin",
	"/database/sources":     "Coin",
	"/database/clusters":    "Connection",
	"/database/storage":     "Files",
	"/database/backups":     "RefreshLeft",
	"/k8s":                  "Box",
	"/k8s/docker":           "Box",
	"/k8s/list":             "Connection",
	"/k8s/node":             "Cpu",
	"/k8s/namespace":        "CollectionTag",
	"/k8s/workload":         "Grid",
	"/k8s/network":          "Connection",
	"/k8s/config":           "SetUp",
	"/k8s/storage":          "Files",
	"/k8s/crd":              "DocumentAdd",
	"/k8s/monitoring":       "Odometer",
	"/monitor":              "AlarmClock",
	"/monitor/domain":       "Lock",
	"/monitor/api-endpoint": "Link",
	"/monitor/alarm/rules":  "Bell",
	"/monitor/alarm/notify": "Message",
	"/monitor/datasource":   "Coin",
	"/monitor/incident":     "Warning",
	"/config":               "SetUp",
	"/config/ecskey":        "Key",
	"/config/accountauth":   "UserFilled",
	"/config/keymanage":     "Lock",
	"/task":                 "Operation",
	"/task/template":        "Tickets",
	"/task/job":             "List",
	"/task/ansible":         "Connection",
	"/task/config":          "SetUp",
	"/tools":                "Tools",
	"/tools/ops":            "Tools",
	"/tools/agent":          "Cpu",
	"/app":                  "Collection",
	"/app/application":      "Grid",
	"/app/quick-release":    "Promotion",
	"/system":               "Setting",
	"/system/authorization": "Lock",
	"/system/api-keys":      "Key",
	"/system/admin":         "User",
	"/system/role":          "UserFilled",
	"/system/menu":          "Menu",
	"/system/dept":          "OfficeBuilding",
	"/system/post":          "Postcard",
	"/system/machine":       "House",
	"/monitor/loginlog":     "UserFilled",
	"/monitor/operator":     "Document",
	"/monitor/dblog":        "DocumentCopy",
}

// UpdateNavigationIcons applies only icon defaults, without running unrelated
// schema/role migrations or restarting the API.
func UpdateNavigationIcons(database *gorm.DB) error {
	return database.Transaction(fillNavigationIcons)
}

func fillNavigationIcons(tx *gorm.DB) error {
	var menus []model.SysMenu
	if e := tx.Where("menu_type IN ?", []int{1, 2}).Find(&menus).Error; e != nil {
		return e
	}
	for _, menu := range menus {
		icon := strings.TrimSpace(menu.Icon)
		if icon != "" && !strings.HasPrefix(icon, "ep/") && !strings.HasPrefix(icon, "ep:") && !strings.HasPrefix(icon, "ri/") && !strings.HasPrefix(icon, "ri:") {
			continue
		}
		path := "/" + strings.Trim(menu.Url, "/")
		fallback := navigationIconNames[path]
		if menu.Url == "" {
			fallback = "Menu"
		}
		if fallback == "" {
			fallback = "Menu"
		}
		if icon == fallback {
			continue
		}
		if e := tx.Model(&model.SysMenu{}).Where("id = ?", menu.ID).Update("icon", fallback).Error; e != nil {
			return e
		}
	}
	return nil
}
