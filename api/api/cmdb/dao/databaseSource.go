package dao

import (
	"dodevops-api/api/cmdb/model"
	"strconv"
	"strings"
)

type DatabaseSourceView struct {
	model.CmdbSQL
	GroupName string `json:"groupName"`
	Address   string `json:"address"`
	Username  string `json:"username"`
}

func descendantDatabaseGroups(groups []model.CmdbGroup, root uint) []uint {
	seen := map[uint]bool{root: true}
	for changed := true; changed; {
		changed = false
		for _, g := range groups {
			if seen[g.ParentID] && !seen[g.ID] {
				seen[g.ID] = true
				changed = true
			}
		}
	}
	ids := []uint{}
	for id := range seen {
		ids = append(ids, id)
	}
	return ids
}
func (d *CmdbSQLDao) Search(page, size int, name string, kind int, group uint, scope map[uint]bool) ([]DatabaseSourceView, int64, error) {
	var groups []model.CmdbGroup
	if err := d.db.Select("id,parent_id,name").Find(&groups).Error; err != nil {
		return nil, 0, err
	}
	q := ApplyAssetIDScope(d.db.Model(&model.CmdbSQL{}), scope)
	if name = strings.TrimSpace(name); name != "" {
		q = q.Where("name LIKE ?", "%"+name+"%")
	}
	if kind != 0 {
		q = q.Where("type = ?", kind)
	}
	if group != 0 {
		q = q.Where("group_id IN ?", descendantDatabaseGroups(groups, group))
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.CmdbSQL
	if err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	groupNames := map[uint]string{}
	for _, g := range groups {
		groupNames[g.ID] = g.Name
	}
	result := []DatabaseSourceView{}
	for _, r := range rows {
		item := DatabaseSourceView{CmdbSQL: r, GroupName: groupNames[r.GroupID]}
		var account struct {
			Host string
			Port int
			Name string
		}
		if err := d.db.Table("config_account").Select("host,port,name").Where("id = ?", r.AccountID).Take(&account).Error; err == nil {
			item.Address = account.Host + ":" + strconv.Itoa(account.Port)
			item.Username = account.Name
		}
		result = append(result, item)
	}
	return result, total, nil
}
func (d *CmdbSQLDao) SourceGroups(scope map[uint]bool) ([]model.CmdbGroup, error) {
	var groups []model.CmdbGroup
	if err := d.db.Select("id,parent_id,name").Order("id").Find(&groups).Error; err != nil {
		return nil, err
	}
	if scope == nil {
		return model.BuildCmdbGroupTree(groups), nil
	}
	var ids []uint
	if err := ApplyAssetIDScope(d.db.Model(&model.CmdbSQL{}), scope).Distinct().Pluck("group_id", &ids).Error; err != nil {
		return nil, err
	}
	parents := map[uint]uint{}
	for _, g := range groups {
		parents[g.ID] = g.ParentID
	}
	allowed := map[uint]bool{}
	for _, id := range ids {
		for id != 0 && !allowed[id] {
			allowed[id] = true
			id = parents[id]
		}
	}
	filtered := []model.CmdbGroup{}
	for _, g := range groups {
		if allowed[g.ID] {
			filtered = append(filtered, g)
		}
	}
	return model.BuildCmdbGroupTree(filtered), nil
}
