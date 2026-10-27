package db

import "gorm.io/gorm"

// 兼容 SQL 初始化文件中的旧版故障记录；保留原因、处理人等旧字段。
// 只转换旧状态，重复启动不会覆盖新记录或人工处理结果。
func migrateLegacyMonitorIncidents(database *gorm.DB) error {
	if !database.Migrator().HasColumn("monitor_incident", "alert_desc") {
		return nil
	}
	return database.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`UPDATE monitor_incident SET
   title = COALESCE(NULLIF(SUBSTR(alert_desc,1,512),''),'历史故障'),
   description = COALESCE(alert_desc,''),
   source = 'legacy',
   level = CASE WHEN alert_level IN ('P0','P1','P2') THEN 'critical' WHEN alert_level = 'P4' THEN 'info' ELSE 'warning' END,
   resolved_at = CASE WHEN status = '3' THEN COALESCE(CAST(update_time AS CHAR),CAST(alert_time AS CHAR)) ELSE '' END,
   status = CASE WHEN status = '3' THEN 'resolved' ELSE 'firing' END
   WHERE status IN ('1','2','3')`).Error; err != nil {
			return err
		}
		return nil
	})
}
