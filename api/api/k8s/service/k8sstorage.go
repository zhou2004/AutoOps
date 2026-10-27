package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"dodevops-api/api/k8s/dao"
	"dodevops-api/api/k8s/model"
	"dodevops-api/common/result"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"sigs.k8s.io/yaml"
)

// IK8sStorageService K8s存储管理服务接口
type IK8sStorageService interface {
	// PVC管理
	GetPVCs(c *gin.Context, clusterId uint, namespaceName string)
	GetPVCDetail(c *gin.Context, clusterId uint, namespaceName string, pvcName string)
	CreatePVC(c *gin.Context, clusterId uint, namespaceName string, req *model.CreatePVCRequest)
	UpdatePVC(c *gin.Context, clusterId uint, namespaceName string, pvcName string, req *model.UpdatePVCRequest)
	DeletePVC(c *gin.Context, clusterId uint, namespaceName string, pvcName string)
	GetPVCYaml(c *gin.Context, clusterId uint, namespaceName string, pvcName string)
	UpdatePVCYaml(c *gin.Context, clusterId uint, namespaceName string, pvcName string, yamlData map[string]interface{})

	// PVC高阶功能
	ExpandPVC(c *gin.Context, clusterId uint, namespaceName string, pvcName string, req *model.ExpandPVCRequest) // PVC扩容
	ClonePVC(c *gin.Context, clusterId uint, namespaceName string, pvcName string, req *model.PVCCloneRequest)   // PVC克隆（数据迁移）

	// 存储迁移
	MigratePVC(c *gin.Context, clusterId uint, req *model.MigratePVCRequest)                                     // 卷到卷数据迁移
	SwitchWorkloadPVC(c *gin.Context, clusterId uint, namespaceName string, req *model.SwitchWorkloadPVCRequest) // 一键换卷

	// PV管理
	GetPVs(c *gin.Context, clusterId uint)
	GetPVDetail(c *gin.Context, clusterId uint, pvName string)
	CreatePV(c *gin.Context, clusterId uint, req *model.CreatePVRequest)
	UpdatePV(c *gin.Context, clusterId uint, pvName string, req *model.UpdatePVRequest)
	DeletePV(c *gin.Context, clusterId uint, pvName string)
	GetPVYaml(c *gin.Context, clusterId uint, pvName string)
	UpdatePVYaml(c *gin.Context, clusterId uint, pvName string, yamlData map[string]interface{})

	// StorageClass管理
	GetStorageClasses(c *gin.Context, clusterId uint)
	GetStorageClassDetail(c *gin.Context, clusterId uint, storageClassName string)
	CreateStorageClass(c *gin.Context, clusterId uint, req *model.CreateStorageClassRequest)
	UpdateStorageClass(c *gin.Context, clusterId uint, storageClassName string, req *model.UpdateStorageClassRequest)
	DeleteStorageClass(c *gin.Context, clusterId uint, storageClassName string)
	GetStorageClassYaml(c *gin.Context, clusterId uint, storageClassName string)
	UpdateStorageClassYaml(c *gin.Context, clusterId uint, storageClassName string, yamlData map[string]interface{})

	// StorageClass高阶功能
	SetDefaultStorageClass(c *gin.Context, clusterId uint, storageClassName string, req *model.SetDefaultSCRequest) // 设置默认存储类
	GetStorageStats(c *gin.Context, clusterId uint)                                                                 // 获取存储统计

	// Ceph 管理（对接 Rook-Ceph）
	GetCephStatus(c *gin.Context, clusterId uint)                                                             // Ceph 集群状态概览
	GetCephPools(c *gin.Context, clusterId uint)                                                              // Ceph pool 列表
	UpdateCephPoolSize(c *gin.Context, clusterId uint, poolName string, req *model.UpdateCephPoolSizeRequest) // 调整 pool 副本数
	GetCephOSDs(c *gin.Context, clusterId uint)                                                               // Ceph OSD 列表
	// Ceph Pool 详细操作
	CreateCephPool(c *gin.Context, clusterId uint, req *model.CreateCephPoolRequest)                                        // 创建 pool
	DeleteCephPool(c *gin.Context, clusterId uint, poolName string)                                                         // 删除 pool
	UpdateCephPoolPGNum(c *gin.Context, clusterId uint, poolName string, req *model.UpdateCephPoolPGNumRequest)             // 调整 pool PG 数
	UpdateCephPoolQuota(c *gin.Context, clusterId uint, poolName string, req *model.UpdateCephPoolQuotaRequest)             // 设置 pool 配额
	UpdateCephPoolCompression(c *gin.Context, clusterId uint, poolName string, req *model.UpdateCephPoolCompressionRequest) // 设置 pool 压缩
	// Ceph RBD 块存储
	GetCephRBDImages(c *gin.Context, clusterId uint)                                                                   // RBD 镜像列表
	CreateCephRBDImage(c *gin.Context, clusterId uint, req *model.CreateCephRBDImageRequest)                           // 创建 RBD 镜像
	GetCephRBDImageDetail(c *gin.Context, clusterId uint, pool, image string)                                          // RBD 镜像详情
	ResizeCephRBDImage(c *gin.Context, clusterId uint, pool, image string, req *model.ResizeCephRBDImageRequest)       // 扩容 RBD 镜像
	DeleteCephRBDImage(c *gin.Context, clusterId uint, pool, image string)                                             // 删除 RBD 镜像
	ListCephRBDSnapshots(c *gin.Context, clusterId uint, pool, image string)                                           // RBD 快照列表
	CreateCephRBDSnapshot(c *gin.Context, clusterId uint, pool, image string, req *model.CreateCephRBDSnapshotRequest) // 创建 RBD 快照
	DeleteCephRBDSnapshot(c *gin.Context, clusterId uint, pool, image, snap string)                                    // 删除 RBD 快照
	RollbackCephRBDImage(c *gin.Context, clusterId uint, pool, image, snap string)                                     // RBD 快照回滚
	CloneCephRBDImage(c *gin.Context, clusterId uint, pool, image string, req *model.CloneCephRBDImageRequest)         // 克隆 RBD 镜像
	// Ceph CephFS 与监控
	GetCephFS(c *gin.Context, clusterId uint)       // CephFS 文件系统列表
	GetCephHealth(c *gin.Context, clusterId uint)   // Ceph 健康详情
	GetCephVersions(c *gin.Context, clusterId uint) // Ceph 版本分布
	GetCephPGStat(c *gin.Context, clusterId uint)   // Ceph PG 状态
	// Ceph OSD 管理
	SetCephOSDStatus(c *gin.Context, clusterId uint, osdId int, action string)                    // 标记 OSD in/out/down
	ReweightCephOSD(c *gin.Context, clusterId uint, osdId int, req *model.ReweightCephOSDRequest) // 调整 OSD 权重
	DeleteCephOSD(c *gin.Context, clusterId uint, osdId int)                                      // 删除 OSD（危险操作）
}

// K8sStorageServiceImpl K8s存储管理服务实现
type K8sStorageServiceImpl struct {
	clusterDao *dao.KubeClusterDao
}

// NewK8sStorageService 创建K8s存储管理服务实例
func NewK8sStorageService(db *gorm.DB) IK8sStorageService {
	return &K8sStorageServiceImpl{
		clusterDao: dao.NewKubeClusterDao(db),
	}
}

// getK8sClient 获取K8s客户端
func (s *K8sStorageServiceImpl) getK8sClient(clusterId uint) (*kubernetes.Clientset, error) {
	cluster, err := s.clusterDao.GetByID(clusterId)
	if err != nil {
		return nil, fmt.Errorf("获取集群信息失败: %v", err)
	}

	if cluster.Credential == "" {
		return nil, fmt.Errorf("集群凭证为空")
	}

	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.Credential))
	if err != nil {
		return nil, fmt.Errorf("解析集群凭证失败: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("创建K8s客户端失败: %v", err)
	}

	return clientset, nil
}

// getK8sClientAndConfig 获取K8s客户端与RESTConfig（exec 需要 config）
func (s *K8sStorageServiceImpl) getK8sClientAndConfig(clusterId uint) (*kubernetes.Clientset, *rest.Config, error) {
	cluster, err := s.clusterDao.GetByID(clusterId)
	if err != nil {
		return nil, nil, fmt.Errorf("获取集群信息失败: %v", err)
	}
	if cluster.Credential == "" {
		return nil, nil, fmt.Errorf("集群凭证为空")
	}
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.Credential))
	if err != nil {
		return nil, nil, fmt.Errorf("解析集群凭证失败: %v", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("创建K8s客户端失败: %v", err)
	}
	return clientset, config, nil
}

// execInPod 在指定Pod容器中执行命令，返回 stdout/stderr
func execInPod(clientset *kubernetes.Clientset, config *rest.Config, namespace, podName, containerName string, command []string) (string, error) {
	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		Param("container", containerName).
		Param("stdin", "false").
		Param("stdout", "true").
		Param("stderr", "true").
		Param("tty", "false").
		VersionedParams(&corev1.PodExecOptions{
			Container: containerName,
			Command:   command,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return "", fmt.Errorf("创建executor失败: %v", err)
	}

	var stdout, stderr bytes.Buffer
	err = executor.StreamWithContext(context.Background(), remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		// 即使返回错误也返回部分输出
		return stdout.String() + stderr.String(), fmt.Errorf("执行命令失败: %v", err)
	}
	return stdout.String(), nil
}

// findRookCephToolsPod 查找 rook-ceph 命名空间下的 rook-ceph-tools pod
func findRookCephToolsPod(clientset *kubernetes.Clientset) (string, string, error) {
	pods, err := clientset.CoreV1().Pods("rook-ceph").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "app=rook-ceph-tools",
	})
	if err != nil {
		return "", "", fmt.Errorf("查找 rook-ceph-tools pod 失败: %v", err)
	}
	for _, p := range pods.Items {
		if p.Status.Phase == corev1.PodRunning && len(p.Spec.Containers) > 0 {
			return p.Name, p.Spec.Containers[0].Name, nil
		}
	}
	if len(pods.Items) > 0 {
		return pods.Items[0].Name, pods.Items[0].Spec.Containers[0].Name, nil
	}
	return "", "", fmt.Errorf("未找到 rook-ceph-tools pod，请先部署 toolbox: kubectl -n rook-ceph apply -f toolbox.yaml")
}

// execToolCommand 在 rook-ceph-tools pod 中执行任意工具命令（ceph/rbd/rados/cephfs 等）
func (s *K8sStorageServiceImpl) execToolCommand(clusterId uint, cmd string, args ...string) (string, error) {
	clientset, config, err := s.getK8sClientAndConfig(clusterId)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pod, err := readyCephPod(ctx, clientset, "app=rook-ceph-tools")
	if err != nil {
		return "", err
	}
	return cephExec(ctx, clientset, config, pod, append([]string{cmd}, args...))
}

// execCephCommand 在 rook-ceph-tools pod 中执行 ceph 命令
func (s *K8sStorageServiceImpl) execCephCommand(clusterId uint, args ...string) (string, error) {
	return s.execToolCommand(clusterId, "ceph", args...)
}

// ===================== Ceph 管理实现 =====================

