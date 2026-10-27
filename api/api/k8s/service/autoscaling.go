package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"dodevops-api/api/k8s/dao"
	"dodevops-api/api/k8s/model"
	"gorm.io/gorm"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/dynamic"
)

var autoscalingResources = map[string]schema.GroupVersionResource{
	"hpa": {Group: "autoscaling", Version: "v2", Resource: "horizontalpodautoscalers"},
	"vpa": {Group: "autoscaling.k8s.io", Version: "v1", Resource: "verticalpodautoscalers"},
	"kpa": {Group: "serving.knative.dev", Version: "v1", Resource: "services"},
}

type K8sAutoscalingService struct {
	dao    dao.IK8sCRDDao
	writes sync.Mutex
}

func NewK8sAutoscalingService(db *gorm.DB) *K8sAutoscalingService {
	return &K8sAutoscalingService{dao: dao.NewK8sCRDDao(db)}
}

func (s *K8sAutoscalingService) Capabilities(ctx context.Context, id uint) (map[string]interface{}, error) {
	client, err := s.dao.GetClientSet(id)
	if err != nil {
		return nil, err
	}
	result := map[string]interface{}{}
	resources := map[string]schema.GroupVersionResource{}
	for k, v := range autoscalingResources {
		resources[k] = v
	}
	resources["metrics"] = schema.GroupVersionResource{Group: "metrics.k8s.io", Version: "v1beta1", Resource: "pods"}
	for kind, gvr := range resources {
		list := &metav1.APIResourceList{}
		err := client.Discovery().RESTClient().Get().AbsPath("/apis/" + gvr.Group + "/" + gvr.Version).Do(ctx).Into(list)
		available := false
		message := ""
		if err != nil {
			if apierrors.IsNotFound(err) {
				message = "集群未提供此 API，请先安装对应组件"
			} else {
				message = err.Error()
			}
		} else {
			for _, r := range list.APIResources {
				if r.Name == gvr.Resource {
					available = true
					break
				}
			}
			if !available {
				message = "集群未提供此资源"
			}
		}
		result[kind] = map[string]interface{}{"available": available, "apiVersion": gvr.Group + "/" + gvr.Version, "message": message}
	}
	return result, nil
}
func scalingAPIError(kind string, err error) error {
	if err != nil && apierrors.IsNotFound(err) {
		return fmt.Errorf("%s API 或指定资源不存在；VPA 需安装 VPA 控制器，KPA 需安装 Knative Serving: %w", strings.ToUpper(kind), err)
	}
	return err
}
func (s *K8sAutoscalingService) List(ctx context.Context, id uint, ns, kind string) ([]unstructured.Unstructured, error) {
	client, err := s.dao.GetDynamicClient(id)
	if err != nil {
		return nil, err
	}
	list, err := client.Resource(autoscalingResources[kind]).Namespace(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, scalingAPIError(kind, err)
	}
	sort.Slice(list.Items, func(i, j int) bool { return list.Items[i].GetName() < list.Items[j].GetName() })
	if list.Items == nil {
		return []unstructured.Unstructured{}, nil
	}
	return list.Items, nil
}
func (s *K8sAutoscalingService) Get(ctx context.Context, id uint, ns, kind, name string) (*unstructured.Unstructured, error) {
	client, err := s.dao.GetDynamicClient(id)
	if err != nil {
		return nil, err
	}
	item, err := client.Resource(autoscalingResources[kind]).Namespace(ns).Get(ctx, name, metav1.GetOptions{})
	return item, scalingAPIError(kind, err)
}
func controllerManaged(obj *unstructured.Unstructured) bool {
	for _, r := range obj.GetOwnerReferences() {
		if r.Controller != nil && *r.Controller {
			return true
		}
	}
	return false
}
func marshalScalingSpec(value interface{}) (map[string]interface{}, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	err = json.Unmarshal(raw, &out)
	return out, err
}
func (s *K8sAutoscalingService) Save(ctx context.Context, id uint, ns, kind, name string, req model.AutoscalingRequest, create, dryRun bool) (*unstructured.Unstructured, error) {
	if len(validation.IsDNS1123Subdomain(req.Name)) > 0 || (name != "" && name != req.Name) {
		return nil, fmt.Errorf("资源名称无效或与路径不一致")
	}
	if !create && req.ResourceVersion == "" {
		return nil, fmt.Errorf("更新需要 resourceVersion，请刷新后重试")
	}
	var spec map[string]interface{}
	var annotations map[string]string
	var err error
	switch kind {
	case "hpa":
		if err = validateHPA(req.HPA); err == nil {
			spec, err = marshalScalingSpec(req.HPA)
		}
	case "vpa":
		if err = validateVPA(req.VPA); err == nil {
			spec, err = marshalScalingSpec(req.VPA)
		}
	case "kpa":
		if create {
			return nil, fmt.Errorf("请对已有 Knative Service 配置 KPA")
		}
		annotations, err = kpaAnnotations(req.KPA)
	default:
		return nil, fmt.Errorf("未知伸缩类型")
	}
	if err != nil {
		return nil, err
	}
	s.writes.Lock()
	defer s.writes.Unlock()
	client, err := s.dao.GetDynamicClient(id)
	if err != nil {
		return nil, err
	}
	resource := client.Resource(autoscalingResources[kind]).Namespace(ns)
	var obj *unstructured.Unstructured
	if !create {
		obj, err = resource.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, scalingAPIError(kind, err)
		}
		if obj.GetResourceVersion() != req.ResourceVersion {
			return nil, apierrors.NewConflict(autoscalingResources[kind].GroupResource(), name, fmt.Errorf("资源已修改，请刷新后重试"))
		}
		if controllerManaged(obj) {
			return nil, fmt.Errorf("资源由其他控制器管理，请修改其上级资源")
		}
	} else {
		obj = &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": autoscalingResources[kind].Group + "/" + autoscalingResources[kind].Version, "kind": map[string]string{"hpa": "HorizontalPodAutoscaler", "vpa": "VerticalPodAutoscaler"}[kind], "metadata": map[string]interface{}{"name": req.Name, "namespace": ns}}}
	}
	if kind == "kpa" {
		existing, _, _ := unstructured.NestedStringMap(obj.Object, "spec", "template", "metadata", "annotations")
		if existing == nil {
			existing = map[string]string{}
		}
		for k, v := range annotations {
			existing[k] = v
		}
		if err = validateKPABounds(existing); err != nil {
			return nil, err
		}
		if err = unstructured.SetNestedStringMap(obj.Object, existing, "spec", "template", "metadata", "annotations"); err != nil {
			return nil, err
		}
		// A changed template must receive a new revision name from Knative.
		unstructured.RemoveNestedField(obj.Object, "spec", "template", "metadata", "name")
	} else {
		if err = checkScalingConflicts(ctx, client, ns, kind, req.Name, spec); err != nil {
			return nil, err
		}
		// Preserve VPA extension fields that this editor does not own.
		if kind == "vpa" && !create {
			old, _, _ := unstructured.NestedMap(obj.Object, "spec")
			if old != nil {
				policy, _ := old["updatePolicy"].(map[string]interface{})
				if policy == nil {
					policy = map[string]interface{}{}
				}
				delete(policy, "minReplicas")
				delete(policy, "updateMode")
				for k, v := range spec["updatePolicy"].(map[string]interface{}) {
					policy[k] = v
				}
				old["targetRef"] = spec["targetRef"]
				old["resourcePolicy"] = spec["resourcePolicy"]
				old["updatePolicy"] = policy
				spec = old
			}
		}
		if err = unstructured.SetNestedMap(obj.Object, spec, "spec"); err != nil {
			return nil, err
		}
	}
	if dryRun {
		if create {
			return resource.Create(ctx, obj, metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}})
		}
		return resource.Update(ctx, obj, metav1.UpdateOptions{DryRun: []string{metav1.DryRunAll}})
	}
	if create {
		return resource.Create(ctx, obj, metav1.CreateOptions{})
	}
	return resource.Update(ctx, obj, metav1.UpdateOptions{})
}

