package service

import (
	"context"
	"dodevops-api/api/webservice/model"
	"encoding/json"
	"errors"
	"time"
)

type TopologyMember struct {
	Instance     model.Resource  `json:"instance"`
	Address      string          `json:"address"`
	Status       string          `json:"status"`
	Identity     string          `json:"identity"`
	Version      string          `json:"version"`
	ConfigDigest string          `json:"configDigest"`
	VIP          *VIPObservation `json:"vip,omitempty"`
}
type ClusterTopology struct {
	Cluster   model.Resource   `json:"cluster"`
	Members   []TopologyMember `json:"members"`
	Entries   []model.Entry    `json:"entries"`
	VIP       *VIPState        `json:"vip"`
	CheckedAt *time.Time       `json:"checkedAt"`
}

func (s *Service) Topology(ctx context.Context, u, cid uint) (ClusterTopology, error) {
	result := ClusterTopology{Members: []TopologyMember{}}
	if err := s.Module(ctx, u, "read"); err != nil {
		return result, err
	}
	c, err := s.Get("clusters", cid)
	if err != nil {
		return result, err
	}
	if err = s.Authorize(u, "clusters", c, "web_view"); err != nil {
		return result, err
	}
	s.Redact(u, "clusters", &c)
	result.Cluster = c
	result.CheckedAt = c.CollectedAt
	spec, _ := decode[model.ClusterSpec](c.Spec)
	result.Entries = spec.Entries
	members, err := s.Members(u, cid)
	if err != nil {
		return result, err
	}
	// The collector owns remote probes. A page read must never wait for SSH,
	// an unreachable peer, or business HTTP checks already collected in the background.
	visible := map[uint]bool{}
	for _, i := range members {
		if s.Authorize(u, "instances", i, "web_view") != nil {
			continue
		}
		visible[i.ID] = true
		ob, _ := decode[Observation](i.Observed)
		s.Redact(u, "instances", &i)
		i.Observed = nil // Config contents are loaded separately by the configuration page.
		node := TopologyMember{Instance: i, Status: i.Status, Identity: ob.Identity, Version: ob.Version, ConfigDigest: ob.Digest}
		if i.Connection != nil {
			node.Address = i.Connection.Address
		}
		result.Members = append(result.Members, node)
	}
	if spec.VIP != nil {
		var observed struct {
			VIP *VIPState `json:"vip"`
		}
		_ = json.Unmarshal(c.Observed, &observed)
		state := VIPState{Config: spec.VIP, Status: "unknown", Nodes: map[uint]VIPObservation{}, Message: "等待后台采集 VIP 状态"}
		if observed.VIP != nil && digest(JSON(observed.VIP.Config)) == digest(JSON(spec.VIP)) {
			state = *observed.VIP
			state.Nodes = map[uint]VIPObservation{}
			for id, node := range observed.VIP.Nodes {
				if visible[id] {
					state.Nodes[id] = node
				}
			}
			if state.CheckedAt.IsZero() || time.Since(state.CheckedAt) > 90*time.Second || state.CheckedAt.After(time.Now().Add(time.Minute)) {
				state.Status = "unknown"
				state.OwnerInstanceID = 0
				state.Nodes = map[uint]VIPObservation{}
				state.Message = "VIP 状态已过期，等待后台采集；可执行核验获取实时证据"
			}
		}
		for _, id := range spec.VIP.InstanceIDs {
			if !visible[id] {
				state.Status = "unknown"
				state.OwnerInstanceID = 0
				state.Message = "部分 VIP 成员不可见，无法确认完整高可用状态"
			}
		}
		result.VIP = &state
		for index := range result.Members {
			if o, ok := state.Nodes[result.Members[index].Instance.ID]; ok {
				result.Members[index].VIP = &o
			}
		}
	}
	// Avoid a second, unfiltered copy of VIP observations inside the resource.
	result.Cluster.Observed = nil
	return result, nil
}

func (s *Service) vipTopologyTargets(cid uint, p *model.VIPSpec, members []model.Resource) ([]Target, error) {
	targets := []Target{}
	for _, id := range p.InstanceIDs {
		found := false
		for _, member := range members {
			if member.ID != id {
				continue
			}
			found = true
			source, err := s.Get("sources", member.SourceID)
			if err != nil {
				return nil, err
			}
			spec, err := decode[model.SourceSpec](source.Spec)
			if err != nil {
				return nil, err
			}
			targets = append(targets, Target{Instance: member, Spec: spec, VIP: &VIPNode{ClusterID: cid, InstanceID: id, Config: *p}})
		}
		if !found {
			return nil, errors.New("VIP 成员已被移出集群")
		}
	}
	return targets, nil
}
