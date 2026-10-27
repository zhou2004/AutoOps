package service

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"dodevops-api/api/cmdb/model"
	configmodel "dodevops-api/api/configcenter/model"
	"dodevops-api/common/util"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"
)

func AssetExists(db *gorm.DB, assetType string, assetID uint) error {
	var target interface{}
	switch assetType {
	case "host":
		target = &model.CmdbHost{}
	case "physical":
		target = &model.CmdbPhysicalMachine{}
	case "network":
		target = &model.CmdbNetworkDevice{}
	default:
		return errors.New("不支持的资产类型")
	}
	if assetID == 0 {
		return errors.New("资产 ID 无效")
	}
	var count int64
	if err := db.Model(target).Where("id = ?", assetID).Count(&count).Error; err != nil {
		return err
	}
	if count != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func ValidateAssetConnection(p *model.CmdbAssetConnection) error {
	p.Name = strings.TrimSpace(p.Name)
	p.Address = strings.TrimSpace(p.Address)
	p.Username = strings.TrimSpace(p.Username)
	if p.Name == "" || len(p.Name) > 100 {
		return errors.New("连接名称必填，最多 100 字节")
	}
	if p.Protocol != "ssh" && p.Protocol != "vnc" && p.Protocol != "rdp" {
		return errors.New("协议仅支持 SSH、VNC、RDP")
	}
	if p.AssetType == "network" && p.Protocol != "ssh" {
		return errors.New("网络设备当前支持 SSH 连接")
	}
	if !validEndpoint(p.Address) {
		return errors.New("连接地址必须是 IP 或主机名，不包含协议、端口或路径")
	}
	if p.Port == 0 {
		p.Port = map[string]int{"ssh": 22, "vnc": 5900, "rdp": 3389}[p.Protocol]
	}
	if p.Port < 1 || p.Port > 65535 {
		return errors.New("端口必须在 1–65535 之间")
	}
	if len(p.Username) > 128 || len(p.Domain) > 128 {
		return errors.New("用户名或域名过长")
	}
	if p.CredentialID == 0 {
		return errors.New("请选择连接凭据")
	}
	if p.Security == "" {
		p.Security = "nla"
	}
	if p.Security != "nla" && p.Security != "tls" && p.Security != "rdp" {
		return errors.New("RDP 安全模式无效")
	}
	if p.Protocol == "ssh" && p.ReadOnly {
		return errors.New("SSH 终端不支持只读模式；请使用图形连接的只读选项")
	}
	if p.Protocol == "ssh" || p.HostKey != "" {
		raw, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(p.HostKey, "SHA256:"))
		if !strings.HasPrefix(p.HostKey, "SHA256:") || err != nil || len(raw) != 32 {
			return errors.New("请获取并确认 SSH 主机的 SHA256 指纹")
		}
	}
	return nil
}

func validEndpoint(address string) bool {
	if ip := net.ParseIP(address); ip != nil {
		return !ip.IsUnspecified() && !ip.IsMulticast()
	}
	if len(address) == 0 || len(address) > 253 {
		return false
	}
	for _, label := range strings.Split(address, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, r := range label {
			if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-') {
				return false
			}
		}
	}
	return true
}