// GetCephStatus 获取 Ceph 集群状态概览
func (s *K8sStorageServiceImpl) GetCephStatus(c *gin.Context, clusterId uint) {
	clientset, err := s.getK8sClient(clusterId)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "连接K8s集群失败: "+err.Error())
		return
	}

	// 获取 CephCluster CR 状态（phase）
	cephCluster := &model.CephClusterStatus{}
	clusterCR, crErr := clientset.RESTClient().Get().
		AbsPath("/apis/ceph.rook.io/v1/namespaces/rook-ceph/cephclusters/rook-ceph").
		Do(context.TODO()).Raw()
	if crErr == nil {
		var crMap map[string]interface{}
		if json.Unmarshal(clusterCR, &crMap) == nil {
			if st, ok := crMap["status"].(map[string]interface{}); ok {
				if phase, ok := st["phase"]; ok {
					cephCluster.Phase = fmt.Sprintf("%v", phase)
				}
			}
		}
	}

	// ceph -s 获取详细状态
	out, err := s.execCephCommand(clusterId, "status", "-f", "json")
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "获取Ceph状态失败: "+err.Error()+"（请确认已部署 rook-ceph-tools toolbox）")
		return
	}
	var status struct {
		FSID   string `json:"fsid"`
		Health struct {
			Status string `json:"status"`
		} `json:"health"`
		QuorumNames []string `json:"quorum_names"`
		MonMap      struct {
			NumMons int `json:"num_mons"`
		} `json:"monmap"`
		MgrMap struct {
			Available bool `json:"available"`
		} `json:"mgrmap"`
		OSDMap struct {
			NumOsds   int `json:"num_osds"`
			NumUpOsds int `json:"num_up_osds"`
			NumInOsds int `json:"num_in_osds"`
		} `json:"osdmap"`
		PGMap struct {
			NumPgs     int `json:"num_pgs"`
			PgsByState []struct {
				StateName string `json:"state_name"`
				Count     int    `json:"count"`
			} `json:"pgs_by_state"`
		} `json:"pgmap"`
	}
	if err := json.Unmarshal([]byte(out), &status); err != nil {
		result.Failed(c, http.StatusInternalServerError, "解析Ceph状态失败: "+err.Error())
		return
	}

	cephCluster.Fsid = status.FSID
	if status.Health.Status != "" {
		cephCluster.Health = status.Health.Status
	}
	cephCluster.MonCount = status.MonMap.NumMons
	cephCluster.MonQuorum = strings.Join(status.QuorumNames, ",")
	if status.MgrMap.Available {
		cephCluster.MgrActive = "active"
	} else {
		cephCluster.MgrActive = "inactive"
	}
	cephCluster.OsdCount = status.OSDMap.NumOsds
	cephCluster.OsdUp = status.OSDMap.NumUpOsds
	cephCluster.OsdIn = status.OSDMap.NumInOsds
	cephCluster.PgCount = status.PGMap.NumPgs
	// 计算 active+clean 的 PG 数
	activeClean := 0
	for _, s := range status.PGMap.PgsByState {
		if strings.Contains(s.StateName, "active") && strings.Contains(s.StateName, "clean") {
			activeClean += s.Count
		}
	}
	cephCluster.PgActiveClean = activeClean

	// 获取版本
	verOut, verErr := s.execCephCommand(clusterId, "--version")
	if verErr == nil {
		cephCluster.Version = strings.TrimSpace(verOut)
	}

	// ceph df 获取容量
	dfOut, err := s.execCephCommand(clusterId, "df", "-f", "json")
	if err == nil {
		var df struct {
			Stats struct {
				TotalBytes      uint64 `json:"total_bytes"`
				TotalUsedBytes  uint64 `json:"total_used_bytes"`
				TotalAvailBytes uint64 `json:"total_avail_bytes"`
			} `json:"stats"`
		}
		if json.Unmarshal([]byte(dfOut), &df) == nil {
			cephCluster.TotalBytes = df.Stats.TotalBytes
			cephCluster.UsedBytes = df.Stats.TotalUsedBytes
			cephCluster.AvailableBytes = df.Stats.TotalAvailBytes
			if df.Stats.TotalBytes > 0 {
				cephCluster.RawUsedPercent = float64(df.Stats.TotalUsedBytes) * 100 / float64(df.Stats.TotalBytes)
			}
		}
	}

	// pool 数量
	poolOut, err := s.execCephCommand(clusterId, "osd", "pool", "ls", "-f", "json")
	if err == nil {
		var pools []string
		if json.Unmarshal([]byte(poolOut), &pools) == nil {
			cephCluster.PoolCount = len(pools)
		}
	}

	result.Success(c, cephCluster)
}

// GetCephPools 获取 Ceph pool 列表（含副本数）
func (s *K8sStorageServiceImpl) GetCephPools(c *gin.Context, clusterId uint) {
	out, err := s.execCephCommand(clusterId, "osd", "pool", "ls", "detail", "-f", "json")
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "获取Ceph Pool失败: "+err.Error())
		return
	}
	var pools []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &pools); err != nil {
		result.Failed(c, http.StatusInternalServerError, "解析Ceph Pool失败: "+err.Error())
		return
	}

	var poolList []model.CephPool
	for _, p := range pools {
		pool := model.CephPool{}
		if v, ok := p["pool_name"].(string); ok {
			pool.Name = v
		}
		if v, ok := p["pool_id"].(float64); ok {
			pool.PoolID = int(v)
		}
		if v, ok := p["size"].(float64); ok {
			pool.Size = int(v)
		}
		if v, ok := p["min_size"].(float64); ok {
			pool.MinSize = int(v)
		}
		if v, ok := p["pg_num"].(float64); ok {
			pool.PgNum = int(v)
		}
		if v, ok := p["pg_placement_num"].(float64); ok {
			pool.PgpNum = int(v)
		}
		if v, ok := p["crush_rule"].(float64); ok {
			pool.CrushRuleID = int(v)
		}
		if app, ok := p["application_metadata"].(map[string]interface{}); ok {
			apps := make([]string, 0, len(app))
			for k := range app {
				apps = append(apps, k)
			}
			pool.Application = strings.Join(apps, ",")
		}
		poolList = append(poolList, pool)
	}

	// 获取每 pool 用量（ceph df detail）
	dfOut, err := s.execCephCommand(clusterId, "df", "detail", "-f", "json")
	if err == nil {
		var df struct {
			Pools []struct {
				Name  string `json:"name"`
				Stats struct {
					Stored    uint64 `json:"stored"`
					Objects   uint64 `json:"objects"`
					KbUsed    uint64 `json:"kb_used"`
					BytesUsed uint64 `json:"bytes_used"`
					MaxAvail  uint64 `json:"max_avail"`
				} `json:"stats"`
			} `json:"pools"`
		}
		if json.Unmarshal([]byte(dfOut), &df) == nil {
			for i := range poolList {
				for _, dp := range df.Pools {
					if dp.Name == poolList[i].Name {
						poolList[i].UsedBytes = dp.Stats.BytesUsed
						poolList[i].AvailableBytes = dp.Stats.MaxAvail
						// 计算使用率
						total := poolList[i].UsedBytes + poolList[i].AvailableBytes
						if total > 0 {
							poolList[i].RawUsedPercent = math.Round(float64(poolList[i].UsedBytes)/float64(total)*1000) / 10
						}
						break
					}
				}
			}
		}
	}

	result.Success(c, poolList)
}

// UpdateCephPoolSize 调整 Ceph pool 副本数
func (s *K8sStorageServiceImpl) UpdateCephPoolSize(c *gin.Context, clusterId uint, poolName string, req *model.UpdateCephPoolSizeRequest) {
	if req.Size < 1 || req.Size > 10 {
		result.Failed(c, http.StatusBadRequest, "副本数必须在 1-10 之间")
		return
	}
	out, err := s.execCephCommand(clusterId, "osd", "pool", "set", poolName, "size", fmt.Sprintf("%d", req.Size))
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "调整副本数失败: "+err.Error()+" "+out)
		return
	}
	result.Success(c, map[string]interface{}{
		"message":  fmt.Sprintf("Pool %s 副本数已调整为 %d", poolName, req.Size),
		"poolName": poolName,
		"size":     req.Size,
	})
}

// GetCephOSDs 获取 Ceph OSD 状态列表
func (s *K8sStorageServiceImpl) GetCephOSDs(c *gin.Context, clusterId uint) {
	// 1. 解析 `ceph osd tree` 文本，获取 OSD -> host 归属（JSON 无 parent，需按缩进层级）
	treeOut, err := s.execCephCommand(clusterId, "osd", "tree")
	hostOfOsd := map[int]string{}
	currentHost := ""
	for _, line := range strings.Split(treeOut, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		id := 0
		if _, err := fmt.Sscanf(fields[0], "%d", &id); err != nil {
			continue
		}
		// 按内容判断（host 行无 CLASS 列，osd 行有 CLASS 列导致列错位，故用 token 匹配）
		for i, f := range fields {
			if f == "host" && i+1 < len(fields) {
				currentHost = fields[i+1]
				break
			}
			if f == "root" {
				currentHost = ""
				break
			}
			if strings.HasPrefix(f, "osd.") {
				hostOfOsd[id] = currentHost
				break
			}
		}
	}

	// 2. 解析 `ceph osd dump -f json`，获取每个 OSD 的 up/in 状态
	dumpOut, err := s.execCephCommand(clusterId, "osd", "dump", "-f", "json")
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "获取Ceph OSD状态失败: "+err.Error())
		return
	}
	var dump struct {
		OSDs []struct {
			ID int `json:"osd"`
			Up int `json:"up"`
			In int `json:"in"`
		} `json:"osds"`
	}
	_ = json.Unmarshal([]byte(dumpOut), &dump)
	osdStatus := map[int]struct{ Up, In bool }{}
	for _, o := range dump.OSDs {
		osdStatus[o.ID] = struct{ Up, In bool }{Up: o.Up == 1, In: o.In == 1}
	}

	// 3. 解析 `ceph osd df -f json`，获取容量
	dfOut, err := s.execCephCommand(clusterId, "osd", "df", "-f", "json")
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "获取Ceph OSD容量失败: "+err.Error())
		return
	}
	var df struct {
		Nodes []struct {
			ID          int     `json:"id"`
			KB          uint64  `json:"kb"`
			KBUsed      uint64  `json:"kb_used"`
			Utilization float64 `json:"utilization"`
		} `json:"nodes"`
	}
	_ = json.Unmarshal([]byte(dfOut), &df)
	dfMap := map[int]struct {
		KB     uint64
		KBUsed uint64
		Util   float64
	}{}
	for _, n := range df.Nodes {
		dfMap[n.ID] = struct {
			KB     uint64
			KBUsed uint64
			Util   float64
		}{KB: n.KB, KBUsed: n.KBUsed, Util: n.Utilization}
	}

	// 汇总
	var osdList []model.CephOSD
	ids := make([]int, 0, len(hostOfOsd))
	for id := range hostOfOsd {
		ids = append(ids, id)
	}
	// 排序（按 ID）
	for i := 0; i < len(ids); i++ {
		for j := i + 1; j < len(ids); j++ {
			if ids[j] < ids[i] {
				ids[i], ids[j] = ids[j], ids[i]
			}
		}
	}
	for _, id := range ids {
		st, _ := osdStatus[id]
		osd := model.CephOSD{
			ID:     id,
			Host:   hostOfOsd[id],
			Up:     st.Up,
			In:     st.In,
			Weight: 1.0,
		}
		if d, ok := dfMap[id]; ok {
			osd.SizeBytes = d.KB * 1024
			osd.UsedBytes = d.KBUsed * 1024
			osd.UsedPercent = d.Util
		}
		osdList = append(osdList, osd)
	}

	result.Success(c, osdList)
}

// ===================== Ceph Pool 详细操作 =====================

// CreateCephPool 创建 Ceph pool
func (s *K8sStorageServiceImpl) CreateCephPool(c *gin.Context, clusterId uint, req *model.CreateCephPoolRequest) {
	pgNum := req.PgNum
	if pgNum <= 0 {
		pgNum = 32
	}
	size := req.Size
	if size <= 0 {
		size = 3
	}
	app := req.Application
	if app == "" {
		app = "rbd"
	}
	// 创建 pool（replicated）
	out, err := s.execCephCommand(clusterId, "osd", "pool", "create", req.Name, fmt.Sprintf("%d", pgNum), "replicated")
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "创建Pool失败: "+err.Error()+" "+out)
		return
	}
	// 关联应用类型
	if _, err := s.execCephCommand(clusterId, "osd", "pool", "application", "enable", req.Name, app); err != nil {
		result.Failed(c, http.StatusInternalServerError, "Pool 已创建，但应用关联失败: "+err.Error())
		return
	}
	// 设置副本数
	if _, err := s.execCephCommand(clusterId, "osd", "pool", "set", req.Name, "size", fmt.Sprintf("%d", size)); err != nil {
		result.Failed(c, http.StatusInternalServerError, "Pool 已创建，但副本设置失败: "+err.Error())
		return
	}
	if app == "rbd" {
		if _, err := s.execToolCommand(clusterId, "rbd", "pool", "init", req.Name); err != nil {
			result.Failed(c, http.StatusInternalServerError, "Pool 已创建，但 RBD 初始化失败: "+err.Error())
			return
		}
	}
	result.Success(c, map[string]interface{}{
		"message":     fmt.Sprintf("Pool %s 创建成功", req.Name),
		"poolName":    req.Name,
		"pgNum":       pgNum,
		"size":        size,
		"application": app,
	})
}

// DeleteCephPool 删除 Ceph pool
func (s *K8sStorageServiceImpl) DeleteCephPool(c *gin.Context, clusterId uint, poolName string) {
	if strings.HasPrefix(poolName, ".") {
		result.Failed(c, http.StatusBadRequest, "系统保留 Pool（. 开头）不可删除")
		return
	}
	out, err := s.execCephCommand(clusterId, "osd", "pool", "delete", poolName, poolName, "--yes-i-really-really-mean-it")
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "删除Pool失败: "+err.Error()+" "+out+"（需在 CephCluster 配置中启用 mon_allow_pool_delete）")
		return
	}
	result.Success(c, map[string]interface{}{
		"message":  fmt.Sprintf("Pool %s 已删除", poolName),
		"poolName": poolName,
	})
}

