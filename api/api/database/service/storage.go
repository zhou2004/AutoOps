package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"dodevops-api/api/database/model"
	"dodevops-api/api/database/policy"
)

const filesystemScript = `import json,sys,os
p=json.load(sys.stdin);path=p['path'];v=os.statvfs(path);st=os.stat(path)
mem={k:int(v.split()[0])*1024 for k,v in (line.split(':',1) for line in open('/proc/meminfo'))}
print(json.dumps(dict(filesystemId=str(st.st_dev),totalBytes=v.f_blocks*v.f_frsize,availableBytes=v.f_bavail*v.f_frsize,usedBytes=(v.f_blocks-v.f_bfree)*v.f_frsize,inodes=v.f_files,freeInodes=v.f_favail,availableMemory=mem.get('MemAvailable',0),path=os.path.realpath(path))))`

func (s *Service) filesystem(ctx context.Context, m model.Member) (map[string]any, error) {
	raw, e := s.remoteJSON(ctx, m.HostID, filesystemScript, map[string]any{"path": m.DataDir})
	if e != nil {
		return nil, e
	}
	return object(raw), nil
}
func (s *Service) collectStorage(ctx context.Context, c *model.Cluster, op model.Operation) error {
	return s.step(op, "采集数据库、日志和文件系统容量", func(log func(string)) error { return s.collectStorageSnapshots(ctx, c, log) })
}
func (s *Service) collectStorageSnapshots(ctx context.Context, c *model.Cluster, log func(string)) error {
	snapshots := []model.StorageSnapshot{}
	seen := map[string]bool{}
	t := time.Now().UTC()
	for _, m := range c.Members {
		if m.Kind != "data" {
			continue
		}
		snapshot := model.StorageSnapshot{ClusterID: c.ID, MemberID: m.ID, HostID: m.HostID, Kind: "instance", Name: m.Name, CollectedAt: t}
		details := map[string]any{"path": m.DataDir, "engine": c.Engine, "role": m.Role}
		if c.Engine == "mysql" {
			db, e := s.sql(ctx, c, m, "")
			if e != nil {
				details["error"] = e.Error()
			} else {
				rows, e := queryMaps(ctx, db, "SELECT table_schema AS name,COALESCE(SUM(data_length),0) AS data_bytes,COALESCE(SUM(index_length),0) AS index_bytes FROM information_schema.tables GROUP BY table_schema")
				if e != nil {
					details["error"] = e.Error()
				}
				details["databases"] = rows
				for _, r := range rows {
					snapshot.UsedBytes += uint64(number(r["data_bytes"]) + number(r["index_bytes"]))
				}
				logs, _ := queryMaps(ctx, db, "SHOW BINARY LOGS")
				details["binlogs"] = logs
				db.Close()
			}
		}
		if c.Engine == "postgresql" {
			db, e := s.sql(ctx, c, m, "")
			if e != nil {
				details["error"] = e.Error()
			} else {
				rows, e := queryMaps(ctx, db, "SELECT datname AS name,pg_database_size(oid) AS bytes FROM pg_database")
				if e != nil {
					details["error"] = e.Error()
				}
				details["databases"] = rows
				for _, r := range rows {
					snapshot.UsedBytes += uint64(number(r["bytes"]))
				}
				tablespaces, _ := queryMaps(ctx, db, "SELECT spcname AS name,pg_tablespace_location(oid) AS path,pg_tablespace_size(oid) AS bytes FROM pg_tablespace")
				details["tablespaces"] = tablespaces
				slots, _ := queryMaps(ctx, db, "SELECT slot_name,slot_type,active,restart_lsn::text,confirmed_flush_lsn::text FROM pg_replication_slots")
				details["replicationSlots"] = slots
				archive, _ := queryMaps(ctx, db, "SELECT archived_count,last_archived_wal,last_archived_time,failed_count,last_failed_wal,last_failed_time FROM pg_stat_archiver")
				details["archive"] = archive
				db.Close()
			}
		}
		if c.Engine == "redis" {
			client, e := s.redis(c, m)
			if e != nil {
				details["error"] = e.Error()
			} else {
				raw, e := client.Info(ctx).Result()
				client.Close()
				if e != nil {
					details["error"] = "Redis 容量采集失败"
				} else {
					metrics := infoMap(raw)
					for _, k := range []string{"used_memory", "used_memory_rss", "maxmemory", "mem_fragmentation_ratio", "rdb_last_bgsave_status", "rdb_last_save_time", "aof_enabled", "aof_current_size", "aof_last_write_status", "aof_last_bgrewrite_status"} {
						details[k] = metrics[k]
					}
					snapshot.UsedBytes = uint64(number(metrics["used_memory"]))
					snapshot.TotalBytes = uint64(number(metrics["maxmemory"]))
				}
			}
		}
		snapshot.Details = JSON(details)
		snapshots = append(snapshots, snapshot)
		if m.HostID > 0 && m.DataDir != "" {
			fs, e := s.filesystem(ctx, m)
			if e != nil {
				snapshots = append(snapshots, model.StorageSnapshot{ClusterID: c.ID, MemberID: m.ID, HostID: m.HostID, Kind: "filesystem", Name: m.DataDir, Details: JSON(map[string]any{"error": e.Error()}), CollectedAt: t})
				continue
			}
			key := fmt.Sprintf("%d:%s", m.HostID, fs["filesystemId"])
			if seen[key] {
				continue
			}
			seen[key] = true
			snapshots = append(snapshots, model.StorageSnapshot{ClusterID: c.ID, MemberID: m.ID, HostID: m.HostID, Kind: "filesystem", Name: m.DataDir, FilesystemID: key, TotalBytes: uint64(number(fs["totalBytes"])), UsedBytes: uint64(number(fs["usedBytes"])), AvailableBytes: uint64(number(fs["availableBytes"])), Details: JSON(fs), CollectedAt: t})
		}
	}
	if len(snapshots) == 0 {
		return errors.New("没有可采集的数据库实例")
	}
	if e := s.DB.Create(&snapshots).Error; e != nil {
		return e
	}
	log(fmt.Sprintf("已采集 %d 条容量记录；同一主机文件系统已去重", len(snapshots)))
	return nil
}
func (s *Service) Storage(user uint, f Filter) ([]model.StorageSnapshot, error) {
	ids := []uint{}
	f.Page, f.Size = 1, 100
	for {
		clusters, total, e := s.List(user, f)
		if e != nil {
			return nil, e
		}
		for _, c := range clusters {
			ids = append(ids, c.ID)
		}
		if int64(len(ids)) >= total || len(clusters) == 0 {
			break
		}
		f.Page++
	}
	rows := []model.StorageSnapshot{}
	if len(ids) == 0 {
		return rows, nil
	}
	latest := s.DB.Model(&model.StorageSnapshot{}).Select("MAX(id)").Where("cluster_id IN ?", ids).Group("cluster_id,member_id,kind,name")
	e := s.DB.Where("id IN (?)", latest).Order("cluster_id,kind,member_id").Find(&rows).Error
	unique := []model.StorageSnapshot{}
	seen := map[string]bool{}
	for _, row := range rows {
		if row.Kind == "filesystem" && row.FilesystemID != "" {
			if seen[row.FilesystemID] {
				continue
			}
			seen[row.FilesystemID] = true
		}
		detail := object(row.Details)
		detail["stale"] = time.Since(row.CollectedAt) > 10*time.Minute
		if row.Kind == "filesystem" && row.TotalBytes > 0 {
			var cluster model.Cluster
			s.DB.Select("id,group_id").First(&cluster, row.ClusterID)
			thresholds := policy.StorageThresholds(s.DB, cluster.GroupID)
			detail["warningPercent"], detail["criticalPercent"], detail["thresholdGroupId"] = thresholds.WarningPercent, thresholds.CriticalPercent, thresholds.GroupID
			var samples []model.StorageSnapshot
			s.DB.Where("filesystem_id = ? AND kind = ? AND collected_at >= ?", row.FilesystemID, "filesystem", time.Now().Add(-7*24*time.Hour)).Order("collected_at ASC").Find(&samples)
			if len(samples) >= 12 {
				first := samples[0]
				elapsed := row.CollectedAt.Sub(first.CollectedAt).Hours()
				if elapsed >= 1 && row.UsedBytes > first.UsedBytes {
					detail["estimatedDaysRemaining"] = float64(row.AvailableBytes) / (float64(row.UsedBytes-first.UsedBytes) / elapsed) / 24
				}
			}
		}
		row.Details = JSON(detail)
		unique = append(unique, row)
	}
	return unique, e
}

