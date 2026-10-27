package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	mysqlDriver "github.com/go-sql-driver/mysql"
	"net"
	"os"
	"strconv"
	"testing"
	"time"

	"dodevops-api/api/cmdb/model"
	"dodevops-api/common/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Stops only the isolated autoops-* services installed by the deployment live
// test. Existing mysqld/orchestrator services and Kubernetes are never stopped.
func TestDatabaseHAFailoverLive(t *testing.T) {
	if os.Getenv("AUTOOPS_DB_HA_LIVE") != "1" {
		t.Skip("opt in with AUTOOPS_DB_HA_LIVE=1 after installing the validation deployment")
	}
	if err := config.LoadConfig(os.Getenv("CMDB_CONFIG_PATH")); err != nil {
		t.Fatal(err)
	}
	cfg := config.GetConfig()
	db, err := gorm.Open(mysql.Open(fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local", cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Db, cfg.Charset)), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("CMDB unavailable")
	}
	var d model.DatabaseDeployment
	if db.Where("name = ? AND status = ?", "autoops-mysql-validation", "ready").First(&d).Error != nil {
		t.Fatal("validation deployment not ready")
	}
	if len(decodeIDs(d.Managers)) != 3 {
		t.Fatal("three Raft nodes required")
	}
	s := NewDatabaseAutomationService(db)
	ops := NewDatabaseOperationService(db)
	sec, err := s.secrets(&d)
	if err != nil {
		t.Fatal(err)
	}
	var assets []model.CmdbSQL
	db.Where("deployment_id = ?", d.ID).Find(&assets)
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	user := uint(0)
	db.Table("sys_admin").Select("id").Where("username = ?", "admin").Scan(&user)
	assetByHost := map[uint]model.CmdbSQL{}
	for _, a := range assets {
		assetByHost[a.HostID] = a
	}
	readonly := func(id uint) (bool, error) {
		conn, _, e := ops.openSQL(ctx, assetByHost[id].ID, "")
		if e != nil {
			return true, e
		}
		defer conn.Close()
		var value bool
		e = conn.QueryRowContext(ctx, "SELECT @@global.read_only").Scan(&value)
		return value, e
	}
	findPrimary := func(exclude uint) uint {
		var found uint
		for _, id := range decodeIDs(d.Nodes) {
			if id == exclude {
				continue
			}
			ro, e := readonly(id)
			if e == nil && !ro {
				if found != 0 {
					return 0
				}
				found = id
			}
		}
		return found
	}
	wait := func(label string, fn func() bool) {
		t.Helper()
		deadline := time.Now().Add(150 * time.Second)
		for time.Now().Before(deadline) {
			if fn() {
				t.Log(label + ": passed")
				return
			}
			time.Sleep(2 * time.Second)
		}
		t.Fatal(label + ": timed out")
	}
	service := func(id uint, unit, state string) error {
		conn, e := ResolveAssetSSH(db, "host", id, 0)
		if e != nil {
			return e
		}
		defer conn.Close()
		session, e := conn.NewSession()
		if e != nil {
			return e
		}
		defer session.Close()
		return session.Run("systemctl " + state + " " + unit)
	}
	primary := findPrimary(0)
	if primary == 0 {
		t.Fatal("expected exactly one writable primary")
	}
	connection, _, err := ops.openSQL(ctx, assetByHost[primary].ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = connection.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS autoops_validation"); err != nil {
		t.Fatal(err)
	}
	if _, err = connection.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS autoops_validation.ha_check (id BIGINT PRIMARY KEY, payload VARCHAR(100))"); err != nil {
		t.Fatal(err)
	}
	marker := time.Now().UnixNano()
	if _, err = connection.ExecContext(ctx, "INSERT INTO autoops_validation.ha_check(id,payload) VALUES (?,?)", marker, "replicated-before-failover"); err != nil {
		t.Fatal(err)
	}
	connection.Close()
	dataPresent := func(exclude uint) bool {
		for _, id := range decodeIDs(d.Nodes) {
			if id == exclude {
				continue
			}
			conn, _, e := ops.openSQL(ctx, assetByHost[id].ID, "autoops_validation")
			if e != nil {
				return false
			}
			var payload string
			e = conn.QueryRowContext(ctx, "SELECT payload FROM ha_check WHERE id=?", marker).Scan(&payload)
			conn.Close()
			if e != nil || payload != "replicated-before-failover" {
				return false
			}
		}
		return true
	}
	writeMarker := func(primary uint) {
		t.Helper()
		marker = time.Now().UnixNano()
		conn, _, e := ops.openSQL(ctx, assetByHost[primary].ID, "autoops_validation")
		if e != nil {
			t.Fatal(e)
		}
		defer conn.Close()
		if _, e = conn.ExecContext(ctx, "INSERT INTO ha_check(id,payload) VALUES (?,?)", marker, "replicated-before-failover"); e != nil {
			t.Fatal(e)
		}
	}
	replicationHealthy := func(primary, exclude uint) bool {
		p, e := s.hostConnection(primary)
		if e != nil || findPrimary(exclude) != primary {
			return false
		}
		for _, id := range decodeIDs(d.Nodes) {
			if id == primary || id == exclude {
				continue
			}
			conn, _, e := ops.openSQL(ctx, assetByHost[id].ID, "")
			if e != nil {
				return false
			}
			rows, e := conn.QueryContext(ctx, "SHOW SLAVE STATUS")
			if e != nil {
				conn.Close()
				return false
			}
			columns, _ := rows.Columns()
			values := make([]sql.NullString, len(columns))
			args := make([]any, len(columns))
			for i := range args {
				args[i] = &values[i]
			}
			ok := rows.Next() && rows.Scan(args...) == nil
			status := map[string]string{}
			for i, column := range columns {
				status[column] = values[i].String
			}
			rows.Close()
			conn.Close()
			if !ok || status["Slave_IO_Running"] != "Yes" || status["Slave_SQL_Running"] != "Yes" || status["Master_Host"] != p.Address || status["Master_Port"] != strconv.Itoa(d.Port) || status["Last_IO_Error"] != "" || status["Last_SQL_Error"] != "" {
				return false
			}
		}
		return true
	}
	wait("GTID replication data consistency", func() bool { return dataPresent(0) })
	wait("initial replica threads healthy", func() bool { return replicationHealthy(primary, 0) })
	candidate := uint(0)
	for _, id := range decodeIDs(d.Nodes) {
		if id != primary {
			candidate = id
			break
		}
	}
	if _, err := s.HAAction(ctx, d.ID, user, DatabaseHARequest{Action: "graceful-master-takeover", HostID: primary, CandidateID: candidate, ConfirmName: d.Name}); err != nil {
		t.Fatal(err)
	}
	wait("planned switchover", func() bool { return findPrimary(0) == candidate && dataPresent(0) })
	writeMarker(candidate)
	wait("new writes replicate after planned switchover", func() bool { return replicationHealthy(candidate, 0) && dataPresent(0) })
	client := s.orchestrator(&d, sec)
	var leaderID uint
	for i, endpoint := range client.endpoints {
		if _, err := client.request(ctx, endpoint, "leader-check"); err == nil {
			leaderID = decodeIDs(d.Managers)[i]
			break
		}
	}
	if leaderID == 0 {
		t.Fatal("no Raft leader")
	}
	if err := service(leaderID, "autoops-orchestrator", "stop"); err != nil {
		t.Fatal(err)
	}
	defer service(leaderID, "autoops-orchestrator", "start")
	wait("Raft leader loss and re-election", func() bool { _, e := s.HAView(ctx, d.ID, "health"); return e == nil })
	if err := service(leaderID, "autoops-orchestrator", "start"); err != nil {
		t.Fatal(err)
	}
	if err := service(candidate, "autoops-mysql", "stop"); err != nil {
		t.Fatal(err)
	}
	defer service(candidate, "autoops-mysql", "start")
	var promoted uint
	wait("automatic primary failure recovery", func() bool { promoted = findPrimary(candidate); return promoted != 0 && dataPresent(candidate) })
	writeMarker(promoted)
	wait("new writes replicate after automatic recovery", func() bool { return replicationHealthy(promoted, candidate) && dataPresent(candidate) })
	if err := service(candidate, "autoops-mysql", "start"); err != nil {
		t.Fatal(err)
	}
	wait("restarted former primary stays read-only", func() bool { ro, e := readonly(candidate); return e == nil && ro })
	// The failure injected here is a clean process stop with all writes replicated;
	// rejoin is therefore safe using GTID, without erasing data or resetting GTIDs.
	candidateProfile, err := s.hostConnection(candidate)
	if err != nil {
		t.Fatal(err)
	}
	rejoinConfig := mysqlDriver.NewConfig()
	rejoinConfig.User, rejoinConfig.Passwd, rejoinConfig.Net = "root", sec.Password, "tcp"
	rejoinConfig.Addr = net.JoinHostPort(candidateProfile.Address, strconv.Itoa(d.Port))
	rejoinConfig.InterpolateParams = true
	conn, err := sql.Open("mysql", rejoinConfig.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.hostConnection(promoted)
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{"STOP REPLICA"} {
		if _, err := conn.ExecContext(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := conn.ExecContext(ctx, "CHANGE REPLICATION SOURCE TO SOURCE_HOST=?, SOURCE_PORT=?, SOURCE_USER='autoops_repl', SOURCE_PASSWORD=?, SOURCE_AUTO_POSITION=1", p.Address, d.Port, sec.Replication); err != nil {
		t.Fatal("GTID rejoin failed")
	}
	if _, err := conn.ExecContext(ctx, "START REPLICA"); err != nil {
		t.Fatal(err)
	}
	conn.Close()
	wait("former primary rejoins without data loss", func() bool { return replicationHealthy(promoted, 0) && dataPresent(0) })
	// Restore the initial primary for a predictable test fixture.
	if promoted != decodeIDs(d.Nodes)[0] {
		time.Sleep(6 * time.Second)
		_, err = s.HAAction(ctx, d.ID, user, DatabaseHARequest{Action: "graceful-master-takeover", HostID: promoted, CandidateID: decodeIDs(d.Nodes)[0], ConfirmName: d.Name})
		if err != nil {
			t.Fatal(err)
		}
	}
	wait("final primary writable", func() bool { return findPrimary(0) == decodeIDs(d.Nodes)[0] })
	writeMarker(decodeIDs(d.Nodes)[0])
	wait("final primary and replicas actively synchronizing", func() bool { return replicationHealthy(decodeIDs(d.Nodes)[0], 0) && dataPresent(0) })
	// Refresh every member before saving the final topology evidence.
	for _, id := range decodeIDs(d.Nodes) {
		if _, err = s.HAAction(ctx, d.ID, user, DatabaseHARequest{Action: "refresh", HostID: id, ConfirmName: d.Name}); err != nil {
			t.Fatal(err)
		}
	}
	for _, view := range []string{"topology", "health", "recoveries", "analysis"} {
		data, e := s.HAView(ctx, d.ID, view)
		if e != nil {
			t.Fatal(e)
		}
		raw, _ := json.MarshalIndent(data, "", "  ")
		dir := os.Getenv("AUTOOPS_DB_EVIDENCE_DIR")
		if dir != "" {
			os.WriteFile(dir+"/after-failover-"+view+".json", raw, 0600)
		}
	}
}
