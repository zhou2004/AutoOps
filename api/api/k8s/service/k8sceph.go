package service

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"dodevops-api/api/k8s/dao"
	"dodevops-api/api/k8s/model"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

//go:embed ceph/worker.py
var cephWorker string

type K8sCephService struct{ storage *K8sStorageServiceImpl }

func NewK8sCephService(db *gorm.DB) *K8sCephService {
	return &K8sCephService{storage: &K8sStorageServiceImpl{clusterDao: dao.NewKubeClusterDao(db)}}
}

// Limited output prevents a large cluster listing or command from exhausting API memory.
type cephOutput struct {
	bytes.Buffer
	remaining int
}

func (b *cephOutput) Write(p []byte) (int, error) {
	n := len(p)
	if len(p) > b.remaining {
		p = p[:b.remaining]
	}
	b.Buffer.Write(p)
	b.remaining -= len(p)
	return n, nil
}

func cephExec(ctx context.Context, client kubernetes.Interface, config *rest.Config, pod *corev1.Pod, command []string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	restClient := client.CoreV1().RESTClient()
	if restClient == nil {
		return "", fmt.Errorf("Kubernetes REST 客户端不可用")
	}
	req := restClient.Post().Resource("pods").Namespace(cephNamespace).Name(pod.Name).SubResource("exec").VersionedParams(&corev1.PodExecOptions{Container: pod.Spec.Containers[0].Name, Command: command, Stdout: true, Stderr: true}, scheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return "", err
	}
	out, errOut := &cephOutput{remaining: 2 * 1024 * 1024}, &cephOutput{remaining: 8192}
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{Stdout: out, Stderr: errOut})
	if err != nil {
		return "", fmt.Errorf("执行失败: %w: %s", err, errOut.String())
	}
	if out.remaining == 0 {
		return "", fmt.Errorf("结果超过 2 MiB，请缩小查询范围")
	}
	return out.String(), nil
}

func readyCephPod(ctx context.Context, client kubernetes.Interface, selector string) (*corev1.Pod, error) {
	pods, err := client.CoreV1().Pods(cephNamespace).List(ctx, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, err
	}
	for i := range pods.Items {
		p := &pods.Items[i]
		if p.DeletionTimestamp != nil || p.Status.Phase != corev1.PodRunning || len(p.Spec.Containers) == 0 {
			continue
		}
		for _, cond := range p.Status.Conditions {
			if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
				return p, nil
			}
		}
	}
	return nil, fmt.Errorf("工具 Pod 尚未就绪 (%s)，请检查部署、镜像和 PVC 挂载", selector)
}

func (s *K8sCephService) client(id uint) (*kubernetes.Clientset, *rest.Config, error) {
	return s.storage.getK8sClientAndConfig(id)
}

func (s *K8sCephService) Inspect(ctx context.Context, id uint, kind string, p map[string]string) (any, error) {
	cmd, err := cephInspectCommand(kind, p)
	if err != nil {
		return nil, err
	}
	client, config, err := s.client(id)
	if err != nil {
		return nil, err
	}
	pod, err := readyCephPod(ctx, client, "app=rook-ceph-tools")
	if err != nil {
		return nil, err
	}
	out, err := cephExec(ctx, client, config, pod, cmd)
	if err != nil {
		return nil, err
	}
	var value any
	if err = json.Unmarshal([]byte(out), &value); err != nil {
		return nil, fmt.Errorf("Ceph 返回非 JSON 结果: %w", err)
	}
	return redactCephSecrets(value), nil
}

