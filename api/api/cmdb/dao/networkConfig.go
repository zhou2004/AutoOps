package dao

import (
	"dodevops-api/api/cmdb/model"
	"gorm.io/gorm"
)

type NetworkConfigDao struct{ db *gorm.DB }

func NewNetworkConfigDao(db *gorm.DB) *NetworkConfigDao { return &NetworkConfigDao{db: db} }

func (d *NetworkConfigDao) Device(id uint) (*model.CmdbNetworkDevice, error) {
	var device model.CmdbNetworkDevice
	err := d.db.First(&device, id).Error
	return &device, err
}

func (d *NetworkConfigDao) CreateSnapshot(snapshot *model.CmdbNetworkConfigSnapshot) error {
	return d.db.Create(snapshot).Error
}

func (d *NetworkConfigDao) Snapshot(networkID, snapshotID uint) (*model.CmdbNetworkConfigSnapshot, error) {
	var snapshot model.CmdbNetworkConfigSnapshot
	err := d.db.Where("network_id = ? AND id = ?", networkID, snapshotID).First(&snapshot).Error
	return &snapshot, err
}

func (d *NetworkConfigDao) Snapshots(networkID uint, page, size int) ([]model.CmdbNetworkConfigSnapshot, int64, error) {
	list := []model.CmdbNetworkConfigSnapshot{}
	var total int64
	query := d.db.Model(&model.CmdbNetworkConfigSnapshot{}).Where("network_id = ?", networkID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size = configPagination(page, size)
	err := query.Omit("content_cipher").Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&list).Error
	return list, total, err
}

func (d *NetworkConfigDao) CreateAudit(audit *model.CmdbNetworkConfigAudit) error {
	return d.db.Create(audit).Error
}

func (d *NetworkConfigDao) FinishAudit(audit *model.CmdbNetworkConfigAudit, status, message string, snapshotID uint) error {
	return d.db.Model(audit).Updates(map[string]interface{}{"status": status, "message": message, "snapshot_id": snapshotID, "connection_id": audit.ConnectionID}).Error
}

func (d *NetworkConfigDao) Audits(networkID uint, page, size int) ([]model.CmdbNetworkConfigAudit, int64, error) {
	list := []model.CmdbNetworkConfigAudit{}
	var total int64
	query := d.db.Model(&model.CmdbNetworkConfigAudit{}).Where("network_id = ?", networkID)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	page, size = configPagination(page, size)
	err := query.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&list).Error
	return list, total, err
}

func configPagination(page, size int) (int, int) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	if page > 100000 {
		page = 100000
	}
	return page, size
}
