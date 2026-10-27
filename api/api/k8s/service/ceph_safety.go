package service

import (
	"dodevops-api/api/k8s/model"
	"encoding/json"
	"fmt"
	"strings"
)

// Omitted fields leave the existing limit intact; an explicit zero clears it.
func cephQuotaCommands(pool string, req *model.UpdateCephPoolQuotaRequest) ([][]string, error) {
	if !cephNamePattern.MatchString(pool) {
		return nil, fmt.Errorf("存储池名称无效")
	}
	commands := [][]string{}
	for _, field := range []struct {
		name  string
		value *uint64
	}{{"max_bytes", req.MaxBytes}, {"max_objects", req.MaxObjects}} {
		if field.value != nil {
			commands = append(commands, []string{"osd", "pool", "set-quota", pool, field.name, fmt.Sprint(*field.value)})
		}
	}
	if len(commands) == 0 {
		return nil, fmt.Errorf("请指定要修改的配额；0 表示解除限制")
	}
	return commands, nil
}

func (s *K8sStorageServiceImpl) requireOfflineCephImage(clusterID uint, pool, image string) error {
	if strings.HasPrefix(image, "csi-vol-") {
		return fmt.Errorf("CSI 镜像请通过 PVC 管理，禁止直接删除、缩容或回滚")
	}
	if !cephNamePattern.MatchString(pool) || !cephNamePattern.MatchString(image) {
		return fmt.Errorf("Pool 或镜像名称无效")
	}
	out, err := s.execToolCommand(clusterID, "rbd", "status", pool+"/"+image, "--format", "json")
	if err != nil {
		return fmt.Errorf("无法确认镜像挂载状态: %w", err)
	}
	var status struct {
		Watchers  []json.RawMessage `json:"watchers"`
		Migration json.RawMessage   `json:"migration"`
	}
	if err = json.Unmarshal([]byte(out), &status); err != nil {
		return fmt.Errorf("无法解析镜像状态: %w", err)
	}
	if len(status.Watchers) > 0 {
		return fmt.Errorf("镜像仍有 watchers，请先停止业务并卸载")
	}
	if len(status.Migration) > 0 && string(status.Migration) != "null" {
		return fmt.Errorf("镜像处于迁移流程，请先处理迁移状态")
	}
	return nil
}