// DiscoverAssetHostKey aborts before authentication. Saving the returned
// fingerprint is an explicit administrator action; it never changes trust itself.
func DiscoverAssetHostKey(address string, port int) (string, error) {
	if !validEndpoint(address) || port < 1 || port > 65535 {
		return "", errors.New("SSH 地址或端口无效")
	}
	endpoint := net.JoinHostPort(address, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", endpoint, 5*time.Second)
	if err != nil {
		return "", errors.New("SSH 端口不可达")
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(8 * time.Second))
	var fingerprint string
	config := &ssh.ClientConfig{User: "autoops-hostkey-probe", HostKeyCallback: func(_ string, _ net.Addr, key ssh.PublicKey) error {
		fingerprint = ssh.FingerprintSHA256(key)
		return errors.New("host key probe completed before authentication")
	}}
	_, _, _, _ = ssh.NewClientConn(conn, endpoint, config)
	if fingerprint == "" {
		return "", errors.New("无法获取 SSH 主机指纹")
	}
	return fingerprint, nil
}

func GetAssetConnection(db *gorm.DB, assetType string, assetID, connectionID uint) (*model.CmdbAssetConnection, error) {
	if err := AssetExists(db, assetType, assetID); err != nil {
		return nil, err
	}
	var p model.CmdbAssetConnection
	err := db.Where("id = ? AND asset_type = ? AND asset_id = ?", connectionID, assetType, assetID).First(&p).Error
	return &p, err
}

func connectionCredential(db *gorm.DB, p *model.CmdbAssetConnection) (*configmodel.EcsAuth, error) {
	var credential configmodel.EcsAuth
	if err := db.First(&credential, p.CredentialID).Error; err != nil {
		return nil, errors.New("连接凭据不存在")
	}
	if p.Protocol != "ssh" && credential.Type != 1 {
		return nil, errors.New("VNC/RDP 需要密码类型凭据")
	}
	if p.Protocol != "vnc" && p.Username == "" && credential.Username == "" {
		return nil, errors.New("请配置登录用户名")
	}
	return &credential, nil
}

func ValidateConnectionCredential(db *gorm.DB, p *model.CmdbAssetConnection) error {
	_, err := connectionCredential(db, p)
	return err
}

// ResolveAssetSSH is shared with network configuration operations. Callers must
// authorize the requested asset/action before calling this helper.
func ResolveAssetSSH(db *gorm.DB, assetType string, assetID, connectionID uint) (*ssh.Client, error) {
	if err := AssetExists(db, assetType, assetID); err != nil {
		return nil, err
	}
	var p model.CmdbAssetConnection
	query := db.Where("asset_type = ? AND asset_id = ? AND protocol = ? AND enabled = ?", assetType, assetID, "ssh", true)
	if connectionID > 0 {
		query = query.Where("id = ?", connectionID)
	}
	if err := query.Order("id ASC").First(&p).Error; err != nil {
		return nil, errors.New("未找到已启用的 SSH 连接档案")
	}
	return DialAssetSSH(db, &p)
}

func DialAssetSSH(db *gorm.DB, p *model.CmdbAssetConnection) (*ssh.Client, error) {
	if p.Protocol != "ssh" || !p.Enabled || p.HostKey == "" {
		return nil, errors.New("SSH 连接未启用或未绑定主机指纹")
	}
	credential, err := connectionCredential(db, p)
	if err != nil {
		return nil, err
	}
	var auth ssh.AuthMethod
	switch credential.Type {
	case 1:
		auth = ssh.Password(credential.Password)
	case 2:
		key, e := ssh.ParsePrivateKey([]byte(credential.PublicKey))
		if e != nil {
			return nil, errors.New("SSH 私钥格式无效")
		}
		auth = ssh.PublicKeys(key)
	case 3:
		auth, err = util.NewSSHUtil().DefaultKeyAuth()
		if err != nil {
			return nil, errors.New("服务器默认 SSH 私钥不可用")
		}
	default:
		return nil, errors.New("不支持的凭据类型")
	}
	username := p.Username
	if username == "" {
		username = credential.Username
	}
	verify := func(_ string, _ net.Addr, key ssh.PublicKey) error {
		if subtle.ConstantTimeCompare([]byte(ssh.FingerprintSHA256(key)), []byte(p.HostKey)) != 1 {
			return errors.New("SSH 主机指纹不匹配")
		}
		return nil
	}
	config := &ssh.ClientConfig{User: username, Auth: []ssh.AuthMethod{auth}, HostKeyCallback: verify, Timeout: 10 * time.Second}
	address := net.JoinHostPort(p.Address, strconv.Itoa(p.Port))
	conn, err := net.DialTimeout("tcp", address, 10*time.Second)
	if err != nil {
		return nil, err
	}
	_ = conn.SetDeadline(time.Now().Add(15 * time.Second))
	c, chans, reqs, err := ssh.NewClientConn(conn, address, config)
	if err != nil {
		conn.Close()
		return nil, err
	}
	_ = conn.SetDeadline(time.Time{})
	return ssh.NewClient(c, chans, reqs), nil
}

func GuacdAddress() string { return strings.TrimSpace(os.Getenv("CMDB_GUACD_ADDR")) }

func CreateConnectionTicket(db *gorm.DB, userID, connectionID uint, origin string) (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	ticket := model.CmdbConnectionTicket{Hash: ticketHash(token), UserID: userID, ConnectionID: connectionID, Origin: origin, ExpiresAt: time.Now().UTC().Add(time.Minute)}
	if err := db.Where("expires_at < ?", time.Now().UTC().Add(-time.Hour)).Delete(&model.CmdbConnectionTicket{}).Error; err != nil {
		return "", err
	}
	return token, db.Create(&ticket).Error
}

func ticketHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

var ErrConnectionOrigin = errors.New("连接票据与当前页面来源不匹配，请重新打开连接")

func ConsumeConnectionTicket(db *gorm.DB, token string, originAllowed func(string) bool) (*model.CmdbConnectionTicket, error) {
	if len(token) != 43 {
		return nil, errors.New("连接票据无效或已过期")
	}
	now := time.Now().UTC()
	var ticket model.CmdbConnectionTicket
	hash := ticketHash(token)
	if err := db.Where("hash = ? AND consumed_at IS NULL AND expires_at > ?", hash, now).First(&ticket).Error; err != nil {
		return nil, err
	}
	if originAllowed == nil || !originAllowed(ticket.Origin) {
		return nil, ErrConnectionOrigin
	}
	// Origin and identity are immutable after issuance. The conditional update
	// remains the atomic claim, so concurrent API replicas cannot replay a ticket.
	// Validate the origin first so an unrelated site cannot burn a valid ticket.
	now = time.Now().UTC()
	update := db.Model(&model.CmdbConnectionTicket{}).Where("hash = ? AND consumed_at IS NULL AND expires_at > ?", hash, now).Update("consumed_at", now)
	if update.Error != nil {
		return nil, update.Error
	}
	if update.RowsAffected != 1 {
		return nil, errors.New("连接票据无效、已使用或已过期")
	}
	ticket.ConsumedAt = &now
	return &ticket, nil
}

func ProbeAssetConnection(db *gorm.DB, p *model.CmdbAssetConnection) (bool, int64, string) {
	start := time.Now()
	if p.Protocol == "ssh" {
		client, err := DialAssetSSH(db, p)
		if err != nil {
			return false, time.Since(start).Milliseconds(), "SSH 连接或认证失败，请检查地址、凭据和主机指纹"
		}
		client.Close()
		return true, time.Since(start).Milliseconds(), "SSH 连接和认证成功"
	}
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(p.Address, strconv.Itoa(p.Port)), 5*time.Second)
	if err != nil {
		return false, time.Since(start).Milliseconds(), fmt.Sprintf("%s 端口不可达，请检查服务与防火墙", strings.ToUpper(p.Protocol))
	}
	conn.Close()
	return true, time.Since(start).Milliseconds(), "TCP 端口可达；图形认证结果以实际会话为准"
}
