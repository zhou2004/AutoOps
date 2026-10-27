package service

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const workbenchTransferLimit = 8 << 20
const workbenchImportRows = 10000

type MySQLColumn struct {
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Nullable  bool    `json:"nullable"`
	Default   *string `json:"default"`
	Key       string  `json:"key"`
	Extra     string  `json:"extra"`
	Comment   string  `json:"comment"`
	Binary    bool    `json:"binary"`
	Generated bool    `json:"generated"`
}
type MySQLTableInfo struct {
	Database   string           `json:"database"`
	Table      string           `json:"table"`
	Engine     string           `json:"engine"`
	Type       string           `json:"type"`
	Comment    string           `json:"comment"`
	Collation  string           `json:"collation"`
	Columns    []MySQLColumn    `json:"columns"`
	PrimaryKey []string         `json:"primaryKey"`
	Indexes    []map[string]any `json:"indexes"`
	DDL        string           `json:"ddl"`
}
type MySQLDataRequest struct {
	Database string `json:"database"`
	Table    string `json:"table"`
	Page     int    `json:"page"`
	Size     int    `json:"size"`
	Sort     string `json:"sort"`
	Desc     bool   `json:"desc"`
	Column   string `json:"column"`
	Operator string `json:"operator"`
	Value    string `json:"value"`
}
type MySQLDataResult struct {
	Columns []string         `json:"columns"`
	Rows    []map[string]any `json:"rows"`
	Total   int64            `json:"total"`
	Page    int              `json:"page"`
	Size    int              `json:"size"`
}
type MySQLRowRequest struct {
	Database    string         `json:"database"`
	Table       string         `json:"table"`
	Action      string         `json:"action"`
	Values      map[string]any `json:"values"`
	Original    map[string]any `json:"original"`
	ConfirmName string         `json:"confirmName"`
}

