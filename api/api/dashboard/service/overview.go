package service

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"
	_ "time/tzdata"

	cmdbdao "dodevops-api/api/cmdb/dao"
	model "dodevops-api/api/dashboard/model1"
	"dodevops-api/pkg/log"
	"gorm.io/gorm"
)

var dashboardLocation = func() *time.Location {
	location, _ := time.LoadLocation("Asia/Shanghai")
	return location
}()

// GetOverview only reads stored business records. It never probes hosts or runs jobs.
// CMDB permissions are evaluated per request; no cross-user cache is used.
func (s *DashboardService) GetOverview(ctx context.Context, userID uint, clusterIDs []uint, days int) (*model.Overview, error) {
	if days != 7 && days != 30 && days != 90 {
		return nil, fmt.Errorf("days 仅支持 7、30、90")
	}
	ctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	db := s.db.WithContext(ctx)
	permissions := cmdbdao.NewCmdbAssetPermissionDao(db)
	active, err := permissions.UserActive(userID)
	if err != nil {
		return nil, err
	}
	if !active {
		return nil, fmt.Errorf("账号不可用")
	}
	admin, err := permissions.IsAdministrator(userID)
	if err != nil {
		return nil, err
	}
	scopes := map[string]map[uint]bool{}
	if !admin {
		if clusterIDs == nil {
			clusterIDs = []uint{}
		}
		rules, err := permissions.EffectivePermissions(userID)
		if err != nil {
			return nil, err
		}
		for _, kind := range []string{"host", "database", "physical", "network"} {
			scopes[kind], err = permissions.AllowedAssetIDs(rules, kind, "list")
			if err != nil {
				return nil, err
			}
		}
	}
	now := time.Now().In(dashboardLocation)
	return buildOverview(db, now, days, scopes, clusterIDs, admin), nil
}