// UpdateCephPoolPGNum 调整 Ceph pool PG 数
func (s *K8sStorageServiceImpl) UpdateCephPoolPGNum(c *gin.Context, clusterId uint, poolName string, req *model.UpdateCephPoolPGNumRequest) {
	if req.PgNum < 1 {
		result.Failed(c, http.StatusBadRequest, "PG 数必须 >= 1")
		return
	}
	out, err := s.execCephCommand(clusterId, "osd", "pool", "set", poolName, "pg_num", fmt.Sprintf("%d", req.PgNum))
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "调整PG数失败: "+err.Error()+" "+out)
		return
	}
	// 同步 pgp_num
	if req.PgNum > 1 {
		_, _ = s.execCephCommand(clusterId, "osd", "pool", "set", poolName, "pgp_num", fmt.Sprintf("%d", req.PgNum))
	}
	result.Success(c, map[string]interface{}{
		"message":  fmt.Sprintf("Pool %s PG 数已调整为 %d", poolName, req.PgNum),
		"poolName": poolName,
		"pgNum":    req.PgNum,
	})
}

// UpdateCephPoolQuota 设置 Ceph pool 配额
func (s *K8sStorageServiceImpl) UpdateCephPoolQuota(c *gin.Context, clusterId uint, poolName string, req *model.UpdateCephPoolQuotaRequest) {
	commands, err := cephQuotaCommands(poolName, req)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, err.Error())
		return
	}
	for _, command := range commands {
		out, err := s.execCephCommand(clusterId, command...)
		if err != nil {
			result.Failed(c, http.StatusInternalServerError, "设置配额失败（此前已成功的字段不会自动回滚）: "+err.Error()+" "+out)
			return
		}
	}
	result.Success(c, map[string]interface{}{
		"message":    fmt.Sprintf("Pool %s 配额已设置", poolName),
		"poolName":   poolName,
		"maxBytes":   req.MaxBytes,
		"maxObjects": req.MaxObjects,
	})
}

// UpdateCephPoolCompression 设置 Ceph pool 压缩
func (s *K8sStorageServiceImpl) UpdateCephPoolCompression(c *gin.Context, clusterId uint, poolName string, req *model.UpdateCephPoolCompressionRequest) {
	validAlg := map[string]bool{"lz4": true, "snappy": true, "zlib": true, "zstd": true}
	validMode := map[string]bool{"none": true, "passive": true, "aggressive": true, "force": true}
	if req.Algorithm != "" && !validAlg[req.Algorithm] {
		result.Failed(c, http.StatusBadRequest, "压缩算法必须为 lz4/snappy/zlib/zstd")
		return
	}
	if req.Mode != "" && !validMode[req.Mode] {
		result.Failed(c, http.StatusBadRequest, "压缩模式必须为 none/passive/aggressive/force")
		return
	}
	applied := []string{}
	if req.Algorithm != "" {
		if _, err := s.execCephCommand(clusterId, "osd", "pool", "set", poolName, "compression_algorithm", req.Algorithm); err != nil {
			result.Failed(c, http.StatusInternalServerError, "设置压缩算法失败: "+err.Error())
			return
		}
		applied = append(applied, "algorithm="+req.Algorithm)
	}
	if req.Mode != "" {
		if _, err := s.execCephCommand(clusterId, "osd", "pool", "set", poolName, "compression_mode", req.Mode); err != nil {
			result.Failed(c, http.StatusInternalServerError, "设置压缩模式失败（已更新的算法不会回滚）: "+err.Error())
			return
		}
		applied = append(applied, "mode="+req.Mode)
	}
	if len(applied) == 0 {
		result.Failed(c, http.StatusBadRequest, "未设置任何压缩参数")
		return
	}
	result.Success(c, map[string]interface{}{
		"message":  fmt.Sprintf("Pool %s 压缩配置已更新: %s", poolName, strings.Join(applied, ", ")),
		"poolName": poolName,
	})
}

// ===================== Ceph RBD 块存储 =====================

// parseSizeToBytes 将容量字符串（如 10G/5T/1024M）解析为字节数
func parseSizeToBytes(size string) uint64 {
	size = strings.ToUpper(strings.TrimSpace(size))
	mult := uint64(1)
	if len(size) > 1 {
		last := size[len(size)-1]
		switch last {
		case 'K':
			mult = 1024
			size = size[:len(size)-1]
		case 'M':
			mult = 1024 * 1024
			size = size[:len(size)-1]
		case 'G':
			mult = 1024 * 1024 * 1024
			size = size[:len(size)-1]
		case 'T':
			mult = 1024 * 1024 * 1024 * 1024
			size = size[:len(size)-1]
		}
	}
	var val uint64
	fmt.Sscanf(size, "%d", &val)
	return val * mult
}

// rbdPoolNames 获取所有已初始化的 RBD pool 名
func (s *K8sStorageServiceImpl) rbdPoolNames(clusterId uint) ([]string, error) {
	out, err := s.execCephCommand(clusterId, "osd", "pool", "ls", "-f", "json")
	if err != nil {
		return nil, err
	}
	var pools []string
	if err := json.Unmarshal([]byte(out), &pools); err != nil {
		return nil, err
	}
	return pools, nil
}

// GetCephRBDImages 获取所有 pool 的 RBD 镜像列表
func (s *K8sStorageServiceImpl) GetCephRBDImages(c *gin.Context, clusterId uint) {
	pools, err := s.rbdPoolNames(clusterId)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "获取Pool列表失败: "+err.Error())
		return
	}
	var images []model.CephRBDImage
	for _, pool := range pools {
		out, err := s.execToolCommand(clusterId, "rbd", "ls", "-l", "--pool", pool, "--format", "json")
		if err != nil {
			continue // pool 未初始化或不可用，跳过
		}
		var list []struct {
			Image  string `json:"image"`
			Size   uint64 `json:"size"`
			Format int    `json:"format"`
		}
		if json.Unmarshal([]byte(out), &list) != nil {
			continue
		}
		for _, img := range list {
			images = append(images, model.CephRBDImage{
				Pool:   pool,
				Name:   img.Image,
				Size:   img.Size,
				Format: img.Format,
			})
		}
	}
	result.Success(c, images)
}

// CreateCephRBDImage 创建 RBD 镜像
func (s *K8sStorageServiceImpl) CreateCephRBDImage(c *gin.Context, clusterId uint, req *model.CreateCephRBDImageRequest) {
	if parseSizeToBytes(req.Size) <= 0 {
		result.Failed(c, http.StatusBadRequest, "容量必须 > 0（如 10G/5T/1024M）")
		return
	}
	// 初始化 pool（若未初始化，RBD 创建会失败）
	if _, err := s.execToolCommand(clusterId, "rbd", "pool", "init", req.Pool); err != nil {
		_ = err // 已初始化则忽略
	}
	out, err := s.execToolCommand(clusterId, "rbd", "create", req.Pool+"/"+req.Name, "--size", req.Size)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "创建RBD镜像失败: "+err.Error()+" "+out)
		return
	}
	result.Success(c, map[string]interface{}{
		"message": fmt.Sprintf("RBD 镜像 %s/%s 创建成功（%s）", req.Pool, req.Name, req.Size),
		"pool":    req.Pool,
		"name":    req.Name,
		"size":    req.Size,
	})
}

// GetCephRBDImageDetail 获取 RBD 镜像详情
func (s *K8sStorageServiceImpl) GetCephRBDImageDetail(c *gin.Context, clusterId uint, pool, image string) {
	out, err := s.execToolCommand(clusterId, "rbd", "info", pool+"/"+image, "--format", "json")
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "获取RBD镜像详情失败: "+err.Error()+" "+out)
		return
	}
	var info struct {
		Name       string   `json:"name"`
		Size       uint64   `json:"size"`
		Objects    int      `json:"objects"`
		Order      int      `json:"order"`
		ObjectSize uint64   `json:"object_size"`
		Format     int      `json:"format"`
		Features   []string `json:"features"`
		SnapCount  int      `json:"snapshot_count"`
		Parent     string   `json:"parent"`
	}
	if json.Unmarshal([]byte(out), &info) != nil {
		result.Failed(c, http.StatusInternalServerError, "解析RBD镜像详情失败")
		return
	}
	detail := model.CephRBDImage{
		Pool:       pool,
		Name:       info.Name,
		Size:       info.Size,
		Objects:    info.Objects,
		Order:      info.Order,
		ObjectSize: info.ObjectSize,
		Format:     info.Format,
		Features:   strings.Join(info.Features, ","),
		SnapCount:  info.SnapCount,
		Parent:     info.Parent,
	}
	// 获取实际使用量
	if du, err := s.execToolCommand(clusterId, "rbd", "du", pool+"/"+image, "--format", "json"); err == nil {
		var d struct {
			Images []struct {
				UsedSize uint64 `json:"used_size"`
			} `json:"images"`
		}
		if json.Unmarshal([]byte(du), &d) == nil && len(d.Images) > 0 {
			detail.UsedSize = d.Images[0].UsedSize
		}
	}
	result.Success(c, detail)
}

// ResizeCephRBDImage 扩容/缩容 RBD 镜像
func (s *K8sStorageServiceImpl) ResizeCephRBDImage(c *gin.Context, clusterId uint, pool, image string, req *model.ResizeCephRBDImageRequest) {
	if strings.HasPrefix(image, "csi-vol-") {
		result.Failed(c, http.StatusBadRequest, "CSI 镜像请通过 PVC 扩容")
		return
	}
	if req.AllowShrink {
		if err := s.requireOfflineCephImage(clusterId, pool, image); err != nil {
			result.Failed(c, http.StatusBadRequest, err.Error())
			return
		}
	}
	if parseSizeToBytes(req.Size) <= 0 {
		result.Failed(c, http.StatusBadRequest, "目标容量必须 > 0")
		return
	}
	args := []string{"resize", pool + "/" + image, "--size", req.Size}
	if req.AllowShrink {
		args = append(args, "--allow-shrink")
	}
	out, err := s.execToolCommand(clusterId, "rbd", args...)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "调整RBD镜像容量失败: "+err.Error()+" "+out)
		return
	}
	result.Success(c, map[string]interface{}{
		"message": fmt.Sprintf("RBD 镜像 %s/%s 容量已调整为 %s", pool, image, req.Size),
		"pool":    pool,
		"name":    image,
		"size":    req.Size,
	})
}

// DeleteCephRBDImage 删除 RBD 镜像
func (s *K8sStorageServiceImpl) DeleteCephRBDImage(c *gin.Context, clusterId uint, pool, image string) {
	if err := s.requireOfflineCephImage(clusterId, pool, image); err != nil {
		result.Failed(c, http.StatusBadRequest, err.Error())
		return
	}
	out, err := s.execToolCommand(clusterId, "rbd", "rm", pool+"/"+image)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "删除RBD镜像失败: "+err.Error()+" "+out+"（若有快照需先删除快照）")
		return
	}
	result.Success(c, map[string]interface{}{
		"message": fmt.Sprintf("RBD 镜像 %s/%s 已删除", pool, image),
		"pool":    pool,
		"name":    image,
	})
}

// ListCephRBDSnapshots 获取 RBD 镜像快照列表
func (s *K8sStorageServiceImpl) ListCephRBDSnapshots(c *gin.Context, clusterId uint, pool, image string) {
	out, err := s.execToolCommand(clusterId, "rbd", "snap", "ls", pool+"/"+image, "--format", "json")
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "获取RBD快照列表失败: "+err.Error()+" "+out)
		return
	}
	var snaps []struct {
		Id        int             `json:"id"` // id 为数字
		Name      string          `json:"name"`
		Size      uint64          `json:"size"`
		Protected json.RawMessage `json:"protected"` // Reef 输出可能为布尔值或字符串
		Timestamp string          `json:"timestamp"`
	}
	if json.Unmarshal([]byte(out), &snaps) != nil {
		result.Failed(c, http.StatusInternalServerError, "解析RBD快照列表失败")
		return
	}
	list := make([]model.CephRBDSnapshot, 0)
	for _, s := range snaps {
		list = append(list, model.CephRBDSnapshot{
			Id:        fmt.Sprintf("%d", s.Id),
			Name:      s.Name,
			Size:      s.Size,
			Protected: string(s.Protected) == "true" || string(s.Protected) == `"true"`,
			Timestamp: s.Timestamp,
		})
	}
	result.Success(c, list)
}

// CreateCephRBDSnapshot 创建 RBD 快照
func (s *K8sStorageServiceImpl) CreateCephRBDSnapshot(c *gin.Context, clusterId uint, pool, image string, req *model.CreateCephRBDSnapshotRequest) {
	if req.SnapName == "" {
		result.Failed(c, http.StatusBadRequest, "快照名不能为空")
		return
	}
	out, err := s.execToolCommand(clusterId, "rbd", "snap", "create", pool+"/"+image+"@"+req.SnapName)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "创建RBD快照失败: "+err.Error()+" "+out)
		return
	}
	result.Success(c, map[string]interface{}{
		"message": fmt.Sprintf("快照 %s/%s@%s 已创建", pool, image, req.SnapName),
		"pool":    pool,
		"image":   image,
		"snap":    req.SnapName,
	})
}

