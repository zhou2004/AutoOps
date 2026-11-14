package service

import (
	"context"
	"dodevops-api/api/webservice/model"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// InstanceConfig is the editable view of the configuration files that the
// runner reads from one running instance on this request. It never contains credentials
// and never includes the certificate private key, which is published by the
// certificate binding instead.
type InstanceConfig struct {
	InstanceID   uint         `json:"instanceId"`
	Name         string       `json:"name"`
	ClusterID    uint         `json:"clusterId"`
	Management   string       `json:"management"`
	ConfigPath   string       `json:"configPath"`
	Prefix       string       `json:"prefix"`
	ManagedPaths []string     `json:"managedPaths"`
	Version      string       `json:"version"`
	Digest       string       `json:"digest"`
	Identity     string       `json:"identity"`
	CollectedAt  *time.Time   `json:"collectedAt"`
	Files        []model.File `json:"files"`
}

// ConfigValidation is a synchronous `nginx -t` result for a candidate file set.
type ConfigValidation struct {
	Valid      bool   `json:"valid"`
	Message    string `json:"message"`
	InstanceID uint   `json:"instanceId"`
	Name       string `json:"name"`
	Digest     string `json:"digest"`
	Checked    int    `json:"checked"`
}

func (s *Service) instanceConfigTarget(u, id uint) (model.Resource, model.Resource, model.SourceSpec, Observation, error) {
	i, e := s.Get("instances", id)
	if e != nil {
		return i, model.Resource{}, model.SourceSpec{}, Observation{}, e
	}
	if e = s.Authorize(u, "instances", i, "web_config_view"); e != nil {
		return i, model.Resource{}, model.SourceSpec{}, Observation{}, e
	}
	src, e := s.Get("sources", i.SourceID)
	if e != nil {
		return i, src, model.SourceSpec{}, Observation{}, e
	}
	var spec model.SourceSpec
	if e = json.Unmarshal(src.Spec, &spec); e != nil {
		return i, src, spec, Observation{}, e
	}
	var ob Observation
	if e = json.Unmarshal(i.Observed, &ob); e != nil {
		return i, src, spec, ob, e
	}
	return i, src, spec, ob, nil
}

func (s *Service) InstanceConfig(u, id uint) (InstanceConfig, error) {
	return s.InstanceConfigContext(context.Background(), u, id)
}

func (s *Service) InstanceConfigContext(ctx context.Context, u, id uint) (InstanceConfig, error) {
	i, _, spec, ob, e := s.instanceConfigTarget(u, id)
	if e != nil {
		return InstanceConfig{}, e
	}
	previous := ob
	ob, e = (Nginx{}).Discover(ctx, s.Remote, spec)
	if e != nil {
		return InstanceConfig{}, e
	}
	if ob.Identity == "" || (previous.Identity != "" && previous.Identity != ob.Identity) {
		return InstanceConfig{}, errors.New("实例身份已变化，请重新发现并核验数据源")
	}
	now := time.Now().UTC()
	files := ob.Files
	if files == nil {
		files = []model.File{}
	}
	out := InstanceConfig{InstanceID: i.ID, Name: i.Name, ClusterID: i.ClusterID, Management: i.Management,
		ConfigPath: spec.ConfigPath, Prefix: spec.Prefix, ManagedPaths: spec.ManagedPaths,
		Version: strings.SplitN(ob.Version, "\n", 2)[0], Digest: ob.Digest, Identity: ob.Identity,
		CollectedAt: &now, Files: files}
	if out.ManagedPaths == nil {
		out.ManagedPaths = []string{}
	}
	return out, nil
}

// withinManaged keeps candidate paths inside the adopted scope. The remote
// runner repeats this check against the resolved path.
func withinManaged(p string, managed []string) bool {
	for _, root := range managed {
		root = strings.TrimSuffix(root, "/")
		if root == "" {
			continue
		}
		if p == root || strings.HasPrefix(p, root+"/") {
			return true
		}
	}
	return false
}

var configMode = regexp.MustCompile(`^0[0-7]{3}$`)

// normalizeConfigChanges validates the file set a user wants to test or publish.
// Only changed files are required; the remote runner merges them with live state.
func normalizeConfigChanges(observed []model.File, changes []model.File, managed []string) ([]model.File, error) {
	if len(changes) == 0 {
		return nil, errors.New("没有需要校验的配置变更")
	}
	if len(changes) > 200 {
		return nil, errors.New("单次最多校验 200 个配置文件")
	}
	existing := make(map[string]model.File, len(observed))
	for _, f := range observed {
		existing[f.Path] = f
	}
	seen := map[string]bool{}
	total := 0
	out := make([]model.File, 0, len(changes))
	for _, f := range changes {
		if !validPath(f.Path) {
			return nil, fmt.Errorf("配置路径无效: %s", f.Path)
		}
		if seen[f.Path] {
			return nil, fmt.Errorf("配置路径重复: %s", f.Path)
		}
		if !withinManaged(f.Path, managed) {
			return nil, fmt.Errorf("配置路径不在受管目录内: %s", f.Path)
		}
		seen[f.Path] = true
		if f.Deleted {
			if _, ok := existing[f.Path]; !ok {
				return nil, fmt.Errorf("要删除的配置不存在: %s", f.Path)
			}
			out = append(out, model.File{Path: f.Path, Deleted: true, Group: f.Group})
			continue
		}
		if strings.ContainsRune(f.Content, 0) {
			return nil, fmt.Errorf("配置内容包含非法字符: %s", f.Path)
		}
		if strings.Contains(f.Content, "PRIVATE KEY-----") {
			return nil, errors.New("私钥请通过证书管理导入")
		}
		mode := f.Mode
		if mode == 0 {
			if old, ok := existing[f.Path]; ok && old.Mode != 0 {
				mode = old.Mode
			} else {
				mode = 0644
			}
		}
		if !configMode.MatchString(fmt.Sprintf("0%o", mode)) {
			return nil, fmt.Errorf("文件权限无效: %s", f.Path)
		}
		total += len(f.Content)
		if total > 4<<20 {
			return nil, errors.New("配置内容不能超过 4 MiB")
		}
		if len([]rune(f.Group)) > 64 {
			return nil, errors.New("配置分组名称最多 64 字符")
		}
		out = append(out, model.File{Path: f.Path, Content: f.Content, Mode: mode, Group: f.Group})
	}
	return out, nil
}

// stagingPath matches the remote candidate tree so Nginx messages can be shown
// against the real file path instead of a random temporary directory.
var stagingPath = regexp.MustCompile(`/tmp/autoops-web-\d+-\d+`)

// selectConfigurationMembers rejects duplicate, stale and cross-cluster targets.
// Empty selection preserves the legacy cluster-wide draft contract.
func selectConfigurationMembers(ids []uint, members []model.Resource) ([]model.Resource, error) {
	if len(ids) == 0 {
		return members, nil
	}
	wanted := make(map[uint]bool, len(ids))
	for _, id := range ids {
		if id == 0 || wanted[id] {
			return nil, errors.New("目标实例为空或重复")
		}
		wanted[id] = true
	}
	selected := make([]model.Resource, 0, len(ids))
	for _, member := range members {
		if wanted[member.ID] {
			selected = append(selected, member)
		}
	}
	if len(selected) != len(ids) {
		return nil, errors.New("目标实例不属于当前集群")
	}
	return selected, nil
}

func (s *Service) ValidateInstanceConfig(ctx context.Context, u, id uint, changes []model.File) (ConfigValidation, error) {
	return s.validateInstanceConfiguration(ctx, u, id, model.ConfigSpec{Files: changes})
}

func (s *Service) validateInstanceConfiguration(ctx context.Context, u, id uint, candidate model.ConfigSpec) (ConfigValidation, error) {
	i, _, spec, previous, e := s.instanceConfigTarget(u, id)
	out := ConfigValidation{InstanceID: i.ID, Name: i.Name}
	if e != nil {
		return out, e
	}
	if e = s.Module(ctx, u, "write"); e != nil {
		return out, e
	}
	if e = s.Authorize(u, "instances", i, "web_config_edit"); e != nil {
		return out, e
	}
	// Reject malformed input before opening a remote connection.
	for _, f := range candidate.Files {
		if !validPath(f.Path) || !withinManaged(f.Path, spec.ManagedPaths) {
			out.Message = "配置路径不在受管目录内: " + f.Path
			return out, nil
		}
	}
	ob, e := (Nginx{}).Discover(ctx, s.Remote, spec)
	if e != nil {
		out.Message = e.Error()
		return out, nil
	}
	out.Digest = ob.Digest
	if ob.Identity == "" || (previous.Identity != "" && previous.Identity != ob.Identity) {
		out.Message = "实例身份已变化，请重新发现并核验数据源"
		return out, nil
	}
	if baseline := candidate.Baselines[id]; baseline != "" && baseline != ob.Digest {
		return out, ErrConflict
	}
	changes := append([]model.File(nil), candidate.Files...)
	for index := range changes {
		for k, v := range candidate.Variables[id] {
			changes[index].Content = stringsReplace(changes[index].Content, "${"+k+"}", v)
		}
	}
	normalized, e := normalizeConfigChanges(ob.Files, changes, spec.ManagedPaths)
	if e != nil {
		out.Message = e.Error()
		return out, nil
	}
	out.Checked = len(normalized)
	// Match publication preparation: keys only exist in the remote request and
	// are never copied into the persisted draft or returned validation result.
	normalized, _, e = s.prepareCertificates(u, i.ClusterID, normalized, 0, nil)
	if e != nil {
		return out, e
	}
	raw, e := s.Remote.Run(ctx, RemoteRequest{Action: "validate", Spec: spec, Files: normalized, Baseline: ob.Digest, Identity: ob.Identity})
	if e != nil {
		out.Message = stagingPath.ReplaceAllString(e.Error(), "")
		return out, nil
	}
	var result struct {
		Validated bool `json:"validated"`
	}
	if e = json.Unmarshal(raw, &result); e != nil {
		out.Message = "远端校验结果格式无效"
		return out, nil
	}
	out.Valid = result.Validated
	if !out.Valid {
		out.Message = "Nginx 语法校验未通过"
	}
	return out, nil
}

// ValidateClusterConfig validates only the selected members; the optional spec
// keeps existing callers compatible while supplying variables and baselines.
func (s *Service) ValidateClusterConfig(ctx context.Context, u, cid uint, changes []model.File, options ...model.ConfigSpec) ([]ConfigValidation, error) {
	if e := s.Asset(u, "web_cluster", cid, "web_config_view"); e != nil {
		return nil, e
	}
	members, e := s.Members(u, cid)
	if e != nil {
		return nil, e
	}
	var candidate model.ConfigSpec
	if len(options) > 0 {
		candidate = options[0]
	}
	candidate.Files = changes
	members, e = selectConfigurationMembers(candidate.InstanceIDs, members)
	if e != nil {
		return nil, e
	}
	if len(members) == 0 {
		return nil, errors.New("集群没有成员节点")
	}
	// Authorize the complete selection before contacting the first host.
	for _, m := range members {
		if e = s.Authorize(u, "instances", m, "web_config_edit"); e != nil {
			return nil, e
		}
	}
	out := make([]ConfigValidation, 0, len(members))
	for _, m := range members {
		if ctx.Err() != nil {
			return out, ctx.Err()
		}
		v, err := s.validateInstanceConfiguration(ctx, u, m.ID, candidate)
		if err != nil {
			return out, err
		}
		out = append(out, v)
	}
	return out, nil
}

func (s *Service) validateConfigGroupPaths(u uint, group model.Resource, spec model.ConfigGroupSpec) error {
	if len(spec.Paths) > 2000 {
		return errors.New("配置分组最多包含 2000 个文件")
	}
	if len(spec.Paths) == 0 {
		return nil
	}
	members, err := s.Members(u, group.ClusterID)
	if err != nil {
		return err
	}
	var scope []string
	for _, member := range members {
		source, e := s.Get("sources", member.SourceID)
		if e != nil {
			return e
		}
		connection, e := decode[model.SourceSpec](source.Spec)
		if e != nil {
			return e
		}
		scope = append(scope, connection.ManagedPaths...)
	}
	seen := map[string]bool{}
	for _, path := range spec.Paths {
		if !validPath(path) || !withinManaged(path, scope) {
			return fmt.Errorf("配置分组路径不在受管目录内: %s", path)
		}
		if seen[path] {
			return fmt.Errorf("配置分组路径重复: %s", path)
		}
		seen[path] = true
	}
	var groups []model.Resource
	if err = s.DB.Table(model.Tables["config-groups"]).Where("cluster_id = ? AND id <> ?", group.ClusterID, group.ID).Find(&groups).Error; err != nil {
		return err
	}
	for _, other := range groups {
		otherSpec, e := decode[model.ConfigGroupSpec](other.Spec)
		if e != nil {
			return e
		}
		for _, path := range otherSpec.Paths {
			if seen[path] {
				return fmt.Errorf("配置文件已属于其他分组: %s", path)
			}
		}
	}
	return nil
}