func buildOverview(db *gorm.DB, now time.Time, days int, scopes map[string]map[uint]bool, clusterIDs []uint, admin bool) *model.Overview {
	now = now.In(dashboardLocation)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, dashboardLocation)
	start := today.AddDate(0, 0, 1-days)
	out := &model.Overview{
		GeneratedAt: now, Days: days, Timezone: "Asia/Shanghai",
		StartDate: start.Format("2006-01-02"), EndDate: today.Format("2006-01-02"),
		AssetScope: "authorized", Errors: map[string]string{},
	}
	if admin {
		out.AssetScope = "all"
	}
	recordError := func(key, label string, err error) {
		if err != nil {
			out.Errors[key] = label + "数据读取失败，请稍后刷新"
			log.Log().Errorf("dashboard %s: %v", key, err)
		}
	}
	loadInventory := func(key, label, table, kind, status, distribution string, statuses, names map[string]string, target **model.Inventory) {
		query := cmdbdao.ApplyAssetIDScope(db.Table(table), scopes[kind])
		value, err := inventory(query, status, distribution, statuses, names)
		recordError(key, label, err)
		if err == nil {
			*target = value
		}
	}
	loadInventory("hosts", "主机", "cmdb_host", "host", "status", "vendor",
		map[string]string{"1": "认证成功", "2": "未认证", "3": "认证失败"},
		map[string]string{"1": "自建主机", "2": "阿里云", "3": "腾讯云"}, &out.Hosts)
	loadInventory("databases", "数据库", "cmdb_sql", "database", "", "type", nil,
		map[string]string{"1": "MySQL", "2": "PostgreSQL", "3": "Redis", "4": "MongoDB", "5": "Elasticsearch"}, &out.Databases)
	loadInventory("physical", "物理机", "cmdb_physical_machine", "physical", "status", "",
		map[string]string{"1": "运行中", "2": "关机", "3": "离线"}, nil, &out.Physical)
	loadInventory("network", "网络设备", "cmdb_network_device", "network", "status", "",
		map[string]string{"1": "运行中", "2": "关机", "3": "离线"}, nil, &out.Network)

	clusters := db.Table("k8s_cluster")
	if clusterIDs != nil {
		clusters = clusters.Where("id IN ?", clusterIDs)
	}
	var err error
	out.Clusters, err = inventory(clusters, "status", "cluster_type",
		map[string]string{"1": "创建中", "2": "运行中", "3": "离线"},
		map[string]string{"1": "自建集群", "2": "导入集群"})
	recordError("clusters", "集群", err)

	out.Applications, err = inventory(db.Table("app_application"), "status", "",
		map[string]string{"1": "未激活", "2": "已激活"}, nil)
	if err == nil {
		var groups []model.CountItem
		err = db.Table("app_application a").
			Select("COALESCE(NULLIF(g.name, ''), '未分组 / 分组已删除') AS name, COUNT(*) AS count").
			Joins("LEFT JOIN cmdb_group g ON g.id = a.business_group_id").
			Group("a.business_group_id, g.name").Order("count DESC, name ASC").Scan(&groups).Error
		if err == nil {
			out.Applications.Distribution = topDistribution(groups, 10)
		} else {
			out.Applications = nil
		}
	}
	recordError("applications", "应用", err)

	for _, source := range []struct {
		key, label, table, name string
		target                  **model.ExecutionOverview
	}{
		{"deployments", "快速发布", "quick_deployments", "title", &out.Deployments},
		{"jobs", "作业执行", "task_execution_history", "task_name", &out.Jobs},
		{"ansible", "Ansible 执行", "task_ansible_history", "", &out.Ansible},
	} {
		*source.target, err = executionOverview(db.Table(source.table), source.name, start, now, days)
		recordError(source.key, source.label, err)
	}

	// Count distinct monitored hosts and ignore agents whose host has been removed.
	hosts := cmdbdao.ApplyAssetIDScope(db.Table("cmdb_host"), scopes["host"]).Select("id")
	agentQuery := db.Table("monitor_agent").Where("host_id IN (?)", hosts).
		Where("id IN (?)", db.Table("monitor_agent").Select("MAX(id)").Group("host_id"))
	// The five-minute heartbeat window matches the agent service's online threshold.
	agentStatus := "CASE WHEN status = 3 AND last_heartbeat >= ? AND last_heartbeat <= ? THEN '心跳正常' WHEN status = 1 THEN '部署中' WHEN status = 2 THEN '部署失败' WHEN status = 4 THEN '启动异常' ELSE '心跳过期 / 未上报' END"
	out.Agents, err = groupedInventory(agentQuery, agentStatus, []interface{}{now.Add(-5 * time.Minute), now}, "COUNT(DISTINCT host_id)")
	recordError("agents", "Agent", err)
	checkedStatus := "CASE WHEN check_time IS NULL OR check_time = '' THEN '未检查' ELSE CAST(status AS CHAR) END"
	out.Endpoints, err = mappedInventory(db.Table("monitor_api_endpoint"), checkedStatus,
		map[string]string{"1": "正常", "2": "异常", "3": "超时", "4": "检查失败"})
	recordError("endpoints", "API 监控", err)
	out.Certificates, err = mappedInventory(db.Table("monitor_domain_cert"), checkedStatus,
		map[string]string{"1": "正常", "2": "即将过期", "3": "已过期", "4": "检查失败"})
	recordError("certificates", "证书监控", err)
	out.Alerts, err = mappedInventory(db.Table("monitor_alert_record"), "alert_status",
		map[string]string{"firing": "触发记录", "resolved": "恢复记录"})
	recordError("alerts", "告警记录", err)
	out.Incidents, err = incidentOverview(db, start, now, days)
	recordError("incidents", "故障告警", err)
	return out
}

