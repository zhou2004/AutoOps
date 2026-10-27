package dao

import (
	"dodevops-api/api/cmdb/model"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"
	"unicode/utf8"
)

func lockBusinessGroups(tx *gorm.DB) ([]model.CmdbGroup, error) {
	var groups []model.CmdbGroup
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Order("id").Find(&groups).Error
	return groups, err
}

func validateBusinessGroup(group *model.CmdbGroup, groups []model.CmdbGroup, editing bool) error {
	group.Name = strings.TrimSpace(group.Name)
	if group.Name == "" || utf8.RuneCountInString(group.Name) > 50 {
		return errors.New("分组名称不能为空且不能超过 50 个字符")
	}
	if editing && group.ID == 1 {
		return errors.New("默认分组不能修改")
	}
	byID := map[uint]model.CmdbGroup{}
	for _, g := range groups {
		byID[g.ID] = g
		if strings.EqualFold(g.Name, group.Name) && (!editing || g.ID != group.ID) {
			return errors.New("分组名称已存在")
		}
	}
	if editing {
		if _, ok := byID[group.ID]; !ok {
			return errors.New("分组不存在")
		}
	}
	if group.ParentID == 1 {
		return errors.New("默认分组不能作为父级分组")
	}
	visited := map[uint]bool{}
	if editing {
		visited[group.ID] = true
	}
	for parent := group.ParentID; parent != 0; {
		if visited[parent] {
			return errors.New("不能将分组移动到自身或子分组下")
		}
		visited[parent] = true
		g, ok := byID[parent]
		if !ok {
			return errors.New("父级分组不存在")
		}
		parent = g.ParentID
	}
	return nil
}

func ensureBusinessGroupEmpty(tx *gorm.DB, id uint, groups []model.CmdbGroup) error {
	if id == 0 || id == 1 {
		return errors.New("默认分组不能删除")
	}
	exists := false
	for _, group := range groups {
		if group.ID == id {
			exists = true
		}
		if group.ParentID == id {
			return errors.New("该分组包含子分组，请先调整子分组")
		}
	}
	if !exists {
		return errors.New("分组不存在")
	}
	for _, ref := range []struct{ table, column, message string }{
		{"cmdb_host", "group_id", "该分组仍关联主机，请先调整主机分组"},
		{"cmdb_sql", "group_id", "该分组仍关联数据库，请先调整数据库分组"},
		{"monitor_alert_group_rule", "cmdb_group_id", "该分组仍关联告警群组，请先调整告警群组业务线"},
	} {
		if !tx.Migrator().HasTable(ref.table) || !tx.Migrator().HasColumn(ref.table, ref.column) {
			continue
		}
		query := tx.Table(ref.table).Where(ref.column+" = ?", id)
		if ref.table == "monitor_alert_group_rule" {
			query = query.Where("deleted_at IS NULL")
		}
		var count int64
		if err := query.Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return errors.New(ref.message)
		}
	}
	return nil
}
