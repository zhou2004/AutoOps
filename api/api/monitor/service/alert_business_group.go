package service

import (
	"context"
	cmdbdao "dodevops-api/api/cmdb/dao"
	cmdb "dodevops-api/api/cmdb/model"
	"dodevops-api/api/monitor/model"
	"encoding/json"
	"errors"
	"gorm.io/gorm"
	"strings"
	"unicode/utf8"
)

type AlertBusinessGroup struct {
	ID       uint   `json:"id"`
	ParentID uint   `json:"parentId"`
	Name     string `json:"name"`
}

func AlertBusinessGroups(ctx context.Context, db *gorm.DB, user uint) ([]AlertBusinessGroup, error) {
	db = db.WithContext(ctx)
	permissions := cmdbdao.NewCmdbAssetPermissionDao(db)
	admin, err := permissions.IsAdministrator(user)
	if err != nil {
		return nil, err
	}
	query := db.Model(&cmdb.CmdbGroup{})
	if !admin {
		grants, err := permissions.EffectivePermissions(user)
		if err != nil {
			return nil, err
		}
		ids, err := permissions.AllowedAssetIDs(grants, "group", "get")
		if err != nil {
			return nil, err
		}
		query = cmdbdao.ApplyAssetIDScope(query, ids)
	}
	groups := []AlertBusinessGroup{}
	if err := query.Select("id, parent_id, name").Order("id").Scan(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

func prepareAlertBusinessGroup(db *gorm.DB, group *model.MonitorAlertGroupRule) error {
	group.GroupName = strings.TrimSpace(group.GroupName)
	if group.GroupName == "" || utf8.RuneCountInString(group.GroupName) > 255 {
		return errors.New("请填写有效的告警群组名称")
	}
	if group.CmdbGroupID == 0 {
		return errors.New("请选择资产管理中的业务分组")
	}
	var business cmdb.CmdbGroup
	if err := db.First(&business, group.CmdbGroupID).Error; err != nil {
		return errors.New("所选业务分组不存在，请刷新后重新选择")
	}
	var count int64
	if err := db.Model(&model.MonitorDataSource{}).Where("id = ?", group.DataSourceID).Count(&count).Error; err != nil || count == 0 {
		return errors.New("请选择有效的数据源")
	}
	labels := map[string]string{}
	if group.Labels != "" && json.Unmarshal([]byte(group.Labels), &labels) != nil {
		return errors.New("群组标签格式无效")
	}
	if labels == nil {
		labels = map[string]string{}
	}
	labels["cluster"] = business.Name
	encoded, _ := json.Marshal(labels)
	group.Labels = string(encoded)
	return nil
}
