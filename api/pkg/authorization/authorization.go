package authorization

import (
	"context"
	"dodevops-api/api/system/model"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"sort"
	"strings"
)

type Identity struct {
	UserID      uint     `json:"userId"`
	Username    string   `json:"username"`
	Admin       bool     `json:"isAdmin"`
	Permissions []string `json:"permissions"`
}

func (i Identity) Allows(permission string) bool {
	if permission == "" {
		return false
	}
	if i.Admin {
		return true
	}
	for _, p := range i.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// Load never caches grants: account, role and grant changes take effect on the next request.
func Load(ctx context.Context, db *gorm.DB, user uint) (Identity, error) {
	i := Identity{UserID: user, Permissions: []string{}}
	var account struct {
		Username string
		Status   int
	}
	db = db.WithContext(ctx)
	if err := db.Table("sys_admin").Select("username, status").Where("id = ?", user).Take(&account).Error; err != nil || account.Status != 1 {
		return i, errors.New("账号已停用或不存在")
	}
	i.Username = account.Username
	var roles []model.SysRole
	if err := db.Table("sys_role r").Select("r.*").Joins("JOIN sys_admin_role ar ON ar.role_id = r.id").Where("ar.admin_id = ? AND r.status = 1", user).Find(&roles).Error; err != nil {
		return i, err
	}
	set := map[string]bool{}
	for _, role := range roles {
		if role.RoleKey == "admin" {
			i.Admin = true
			continue
		}
		permissions, _, err := ForRole(ctx, db, role.ID)
		if err != nil {
			return i, err
		}
		for _, p := range permissions {
			set[p] = true
		}
	}
	if i.Admin {
		for _, m := range Catalog {
			for _, a := range m.Actions {
				set[m.Key+":"+a] = true
			}
		}
	}
	for p := range set {
		i.Permissions = append(i.Permissions, p)
	}
	sort.Strings(i.Permissions)
	return i, nil
}

func ForRole(ctx context.Context, db *gorm.DB, role uint) ([]string, bool, error) {
	permissions := []string{}
	var row model.RolePermission
	err := db.WithContext(ctx).Where("role_id = ?", role).Take(&row).Error
	if err == nil {
		err = json.Unmarshal([]byte(row.Permissions), &permissions)
		return permissions, false, err
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, err
	}
	// Preserve explicit legacy button grants; a menu's existence alone never grants writes.
	var menus []model.SysMenu
	err = db.WithContext(ctx).Table("sys_menu m").Select("m.*").Joins("JOIN sys_role_menu rm ON rm.menu_id = m.id").Where("rm.role_id = ? AND m.menu_status = 2", role).Find(&menus).Error
	if err != nil {
		return nil, true, err
	}
	set := map[string]bool{}
	for _, m := range menus {
		if Known(m.Value) {
			set[m.Value] = true
			continue
		}
		prefix := strings.Split(m.Value, ":")[0]
		module := map[string]string{"base": "system", "cmdb": "cmdb", "k8s": "k8s", "task": "task", "app": "app", "config": "config", "tool": "tool", "aiops": "aiops", "monitor": "monitor", "dashboard": "dashboard"}[prefix]
		if module == "" {
			continue
		}
		if strings.Contains(m.Value, "operator") || strings.Contains(m.Value, "loginLog") {
			module = "audit"
		}
		action := ""
		parts := strings.Split(m.Value, ":")
		switch parts[len(parts)-1] {
		case "list", "info", "get", "query":
			action = "read"
		case "add", "edit", "update", "delete", "clean", "create":
			if m.MenuType == 3 {
				action = "write"
			}
		case "run", "start", "stop", "execute", "connect", "jobstart", "jobstop":
			if m.MenuType == 3 {
				action = "execute"
			}
		}
		if Known(module + ":" + action) {
			set[module+":"+action] = true
		}
	}
	for p := range set {
		permissions = append(permissions, p)
	}
	sort.Strings(permissions)
	return permissions, true, nil
}
