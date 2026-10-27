package service

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"dodevops-api/api/cmdb/dao"
	"dodevops-api/api/cmdb/model"
	"github.com/pmezard/go-difflib/difflib"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

const networkConfigMaxBytes = 4 << 20

type NetworkConfigCapability struct {
	Platform    string `json:"platform"`
	Label       string `json:"label"`
	Backup      bool   `json:"backup"`
	Restore     bool   `json:"restore"`
	Description string `json:"description"`
}

var networkConfigCapabilities = []NetworkConfigCapability{
	{"openwrt", "OpenWrt / UCI", true, true, "UCI 配置备份与恢复；恢复写入持久配置，需在维护窗口手动重载服务"},
	{"cisco", "Cisco IOS", true, false, "备份 running-config；需要 SSH exec 与读取完整配置权限"},
	{"huawei", "Huawei VRP", true, false, "备份 current-configuration；需要 SSH exec 与读取配置权限"},
	{"h3c", "H3C Comware", true, false, "备份 current-configuration；需要 SSH exec 与读取配置权限"},
	{"linux", "Linux 网络诊断", true, false, "记录接口和路由状态；此快照不能用于恢复系统配置"},
}

type NetworkConfigBackupRequest struct {
	ConnectionID uint   `json:"connectionId"`
	Platform     string `json:"platform" binding:"required"`
	Remark       string `json:"remark" binding:"max=500"`
}

type NetworkConfigRestoreRequest struct {
	ConnectionID uint   `json:"connectionId"`
	SnapshotID   uint   `json:"snapshotId" binding:"required"`
	ConfirmName  string `json:"confirmName" binding:"required"`
}

type NetworkConfigSnapshotDetail struct {
	model.CmdbNetworkConfigSnapshot
	Content string `json:"content"`
}

type NetworkConfigDiff struct {
	From    uint   `json:"from"`
	To      uint   `json:"to"`
	Diff    string `json:"diff"`
	Changed bool   `json:"changed"`
}

type NetworkConfigRestoreResult struct {
	SnapshotID         uint   `json:"snapshotId"`
	RollbackSnapshotID uint   `json:"rollbackSnapshotId"`
	Message            string `json:"message"`
}

// The injected runner keeps remote device access separate from snapshot tests.
type NetworkConfigService struct {
	dao                 *dao.NetworkConfigDao
	dial                func(uint, uint) (*ssh.Client, error)
	run                 func(context.Context, *ssh.Client, string, string) (string, error)
	authorizeConnection func(userID, assetID, connectionID uint, restore bool) (uint, error)
}

func NewNetworkConfigService(db *gorm.DB) *NetworkConfigService {
	return &NetworkConfigService{
		dao: dao.NewNetworkConfigDao(db),
		dial: func(assetID, connectionID uint) (*ssh.Client, error) {
			return ResolveAssetSSH(db, "network", assetID, connectionID)
		},
		run: runNetworkConfigCommand,
		authorizeConnection: func(userID, assetID, connectionID uint, restore bool) (uint, error) {
			var profile model.CmdbAssetConnection
			query := db.Where("asset_type = ? AND asset_id = ? AND protocol = ? AND enabled = ?", "network", assetID, "ssh", true)
			if connectionID > 0 {
				query = query.Where("id = ?", connectionID)
			}
			if err := query.Order("id ASC").First(&profile).Error; err != nil {
				return 0, errors.New("未找到该设备已启用的 SSH 连接档案")
			}
			if restore && profile.ReadOnly {
				return 0, errors.New("只读连接档案不能恢复配置")
			}
			action := "config_backup"
			if restore {
				action = "config_restore"
			}
			allowed, err := dao.NewCmdbAssetPermissionDao(db).CanUserAccess(userID, "network", assetID, action)
			if err != nil {
				return 0, errors.New("资产权限校验失败")
			}
			if !allowed {
				return 0, errors.New("没有该设备的配置操作权限")
			}
			return profile.ID, nil
		},
	}
}

func (s *NetworkConfigService) Capabilities() ([]NetworkConfigCapability, bool) {
	_, err := networkConfigCipher()
	return networkConfigCapabilities, err == nil
}

