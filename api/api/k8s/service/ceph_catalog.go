package service

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"dodevops-api/api/k8s/model"
)

const cephNamespace = "rook-ceph"
const cephManaged = "autoops.io/ceph-maintenance"

func cephField(key, label, hint string, required bool, options ...string) model.CephField {
	return model.CephField{Key: key, Label: label, Required: required, Hint: hint, Options: options}
}

// CephOperations is also the UI's form schema. Only listed operations and fields are accepted.
func CephOperations() []model.CephOperation {
	pool := cephField("pool", "存储池", "RBD 存储池名称", true)
	img := cephField("image", "镜像", "填写镜像名，不包含 pool/", true)
	snap := cephField("snapshot", "快照", "已有快照名称", true)
	dstPool := cephField("targetPool", "目标存储池", "必须已存在并初始化 RBD", true)
	dstImg := cephField("targetImage", "目标镜像", "新镜像名；恢复、复制不覆盖已有镜像", true)
	repo := cephField("repository", "备份仓库", "已登记的持久化 PVC 仓库", true)
	backup := cephField("backupId", "备份编号", "备份列表中的编号", true)
	fs := cephField("filesystem", "文件系统", "CephFS 文件系统名称", true)
	group := cephField("group", "子卷组", "留空使用默认组；CSI 子卷通常在 csi 组", false)
	vol := cephField("subvolume", "子卷", "子卷名称", true)
	pg := cephField("pg", "PG ID", "例如 1.a", true)
	uid := cephField("uid", "RGW 用户", "已有对象存储用户 ID", true)
	return []model.CephOperation{
		{ID: "rbd-copy", Group: "RBD 迁移与克隆", Label: "从快照跨池复制", Description: "复制到独立镜像，保留源快照。完成后需单独验证并切换业务。", Fields: []model.CephField{pool, img, snap, dstPool, dstImg}},
		{ID: "rbd-deep-copy", Group: "RBD 迁移与克隆", Label: "复制镜像及全部快照", Description: "需先停止业务并卸载源镜像；检查无 watchers 后复制，保留源数据。", Dangerous: true, Fields: []model.CephField{pool, img, dstPool, dstImg}},
		{ID: "rbd-migration-prepare", Group: "RBD 迁移与克隆", Label: "迁移：准备", Description: "停止业务并卸载源卷；准备后改用目标镜像。禁止直接迁移 csi-vol-* 卷，请使用 PVC 迁移流程。", Dangerous: true, Fields: []model.CephField{pool, img, dstPool, dstImg}},
		{ID: "rbd-migration-execute", Group: "RBD 迁移与克隆", Label: "迁移：执行数据复制", Description: "pool/image 填写已准备的目标镜像。执行后仍需单独提交。", Fields: []model.CephField{pool, img}},
		{ID: "rbd-migration-commit", Group: "RBD 迁移与克隆", Label: "迁移：提交", Description: "仅允许 executed 状态；提交会删除源镜像数据。先校验业务及备份。", Dangerous: true, Fields: []model.CephField{pool, img}},
		{ID: "rbd-migration-abort", Group: "RBD 迁移与克隆", Label: "迁移：中止并回退", Description: "卸载目标镜像后中止，目标侧修改会丢失，恢复源镜像。", Dangerous: true, Fields: []model.CephField{pool, img}},
		{ID: "rbd-flatten", Group: "RBD 迁移与克隆", Label: "解除克隆依赖", Description: "将父快照的数据复制到当前克隆镜像，会增加实际占用。", Fields: []model.CephField{pool, img}},
		{ID: "rbd-snap-protect", Group: "RBD 快照", Label: "保护快照", Fields: []model.CephField{pool, img, snap}},
		{ID: "rbd-snap-unprotect", Group: "RBD 快照", Label: "取消快照保护", Description: "存在依赖克隆时 Ceph 会拒绝；请先 flatten。", Dangerous: true, Fields: []model.CephField{pool, img, snap}},
		{ID: "rbd-snapshot", Group: "RBD 快照", Label: "创建快照", Description: "自动生成 autoops- 前缀快照；数据库需先完成应用一致性处理。", Schedulable: true, Fields: []model.CephField{pool, img}},
		{ID: "backup-full", Group: "备份与恢复", Label: "全量备份", Description: "创建快照并导出完整 diff 基线、SHA-256 和清单；保留快照供增量使用。", Schedulable: true, Fields: []model.CephField{pool, img, repo}},
		{ID: "backup-incremental", Group: "备份与恢复", Label: "增量备份", Description: "从同一镜像的已有备份快照导出差异；备份链和对应快照都必须保留。", Fields: []model.CephField{pool, img, repo, cephField("baseBackup", "父备份编号", "已完成的全量或增量备份编号", true)}},
		{ID: "backup-restore", Group: "备份与恢复", Label: "恢复到新镜像", Description: "先校验整条备份链和校验和，再按顺序导入新镜像。失败时保留未完成目标供检查。", Fields: []model.CephField{repo, backup, dstPool, dstImg}},
		{ID: "backup-verify", Group: "备份与恢复", Label: "校验备份链", Description: "校验清单、父子快照关系和每个文件的 SHA-256。", Fields: []model.CephField{repo, backup}},
		{ID: "backup-delete", Group: "备份与恢复", Label: "删除备份文件", Description: "仅删除没有增量后继的备份；不自动删除源快照。", Dangerous: true, Fields: []model.CephField{repo, backup}},
		{ID: "cephfs-create", Group: "CephFS 子卷", Label: "创建子卷", Fields: []model.CephField{fs, group, vol, cephField("sizeGiB", "配额 GiB", "正整数", true)}},
		{ID: "cephfs-resize", Group: "CephFS 子卷", Label: "调整子卷配额", Description: "使用 no_shrink 防止配额小于当前已用量。CSI 子卷应从 PVC 扩容。", Fields: []model.CephField{fs, group, vol, cephField("sizeGiB", "配额 GiB", "正整数", true)}},
		{ID: "cephfs-snapshot", Group: "CephFS 子卷", Label: "创建子卷快照", Fields: []model.CephField{fs, group, vol, snap}},
		{ID: "cephfs-clone", Group: "CephFS 子卷", Label: "从快照克隆子卷", Description: "提交 CephFS 异步克隆并持续检查 clone status，完成复制后任务才成功。", Fields: []model.CephField{fs, group, vol, snap, cephField("targetSubvolume", "目标子卷", "新子卷名称", true)}},
		{ID: "cephfs-snap-delete", Group: "CephFS 子卷", Label: "删除子卷快照", Dangerous: true, Fields: []model.CephField{fs, group, vol, snap}},
		{ID: "cephfs-delete", Group: "CephFS 子卷", Label: "删除子卷", Description: "删除子卷全部数据；不使用 force，不允许操作 csi 组。", Dangerous: true, Fields: []model.CephField{fs, group, vol}},
		{ID: "osd-scrub", Group: "集群维护", Label: "OSD 巡检", Fields: []model.CephField{cephField("osd", "OSD ID", "非负整数", true), cephField("mode", "检查方式", "深度检查增加磁盘 I/O", true, "scrub", "deep-scrub")}},
		{ID: "pg-repair", Group: "集群维护", Label: "修复 PG", Description: "先定位不一致原因再提交修复；命令成功仅表示请求已受理，需继续观察 PG 状态。", Dangerous: true, Fields: []model.CephField{pg}},
		{ID: "osd-flag", Group: "集群维护", Label: "设置或解除维护标志", Description: "标志不会自动解除，维护结束必须恢复。", Dangerous: true, Fields: []model.CephField{cephField("flag", "标志", "noout 常用于短期主机维护", true, "noout", "norebalance", "nobackfill", "norecover", "noscrub", "nodeep-scrub"), cephField("action", "动作", "set 设置，unset 解除", true, "set", "unset")}},
		{ID: "balancer", Group: "集群维护", Label: "自动均衡开关", Fields: []model.CephField{cephField("action", "动作", "按现有 balancer 模式启停", true, "on", "off")}},
		{ID: "pool-autoscale", Group: "集群维护", Label: "PG 自动伸缩", Fields: []model.CephField{pool, cephField("mode", "模式", "Rook 管理的池应同时在 CR spec.parameters 持久化", true, "on", "warn", "off")}},
		{ID: "rgw-user-suspend", Group: "RGW 对象存储", Label: "暂停 RGW 用户", Description: "暂停用户及其子用户的对象访问。", Dangerous: true, Fields: []model.CephField{uid}},
		{ID: "rgw-user-enable", Group: "RGW 对象存储", Label: "启用 RGW 用户", Fields: []model.CephField{uid}},
		{ID: "rgw-quota", Group: "RGW 对象存储", Label: "设置用户或桶配额", Description: "设置并启用配额。桶配额针对该用户的每个桶。", Fields: []model.CephField{uid, cephField("scope", "范围", "", true, "user", "bucket"), cephField("sizeGiB", "容量 GiB", "正整数", true), cephField("maxObjects", "对象数", "正整数", true)}},
	}
}