// Cleanup requires a restorable archive covering the entire configured window,
// an independently verified copy, and live execution coordinates from every
// registered replica. No cached UI health or fabricated manifest flag suffices.
func (s *Service) binlogPurgeSafe(ctx context.Context, c *model.Cluster, before string) bool {
	root, e := primary(c)
	if e != nil || c.Engine != "mysql" || before == "" {
		return false
	}
	db, e := s.sql(ctx, c, *root, "")
	if e != nil {
		return false
	}
	defer db.Close()
	logs, e := queryMaps(ctx, db, "SHOW BINARY LOGS")
	if e != nil {
		return false
	}
	positions := map[string]int{}
	boundary := -1
	for i, r := range logs {
		name := text(r["Log_name"])
		positions[name] = i
		if name == before {
			boundary = i
		}
	}
	if boundary <= 0 {
		return false
	}
	for _, m := range c.Members {
		if m.Kind != "data" || m.ID == root.ID {
			continue
		}
		replica, e := s.sql(ctx, c, m, "")
		if e != nil {
			return false
		}
		status, e := queryMaps(ctx, replica, "SHOW SLAVE STATUS")
		replica.Close()
		if e != nil || len(status) != 1 {
			return false
		}
		r := status[0]
		index, ok := positions[text(r["Relay_Master_Log_File"])]
		if !ok || index < boundary || text(r["Master_UUID"]) != root.NativeID || text(r["Slave_IO_Running"]) != "Yes" || text(r["Slave_SQL_Running"]) != "Yes" {
			return false
		}
	}
	var backups []model.Backup
	if s.DB.Where("cluster_id = ? AND status = ? AND method = ? AND verified_at IS NOT NULL AND repository_id > 0", c.ID, "complete", "binlog-archive").Order("id DESC").Find(&backups).Error != nil {
		return false
	}
	windowStart := time.Now().AddDate(0, 0, -s.retentionDays(c.ID)).Unix()
	for _, b := range backups {
		manifest := object(b.Manifest)
		if manifest["remoteComplete"] != true {
			continue
		}
		key := fmt.Sprintf("member_%d", root.ID)
		artifacts := object(JSON(manifest["artifacts"]))
		a := object(JSON(artifacts[key]))
		if a["archiveCheck"] != true || text(a["nativeId"]) != root.NativeID || number(a["backupStop"]) > float64(windowStart) {
			continue
		}
		var names []string
		_ = json.Unmarshal(JSON(a["binlogs"]), &names)
		covered := map[string]bool{}
		for _, name := range names {
			covered[name] = true
		}
		complete := true
		for _, r := range logs[:boundary] {
			if !covered[text(r["Log_name"])] {
				complete = false
			}
		}
		if !complete {
			continue
		}
		var repo model.Repository
		if s.DB.First(&repo, b.RepositoryID).Error != nil {
			continue
		}
		report := object(JSON(object(JSON(manifest["repositoryReport"]))[key]))
		if report["verified"] != true || report["sha256"] != a["sha256"] {
			continue
		}
		credentials := map[string]string{}
		if repo.Kind == "s3" {
			credentials, e = s.RepositoryCredentials(repo.KeyID)
			if e != nil {
				continue
			}
		}
		raw, e := s.remoteJSON(ctx, repo.HostID, repositoryVerifyScript, map[string]any{"repository": repo, "report": report, "credentials": credentials})
		if e == nil && object(raw)["verified"] == true {
			return true
		}
	}
	return false
}

const repositoryVerifyScript = `import json,sys,pathlib,hashlib
p=json.load(sys.stdin);r=p['repository'];a=p['report']
if r['kind']=='s3':
 import boto3
 client=boto3.client('s3',endpoint_url=r['endpoint'],region_name=r.get('region') or 'us-east-1',aws_access_key_id=p['credentials']['accessKey'],aws_secret_access_key=p['credentials']['secretKey'])
 args={'Bucket':r['bucket'],'Key':a['key']}
 if a.get('versionId'):args['VersionId']=a['versionId']
 source=client.get_object(**args)['Body']
else:
 path=pathlib.Path(a['path']);assert not path.is_symlink() and path.resolve().is_relative_to(pathlib.Path(r['path']).resolve());source=path.open('rb')
try:
 h=hashlib.sha256()
 for chunk in iter(lambda:source.read(1048576),b''):h.update(chunk)
 assert h.hexdigest()==a['sha256']
finally:source.close()
print(json.dumps({'verified':True}))`