// DeleteCephRBDSnapshot 删除 RBD 快照
func (s *K8sStorageServiceImpl) DeleteCephRBDSnapshot(c *gin.Context, clusterId uint, pool, image, snap string) {
	out, err := s.execToolCommand(clusterId, "rbd", "snap", "rm", pool+"/"+image+"@"+snap)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "删除RBD快照失败: "+err.Error()+" "+out+"（受保护快照需先取消保护）")
		return
	}
	result.Success(c, map[string]interface{}{
		"message": fmt.Sprintf("快照 %s/%s@%s 已删除", pool, image, snap),
		"pool":    pool,
		"image":   image,
		"snap":    snap,
	})
}

// RollbackCephRBDImage 回滚 RBD 镜像到快照
func (s *K8sStorageServiceImpl) RollbackCephRBDImage(c *gin.Context, clusterId uint, pool, image, snap string) {
	if err := s.requireOfflineCephImage(clusterId, pool, image); err != nil {
		result.Failed(c, http.StatusBadRequest, err.Error())
		return
	}
	out, err := s.execToolCommand(clusterId, "rbd", "snap", "rollback", pool+"/"+image+"@"+snap)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "回滚失败: "+err.Error()+" "+out+"（回滚会覆盖当前数据，请确认镜像未挂载使用）")
		return
	}
	result.Success(c, map[string]interface{}{
		"message": fmt.Sprintf("镜像 %s/%s 已回滚到快照 %s", pool, image, snap),
		"pool":    pool,
		"image":   image,
		"snap":    snap,
	})
}

// CloneCephRBDImage 克隆 RBD 镜像（基于受保护快照）
func (s *K8sStorageServiceImpl) CloneCephRBDImage(c *gin.Context, clusterId uint, pool, image string, req *model.CloneCephRBDImageRequest) {
	out, err := s.execToolCommand(clusterId, "rbd", "clone", pool+"/"+image+"@"+req.SnapName, req.TargetPool+"/"+req.TargetName)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "克隆失败: "+err.Error()+" "+out+"（源快照需已保护，且目标 pool 需已初始化）")
		return
	}
	result.Success(c, map[string]interface{}{
		"message":    fmt.Sprintf("已从 %s/%s@%s 克隆到 %s/%s", pool, image, req.SnapName, req.TargetPool, req.TargetName),
		"pool":       req.TargetPool,
		"name":       req.TargetName,
		"sourceSnap": req.SnapName,
	})
}

// ===================== Ceph CephFS / 监控 / OSD 管理 =====================

// GetCephFS 获取 CephFS 文件系统列表
func (s *K8sStorageServiceImpl) GetCephFS(c *gin.Context, clusterId uint) {
	out, err := s.execCephCommand(clusterId, "fs", "ls", "-f", "json")
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "获取CephFS失败: "+err.Error())
		return
	}
	var fsList []struct {
		Name         string   `json:"name"`
		MetadataPool string   `json:"metadata_pool"`
		DataPools    []string `json:"data_pools"`
	}
	if json.Unmarshal([]byte(out), &fsList) != nil {
		result.Failed(c, http.StatusInternalServerError, "解析CephFS失败")
		return
	}
	var resultList []model.CephFS
	for _, f := range fsList {
		item := model.CephFS{
			Name:         f.Name,
			MetadataPool: f.MetadataPool,
			DataPools:    f.DataPools,
		}
		// 获取 fs status 判断状态和 MDS 数量
		if st, err := s.execCephCommand(clusterId, "fs", "status", f.Name, "-f", "json"); err == nil {
			var stMap map[string]interface{}
			if json.Unmarshal([]byte(st), &stMap) == nil {
				// mdsmap 是数组（每个 MDS 一项）
				if mdsmap, ok := stMap["mdsmap"].([]interface{}); ok {
					for _, m := range mdsmap {
						if md, ok := m.(map[string]interface{}); ok {
							item.MdsCount++
							if state, ok := md["state"].(string); ok && item.State == "" {
								item.State = state
							}
						}
					}
				}
			}
		}
		resultList = append(resultList, item)
	}
	result.Success(c, resultList)
}

// GetCephHealth 获取 Ceph 健康详情
func (s *K8sStorageServiceImpl) GetCephHealth(c *gin.Context, clusterId uint) {
	out, err := s.execCephCommand(clusterId, "health", "detail", "-f", "json")
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "获取Ceph健康详情失败: "+err.Error())
		return
	}
	var health map[string]interface{}
	if json.Unmarshal([]byte(out), &health) != nil {
		result.Failed(c, http.StatusInternalServerError, "解析Ceph健康详情失败")
		return
	}
	result.Success(c, health)
}

// GetCephVersions 获取 Ceph 组件版本分布
func (s *K8sStorageServiceImpl) GetCephVersions(c *gin.Context, clusterId uint) {
	out, err := s.execCephCommand(clusterId, "versions", "-f", "json")
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "获取Ceph版本失败: "+err.Error())
		return
	}
	var versions map[string]interface{}
	if json.Unmarshal([]byte(out), &versions) != nil {
		result.Failed(c, http.StatusInternalServerError, "解析Ceph版本失败")
		return
	}
	result.Success(c, versions)
}

// GetCephPGStat 获取 Ceph PG 状态统计
func (s *K8sStorageServiceImpl) GetCephPGStat(c *gin.Context, clusterId uint) {
	out, err := s.execCephCommand(clusterId, "pg", "stat", "-f", "json")
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "获取Ceph PG状态失败: "+err.Error())
		return
	}
	var pgStat map[string]interface{}
	if json.Unmarshal([]byte(out), &pgStat) != nil {
		result.Failed(c, http.StatusInternalServerError, "解析Ceph PG状态失败")
		return
	}
	result.Success(c, pgStat)
}

// SetCephOSDStatus 标记 OSD in/out/down
func (s *K8sStorageServiceImpl) SetCephOSDStatus(c *gin.Context, clusterId uint, osdId int, action string) {
	if action != "in" && action != "out" && action != "down" {
		result.Failed(c, http.StatusBadRequest, "动作必须为 in/out/down")
		return
	}
	out, err := s.execCephCommand(clusterId, "osd", action, fmt.Sprintf("%d", osdId))
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "标记OSD失败: "+err.Error()+" "+out)
		return
	}
	result.Success(c, map[string]interface{}{
		"message": fmt.Sprintf("OSD.%d 已标记为 %s", osdId, action),
		"osd":     osdId,
		"action":  action,
	})
}

// ReweightCephOSD 调整 OSD 权重
func (s *K8sStorageServiceImpl) ReweightCephOSD(c *gin.Context, clusterId uint, osdId int, req *model.ReweightCephOSDRequest) {
	if req.Weight < 0 || req.Weight > 1 {
		result.Failed(c, http.StatusBadRequest, "权重必须在 0-1 之间")
		return
	}
	out, err := s.execCephCommand(clusterId, "osd", "reweight", fmt.Sprintf("%d", osdId), fmt.Sprintf("%.6f", req.Weight))
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "调整OSD权重失败: "+err.Error()+" "+out)
		return
	}
	result.Success(c, map[string]interface{}{
		"message": fmt.Sprintf("OSD.%d 权重已调整为 %.4f", osdId, req.Weight),
		"osd":     osdId,
		"weight":  req.Weight,
	})
}

// DeleteCephOSD 删除 OSD（危险操作，需二次确认）
func (s *K8sStorageServiceImpl) DeleteCephOSD(c *gin.Context, clusterId uint, osdId int) {
	if osdId < 0 {
		result.Failed(c, http.StatusBadRequest, "无效的 OSD ID")
		return
	}
	if _, err := s.execCephCommand(clusterId, "osd", "safe-to-destroy", fmt.Sprintf("%d", osdId)); err != nil {
		result.Failed(c, http.StatusBadRequest, "OSD 尚不满足 safe-to-destroy，拒绝移除: "+err.Error())
		return
	}
	out, err := s.execCephCommand(clusterId, "osd", "purge", fmt.Sprintf("%d", osdId), "--yes-i-really-mean-it")
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "删除OSD失败: "+err.Error()+" "+out)
		return
	}
	result.Success(c, map[string]interface{}{
		"message": fmt.Sprintf("OSD.%d 已从集群移除（请同步清理节点上的相关存储配置）", osdId),
		"osd":     osdId,
	})
}

// ===================== PVC 管理实现 =====================

// GetPVCs 获取PVC列表
func (s *K8sStorageServiceImpl) GetPVCs(c *gin.Context, clusterId uint, namespaceName string) {
	clientset, err := s.getK8sClient(clusterId)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "连接K8s集群失败: "+err.Error())
		return
	}

	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(namespaceName).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "获取PVC列表失败: "+err.Error())
		return
	}

	var pvcList []model.K8sPersistentVolumeClaim
	for _, pvc := range pvcs.Items {
		k8sPVC := s.convertToK8sPVC(&pvc)
		pvcList = append(pvcList, k8sPVC)
	}

	response := model.PVCListResponse{
		PVCs:  pvcList,
		Total: len(pvcList),
	}

	result.Success(c, response)
}

// GetPVCDetail 获取PVC详情
func (s *K8sStorageServiceImpl) GetPVCDetail(c *gin.Context, clusterId uint, namespaceName string, pvcName string) {
	clientset, err := s.getK8sClient(clusterId)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "连接K8s集群失败: "+err.Error())
		return
	}

	pvc, err := clientset.CoreV1().PersistentVolumeClaims(namespaceName).Get(context.TODO(), pvcName, metav1.GetOptions{})
	if err != nil {
		result.Failed(c, http.StatusNotFound, "PVC不存在: "+err.Error())
		return
	}

	k8sPVC := s.convertToK8sPVC(pvc)

	// 获取相关事件
	events, _ := s.getPVCEvents(clientset, namespaceName, pvcName)

	pvcDetail := model.PVCDetail{
		K8sPersistentVolumeClaim: k8sPVC,
		Events:                   events,
		Spec:                     pvc.Spec,
	}

	result.Success(c, pvcDetail)
}

// CreatePVC 创建PVC
func (s *K8sStorageServiceImpl) CreatePVC(c *gin.Context, clusterId uint, namespaceName string, req *model.CreatePVCRequest) {
	clientset, err := s.getK8sClient(clusterId)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "连接K8s集群失败: "+err.Error())
		return
	}

	// 构建PVC对象
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: namespaceName,
			Labels:    req.Labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{},
			},
		},
	}

	// 设置访问模式
	for _, mode := range req.AccessModes {
		pvc.Spec.AccessModes = append(pvc.Spec.AccessModes, corev1.PersistentVolumeAccessMode(mode))
	}

	// 设置资源请求
	for key, value := range req.Resources.Requests {
		quantity, err := resource.ParseQuantity(value)
		if err != nil {
			result.Failed(c, http.StatusBadRequest, fmt.Sprintf("无效的资源请求 %s: %v", key, err))
			return
		}
		pvc.Spec.Resources.Requests[corev1.ResourceName(key)] = quantity
	}

	// 设置资源限制
	if len(req.Resources.Limits) > 0 {
		pvc.Spec.Resources.Limits = corev1.ResourceList{}
		for key, value := range req.Resources.Limits {
			quantity, err := resource.ParseQuantity(value)
			if err != nil {
				result.Failed(c, http.StatusBadRequest, fmt.Sprintf("无效的资源限制 %s: %v", key, err))
				return
			}
			pvc.Spec.Resources.Limits[corev1.ResourceName(key)] = quantity
		}
	}

	// 设置存储类
	if req.StorageClass != "" {
		pvc.Spec.StorageClassName = &req.StorageClass
	}

	// 设置卷模式
	if req.VolumeMode != "" {
		volumeMode := corev1.PersistentVolumeMode(req.VolumeMode)
		pvc.Spec.VolumeMode = &volumeMode
	}

	// 设置选择器
	if req.Selector != nil {
		pvc.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: req.Selector.MatchLabels,
		}
		if len(req.Selector.MatchExpressions) > 0 {
			for _, expr := range req.Selector.MatchExpressions {
				pvc.Spec.Selector.MatchExpressions = append(pvc.Spec.Selector.MatchExpressions, metav1.LabelSelectorRequirement{
					Key:      expr.Key,
					Operator: metav1.LabelSelectorOperator(expr.Operator),
					Values:   expr.Values,
				})
			}
		}
	}

	// 创建PVC
	createdPVC, err := clientset.CoreV1().PersistentVolumeClaims(namespaceName).Create(context.TODO(), pvc, metav1.CreateOptions{})
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "创建PVC失败: "+err.Error())
		return
	}

	k8sPVC := s.convertToK8sPVC(createdPVC)
	result.Success(c, k8sPVC)
}

