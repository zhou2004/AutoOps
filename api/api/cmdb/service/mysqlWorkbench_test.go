package service

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestMySQLTransferRoundTrip(t *testing.T) {
	data := &MySQLDataResult{Columns: []string{"id", "name", "note", "optional"}, Rows: []map[string]any{{"id": "9223372036854775807", "name": "中文,\"quoted\"\nline", "note": `\N`, "optional": nil}, {"id": "2", "name": "", "note": `\\prefix`, "optional": "NULL"}}}
	for _, format := range []string{"csv", "json"} {
		raw, err := EncodeMySQLExport(data, format)
		if err != nil {
			t.Fatal(err)
		}
		got, err := parseMySQLImport(format, raw)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, data.Rows) {
			t.Fatalf("%s round trip corrupted values: %#v", format, got)
		}
	}
	for _, raw := range []string{`[{"id":1}] false`, `{"id":1}`, `[]`} {
		if _, err := parseMySQLImport("json", []byte(raw)); err == nil {
			t.Fatal("invalid JSON accepted")
		}
	}
	for _, raw := range []string{"id,id\n1,2", "id,name\n1", ""} {
		if _, err := parseMySQLImport("csv", []byte(raw)); err == nil {
			t.Fatal("invalid CSV accepted")
		}
	}
	rows, err := parseMySQLImport("json", []byte(`[{"id":9223372036854775807}]`))
	if err != nil || rows[0]["id"] != json.Number("9223372036854775807") {
		t.Fatal("JSON integer lost precision")
	}
}
func TestMySQLRowGuardsAndParameterizedPredicates(t *testing.T) {
	info := &MySQLTableInfo{Engine: "InnoDB", Type: "BASE TABLE", PrimaryKey: []string{"id"}, Columns: []MySQLColumn{{Name: "id", Type: "bigint"}, {Name: "name", Type: "varchar(100)", Nullable: true}}}
	r := MySQLRowRequest{Database: "business", Table: "users", Action: "update", Values: map[string]any{"name": "x' OR 1=1 --"}, Original: map[string]any{"id": "1", "name": "previous"}}
	q, args, err := mysqlRowStatement(info, r)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(q, "OR 1=1") || len(args) != 3 || !strings.Contains(q, "BINARY `name` <=> BINARY ?") || !strings.HasSuffix(q, "LIMIT 1") {
		t.Fatalf("unsafe update: %s %#v", q, args)
	}
	for _, change := range []func(*MySQLRowRequest){func(r *MySQLRowRequest) { r.Database = "mysql" }, func(r *MySQLRowRequest) { r.Original = map[string]any{"name": "previous"} }, func(r *MySQLRowRequest) { r.Values = map[string]any{"missing": 1} }} {
		copy := r
		change(&copy)
		if _, _, e := mysqlRowStatement(info, copy); e == nil {
			t.Fatal("invalid write accepted")
		}
	}
	info.PrimaryKey = nil
	if _, _, e := mysqlRowStatement(info, r); e == nil {
		t.Fatal("keyless update accepted")
	}
	if _, _, e := mysqlDataClause(info, MySQLDataRequest{Column: "name` OR 1=1", Operator: "eq"}); e == nil {
		t.Fatal("injected column accepted")
	}
	if _, e := mysqlOrder(info, MySQLDataRequest{Sort: "id;DROP TABLE users"}); e == nil {
		t.Fatal("injected sort accepted")
	}
}
func TestMySQLDefinitionsAndSQLLexing(t *testing.T) {
	if _, err := mysqlDefinition(MySQLDefinitionRequest{Action: "create-database", Database: "mysql"}); err == nil {
		t.Fatal("system schema mutation accepted")
	}
	if _, err := mysqlDefinition(MySQLDefinitionRequest{Action: "create-database", Database: "test", Charset: "utf8;DROP DATABASE mysql"}); err == nil {
		t.Fatal("charset injection accepted")
	}
	for _, query := range []string{"SELECT 'a;b' AS text;", "SELECT `semi;column` FROM t; -- comment", "/* hello */ SELECT 1;"} {
		if op, e := sqlOperation(query); e != nil || op != "SELECT" {
			t.Fatalf("valid query rejected: %s %v", query, e)
		}
	}
	for _, query := range []string{"SELECT 1; DELETE FROM users", "SELECT 1 /*! INTO OUTFILE '/tmp/leak' */", "SELECT 'unterminated"} {
		if _, e := sqlOperation(query); e == nil {
			t.Fatal("unsafe SQL accepted")
		}
	}
	words, _ := sqlWords("DELETE FROM t /* WHERE id=1 */ -- WHERE\n")
	for _, word := range words {
		if word == "WHERE" {
			t.Fatal("comment bypasses WHERE guard")
		}
	}
	if normalizeSQLValue(int64(9223372036854775807)) != "9223372036854775807" {
		t.Fatal("query integer precision lost")
	}
}

func TestBinaryPrimaryKeyDoesNotLoseRowPredicate(t *testing.T) {
	info := &MySQLTableInfo{Engine: "InnoDB", Type: "BASE TABLE", PrimaryKey: []string{"id"}, Columns: []MySQLColumn{{Name: "id", Type: "binary(16)", Binary: true}, {Name: "name", Type: "text"}}}
	q, args, e := mysqlRowStatement(info, MySQLRowRequest{Database: "app", Table: "items", Action: "delete", Original: map[string]any{"id": "0x000102", "name": "original"}})
	if e != nil || !strings.Contains(q, "`id` <=> UNHEX(?)") || args[0] != "000102" {
		t.Fatalf("binary key predicate lost: %s %#v %v", q, args, e)
	}
	if _, e := parseMySQLImport("csv", []byte{0xff, 0xfe}); e == nil {
		t.Fatal("non UTF-8 import accepted")
	}
}
