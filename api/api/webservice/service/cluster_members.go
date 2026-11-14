package service

import (
	"context"
	"dodevops-api/api/webservice/model"
	"errors"
	"fmt"
	"sort"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Explicit deltas preserve members outside the editor's visible source scope.
// Omitted/empty changes only update the cluster's metadata.
type ClusterMemberChanges struct {
	Add    []uint `json:"add"`
	Remove []uint `json:"remove"`
}

func (s *Service) SaveCluster(ctx context.Context, user uint, resource model.Resource, changes *ClusterMemberChanges) (model.Resource, error) {
	if err := s.Module(ctx, user, "write"); err != nil {
		return model.Resource{}, err
	}
	if changes == nil {
		changes = &ClusterMemberChanges{}
	}
	if len(changes.Add)+len(changes.Remove) > 200 {
		return model.Resource{}, errors.New("单次最多调整 200 个集群成员")
	}
	seen := map[uint]bool{}
	for _, ids := range [][]uint{changes.Add, changes.Remove} {
		for _, id := range ids {
			if id == 0 || seen[id] {
				return model.Resource{}, errors.New("实例 ID 无效或成员变更重复")
			}
			seen[id] = true
		}
	}
	// A consistent order avoids inversions when two clusters claim shared free instances.
	add, remove := append([]uint{}, changes.Add...), append([]uint{}, changes.Remove...)
	sort.Slice(add, func(i, j int) bool { return add[i] < add[j] })
	sort.Slice(remove, func(i, j int) bool { return remove[i] < remove[j] })
	var saved model.Resource
	err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		local := *s
		local.DB = tx
		if resource.ID != 0 {
			var locked model.Resource
			if err := tx.Table(model.Tables["clusters"]).Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, resource.ID).Error; err != nil {
				return err
			}
		}
		var err error
		saved, err = local.Save(ctx, user, "clusters", resource, "")
		if err != nil {
			return err
		}
		for _, id := range remove {
			if err = local.BindMember(ctx, user, saved.ID, id, true); err != nil {
				return err
			}
		}
		for _, id := range add {
			if err = local.BindMember(ctx, user, saved.ID, id, false); err != nil {
				return err
			}
		}
		saved, err = local.Get("clusters", saved.ID)
		return err
	})
	if err != nil {
		return model.Resource{}, err
	}
	return saved, nil
}

func (s *Service) BindMember(ctx context.Context, user, cid, iid uint, remove bool) error {
	if err := s.Module(ctx, user, "execute"); err != nil {
		return err
	}
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		local := *s
		local.DB = tx
		var cluster, instance model.Resource
		if err := tx.Table(model.Tables["clusters"]).Clauses(clause.Locking{Strength: "UPDATE"}).First(&cluster, cid).Error; err != nil {
			return err
		}
		if err := local.Authorize(user, "clusters", cluster, "web_manage"); err != nil {
			return err
		}
		if err := tx.Table(model.Tables["instances"]).Clauses(clause.Locking{Strength: "UPDATE"}).First(&instance, iid).Error; err != nil {
			return err
		}
		if err := local.Authorize(user, "instances", instance, "web_manage"); err != nil {
			return err
		}
		if instance.GroupID != cluster.GroupID || instance.Engine != cluster.Engine {
			return errors.New("实例与集群的引擎和业务分组必须一致")
		}
		var active int64
		if err := tx.Model(&model.Lock{}).Where("`key` IN ?", []string{fmt.Sprintf("cluster:%d", cid), fmt.Sprintf("sources:%d", instance.SourceID)}).Count(&active).Error; err != nil {
			return err
		}
		if active > 0 {
			return ErrConflict
		}
		if remove {
			spec, _ := decode[model.ClusterSpec](cluster.Spec)
			if spec.VIP != nil {
				for _, id := range spec.VIP.InstanceIDs {
					if id == iid {
						return errors.New("VIP 成员不能直接移出，请先退役高可用配置")
					}
				}
			}
		}
		previous, next := uint(0), cid
		action := "cluster-member-added"
		if remove {
			previous, next = cid, 0
			action = "cluster-member-removed"
		}
		if instance.ClusterID != previous {
			return fmt.Errorf("%w：实例归属已变化，请重新加载成员", ErrConflict)
		}
		update := tx.Table(model.Tables["instances"]).Where("id = ? AND cluster_id = ? AND revision = ?", iid, previous, instance.Revision).Updates(map[string]any{"cluster_id": next, "revision": gorm.Expr("revision + 1")})
		if update.Error != nil {
			return update.Error
		}
		if update.RowsAffected != 1 {
			return ErrConflict
		}
		if err := tx.Table(model.Tables["clusters"]).Where("id = ?", cid).Updates(map[string]any{"revision": gorm.Expr("revision + 1"), "observed": nil, "collected_at": nil, "status": "unknown"}).Error; err != nil {
			return err
		}
		return tx.Create(&model.Event{UserID: user, ClusterID: cid, SourceID: instance.SourceID, Kind: action, Message: fmt.Sprintf("instance=%d", iid)}).Error
	})
}
