package service

import (
	"context"
	"dodevops-api/api/database/model"
	"dodevops-api/api/database/policy"
	"errors"
	"sort"
	"time"
)

func memberDomain(m model.Member) string          { return policy.MemberDomain(m) }
func placementWarnings(c *model.Cluster) []string { return policy.RedisPlacementWarnings(c) }

// Match each shard to a host capacity slot. Augmenting paths avoid greedy
// choices stranding a shard whose replicas have fewer placement options.
func primaryPlacement(c *model.Cluster) ([]model.Member, error) {
	if c.Topology != "redis-cluster" {
		return nil, errors.New("仅支持 Redis Cluster")
	}
	roots := []model.Member{}
	hosts := map[string]bool{}
	for _, m := range c.Members {
		if m.Kind == "data" {
			if m.State != "healthy" {
				return nil, errors.New("重新分布前所有数据节点须健康")
			}
			hosts[memberDomain(m)] = true
			if m.Role == "primary" {
				roots = append(roots, m)
			}
		}
	}
	if len(roots) < 3 || len(hosts) < 3 {
		return nil, errors.New("至少需要三个分片主节点与三个主机故障域")
	}
	sort.Slice(roots, func(i, j int) bool { return roots[i].ID < roots[j].ID })
	domains := []string{}
	for host := range hosts {
		domains = append(domains, host)
	}
	sort.Strings(domains)
	slots := []string{}
	for i, host := range domains {
		n := len(roots) / len(domains)
		if i < len(roots)%len(domains) {
			n++
		}
		for k := 0; k < n; k++ {
			slots = append(slots, host)
		}
	}
	candidates := make([]map[string]model.Member, len(roots))
	for i, root := range roots {
		candidates[i] = map[string]model.Member{memberDomain(root): root}
		for _, m := range c.Members {
			if m.Kind == "data" && m.Role == "replica" && m.ParentID == root.NativeID && object(m.Metrics)["master_link_status"] == "up" {
				domain := memberDomain(m)
				if _, ok := candidates[i][domain]; !ok {
					candidates[i][domain] = m
				}
			}
		}
	}
	assigned := make([]int, len(slots))
	for i := range assigned {
		assigned[i] = -1
	}
	var match func(int, []bool) bool
	match = func(shard int, seen []bool) bool {
		for _, preserve := range []bool{true, false} {
			for slot, domain := range slots {
				if seen[slot] || (domain == memberDomain(roots[shard])) != preserve {
					continue
				}
				if _, ok := candidates[shard][domain]; !ok {
					continue
				}
				seen[slot] = true
				if assigned[slot] < 0 || match(assigned[slot], seen) {
					assigned[slot] = shard
					return true
				}
			}
		}
		return false
	}
	for i := range roots {
		if !match(i, make([]bool, len(slots))) {
			return nil, errors.New("现有副本无法覆盖目标故障域，需先调整副本归属")
		}
	}
	out := make([]model.Member, len(roots))
	for slot, shard := range assigned {
		out[shard] = candidates[shard][slots[slot]]
	}
	return out, nil
}

func (s *Service) balancePrimaries(ctx context.Context, c *model.Cluster, op model.Operation) error {
	chosen, e := primaryPlacement(c)
	if e != nil {
		return e
	}
	for _, candidate := range chosen {
		if candidate.Role == "primary" {
			continue
		}
		if e = s.step(op, "原生计划切换至 "+candidate.Name, func(log func(string)) error {
			fresh, err := s.Refresh(ctx, c.ID)
			if err != nil {
				return err
			}
			node, err := member(fresh, candidate.ID)
			if err != nil || node.Role != "replica" || node.ParentID != candidate.ParentID {
				return errors.New("副本归属已变，重新生成分布计划")
			}
			client, err := s.redis(fresh, *node)
			if err != nil {
				return err
			}
			defer client.Close()
			if err = client.ClusterFailover(ctx).Err(); err != nil {
				return err
			}
			return s.waitHealthy(ctx, c.ID, 90*time.Second, candidate.ID)
		}); e != nil {
			return e
		}
	}
	fresh, e := s.Refresh(ctx, c.ID)
	if e != nil {
		return e
	}
	for _, warning := range placementWarnings(fresh) {
		s.event(c.ID, op.UserID, op.ID, "placement-risk", "warning", warning)
	}
	return nil
}
