package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"dodevops-api/common/config"
	"dodevops-api/internal/datacenterseed"
	driver "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	path := flag.String("config", "config.yaml", "API 配置文件")
	apply := flag.Bool("apply", false, "明确写入测试台账；省略时仅检查数据库")
	flag.Parse()
	if err := run(*path, *apply); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(path string, apply bool) error {
	if err := config.LoadConfig(path); err != nil {
		return err
	}
	c := config.Config.Db
	cfg := driver.NewConfig()
	cfg.User = c.Username
	cfg.Passwd = c.Password
	cfg.Net = "tcp"
	cfg.Addr = net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
	cfg.DBName = c.Db
	// Match the API connection policy for historical CMDB schemas, which can
	// retain obsolete non-null columns from earlier model field names.
	cfg.Params = map[string]string{"sql_mode": "''", "charset": c.Charset}
	cfg.ParseTime = true
	cfg.Timeout = 5 * time.Second
	db, err := gorm.Open(mysql.Open(cfg.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		return fmt.Errorf("无法连接配置中的数据库: %w", err)
	}
	raw, err := db.DB()
	if err != nil {
		return err
	}
	defer raw.Close()
	counts := map[string]int64{}
	for _, table := range []string{"cmdb_idc", "cmdb_cabinet", "cmdb_physical_machine", "cmdb_network_device"} {
		var n int64
		if err = db.Table(table).Count(&n).Error; err != nil {
			return err
		}
		counts[table] = n
	}
	output := map[string]any{"database": c.Db, "before": counts, "apply": apply}
	if apply {
		receipt, err := datacenterseed.Seed(db)
		if err != nil {
			return err
		}
		output["receipt"] = receipt
	}
	return json.NewEncoder(os.Stdout).Encode(output)
}