// UpdatePVC 更新PVC
func (s *K8sStorageServiceImpl) UpdatePVC(c *gin.Context, clusterId uint, namespaceName string, pvcName string, req *model.UpdatePVCRequest) {
	clientset, err := s.getK8sClient(clusterId)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "连接K8s集群失败: "+err.Error())
		return
	}

	// 获取现有PVC
	existingPVC, err := clientset.CoreV1().PersistentVolumeClaims(namespaceName).Get(context.TODO(), pvcName, metav1.GetOptions{})
	if err != nil {
		result.Failed(c, http.StatusNotFound, "PVC不存在: "+err.Error())
		return
	}

	// 更新标签
	if req.Labels != nil {
		existingPVC.Labels = req.Labels
	}

	// 更新资源请求（仅支持存储扩容）
	if len(req.Resources.Requests) > 0 {
		for key, value := range req.Resources.Requests {
			quantity, err := resource.ParseQuantity(value)
			if err != nil {
				result.Failed(c, http.StatusBadRequest, fmt.Sprintf("无效的资源请求 %s: %v", key, err))
				return
			}
			existingPVC.Spec.Resources.Requests[corev1.ResourceName(key)] = quantity
		}
	}

	// 更新PVC
	updatedPVC, err := clientset.CoreV1().PersistentVolumeClaims(namespaceName).Update(context.TODO(), existingPVC, metav1.UpdateOptions{})
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "更新PVC失败: "+err.Error())
		return
	}

	k8sPVC := s.convertToK8sPVC(updatedPVC)
	result.Success(c, k8sPVC)
}

// DeletePVC 删除PVC
func (s *K8sStorageServiceImpl) DeletePVC(c *gin.Context, clusterId uint, namespaceName string, pvcName string) {
	clientset, err := s.getK8sClient(clusterId)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "连接K8s集群失败: "+err.Error())
		return
	}

	err = clientset.CoreV1().PersistentVolumeClaims(namespaceName).Delete(context.TODO(), pvcName, metav1.DeleteOptions{})
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "删除PVC失败: "+err.Error())
		return
	}

	result.Success(c, "PVC删除成功")
}

// GetPVCYaml 获取PVC的YAML配置
func (s *K8sStorageServiceImpl) GetPVCYaml(c *gin.Context, clusterId uint, namespaceName string, pvcName string) {
	clientset, err := s.getK8sClient(clusterId)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "连接K8s集群失败: "+err.Error())
		return
	}

	pvc, err := clientset.CoreV1().PersistentVolumeClaims(namespaceName).Get(context.TODO(), pvcName, metav1.GetOptions{})
	if err != nil {
		result.Failed(c, http.StatusNotFound, "PVC不存在: "+err.Error())
		return
	}

	// 清理不需要的字段，但保留 status
	pvc.ManagedFields = nil

	// 确保 apiVersion 和 kind 字段存在
	if pvc.APIVersion == "" {
		pvc.APIVersion = "v1"
	}
	if pvc.Kind == "" {
		pvc.Kind = "PersistentVolumeClaim"
	}

	// 清理元数据字段
	pvc.ResourceVersion = ""
	pvc.UID = ""
	pvc.Generation = 0
	pvc.CreationTimestamp = metav1.Time{}
	pvc.SelfLink = ""

	yamlData, err := yaml.Marshal(pvc)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "转换YAML失败: "+err.Error())
		return
	}

	result.Success(c, map[string]interface{}{
		"yaml": string(yamlData),
	})
}

// UpdatePVCYaml 通过YAML更新PVC
func (s *K8sStorageServiceImpl) UpdatePVCYaml(c *gin.Context, clusterId uint, namespaceName string, pvcName string, yamlData map[string]interface{}) {
	clientset, err := s.getK8sClient(clusterId)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "连接K8s集群失败: "+err.Error())
		return
	}

	// 将map转换为YAML字符串
	yamlBytes, err := json.Marshal(yamlData)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "YAML数据格式错误: "+err.Error())
		return
	}

	var pvc corev1.PersistentVolumeClaim
	err = yaml.Unmarshal(yamlBytes, &pvc)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "YAML解析失败: "+err.Error())
		return
	}

	// 确保名称和命名空间正确
	pvc.Name = pvcName
	pvc.Namespace = namespaceName

	// 更新PVC
	_, err = clientset.CoreV1().PersistentVolumeClaims(namespaceName).Update(context.TODO(), &pvc, metav1.UpdateOptions{})
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "更新PVC失败: "+err.Error())
		return
	}

	result.Success(c, "PVC YAML更新成功")
}

// ===================== PV 管理实现 =====================

// GetPVs 获取PV列表
func (s *K8sStorageServiceImpl) GetPVs(c *gin.Context, clusterId uint) {
	clientset, err := s.getK8sClient(clusterId)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "连接K8s集群失败: "+err.Error())
		return
	}

	pvs, err := clientset.CoreV1().PersistentVolumes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "获取PV列表失败: "+err.Error())
		return
	}

	var pvList []model.K8sPersistentVolume
	for _, pv := range pvs.Items {
		k8sPV := s.convertToK8sPV(&pv)
		pvList = append(pvList, k8sPV)
	}

	response := model.PVListResponse{
		PVs:   pvList,
		Total: len(pvList),
	}

	result.Success(c, response)
}

// GetPVDetail 获取PV详情
func (s *K8sStorageServiceImpl) GetPVDetail(c *gin.Context, clusterId uint, pvName string) {
	clientset, err := s.getK8sClient(clusterId)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "连接K8s集群失败: "+err.Error())
		return
	}

	pv, err := clientset.CoreV1().PersistentVolumes().Get(context.TODO(), pvName, metav1.GetOptions{})
	if err != nil {
		result.Failed(c, http.StatusNotFound, "PV不存在: "+err.Error())
		return
	}

	k8sPV := s.convertToK8sPV(pv)

	// 获取相关事件
	events, _ := s.getPVEvents(clientset, pvName)

	pvDetail := model.PVDetail{
		K8sPersistentVolume: k8sPV,
		Events:              events,
		Spec:                pv.Spec,
	}

	result.Success(c, pvDetail)
}

// CreatePV 创建PV
func (s *K8sStorageServiceImpl) CreatePV(c *gin.Context, clusterId uint, req *model.CreatePVRequest) {
	clientset, err := s.getK8sClient(clusterId)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "连接K8s集群失败: "+err.Error())
		return
	}

	// 构建PV对象
	pv := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{
			Name:   req.Name,
			Labels: req.Labels,
		},
		Spec: corev1.PersistentVolumeSpec{
			Capacity:    corev1.ResourceList{},
			AccessModes: []corev1.PersistentVolumeAccessMode{},
		},
	}

	// 设置容量
	for key, value := range req.Capacity {
		quantity, err := resource.ParseQuantity(value)
		if err != nil {
			result.Failed(c, http.StatusBadRequest, fmt.Sprintf("无效的容量 %s: %v", key, err))
			return
		}
		pv.Spec.Capacity[corev1.ResourceName(key)] = quantity
	}

	// 设置访问模式
	for _, mode := range req.AccessModes {
		pv.Spec.AccessModes = append(pv.Spec.AccessModes, corev1.PersistentVolumeAccessMode(mode))
	}

	// 设置回收策略
	if req.ReclaimPolicy != "" {
		reclaimPolicy := corev1.PersistentVolumeReclaimPolicy(req.ReclaimPolicy)
		pv.Spec.PersistentVolumeReclaimPolicy = reclaimPolicy
	}

	// 设置存储类
	if req.StorageClassName != "" {
		pv.Spec.StorageClassName = req.StorageClassName
	}

	// 设置卷模式
	if req.VolumeMode != "" {
		volumeMode := corev1.PersistentVolumeMode(req.VolumeMode)
		pv.Spec.VolumeMode = &volumeMode
	}

	// 设置挂载选项
	pv.Spec.MountOptions = req.MountOptions

	// 设置存储源
	if req.PersistentVolumeSource.HostPath != nil {
		pv.Spec.PersistentVolumeSource.HostPath = &corev1.HostPathVolumeSource{
			Path: req.PersistentVolumeSource.HostPath.Path,
			Type: (*corev1.HostPathType)(&req.PersistentVolumeSource.HostPath.Type),
		}
	}

	if req.PersistentVolumeSource.NFS != nil {
		pv.Spec.PersistentVolumeSource.NFS = &corev1.NFSVolumeSource{
			Server:   req.PersistentVolumeSource.NFS.Server,
			Path:     req.PersistentVolumeSource.NFS.Path,
			ReadOnly: req.PersistentVolumeSource.NFS.ReadOnly,
		}
	}

	if req.PersistentVolumeSource.ISCSI != nil {
		pv.Spec.PersistentVolumeSource.ISCSI = &corev1.ISCSIPersistentVolumeSource{
			TargetPortal:      req.PersistentVolumeSource.ISCSI.TargetPortal,
			IQN:               req.PersistentVolumeSource.ISCSI.IQN,
			Lun:               req.PersistentVolumeSource.ISCSI.Lun,
			ISCSIInterface:    req.PersistentVolumeSource.ISCSI.ISCSIInterface,
			FSType:            req.PersistentVolumeSource.ISCSI.FSType,
			ReadOnly:          req.PersistentVolumeSource.ISCSI.ReadOnly,
			Portals:           req.PersistentVolumeSource.ISCSI.Portals,
			DiscoveryCHAPAuth: false,
			SessionCHAPAuth:   false,
		}
	}

	if req.PersistentVolumeSource.CSI != nil {
		pv.Spec.PersistentVolumeSource.CSI = &corev1.CSIPersistentVolumeSource{
			Driver:           req.PersistentVolumeSource.CSI.Driver,
			VolumeHandle:     req.PersistentVolumeSource.CSI.VolumeHandle,
			ReadOnly:         req.PersistentVolumeSource.CSI.ReadOnly,
			FSType:           req.PersistentVolumeSource.CSI.FSType,
			VolumeAttributes: req.PersistentVolumeSource.CSI.VolumeAttributes,
		}
	}

	if req.PersistentVolumeSource.Local != nil {
		pv.Spec.PersistentVolumeSource.Local = &corev1.LocalVolumeSource{
			Path:   req.PersistentVolumeSource.Local.Path,
			FSType: &req.PersistentVolumeSource.Local.FSType,
		}
	}

	if req.PersistentVolumeSource.AWSElasticBlockStore != nil {
		pv.Spec.PersistentVolumeSource.AWSElasticBlockStore = &corev1.AWSElasticBlockStoreVolumeSource{
			VolumeID:  req.PersistentVolumeSource.AWSElasticBlockStore.VolumeID,
			FSType:    req.PersistentVolumeSource.AWSElasticBlockStore.FSType,
			Partition: req.PersistentVolumeSource.AWSElasticBlockStore.Partition,
			ReadOnly:  req.PersistentVolumeSource.AWSElasticBlockStore.ReadOnly,
		}
	}

	// 设置节点亲和性（用户显式传入）
	if req.NodeAffinity != nil && req.NodeAffinity.Required != nil {
		var terms []corev1.NodeSelectorTerm
		for _, t := range req.NodeAffinity.Required.NodeSelectorTerms {
			term := corev1.NodeSelectorTerm{}
			for _, e := range t.MatchExpressions {
				term.MatchExpressions = append(term.MatchExpressions, corev1.NodeSelectorRequirement{
					Key:      e.Key,
					Operator: corev1.NodeSelectorOperator(e.Operator),
					Values:   e.Values,
				})
			}
			for _, f := range t.MatchFields {
				term.MatchFields = append(term.MatchFields, corev1.NodeSelectorRequirement{
					Key:      f.Key,
					Operator: corev1.NodeSelectorOperator(f.Operator),
					Values:   f.Values,
				})
			}
			terms = append(terms, term)
		}
		pv.Spec.NodeAffinity = &corev1.VolumeNodeAffinity{
			Required: &corev1.NodeSelector{NodeSelectorTerms: terms},
		}
	}

	// Local 本地卷必须设置 nodeAffinity（K8s 强制校验: Local volume requires node affinity）
	// 若用户未传入 nodeAffinity，则根据 nodeName 或自动选择集群中第一个可用 worker 节点补充
	if req.PersistentVolumeSource.Local != nil && pv.Spec.NodeAffinity == nil {
		nodeName := req.NodeName
		if nodeName == "" {
			nodeName = s.getFirstWorkerNode(clientset)
		}
		if nodeName == "" {
			result.Failed(c, http.StatusBadRequest, "创建Local卷必须指定节点(nodeName)，且集群中无可用worker节点")
			return
		}
		pv.Spec.NodeAffinity = &corev1.VolumeNodeAffinity{
			Required: &corev1.NodeSelector{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{
					{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      "kubernetes.io/hostname",
								Operator: corev1.NodeSelectorOpIn,
								Values:   []string{nodeName},
							},
						},
					},
				},
			},
		}
	}

	// 创建PV
	createdPV, err := clientset.CoreV1().PersistentVolumes().Create(context.TODO(), pv, metav1.CreateOptions{})
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "创建PV失败: "+err.Error())
		return
	}

	k8sPV := s.convertToK8sPV(createdPV)
	result.Success(c, k8sPV)
}

