package service

import (
	"context"
	"strings"
	"testing"

	"dodevops-api/api/k8s/model"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	"k8s.io/client-go/kubernetes"
	ktesting "k8s.io/client-go/testing"
)

type autoscalingTestDAO struct{ client dynamic.Interface }

func (d autoscalingTestDAO) GetDynamicClient(uint) (dynamic.Interface, error) { return d.client, nil }
func (d autoscalingTestDAO) GetClientSet(uint) (*kubernetes.Clientset, error) { return nil, nil }
func scalingTestClient(objects ...runtime.Object) *dynamicfake.FakeDynamicClient {
	lists := map[schema.GroupVersionResource]string{}
	for k, v := range autoscalingResources {
		lists[v] = map[string]string{"hpa": "HorizontalPodAutoscalerList", "vpa": "VerticalPodAutoscalerList", "kpa": "ServiceList"}[k]
	}
	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), lists, objects...)
}
func scalingObject(api, kind, name string, spec map[string]interface{}) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": api, "kind": kind, "metadata": map[string]interface{}{"name": name, "namespace": "team", "resourceVersion": "1"}, "spec": spec}}
}
func targetRef() map[string]interface{} {
	return map[string]interface{}{"apiVersion": "apps/v1", "kind": "Deployment", "name": "app"}
}
func basicHPA() *autoscalingv2.HorizontalPodAutoscalerSpec {
	min, util := int32(1), int32(70)
	return &autoscalingv2.HorizontalPodAutoscalerSpec{ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{APIVersion: "apps/v1", Kind: "Deployment", Name: "app"}, MinReplicas: &min, MaxReplicas: 5, Metrics: []autoscalingv2.MetricSpec{{Type: "Resource", Resource: &autoscalingv2.ResourceMetricSource{Name: "cpu", Target: autoscalingv2.MetricTarget{Type: "Utilization", AverageUtilization: &util}}}}}
}
func basicKPA() *model.KPAConfig {
	return &model.KPAConfig{Metric: "concurrency", Target: scalingTestPtr(100.0), MinScale: scalingTestPtr(int64(0)), MaxScale: scalingTestPtr(int64(10)), WindowSeconds: scalingTestPtr(int64(60)), PanicWindowPercentage: scalingTestPtr(10.0), PanicThresholdPercentage: scalingTestPtr(200.0), TargetUtilizationPercentage: scalingTestPtr(70.0)}
}

func TestAutoscalingValidationRejectsUnsafePolicies(t *testing.T) {
	tests := []struct {
		name   string
		change func(*autoscalingv2.HorizontalPodAutoscalerSpec)
	}{
		{"invalid bounds", func(s *autoscalingv2.HorizontalPodAutoscalerSpec) { s.MaxReplicas = 0 }},
		{"unsupported target", func(s *autoscalingv2.HorizontalPodAutoscalerSpec) { s.ScaleTargetRef.Kind = "DaemonSet" }},
		{"no metrics", func(s *autoscalingv2.HorizontalPodAutoscalerSpec) { s.Metrics = nil }},
		{"multiple metric sources", func(s *autoscalingv2.HorizontalPodAutoscalerSpec) {
			s.Metrics[0].Pods = &autoscalingv2.PodsMetricSource{}
		}},
		{"wrong target type", func(s *autoscalingv2.HorizontalPodAutoscalerSpec) { s.Metrics[0].Resource.Target.Type = "Value" }},
	}
	if err := validateHPA(basicHPA()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := basicHPA()
			tc.change(s)
			if validateHPA(s) == nil {
				t.Fatal("unsafe HPA accepted")
			}
		})
	}
	vpa := &model.VPASpec{TargetRef: basicHPA().ScaleTargetRef, UpdatePolicy: model.VPAUpdatePolicy{UpdateMode: "Off"}, ResourcePolicy: model.VPAResourcePolicy{ContainerPolicies: []model.VPAContainerPolicy{{ContainerName: "*", MinAllowed: map[string]string{"cpu": "2"}, MaxAllowed: map[string]string{"cpu": "100m"}}}}}
	if validateVPA(vpa) == nil {
		t.Fatal("inverted VPA bounds accepted")
	}
	kpa := basicKPA()
	kpa.PanicThresholdPercentage = scalingTestPtr(100.0)
	if _, err := kpaAnnotations(kpa); err == nil {
		t.Fatal("unsafe panic threshold accepted")
	}
}

