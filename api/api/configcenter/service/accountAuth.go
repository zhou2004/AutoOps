package service

import (
	"dodevops-api/api/configcenter/dao"
	"dodevops-api/api/configcenter/model"
	"fmt"
	"strings"
)

type AccountAuthService struct {
	dao *dao.AccountAuthDao
}

func NewAccountAuthService() *AccountAuthService {
	return &AccountAuthService{
		dao: dao.NewAccountAuthDao(),
	}
}

// Create 创建账号
func (s *AccountAuthService) Create(account *model.AccountAuth) error {
	if err := validateAccount(account, true); err != nil {
		return err
	}
	// 检查alias是否已存在
	existing, err := s.GetByAlias(account.Alias)
	if err != nil {
		return err
	}
	if existing != nil {
		return fmt.Errorf("alias '%s' 已存在", account.Alias)
	}
	return s.dao.Create(account)
}

// Update 更新账号
func (s *AccountAuthService) Update(account *model.AccountAuth) error {
	if account.ID == 0 {
		return fmt.Errorf("账号 ID 不能为空")
	}
	if err := validateAccount(account, false); err != nil {
		return err
	}
	return s.dao.Update(account)
}

func validateAccount(account *model.AccountAuth, requirePassword bool) error {
	account.Alias = strings.TrimSpace(account.Alias)
	account.Host = strings.TrimSpace(account.Host)
	account.Name = strings.TrimSpace(account.Name)
	account.Remark = strings.TrimSpace(account.Remark)
	if account.Alias == "" || account.Host == "" || account.Name == "" {
		return fmt.Errorf("账号别名、服务地址和用户名不能为空")
	}
	if account.Port < 1 || account.Port > 65535 {
		return fmt.Errorf("服务端口必须是 1 到 65535")
	}
	if account.Type < 1 || account.Type > 6 {
		return fmt.Errorf("账号类型无效")
	}
	if requirePassword && account.Password == "" {
		return fmt.Errorf("密码不能为空")
	}
	return nil
}

// Delete 删除账号
func (s *AccountAuthService) Delete(id uint) error {
	return s.dao.Delete(id)
}

// GetByID 根据ID查询账号
func (s *AccountAuthService) GetByID(id uint) (*model.AccountAuth, error) {
	return s.dao.GetByID(id)
}

// List 获取账号列表
func (s *AccountAuthService) List() ([]model.AccountAuth, error) {
	return s.dao.List()
}

// ListWithPage 获取账号列表（分页）
func (s *AccountAuthService) ListWithPage(page, pageSize int) ([]model.AccountAuth, int64, error) {
	return s.dao.ListWithPage(page, pageSize)
}

// DecryptPassword 解密密码
func (s *AccountAuthService) DecryptPassword(id uint) (string, error) {
	account, err := s.dao.GetByID(id)
	if err != nil {
		return "", err
	}
	return account.DecryptPassword()
}

// GetByType 根据类型查询账号
func (s *AccountAuthService) GetByType(accountType string) ([]model.AccountAuth, error) {
	return s.dao.GetByType(accountType)
}

// GetByAlias 根据别名查询账号
func (s *AccountAuthService) GetByAlias(alias string) (*model.AccountAuth, error) {
	return s.dao.GetByAlias(alias)
}