func (s *NetworkConfigService) Snapshots(networkID uint, page, size int) ([]model.CmdbNetworkConfigSnapshot, int64, error) {
	return s.dao.Snapshots(networkID, page, size)
}

func (s *NetworkConfigService) Audits(networkID uint, page, size int) ([]model.CmdbNetworkConfigAudit, int64, error) {
	return s.dao.Audits(networkID, page, size)
}

func (s *NetworkConfigService) Snapshot(networkID, snapshotID uint) (*NetworkConfigSnapshotDetail, error) {
	snapshot, err := s.dao.Snapshot(networkID, snapshotID)
	if err != nil {
		return nil, errors.New("配置版本不存在")
	}
	content, err := decryptNetworkConfig(snapshot)
	if err != nil {
		return nil, err
	}
	return &NetworkConfigSnapshotDetail{CmdbNetworkConfigSnapshot: *snapshot, Content: redactNetworkConfig(content)}, nil
}

func (s *NetworkConfigService) Diff(networkID, from, to uint) (*NetworkConfigDiff, error) {
	a, err := s.Snapshot(networkID, from)
	if err != nil {
		return nil, err
	}
	b, err := s.Snapshot(networkID, to)
	if err != nil {
		return nil, err
	}
	if a.Platform != b.Platform {
		return nil, errors.New("只能比较相同平台的配置版本")
	}
	if len(a.Content)+len(b.Content) > 512*1024 || strings.Count(a.Content+b.Content, "\n") > 6000 {
		return nil, errors.New("配置过大，请分别查看版本；在线比较上限为 512 KiB、6000 行")
	}
	diff, err := difflib.GetUnifiedDiffString(difflib.UnifiedDiff{
		A: difflib.SplitLines(a.Content), B: difflib.SplitLines(b.Content),
		FromFile: fmt.Sprintf("version-%d", from), ToFile: fmt.Sprintf("version-%d", to), Context: 3,
	})
	if err != nil {
		return nil, errors.New("配置比较失败")
	}
	return &NetworkConfigDiff{From: from, To: to, Diff: diff, Changed: a.Checksum != b.Checksum}, nil
}

var networkConfigLocks sync.Map

func lockNetworkConfig(networkID uint) (func(), error) {
	value, _ := networkConfigLocks.LoadOrStore(networkID, &sync.Mutex{})
	lock := value.(*sync.Mutex)
	if !lock.TryLock() {
		return nil, errors.New("该设备有配置任务正在运行，请稍后重试")
	}
	return lock.Unlock, nil
}

func (s *NetworkConfigService) Backup(ctx context.Context, userID, networkID uint, req NetworkConfigBackupRequest) (snapshot *model.CmdbNetworkConfigSnapshot, err error) {
	if _, err = s.dao.Device(networkID); err != nil {
		return nil, errors.New("网络设备不存在")
	}
	unlock, err := lockNetworkConfig(networkID)
	if err != nil {
		return nil, err
	}
	defer unlock()
	audit := &model.CmdbNetworkConfigAudit{NetworkID: networkID, ConnectionID: req.ConnectionID, Action: "backup", Status: "running", CreatedBy: userID}
	if err = s.dao.CreateAudit(audit); err != nil {
		return nil, errors.New("无法保存操作审计，备份已停止")
	}
	defer func() {
		status, message, snapshotID := "success", "配置已加密保存", uint(0)
		if err != nil {
			status, message = "failed", err.Error()
		}
		if snapshot != nil {
			snapshotID = snapshot.ID
		}
		if auditErr := s.dao.FinishAudit(audit, status, message, snapshotID); auditErr != nil && err == nil {
			err = errors.New("配置已备份，但完成审计写入失败，请检查任务记录")
		}
	}()
	command, err := networkConfigBackupCommand(req.Platform)
	if err != nil {
		return nil, err
	}
	if _, err = networkConfigCipher(); err != nil {
		return nil, err
	}
	req.ConnectionID, err = s.authorizeConnection(userID, networkID, req.ConnectionID, false)
	if err != nil {
		return nil, err
	}
	audit.ConnectionID = req.ConnectionID
	client, err := s.dial(networkID, req.ConnectionID)
	if err != nil {
		return nil, errors.New("SSH 连接失败，请检查连接档案、凭据及主机指纹")
	}
	if client != nil {
		defer client.Close()
	}
	content, err := s.run(ctx, client, command, "")
	if err != nil {
		return nil, err
	}
	if err = validateNetworkConfig(req.Platform, content); err != nil {
		return nil, err
	}
	return s.saveSnapshot(userID, networkID, req.ConnectionID, req.Platform, content, "manual", req.Remark)
}

