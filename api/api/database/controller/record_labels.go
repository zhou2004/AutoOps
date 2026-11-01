package controller

import (
	"dodevops-api/api/database/model"
	"encoding/json"
)

// Input rows have already been scoped. Only decorate clusters referenced by that page.
func (ctrl *Controller) recordLabels(value any) []map[string]any {
	rows := []map[string]any{}
	b, _ := json.Marshal(value)
	_ = json.Unmarshal(b, &rows)
	ids := []uint{}
	for _, r := range rows {
		if id, ok := r["clusterId"].(float64); ok {
			ids = append(ids, uint(id))
		}
	}
	var clusters []model.Cluster
	if len(ids) > 0 {
		ctrl.s.DB.Select("id,name,engine").Where("id IN ?", ids).Find(&clusters)
	}
	lookup := map[uint]model.Cluster{}
	for _, c := range clusters {
		lookup[c.ID] = c
	}
	for _, r := range rows {
		if id, ok := r["clusterId"].(float64); ok {
			c := lookup[uint(id)]
			r["clusterName"] = c.Name
			r["engine"] = c.Engine
		}
	}
	return rows
}
