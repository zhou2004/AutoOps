package dao

import (
	"time"

	"dodevops-api/api/monitor/model"
	"dodevops-api/common"

	"gorm.io/gorm"
)

type IncidentDao struct {
	db *gorm.DB
}

func NewIncidentDao() *IncidentDao {
	return &IncidentDao{
		db: common.GetDB(),
	}
}

func (d *IncidentDao) Create(m *model.MonitorIncident) error {
	return d.db.Create(m).Error
}

func (d *IncidentDao) GetByID(id uint) (*model.MonitorIncident, error) {
	var m model.MonitorIncident
	err := d.db.First(&m, id).Error
	return &m, err
}

func (d *IncidentDao) GetList(page, pageSize int, status, level, source string) ([]model.MonitorIncident, int64, error) {
	var list []model.MonitorIncident
	var total int64
	query := d.db.Model(&model.MonitorIncident{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if level != "" {
		query = query.Where("level = ?", level)
	}
	if source != "" {
		query = query.Where("source = ?", source)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	if err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// GetStats 获取故障统计数据
func (d *IncidentDao) GetStats() (*model.IncidentStats, error) {
	stats := &model.IncidentStats{
		ByLevel:  make(map[string]int64),
		BySource: make(map[string]int64),
	}

	// 总数 - 活跃
	d.db.Model(&model.MonitorIncident{}).Where("status = ?", "firing").Count(&stats.TotalFiring)
	// 总数 - 已解决
	d.db.Model(&model.MonitorIncident{}).Where("status = ?", "resolved").Count(&stats.TotalResolved)

	// 按等级统计
	type LevelCount struct {
		Level string
		Count int64
	}
	var levelCounts []LevelCount
	d.db.Model(&model.MonitorIncident{}).Select("level, count(*) as count").Group("level").Find(&levelCounts)
	for _, lc := range levelCounts {
		stats.ByLevel[lc.Level] = lc.Count
	}

	// 按来源统计
	type SourceCount struct {
		Source string
		Count  int64
	}
	var sourceCounts []SourceCount
	d.db.Model(&model.MonitorIncident{}).Select("source, count(*) as count").Group("source").Find(&sourceCounts)
	for _, sc := range sourceCounts {
		stats.BySource[sc.Source] = sc.Count
	}

	// 过去24小时
	since24h := time.Now().Add(-24 * time.Hour)
	d.db.Model(&model.MonitorIncident{}).Where("create_time >= ?", since24h).Count(&stats.Last24hCount)

	// 今天
	today := time.Now().Format("2006-01-02")
	d.db.Model(&model.MonitorIncident{}).Where("create_time >= ?", today).Count(&stats.TodayCount)

	return stats, nil
}

func (d *IncidentDao) Resolve(id uint) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	return d.db.Model(&model.MonitorIncident{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":      "resolved",
		"resolved_at": now,
	}).Error
}

func (d *IncidentDao) Delete(id uint) error {
	return d.db.Delete(&model.MonitorIncident{}, id).Error
}