var cephNamePattern = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9_.-]{0,127}$`)
var cephPGPattern = regexp.MustCompile(`^[0-9]+\.[0-9a-f]+$`)

func validateCephTask(req *model.CephTaskRequest) (*model.CephOperation, error) {
	var op *model.CephOperation
	for _, item := range CephOperations() {
		if item.ID == req.Operation {
			copy := item
			op = &copy
			break
		}
	}
	if op == nil {
		return nil, fmt.Errorf("不支持的维护操作")
	}
	if req.TimeoutSeconds == 0 {
		req.TimeoutSeconds = 21600
	}
	if req.TimeoutSeconds < 60 || req.TimeoutSeconds > 604800 {
		return nil, fmt.Errorf("任务超时必须为 60–604800 秒")
	}
	allowed := map[string]bool{}
	for _, field := range op.Fields {
		allowed[field.Key] = true
		value := req.Params[field.Key]
		if value == "" {
			if field.Required {
				return nil, fmt.Errorf("%s不能为空", field.Label)
			}
			continue
		}
		if len(field.Options) > 0 {
			ok := false
			for _, option := range field.Options {
				if option == value {
					ok = true
				}
			}
			if !ok {
				return nil, fmt.Errorf("%s选项无效", field.Label)
			}
		} else if field.Key == "sizeGiB" || field.Key == "maxObjects" || field.Key == "osd" {
			n, err := strconv.ParseInt(value, 10, 64)
			if err != nil || n < 0 || (field.Key != "osd" && n == 0) || n > 104857600 {
				return nil, fmt.Errorf("%s数值无效", field.Label)
			}
		} else if field.Key == "pg" {
			if !cephPGPattern.MatchString(value) {
				return nil, fmt.Errorf("PG ID 无效")
			}
		} else if !cephNamePattern.MatchString(value) {
			return nil, fmt.Errorf("%s仅允许字母、数字、下划线、点和连字符，且不能以点或连字符开头", field.Label)
		}
	}
	for key := range req.Params {
		if !allowed[key] {
			return nil, fmt.Errorf("不支持的参数: %s", key)
		}
	}
	if op.Dangerous && req.Confirmation != req.Operation {
		return nil, fmt.Errorf("请输入操作标识 %s 确认风险", req.Operation)
	}
	if strings.HasPrefix(req.Operation, "rbd-migration-") && (strings.HasPrefix(req.Params["image"], "csi-vol-") || strings.HasPrefix(req.Params["targetImage"], "csi-vol-")) {
		return nil, fmt.Errorf("CSI 镜像请通过 PVC 迁移，禁止直接 RBD 迁移")
	}
	if (req.Operation == "cephfs-delete" || req.Operation == "cephfs-resize") && (req.Params["group"] == "csi" || strings.HasPrefix(req.Params["subvolume"], "csi-vol-")) {
		return nil, fmt.Errorf("CSI 子卷请通过 PVC 管理")
	}
	if req.Params["targetPool"] != "" && req.Params["pool"] == req.Params["targetPool"] && req.Params["image"] == req.Params["targetImage"] {
		return nil, fmt.Errorf("源镜像与目标镜像不能相同")
	}
	return op, nil
}