func cephInspectCommand(kind string, p map[string]string) ([]string, error) {
	for k, v := range p {
		if v != "" && !cephNamePattern.MatchString(v) {
			return nil, fmt.Errorf("%s格式无效", k)
		}
	}
	need := func(keys ...string) error {
		for _, k := range keys {
			if p[k] == "" {
				return fmt.Errorf("缺少参数 %s", k)
			}
		}
		return nil
	}
	var cmd []string
	switch kind {
	case "osd-perf":
		cmd = []string{"ceph", "osd", "perf", "-f", "json"}
	case "osd-tree":
		cmd = []string{"ceph", "osd", "tree", "-f", "json"}
	case "osd-flags":
		cmd = []string{"ceph", "osd", "dump", "-f", "json"}
	case "pg-stuck":
		cmd = []string{"ceph", "pg", "dump_stuck", "-f", "json"}
	case "balancer":
		cmd = []string{"ceph", "balancer", "status", "-f", "json"}
	case "pool-autoscale":
		cmd = []string{"ceph", "osd", "pool", "autoscale-status", "-f", "json"}
	case "rbd-status", "rbd-children", "rbd-trash", "rbd-mirror":
		if err := need("pool"); err != nil {
			return nil, err
		}
		switch kind {
		case "rbd-trash":
			cmd = []string{"rbd", "trash", "ls", "--pool", p["pool"], "--format", "json"}
		case "rbd-mirror":
			cmd = []string{"rbd", "mirror", "pool", "status", p["pool"], "--format", "json"}
		default:
			if err := need("image"); err != nil {
				return nil, err
			}
			spec := p["pool"] + "/" + p["image"]
			if kind == "rbd-children" {
				if err := need("snapshot"); err != nil {
					return nil, err
				}
				spec += "@" + p["snapshot"]
				cmd = []string{"rbd", "children", spec, "--format", "json"}
			} else {
				cmd = []string{"rbd", "status", spec, "--format", "json"}
			}
		}
	case "cephfs-groups", "cephfs-subvolumes", "cephfs-info", "cephfs-snapshots", "cephfs-clone-status":
		if err := need("filesystem"); err != nil {
			return nil, err
		}
		switch kind {
		case "cephfs-groups":
			cmd = []string{"ceph", "fs", "subvolumegroup", "ls", p["filesystem"]}
		case "cephfs-subvolumes":
			cmd = []string{"ceph", "fs", "subvolume", "ls", p["filesystem"]}
		default:
			if err := need("subvolume"); err != nil {
				return nil, err
			}
			switch kind {
			case "cephfs-info":
				cmd = []string{"ceph", "fs", "subvolume", "info", p["filesystem"], p["subvolume"]}
			case "cephfs-snapshots":
				cmd = []string{"ceph", "fs", "subvolume", "snapshot", "ls", p["filesystem"], p["subvolume"]}
			case "cephfs-clone-status":
				cmd = []string{"ceph", "fs", "clone", "status", p["filesystem"], p["subvolume"]}
			}
		}
		if kind != "cephfs-groups" && p["group"] != "" {
			cmd = append(cmd, "--group_name", p["group"])
		}
		cmd = append(cmd, "-f", "json")
	case "rgw-users":
		cmd = []string{"radosgw-admin", "user", "list", "--format", "json"}
	case "rgw-user":
		if err := need("uid"); err != nil {
			return nil, err
		}
		cmd = []string{"radosgw-admin", "user", "info", "--uid", p["uid"], "--format", "json"}
	case "rgw-buckets":
		cmd = []string{"radosgw-admin", "bucket", "list", "--format", "json"}
		if p["uid"] != "" {
			cmd = append(cmd, "--uid", p["uid"])
		}
	case "rgw-bucket-stats":
		if err := need("bucket"); err != nil {
			return nil, err
		}
		cmd = []string{"radosgw-admin", "bucket", "stats", "--bucket", p["bucket"], "--format", "json"}
	default:
		return nil, fmt.Errorf("不支持的诊断类型")
	}
	return cmd, nil
}

func redactCephSecrets(value any) any {
	switch v := value.(type) {
	case map[string]any:
		for k, item := range v {
			switch k {
			case "keys", "swift_keys", "secret_key", "access_key", "temp_url_keys":
				delete(v, k)
			default:
				v[k] = redactCephSecrets(item)
			}
		}
	case []any:
		for i := range v {
			v[i] = redactCephSecrets(v[i])
		}
	}
	return value
}

