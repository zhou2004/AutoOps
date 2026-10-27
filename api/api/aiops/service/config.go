package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	model "dodevops-api/api/aiops/model"
	cmdbdao "dodevops-api/api/cmdb/dao"
	"gorm.io/gorm"
)

type Service struct{ DB *gorm.DB }

func New(db *gorm.DB) *Service { return &Service{DB: db} }

var Modules = []string{"cmdb", "k8s", "monitor", "task", "application", "configcenter", "system", "tools"}

func validModules(values []string) bool {
	if len(values) == 0 || len(values) > len(Modules) {
		return false
	}
	seen := map[string]bool{}
	for _, v := range values {
		valid := false
		for _, m := range Modules {
			if v == m {
				valid = true
			}
		}
		if !valid || seen[v] {
			return false
		}
		seen[v] = true
	}
	return true
}
func (s *Service) IsAdmin(ctx context.Context, user uint) (bool, error) {
	return cmdbdao.NewCmdbAssetPermissionDao(s.DB.WithContext(ctx)).IsAdministrator(user)
}
func (s *Service) Active(ctx context.Context, user uint) error {
	ok, err := cmdbdao.NewCmdbAssetPermissionDao(s.DB.WithContext(ctx)).UserActive(user)
	if err != nil || !ok {
		return errors.New("Account unavailable / 账号不可用")
	}
	return nil
}
func dto(c model.Config, admin bool) model.ConfigDTO {
	d := model.ConfigDTO{Config: c, Modules: []string{}, Steps: []model.Step{}, HasKey: c.Secret != ""}
	_ = json.Unmarshal([]byte(c.Modules), &d.Modules)
	_ = json.Unmarshal([]byte(c.Steps), &d.Steps)
	d.Secret = ""
	if !admin {
		d.BaseURL = ""
	}
	return d
}
func (s *Service) List(ctx context.Context, admin bool) ([]model.ConfigDTO, error) {
	items := []model.Config{}
	q := s.DB.WithContext(ctx).Order("id DESC")
	if !admin {
		q = q.Where("enabled = ?", true)
	}
	if err := q.Find(&items).Error; err != nil {
		return nil, err
	}
	out := []model.ConfigDTO{}
	for _, c := range items {
		out = append(out, dto(c, admin))
	}
	return out, nil
}
func (s *Service) Save(ctx context.Context, id uint, d model.ConfigDTO) (model.ConfigDTO, error) {
	c := d.Config
	c.ID = id
	c.Name = strings.TrimSpace(c.Name)
	c.BaseURL = strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if c.Name == "" || len(c.Name) > 100 || len(c.Description) > 1000 || len(c.Instructions) > 16000 || len(d.APIKey) > 8192 || len(c.Model) > 200 {
		return d, errors.New("Invalid configuration length / 配置长度不合法")
	}
	if c.TimeoutSeconds == 0 {
		c.TimeoutSeconds = 180
	}
	if c.TimeoutSeconds < 10 || c.TimeoutSeconds > 600 {
		return d, errors.New("Timeout must be 10–600 seconds / 超时范围 10–600 秒")
	}
	switch c.Kind {
	case "model":
		if c.Provider != "openai" && c.Provider != "ollama" && c.Provider != "harness" {
			return d, errors.New("Unsupported model provider / 不支持的模型协议")
		}
		if strings.TrimSpace(c.Model) == "" {
			return d, errors.New("Model ID is required / 请输入模型 ID")
		}
		if err := validateURL(c.BaseURL); err != nil {
			return d, err
		}
	case "skill":
		if !validModules(d.Modules) || strings.TrimSpace(c.Instructions) == "" {
			return d, errors.New("Skill needs valid modules and instructions / Skill 需要模块和指令")
		}
		c.Provider = ""
		c.BaseURL = ""
		d.APIKey = ""
	case "workflow":
		if c.Provider == "internal" {
			if len(d.Steps) == 0 || len(d.Steps) > 8 {
				return d, errors.New("Workflow needs 1–8 steps / 工作流需要 1–8 个步骤")
			}
			for _, step := range d.Steps {
				if strings.TrimSpace(step.Name) == "" || len(step.Name) > 100 || len(step.Instructions) > 8000 || step.SkillID == 0 && strings.TrimSpace(step.Instructions) == "" {
					return d, errors.New("Invalid workflow step / 工作流步骤不完整")
				}
				if step.SkillID > 0 {
					var count int64
					if err := s.DB.WithContext(ctx).Model(&model.Config{}).Where("id = ? AND kind = ?", step.SkillID, "skill").Count(&count).Error; err != nil || count == 0 {
						return d, errors.New("Skill does not exist / Skill 不存在")
					}
				}
			}
			c.BaseURL = ""
			d.APIKey = ""
		} else if c.Provider == "dify" || c.Provider == "webhook" {
			if err := validateURL(c.BaseURL); err != nil {
				return d, err
			}
		} else {
			return d, errors.New("Unsupported workflow provider / 不支持的工作流协议")
		}
	default:
		return d, errors.New("Invalid configuration kind / 配置类型不合法")
	}
	var previous model.Config
	if id > 0 {
		if err := s.DB.WithContext(ctx).First(&previous, id).Error; err != nil {
			return d, errors.New("Configuration not found / 配置不存在")
		}
		if previous.Kind != c.Kind {
			return d, errors.New("Configuration kind cannot change / 不能更改配置类型")
		}
		c.CreatedAt = previous.CreatedAt
		c.Secret = previous.Secret
	} else {
		c.Secret = ""
	}
	if d.ClearKey {
		c.Secret = ""
	}
	if d.APIKey != "" {
		secret, err := sealSecret(d.APIKey)
		if err != nil {
			return d, err
		}
		c.Secret = secret
	}
	if c.Kind == "skill" || c.Provider == "internal" {
		c.Secret = ""
	}
	modules, _ := json.Marshal(d.Modules)
	steps, _ := json.Marshal(d.Steps)
	c.Modules = string(modules)
	c.Steps = string(steps)
	c.UpdatedAt = time.Time{}
	// Explicit fields prevent clients from overriding persistence timestamps and credentials.
	if id == 0 {
		c.CreatedAt = previous.CreatedAt
		c.UpdatedAt = previous.UpdatedAt
	}
	if err := s.DB.WithContext(ctx).Save(&c).Error; err != nil {
		return d, errors.New("Failed to save configuration / 保存配置失败")
	}
	return dto(c, true), nil
}
func (s *Service) Delete(ctx context.Context, id uint) error {
	var c model.Config
	if err := s.DB.WithContext(ctx).First(&c, id).Error; err != nil {
		return errors.New("Configuration not found / 配置不存在")
	}
	if c.Kind == "skill" {
		var workflows []model.Config
		if err := s.DB.WithContext(ctx).Where("kind = ?", "workflow").Find(&workflows).Error; err != nil {
			return err
		}
		for _, w := range workflows {
			var steps []model.Step
			_ = json.Unmarshal([]byte(w.Steps), &steps)
			for _, step := range steps {
				if step.SkillID == id {
					return fmt.Errorf("Skill is used by workflow #%d / 请先移除工作流引用", w.ID)
				}
			}
		}
	}
	return s.DB.WithContext(ctx).Delete(&c).Error
}
func (s *Service) enabled(ctx context.Context, id uint, kind string) (model.Config, error) {
	var c model.Config
	err := s.DB.WithContext(ctx).Where("id = ? AND kind = ? AND enabled = ?", id, kind, true).First(&c).Error
	if err != nil {
		return c, fmt.Errorf("Enabled %s not found / 配置不存在或已停用", kind)
	}
	return c, nil
}
