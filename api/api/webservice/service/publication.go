package service

import (
	"dodevops-api/api/webservice/model"
	"fmt"
)

// A saved site is a draft until its rendered file is observed on every member.
// Recompute after collection so rollback and terminal edits also change the UI.
func (s *Service) refreshPublicationState(clusterID uint, members []model.Resource) {
	var locks int64
	if err := s.DB.Model(&model.Lock{}).Where("`key` = ?", fmt.Sprintf("cluster:%d", clusterID)).Count(&locks).Error; err != nil || locks != 0 {
		return
	}
	for _, kind := range []string{"sites", "upstreams"} {
		var resources []model.Resource
		if s.DB.Table(model.Tables[kind]).Where("cluster_id = ?", clusterID).Find(&resources).Error != nil {
			continue
		}
		for _, resource := range resources {
			var path, content string
			var err error
			if kind == "sites" {
				spec, e := decode[model.SiteSpec](resource.Spec)
				if e != nil {
					continue
				}
				path = spec.Path
				content, err = RenderSite(spec)
			} else {
				spec, e := decode[UpstreamSpec](resource.Spec)
				if e != nil {
					continue
				}
				path = spec.Path
				content, err = RenderUpstream(spec)
			}
			if err != nil {
				continue
			}
			matched := 0
			for _, member := range members {
				observation, e := decode[Observation](member.Observed)
				if e != nil {
					continue
				}
				for _, file := range observation.Files {
					if file.Path == path && !file.Deleted && file.Content == content {
						matched++
						break
					}
				}
			}
			status := "draft"
			if matched > 0 {
				status = "partial"
				if matched == len(members) {
					status = "published"
				}
			}
			if resource.Status != status {
				s.DB.Table(model.Tables[kind]).Where("id = ? AND revision = ?", resource.ID, resource.Revision).Update("status", status)
			}
		}
	}
}
