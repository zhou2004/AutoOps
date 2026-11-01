package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	cmdb "dodevops-api/api/cmdb/model"
	cmdbservice "dodevops-api/api/cmdb/service"
	accounts "dodevops-api/api/configcenter/model"
	"dodevops-api/api/database/model"
	"dodevops-api/common/util"
	"gorm.io/gorm"
)

func (s *Service) endpointPlan(c *model.Cluster, p Parameters) ([]model.Member, bool, error) {
	proxies := []model.Member{}
	for _, m := range c.Members {
		if m.Kind == "proxy" {
			if !m.Managed {
				return nil, false, errors.New("接管代理必须使用既有入口，不覆盖其配置")
			}
			proxies = append(proxies, m)
		}
	}
	if len(proxies) > 0 {
		if len(proxies) != 2 {
			return nil, false, errors.New("SQL 入口需要两个代理")
		}
		return proxies, false, nil
	}
	if len(p.Members) != 2 {
		return nil, false, errors.New("请选择两个代理主机与独立端口")
	}
	seen := map[uint]bool{}
	for _, in := range p.Members {
		if in.HostID == 0 || seen[in.HostID] || in.Port < 1024 || in.Port > 65534 {
			return nil, false, errors.New("代理须位于两个不同主机，并预留读取端口 +1")
		}
		seen[in.HostID] = true
		profile, e := cmdbservice.NewDatabaseAutomationService(s.DB).HostProfile(in.HostID)
		if e != nil {
			return nil, false, e
		}
		m := model.Member{ClusterID: c.ID, HostID: in.HostID, Name: fmt.Sprintf("proxy-%d", in.HostID), Kind: "proxy", Address: profile.Address, Port: in.Port, Role: "proxy", Managed: true, State: "unknown", Metrics: JSON(map[string]any{}), Service: fmt.Sprintf("autoops-db-proxy-%d-%d", c.ID, in.HostID), DataDir: fmt.Sprintf("/var/lib/autoops-database/%s/proxy-%d", c.Name, in.HostID)}
		for _, node := range c.Members {
			if node.Address == m.Address && (node.Port == m.Port || node.Port == m.Port+1) {
				return nil, false, errors.New("代理端口与已有成员冲突")
			}
		}
		proxies = append(proxies, m)
	}
	return proxies, true, nil
}

func (s *Service) configureEndpoints(ctx context.Context, c *model.Cluster, op model.Operation, p Parameters) error {
	proxies, create, e := s.endpointPlan(c, p)
	if e != nil {
		return e
	}
	if create {
		if e = s.DB.Transaction(func(tx *gorm.DB) error {
			for i := range proxies {
				if e := tx.Create(&proxies[i]).Error; e != nil {
					return e
				}
			}
			return tx.Model(&model.Cluster{}).Where("id = ?", c.ID).Update("revision", gorm.Expr("revision + 1")).Error
		}); e != nil {
			return e
		}
		c.Members = append(c.Members, proxies...)
	}
	nodes := append([]model.Member{}, proxies...)
	if c.Engine == "mysql" {
		for _, m := range c.Members {
			if m.Kind == "data" {
				nodes = append(nodes, m)
			}
		}
	}
	return s.step(op, "部署双代理并核验角色路由", func(log func(string)) error {
		if _, e := s.ansible(ctx, c, nodes, "endpoint.yml", nil, log); e != nil {
			return e
		}
		if e := s.registerSources(c); e != nil {
			return e
		}
		return s.waitEndpoints(ctx, c)
	})
}

