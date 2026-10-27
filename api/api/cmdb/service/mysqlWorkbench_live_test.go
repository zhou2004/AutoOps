package service

import (
	"context"
	"dodevops-api/api/cmdb/model"
	"dodevops-api/common/config"
	"encoding/json"
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"os"
	"testing"
	"time"
)

func TestMySQLWorkbenchLive(t *testing.T) {
	if os.Getenv("AUTOOPS_MYSQL_WORKBENCH_LIVE") != "1" {
		t.Skip("opt in to test the isolated CMDB MySQL validation source")
	}
	if err := config.LoadConfig(os.Getenv("CMDB_CONFIG_PATH")); err != nil {
		t.Fatal(err)
	}
	cfg := config.GetConfig()
	db, err := gorm.Open(mysql.Open(fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local", cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Db, cfg.Charset)), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal("CMDB unavailable")
	}
	s := NewDatabaseOperationService(db)
	var assets []model.CmdbSQL
	if err = db.Where("deployment_id = ? AND type = ?", 1, 1).Find(&assets).Error; err != nil {
		t.Fatal("validation assets unavailable")
	}
	var asset model.CmdbSQL
	for _, candidate := range assets {
		connection, _, e := s.mysqlConnection(context.Background(), candidate.ID, "")
		if e != nil {
			continue
		}
		var readonly bool
		e = connection.QueryRow("SELECT @@global.read_only").Scan(&readonly)
		connection.Close()
		if e == nil && !readonly {
			if asset.ID != 0 {
				t.Fatal("more than one writable validation primary")
			}
			asset = candidate
		}
	}
	if asset.ID == 0 {
		t.Fatal("no writable validation primary")
	}
	t.Logf("Testing current primary source %d / CMDB host %d", asset.ID, asset.HostID)
	var deployment model.DatabaseDeployment
	if db.First(&deployment, 1).Error != nil || deployment.Name != "autoops-mysql-validation" || deployment.Port != 13306 {
		t.Fatal("refusing a non-validation deployment")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	name := fmt.Sprintf("autoops_workbench_%d", time.Now().UnixNano())
	quoted, _ := mysqlIdentifier(name)
	conn, _, err := s.mysqlConnection(ctx, asset.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if err = s.MySQLDefinition(ctx, asset.ID, MySQLDefinitionRequest{Action: "create-database", Database: name, ConfirmName: asset.Name}); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, e := conn.ExecContext(context.Background(), "DROP DATABASE IF EXISTS "+quoted); e != nil {
			t.Errorf("test schema cleanup failed: %v", e)
		}
	}()
	var definition MySQLDefinitionRequest
	if err = json.Unmarshal([]byte(`{"action":"create-table","table":"records","columns":[{"name":"id","type":"BIGINT","primary":true},{"name":"name","type":"VARCHAR(255)","nullable":false},{"name":"note","type":"TEXT","nullable":true},{"name":"amount","type":"DECIMAL(20,4)","nullable":true}]}`), &definition); err != nil {
		t.Fatal(err)
	}
	definition.Database = name
	definition.ConfirmName = asset.Name
	if err = s.MySQLDefinition(ctx, asset.ID, definition); err != nil {
		t.Fatal(err)
	}
	if _, err = conn.ExecContext(ctx, "ALTER TABLE "+quoted+".records ADD COLUMN name_length INT GENERATED ALWAYS AS (char_length(name)) STORED"); err != nil {
		t.Fatal(err)
	}
	info, err := s.MySQLTable(ctx, asset.ID, name, "records")
	if err != nil || len(info.Columns) != 5 || len(info.PrimaryKey) != 1 || !info.Columns[4].Generated {
		t.Fatalf("metadata: %+v %v", info, err)
	}
	row := MySQLRowRequest{Database: name, Table: "records", Action: "insert", Values: map[string]any{"id": "9223372036854775807", "name": "中文,';\\ value", "note": nil, "amount": "1234567890123456.1234"}, ConfirmName: asset.Name}
	if _, err = s.MySQLMutateRow(ctx, asset.ID, row); err != nil {
		t.Fatal(err)
	}
	r := MySQLDataRequest{Database: name, Table: "records", Size: 25}
	data, err := s.MySQLData(ctx, asset.ID, r, false)
	if err != nil || data.Total != 1 || data.Rows[0]["id"] != "9223372036854775807" || data.Rows[0]["note"] != nil {
		t.Fatalf("lossless data: %+v %v", data, err)
	}
	original := data.Rows[0]
	row.Action = "update"
	row.Original = original
	row.Values = map[string]any{"name": "changed"}
	if _, err = s.MySQLMutateRow(ctx, asset.ID, row); err != nil {
		t.Fatal(err)
	}
	current, e := s.MySQLData(ctx, asset.ID, r, false)
	if e != nil {
		t.Fatal(e)
	}
	row.Original = current.Rows[0]
	if _, e = conn.ExecContext(ctx, "UPDATE "+quoted+".records SET name='CHANGED'"); e != nil {
		t.Fatal(e)
	}
	row.Values = map[string]any{"name": "stale"}
	if _, err = s.MySQLMutateRow(ctx, asset.ID, row); err == nil {
		t.Fatal("lost-update protection failed")
	}
	r.Column = "name"
	r.Operator = "eq"
	r.Value = "changed"
	data, err = s.MySQLData(ctx, asset.ID, r, true)
	if err != nil || len(data.Rows) != 1 {
		t.Fatal("filter/export failed", err)
	}
	// A generated value in an exported file must be ignored and recalculated on import.
	for _, format := range []string{"csv", "json"} {
		definition.Table = "copy_" + format
		if err = s.MySQLDefinition(ctx, asset.ID, definition); err != nil {
			t.Fatal(err)
		}
		if _, e := conn.ExecContext(ctx, "ALTER TABLE "+quoted+"."+definition.Table+" ADD COLUMN name_length INT GENERATED ALWAYS AS (char_length(name)) STORED"); e != nil {
			t.Fatal(e)
		}
		raw, e := EncodeMySQLExport(data, format)
		if e != nil {
			t.Fatal(e)
		}
		n, e := s.MySQLImport(ctx, asset.ID, name, definition.Table, format, asset.Name, raw)
		if e != nil || n != 1 {
			t.Fatalf("%s import: %d %v", format, n, e)
		}
		copy, e := s.MySQLData(ctx, asset.ID, MySQLDataRequest{Database: name, Table: definition.Table}, false)
		if e != nil || copy.Rows[0]["amount"] != "1234567890123456.1234" || copy.Rows[0]["note"] != nil || copy.Rows[0]["name_length"] != "7" {
			t.Fatal("transfer corrupted data", e)
		}
	}
	bad := []byte("id,name\n1,new-row\n9223372036854775807,duplicate\n")
	if _, err = s.MySQLImport(ctx, asset.ID, name, "copy_csv", "csv", asset.Name, bad); err == nil {
		t.Fatal("duplicate import should fail")
	}
	copy, err := s.MySQLData(ctx, asset.ID, MySQLDataRequest{Database: name, Table: "copy_csv"}, false)
	if err != nil || copy.Total != 1 {
		t.Fatal("failed import was not rolled back", err)
	}
	if _, err = s.MySQLMutateRow(ctx, asset.ID, MySQLRowRequest{Database: name, Table: "copy_csv", Action: "delete", Original: copy.Rows[0], ConfirmName: asset.Name}); err != nil {
		t.Fatal(err)
	}
	settings, e := s.MySQLVariables(ctx, asset.ID)
	if e != nil {
		t.Fatal(e)
	}
	if e = s.MySQLSetVariable(ctx, asset.ID, "max_connections", settings[0].Value, asset.Name); e != nil {
		t.Fatal(e)
	}
	if err = s.MySQLSetVariable(ctx, asset.ID, "read_only", "0", asset.Name); err == nil {
		t.Fatal("unapproved global variable accepted")
	}
	if _, _, err = s.Query(ctx, asset.ID, name, "SELECT 'semi;colon' AS text"); err != nil {
		t.Fatal(err)
	}
	if _, _, err = s.Execute(ctx, asset.ID, name, "DELETE FROM records /* WHERE 1=1 */", asset.Name); err == nil {
		t.Fatal("comment bypassed WHERE protection")
	}
	t.Log("PASS: schema/columns/indexes/DDL, BIGINT/DECIMAL/NULL, pagination/filter, row CRUD, conflict protection, CSV+JSON transfer including generated columns, atomic import rollback, runtime variable read/write and SQL guards; test schema cleaned up")
}