func groupCounts(query *gorm.DB, expression string, args []interface{}, count string) ([]model.CountItem, error) {
	items := []model.CountItem{}
	var rows []struct {
		Bucket string `gorm:"column:dashboard_bucket"`
		Count  int64
	}
	err := query.Session(&gorm.Session{}).Select(expression+" AS dashboard_bucket, "+count+" AS count", args...).
		Group("dashboard_bucket").Order("count DESC, dashboard_bucket ASC").Scan(&rows).Error
	for _, row := range rows {
		items = append(items, model.CountItem{Name: row.Bucket, Count: row.Count})
	}
	return items, err
}

func translateCounts(items []model.CountItem, names map[string]string) {
	for i := range items {
		if name, ok := names[items[i].Name]; ok {
			items[i].Name = name
		} else if items[i].Name == "" {
			items[i].Name = "未分类"
		} else if names != nil {
			if _, err := strconv.Atoi(items[i].Name); err == nil {
				items[i].Name = "未知（" + items[i].Name + "）"
			}
		}
	}
}

func groupedInventory(query *gorm.DB, expression string, args []interface{}, count string) (*model.Inventory, error) {
	items, err := groupCounts(query, expression, args, count)
	if err != nil {
		return nil, err
	}
	out := &model.Inventory{Statuses: items, Distribution: []model.CountItem{}}
	for _, item := range items {
		out.Total += item.Count
	}
	return out, nil
}

func mappedInventory(query *gorm.DB, expression string, names map[string]string) (*model.Inventory, error) {
	out, err := groupedInventory(query, expression, nil, "COUNT(*)")
	if err == nil {
		translateCounts(out.Statuses, names)
	}
	return out, err
}

func inventory(query *gorm.DB, status, distribution string, statuses, names map[string]string) (*model.Inventory, error) {
	out := &model.Inventory{Statuses: []model.CountItem{}, Distribution: []model.CountItem{}}
	if err := query.Session(&gorm.Session{}).Count(&out.Total).Error; err != nil {
		return nil, err
	}
	var err error
	if status != "" {
		out.Statuses, err = groupCounts(query, status, nil, "COUNT(*)")
		if err != nil {
			return nil, err
		}
		translateCounts(out.Statuses, statuses)
	}
	if distribution != "" {
		out.Distribution, err = groupCounts(query, distribution, nil, "COUNT(*)")
		if err != nil {
			return nil, err
		}
		translateCounts(out.Distribution, names)
	}
	return out, nil
}

func topDistribution(items []model.CountItem, limit int) []model.CountItem {
	if len(items) <= limit {
		if items == nil {
			return []model.CountItem{}
		}
		return items
	}
	result := append([]model.CountItem{}, items[:limit]...)
	other := model.CountItem{Name: "其他业务组"}
	for _, item := range items[limit:] {
		other.Count += item.Count
	}
	return append(result, other)
}

func executionOverview(query *gorm.DB, name string, start, now time.Time, days int) (*model.ExecutionOverview, error) {
	out := &model.ExecutionOverview{Trend: make([]model.ExecutionDay, days), Recent: []model.ExecutionRecord{}}
	if err := query.Session(&gorm.Session{}).Count(&out.AllTimeTotal).Error; err != nil {
		return nil, err
	}
	period := query.Session(&gorm.Session{}).Where("created_at >= ? AND created_at <= ?", start, now)
	var rows []struct {
		Date   string
		Status int
		Count  int64
	}
	dateExpression := calendarDateSQL(query, "created_at")
	if err := period.Session(&gorm.Session{}).Select(dateExpression + " AS date, status, COUNT(*) AS count").
		Group(dateExpression + ", status").Scan(&rows).Error; err != nil {
		return nil, err
	}
	index := map[string]int{}
	for i := range out.Trend {
		date := start.AddDate(0, 0, i).Format("2006-01-02")
		out.Trend[i].Date = date
		index[date] = i
	}
	for _, row := range rows {
		i, ok := index[row.Date]
		if !ok {
			continue
		}
		out.Total += row.Count
		out.Trend[i].Total += row.Count
		switch row.Status {
		case 1:
			out.Pending += row.Count
		case 2:
			out.Running += row.Count
		case 3:
			out.Success += row.Count
			out.Trend[i].Success += row.Count
		case 4:
			out.Failed += row.Count
			out.Trend[i].Failed += row.Count
		default:
			out.Other += row.Count
		}
	}
	if completed := out.Success + out.Failed; completed > 0 {
		rate := math.Round(float64(out.Success)/float64(completed)*10000) / 100
		out.SuccessRate = &rate
	}
	selection := "id, status, created_at"
	if name != "" {
		selection += ", " + name + " AS name"
	}
	if err := query.Session(&gorm.Session{}).Where("created_at <= ?", now).Select(selection).Order("created_at DESC, id DESC").
		Limit(8).Scan(&out.Recent).Error; err != nil {
		return nil, err
	}
	for i := range out.Recent {
		if out.Recent[i].Name == "" {
			out.Recent[i].Name = fmt.Sprintf("执行记录 #%d", out.Recent[i].ID)
		}
	}
	return out, nil
}