func (s *NetworkConfigService) Restore(ctx context.Context, userID, networkID uint, req NetworkConfigRestoreRequest) (output *NetworkConfigRestoreResult, err error) {
	device, err := s.dao.Device(networkID)
	if err != nil {
		return nil, errors.New("网络设备不存在")
	}
	unlock, err := lockNetworkConfig(networkID)
	if err != nil {
		return nil, err
	}
	defer unlock()
	audit := &model.CmdbNetworkConfigAudit{NetworkID: networkID, ConnectionID: req.ConnectionID, SnapshotID: req.SnapshotID, Action: "restore", Status: "running", CreatedBy: userID}
	if err = s.dao.CreateAudit(audit); err != nil {
		return nil, errors.New("无法保存操作审计，恢复已停止")
	}
	defer func() {
		status, message := "success", "配置已写入；等待维护窗口手动重载服务"
		if err != nil {
			status, message = "failed", err.Error()
		}
		if output != nil {
			message = output.Message
		}
		if auditErr := s.dao.FinishAudit(audit, status, message, req.SnapshotID); auditErr != nil && err == nil {
			err = errors.New("配置已写入，但完成审计写入失败；请检查设备和任务记录")
		}
	}()
	if strings.TrimSpace(req.ConfirmName) != device.Name || device.Name == "" {
		return nil, errors.New("请输入完整设备名称确认恢复")
	}
	snapshot, err := s.dao.Snapshot(networkID, req.SnapshotID)
	if err != nil {
		return nil, errors.New("配置版本不存在或不属于当前设备")
	}
	if snapshot.Platform != "openwrt" || !snapshot.Restorable {
		return nil, errors.New("此版本不支持恢复，仅 OpenWrt UCI 配置可恢复")
	}
	content, err := decryptNetworkConfig(snapshot)
	if err != nil {
		return nil, err
	}
	if err = validateNetworkConfig(snapshot.Platform, content); err != nil {
		return nil, err
	}
	req.ConnectionID, err = s.authorizeConnection(userID, networkID, req.ConnectionID, true)
	if err != nil {
		return nil, err
	}
	audit.ConnectionID = req.ConnectionID
	client, err := s.dial(networkID, req.ConnectionID)
	if err != nil {
		return nil, errors.New("SSH 连接失败，请检查连接档案、凭据及主机指纹")
	}
	if client != nil {
		defer client.Close()
	}
	command, _ := networkConfigBackupCommand("openwrt")
	current, err := s.run(ctx, client, command, "")
	if err != nil {
		return nil, errors.New("恢复前备份失败，尚未写入设备")
	}
	if err = validateNetworkConfig("openwrt", current); err != nil {
		return nil, err
	}
	rollback, err := s.saveSnapshot(userID, networkID, req.ConnectionID, "openwrt", current, "pre_restore", fmt.Sprintf("恢复版本 %d 前的自动备份", snapshot.ID))
	if err != nil {
		return nil, errors.New("恢复前备份保存失败，尚未写入设备")
	}
	_, err = s.run(ctx, client, networkConfigRestoreCommand, content)
	if err != nil {
		return nil, fmt.Errorf("配置恢复未确认完成，请检查设备；恢复前备份版本为 %d。设备存在未提交 UCI 修改时会拒绝写入", rollback.ID)
	}
	return &NetworkConfigRestoreResult{SnapshotID: snapshot.ID, RollbackSnapshotID: rollback.ID,
		Message: fmt.Sprintf("配置已写入，恢复前备份版本为 %d；请在维护窗口手动重载相关服务", rollback.ID)}, nil
}

