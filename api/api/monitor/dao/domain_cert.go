package dao

import (
	"dodevops-api/api/monitor/model"
	"dodevops-api/common"

	"gorm.io/gorm"
)

type DomainCertDao struct {
	db *gorm.DB
}

func NewDomainCertDao() *DomainCertDao {
	return &DomainCertDao{
		db: common.GetDB(),
	}
}

// Create 创建域名证书记录
func (d *DomainCertDao) Create(cert *model.DomainCert) error {
	return d.db.Create(cert).Error
}

// GetByID 根据ID获取记录
func (d *DomainCertDao) GetByID(id uint) (*model.DomainCert, error) {
	var cert model.DomainCert
	err := d.db.First(&cert, id).Error
	if err != nil {
		return nil, err
	}
	return &cert, nil
}

// GetByDomain 根据域名获取记录
func (d *DomainCertDao) GetByDomain(domain string) (*model.DomainCert, error) {
	var cert model.DomainCert
	err := d.db.Where("domain = ?", domain).First(&cert).Error
	if err != nil {
		return nil, err
	}
	return &cert, nil
}

// GetList 获取域名证书列表（分页）
func (d *DomainCertDao) GetList(req *model.DomainCertListReq) ([]model.DomainCert, int64, error) {
	var list []model.DomainCert
	var total int64
	query := d.db.Model(&model.DomainCert{})

	if req.Domain != "" {
		query = query.Where("domain LIKE ?", "%"+req.Domain+"%")
	}
	if req.Status > 0 {
		query = query.Where("status = ?", req.Status)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	offset := (req.Page - 1) * req.PageSize

	if err := query.Order("id DESC").Offset(offset).Limit(req.PageSize).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// GetAll 获取所有域名证书（用于检查）
func (d *DomainCertDao) GetAll() ([]model.DomainCert, error) {
	var list []model.DomainCert
	err := d.db.Find(&list).Error
	return list, err
}

// Update 更新域名证书记录
func (d *DomainCertDao) Update(cert *model.DomainCert) error {
	return d.db.Save(cert).Error
}

// Delete 删除域名证书记录
func (d *DomainCertDao) Delete(id uint) error {
	return d.db.Delete(&model.DomainCert{}, id).Error
}

// BatchDelete 批量删除
func (d *DomainCertDao) BatchDelete(ids []uint) error {
	return d.db.Delete(&model.DomainCert{}, ids).Error
}