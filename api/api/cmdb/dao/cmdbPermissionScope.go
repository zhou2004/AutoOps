package dao

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"dodevops-api/api/cmdb/model"
	"gorm.io/gorm"
)

// PermissionActions is the common vocabulary shared by asset management and sessions.
var PermissionActions = []string{"get", "list", "connect", "connect_ssh", "connect_vnc", "connect_rdp", "file_list", "file_upload", "file_download", "file_delete", "command", "config_view", "config_backup", "config_restore", "db_query", "db_execute", "docker_view", "docker_logs", "docker_operate", "docker_remove", "audit", "create", "update", "delete", "admin"}

func ValidPermissionAction(action string) bool {
	for _, a := range PermissionActions {
		if a == action {
			return true
		}
	}
	return false
}

func JSONContainsID(value string, id uint) bool {
	if id == 0 {
		return false
	}
	var ids []uint
	if json.Unmarshal([]byte(value), &ids) != nil {
		return false
	}
	for _, candidate := range ids {
		if candidate == id {
			return true
		}
	}
	return false
}

func permissionIDs(value string) []uint {
	var ids []uint
	if json.Unmarshal([]byte(value), &ids) != nil {
		return nil
	}
	return ids
}

func RuleAllowsAction(p model.CmdbAssetPermission, assetType, action string) bool {
	if !ValidPermissionAction(action) {
		return false
	}
	var types, actions []string
	if json.Unmarshal([]byte(p.AssetTypes), &types) != nil || json.Unmarshal([]byte(p.PermissionActions), &actions) != nil {
		return false
	}
	typeMatches := false
	for _, t := range types {
		if t == assetType || t == "*" {
			typeMatches = true
		}
	}
	if !typeMatches {
		return false
	}
	for _, a := range actions {
		// Historical connect meant SSH. It never grants desktop or file permissions.
		if a == action || a == "admin" || (a == "connect" && action == "connect_ssh") {
			return true
		}
	}
	return false
}

func ruleEffective(p model.CmdbAssetPermission, now time.Time) bool {
	if p.IsActive != 1 {
		return false
	}
	today := now.Format("2006-01-02")
	for _, value := range []string{p.DateStart, p.DateExpired} {
		if value != "" {
			if _, err := time.Parse("2006-01-02", value); err != nil {
				return false
			}
		}
	}
	return (p.DateStart == "" || p.DateStart <= today) && (p.DateExpired == "" || p.DateExpired >= today)
}

// UserActive rejects disabled or deleted accounts even when a previously issued JWT or ticket remains valid.
func (d *CmdbAssetPermissionDao) UserActive(userID uint) (bool, error) {
	if userID == 0 {
		return false, nil
	}
	var count int64
	err := d.db.Table("sys_admin").Where("id = ? AND status = 1", userID).Count(&count).Error
	return err == nil && count == 1, err
}

// userAccessState checks account and administrator status in one query.
func (d *CmdbAssetPermissionDao) userAccessState(userID uint) (active, administrator bool, err error) {
	if userID == 0 {
		return false, false, nil
	}
	var state struct {
		UserID        uint
		Administrator bool
	}
	err = d.db.Table("sys_admin u").
		Select("u.id AS user_id, EXISTS (SELECT 1 FROM sys_admin_role ar JOIN sys_role r ON r.id = ar.role_id WHERE ar.admin_id = u.id AND r.role_key = ? AND r.status = 1) AS administrator", "admin").
		Where("u.id = ? AND u.status = 1", userID).Scan(&state).Error
	if err != nil {
		return false, false, err
	}
	return state.UserID == userID, state.Administrator, nil
}

func (d *CmdbAssetPermissionDao) IsAdministrator(userID uint) (bool, error) {
	active, administrator, err := d.userAccessState(userID)
	return active && administrator, err
}

// UserGroupIDs uses only enabled CMDB groups; system-role IDs are a separate namespace.
func (d *CmdbAssetPermissionDao) UserGroupIDs(userID uint) ([]uint, error) {
	var ids []uint
	err := d.db.Table("cmdb_user_group_member m").Joins("JOIN cmdb_user_group g ON g.id = m.group_id").Where("m.user_id = ? AND g.status = 1", userID).Pluck("m.group_id", &ids).Error
	return ids, err
}

