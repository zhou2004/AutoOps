package dao

import (
	"dodevops-api/api/monitor/model"
	"dodevops-api/common"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type IncidentDao struct{ db *gorm.DB }

func NewIncidentDao() *IncidentDao                           { return &IncidentDao{db: common.GetDB()} }
func NewIncidentDaoWithDB(db *gorm.DB) *IncidentDao          { return &IncidentDao{db: db} }
func (d *IncidentDao) Create(m *model.MonitorIncident) error { return d.db.Create(m).Error }
func (d *IncidentDao) GetByID(id uint) (*model.MonitorIncident, error) {
	var m model.MonitorIncident
	err := d.db.First(&m, id).Error
	return &m, err
}
func (d *IncidentDao) GetList(req *model.IncidentListReq) ([]model.MonitorIncident, int64, error) {
	list := []model.MonitorIncident{}
	var total int64
	query := d.db.Model(&model.MonitorIncident{})
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.Level != "" {
		query = query.Where("level = ?", req.Level)
	}
	if req.Source != "" {
		query = query.Where("source = ?", req.Source)
	}
	if req.Keyword != "" {
		query = query.Where("(title LIKE ? OR description LIKE ?)", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}
	if req.StartDate != "" {
		query = query.Where("alert_time >= ?", req.StartDate)
	}
	if req.EndDate != "" {
		end, err := time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			return nil, 0, err
		}
		query = query.Where("alert_time < ?", end.AddDate(0, 0, 1).Format("2006-01-02"))
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}
	err := query.Order("CASE WHEN status = 'firing' THEN 0 ELSE 1 END, alert_time DESC, id DESC").Offset((req.Page - 1) * req.PageSize).Limit(req.PageSize).Find(&list).Error
	return list, total, err
}

// 每个指标都来自完整数据集；任一查询失败都会返回错误，避免把故障伪装为零。
func (d *IncidentDao) GetStats() (*model.IncidentStats, error) {
	now := time.Now()
	today := now.Format("2006-01-02")
	stats := &model.IncidentStats{ByLevel: map[string]int64{}, BySource: map[string]int64{}, Trend: []model.IncidentTrendPoint{}}
	counts := []struct {
		condition string
		value     interface{}
		out       *int64
	}{
		{"status = ?", "firing", &stats.TotalFiring}, {"status = ?", "resolved", &stats.TotalResolved},
		{"alert_time >= ?", now.Add(-24 * time.Hour).Format("2006-01-02 15:04:05"), &stats.Last24hCount},
		{"alert_time >= ?", today, &stats.TodayCount},
		{"status = 'firing' AND level = ?", "critical", &stats.CriticalFiring},
	}
	for _, c := range counts {
		if err := d.db.Model(&model.MonitorIncident{}).Where(c.condition, c.value).Count(c.out).Error; err != nil {
			return nil, err
		}
	}
	for _, group := range []struct {
		column string
		out    map[string]int64
	}{{"level", stats.ByLevel}, {"source", stats.BySource}} {
		var rows []struct {
			Key   string
			Count int64
		}
		if err := d.db.Model(&model.MonitorIncident{}).Select(group.column + " AS `key`, COUNT(*) AS count").Group(group.column).Scan(&rows).Error; err != nil {
			return nil, err
		}
		for _, r := range rows {
			group.out[r.Key] = r.Count
		}
	}
	var trend []model.IncidentTrendPoint
	start := now.AddDate(0, 0, -6).Format("2006-01-02")
	if err := d.db.Model(&model.MonitorIncident{}).Select("SUBSTR(alert_time, 1, 10) AS date, COUNT(*) AS count").Where("alert_time >= ? AND alert_time < ?", start, now.AddDate(0, 0, 1).Format("2006-01-02")).Group("SUBSTR(alert_time, 1, 10)").Scan(&trend).Error; err != nil {
		return nil, err
	}
	byDate := map[string]int64{}
	for _, point := range trend {
		byDate[point.Date] = point.Count
	}
	for i := -6; i <= 0; i++ {
		date := now.AddDate(0, 0, i).Format("2006-01-02")
		stats.Trend = append(stats.Trend, model.IncidentTrendPoint{Date: date, Count: byDate[date]})
	}
	return stats, nil
}
func (d *IncidentDao) Resolve(id uint) error {
	m, err := d.GetByID(id)
	if err != nil {
		return err
	}
	if m.Status == "resolved" {
		return nil
	}
	return d.db.Model(&model.MonitorIncident{}).Where("id = ? AND status = ?", id, "firing").Updates(map[string]interface{}{"status": "resolved", "resolved_at": time.Now().Format("2006-01-02 15:04:05")}).Error
}
func (d *IncidentDao) Delete(id uint) error {
	r := d.db.Delete(&model.MonitorIncident{}, id)
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// EventKey 是“告警身份 + 本轮开始时间”，重复推送只更新同一故障；已恢复/人工解决的事件不会被迟到的 firing 重开。
func (d *IncidentDao) Ingest(m *model.MonitorIncident) error {
	if m.EventKey == nil {
		return errors.New("故障事件标识不能为空")
	}
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "event_key"}}, DoNothing: true}).Create(m).Error; err != nil {
			return err
		}
		if m.Status == "resolved" {
			return tx.Model(&model.MonitorIncident{}).Where("event_key = ? AND status = ?", *m.EventKey, "firing").Updates(map[string]interface{}{"status": "resolved", "resolved_at": m.ResolvedAt}).Error
		}
		return nil
	})
}