func mysqlIdentifier(name string) (string, error) {
	if name == "" || len([]rune(name)) > 64 || strings.ContainsAny(name, "\x00\r\n") {
		return "", errors.New("数据库、表或字段名称无效")
	}
	return "`" + strings.ReplaceAll(name, "`", "``") + "`", nil
}
func mysqlQualified(database, table string) (string, error) {
	a, e := mysqlIdentifier(database)
	if e != nil {
		return "", e
	}
	b, e := mysqlIdentifier(table)
	return a + "." + b, e
}
func systemMySQLDatabase(database string) bool {
	return map[string]bool{"mysql": true, "sys": true, "information_schema": true, "performance_schema": true}[strings.ToLower(database)]
}
func (s *DatabaseOperationService) mysqlConnection(ctx context.Context, id uint, database string) (*sql.DB, *databaseTarget, error) {
	db, target, err := s.openSQL(ctx, id, database)
	if err != nil {
		return nil, nil, err
	}
	if target.Asset.Type != 1 {
		db.Close()
		return nil, nil, errors.New("此工作台操作仅支持 MySQL")
	}
	return db, target, nil
}
func mysqlOperationError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return errors.New("操作超时或已取消")
	}
	// Avoid returning SQL values, passwords or DSNs in database error messages.
	return errors.New("MySQL 操作失败，请检查字段类型、约束、只读状态及数据库账号权限")
}
func mysqlTableInfo(ctx context.Context, db *sql.DB, database, table string) (*MySQLTableInfo, error) {
	qualified, err := mysqlQualified(database, table)
	if err != nil {
		return nil, err
	}
	info := &MySQLTableInfo{Database: database, Table: table, Columns: []MySQLColumn{}, PrimaryKey: []string{}, Indexes: []map[string]any{}}
	err = db.QueryRowContext(ctx, "SELECT TABLE_TYPE,COALESCE(ENGINE,''),TABLE_COMMENT,COALESCE(TABLE_COLLATION,'') FROM information_schema.tables WHERE table_schema=? AND table_name=?", database, table).Scan(&info.Type, &info.Engine, &info.Comment, &info.Collation)
	if err != nil {
		return nil, errors.New("数据表不存在或没有访问权限")
	}
	rows, err := db.QueryContext(ctx, "SELECT COLUMN_NAME,COLUMN_TYPE,IS_NULLABLE,COLUMN_DEFAULT,COLUMN_KEY,EXTRA,COLUMN_COMMENT,DATA_TYPE FROM information_schema.columns WHERE table_schema=? AND table_name=? ORDER BY ORDINAL_POSITION", database, table)
	if err != nil {
		return nil, mysqlOperationError(err)
	}
	for rows.Next() {
		var c MySQLColumn
		var nullable, kind string
		var def sql.NullString
		if err = rows.Scan(&c.Name, &c.Type, &nullable, &def, &c.Key, &c.Extra, &c.Comment, &kind); err != nil {
			rows.Close()
			return nil, err
		}
		c.Nullable = nullable == "YES"
		if def.Valid {
			v := def.String
			c.Default = &v
		}
		c.Generated = strings.Contains(c.Extra, "GENERATED") && !strings.Contains(c.Extra, "DEFAULT_GENERATED")
		c.Binary = regexp.MustCompile(`(?i)blob|binary|geometry|point|polygon|linestring|^bit$`).MatchString(kind)
		info.Columns = append(info.Columns, c)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, mysqlOperationError(err)
	}
	rows, err = db.QueryContext(ctx, "SELECT INDEX_NAME,NON_UNIQUE,SEQ_IN_INDEX,COALESCE(COLUMN_NAME,''),INDEX_TYPE FROM information_schema.statistics WHERE table_schema=? AND table_name=? ORDER BY INDEX_NAME,SEQ_IN_INDEX", database, table)
	if err != nil {
		return nil, mysqlOperationError(err)
	}
	for rows.Next() {
		var name, col, kind string
		var nonUnique, seq int
		if err = rows.Scan(&name, &nonUnique, &seq, &col, &kind); err != nil {
			rows.Close()
			return nil, err
		}
		info.Indexes = append(info.Indexes, map[string]any{"name": name, "unique": nonUnique == 0, "sequence": seq, "column": col, "type": kind})
		if name == "PRIMARY" {
			info.PrimaryKey = append(info.PrimaryKey, col)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, mysqlOperationError(err)
	}
	rows, err = db.QueryContext(ctx, "SHOW CREATE TABLE "+qualified)
	if err != nil {
		return nil, mysqlOperationError(err)
	}
	defer rows.Close()
	if rows.Next() {
		cols, _ := rows.Columns()
		values := make([]sql.RawBytes, len(cols))
		args := make([]any, len(cols))
		for i := range args {
			args[i] = &values[i]
		}
		if rows.Scan(args...) == nil && len(values) > 1 {
			info.DDL = string(values[1])
		}
	}
	return info, rows.Err()
}
func (s *DatabaseOperationService) MySQLTable(ctx context.Context, id uint, database, table string) (*MySQLTableInfo, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	db, _, err := s.mysqlConnection(ctx, id, database)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	return mysqlTableInfo(ctx, db, database, table)
}
func mysqlDataClause(info *MySQLTableInfo, r MySQLDataRequest) (string, []any, error) {
	allowed := map[string]bool{}
	for _, c := range info.Columns {
		allowed[c.Name] = true
	}
	clause := ""
	args := []any{}
	if r.Column != "" {
		if !allowed[r.Column] {
			return "", nil, errors.New("筛选字段不存在")
		}
		col, _ := mysqlIdentifier(r.Column)
		op := map[string]string{"eq": "=", "ne": "<>", "gt": ">", "gte": ">=", "lt": "<", "lte": "<=", "contains": "LIKE", "null": "IS NULL", "not-null": "IS NOT NULL"}[r.Operator]
		if op == "" {
			return "", nil, errors.New("筛选运算符不受支持")
		}
		clause = " WHERE " + col + " " + op
		if r.Operator != "null" && r.Operator != "not-null" {
			clause += " ?"
			value := r.Value
			if r.Operator == "contains" {
				value = "%" + value + "%"
			}
			args = append(args, value)
		}
	}
	return clause, args, nil
}
func mysqlOrder(info *MySQLTableInfo, r MySQLDataRequest) (string, error) {
	names := info.PrimaryKey
	if r.Sort != "" {
		found := false
		for _, c := range info.Columns {
			if c.Name == r.Sort {
				found = true
			}
		}
		if !found {
			return "", errors.New("排序字段不存在")
		}
		names = []string{r.Sort}
	}
	quoted := []string{}
	for _, name := range names {
		q, _ := mysqlIdentifier(name)
		quoted = append(quoted, q)
	}
	if len(quoted) == 0 {
		return "", nil
	}
	out := " ORDER BY " + strings.Join(quoted, ",")
	if r.Desc {
		out += " DESC"
	}
	return out, nil
}
func readMySQLRows(rows *sql.Rows, limit int, info *MySQLTableInfo) ([]string, []map[string]any, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, nil, err
	}
	result := []map[string]any{}
	size := 0
	binary := map[string]bool{}
	for _, c := range info.Columns {
		binary[c.Name] = c.Binary
	}
	for rows.Next() {
		values := make([]any, len(cols))
		args := make([]any, len(cols))
		for i := range args {
			args[i] = &values[i]
		}
		if err := rows.Scan(args...); err != nil {
			return nil, nil, err
		}
		item := map[string]any{}
		for i, v := range values {
			if v != nil {
				switch x := v.(type) {
				case []byte:
					if binary[cols[i]] {
						v = fmt.Sprintf("0x%x", x)
					} else {
						v = string(x)
					}
				case time.Time:
					v = x.Format("2006-01-02 15:04:05.999999")
				case int64:
					v = strconv.FormatInt(x, 10)
				case uint64:
					v = strconv.FormatUint(x, 10)
				}
			}
			item[cols[i]] = v
		}
		raw, _ := json.Marshal(item)
		size += len(raw)
		if size > workbenchTransferLimit {
			return nil, nil, errors.New("结果超过 8 MiB，请缩小筛选范围或每页行数")
		}
		result = append(result, item)
		if len(result) > limit {
			return nil, nil, fmt.Errorf("结果超过 %d 行，请缩小筛选范围后导出", limit)
		}
	}
	return cols, result, rows.Err()
}
func (s *DatabaseOperationService) MySQLData(ctx context.Context, id uint, r MySQLDataRequest, export bool) (*MySQLDataResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if r.Page < 1 {
		r.Page = 1
	}
	if r.Page > 100000 {
		return nil, errors.New("页码超出范围")
	}
	if r.Size < 1 || r.Size > 500 {
		r.Size = 50
	}
	if export {
		r.Page = 1
		r.Size = workbenchImportRows
	}
	db, _, err := s.mysqlConnection(ctx, id, r.Database)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	info, err := mysqlTableInfo(ctx, db, r.Database, r.Table)
	if err != nil {
		return nil, err
	}
	table, _ := mysqlQualified(r.Database, r.Table)
	where, args, err := mysqlDataClause(info, r)
	if err != nil {
		return nil, err
	}
	order, err := mysqlOrder(info, r)
	if err != nil {
		return nil, err
	}
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, mysqlOperationError(err)
	}
	defer tx.Rollback()
	out := &MySQLDataResult{Page: r.Page, Size: r.Size}
	if err = tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table+where, args...).Scan(&out.Total); err != nil {
		return nil, mysqlOperationError(err)
	}
	if export && out.Total > workbenchImportRows {
		return nil, errors.New("单次导出最多 10000 行，请增加筛选条件")
	}
	q := "SELECT * FROM " + table + where + order + " LIMIT ? OFFSET ?"
	args = append(args, r.Size, (r.Page-1)*r.Size)
	rows, err := tx.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, mysqlOperationError(err)
	}
	defer rows.Close()
	out.Columns, out.Rows, err = readMySQLRows(rows, r.Size, info)
	return out, err
}
func checkedMySQLValues(info *MySQLTableInfo, values map[string]any) ([]string, []any, error) {
	if len(values) == 0 {
		return nil, nil, errors.New("至少填写一个字段")
	}
	columns := map[string]MySQLColumn{}
	for _, c := range info.Columns {
		columns[c.Name] = c
	}
	names := []string{}
	for name := range values {
		names = append(names, name)
	}
	sort.Strings(names)
	args := []any{}
	for _, name := range names {
		c, ok := columns[name]
		if !ok || c.Generated || c.Binary {
			return nil, nil, fmt.Errorf("字段 %s 不存在或不支持直接写入", name)
		}
		value := values[name]
		switch value.(type) {
		case nil, string, bool, float64, json.Number:
		default:
			return nil, nil, errors.New("字段值必须为文本、数字、布尔值或 NULL")
		}
		if value == nil && !c.Nullable && !strings.Contains(c.Extra, "auto_increment") {
			return nil, nil, fmt.Errorf("字段 %s 不允许 NULL", name)
		}
		if n, ok := value.(json.Number); ok {
			value = n.String()
		}
		args = append(args, value)
	}
	return names, args, nil
}
func mysqlRowStatement(info *MySQLTableInfo, r MySQLRowRequest) (string, []any, error) {
	if systemMySQLDatabase(r.Database) || info.Type != "BASE TABLE" || info.Engine != "InnoDB" {
		return "", nil, errors.New("可视化写入仅支持业务库的 InnoDB 基表")
	}
	qualified, err := mysqlQualified(r.Database, r.Table)
	if err != nil {
		return "", nil, err
	}
	if r.Action == "insert" {
		names, args, e := checkedMySQLValues(info, r.Values)
		if e != nil {
			return "", nil, e
		}
		qs := []string{}
		for _, n := range names {
			q, _ := mysqlIdentifier(n)
			qs = append(qs, q)
		}
		return "INSERT INTO " + qualified + " (" + strings.Join(qs, ",") + ") VALUES (" + strings.TrimSuffix(strings.Repeat("?,", len(qs)), ",") + ")", args, nil
	}
	if r.Action != "update" && r.Action != "delete" {
		return "", nil, errors.New("不支持的行操作")
	}
	if len(info.PrimaryKey) == 0 {
		return "", nil, errors.New("没有主键的表不能定位单行，请使用 SQL 控制台")
	}
	for _, key := range info.PrimaryKey {
		if value, ok := r.Original[key]; !ok || value == nil {
			return "", nil, errors.New("必须提供完整主键")
		}
	}
	where := []string{}
	args := []any{}
	// Compare all readable original fields, not just the key, to prevent lost updates.
	for _, c := range info.Columns {
		if c.Generated {
			continue
		}
		v, ok := r.Original[c.Name]
		if !ok {
			return "", nil, errors.New("原始记录不完整，请重新加载表数据")
		}
		if n, ok := v.(json.Number); ok {
			v = n.String()
		}
		switch v.(type) {
		case nil, string, bool, float64:
		default:
			return "", nil, errors.New("原始记录类型无效")
		}
		q, _ := mysqlIdentifier(c.Name)
		if c.Binary {
			if v != nil {
				value, ok := v.(string)
				if !ok || !strings.HasPrefix(value, "0x") {
					return "", nil, errors.New("二进制原始记录格式无效")
				}
				if _, e := hex.DecodeString(value[2:]); e != nil {
					return "", nil, errors.New("二进制原始记录格式无效")
				}
				v = value[2:]
			}
			where = append(where, q+" <=> UNHEX(?)")
		} else if regexp.MustCompile(`(?i)char|text|json`).MatchString(c.Type) {
			where = append(where, "BINARY "+q+" <=> BINARY ?")
		} else {
			where = append(where, q+" <=> ?")
		}
		args = append(args, v)
	}
	prefix := "DELETE FROM " + qualified
	if r.Action == "update" {
		names, values, e := checkedMySQLValues(info, r.Values)
		if e != nil {
			return "", nil, e
		}
		sets := []string{}
		for _, n := range names {
			q, _ := mysqlIdentifier(n)
			sets = append(sets, q+"=?")
		}
		prefix = "UPDATE " + qualified + " SET " + strings.Join(sets, ",")
		args = append(values, args...)
	}
	return prefix + " WHERE " + strings.Join(where, " AND ") + " LIMIT 1", args, nil
}
func (s *DatabaseOperationService) MySQLMutateRow(ctx context.Context, id uint, r MySQLRowRequest) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	db, target, err := s.mysqlConnection(ctx, id, r.Database)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	if r.ConfirmName != target.Asset.Name {
		return 0, errors.New("请输入完整数据源名称确认写操作")
	}
	info, err := mysqlTableInfo(ctx, db, r.Database, r.Table)
	if err != nil {
		return 0, err
	}
	q, args, err := mysqlRowStatement(info, r)
	if err != nil {
		return 0, err
	}
	res, err := db.ExecContext(ctx, q, args...)
	if err != nil {
		return 0, mysqlOperationError(err)
	}
	n, _ := res.RowsAffected()
	if n == 0 && r.Action != "insert" {
		return 0, errors.New("记录已变化、已删除或没有修改，请刷新后重试")
	}
	return n, nil
}
func (s *DatabaseOperationService) MySQLImport(ctx context.Context, id uint, database, table, format, confirm string, content []byte) (int64, error) {
	if len(content) > workbenchTransferLimit {
		return 0, errors.New("导入文件不能超过 8 MiB")
	}
	records, err := parseMySQLImport(format, content)
	if err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	db, target, err := s.mysqlConnection(ctx, id, database)
	if err != nil {
		return 0, err
	}
	defer db.Close()
	if confirm != target.Asset.Name {
		return 0, errors.New("请输入完整数据源名称确认导入")
	}
	info, err := mysqlTableInfo(ctx, db, database, table)
	if err != nil {
		return 0, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, mysqlOperationError(err)
	}
	defer tx.Rollback()
	for _, row := range records {
		for _, column := range info.Columns {
			if column.Generated {
				delete(row, column.Name)
			}
		}
	}
	for i, row := range records {
		q, args, e := mysqlRowStatement(info, MySQLRowRequest{Database: database, Table: table, Action: "insert", Values: row})
		if e != nil {
			return 0, fmt.Errorf("第 %d 行：%s", i+1, e)
		}
		if _, e = tx.ExecContext(ctx, q, args...); e != nil {
			return 0, fmt.Errorf("第 %d 行导入失败（类型、主键或约束冲突），本次导入已回滚", i+1)
		}
	}
	if err = tx.Commit(); err != nil {
		return 0, mysqlOperationError(err)
	}
	return int64(len(records)), nil
}
func parseMySQLImport(format string, content []byte) ([]map[string]any, error) {
	if len(content) > workbenchTransferLimit || !utf8.Valid(content) {
		return nil, errors.New("文件必须使用 UTF-8 编码且不超过 8 MiB")
	}
	content = bytes.TrimPrefix(content, []byte{0xef, 0xbb, 0xbf})
	rows := []map[string]any{}
	switch format {
	case "json":
		d := json.NewDecoder(bytes.NewReader(content))
		d.UseNumber()
		if d.Decode(&rows) != nil {
			return nil, errors.New("JSON 必须是记录对象数组")
		}
		var trailing any
		if d.Decode(&trailing) != io.EOF {
			return nil, errors.New("JSON 文件包含额外内容")
		}
	case "csv":
		reader := csv.NewReader(bytes.NewReader(content))
		header, err := reader.Read()
		if err != nil {
			return nil, errors.New("CSV 缺少字段名表头")
		}
		seen := map[string]bool{}
		for _, col := range header {
			if col == "" || seen[col] {
				return nil, errors.New("CSV 字段名为空或重复")
			}
			seen[col] = true
		}
		for {
			line, e := reader.Read()
			if e == io.EOF {
				break
			}
			if e != nil || len(line) != len(header) {
				return nil, errors.New("CSV 行列数量或引号格式错误")
			}
			row := map[string]any{}
			for i, v := range line {
				if v == `\N` {
					row[header[i]] = nil
				} else if strings.HasPrefix(v, `\\`) {
					row[header[i]] = v[1:]
				} else {
					row[header[i]] = v
				}
			}
			rows = append(rows, row)
			if len(rows) > workbenchImportRows {
				break
			}
		}
	default:
		return nil, errors.New("仅支持 CSV 或 JSON 数据文件")
	}
	if len(rows) == 0 || len(rows) > workbenchImportRows {
		return nil, errors.New("单次导入需为 1–10000 条记录")
	}
	return rows, nil
}
func EncodeMySQLExport(data *MySQLDataResult, format string) ([]byte, error) {
	if format == "json" {
		return json.MarshalIndent(data.Rows, "", "  ")
	}
	if format != "csv" {
		return nil, errors.New("仅支持 CSV 或 JSON 导出")
	}
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	_ = w.Write(data.Columns)
	for _, row := range data.Rows {
		line := []string{}
		for _, c := range data.Columns {
			v := row[c]
			s := `\N`
			if v != nil {
				s = fmt.Sprint(v)
				if strings.HasPrefix(s, `\`) {
					s = `\` + s
				}
			}
			line = append(line, s)
		}
		if e := w.Write(line); e != nil {
			return nil, e
		}
	}
	w.Flush()
	return b.Bytes(), w.Error()
}

type MySQLDefinitionRequest struct {
	Action   string `json:"action"`
	Database string `json:"database"`
	Table    string `json:"table"`
	Charset  string `json:"charset"`
	Columns  []struct {
		Name          string `json:"name"`
		Type          string `json:"type"`
		Nullable      bool   `json:"nullable"`
		Primary       bool   `json:"primary"`
		AutoIncrement bool   `json:"autoIncrement"`
	} `json:"columns"`
	ConfirmName string `json:"confirmName"`
}

func mysqlDefinition(r MySQLDefinitionRequest) (string, error) {
	db, err := mysqlIdentifier(r.Database)
	if err != nil {
		return "", err
	}
	if systemMySQLDatabase(r.Database) {
		return "", errors.New("不能修改系统数据库")
	}
	charset := r.Charset
	if charset == "" {
		charset = "utf8mb4"
	}
	if charset != "utf8mb4" && charset != "utf8" && charset != "latin1" {
		return "", errors.New("字符集不受支持")
	}
	if r.Action == "create-database" {
		return "CREATE DATABASE " + db + " CHARACTER SET " + charset, nil
	}
	if r.Action == "alter-database" {
		return "ALTER DATABASE " + db + " CHARACTER SET " + charset, nil
	}
	table, err := mysqlQualified(r.Database, r.Table)
	if err != nil {
		return "", err
	}
	if r.Action == "drop-table" {
		return "DROP TABLE " + table, nil
	}
	if r.Action != "create-table" || len(r.Columns) == 0 || len(r.Columns) > 100 {
		return "", errors.New("建表需要 1–100 个字段")
	}
	parts := []string{}
	keys := []string{}
	seen := map[string]bool{}
	autoCount := 0
	for _, c := range r.Columns {
		q, e := mysqlIdentifier(c.Name)
		if e != nil {
			return "", e
		}
		if seen[strings.ToLower(c.Name)] {
			return "", errors.New("字段名重复")
		}
		seen[strings.ToLower(c.Name)] = true
		kind := strings.ToUpper(c.Type)
		if !regexp.MustCompile(`^(?:INT|BIGINT|TINYINT|TEXT|LONGTEXT|DATETIME|TIMESTAMP|DATE|JSON|BOOLEAN|DOUBLE|VARCHAR\([1-9][0-9]{0,3}\)|DECIMAL\([1-9][0-9]?,[0-9]{1,2}\))$`).MatchString(kind) {
			return "", errors.New("字段类型不受支持")
		}
		field := q + " " + kind
		if c.Primary || !c.Nullable {
			field += " NOT NULL"
		} else {
			field += " NULL"
		}
		if c.Primary {
			keys = append(keys, q)
		}
		if c.AutoIncrement {
			autoCount++
			if !c.Primary || (kind != "INT" && kind != "BIGINT") {
				return "", errors.New("自增字段必须是 INT/BIGINT 主键")
			}
			field += " AUTO_INCREMENT"
		}
		parts = append(parts, field)
	}
	if autoCount > 1 {
		return "", errors.New("最多一个自增字段")
	}
	if len(keys) > 0 {
		parts = append(parts, "PRIMARY KEY ("+strings.Join(keys, ",")+")")
	}
	return "CREATE TABLE " + table + " (" + strings.Join(parts, ",") + ") ENGINE=InnoDB DEFAULT CHARSET=" + charset, nil
}
func (s *DatabaseOperationService) MySQLDefinition(ctx context.Context, id uint, r MySQLDefinitionRequest) error {
	q, err := mysqlDefinition(r)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	db, target, err := s.mysqlConnection(ctx, id, "")
	if err != nil {
		return err
	}
	defer db.Close()
	if r.ConfirmName != target.Asset.Name {
		return errors.New("请输入完整数据源名称确认结构操作")
	}
	if _, err = db.ExecContext(ctx, q); err != nil {
		return mysqlOperationError(err)
	}
	return nil
}

type MySQLVariable struct {
	Name  string  `json:"name"`
	Value string  `json:"value"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	Label string  `json:"label"`
}

var mysqlRuntimeVariables = []MySQLVariable{{Name: "max_connections", Min: 10, Max: 10000, Label: "最大连接数"}, {Name: "wait_timeout", Min: 1, Max: 31536000, Label: "非交互连接空闲超时（秒）"}, {Name: "interactive_timeout", Min: 1, Max: 31536000, Label: "交互连接空闲超时（秒）"}, {Name: "long_query_time", Min: 0, Max: 3600, Label: "慢查询阈值（秒）"}, {Name: "slow_query_log", Min: 0, Max: 1, Label: "慢查询日志（0/1）"}}

func (s *DatabaseOperationService) MySQLVariables(ctx context.Context, id uint) ([]MySQLVariable, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	db, _, err := s.mysqlConnection(ctx, id, "")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	out := append([]MySQLVariable{}, mysqlRuntimeVariables...)
	for i := range out {
		if err = db.QueryRowContext(ctx, "SELECT @@GLOBAL."+out[i].Name).Scan(&out[i].Value); err != nil {
			return nil, mysqlOperationError(err)
		}
	}
	return out, nil
}
func (s *DatabaseOperationService) MySQLSetVariable(ctx context.Context, id uint, name, value, confirm string) error {
	var definition *MySQLVariable
	for i := range mysqlRuntimeVariables {
		if mysqlRuntimeVariables[i].Name == name {
			definition = &mysqlRuntimeVariables[i]
		}
	}
	if definition == nil {
		return errors.New("该运行参数不允许修改")
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil || !regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?$`).MatchString(value) || number < definition.Min || number > definition.Max || (name != "long_query_time" && number != float64(int64(number))) {
		return errors.New("参数值不在允许范围内")
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	db, target, err := s.mysqlConnection(ctx, id, "")
	if err != nil {
		return err
	}
	defer db.Close()
	if confirm != target.Asset.Name {
		return errors.New("请输入完整数据源名称确认配置变更")
	}
	if _, err = db.ExecContext(ctx, "SET GLOBAL "+name+" = "+strconv.FormatFloat(number, 'f', -1, 64)); err != nil {
		return mysqlOperationError(err)
	}
	return nil
}