// The mon secret and endpoints are mounted by reference. No credentials enter task JSON or logs.
const cephConfigInit = `set -eu
umask 077
username=client.admin
if [ -s /run/ceph-secret/ceph-username ]; then username=$(cat /run/ceph-secret/ceph-username); fi
endpoints=$(sed 's/[a-zA-Z0-9_-]*=//g' /run/ceph-mon/data)
printf '[global]\nmon_host = %s\n[client]\nkeyring = /etc/ceph/keyring\n' "$endpoints" > /etc/ceph/ceph.conf
printf '[%s]\nkey = %s\n' "$username" "$(cat /run/ceph-secret/ceph-secret)" > /etc/ceph/keyring
printf '%s' "$username" > /etc/ceph/username
`

// Ceph uses client.admin by default. ROOK_CEPH_USERNAME may be a different client.
const cephWorkerStart = `import os,sys
os.environ['CEPH_ARGS']='--name '+open('/etc/ceph/username').read().strip()
code=sys.argv.pop(1)
exec(compile(code,'<ceph-maintenance>','exec'))
`

func newCephJob(image, actor string, req model.CephTaskRequest, pvc string) (*batchv1.Job, error) {
	if _, err := validateCephTask(&req); err != nil {
		return nil, err
	}
	body, _ := json.Marshal(req)
	zero := int32(0)
	one := int32(1)
	no := false
	labels := map[string]string{cephManaged: "task"}
	annotations := map[string]string{"autoops.io/request": string(body), "autoops.io/actor": actor}
	job := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{GenerateName: "autoops-ceph-", Namespace: cephNamespace, Labels: labels, Annotations: annotations}, Spec: batchv1.JobSpec{BackoffLimit: &zero, Completions: &one, Parallelism: &one, ActiveDeadlineSeconds: &req.TimeoutSeconds}}
	configMount := corev1.VolumeMount{Name: "ceph-config", MountPath: "/etc/ceph"}
	job.Spec.Template = corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: labels}, Spec: corev1.PodSpec{RestartPolicy: corev1.RestartPolicyNever, AutomountServiceAccountToken: &no,
		Volumes:        []corev1.Volume{{Name: "ceph-config", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}, {Name: "ceph-secret", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: "rook-ceph-mon"}}}, {Name: "ceph-mon", VolumeSource: corev1.VolumeSource{ConfigMap: &corev1.ConfigMapVolumeSource{LocalObjectReference: corev1.LocalObjectReference{Name: "rook-ceph-mon-endpoints"}}}}},
		InitContainers: []corev1.Container{{Name: "ceph-config", Image: image, Command: []string{"bash", "-c", cephConfigInit}, VolumeMounts: []corev1.VolumeMount{configMount, {Name: "ceph-secret", MountPath: "/run/ceph-secret", ReadOnly: true}, {Name: "ceph-mon", MountPath: "/run/ceph-mon", ReadOnly: true}}}},
		Containers:     []corev1.Container{{Name: "maintenance", Image: image, Command: []string{"python3", "-u", "-c", cephWorkerStart, cephWorker, string(body)}, Env: []corev1.EnvVar{{Name: "TASK_ID", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.labels['job-name']"}}}}, VolumeMounts: []corev1.VolumeMount{configMount}, Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m"), corev1.ResourceMemory: resource.MustParse("128Mi")}, Limits: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("1Gi")}}, SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: &no}}},
	}}
	if pvc != "" {
		job.Spec.Template.Spec.Volumes = append(job.Spec.Template.Spec.Volumes, corev1.Volume{Name: "backup", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: pvc}}})
		job.Spec.Template.Spec.Containers[0].VolumeMounts = append(job.Spec.Template.Spec.Containers[0].VolumeMounts, corev1.VolumeMount{Name: "backup", MountPath: "/autoops-backups"})
	}
	return job, nil
}