func incidentOverview(db *gorm.DB, start, now time.Time, days int) (*model.IncidentOverview, error) {
	out := &model.IncidentOverview{Trend: make([]model.CountItem, days), Recent: []model.IncidentRecord{}}
	// Both schemas exist in shipped SQL dumps. Read legacy records without migrating
	// or remapping their P1/P2/P3/P4 priority vocabulary.
	columns, err := db.Migrator().ColumnTypes("monitor_incident")
	if err != nil {
		return nil, err
	}
	present := map[string]bool{}
	for _, column := range columns {
		present[column.Name()] = true
	}
	level, title, source, predicate := "level", "title", "source", "status = 'firing'"
	if !present["level"] || !present["title"] {
		if !present["alert_level"] || !present["alert_desc"] {
			return nil, fmt.Errorf("无法识别故障记录表结构")
		}
		level, title, source, predicate = "alert_level", "alert_desc", "'legacy'", "status IN (1, 2)"
	}
	active, err := groupCounts(db.Table("monitor_incident").Where(predicate), level, nil, "COUNT(*)")
	if err != nil {
		return nil, err
	}
	out.Levels = active
	for _, item := range active {
		out.Firing += item.Count
		switch item.Name {
		case "critical":
			out.Critical += item.Count
		case "warning":
			out.Warning += item.Count
		case "info":
			out.Info += item.Count
		}
	}
	daily, err := groupCounts(db.Table("monitor_incident").Where("create_time >= ? AND create_time <= ?", start, now),
		calendarDateSQL(db, "create_time"), nil, "COUNT(*)")
	if err != nil {
		return nil, err
	}
	counts := map[string]int64{}
	for _, day := range daily {
		counts[day.Name] = day.Count
	}
	for i := range out.Trend {
		date := start.AddDate(0, 0, i).Format("2006-01-02")
		out.Trend[i] = model.CountItem{Name: date, Count: counts[date]}
		out.PeriodTotal += counts[date]
	}
	err = db.Table("monitor_incident").Select("id, " + title + " AS title, " + level + " AS level, " + source + " AS source, create_time AS created_at").
		Where(predicate).
		Order("CASE " + level + " WHEN 'critical' THEN 0 WHEN 'P1' THEN 0 WHEN 'P2' THEN 1 WHEN 'warning' THEN 2 WHEN 'P3' THEN 2 ELSE 3 END, create_time DESC, id DESC").
		Limit(8).Scan(&out.Recent).Error
	if err != nil {
		return nil, err
	}
	return out, nil
}

func calendarDateSQL(db *gorm.DB, column string) string {
	// SQLite normalizes explicit offsets to UTC in DATE(); preserve the stored
	// calendar date, matching MySQL DATETIME and the dashboard's business timezone.
	if db.Dialector.Name() == "sqlite" {
		return "SUBSTR(" + column + ", 1, 10)"
	}
	return "DATE(" + column + ")"
}
