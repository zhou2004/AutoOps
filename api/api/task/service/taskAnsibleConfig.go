package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"dodevops-api/api/task/model"
)

func (s *TaskAnsibleServiceImpl) prepareAnsibleManualConfig(task *model.TaskAnsible, directory string) error {
	if task.UseConfig != 1 && task.HostGroups != "" {
		var groups map[string][]uint
		if err := json.Unmarshal([]byte(task.HostGroups), &groups); err != nil {
			return fmt.Errorf("解析主机分组失败: %w", err)
		}
		hosts, err := s.GetHostSSHInfo(groups)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(directory, "hosts"), []byte(hosts.GenerateInventory()), 0600); err != nil {
			return err
		}
	}
	if (task.UseConfig != 1 || task.GlobalVarsConfigID == nil) && task.GlobalVars != "" {
		return writeAnsibleGlobalVars(directory, task.GlobalVars)
	}
	return nil
}

func writeAnsibleGlobalVars(directory, content string) error {
	var values map[string]string
	if err := json.Unmarshal([]byte(content), &values); err != nil {
		return fmt.Errorf("解析全局变量失败: %w", err)
	}
	if values == nil {
		values = map[string]string{}
	}
	// JSON is valid YAML, and encoding preserves quotes, newlines and empty values.
	data, err := json.Marshal(values)
	if err != nil {
		return err
	}
	path := filepath.Join(directory, "vars", "all.yml")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