func (s *K8sCephService) prepareJob(ctx context.Context, id uint, actor string, req model.CephTaskRequest) (*kubernetes.Clientset, *batchv1.Job, error) {
	if _, err := validateCephTask(&req); err != nil {
		return nil, nil, err
	}
	client, _, err := s.client(id)
	if err != nil {
		return nil, nil, err
	}
	pod, err := readyCephPod(ctx, client, "app=rook-ceph-tools")
	if err != nil {
		return nil, nil, err
	}
	pvc := ""
	if repo := req.Params["repository"]; repo != "" {
		cm, e := client.CoreV1().ConfigMaps(cephNamespace).Get(ctx, "autoops-ceph-repo-"+repo, metav1.GetOptions{})
		if e != nil {
			return nil, nil, e
		}
		if cm.Labels[cephManaged] != "repository" {
			return nil, nil, fmt.Errorf("不是 AutoOps 备份仓库")
		}
		pvc = cm.Data["pvc"]
	}
	job, err := newCephJob(pod.Spec.Containers[0].Image, actor, req, pvc)
	if err == nil {
		job.Spec.Template.Spec.ImagePullSecrets = pod.Spec.ImagePullSecrets
		job.Spec.Template.Spec.Tolerations = pod.Spec.Tolerations
	}
	return client, job, err
}

func (s *K8sCephService) CreateTask(ctx context.Context, id uint, actor string, req model.CephTaskRequest) (any, error) {
	client, job, err := s.prepareJob(ctx, id, actor, req)
	if err != nil {
		return nil, err
	}
	created, err := client.BatchV1().Jobs(cephNamespace).Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return cephJobView(created), nil
}

func cephJobView(job *batchv1.Job) map[string]any {
	state := "Pending"
	message := "等待调度"
	if job.Status.Active > 0 {
		state = "Running"
		message = "执行中，详情查看日志"
	}
	for _, c := range job.Status.Conditions {
		if c.Status == corev1.ConditionTrue {
			switch c.Type {
			case batchv1.JobComplete:
				state = "Succeeded"
			case batchv1.JobFailed:
				state = "Failed"
			case batchv1.JobSuspended:
				state = "Cancelled"
			}
			message = c.Message
		}
	}
	if job.Annotations["autoops.io/cancelled"] == "true" && state != "Succeeded" && state != "Failed" {
		state = "Cancelling"
		if job.Status.Active == 0 {
			state = "Cancelled"
		}
	}
	var req model.CephTaskRequest
	_ = json.Unmarshal([]byte(job.Annotations["autoops.io/request"]), &req)
	return map[string]any{"name": job.Name, "operation": req.Operation, "params": req.Params, "state": state, "message": message, "actor": job.Annotations["autoops.io/actor"], "createdAt": job.CreationTimestamp, "startedAt": job.Status.StartTime, "finishedAt": job.Status.CompletionTime}
}

func (s *K8sCephService) Tasks(ctx context.Context, id uint) (any, error) {
	client, _, err := s.client(id)
	if err != nil {
		return nil, err
	}
	jobs, err := client.BatchV1().Jobs(cephNamespace).List(ctx, metav1.ListOptions{LabelSelector: cephManaged + "=task"})
	if err != nil {
		return nil, err
	}
	sort.Slice(jobs.Items, func(i, j int) bool {
		return jobs.Items[i].CreationTimestamp.After(jobs.Items[j].CreationTimestamp.Time)
	})
	items := make([]map[string]any, 0, len(jobs.Items))
	for i := range jobs.Items {
		items = append(items, cephJobView(&jobs.Items[i]))
	}
	return items, nil
}