func (d *CmdbAssetPermissionDao) EffectivePermissions(userID uint) ([]model.CmdbAssetPermission, error) {
	active, err := d.UserActive(userID)
	if err != nil {
		return nil, err
	}
	if !active {
		return []model.CmdbAssetPermission{}, nil
	}
	return d.effectivePermissionsForActiveUser(userID)
}

func (d *CmdbAssetPermissionDao) effectivePermissionsForActiveUser(userID uint) ([]model.CmdbAssetPermission, error) {
	ids, err := d.UserGroupIDs(userID)
	if err != nil {
		return nil, err
	}
	return d.getUserPermissions(userID, ids)
}

// descendantGroups expands a group subtree with a visited set, including cyclic legacy data safely.
func (d *CmdbAssetPermissionDao) descendantGroups(ids []uint) ([]uint, error) {
	seen := map[uint]bool{}
	frontier := []uint{}
	for _, id := range ids {
		if id > 0 && !seen[id] {
			seen[id] = true
			frontier = append(frontier, id)
		}
	}
	for len(frontier) > 0 {
		var children []uint
		if err := d.db.Model(&model.CmdbGroup{}).Where("parent_id IN ?", frontier).Pluck("id", &children).Error; err != nil {
			return nil, err
		}
		frontier = nil
		for _, id := range children {
			if !seen[id] {
				seen[id] = true
				frontier = append(frontier, id)
			}
		}
	}
	result := make([]uint, 0, len(seen))
	for id := range seen {
		result = append(result, id)
	}
	return result, nil
}

// AllowedAssetIDs joins action and scope within each rule before taking their union.
// An empty result grants no assets, including when the rule contains admin.
func (d *CmdbAssetPermissionDao) AllowedAssetIDs(perms []model.CmdbAssetPermission, assetType, action string) (map[uint]bool, error) {
	result := map[uint]bool{}
	for _, p := range perms {
		if !RuleAllowsAction(p, assetType, action) {
			continue
		}
		var direct string
		switch assetType {
		case "host":
			direct = p.HostIDs
		case "physical":
			direct = p.PhysicalIDs
		case "network":
			direct = p.NetworkIDs
		case "database":
			direct = p.DatabaseIDs
		case "idc":
			direct = p.IDCIDs
		case "group":
			direct = p.HostGroupIDs
		case "cabinet":
		default:
			return nil, fmt.Errorf("不支持的资产类型: %s", assetType)
		}
		for _, id := range permissionIDs(direct) {
			if id > 0 {
				result[id] = true
			}
		}
		if assetType == "host" || assetType == "group" {
			groupIDs := permissionIDs(p.HostGroupIDs)
			if len(groupIDs) > 0 {
				expanded, err := d.descendantGroups(groupIDs)
				if err != nil {
					return nil, err
				}
				if assetType == "group" {
					for _, id := range expanded {
						result[id] = true
					}
				} else {
					var ids []uint
					if err := d.db.Model(&model.CmdbHost{}).Where("group_id IN ?", expanded).Pluck("id", &ids).Error; err != nil {
						return nil, err
					}
					for _, id := range ids {
						result[id] = true
					}
				}
			}
		}
		idcIDs := permissionIDs(p.IDCIDs)
		if len(idcIDs) > 0 && (assetType == "physical" || assetType == "network" || assetType == "cabinet") {
			table := map[string]string{"physical": "cmdb_physical_machine", "network": "cmdb_network_device", "cabinet": "cmdb_cabinet"}[assetType]
			var ids []uint
			if err := d.db.Table(table).Where("idc_id IN ?", idcIDs).Pluck("id", &ids).Error; err != nil {
				return nil, err
			}
			for _, id := range ids {
				result[id] = true
			}
		}
	}
	return result, nil
}

