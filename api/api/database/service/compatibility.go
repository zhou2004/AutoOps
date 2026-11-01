package service

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"dodevops-api/api/database/model"
)

var serverVersion = regexp.MustCompile(`^(\d+)\.(\d+)(?:\.(\d+))?`)

func compatibilityReason(c *model.Cluster) string {
	if c.Status == "planned" {
		return ""
	}
	count := 0
	for _, m := range c.Members {
		if m.Kind != "data" {
			continue
		}
		count++
		v := serverVersion.FindStringSubmatch(m.Version)
		if len(v) == 0 {
			return "尚未取得所有数据实例的版本，仅开放观察与状态采集"
		}
		major, _ := strconv.Atoi(v[1])
		minor, _ := strconv.Atoi(v[2])
		ok := (c.Engine == "mysql" && major == 8 && minor == 0 && !strings.Contains(strings.ToLower(m.Version), "mariadb")) ||
			(c.Engine == "postgresql" && major == 15) || (c.Engine == "redis" && major == 7 && minor == 0)
		if !ok {
			return fmt.Sprintf("%s %s 尚未通过当前适配器兼容验证，保持观察；支持 MySQL 8.0、PostgreSQL 15、Redis 7.0", m.Name, m.Version)
		}
	}
	if count == 0 {
		return "没有可核验版本的数据实例"
	}
	return ""
}

func validateTLS(engine, mode string) error {
	valid := mode == "" || mode == "disable" || mode == "verify-full"
	if engine == "mysql" {
		valid = valid || mode == "require"
	}
	if engine == "postgresql" {
		valid = valid || mode == "prefer" || mode == "require" || mode == "verify-ca"
	}
	if !valid {
		return errors.New("当前引擎不支持此 TLS 策略；不能静默退回明文连接")
	}
	return nil
}
