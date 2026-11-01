package service

import (
	"dodevops-api/api/cmdb/dao"
	"dodevops-api/api/cmdb/model"
	"errors"
	"strconv"
	"strings"
)

type CmdbSQLService struct {
	dao *dao.CmdbSQLDao
}

func NewCmdbSQLService(dao *dao.CmdbSQLDao) *CmdbSQLService {
	return &CmdbSQLService{dao: dao}
}

// CreateDatabase 创建数据库记录
func (s *CmdbSQLService) CreateDatabase(db *model.CmdbSQL) error {
	db.Retired = false
	db.ClusterID, db.EndpointID, db.ConnectionMode = 0, 0, "direct"
	db.DeploymentID, db.HostID = 0, 0 // Managed bindings are created only by the deployment service.
	if err := s.validate(db); err != nil {
		return err
	}
	return s.dao.Create(db)
}

// UpdateDatabase 更新数据库记录
func (s *CmdbSQLService) UpdateDatabase(db *model.CmdbSQL) error {
	existing, err := s.dao.GetByID(db.ID)
	if err != nil {
		return errors.New("数据源不存在")
	}
	db.Retired = existing.Retired
	db.ClusterID, db.EndpointID, db.ConnectionMode = existing.ClusterID, existing.EndpointID, existing.ConnectionMode
	db.DeploymentID, db.HostID = existing.DeploymentID, existing.HostID
	if (existing.DeploymentID != 0 || existing.ClusterID != 0) && (db.AccountID != existing.AccountID || db.Type != existing.Type) {
		return errors.New("自动部署数据源不能更换连接账号或类型")
	}
	if err := s.validate(db); err != nil {
		return err
	}
	return s.dao.Update(db)
}

func (s *CmdbSQLService) validate(database *model.CmdbSQL) error {
	database.Name = strings.TrimSpace(database.Name)
	database.DefaultDB = strings.TrimSpace(database.DefaultDB)
	database.TLSMode = strings.ToLower(strings.TrimSpace(database.TLSMode))
	database.Tags = strings.TrimSpace(database.Tags)
	database.Description = strings.TrimSpace(database.Description)
	if database.Name == "" || len(database.Name) > 100 {
		return errors.New("数据源名称不能为空且不能超过 100 个字符")
	}
	if database.Type < 1 || database.Type > 3 {
		return errors.New("仅支持 MySQL、PostgreSQL 和 Redis")
	}
	if database.AccountID == 0 || database.GroupID == 0 {
		return errors.New("连接账号和业务分组不能为空")
	}
	account, err := s.dao.GetAccount(database.AccountID)
	if err != nil {
		return errors.New("连接账号不存在")
	}
	if account.Type != database.Type {
		return errors.New("连接账号类型与数据源类型不一致")
	}
	if strings.TrimSpace(account.Host) == "" || account.Port < 1 || account.Port > 65535 {
		return errors.New("连接账号的地址或端口无效")
	}
	exists, err := s.dao.GroupExists(database.GroupID)
	if err != nil || !exists {
		return errors.New("业务分组不存在")
	}
	if len(database.DefaultDB) > 128 {
		return errors.New("默认数据库不能超过 128 个字符")
	}
	if database.Type == 3 && database.DefaultDB != "" {
		number, err := strconv.Atoi(database.DefaultDB)
		if err != nil || number < 0 || number > 1024 {
			return errors.New("Redis DB 编号必须是 0 到 1024")
		}
	}
	normalizedTLS, err := normalizeTLS(database.Type, database.TLSMode)
	if err != nil {
		return err
	}
	if database.ConnectionMode == "cluster" && database.Type == 3 && database.DefaultDB != "" && database.DefaultDB != "0" {
		mode, err := s.dao.EndpointMode(database.EndpointID, database.ClusterID)
		if err != nil {
			return err
		}
		if mode == "cluster" {
			return errors.New("Redis Cluster 数据源仅允许 DB 0")
		}
	}
	database.TLSMode = normalizedTLS
	return nil
}

// DeleteDatabase 删除数据库记录
func (s *CmdbSQLService) DeleteDatabase(id uint) error {
	return s.dao.Delete(id)
}

// GetDatabase 获取单个数据库详情
func (s *CmdbSQLService) GetDatabase(id uint) (*model.CmdbSQL, error) {
	return s.dao.GetByID(id)
}

// ListDatabases 分页查询数据库列表
func (s *CmdbSQLService) ListDatabases(page, pageSize int, scopes ...map[uint]bool) ([]model.CmdbSQL, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	return s.dao.List(page, pageSize, scopes...)
}

// GetDatabasesByAccount 根据账号查询数据库
func (s *CmdbSQLService) GetDatabasesByAccount(accountID uint, scopes ...map[uint]bool) ([]model.CmdbSQL, error) {
	return s.dao.GetByAccountID(accountID, scopes...)
}

// GetDatabasesByGroup 根据业务组查询数据库
func (s *CmdbSQLService) GetDatabasesByGroup(groupID uint, scopes ...map[uint]bool) ([]model.CmdbSQL, error) {
	return s.dao.GetByGroupID(groupID, scopes...)
}

// GetDatabasesByName 根据名称查询数据库
func (s *CmdbSQLService) GetDatabasesByName(name string, scopes ...map[uint]bool) ([]model.CmdbSQL, error) {
	return s.dao.GetByName(name, scopes...)
}

// GetDatabasesByType 根据类型查询数据库
func (s *CmdbSQLService) GetDatabasesByType(dbType int, scopes ...map[uint]bool) ([]model.CmdbSQL, error) {
	if dbType < 1 || dbType > 3 {
		return nil, errors.New("无效的数据库类型")
	}
	return s.dao.GetByType(dbType, scopes...)
}
