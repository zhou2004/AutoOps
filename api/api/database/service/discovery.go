package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	cmdb "dodevops-api/api/cmdb/model"
	accounts "dodevops-api/api/configcenter/model"
	"dodevops-api/api/database/model"
)

type discoveredAddress struct {
	address string
	port    int
	kind    string
}

func (s *Service) discoverMembers(ctx context.Context, c *model.Cluster) {
	seen := map[string]bool{}
	key := func(m model.Member) string { return net.JoinHostPort(m.Address, strconv.Itoa(m.Port)) }
	for _, m := range c.Members {
		seen[key(m)] = true
	}
	for i := 0; i < len(c.Members) && i < 64; i++ {
		if ctx.Err() != nil {
			break
		}
		m := &c.Members[i]
		if e := s.probeMember(ctx, c, m); e != nil {
			m.State = "unreachable"
			m.Metrics = JSON(map[string]any{"error": e.Error()})
			continue
		}
		seed := *m
		for _, a := range s.neighbors(ctx, c, seed) {
			if a.port < 1 || a.port > 65535 || !validAddress(a.address) {
				continue
			}
			n := model.Member{Address: a.address, Port: a.port, Kind: a.kind, State: "unknown", Metrics: JSON(map[string]any{})}
			if seen[key(n)] || len(c.Members) >= 64 {
				continue
			}
			// Data credentials can be shared across replication peers. A control
			// component always needs its own explicitly supplied credential.
			if n.Kind == seed.Kind {
				n.AccountID = seed.AccountID
			}
			n.Name = fmt.Sprintf("%s-%d", n.Kind, len(c.Members)+1)
			s.matchDiscovered(c, &n)
			seen[key(n)] = true
			c.Members = append(c.Members, n)
		}
	}
}

func (s *Service) matchDiscovered(c *model.Cluster, m *model.Member) {
	var rows []struct {
		ID, HostID, AccountID uint
		TLSMode               string
	}
	s.DB.Table("cmdb_sql s").Select("s.id,s.host_id,s.account_id,s.tls_mode").Joins("JOIN config_account a ON a.id = s.account_id").Where("a.host = ? AND a.port = ? AND s.type = ?", m.Address, m.Port, map[string]int{"mysql": 1, "postgresql": 2, "redis": 3}[c.Engine]).Find(&rows)
	if len(rows) == 1 && m.Kind == "data" {
		m.SourceID, m.HostID, m.AccountID = rows[0].ID, rows[0].HostID, rows[0].AccountID
		m.TLSMode = rows[0].TLSMode
	}
	if m.HostID == 0 {
		var ids []uint
		s.DB.Model(&cmdb.CmdbAssetConnection{}).Distinct("asset_id").Where("asset_type = ? AND address = ? AND protocol = ? AND enabled = ?", "host", m.Address, "ssh", true).Pluck("asset_id", &ids)
		if len(ids) == 1 {
			m.HostID = ids[0]
		}
	}
}

func (s *Service) neighbors(ctx context.Context, c *model.Cluster, m model.Member) []discoveredAddress {
	out := []discoveredAddress{}
	add := func(address string, port int, kind string) { out = append(out, discoveredAddress{address, port, kind}) }
	metrics := object(m.Metrics)
	if c.Engine == "mysql" && m.Kind == "data" {
		add(text(metrics["upstreamAddress"]), int(number(metrics["upstreamPort"])), "data")
		db, e := s.sql(ctx, c, m, "")
		if e != nil {
			return out
		}
		defer db.Close()
		rows, _ := queryMaps(ctx, db, "SHOW SLAVE HOSTS")
		for _, row := range rows {
			add(text(row["Host"]), int(number(row["Port"])), "data")
		}
	} else if c.Engine == "redis" {
		if m.Kind == "data" {
			add(text(metrics["master_host"]), int(number(metrics["master_port"])), "data")
			for k, v := range metrics {
				if !strings.HasPrefix(k, "slave") {
					continue
				}
				parts := map[string]string{}
				for _, field := range strings.Split(text(v), ",") {
					kv := strings.SplitN(field, "=", 2)
					if len(kv) == 2 {
						parts[kv[0]] = kv[1]
					}
				}
				p, _ := strconv.Atoi(parts["port"])
				add(parts["ip"], p, "data")
			}
			client, e := s.redis(c, m)
			if e == nil {
				defer client.Close()
				raw, e := client.ClusterNodes(ctx).Result()
				if e == nil {
					for _, n := range parseClusterNodes(raw) {
						host, p, e := net.SplitHostPort(strings.Split(text(n["address"]), "@")[0])
						if e == nil {
							port, _ := strconv.Atoi(p)
							add(host, port, "data")
						}
					}
				}
			}
		} else if m.Kind == "sentinel" {
			client, e := s.redis(c, m)
			if e != nil {
				return out
			}
			defer client.Close()
			name := text(object(c.Policy)["masterName"])
			if name == "" {
				name = c.Name
			}
			for _, command := range []string{"MASTER", "REPLICAS", "SENTINELS"} {
				value, e := client.Do(ctx, "SENTINEL", command, name).Result()
				if e != nil {
					continue
				}
				rows, _ := value.([]interface{})
				if command == "MASTER" {
					rows = []interface{}{rows}
				}
				for _, row := range rows {
					fields, _ := row.([]interface{})
					v := map[string]any{}
					for i := 0; i+1 < len(fields); i += 2 {
						v[text(fields[i])] = fields[i+1]
					}
					kind := "data"
					if command == "SENTINELS" {
						kind = "sentinel"
					}
					add(text(v["ip"]), int(number(v["port"])), kind)
				}
			}
		}
	} else if c.Engine == "postgresql" && m.Kind == "patroni" {
		raw, e := s.control(ctx, c, m, http.MethodGet, "/cluster", nil)
		if e != nil {
			return out
		}
		rows, _ := object(raw)["members"].([]interface{})
		for _, row := range rows {
			v := object(JSON(row))
			add(text(v["host"]), int(number(v["port"])), "data")
		}
	} else if c.Engine == "postgresql" && m.Kind == "data" {
		var rows []map[string]any
		_ = json.Unmarshal(JSON(metrics["receiver"]), &rows)
		for _, v := range rows {
			add(text(v["sender_host"]), int(number(v["sender_port"])), "data")
		}
		// PostgreSQL reports replica client ports, not listening ports. Only use
		// existing account endpoints for those addresses, never guess a port.
		var replicas []map[string]any
		_ = json.Unmarshal(JSON(metrics["replicas"]), &replicas)
		for _, r := range replicas {
			var a []accounts.AccountAuth
			address := text(r["client_addr"])
			if ip, _, err := net.ParseCIDR(address); err == nil {
				address = ip.String()
			}
			s.DB.Select("host,port").Where("host = ? AND type = ?", address, 2).Find(&a)
			for _, endpoint := range a {
				add(endpoint.Host, endpoint.Port, "data")
			}
		}
	}
	return out
}