func managedCephJob(ctx context.Context, client kubernetes.Interface, name string) (*batchv1.Job, error) {
	job, err := client.BatchV1().Jobs(cephNamespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if job.Labels[cephManaged] != "task" {
		return nil, fmt.Errorf("仅可操作 AutoOps Ceph 任务")
	}
	return job, nil
}

func (s *K8sCephService) TaskLogs(ctx context.Context, id uint, name string) (any, error) {
	client, _, err := s.client(id)
	if err != nil {
		return nil, err
	}
	job, err := managedCephJob(ctx, client, name)
	if err != nil {
		return nil, err
	}
	pods, err := client.CoreV1().Pods(cephNamespace).List(ctx, metav1.ListOptions{LabelSelector: "job-name=" + name})
	if err != nil {
		return nil, err
	}
	logs := []map[string]any{}
	for _, pod := range pods.Items {
		owned := false
		for _, owner := range pod.OwnerReferences {
			if owner.UID == job.UID {
				owned = true
			}
		}
		if !owned {
			continue
		}
		entry := map[string]any{"pod": pod.Name, "phase": pod.Status.Phase, "conditions": pod.Status.Conditions, "containers": pod.Status.ContainerStatuses, "initContainers": pod.Status.InitContainerStatuses}
		limit := int64(256 * 1024)
		tail := int64(1000)
		stream, e := client.CoreV1().Pods(cephNamespace).GetLogs(pod.Name, &corev1.PodLogOptions{Container: "maintenance", TailLines: &tail, LimitBytes: &limit}).Stream(ctx)
		if e != nil {
			entry["logError"] = e.Error()
		} else {
			buf, readErr := io.ReadAll(io.LimitReader(stream, limit))
			stream.Close()
			entry["log"] = string(buf)
			if readErr != nil {
				entry["logError"] = readErr.Error()
			}
		}
		logs = append(logs, entry)
	}
	return map[string]any{"task": cephJobView(job), "pods": logs}, nil
}

func (s *K8sCephService) CancelTask(ctx context.Context, id uint, name string) (any, error) {
	client, _, err := s.client(id)
	if err != nil {
		return nil, err
	}
	job, err := managedCephJob(ctx, client, name)
	if err != nil {
		return nil, err
	}
	state := cephJobView(job)["state"]
	if state == "Succeeded" || state == "Failed" {
		return nil, fmt.Errorf("已结束的任务不能取消")
	}
	// Suspension stops Pods but keeps the Job and metadata. Never automatically resume a mutation.
	patch, _ := json.Marshal(map[string]any{"metadata": map[string]any{"resourceVersion": job.ResourceVersion, "annotations": map[string]string{"autoops.io/cancelled": "true"}}, "spec": map[string]any{"suspend": true}})
	updated, err := client.BatchV1().Jobs(cephNamespace).Patch(ctx, name, types.MergePatchType, patch, metav1.PatchOptions{})
	if err != nil {
		return nil, err
	}
	return cephJobView(updated), nil
}

func (s *K8sCephService) CreateSchedule(ctx context.Context, id uint, actor string, req model.CephScheduleRequest) (any, error) {
	op, err := validateCephTask(&req.Task)
	if err != nil {
		return nil, err
	}
	if !op.Schedulable {
		return nil, fmt.Errorf("仅支持定时快照和定时全量备份")
	}
	if len(validation.IsDNS1123Label(req.Name)) > 0 || len(req.Name) > 32 {
		return nil, fmt.Errorf("计划名须为不超过 32 位的小写 DNS 名称")
	}
	// Use five cron fields; disallow CRON_TZ/TZ prefixes bypassing the explicit timeZone field.
	if len(strings.Fields(req.Schedule)) != 5 {
		return nil, fmt.Errorf("Cron 表达式必须为五个字段")
	}
	if _, err = cron.ParseStandard(req.Schedule); err != nil {
		return nil, fmt.Errorf("Cron 表达式无效: %w", err)
	}
	if req.TimeZone == "" {
		req.TimeZone = "Asia/Shanghai"
	}
	if _, err = time.LoadLocation(req.TimeZone); err != nil {
		return nil, fmt.Errorf("时区无效")
	}
	client, job, err := s.prepareJob(ctx, id, actor, req.Task)
	if err != nil {
		return nil, err
	}
	success, failed := int32(10), int32(10)
	deadline := int64(300)
	item := &batchv1.CronJob{ObjectMeta: metav1.ObjectMeta{Name: "autoops-ceph-" + req.Name, Namespace: cephNamespace, Labels: map[string]string{cephManaged: "schedule"}, Annotations: job.Annotations}, Spec: batchv1.CronJobSpec{Schedule: req.Schedule, TimeZone: &req.TimeZone, ConcurrencyPolicy: batchv1.ForbidConcurrent, StartingDeadlineSeconds: &deadline, SuccessfulJobsHistoryLimit: &success, FailedJobsHistoryLimit: &failed, JobTemplate: batchv1.JobTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: job.Labels, Annotations: job.Annotations}, Spec: job.Spec}}}
	created, err := client.BatchV1().CronJobs(cephNamespace).Create(ctx, item, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}
	return cephScheduleView(created), nil
}