func TestAutoscalingConflictChecks(t *testing.T) {
	ctx := context.Background()
	hpaSpec, _ := marshalScalingSpec(basicHPA())
	vpa := scalingObject("autoscaling.k8s.io/v1", "VerticalPodAutoscaler", "vertical", map[string]interface{}{"targetRef": targetRef(), "updatePolicy": map[string]interface{}{"updateMode": "Auto"}})
	workload := scalingObject("apps/v1", "Deployment", "app", map[string]interface{}{})
	client := scalingTestClient(workload, vpa)
	if err := checkScalingConflicts(ctx, client, "team", "hpa", "horizontal", hpaSpec); err == nil || !strings.Contains(err.Error(), "冲突") {
		t.Fatalf("expected HPA/VPA conflict, got %v", err)
	}
	vpa.Object["spec"].(map[string]interface{})["updatePolicy"] = map[string]interface{}{"updateMode": "Off"}
	client = scalingTestClient(workload, vpa)
	if err := checkScalingConflicts(ctx, client, "team", "hpa", "horizontal", hpaSpec); err != nil {
		t.Fatalf("recommendation-only VPA should coexist: %v", err)
	}
	existing := scalingObject("autoscaling/v2", "HorizontalPodAutoscaler", "other", hpaSpec)
	client = scalingTestClient(workload, existing)
	if err := checkScalingConflicts(ctx, client, "team", "hpa", "horizontal", hpaSpec); err == nil {
		t.Fatal("duplicate controller target accepted")
	}
	workload.SetLabels(map[string]string{"serving.knative.dev/service": "app"})
	client = scalingTestClient(workload)
	if err := checkScalingConflicts(ctx, client, "team", "hpa", "horizontal", hpaSpec); err == nil {
		t.Fatal("Knative-owned workload accepted")
	}
}

func TestAutoscalingKPAUpdatePreservesServiceAndReset(t *testing.T) {
	ctx := context.Background()
	spec := map[string]interface{}{"template": map[string]interface{}{"metadata": map[string]interface{}{"name": "app-old-revision", "annotations": map[string]interface{}{"example.com/keep": "yes"}}, "spec": map[string]interface{}{"containers": []interface{}{map[string]interface{}{"image": "example/app:v1"}}}}, "traffic": []interface{}{map[string]interface{}{"latestRevision": true, "percent": int64(100)}}}
	client := scalingTestClient(scalingObject("serving.knative.dev/v1", "Service", "app", spec))
	svc := &K8sAutoscalingService{dao: autoscalingTestDAO{client}}
	req := model.AutoscalingRequest{Name: "app", ResourceVersion: "1", KPA: basicKPA()}
	updated, err := svc.Save(ctx, 1, "team", "kpa", "app", req, false, false)
	if err != nil {
		t.Fatal(err)
	}
	a, _, _ := unstructured.NestedStringMap(updated.Object, "spec", "template", "metadata", "annotations")
	if a["example.com/keep"] != "yes" || a[kpaPrefix+"class"] != "kpa.autoscaling.knative.dev" {
		t.Fatalf("annotations not merged: %v", a)
	}
	if _, found, _ := unstructured.NestedString(updated.Object, "spec", "template", "metadata", "name"); found {
		t.Fatal("old revision name retained")
	}
	containers, _, _ := unstructured.NestedSlice(updated.Object, "spec", "template", "spec", "containers")
	if len(containers) != 1 {
		t.Fatal("service containers removed")
	}
	value, err := svc.Delete(ctx, 1, "team", "kpa", "app", "1", false)
	if err != nil {
		t.Fatal(err)
	}
	reset := value.(*unstructured.Unstructured)
	a, _, _ = unstructured.NestedStringMap(reset.Object, "spec", "template", "metadata", "annotations")
	if len(a) != 1 || a["example.com/keep"] != "yes" {
		t.Fatalf("reset altered unrelated annotations: %v", a)
	}
	if _, err = client.Resource(autoscalingResources["kpa"]).Namespace("team").Get(ctx, "app", metav1.GetOptions{}); err != nil {
		t.Fatal("reset deleted Service")
	}
	req.ResourceVersion = "stale"
	if _, err = svc.Save(ctx, 1, "team", "kpa", "app", req, false, false); !apierrors.IsConflict(err) {
		t.Fatalf("stale resourceVersion accepted: %v", err)
	}
}
func TestAutoscalingDryRunForwardedAndOwnershipProtected(t *testing.T) {
	ctx := context.Background()
	client := scalingTestClient(scalingObject("apps/v1", "Deployment", "app", map[string]interface{}{}))
	svc := &K8sAutoscalingService{dao: autoscalingTestDAO{client}}
	var dryRun bool
	client.PrependReactor("create", "horizontalpodautoscalers", func(action ktesting.Action) (bool, runtime.Object, error) {
		create := action.(ktesting.CreateActionImpl)
		dryRun = len(create.GetCreateOptions().DryRun) == 1 && create.GetCreateOptions().DryRun[0] == metav1.DryRunAll
		return true, create.GetObject(), nil
	})
	_, err := svc.Save(ctx, 1, "team", "hpa", "", model.AutoscalingRequest{Name: "scale", HPA: basicHPA()}, true, true)
	if err != nil || !dryRun {
		t.Fatalf("dry-run not forwarded: %v", err)
	}
	obj := scalingObject("autoscaling/v2", "HorizontalPodAutoscaler", "owned", map[string]interface{}{})
	yes := true
	obj.SetOwnerReferences([]metav1.OwnerReference{{APIVersion: "example/v1", Kind: "Controller", Name: "owner", Controller: &yes}})
	client = scalingTestClient(obj)
	svc.dao = autoscalingTestDAO{client}
	if _, err = svc.Delete(ctx, 1, "team", "hpa", "owned", "1", false); err == nil {
		t.Fatal("controller-owned HPA deleted")
	}
}

