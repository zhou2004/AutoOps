package db

import (
	cmdb "dodevops-api/api/cmdb/model"
	account "dodevops-api/api/configcenter/model"
	database "dodevops-api/api/database/model"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Backfill references only. Never changes database servers, credentials or HA policies.
func migrateDatabaseClusters(db *gorm.DB) error {
	var deployments []cmdb.DatabaseDeployment
	if err := db.Find(&deployments).Error; err != nil {
		return err
	}
	for _, d := range deployments {
		err := db.Transaction(func(tx *gorm.DB) error {
			var locked cmdb.DatabaseDeployment
			if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&locked, d.ID).Error; e != nil {
				return e
			}
			var existing database.Cluster
			if e := tx.Where("deployment_id = ?", d.ID).First(&existing).Error; e == nil {
				return nil
			} else if e != gorm.ErrRecordNotFound {
				return e
			}
			topology := d.Mode
			if d.Engine == "mysql" && d.Mode == "replication" {
				topology = "mysql-gtid"
			}
			c := database.Cluster{Name: d.Name, Engine: d.Engine, Topology: topology, GroupID: d.GroupID, DeploymentID: &d.ID, Environment: "legacy", Management: "observe", Status: "unknown", Revision: 1, Policy: json.RawMessage(`{"priority":"data-protection","origin":"legacy","autoRecovery":"external"}`), Health: json.RawMessage(`{}`), CreatedBy: d.CreatedBy}
			if e := tx.Create(&c).Error; e != nil {
				return e
			}
			var sources []cmdb.CmdbSQL
			if e := tx.Where("deployment_id = ?", d.ID).Find(&sources).Error; e != nil {
				return e
			}
			for _, src := range sources {
				var a account.AccountAuth
				if e := tx.First(&a, src.AccountID).Error; e != nil {
					return e
				}
				m := database.Member{ClusterID: c.ID, HostID: src.HostID, SourceID: src.ID, AccountID: src.AccountID, Name: src.Name, Kind: "data", Address: a.Host, Port: a.Port, Role: "unknown", State: "unknown", Managed: true, Metrics: json.RawMessage(`{}`)}
				if d.Engine == "mysql" {
					m.DataDir = "/var/lib/autoops-mysql"
					m.Service = "autoops-mysql"
				}
				if e := tx.Create(&m).Error; e != nil {
					return e
				}
				if e := tx.Model(&src).Update("cluster_id", c.ID).Error; e != nil {
					return e
				}
			}
			var managers []uint
			_ = json.Unmarshal([]byte(d.Managers), &managers)
			for _, id := range managers {
				var p cmdb.CmdbAssetConnection
				if tx.Where("asset_type = 'host' AND asset_id = ? AND protocol = 'ssh' AND enabled = ?", id, true).Order("id").First(&p).Error != nil {
					continue
				}
				m := database.Member{ClusterID: c.ID, HostID: id, Name: fmt.Sprintf("orchestrator-%d", id), Kind: "orchestrator", Address: p.Address, Port: 13000, Service: "autoops-orchestrator", Managed: true, State: "unknown", Metrics: json.RawMessage(`{}`)}
				if e := tx.Create(&m).Error; e != nil {
					return e
				}
			}
			var runs []cmdb.DatabaseAutomationRun
			if e := tx.Where("deployment_id = ? AND action = ? AND status = ?", d.ID, "backup", "success").Find(&runs).Error; e != nil {
				return e
			}
			for _, r := range runs {
				manifest, _ := json.Marshal(map[string]any{"legacyRunId": r.ID, "artifact": r.Artifact, "integrity": "legacy-not-revalidated"})
				b := database.Backup{ClusterID: c.ID, LegacyRunID: &r.ID, Method: "legacy", Status: "legacy", Manifest: manifest, CreatedAt: r.CreatedAt}
				if e := tx.Create(&b).Error; e != nil {
					return e
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}