// getFirstWorkerNode 获取集群中第一个可用 worker 节点（优先非 control-plane 且 Ready）
func (s *K8sStorageServiceImpl) getFirstWorkerNode(clientset *kubernetes.Clientset) string {
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return ""
	}
	for _, n := range nodes.Items {
		// 跳过 control-plane/master 节点
		isMaster := false
		for label := range n.Labels {
			if label == "node-role.kubernetes.io/master" || label == "node-role.kubernetes.io/control-plane" {
				isMaster = true
				break
			}
		}
		if isMaster {
			continue
		}
		// 只选择 Ready 节点
		for _, cond := range n.Status.Conditions {
			if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
				return n.Name
			}
		}
	}
	return ""
}

// UpdatePV 更新PV
func (s *K8sStorageServiceImpl) UpdatePV(c *gin.Context, clusterId uint, pvName string, req *model.UpdatePVRequest) {
	clientset, err := s.getK8sClient(clusterId)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "连接K8s集群失败: "+err.Error())
		return
	}

	// 获取现有PV
	existingPV, err := clientset.CoreV1().PersistentVolumes().Get(context.TODO(), pvName, metav1.GetOptions{})
	if err != nil {
		result.Failed(c, http.StatusNotFound, "PV不存在: "+err.Error())
		return
	}

	// 更新标签
	if req.Labels != nil {
		existingPV.Labels = req.Labels
	}

	// 更新回收策略
	if req.ReclaimPolicy != "" {
		reclaimPolicy := corev1.PersistentVolumeReclaimPolicy(req.ReclaimPolicy)
		existingPV.Spec.PersistentVolumeReclaimPolicy = reclaimPolicy
	}

	// 更新挂载选项
	if req.MountOptions != nil {
		existingPV.Spec.MountOptions = req.MountOptions
	}

	// 更新PV
	updatedPV, err := clientset.CoreV1().PersistentVolumes().Update(context.TODO(), existingPV, metav1.UpdateOptions{})
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "更新PV失败: "+err.Error())
		return
	}

	k8sPV := s.convertToK8sPV(updatedPV)
	result.Success(c, k8sPV)
}

// DeletePV 删除PV
func (s *K8sStorageServiceImpl) DeletePV(c *gin.Context, clusterId uint, pvName string) {
	clientset, err := s.getK8sClient(clusterId)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "连接K8s集群失败: "+err.Error())
		return
	}

	err = clientset.CoreV1().PersistentVolumes().Delete(context.TODO(), pvName, metav1.DeleteOptions{})
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "删除PV失败: "+err.Error())
		return
	}

	result.Success(c, "PV删除成功")
}

// GetPVYaml 获取PV的YAML配置
func (s *K8sStorageServiceImpl) GetPVYaml(c *gin.Context, clusterId uint, pvName string) {
	clientset, err := s.getK8sClient(clusterId)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "连接K8s集群失败: "+err.Error())
		return
	}

	pv, err := clientset.CoreV1().PersistentVolumes().Get(context.TODO(), pvName, metav1.GetOptions{})
	if err != nil {
		result.Failed(c, http.StatusNotFound, "PV不存在: "+err.Error())
		return
	}

	// 清理不需要的字段，但保留 status
	pv.ManagedFields = nil

	// 确保 apiVersion 和 kind 字段存在
	if pv.APIVersion == "" {
		pv.APIVersion = "v1"
	}
	if pv.Kind == "" {
		pv.Kind = "PersistentVolume"
	}

	// 清理元数据字段
	pv.ResourceVersion = ""
	pv.UID = ""
	pv.Generation = 0
	pv.CreationTimestamp = metav1.Time{}
	pv.SelfLink = ""

	yamlData, err := yaml.Marshal(pv)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "转换YAML失败: "+err.Error())
		return
	}

	result.Success(c, map[string]interface{}{
		"yaml": string(yamlData),
	})
}

// UpdatePVYaml 通过YAML更新PV
func (s *K8sStorageServiceImpl) UpdatePVYaml(c *gin.Context, clusterId uint, pvName string, yamlData map[string]interface{}) {
	clientset, err := s.getK8sClient(clusterId)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "连接K8s集群失败: "+err.Error())
		return
	}

	// 将map转换为YAML字符串
	yamlBytes, err := json.Marshal(yamlData)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "YAML数据格式错误: "+err.Error())
		return
	}

	var pv corev1.PersistentVolume
	err = yaml.Unmarshal(yamlBytes, &pv)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "YAML解析失败: "+err.Error())
		return
	}

	// 确保名称正确
	pv.Name = pvName

	// 更新PV
	_, err = clientset.CoreV1().PersistentVolumes().Update(context.TODO(), &pv, metav1.UpdateOptions{})
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "更新PV失败: "+err.Error())
		return
	}

	result.Success(c, "PV YAML更新成功")
}

// ===================== StorageClass 管理实现 =====================

// GetStorageClasses 获取存储类列表
func (s *K8sStorageServiceImpl) GetStorageClasses(c *gin.Context, clusterId uint) {
	clientset, err := s.getK8sClient(clusterId)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "连接K8s集群失败: "+err.Error())
		return
	}

	storageClasses, err := clientset.StorageV1().StorageClasses().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "获取存储类列表失败: "+err.Error())
		return
	}

	var scList []model.K8sStorageClass
	for _, sc := range storageClasses.Items {
		k8sSC := s.convertToK8sStorageClass(&sc)
		scList = append(scList, k8sSC)
	}

	response := model.StorageClassListResponse{
		StorageClasses: scList,
		Total:          len(scList),
	}

	result.Success(c, response)
}

// GetStorageClassDetail 获取存储类详情
func (s *K8sStorageServiceImpl) GetStorageClassDetail(c *gin.Context, clusterId uint, storageClassName string) {
	clientset, err := s.getK8sClient(clusterId)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "连接K8s集群失败: "+err.Error())
		return
	}

	sc, err := clientset.StorageV1().StorageClasses().Get(context.TODO(), storageClassName, metav1.GetOptions{})
	if err != nil {
		result.Failed(c, http.StatusNotFound, "存储类不存在: "+err.Error())
		return
	}

	k8sSC := s.convertToK8sStorageClass(sc)

	// 获取相关事件
	events, _ := s.getStorageClassEvents(clientset, storageClassName)

	scDetail := model.StorageClassDetail{
		K8sStorageClass: k8sSC,
		Events:          events,
		Spec:            sc,
	}

	result.Success(c, scDetail)
}

// CreateStorageClass 创建存储类
func (s *K8sStorageServiceImpl) CreateStorageClass(c *gin.Context, clusterId uint, req *model.CreateStorageClassRequest) {
	clientset, err := s.getK8sClient(clusterId)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "连接K8s集群失败: "+err.Error())
		return
	}

	// 构建StorageClass对象
	sc := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name:   req.Name,
			Labels: req.Labels,
		},
		Provisioner:  req.Provisioner,
		Parameters:   req.Parameters,
		MountOptions: req.MountOptions,
	}

	// 设置回收策略
	if req.ReclaimPolicy != "" {
		reclaimPolicy := corev1.PersistentVolumeReclaimPolicy(req.ReclaimPolicy)
		sc.ReclaimPolicy = &reclaimPolicy
	}

	// 设置卷绑定模式
	if req.VolumeBindingMode != "" {
		volumeBindingMode := storagev1.VolumeBindingMode(req.VolumeBindingMode)
		sc.VolumeBindingMode = &volumeBindingMode
	}

	// 设置允许卷扩展
	sc.AllowVolumeExpansion = &req.AllowVolumeExpansion

	// 设置允许的拓扑
	if len(req.AllowedTopologies) > 0 {
		var topologies []corev1.TopologySelectorTerm
		for _, topology := range req.AllowedTopologies {
			term := corev1.TopologySelectorTerm{}
			for _, expr := range topology.MatchLabelExpressions {
				term.MatchLabelExpressions = append(term.MatchLabelExpressions, corev1.TopologySelectorLabelRequirement{
					Key:    expr.Key,
					Values: expr.Values,
				})
			}
			topologies = append(topologies, term)
		}
		sc.AllowedTopologies = topologies
	}

	// 创建StorageClass
	createdSC, err := clientset.StorageV1().StorageClasses().Create(context.TODO(), sc, metav1.CreateOptions{})
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "创建存储类失败: "+err.Error())
		return
	}

	k8sSC := s.convertToK8sStorageClass(createdSC)
	result.Success(c, k8sSC)
}

// UpdateStorageClass 更新存储类
func (s *K8sStorageServiceImpl) UpdateStorageClass(c *gin.Context, clusterId uint, storageClassName string, req *model.UpdateStorageClassRequest) {
	clientset, err := s.getK8sClient(clusterId)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "连接K8s集群失败: "+err.Error())
		return
	}

	// 获取现有StorageClass
	existingSC, err := clientset.StorageV1().StorageClasses().Get(context.TODO(), storageClassName, metav1.GetOptions{})
	if err != nil {
		result.Failed(c, http.StatusNotFound, "存储类不存在: "+err.Error())
		return
	}

	// 更新标签
	if req.Labels != nil {
		existingSC.Labels = req.Labels
	}

	// 更新参数
	if req.Parameters != nil {
		existingSC.Parameters = req.Parameters
	}

	// 更新挂载选项
	if req.MountOptions != nil {
		existingSC.MountOptions = req.MountOptions
	}

	// 更新允许卷扩展
	existingSC.AllowVolumeExpansion = &req.AllowVolumeExpansion

	// 更新StorageClass
	updatedSC, err := clientset.StorageV1().StorageClasses().Update(context.TODO(), existingSC, metav1.UpdateOptions{})
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "更新存储类失败: "+err.Error())
		return
	}

	k8sSC := s.convertToK8sStorageClass(updatedSC)
	result.Success(c, k8sSC)
}

// DeleteStorageClass 删除存储类
func (s *K8sStorageServiceImpl) DeleteStorageClass(c *gin.Context, clusterId uint, storageClassName string) {
	clientset, err := s.getK8sClient(clusterId)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "连接K8s集群失败: "+err.Error())
		return
	}

	err = clientset.StorageV1().StorageClasses().Delete(context.TODO(), storageClassName, metav1.DeleteOptions{})
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "删除存储类失败: "+err.Error())
		return
	}

	result.Success(c, "存储类删除成功")
}

// GetStorageClassYaml 获取存储类的YAML配置
func (s *K8sStorageServiceImpl) GetStorageClassYaml(c *gin.Context, clusterId uint, storageClassName string) {
	clientset, err := s.getK8sClient(clusterId)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "连接K8s集群失败: "+err.Error())
		return
	}

	sc, err := clientset.StorageV1().StorageClasses().Get(context.TODO(), storageClassName, metav1.GetOptions{})
	if err != nil {
		result.Failed(c, http.StatusNotFound, "存储类不存在: "+err.Error())
		return
	}

	// 清理不需要的字段
	sc.ManagedFields = nil

	// 确保 apiVersion 和 kind 字段存在
	if sc.APIVersion == "" {
		sc.APIVersion = "storage.k8s.io/v1"
	}
	if sc.Kind == "" {
		sc.Kind = "StorageClass"
	}

	// 清理元数据字段
	sc.ResourceVersion = ""
	sc.UID = ""
	sc.Generation = 0
	sc.CreationTimestamp = metav1.Time{}
	sc.SelfLink = ""

	yamlData, err := yaml.Marshal(sc)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "转换YAML失败: "+err.Error())
		return
	}

	result.Success(c, map[string]interface{}{
		"yaml": string(yamlData),
	})
}

// UpdateStorageClassYaml 通过YAML更新存储类
func (s *K8sStorageServiceImpl) UpdateStorageClassYaml(c *gin.Context, clusterId uint, storageClassName string, yamlData map[string]interface{}) {
	clientset, err := s.getK8sClient(clusterId)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "连接K8s集群失败: "+err.Error())
		return
	}

	// 将map转换为YAML字符串
	yamlBytes, err := json.Marshal(yamlData)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "YAML数据格式错误: "+err.Error())
		return
	}

	var sc storagev1.StorageClass
	err = yaml.Unmarshal(yamlBytes, &sc)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "YAML解析失败: "+err.Error())
		return
	}

	// 确保名称正确
	sc.Name = storageClassName

	// 更新StorageClass
	_, err = clientset.StorageV1().StorageClasses().Update(context.TODO(), &sc, metav1.UpdateOptions{})
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "更新存储类失败: "+err.Error())
		return
	}

	result.Success(c, "存储类YAML更新成功")
}

