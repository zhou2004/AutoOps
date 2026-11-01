package service

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	cmdbservice "dodevops-api/api/cmdb/service"
	"dodevops-api/api/database/model"
	"dodevops-api/common/util"
	"golang.org/x/crypto/ssh"
)

func (s *Service) configureFencing(ctx context.Context, c *model.Cluster, op model.Operation) error {
	if c.Engine != "mysql" {
		return errors.New("独立 SSH 隔离钩子仅适用于 MySQL Orchestrator")
	}
	targets := map[string]any{}
	nodes := []model.Member{}
	managers := 0
	for _, m := range c.Members {
		if m.Kind != "data" && m.Kind != "orchestrator" {
			continue
		}
		if !m.Managed || m.HostID == 0 || !unitPattern.MatchString(m.Service) {
			return errors.New("隔离只能配置在已确认服务归属的受管节点")
		}
		profile, e := cmdbservice.NewDatabaseAutomationService(s.DB).HostProfile(m.HostID)
		if e != nil {
			return e
		}
		if m.Kind == "data" {
			targets[fmt.Sprintf("%s:%d", m.Address, m.Port)] = map[string]any{"address": profile.Address, "sshPort": profile.Port, "username": "root", "service": m.Service}
		} else {
			managers++
		}
		nodes = append(nodes, m)
	}
	if managers != 3 {
		return errors.New("高可用隔离要求三个 Orchestrator 控制节点")
	}
	sec, e := s.secrets(c)
	if e != nil {
		return e
	}
	keyPEM := sec.Fencing
	if keyPEM == "" {
		_, private, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return err
		}
		key, err := ssh.MarshalPrivateKey(private, "autoops-fence-"+c.Name)
		if err != nil {
			return err
		}
		keyPEM = string(pem.EncodeToMemory(key))
	}
	signer, e := ssh.ParsePrivateKey([]byte(keyPEM))
	if e != nil {
		return e
	}
	pubString := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey())))
	// Persist the generated private key encrypted so subsequent configuration
	// remains recoverable. It is never exposed in the cluster JSON or plan.
	sec.Fencing = keyPEM
	encrypted, e := util.AESEncrypt(string(JSON(sec)))
	if e != nil {
		return e
	}
	if e = s.DB.Model(&model.Cluster{}).Where("id = ?", c.ID).Update("credentials", encrypted).Error; e != nil {
		return e
	}
	c.Credentials = encrypted
	return s.step(op, "安装并验证独立隔离执行器", func(log func(string)) error {
		dir, e := os.MkdirTemp("", "autoops-fence-")
		if e != nil {
			return e
		}
		defer os.RemoveAll(dir)
		hosts := map[uint]bool{}
		for _, m := range nodes {
			hosts[m.HostID] = true
		}
		if _, e = cmdbservice.NewDatabaseAutomationService(s.DB).InventoryHosts(sortedKeys(hosts), dir); e != nil {
			return e
		}
		known, e := os.ReadFile(filepath.Join(dir, "known_hosts"))
		if e != nil {
			return e
		}
		_, e = s.ansible(ctx, c, nodes, "fencing.yml", map[string]any{"fencing_targets": targets, "fencing_public_key": pubString, "fencing_private_key": keyPEM, "fencing_known_hosts": string(known)}, log)
		if e != nil {
			return e
		}
		if e = waitUntil(ctx, 45*time.Second, func() bool { _, err := s.orchestrator(ctx, c, "leader-check"); return err == nil }); e != nil {
			return errors.New("隔离配置已安装，但 Orchestrator 多数派尚未恢复")
		}
		policy := object(c.Policy)
		policy["fencingInstalled"] = true
		policy["fencingMembersHash"] = fencingMembersHash(c)
		policy["fencingChannel"] = "CMDB SSH with pinned host keys and forced command"
		c.Policy = JSON(policy)
		return s.DB.Model(&model.Cluster{}).Where("id = ?", c.ID).Updates(map[string]any{"policy": c.Policy, "ha_validated_at": nil}).Error
	})
}

func fencingMembersHash(c *model.Cluster) string {
	// Map encoding is deterministic; membership order is not part of identity.
	members := map[uint]any{}
	for _, m := range c.Members {
		if m.Kind == "data" || m.Kind == "orchestrator" {
			members[m.ID] = []any{m.HostID, m.Address, m.Port, m.Service, m.Kind}
		}
	}
	return fmt.Sprintf("%x", sha256.Sum256(JSON(members)))
}