func (d *CmdbAssetPermissionDao) CanUserAccess(userID uint, assetType string, assetID uint, action string) (bool, error) {
	if assetID == 0 || !ValidPermissionAction(action) {
		return false, nil
	}
	active, admin, err := d.userAccessState(userID)
	if err != nil || !active || admin {
		return active && admin, err
	}
	perms, err := d.effectivePermissionsForActiveUser(userID)
	if err != nil {
		return false, err
	}
	ids, err := d.AllowedAssetIDs(perms, assetType, action)
	if err != nil {
		return false, err
	}
	return ids[assetID], nil
}

// Deprecated: retained only for legacy data tooling. Runtime operations use asset permissions.
func (d *CmdbAssetPermissionDao) CanUseCredential(userID, credentialID uint) (bool, error) {
	return d.CanUseCredentialSource(userID, credentialID, "ssh")
}

func (d *CmdbAssetPermissionDao) CanUseCredentialSource(userID, credentialID uint, source string) (bool, error) {
	if credentialID == 0 {
		return false, nil
	}
	if source != "ssh" && source != "database" {
		return false, nil
	}
	active, admin, err := d.userAccessState(userID)
	if err != nil || !active || admin {
		return active && admin, err
	}
	groups, err := d.UserGroupIDs(userID)
	if err != nil {
		return false, err
	}
	var rules []model.CmdbCredentialPermission
	if err = d.db.Where("credential_type = ? AND credential_id = ? AND is_active = 1", source, credentialID).Find(&rules).Error; err != nil {
		return false, err
	}
	for _, p := range rules {
		if JSONContainsID(p.UserIDs, userID) {
			return true, nil
		}
		for _, id := range groups {
			if JSONContainsID(p.GroupIDs, id) {
				return true, nil
			}
		}
	}
	return false, nil
}

// ApplyAssetIDScope applies authorization before count and pagination. A nil scope is unrestricted;
// a present empty scope must return no rows.
func ApplyAssetIDScope(db *gorm.DB, scopes ...map[uint]bool) *gorm.DB {
	if len(scopes) == 0 || scopes[0] == nil {
		return db
	}
	ids := make([]uint, 0, len(scopes[0]))
	for id, allowed := range scopes[0] {
		if allowed {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if len(ids) == 0 {
		return db.Where("1 = 0")
	}
	return db.Where("id IN ?", ids)
}

// CanPlaceAsset requires one rule to cover both the current asset and its new container.
// A grant for an individual asset permits editing it in place, not moving it into another scope.
func (d *CmdbAssetPermissionDao) CanPlaceAsset(perms []model.CmdbAssetPermission, assetType string, assetID, destination uint, action string) (bool, error) {
	table, column := "", ""
	switch assetType {
	case "host":
		table, column = "cmdb_host", "group_id"
	case "group":
		table, column = "cmdb_group", "parent_id"
	case "physical":
		table, column = "cmdb_physical_machine", "idc_id"
	case "network":
		table, column = "cmdb_network_device", "idc_id"
	case "cabinet":
		table, column = "cmdb_cabinet", "idc_id"
	default:
		return true, nil
	}
	var current uint
	if assetID > 0 {
		var rows []uint
		if err := d.db.Table(table).Where("id = ?", assetID).Pluck(column, &rows).Error; err != nil {
			return false, err
		}
		if len(rows) != 1 {
			return false, nil
		}
		current = rows[0]
	}
	for _, p := range perms {
		if !RuleAllowsAction(p, assetType, action) {
			continue
		}
		if assetID > 0 {
			allowed, err := d.AllowedAssetIDs([]model.CmdbAssetPermission{p}, assetType, action)
			if err != nil {
				return false, err
			}
			if !allowed[assetID] {
				continue
			}
			if current == destination {
				return true, nil
			}
		}
		if destination == 0 {
			continue
		}
		if assetType == "host" || assetType == "group" {
			groups, err := d.descendantGroups(permissionIDs(p.HostGroupIDs))
			if err != nil {
				return false, err
			}
			for _, id := range groups {
				if id == destination {
					return true, nil
				}
			}
		} else if JSONContainsID(p.IDCIDs, destination) {
			return true, nil
		}
	}
	return false, nil
}