func (s *Service) registerSources(c *model.Cluster) error {
	return s.DB.Transaction(func(tx *gorm.DB) error {
		var firstAccount uint
		for i := range c.Members {
			m := &c.Members[i]
			if m.Kind != "data" {
				continue
			}
			if m.AccountID == 0 {
				user, password, e := s.memberAccount(c, *m)
				if e != nil {
					return e
				}
				a := accounts.AccountAuth{Alias: c.Name + "/" + m.Name, Host: m.Address, Port: m.Port, Name: user, Password: password, Type: map[string]int{"mysql": 1, "postgresql": 2, "redis": 3}[c.Engine]}
				if e = a.EncryptPassword(); e != nil {
					return e
				}
				if e = tx.Create(&a).Error; e != nil {
					return e
				}
				m.AccountID = a.ID
			}
			if firstAccount == 0 {
				firstAccount = m.AccountID
			}
			if m.SourceID == 0 {
				source := cmdb.CmdbSQL{Name: c.Name + "/" + m.Name, Type: map[string]int{"mysql": 1, "postgresql": 2, "redis": 3}[c.Engine], AccountID: m.AccountID, GroupID: c.GroupID, HostID: m.HostID, ClusterID: c.ID, ConnectionMode: "direct", Description: "集群受管实例", TLSMode: "disable", CreatedAt: util.HTime{Time: *nowPtr()}, UpdatedAt: util.HTime{Time: *nowPtr()}}
				if e := tx.Create(&source).Error; e != nil {
					return e
				}
				m.SourceID = source.ID
			}
			if e := tx.Model(m).Updates(map[string]any{"account_id": m.AccountID, "source_id": m.SourceID}).Error; e != nil {
				return e
			}
		}
		if len(c.Endpoints) == 0 {
			endpoints := []model.Endpoint{}
			if c.Engine == "redis" && c.Topology != "standalone" {
				addresses := []string{}
				mode := "cluster"
				kind := "data"
				if c.Topology == "redis-sentinel" {
					mode = "sentinel"
					kind = "sentinel"
				}
				for _, m := range c.Members {
					if m.Kind == kind {
						addresses = append(addresses, net.JoinHostPort(m.Address, strconv.Itoa(m.Port)))
					}
				}
				endpoint := model.Endpoint{ClusterID: c.ID, Name: c.Name, Mode: mode, Addresses: JSON(addresses), AccountID: firstAccount, MasterName: c.Name, Status: "unknown"}
				if name := text(object(c.Policy)["masterName"]); name != "" {
					endpoint.MasterName = name
				}
				if mode == "sentinel" {
					sec, e := s.secrets(c)
					if e != nil {
						return e
					}
					a := accounts.AccountAuth{Alias: c.Name + "/Sentinel", Host: c.Members[0].Address, Port: 26380, Name: "default", Password: sec.Sentinel, Type: 3}
					if e = a.EncryptPassword(); e != nil {
						return e
					}
					if e = tx.Create(&a).Error; e != nil {
						return e
					}
					endpoint.SentinelAccountID = a.ID
				}
				endpoints = append(endpoints, endpoint)
			} else {
				for _, mode := range []string{"read-write", "read-only"} {
					addresses := []string{}
					for _, m := range c.Members {
						if m.Kind == "proxy" {
							port := m.Port
							if mode == "read-only" {
								port++
							}
							addresses = append(addresses, net.JoinHostPort(m.Address, strconv.Itoa(port)))
						}
					}
					if len(addresses) > 0 {
						endpoints = append(endpoints, model.Endpoint{ClusterID: c.ID, Name: c.Name + "/" + mode, Mode: mode, Addresses: JSON(addresses), AccountID: firstAccount, Status: "unknown"})
					}
				}
			}
			for i := range endpoints {
				if e := tx.Create(&endpoints[i]).Error; e != nil {
					return e
				}
			}
			c.Endpoints = endpoints
		}
		if c.Topology == "redis-cluster" && c.Credentials != "" {
			addresses := []string{}
			for _, m := range c.Members {
				if m.Kind == "data" {
					addresses = append(addresses, net.JoinHostPort(m.Address, strconv.Itoa(m.Port)))
				}
			}
			if e := tx.Model(&model.Endpoint{}).Where("cluster_id = ? AND mode = ?", c.ID, "cluster").Update("addresses", JSON(addresses)).Error; e != nil {
				return e
			}
		}
		for _, ep := range c.Endpoints {
			if ep.Mode == "read-only" {
				continue
			}
			var count int64
			tx.Model(&cmdb.CmdbSQL{}).Where("cluster_id = ? AND endpoint_id = ?", c.ID, ep.ID).Count(&count)
			if count > 0 {
				continue
			}
			a := ep.AccountID
			if a == 0 {
				a = firstAccount
				tx.Model(&ep).Update("account_id", a)
			}
			sourceName := c.Name + "/集群入口"
			if c.VIP != nil && ep.Name == c.Name+"/VIP" {
				sourceName = ep.Name
			}
			source := cmdb.CmdbSQL{Name: sourceName, Type: map[string]int{"mysql": 1, "postgresql": 2, "redis": 3}[c.Engine], AccountID: a, GroupID: c.GroupID, ClusterID: c.ID, EndpointID: ep.ID, ConnectionMode: "cluster", Description: "逻辑集群连接", TLSMode: "disable", CreatedAt: util.HTime{Time: *nowPtr()}, UpdatedAt: util.HTime{Time: *nowPtr()}}
			if e := tx.Create(&source).Error; e != nil {
				return e
			}
		}
		return nil
	})
}
func (s *Service) verifyEndpoints(ctx context.Context, c *model.Cluster) error {
	if len(c.Endpoints) == 0 {
		return nil
	}
	var failures []error
	for _, ep := range c.Endpoints {
		if ep.Mode == "sentinel" || ep.Mode == "cluster" {
			var source cmdb.CmdbSQL
			if s.DB.Where("cluster_id = ? AND endpoint_id = ?", c.ID, ep.ID).First(&source).Error != nil {
				continue
			}
			client, e := cmdbservice.NewDatabaseOperationService(s.DB).OpenRedis(source.ID)
			if e == nil {
				e = client.Ping(ctx).Err()
				client.Close()
			}
			status := "healthy"
			if e != nil {
				status = "unavailable"
			}
			s.DB.Model(&ep).Updates(map[string]any{"status": status, "verified_at": nowPtr()})
			if e != nil {
				failures = append(failures, errors.New("Redis 逻辑入口连接失败"))
			}
			continue
		}
		var addresses []string
		if json.Unmarshal(ep.Addresses, &addresses) != nil {
			return errors.New("入口地址无效")
		}
		healthy, failed := 0, 0
		conflict := false
		for _, address := range addresses {
			host, port, e := net.SplitHostPort(address)
			if e != nil {
				return e
			}
			n, e := strconv.Atoi(port)
			if e != nil {
				return e
			}
			m := model.Member{Kind: "data", Address: host, Port: n, AccountID: ep.AccountID, TLSMode: ep.TLSMode}
			if m.AccountID == 0 {
				for _, node := range c.Members {
					if node.Kind == "data" {
						m.AccountID = node.AccountID
						break
					}
				}
			}
			db, e := s.sql(ctx, c, m, "")
			if e != nil {
				failed++
				continue
			}
			q := "SELECT @@global.read_only"
			if c.Engine == "postgresql" {
				q = "SELECT pg_is_in_recovery()"
			}
			var ro any
			e = db.QueryRowContext(ctx, q).Scan(&ro)
			db.Close()
			if e != nil {
				failed++
				continue
			}
			if ep.Mode == "read-write" && flag(ro) || ep.Mode == "read-only" && !flag(ro) {
				conflict = true
				continue
			}
			healthy++
		}
		status := "healthy"
		if healthy == 0 {
			status = "unavailable"
		} else if failed > 0 {
			status = "degraded"
		}
		if conflict {
			status = "conflict"
		}
		if status == "unavailable" || status == "conflict" {
			failures = append(failures, fmt.Errorf("入口 %q（%s）核验失败：%s", ep.Name, ep.Mode, map[string]string{"unavailable": "无法连接可用节点", "conflict": "返回了错误的读写角色"}[status]))
		}
		if e := s.DB.Model(&ep).Updates(map[string]any{"status": status, "verified_at": nowPtr()}).Error; e != nil {
			return e
		}
	}
	return errors.Join(failures...)
}

func (s *Service) waitEndpoints(ctx context.Context, c *model.Cluster) error {
	if len(c.Endpoints) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return waitEndpointConvergence(ctx, 2*time.Second, func(ctx context.Context) error {
		return s.verifyEndpoints(ctx, c)
	})
}

// Retry read-only connection/role probes, never the native switchover. Require
// consecutive successes so a transient successful probe cannot end verification.
func waitEndpointConvergence(ctx context.Context, interval time.Duration, check func(context.Context) error) error {
	var lastErr error
	successes := 0
	for {
		if ctx.Err() != nil {
			return fmt.Errorf("业务入口未完成稳定核验：%w", errors.Join(ctx.Err(), lastErr))
		}
		if err := check(ctx); err != nil {
			lastErr = err
			successes = 0
		} else {
			successes++
			if successes >= 2 && ctx.Err() == nil {
				return nil
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("业务入口未完成稳定核验：%w", errors.Join(ctx.Err(), lastErr))
		case <-time.After(interval):
		}
	}
}