func (s *NetworkConfigService) saveSnapshot(userID, networkID, connectionID uint, platform, content, source, remark string) (*model.CmdbNetworkConfigSnapshot, error) {
	hash := sha256.Sum256([]byte(content))
	snapshot := &model.CmdbNetworkConfigSnapshot{
		NetworkID: networkID, ConnectionID: connectionID, Platform: platform,
		Checksum: hex.EncodeToString(hash[:]), ByteSize: len(content), Source: source, Remark: remark,
		Restorable: platform == "openwrt", CreatedBy: userID,
	}
	encrypted, err := encryptNetworkConfig(snapshot, content)
	if err != nil {
		return nil, err
	}
	snapshot.ContentCipher = encrypted
	if err := s.dao.CreateSnapshot(snapshot); err != nil {
		return nil, errors.New("配置版本保存失败")
	}
	return snapshot, nil
}

func networkConfigBackupCommand(platform string) (string, error) {
	switch platform {
	case "openwrt":
		return "test -r /etc/openwrt_release && command -v uci >/dev/null && uci export", nil
	case "cisco":
		return "show running-config", nil
	case "huawei", "h3c":
		return "display current-configuration | no-more", nil
	case "linux":
		return "ip -details address show && ip route show table all && ip -6 route show table all", nil
	default:
		return "", errors.New("请选择支持的设备平台")
	}
}

// Import first into an isolated directory to validate the entire input. UCI
// import writes packages itself (https://openwrt.org/docs/guide-user/base-system/uci).
// Never interpolate snapshot text into a command, reload services, or reboot.
const networkConfigRestoreCommand = `set -eu
test -r /etc/openwrt_release
test -z "$(uci changes)"
umask 077
autoops_config_tmp=$(mktemp -d /tmp/autoops-config.XXXXXX)
trap 'rm -rf "$autoops_config_tmp"' EXIT HUP INT TERM
mkdir "$autoops_config_tmp/config" "$autoops_config_tmp/delta"
cat > "$autoops_config_tmp/source"
uci -c "$autoops_config_tmp/config" -t "$autoops_config_tmp/delta" import < "$autoops_config_tmp/source"
test -z "$(uci changes)"
uci import < "$autoops_config_tmp/source"
`

func validateNetworkConfig(platform, content string) error {
	if len(strings.TrimSpace(content)) == 0 {
		return errors.New("设备返回空配置，未创建版本")
	}
	if len(content) > networkConfigMaxBytes {
		return errors.New("配置超过 4 MiB 上限，未创建版本")
	}
	lower := strings.ToLower(content)
	for _, marker := range []string{"--more--", "---- more ----", "% invalid input", "unrecognized command", "error: unrecognized", "permission denied"} {
		if strings.Contains(lower, marker) {
			return errors.New("设备返回分页或命令错误，请检查平台及读取配置权限")
		}
	}
	if platform == "openwrt" && !regexp.MustCompile(`(?m)^package\s+`).MatchString(content) {
		return errors.New("设备返回的内容不是完整 UCI 配置")
	}
	return nil
}

type configBoundedBuffer struct {
	buffer bytes.Buffer
	max    int
}

func (b *configBoundedBuffer) Write(p []byte) (int, error) {
	if len(p) > b.max-b.buffer.Len() {
		return 0, errors.New("配置输出超限")
	}
	return b.buffer.Write(p)
}

func runNetworkConfigCommand(parent context.Context, client *ssh.Client, command, input string) (string, error) {
	ctx, cancel := context.WithTimeout(parent, 45*time.Second)
	defer cancel()
	// Closing the transport also bounds NewSession, which can otherwise wait
	// indefinitely for a server to acknowledge opening the SSH channel.
	stopClosing := context.AfterFunc(ctx, func() { _ = client.Close() })
	defer stopClosing()
	session, err := client.NewSession()
	if err != nil {
		return "", errors.New("无法建立 SSH 命令会话")
	}
	defer session.Close()
	stdout := &configBoundedBuffer{max: networkConfigMaxBytes}
	stderr := &configBoundedBuffer{max: 64 * 1024}
	session.Stdout, session.Stderr = stdout, stderr
	if input != "" {
		session.Stdin = strings.NewReader(input)
	}
	done := make(chan error, 1)
	go func() { done <- session.Run(command) }()
	select {
	case <-ctx.Done():
		_ = client.Close()
		<-done
		return "", errors.New("配置任务超时或已取消")
	case err = <-done:
		if err != nil {
			return "", errors.New("设备命令执行失败，请检查平台、SSH exec 支持及配置权限")
		}
		return stdout.buffer.String(), nil
	}
}