func cephScheduleView(item *batchv1.CronJob) map[string]any {
	var req model.CephTaskRequest
	_ = json.Unmarshal([]byte(item.Annotations["autoops.io/request"]), &req)
	return map[string]any{"name": item.Name, "schedule": item.Spec.Schedule, "timeZone": item.Spec.TimeZone, "suspended": item.Spec.Suspend != nil && *item.Spec.Suspend, "task": req, "lastScheduleTime": item.Status.LastScheduleTime, "lastSuccessfulTime": item.Status.LastSuccessfulTime, "active": len(item.Status.Active)}
}

func (s *K8sCephService) Schedules(ctx context.Context, id uint) (any, error) {
	client, _, err := s.client(id)
	if err != nil {
		return nil, err
	}
	list, err := client.BatchV1().CronJobs(cephNamespace).List(ctx, metav1.ListOptions{LabelSelector: cephManaged + "=schedule"})
	if err != nil {
		return nil, err
	}
	items := []map[string]any{}
	for i := range list.Items {
		items = append(items, cephScheduleView(&list.Items[i]))
	}
	return items, nil
}

func (s *K8sCephService) ChangeSchedule(ctx context.Context, id uint, name string, suspended bool, remove bool) (any, error) {
	client, _, err := s.client(id)
	if err != nil {
		return nil, err
	}
	item, err := client.BatchV1().CronJobs(cephNamespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if item.Labels[cephManaged] != "schedule" {
		return nil, fmt.Errorf("仅可操作 AutoOps Ceph 计划")
	}
	if remove {
		orphan := metav1.DeletePropagationOrphan
		err = client.BatchV1().CronJobs(cephNamespace).Delete(ctx, name, metav1.DeleteOptions{PropagationPolicy: &orphan, Preconditions: &metav1.Preconditions{UID: &item.UID}})
		return map[string]string{"message": "计划已删除，已有任务和备份保留"}, err
	}
	item.Spec.Suspend = &suspended
	updated, err := client.BatchV1().CronJobs(cephNamespace).Update(ctx, item, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	return cephScheduleView(updated), nil
}

func (s *K8sCephService) Repositories(ctx context.Context, id uint) (any, error) {
	client, _, err := s.client(id)
	if err != nil {
		return nil, err
	}
	list, err := client.CoreV1().ConfigMaps(cephNamespace).List(ctx, metav1.ListOptions{LabelSelector: cephManaged + "=repository"})
	if err != nil {
		return nil, err
	}
	items := []map[string]any{}
	for _, cm := range list.Items {
		items = append(items, map[string]any{"name": cm.Data["name"], "pvc": cm.Data["pvc"], "createdAt": cm.CreationTimestamp})
	}
	return items, nil
}

func (s *K8sCephService) CreateRepository(ctx context.Context, id uint, req model.CephRepositoryRequest) (any, error) {
	if len(validation.IsDNS1123Label(req.Name)) > 0 || len(req.Name) > 32 || len(validation.IsDNS1123Subdomain(req.PVC)) > 0 {
		return nil, fmt.Errorf("仓库名须为不超过 32 位的小写 DNS 名称，PVC 名称必须合法")
	}
	client, _, err := s.client(id)
	if err != nil {
		return nil, err
	}
	pvc, err := client.CoreV1().PersistentVolumeClaims(cephNamespace).Get(ctx, req.PVC, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if pvc.Labels["autoops.io/ceph-backup"] != "true" {
		return nil, fmt.Errorf("请先为专用备份 PVC 添加 autoops.io/ceph-backup=true 标签")
	}
	if pvc.Spec.VolumeMode != nil && *pvc.Spec.VolumeMode == corev1.PersistentVolumeBlock {
		return nil, fmt.Errorf("备份仓库需要 Filesystem PVC")
	}
	rwx := false
	for _, mode := range pvc.Spec.AccessModes {
		if mode == corev1.ReadWriteMany {
			rwx = true
		}
	}
	if !rwx {
		return nil, fmt.Errorf("备份仓库需要 ReadWriteMany，供备份任务和目录读取共享挂载")
	}
	pod, err := readyCephPod(ctx, client, "app=rook-ceph-tools")
	if err != nil {
		return nil, err
	}
	name := "autoops-ceph-repo-" + req.Name
	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: cephNamespace, Labels: map[string]string{cephManaged: "repository"}}, Data: map[string]string{"name": req.Name, "pvc": req.PVC}}
	existing, err := client.CoreV1().ConfigMaps(cephNamespace).Create(ctx, cm, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		existing, err = client.CoreV1().ConfigMaps(cephNamespace).Get(ctx, name, metav1.GetOptions{})
		if err == nil && (existing.Labels[cephManaged] != "repository" || existing.Data["pvc"] != req.PVC) {
			return nil, fmt.Errorf("仓库名已被占用，不能更换 PVC")
		}
	}
	if err != nil {
		return nil, err
	}
	labels := map[string]string{cephManaged: "repository-reader", "autoops.io/repository": req.Name}
	no := false
	one := int32(1)
	dep := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: cephNamespace, OwnerReferences: []metav1.OwnerReference{{APIVersion: "v1", Kind: "ConfigMap", Name: name, UID: existing.UID}}}, Spec: appsv1.DeploymentSpec{Replicas: &one, Selector: &metav1.LabelSelector{MatchLabels: labels}, Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: labels}, Spec: corev1.PodSpec{AutomountServiceAccountToken: &no, ImagePullSecrets: pod.Spec.ImagePullSecrets, Containers: []corev1.Container{{Name: "catalog", Image: pod.Spec.Containers[0].Image, Command: []string{"python3", "-c", "import time; time.sleep(2147483647)"}, VolumeMounts: []corev1.VolumeMount{{Name: "backup", MountPath: "/autoops-backups", ReadOnly: true}}, SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: &no}, Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("10m"), corev1.ResourceMemory: resource.MustParse("32Mi")}, Limits: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("128Mi")}}}}, Volumes: []corev1.Volume{{Name: "backup", VolumeSource: corev1.VolumeSource{PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: req.PVC, ReadOnly: true}}}}}}}}
	_, err = client.AppsV1().Deployments(cephNamespace).Create(ctx, dep, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return nil, fmt.Errorf("仓库记录已创建，但目录读取部署失败，可重新登记修复: %w", err)
	}
	return map[string]string{"name": req.Name, "pvc": req.PVC}, nil
}