// ===================== 辅助方法 =====================

// convertToK8sPVC 转换PVC对象
func (s *K8sStorageServiceImpl) convertToK8sPVC(pvc *corev1.PersistentVolumeClaim) model.K8sPersistentVolumeClaim {
	// 获取容量
	var capacity string
	if storage, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
		capacity = storage.String()
	}

	// 获取访问模式
	var accessModes []string
	for _, mode := range pvc.Spec.AccessModes {
		accessModes = append(accessModes, string(mode))
	}

	// 获取存储类
	var storageClass string
	if pvc.Spec.StorageClassName != nil {
		storageClass = *pvc.Spec.StorageClassName
	}

	// 获取卷模式
	var volumeMode string
	if pvc.Spec.VolumeMode != nil {
		volumeMode = string(*pvc.Spec.VolumeMode)
	}

	return model.K8sPersistentVolumeClaim{
		Name:         pvc.Name,
		Namespace:    pvc.Namespace,
		Labels:       pvc.Labels,
		Capacity:     capacity,
		AccessModes:  accessModes,
		Status:       string(pvc.Status.Phase),
		StorageClass: storageClass,
		VolumeMode:   volumeMode,
		VolumeName:   pvc.Spec.VolumeName,
		CreatedAt:    pvc.CreationTimestamp.Format(time.RFC3339),
	}
}

// convertToK8sPV 转换PV对象
func (s *K8sStorageServiceImpl) convertToK8sPV(pv *corev1.PersistentVolume) model.K8sPersistentVolume {
	// 获取容量
	var capacity string
	if storage, ok := pv.Spec.Capacity[corev1.ResourceStorage]; ok {
		capacity = storage.String()
	}

	// 获取访问模式
	var accessModes []string
	for _, mode := range pv.Spec.AccessModes {
		accessModes = append(accessModes, string(mode))
	}

	// 获取卷模式
	var volumeMode string
	if pv.Spec.VolumeMode != nil {
		volumeMode = string(*pv.Spec.VolumeMode)
	}

	// 构建声明引用
	var claimRef *model.PVClaimRef
	if pv.Spec.ClaimRef != nil {
		claimRef = &model.PVClaimRef{
			Kind:       pv.Spec.ClaimRef.Kind,
			Namespace:  pv.Spec.ClaimRef.Namespace,
			Name:       pv.Spec.ClaimRef.Name,
			UID:        string(pv.Spec.ClaimRef.UID),
			APIVersion: pv.Spec.ClaimRef.APIVersion,
		}
	}

	// 构建存储源
	pvSource := model.PVSource{}
	if pv.Spec.PersistentVolumeSource.HostPath != nil {
		pvSource.HostPath = &model.PVHostPathVolumeSource{
			Path: pv.Spec.PersistentVolumeSource.HostPath.Path,
			Type: string(*pv.Spec.PersistentVolumeSource.HostPath.Type),
		}
	}
	if pv.Spec.PersistentVolumeSource.NFS != nil {
		pvSource.NFS = &model.PVNFSVolumeSource{
			Server:   pv.Spec.PersistentVolumeSource.NFS.Server,
			Path:     pv.Spec.PersistentVolumeSource.NFS.Path,
			ReadOnly: pv.Spec.PersistentVolumeSource.NFS.ReadOnly,
		}
	}

	return model.K8sPersistentVolume{
		Name:                   pv.Name,
		Labels:                 pv.Labels,
		Capacity:               capacity,
		AccessModes:            accessModes,
		ReclaimPolicy:          string(pv.Spec.PersistentVolumeReclaimPolicy),
		Status:                 string(pv.Status.Phase),
		StorageClass:           pv.Spec.StorageClassName,
		VolumeMode:             volumeMode,
		ClaimRef:               claimRef,
		PersistentVolumeSource: pvSource,
		MountOptions:           pv.Spec.MountOptions,
		CreatedAt:              pv.CreationTimestamp.Format(time.RFC3339),
	}
}

// convertToK8sStorageClass 转换StorageClass对象
func (s *K8sStorageServiceImpl) convertToK8sStorageClass(sc *storagev1.StorageClass) model.K8sStorageClass {
	// 获取回收策略
	var reclaimPolicy string
	if sc.ReclaimPolicy != nil {
		reclaimPolicy = string(*sc.ReclaimPolicy)
	}

	// 获取卷绑定模式
	var volumeBindingMode string
	if sc.VolumeBindingMode != nil {
		volumeBindingMode = string(*sc.VolumeBindingMode)
	}

	// 获取允许卷扩展
	var allowVolumeExpansion bool
	if sc.AllowVolumeExpansion != nil {
		allowVolumeExpansion = *sc.AllowVolumeExpansion
	}

	// 构建允许的拓扑
	var allowedTopologies []model.StorageClassTopology
	for _, topology := range sc.AllowedTopologies {
		scTopology := model.StorageClassTopology{}
		for _, expr := range topology.MatchLabelExpressions {
			scTopology.MatchLabelExpressions = append(scTopology.MatchLabelExpressions, model.StorageClassTopologyExp{
				Key:    expr.Key,
				Values: expr.Values,
			})
		}
		allowedTopologies = append(allowedTopologies, scTopology)
	}

	return model.K8sStorageClass{
		Name:                 sc.Name,
		Labels:               sc.Labels,
		Provisioner:          sc.Provisioner,
		Parameters:           sc.Parameters,
		ReclaimPolicy:        reclaimPolicy,
		VolumeBindingMode:    volumeBindingMode,
		AllowVolumeExpansion: allowVolumeExpansion,
		MountOptions:         sc.MountOptions,
		AllowedTopologies:    allowedTopologies,
		CreatedAt:            sc.CreationTimestamp.Format(time.RFC3339),
	}
}

// getPVCEvents 获取PVC相关事件
func (s *K8sStorageServiceImpl) getPVCEvents(clientset *kubernetes.Clientset, namespaceName string, pvcName string) ([]model.K8sEvent, error) {
	events, err := clientset.CoreV1().Events(namespaceName).List(context.TODO(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=PersistentVolumeClaim", pvcName),
	})
	if err != nil {
		return nil, err
	}

	var k8sEvents []model.K8sEvent
	for _, event := range events.Items {
		k8sEvent := model.K8sEvent{
			Type:      event.Type,
			Reason:    event.Reason,
			Message:   event.Message,
			Source:    event.Source.Component,
			Count:     event.Count,
			FirstTime: event.FirstTimestamp.Format(time.RFC3339),
			LastTime:  event.LastTimestamp.Format(time.RFC3339),
		}
		k8sEvents = append(k8sEvents, k8sEvent)
	}

	return k8sEvents, nil
}

// getPVEvents 获取PV相关事件
func (s *K8sStorageServiceImpl) getPVEvents(clientset *kubernetes.Clientset, pvName string) ([]model.K8sEvent, error) {
	events, err := clientset.CoreV1().Events("").List(context.TODO(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=PersistentVolume", pvName),
	})
	if err != nil {
		return nil, err
	}

	var k8sEvents []model.K8sEvent
	for _, event := range events.Items {
		k8sEvent := model.K8sEvent{
			Type:      event.Type,
			Reason:    event.Reason,
			Message:   event.Message,
			Source:    event.Source.Component,
			Count:     event.Count,
			FirstTime: event.FirstTimestamp.Format(time.RFC3339),
			LastTime:  event.LastTimestamp.Format(time.RFC3339),
		}
		k8sEvents = append(k8sEvents, k8sEvent)
	}

	return k8sEvents, nil
}

// getStorageClassEvents 获取StorageClass相关事件
func (s *K8sStorageServiceImpl) getStorageClassEvents(clientset *kubernetes.Clientset, storageClassName string) ([]model.K8sEvent, error) {
	events, err := clientset.CoreV1().Events("").List(context.TODO(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=StorageClass", storageClassName),
	})
	if err != nil {
		return nil, err
	}

	var k8sEvents []model.K8sEvent
	for _, event := range events.Items {
		k8sEvent := model.K8sEvent{
			Type:      event.Type,
			Reason:    event.Reason,
			Message:   event.Message,
			Source:    event.Source.Component,
			Count:     event.Count,
			FirstTime: event.FirstTimestamp.Format(time.RFC3339),
			LastTime:  event.LastTimestamp.Format(time.RFC3339),
		}
		k8sEvents = append(k8sEvents, k8sEvent)
	}

	return k8sEvents, nil
}

// ===================== PVC 高阶功能实现 =====================

// ExpandPVC PVC扩容
func (s *K8sStorageServiceImpl) ExpandPVC(c *gin.Context, clusterId uint, namespaceName string, pvcName string, req *model.ExpandPVCRequest) {
	clientset, err := s.getK8sClient(clusterId)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "连接K8s集群失败: "+err.Error())
		return
	}

	// 获取现有PVC
	pvc, err := clientset.CoreV1().PersistentVolumeClaims(namespaceName).Get(context.TODO(), pvcName, metav1.GetOptions{})
	if err != nil {
		result.Failed(c, http.StatusNotFound, "PVC不存在: "+err.Error())
		return
	}

	// 解析新容量
	newSize, err := resource.ParseQuantity(req.NewSize)
	if err != nil {
		result.Failed(c, http.StatusBadRequest, "无效的容量格式: "+err.Error())
		return
	}

	// 检查新容量是否大于当前容量
	currentSize := pvc.Spec.Resources.Requests[corev1.ResourceStorage]
	if newSize.Cmp(currentSize) <= 0 {
		result.Failed(c, http.StatusBadRequest, "新容量必须大于当前容量 ("+currentSize.String()+")")
		return
	}

	// 更新PVC的存储请求
	pvc.Spec.Resources.Requests[corev1.ResourceStorage] = newSize

	// 应用更新
	updatedPVC, err := clientset.CoreV1().PersistentVolumeClaims(namespaceName).Update(context.TODO(), pvc, metav1.UpdateOptions{})
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "PVC扩容失败: "+err.Error())
		return
	}

	result.Success(c, gin.H{
		"message":   "PVC扩容请求已提交",
		"pvcName":   updatedPVC.Name,
		"namespace": updatedPVC.Namespace,
		"newSize":   req.NewSize,
		"status":    string(updatedPVC.Status.Phase),
		"tip":       "扩容正在后台进行，具体耗时取决于存储后端。请刷新页面查看最新状态。",
	})
}

// ClonePVC PVC克隆（用于数据备份/迁移）
func (s *K8sStorageServiceImpl) ClonePVC(c *gin.Context, clusterId uint, namespaceName string, pvcName string, req *model.PVCCloneRequest) {
	clientset, err := s.getK8sClient(clusterId)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "连接K8s集群失败: "+err.Error())
		return
	}

	// 获取源PVC
	sourcePVC, err := clientset.CoreV1().PersistentVolumeClaims(namespaceName).Get(context.TODO(), pvcName, metav1.GetOptions{})
	if err != nil {
		result.Failed(c, http.StatusNotFound, "源PVC不存在: "+err.Error())
		return
	}

	// 确定目标命名空间
	targetNS := namespaceName
	if req.TargetNamespace != "" {
		targetNS = req.TargetNamespace
	}

	// 检查目标PVC是否已存在
	_, err = clientset.CoreV1().PersistentVolumeClaims(targetNS).Get(context.TODO(), req.TargetName, metav1.GetOptions{})
	if err == nil {
		result.Failed(c, http.StatusConflict, "目标PVC '"+req.TargetName+"' 已存在")
		return
	}

	// 构建目标PVC（克隆源PVC的配置）
	cloneSize := sourcePVC.Spec.Resources.Requests[corev1.ResourceStorage]
	if req.NewSize != "" {
		parsedSize, err := resource.ParseQuantity(req.NewSize)
		if err != nil {
			result.Failed(c, http.StatusBadRequest, "无效的目标容量格式: "+err.Error())
			return
		}
		cloneSize = parsedSize
	}

	targetPVC := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.TargetName,
			Namespace: targetNS,
			Labels:    sourcePVC.Labels,
			Annotations: map[string]string{
				"dodevops.io/cloned-from":      pvcName,
				"dodevops.io/cloned-namespace": namespaceName,
				"dodevops.io/clone-timestamp":  time.Now().Format(time.RFC3339),
			},
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: sourcePVC.Spec.AccessModes,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: cloneSize,
				},
			},
			StorageClassName: sourcePVC.Spec.StorageClassName,
			VolumeMode:       sourcePVC.Spec.VolumeMode,
		},
	}

	// 创建目标PVC
	createdPVC, err := clientset.CoreV1().PersistentVolumeClaims(targetNS).Create(context.TODO(), targetPVC, metav1.CreateOptions{})
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "创建克隆PVC失败: "+err.Error())
		return
	}

	result.Success(c, gin.H{
		"message":         "PVC克隆请求已提交（仅创建了相同规格的PVC声明）",
		"sourcePvc":       pvcName,
		"sourceNamespace": namespaceName,
		"targetPvc":       createdPVC.Name,
		"targetNamespace": createdPVC.Namespace,
		"size":            cloneSize.String(),
		"status":          string(createdPVC.Status.Phase),
		"backupInfo": model.PVCBackupInfo{
			SourcePVCName:   pvcName,
			SourceNamespace: namespaceName,
			TargetPVCName:   req.TargetName,
			TargetNamespace: targetNS,
			Size:            cloneSize.String(),
			StorageClass:    getStorageClassName(sourcePVC.Spec.StorageClassName),
			CreatedAt:       time.Now().Format(time.RFC3339),
			Status:          string(createdPVC.Status.Phase),
		},
	})
}

