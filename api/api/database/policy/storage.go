package policy

import (
	cmdb "dodevops-api/api/cmdb/model"
	"dodevops-api/api/database/model"
	"gorm.io/gorm"
)

// The nearest ancestor override wins. Cycles and absent groups fall back to
// conservative platform defaults rather than looping indefinitely.
func StorageThresholds(db *gorm.DB, group uint) model.StorageThreshold {
	defaults := model.StorageThreshold{WarningPercent: 80, CriticalPercent: 90}
	seen := map[uint]bool{}
	for group > 0 && !seen[group] {
		seen[group] = true
		var rule model.StorageThreshold
		if db.First(&rule, "group_id = ?", group).Error == nil {
			return rule
		}
		var g cmdb.CmdbGroup
		if db.Select("id,parent_id").First(&g, group).Error != nil {
			break
		}
		group = g.ParentID
	}
	return defaults
}