func scalingTestPtr[T any](v T) *T { return &v }

func TestAutoscalingEditingRetainsInheritedDefaults(t *testing.T) {
	ctx := context.Background()
	annotations, err := kpaAnnotations(&model.KPAConfig{MinScale: scalingTestPtr(int64(2))})
	if err != nil {
		t.Fatal(err)
	}
	if len(annotations) != 2 || annotations[kpaPrefix+"min-scale"] != "2" {
		t.Fatalf("sparse KPA edit pinned inherited defaults: %v", annotations)
	}
	if validateKPABounds(map[string]string{kpaPrefix + "min-scale": "10", kpaPrefix + "max-scale": "2"}) == nil {
		t.Fatal("merged KPA bounds not checked")
	}
	spec := map[string]interface{}{"targetRef": targetRef(), "updatePolicy": map[string]interface{}{"evictAfterOOMSeconds": int64(600)}, "recommenders": []interface{}{map[string]interface{}{"name": "custom"}}}
	client := scalingTestClient(scalingObject("apps/v1", "Deployment", "app", map[string]interface{}{}), scalingObject("autoscaling.k8s.io/v1", "VerticalPodAutoscaler", "vertical", spec))
	svc := &K8sAutoscalingService{dao: autoscalingTestDAO{client}}
	req := model.AutoscalingRequest{Name: "vertical", ResourceVersion: "1", VPA: &model.VPASpec{TargetRef: basicHPA().ScaleTargetRef, ResourcePolicy: model.VPAResourcePolicy{ContainerPolicies: []model.VPAContainerPolicy{}}}}
	updated, err := svc.Save(ctx, 1, "team", "vpa", "vertical", req, false, false)
	if err != nil {
		t.Fatal(err)
	}
	policy, _, _ := unstructured.NestedMap(updated.Object, "spec", "updatePolicy")
	if _, ok := policy["updateMode"]; ok {
		t.Fatal("editing default VPA injected mode")
	}
	if _, ok := policy["minReplicas"]; ok {
		t.Fatal("editing default VPA injected minReplicas")
	}
	if policy["evictAfterOOMSeconds"] != int64(600) {
		t.Fatal("VPA extension policy lost")
	}
	if _, found, _ := unstructured.NestedSlice(updated.Object, "spec", "recommenders"); !found {
		t.Fatal("custom recommender lost")
	}
}