// ===================== StorageClass 高阶功能实现 =====================

// SetDefaultStorageClass 设置默认存储类
func (s *K8sStorageServiceImpl) SetDefaultStorageClass(c *gin.Context, clusterId uint, storageClassName string, req *model.SetDefaultSCRequest) {
	clientset, err := s.getK8sClient(clusterId)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "连接K8s集群失败: "+err.Error())
		return
	}

	// 如果要设为默认，先取消所有现有默认
	if req.IsDefault {
		allSCs, err := clientset.StorageV1().StorageClasses().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			result.Failed(c, http.StatusInternalServerError, "获取存储类列表失败: "+err.Error())
			return
		}
		for _, sc := range allSCs.Items {
			if sc.Annotations != nil {
				if _, hasDefault := sc.Annotations["storageclass.kubernetes.io/is-default-class"]; hasDefault {
					// 取消现有默认
					scCopy := sc.DeepCopy()
					delete(scCopy.Annotations, "storageclass.kubernetes.io/is-default-class")
					_, err := clientset.StorageV1().StorageClasses().Update(context.TODO(), scCopy, metav1.UpdateOptions{})
					if err != nil {
						result.Failed(c, http.StatusInternalServerError, "取消现有默认存储类失败: "+err.Error())
						return
					}
				}
			}
		}
	}

	// 获取目标存储类
	sc, err := clientset.StorageV1().StorageClasses().Get(context.TODO(), storageClassName, metav1.GetOptions{})
	if err != nil {
		result.Failed(c, http.StatusNotFound, "存储类不存在: "+err.Error())
		return
	}

	// 设置或取消默认标记
	if sc.Annotations == nil {
		sc.Annotations = make(map[string]string)
	}

	if req.IsDefault {
		sc.Annotations["storageclass.kubernetes.io/is-default-class"] = "true"
	} else {
		delete(sc.Annotations, "storageclass.kubernetes.io/is-default-class")
	}

	_, err = clientset.StorageV1().StorageClasses().Update(context.TODO(), sc, metav1.UpdateOptions{})
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "更新存储类失败: "+err.Error())
		return
	}

	action := "取消默认"
	if req.IsDefault {
		action = "设为默认"
	}

	result.Success(c, gin.H{
		"message":          "存储类'" + storageClassName + "'已" + action,
		"storageClassName": storageClassName,
		"isDefault":        req.IsDefault,
	})
}

// GetStorageStats 获取存储统计信息
func (s *K8sStorageServiceImpl) GetStorageStats(c *gin.Context, clusterId uint) {
	clientset, err := s.getK8sClient(clusterId)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "连接K8s集群失败: "+err.Error())
		return
	}

	// 获取PV列表
	pvs, err := clientset.CoreV1().PersistentVolumes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "获取PV列表失败: "+err.Error())
		return
	}

	// 获取PVC列表
	pvcs, err := clientset.CoreV1().PersistentVolumeClaims("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "获取PVC列表失败: "+err.Error())
		return
	}

	// 获取StorageClass列表
	scs, err := clientset.StorageV1().StorageClasses().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "获取存储类列表失败: "+err.Error())
		return
	}

	// 计算存储统计
	var totalStorage resource.Quantity
	var usedStorage resource.Quantity
	var defaultSC string
	var scNames []string

	for _, pv := range pvs.Items {
		if capacity, ok := pv.Spec.Capacity[corev1.ResourceStorage]; ok {
			totalStorage.Add(capacity)
		}
		if pv.Status.Phase == corev1.VolumeBound {
			if capacity, ok := pv.Spec.Capacity[corev1.ResourceStorage]; ok {
				usedStorage.Add(capacity)
			}
		}
	}

	for _, sc := range scs.Items {
		scNames = append(scNames, sc.Name)
		if sc.Annotations != nil {
			if val, ok := sc.Annotations["storageclass.kubernetes.io/is-default-class"]; ok && val == "true" {
				defaultSC = sc.Name
			}
		}
	}

	// 计算可用存储
	availableStorage := totalStorage.DeepCopy()
	availableStorage.Sub(usedStorage)
	if availableStorage.Sign() < 0 {
		availableStorage = resource.MustParse("0")
	}

	stats := model.StorageStats{
		TotalPVCs:           len(pvcs.Items),
		TotalPVs:            len(pvs.Items),
		TotalStorage:        totalStorage.String(),
		UsedStorage:         usedStorage.String(),
		AvailableStorage:    availableStorage.String(),
		StorageClasses:      scNames,
		DefaultStorageClass: defaultSC,
	}

	result.Success(c, stats)
}

// getStorageClassName 安全获取存储类名称
func getStorageClassName(scName *string) string {
	if scName == nil {
		return ""
	}
	return *scName
}

// ===================== 存储迁移实现 =====================

// MigratePVC 卷到卷数据迁移
// 通过创建 rsync 迁移 Job，将源 PVC 的数据复制到目标 PVC
func (s *K8sStorageServiceImpl) MigratePVC(c *gin.Context, clusterId uint, req *model.MigratePVCRequest) {
	clientset, err := s.getK8sClient(clusterId)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "连接K8s集群失败: "+err.Error())
		return
	}

	// 确定目标命名空间
	targetNS := req.TargetNS
	if targetNS == "" {
		targetNS = req.SourceNS
	}

	// 校验源 PVC 存在
	if _, err := clientset.CoreV1().PersistentVolumeClaims(req.SourceNS).Get(context.TODO(), req.SourcePVC, metav1.GetOptions{}); err != nil {
		result.Failed(c, http.StatusNotFound, "源PVC不存在: "+err.Error())
		return
	}

	// 检查目标 PVC 是否存在，不存在则按需创建
	if _, err := clientset.CoreV1().PersistentVolumeClaims(targetNS).Get(context.TODO(), req.TargetPVC, metav1.GetOptions{}); err != nil {
		if !req.CreateIfAbsent {
			result.Failed(c, http.StatusNotFound, "目标PVC不存在，请先创建或开启自动创建")
			return
		}
		// 自动创建目标 PVC
		size := req.TargetSize
		if size == "" {
			sourcePVC, _ := clientset.CoreV1().PersistentVolumeClaims(req.SourceNS).Get(context.TODO(), req.SourcePVC, metav1.GetOptions{})
			storageSize := sourcePVC.Spec.Resources.Requests[corev1.ResourceStorage]
			size = storageSize.String()
		}
		newPVC := &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      req.TargetPVC,
				Namespace: targetNS,
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse(size),
					},
				},
			},
		}
		if req.TargetSC != "" {
			newPVC.Spec.StorageClassName = &req.TargetSC
		}
		if _, err := clientset.CoreV1().PersistentVolumeClaims(targetNS).Create(context.TODO(), newPVC, metav1.CreateOptions{}); err != nil {
			result.Failed(c, http.StatusInternalServerError, "创建目标PVC失败: "+err.Error())
			return
		}
	}

	// 创建 rsync 数据迁移 Job
	jobName := fmt.Sprintf("pvc-migrate-%s-%d", strings.ReplaceAll(req.SourcePVC, ".", "-"), time.Now().Unix())
	migrateJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: targetNS,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "rsync",
							Image: "alpine:3.18",
							Command: []string{
								"/bin/sh", "-c",
								"apk add --no-cache rsync >/dev/null 2>&1 && rsync -av --delete /source/ /target/ && echo 'MIGRATION_COMPLETE'",
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "source", MountPath: "/source"},
								{Name: "target", MountPath: "/target"},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "source",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: req.SourcePVC},
							},
						},
						{
							Name: "target",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: req.TargetPVC},
							},
						},
					},
				},
			},
		},
	}

	createdJob, err := clientset.BatchV1().Jobs(targetNS).Create(context.TODO(), migrateJob, metav1.CreateOptions{})
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "创建迁移Job失败: "+err.Error())
		return
	}

	result.Success(c, gin.H{
		"message":         "卷数据迁移任务已启动",
		"jobName":         createdJob.Name,
		"jobNamespace":    createdJob.Namespace,
		"sourcePvc":       req.SourcePVC,
		"sourceNamespace": req.SourceNS,
		"targetPvc":       req.TargetPVC,
		"targetNamespace": targetNS,
		"tip":             "迁移Job使用rsync将源卷数据复制到目标卷，完成后可查看Job日志确认数据完整性，再执行一键换卷。",
	})
}

// SwitchWorkloadPVC 一键换卷：将工作负载（Deployment/StatefulSet）中引用的旧PVC替换为新PVC
func (s *K8sStorageServiceImpl) SwitchWorkloadPVC(c *gin.Context, clusterId uint, namespaceName string, req *model.SwitchWorkloadPVCRequest) {
	clientset, err := s.getK8sClient(clusterId)
	if err != nil {
		result.Failed(c, http.StatusInternalServerError, "连接K8s集群失败: "+err.Error())
		return
	}

	// 校验新PVC存在
	if _, err := clientset.CoreV1().PersistentVolumeClaims(namespaceName).Get(context.TODO(), req.NewPVCName, metav1.GetOptions{}); err != nil {
		result.Failed(c, http.StatusNotFound, "新PVC不存在: "+err.Error())
		return
	}

	replacedCount := 0

	switch strings.ToLower(req.WorkloadType) {
	case "deployment", "deployments":
		deployment, err := clientset.AppsV1().Deployments(namespaceName).Get(context.TODO(), req.WorkloadName, metav1.GetOptions{})
		if err != nil {
			result.Failed(c, http.StatusNotFound, "Deployment不存在: "+err.Error())
			return
		}
		for i := range deployment.Spec.Template.Spec.Volumes {
			vol := &deployment.Spec.Template.Spec.Volumes[i]
			if vol.PersistentVolumeClaim != nil {
				if req.OldPVCName == "" || vol.PersistentVolumeClaim.ClaimName == req.OldPVCName {
					vol.PersistentVolumeClaim.ClaimName = req.NewPVCName
					replacedCount++
				}
			}
		}
		if replacedCount == 0 {
			result.Failed(c, http.StatusBadRequest, "Deployment中未找到匹配的PVC卷")
			return
		}
		if _, err := clientset.AppsV1().Deployments(namespaceName).Update(context.TODO(), deployment, metav1.UpdateOptions{}); err != nil {
			result.Failed(c, http.StatusInternalServerError, "更新Deployment失败: "+err.Error())
			return
		}

	case "statefulset", "statefulsets":
		statefulSet, err := clientset.AppsV1().StatefulSets(namespaceName).Get(context.TODO(), req.WorkloadName, metav1.GetOptions{})
		if err != nil {
			result.Failed(c, http.StatusNotFound, "StatefulSet不存在: "+err.Error())
			return
		}
		for i := range statefulSet.Spec.Template.Spec.Volumes {
			vol := &statefulSet.Spec.Template.Spec.Volumes[i]
			if vol.PersistentVolumeClaim != nil {
				if req.OldPVCName == "" || vol.PersistentVolumeClaim.ClaimName == req.OldPVCName {
					vol.PersistentVolumeClaim.ClaimName = req.NewPVCName
					replacedCount++
				}
			}
		}
		if replacedCount == 0 {
			result.Failed(c, http.StatusBadRequest, "StatefulSet中未找到匹配的PVC卷")
			return
		}
		if _, err := clientset.AppsV1().StatefulSets(namespaceName).Update(context.TODO(), statefulSet, metav1.UpdateOptions{}); err != nil {
			result.Failed(c, http.StatusInternalServerError, "更新StatefulSet失败: "+err.Error())
			return
		}

	default:
		result.Failed(c, http.StatusBadRequest, "不支持的工作负载类型: "+req.WorkloadType)
		return
	}

	result.Success(c, gin.H{
		"message":       "一键换卷成功",
		"workloadType":  req.WorkloadType,
		"workloadName":  req.WorkloadName,
		"oldPvcName":    req.OldPVCName,
		"newPvcName":    req.NewPVCName,
		"replacedCount": replacedCount,
		"tip":           "工作负载将滚动重启以挂载新卷，请确认数据已迁移完成。",
	})
}
