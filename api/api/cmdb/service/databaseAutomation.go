package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"dodevops-api/api/cmdb/model"
	configmodel "dodevops-api/api/configcenter/model"
	"dodevops-api/common/util"
	"gorm.io/gorm"
)

type DatabaseDeployRequest struct {
	Name         string `json:"name"`
	Engine       string `json:"engine"`
	Mode         string `json:"mode"`
	GroupID      uint   `json:"groupId"`
	HostIDs      []uint `json:"hostIds"`
	ManagerIDs   []uint `json:"managerIds"`
	Port         int    `json:"port"`
	Password     string `json:"password"`
	AutoRecovery bool   `json:"autoRecovery"`
}
type databaseSecrets struct {
	Password     string `json:"password"`
	Replication  string `json:"replication"`
	Topology     string `json:"topology"`
	HTTP         string `json:"http"`
	AutoRecovery bool   `json:"autoRecovery"`
}
type DatabaseAutomationService struct{ db *gorm.DB }

func NewDatabaseAutomationService(db *gorm.DB) *DatabaseAutomationService {
	return &DatabaseAutomationService{db: db}
}

var deploymentNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$`)

func randomDatabaseSecret() (string, error) {
	b := make([]byte, 16)
	_, e := rand.Read(b)
	return hex.EncodeToString(b), e
}
func validateDatabaseDeploy(r *DatabaseDeployRequest) error {
	if !deploymentNamePattern.MatchString(r.Name) {
		return errors.New("名称须为 1–64 位字母、数字、下划线或连字符")
	}
	if r.Engine != "mysql" && r.Engine != "postgresql" && r.Engine != "redis" {
		return errors.New("仅支持 MySQL、PostgreSQL、Redis")
	}
	if r.Mode != "standalone" && r.Mode != "replication" {
		return errors.New("部署模式必须为 standalone 或 replication")
	}
	if len(r.HostIDs) < 1 || len(r.HostIDs) > 9 || (r.Mode == "standalone" && len(r.HostIDs) != 1) || (r.Mode == "replication" && len(r.HostIDs) < 2) {
		return errors.New("单机需要 1 台主机，主从需要 2–9 台，第一台为初始主库")
	}
	if r.Engine != "mysql" && r.Mode != "standalone" {
		return errors.New("当前主从自动部署支持 MySQL；PostgreSQL 和 Redis 先提供单机安装与备份")
	}
	if len(r.ManagerIDs) != 0 && (r.Engine != "mysql" || r.Mode != "replication" || (len(r.ManagerIDs) != 1 && len(r.ManagerIDs) != 3)) {
		return errors.New("MySQL 主从可选择 1 台或 3 台 orchestrator 管理主机")
	}
	if r.AutoRecovery && len(r.ManagerIDs) == 0 {
		return errors.New("自动恢复需要部署 orchestrator")
	}
	if r.Port == 0 {
		r.Port = map[string]int{"mysql": 3306, "postgresql": 5432, "redis": 6379}[r.Engine]
	}
	if r.Port < 1024 || r.Port > 65535 {
		return errors.New("数据库端口必须在 1024–65535 之间")
	}
	if r.GroupID == 0 {
		return errors.New("必须选择资产业务分组")
	}
	// The upstream installer is shell based. Restrict credentials until all downstream
	// templates have been migrated; generated internal passwords are hex only.
	if len(r.Password) < 12 || len(r.Password) > 128 || !regexp.MustCompile(`^[A-Za-z0-9_!@%+=.,:-]+$`).MatchString(r.Password) {
		return errors.New("密码需 12–128 位，可用字母、数字和 _!@%+=.,:-")
	}
	for _, ids := range [][]uint{r.HostIDs, r.ManagerIDs} {
		seen := map[uint]bool{}
		for _, id := range ids {
			if id == 0 || seen[id] {
				return errors.New("主机 ID 不得为空或重复")
			}
			seen[id] = true
		}
	}
	return nil
}
func decodeIDs(value string) []uint {
	var ids []uint
	_ = json.Unmarshal([]byte(value), &ids)
	return ids
}
func (s *DatabaseAutomationService) Deployment(id uint) (*model.DatabaseDeployment, error) {
	var d model.DatabaseDeployment
	if id == 0 || s.db.First(&d, id).Error != nil {
		return nil, errors.New("数据库部署不存在")
	}
	return &d, nil
}
func (s *DatabaseAutomationService) secrets(d *model.DatabaseDeployment) (*databaseSecrets, error) {
	raw, e := util.AESDecrypt(d.Secrets)
	if e != nil {
		return nil, errors.New("部署凭据解密失败")
	}
	var v databaseSecrets
	if json.Unmarshal([]byte(raw), &v) != nil {
		return nil, errors.New("部署凭据无效")
	}
	return &v, nil
}
func (s *DatabaseAutomationService) Create(r DatabaseDeployRequest, user uint) (*model.DatabaseDeployment, *model.DatabaseAutomationRun, error) {
	if err := validateDatabaseDeploy(&r); err != nil {
		return nil, nil, err
	}
	var group model.CmdbGroup
	if s.db.First(&group, r.GroupID).Error != nil {
		return nil, nil, errors.New("业务分组不存在")
	}
	for _, id := range append(append([]uint{}, r.HostIDs...), r.ManagerIDs...) {
		if _, err := s.hostConnection(id); err != nil {
			return nil, nil, err
		}
	}
	if _, err := databaseAutomationRoot(); err != nil {
		return nil, nil, err
	}
	secrets := databaseSecrets{Password: r.Password, AutoRecovery: r.AutoRecovery}
	var err error
	for _, p := range []*string{&secrets.Replication, &secrets.Topology, &secrets.HTTP} {
		*p, err = randomDatabaseSecret()
		if err != nil {
			return nil, nil, err
		}
	}
	raw, _ := json.Marshal(secrets)
	encrypted, err := util.AESEncrypt(string(raw))
	if err != nil {
		return nil, nil, errors.New("部署凭据加密失败")
	}
	nodes, _ := json.Marshal(r.HostIDs)
	managers, _ := json.Marshal(r.ManagerIDs)
	d := model.DatabaseDeployment{Name: r.Name, Engine: r.Engine, Mode: r.Mode, GroupID: r.GroupID, Nodes: string(nodes), Managers: string(managers), Port: r.Port, Version: map[string]string{"mysql": "8.0.24", "postgresql": "distribution", "redis": "distribution"}[r.Engine], Secrets: encrypted, Status: "pending", CreatedBy: user}
	run := model.DatabaseAutomationRun{Action: "install", Status: "queued", UserID: user}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&d).Error; err != nil {
			return errors.New("创建失败，部署名称可能已存在")
		}
		run.DeploymentID = d.ID
		if err := tx.Create(&run).Error; err != nil {
			return err
		}
		if err := tx.Create(&model.DatabaseAutomationLock{Key: fmt.Sprintf("deployment:%d", d.ID), RunID: run.ID}).Error; err != nil {
			return err
		}
		// Installation claims a database port permanently, preventing concurrent/new
		// deployments from modifying an already managed instance.
		for _, id := range r.HostIDs {
			// Each engine has a fixed service name and data directory, so different
			// ports on the same host must not bypass installation exclusion.
			if err := tx.Create(&model.DatabaseAutomationLock{Key: fmt.Sprintf("host:%d:engine:%s", id, r.Engine), RunID: run.ID}).Error; err != nil {
				return errors.New("目标主机已被另一个同类型数据库部署占用")
			}
			if err := tx.Create(&model.DatabaseAutomationLock{Key: fmt.Sprintf("host:%d:port:%d", id, r.Port), RunID: run.ID}).Error; err != nil {
				return errors.New("目标主机端口已被另一个部署占用")
			}
		}
		for _, id := range r.ManagerIDs {
			if err := tx.Create(&model.DatabaseAutomationLock{Key: fmt.Sprintf("host:%d:orchestrator", id), RunID: run.ID}).Error; err != nil {
				return errors.New("orchestrator 主机已属于另一个部署")
			}
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	go s.execute(d, run)
	return &d, &run, nil
}
func (s *DatabaseAutomationService) Start(id, user uint, action, confirm string) (*model.DatabaseAutomationRun, error) {
	d, err := s.Deployment(id)
	if err != nil {
		return nil, err
	}
	if confirm != d.Name {
		return nil, errors.New("请输入完整部署名称确认操作")
	}
	if action != "backup" && action != "retry-install" {
		return nil, errors.New("不支持的 Ansible 操作")
	}
	if action == "backup" && d.Status != "ready" {
		return nil, errors.New("只有已就绪部署可以备份")
	}
	if action == "retry-install" && d.Status != "failed" && d.Status != "registration_failed" {
		return nil, errors.New("仅可重试失败的安装")
	}
	run, err := s.acquire(d.ID, user, action)
	if err != nil {
		return nil, err
	}
	go s.execute(*d, *run)
	return run, nil
}
func (s *DatabaseAutomationService) acquire(id, user uint, action string) (*model.DatabaseAutomationRun, error) {
	r := model.DatabaseAutomationRun{DeploymentID: id, Action: action, Status: "queued", UserID: user}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if e := tx.Create(&r).Error; e != nil {
			return e
		}
		if e := tx.Create(&model.DatabaseAutomationLock{Key: fmt.Sprintf("deployment:%d", id), RunID: r.ID}).Error; e != nil {
			return errors.New("该部署已有运行中的操作；请先检查执行记录")
		}
		return nil
	})
	return &r, err
}
func (s *DatabaseAutomationService) finish(run model.DatabaseAutomationRun, status, log, artifact string) {
	now := time.Now().UTC()
	// Keep the lock if durable completion fails; releasing it would allow a second
	// mutation while the persisted run still appears active.
	_ = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&run).Updates(map[string]any{"status": status, "log": log, "artifact": artifact, "finished_at": now}).Error; err != nil {
			return err
		}
		return tx.Where("`key` = ? AND run_id = ?", fmt.Sprintf("deployment:%d", run.DeploymentID), run.ID).Delete(&model.DatabaseAutomationLock{}).Error
	})
}
func (s *DatabaseAutomationService) hostConnection(id uint) (*model.CmdbAssetConnection, error) {
	var host model.CmdbHost
	if s.db.First(&host, id).Error != nil {
		return nil, errors.New("部署主机不存在")
	}
	var p model.CmdbAssetConnection
	if s.db.Where("asset_type = ? AND asset_id = ? AND protocol = ? AND enabled = ?", "host", id, "ssh", true).Order("id").First(&p).Error != nil || p.HostKey == "" {
		return nil, fmt.Errorf("主机 %d 需要已启用且已确认指纹的 SSH 连接档案", id)
	}
	return &p, nil
}
func (s *DatabaseAutomationService) registerAssets(d *model.DatabaseDeployment, sec *databaseSecrets) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, id := range decodeIDs(d.Nodes) {
			var existing model.CmdbSQL
			err := tx.Where("deployment_id = ? AND host_id = ?", d.ID, id).First(&existing).Error
			if err == nil {
				continue
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			p, err := s.hostConnection(id)
			if err != nil {
				return err
			}
			kind := map[string]int{"mysql": 1, "postgresql": 2, "redis": 3}[d.Engine]
			username := map[string]string{"mysql": "root", "postgresql": "postgres", "redis": "default"}[d.Engine]
			account := configmodel.AccountAuth{Alias: fmt.Sprintf("%s-%d", d.Name, id), Host: p.Address, Port: d.Port, Name: username, Password: sec.Password, Type: kind}
			if err := account.EncryptPassword(); err != nil {
				return err
			}
			if err := tx.Create(&account).Error; err != nil {
				return err
			}
			source := model.CmdbSQL{Name: account.Alias, Type: kind, AccountID: account.ID, GroupID: d.GroupID, TLSMode: "disable", DeploymentID: d.ID, HostID: id, DefaultDB: map[string]string{"mysql": "mysql", "postgresql": "postgres", "redis": "0"}[d.Engine]}
			if err := tx.Create(&source).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
func databaseAutomationRoot() (string, error) {
	root := os.Getenv("AUTOOPS_DATABASE_AUTOMATION_DIR")
	if root == "" {
		root = "automation/database"
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if _, err = os.Stat(filepath.Join(root, "playbooks", "install.yml")); err != nil {
		return "", errors.New("数据库 Ansible 安装资源未部署")
	}
	return root, nil
}
func databaseInstancePath(host string, port int) (string, error) {
	if host == "" || strings.ContainsAny(host, "/\\?#%\r\n \t") || port < 1 || port > 65535 {
		return "", errors.New("数据库实例地址无效")
	}
	// Restrict path segments while permitting IPv6 and DNS names.
	if net.ParseIP(host) == nil && !regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9.-]*$`).MatchString(host) {
		return "", errors.New("数据库实例地址无效")
	}
	return host + "/" + strconv.Itoa(port), nil
}