const cephCatalogReader = `import json,pathlib
root=pathlib.Path('/autoops-backups')
items=[]
for path in sorted(root.glob('*.json'),reverse=True):
 if len(items)>=5000: raise RuntimeError('备份目录超过 5000 条，请归档旧备份')
 if path.is_symlink(): continue
 try:
  data=json.loads(path.read_text())
  if data.get('version')!=1 or data.get('id')!=path.stem: continue
  data['fileExists']=(root/(path.stem+'.diff')).is_file()
  items.append(data)
 except (ValueError,OSError) as exc:
  items.append({'id':path.stem,'error':str(exc),'fileExists':False})
print(json.dumps(items))
`

func (s *K8sCephService) Backups(ctx context.Context, id uint, repository string) (any, error) {
	if len(validation.IsDNS1123Label(repository)) > 0 || len(repository) > 32 {
		return nil, fmt.Errorf("仓库名无效")
	}
	client, config, err := s.client(id)
	if err != nil {
		return nil, err
	}
	cm, err := client.CoreV1().ConfigMaps(cephNamespace).Get(ctx, "autoops-ceph-repo-"+repository, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if cm.Labels[cephManaged] != "repository" {
		return nil, fmt.Errorf("不是 AutoOps 备份仓库")
	}
	pod, err := readyCephPod(ctx, client, cephManaged+"=repository-reader,autoops.io/repository="+repository)
	if err != nil {
		return nil, err
	}
	out, err := cephExec(ctx, client, config, pod, []string{"python3", "-c", cephCatalogReader})
	if err != nil {
		return nil, err
	}
	var items []map[string]any
	err = json.Unmarshal([]byte(out), &items)
	return items, err
}