func networkConfigCipher() (cipher.AEAD, error) {
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(os.Getenv("CMDB_CONFIG_ENCRYPTION_KEY")))
	if err != nil || len(key) != 32 {
		return nil, errors.New("请配置 CMDB_CONFIG_ENCRYPTION_KEY（Base64 编码的 32 字节密钥）")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.New("配置加密初始化失败")
	}
	return cipher.NewGCM(block)
}

func networkConfigAAD(snapshot *model.CmdbNetworkConfigSnapshot) []byte {
	return []byte(fmt.Sprintf("%d:%s:%s", snapshot.NetworkID, snapshot.Platform, snapshot.Checksum))
}

func encryptNetworkConfig(snapshot *model.CmdbNetworkConfigSnapshot, content string) (string, error) {
	aead, err := networkConfigCipher()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", errors.New("配置加密失败")
	}
	encrypted := aead.Seal(nonce, nonce, []byte(content), networkConfigAAD(snapshot))
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func decryptNetworkConfig(snapshot *model.CmdbNetworkConfigSnapshot) (string, error) {
	aead, err := networkConfigCipher()
	if err != nil {
		return "", err
	}
	data, err := base64.StdEncoding.DecodeString(snapshot.ContentCipher)
	if err != nil || len(data) < aead.NonceSize()+aead.Overhead() {
		return "", errors.New("配置密文损坏")
	}
	plain, err := aead.Open(nil, data[:aead.NonceSize()], data[aead.NonceSize():], networkConfigAAD(snapshot))
	if err != nil {
		return "", errors.New("配置解密失败，请检查加密密钥或版本完整性")
	}
	hash := sha256.Sum256(plain)
	if hex.EncodeToString(hash[:]) != snapshot.Checksum {
		return "", errors.New("配置完整性校验失败")
	}
	return string(plain), nil
}

var networkConfigSecretLine = regexp.MustCompile(`(?i)(password|passwd|passphrase|secret|community|token|key|preshared|pre-shared|psk|credential|authentication|\bauth\b)`)

func redactNetworkConfig(content string) string {
	lines := strings.Split(content, "\n")
	insidePrivateKey := false
	var secretQuote byte
	for i, line := range lines {
		if strings.Contains(line, "-----BEGIN ") && strings.Contains(line, "PRIVATE KEY-----") {
			insidePrivateKey = true
		}
		secret := insidePrivateKey || secretQuote != 0 || networkConfigSecretLine.MatchString(line)
		if strings.Contains(line, "-----END ") && strings.Contains(line, "PRIVATE KEY-----") {
			insidePrivateKey = false
		}
		if secret {
			lines[i] = "[敏感配置已隐藏]"
			trimmed := strings.TrimSpace(line)
			if secretQuote != 0 || strings.HasPrefix(trimmed, "option ") || strings.HasPrefix(trimmed, "list ") {
				secretQuote = configLineQuote(line, secretQuote)
			}
		}
	}
	return strings.Join(lines, "\n")
}

// UCI values may contain literal newlines. Preserve quote state so the rest of
// a multiline password is redacted as well as the option's first line.
func configLineQuote(line string, quote byte) byte {
	escaped := false
	for i := 0; i < len(line); i++ {
		ch := line[i]
		if escaped {
			escaped = false
			continue
		}
		if ch == '\\' && quote != '\'' {
			escaped = true
			continue
		}
		if quote != 0 {
			if ch == quote {
				quote = 0
			}
		} else if ch == '\'' || ch == '"' {
			quote = ch
		}
	}
	return quote
}
