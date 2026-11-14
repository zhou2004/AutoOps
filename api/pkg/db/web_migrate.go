package db

import (
	monitor "dodevops-api/api/monitor/model"
	"dodevops-api/api/system/model"
	"dodevops-api/common/util"
	"gorm.io/gorm"
	"time"
)

func migrateWebNavigation(db *gorm.DB) error {
	if db.Migrator().HasTable(&monitor.MonitorAlertRuleStyle{}) {
		if err := db.Where("name = ?", "web").FirstOrCreate(&monitor.MonitorAlertRuleStyle{Name: "web", Description: "Web 服务健康、配置漂移、证书与发布；例如 instance_unavailable == 1"}).Error; err != nil {
			return err
		}
	}
	if !db.Migrator().HasTable(&model.SysMenu{}) {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		root := model.SysMenu{MenuName: "Web 管理", Url: "web", Icon: "Monitor", MenuType: 1, MenuStatus: 2, Sort: 12, Value: "web:read", CreateTime: util.HTime{Time: time.Now()}}
		if e := tx.Where("url = ? AND menu_type = ?", "web", 1).FirstOrCreate(&root).Error; e != nil {
			return e
		}
		// Retire the Web terminal entry while retaining historical roles and audit records.
		if err := tx.Model(&model.SysMenu{}).Where("url IN ? AND menu_type = ?", []string{"web/terminal", "/web/terminal"}, 2).Update("menu_status", 1).Error; err != nil {
			return err
		}
		ids := []uint{root.ID}
		for i, v := range []struct{ Name, URL, Icon string }{{"数据源管理", "sources", "Connection"}, {"集群管理", "clusters", "SetUp"}, {"站点管理", "sites", "Monitor"}, {"配置管理", "configurations", "Document"}, {"证书管理", "certificates", "Lock"}, {"日志管理", "logs", "Tickets"}, {"流量监控", "traffic", "TrendCharts"}} {
			row := model.SysMenu{ParentId: root.ID, MenuName: v.Name, Url: "web/" + v.URL, Icon: v.Icon, MenuType: 2, MenuStatus: 2, Sort: uint(i + 1), Value: "web:read", CreateTime: util.HTime{Time: time.Now()}}
			if e := tx.Where("url = ? AND menu_type = ?", row.Url, 2).FirstOrCreate(&row).Error; e != nil {
				return e
			}
			ids = append(ids, row.ID)
		}
		if tx.Migrator().HasTable("sys_role") && tx.Migrator().HasTable(&model.SysRoleMenu{}) {
			var roles []uint
			if e := tx.Table("sys_role").Where("role_key = ? AND status = ?", "admin", 1).Pluck("id", &roles).Error; e != nil {
				return e
			}
			for _, r := range roles {
				for _, id := range ids {
					row := model.SysRoleMenu{RoleId: r, MenuId: id}
					if e := tx.Where("role_id = ? AND menu_id = ?", r, id).FirstOrCreate(&row).Error; e != nil {
						return e
					}
				}
			}
		}
		return nil
	})
}