func checkScalingConflicts(ctx context.Context, client dynamic.Interface, ns, kind, name string, spec map[string]interface{}) error {
	targetField := "scaleTargetRef"
	if kind == "vpa" {
		targetField = "targetRef"
	}
	target, _ := spec[targetField].(map[string]interface{})
	targetName, _ := target["name"].(string)
	targetKind, _ := target["kind"].(string)
	plural := map[string]string{"Deployment": "deployments", "StatefulSet": "statefulsets", "DaemonSet": "daemonsets"}[targetKind]
	workload, err := client.Resource(schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: plural}).Namespace(ns).Get(ctx, targetName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("读取伸缩目标失败: %w", err)
	}
	if workload.GetLabels()["serving.knative.dev/service"] != "" || workload.GetLabels()["serving.knative.dev/revision"] != "" {
		return fmt.Errorf("Knative 工作负载必须通过 Service 配置 KPA，不能添加独立 HPA/VPA")
	}
	for _, owner := range workload.GetOwnerReferences() {
		if strings.HasPrefix(owner.APIVersion, "serving.knative.dev/") {
			return fmt.Errorf("目标由 Knative 控制器管理，请通过 Service 配置 KPA")
		}
	}
	for _, otherKind := range []string{"hpa", "vpa"} {
		list, err := client.Resource(autoscalingResources[otherKind]).Namespace(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			if otherKind == "vpa" && apierrors.IsNotFound(err) {
				continue
			}
			return fmt.Errorf("无法检查伸缩策略冲突: %w", err)
		}
		for _, item := range list.Items {
			if otherKind == kind && item.GetName() == name {
				continue
			}
			otherSpec, _, _ := unstructured.NestedMap(item.Object, "spec")
			field := "scaleTargetRef"
			if otherKind == "vpa" {
				field = "targetRef"
			}
			otherTarget, _ := otherSpec[field].(map[string]interface{})
			if scalingTargetKey(target) != scalingTargetKey(otherTarget) {
				continue
			}
			if otherKind == kind {
				return fmt.Errorf("同一工作负载已存在同类型伸缩策略")
			}
			hpa, vpa := spec, otherSpec
			if kind == "vpa" {
				hpa, vpa = otherSpec, spec
			}
			policy, _ := vpa["updatePolicy"].(map[string]interface{})
			mode, _ := policy["updateMode"].(string)
			if mode != "Off" && hpaUsesResourceMetrics(hpa) {
				return fmt.Errorf("资源指标 HPA 与会修改 requests 的 VPA 存在冲突；请将 VPA 设为 Off，或让 HPA 使用业务指标")
			}
		}
	}
	return nil
}
func (s *K8sAutoscalingService) Delete(ctx context.Context, id uint, ns, kind, name, version string, dryRun bool) (interface{}, error) {
	if version == "" {
		return nil, fmt.Errorf("删除/重置需要 resourceVersion，请刷新后重试")
	}
	s.writes.Lock()
	defer s.writes.Unlock()
	client, err := s.dao.GetDynamicClient(id)
	if err != nil {
		return nil, err
	}
	r := client.Resource(autoscalingResources[kind]).Namespace(ns)
	obj, err := r.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, scalingAPIError(kind, err)
	}
	if version != obj.GetResourceVersion() {
		return nil, apierrors.NewConflict(autoscalingResources[kind].GroupResource(), name, fmt.Errorf("资源已修改，请刷新后重试"))
	}
	if controllerManaged(obj) {
		return nil, fmt.Errorf("资源由其他控制器管理，请修改其上级资源")
	}
	dry := []string{}
	if dryRun {
		dry = []string{metav1.DryRunAll}
	}
	if kind == "kpa" {
		annotations, _, _ := unstructured.NestedStringMap(obj.Object, "spec", "template", "metadata", "annotations")
		for _, key := range managedKPAKeys {
			delete(annotations, kpaPrefix+key)
		}
		if err = unstructured.SetNestedStringMap(obj.Object, annotations, "spec", "template", "metadata", "annotations"); err != nil {
			return nil, err
		}
		unstructured.RemoveNestedField(obj.Object, "spec", "template", "metadata", "name")
		return r.Update(ctx, obj, metav1.UpdateOptions{DryRun: dry})
	}
	err = r.Delete(ctx, name, metav1.DeleteOptions{Preconditions: &metav1.Preconditions{ResourceVersion: &version}, DryRun: dry})
	return map[string]interface{}{"name": name, "dryRun": dryRun}, err
}
