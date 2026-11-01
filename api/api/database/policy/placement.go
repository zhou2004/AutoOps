package policy

import (
	"dodevops-api/api/database/model"
	"fmt"
	"sort"
)

func MemberDomain(m model.Member) string {
	if m.HostID > 0 {
		return fmt.Sprintf("host:%d", m.HostID)
	}
	return "address:" + m.Address
}
func RedisPlacementWarnings(c *model.Cluster) []string {
	if c.Topology != "redis-cluster" {
		return nil
	}
	masters := []model.Member{}
	votes := map[string]int{}
	for _, m := range c.Members {
		if m.Kind == "data" && m.Role == "primary" {
			masters = append(masters, m)
			votes[MemberDomain(m)]++
		}
	}
	warnings := []string{}
	for host, n := range votes {
		if len(masters)-n < len(masters)/2+1 {
			warnings = append(warnings, fmt.Sprintf("故障域 %s 承载 %d/%d 个主节点，失去该域会失去选主多数派；请重新分布主节点", host, n, len(masters)))
		}
	}
	for _, root := range masters {
		domains := map[string]bool{MemberDomain(root): true}
		for _, m := range c.Members {
			if m.Kind == "data" && m.Role == "replica" && m.ParentID == root.NativeID {
				domains[MemberDomain(m)] = true
			}
		}
		if len(domains) < 3 {
			warnings = append(warnings, fmt.Sprintf("分片 %s 仅覆盖 %d 个主机故障域；单主机故障后可能不满足至少一个有效副本的写入保护", root.Name, len(domains)))
		}
	}
	sort.Strings(warnings)
	return warnings
}
