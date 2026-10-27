package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"dodevops-api/api/k8s/model"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"
)

// These tests deliberately bypass the application's MySQL, Redis and scheduler.
// They exercise its real service against a real Kubernetes API. Normal go test
// skips them. Never print kubeconfig contents or authentication headers.
//
// Read-only discovery:
//
//	AUTOOPS_AUTOSCALING_LIVE_KUBECONFIG=/secure/test-config go test ./api/k8s/service -run '^TestAutoscalingLive$' -v -count=1
//
// CRUD in a newly generated namespace, removed with its UID on completion:
//
//	AUTOOPS_AUTOSCALING_LIVE_KUBECONFIG=/secure/test-config \
//	AUTOOPS_AUTOSCALING_LIVE_MUTATE=1 \
//	AUTOOPS_AUTOSCALING_LIVE_IMAGE=registry.example/test-http@sha256:... \
//	go test ./api/k8s/service -run '^TestAutoscalingLive$' -v -count=1 -timeout=8m
//
// Optional: AUTOOPS_AUTOSCALING_LIVE_KINDS=hpa runs the baseline before VPA and
// Knative installation. HPA is always part of the baseline. Set
// AUTOOPS_AUTOSCALING_LIVE_NAMESPACE=autoops-api-e2e-<unique-suffix> to reserve a
// specific name; the test refuses to adopt an existing namespace. A generated
// name is used otherwise. Fixtures are restricted to worker nodes carrying
// kubernetes.io/role=node. Knative must enable its podspec-nodeselector feature.
// AUTOOPS_AUTOSCALING_LIVE_HTTP_COMMAND_JSON can provide a JSON command array,
// Python fixtures must explicitly handle SIGTERM when running as PID 1;
// otherwise Knative may wait its full 300-second termination grace period.
// The image must serve HTTP on port 8080 for the Knative fixture. This suite
// verifies service/API behavior; load-driven scaling and resource recommendations
// need separate sustained workload tests, not merely successful CRUD.
type autoscalingLiveDAO struct {
	dynamic dynamic.Interface
	typed   *kubernetes.Clientset
}

func (d autoscalingLiveDAO) GetDynamicClient(uint) (dynamic.Interface, error) { return d.dynamic, nil }
func (d autoscalingLiveDAO) GetClientSet(uint) (*kubernetes.Clientset, error) { return d.typed, nil }

func TestAutoscalingLive(t *testing.T) {
	path := os.Getenv("AUTOOPS_AUTOSCALING_LIVE_KUBECONFIG")
	if path == "" {
		t.Skip("set AUTOOPS_AUTOSCALING_LIVE_KUBECONFIG to opt into real-cluster tests")
	}
	cfg, err := clientcmd.BuildConfigFromFlags("", path)
	if err != nil {
		t.Fatalf("load isolated kubeconfig: %v", err)
	}
	cfg.Timeout = 25 * time.Second
	cfg.UserAgent = "autoops-autoscaling-integration-test"
	typed, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	svc := &K8sAutoscalingService{dao: autoscalingLiveDAO{dynamic: dyn, typed: typed}}
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()
	version, err := typed.Discovery().ServerVersion()
	if err != nil {
		t.Fatalf("Kubernetes discovery: %v", err)
	}
	t.Logf("Kubernetes version: %s", version.GitVersion)
	selected := map[string]bool{"hpa": true, "vpa": true, "kpa": true}
	if requested := os.Getenv("AUTOOPS_AUTOSCALING_LIVE_KINDS"); requested != "" {
		selected = map[string]bool{"hpa": true}
		for _, kind := range strings.Split(requested, ",") {
			if kind != "hpa" && kind != "vpa" && kind != "kpa" {
				t.Fatalf("invalid live test kind %q", kind)
			}
			selected[kind] = true
		}
	}
	ready := t.Run("API_capabilities", func(t *testing.T) {
		caps, err := svc.Capabilities(ctx, 1)
		if err != nil {
			t.Fatal(err)
		}
		for _, kind := range []string{"hpa", "vpa", "kpa", "metrics"} {
			cap, ok := caps[kind].(map[string]interface{})
			if !ok {
				t.Fatalf("%s capability shape mismatch", kind)
			}
			t.Logf("%s API=%s available=%v", kind, cap["apiVersion"], cap["available"])
			if (selected[kind] || kind == "metrics") && cap["available"] != true {
				t.Errorf("%s API unavailable: %v", kind, cap["message"])
			}
		}
	})
	if !ready {
		return
	}
	if os.Getenv("AUTOOPS_AUTOSCALING_LIVE_MUTATE") != "1" {
		t.Log("read-only checks complete; mutations not enabled")
		return
	}
	image := os.Getenv("AUTOOPS_AUTOSCALING_LIVE_IMAGE")
	if image == "" {
		t.Fatal("mutation tests require AUTOOPS_AUTOSCALING_LIVE_IMAGE (HTTP on port 8080)")
	}
	nsSpec := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{GenerateName: "autoops-autoscaling-test-", Labels: map[string]string{"autoops.io/integration-test": "autoscaling"}}}
	if requested := os.Getenv("AUTOOPS_AUTOSCALING_LIVE_NAMESPACE"); requested != "" {
		if !strings.HasPrefix(requested, "autoops-api-e2e-") {
			t.Fatal("explicit live test namespace must start with autoops-api-e2e-")
		}
		nsSpec.Name = requested
		nsSpec.GenerateName = ""
	}
	var httpCommand []string
	if value := os.Getenv("AUTOOPS_AUTOSCALING_LIVE_HTTP_COMMAND_JSON"); value != "" {
		if err = json.Unmarshal([]byte(value), &httpCommand); err != nil || len(httpCommand) == 0 {
			t.Fatal("HTTP command must be a nonempty JSON string array")
		}
	}
	ns, err := typed.CoreV1().Namespaces().Create(ctx, nsSpec, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("create isolated test namespace: %v", err)
	}
	namespace := ns.Name
	t.Logf("isolated namespace: %s", namespace)
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 180*time.Second)
		defer cleanupCancel()
		uid := ns.UID
		err := typed.CoreV1().Namespaces().Delete(cleanupCtx, namespace, metav1.DeleteOptions{Preconditions: &metav1.Preconditions{UID: &uid}})
		if err != nil && !apierrors.IsNotFound(err) {
			t.Errorf("cleanup namespace %s: %v", namespace, err)
			return
		}
		err = wait.PollUntilContextTimeout(cleanupCtx, time.Second, 175*time.Second, true, func(ctx context.Context) (bool, error) {
			_, err := typed.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
			return apierrors.IsNotFound(err), ignoreAutoscalingNotFound(err)
		})
		if err != nil {
			t.Errorf("namespace %s deletion incomplete: %v", namespace, err)
		} else {
			t.Logf("cleanup confirmed: namespace %s removed", namespace)
		}
	})
	createDeployment := func(t *testing.T, name string) {
		t.Helper()
		zero := int32(0)
		_, err := typed.AppsV1().Deployments(namespace).Create(ctx, &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: name}, Spec: appsv1.DeploymentSpec{Replicas: &zero, Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": name}}, Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": name}}, Spec: corev1.PodSpec{NodeSelector: map[string]string{"kubernetes.io/role": "node"}, Containers: []corev1.Container{{Name: "http", Image: image, Ports: []corev1.ContainerPort{{ContainerPort: 8080}}, Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("50m"), corev1.ResourceMemory: resource.MustParse("32Mi")}, Limits: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("100m"), corev1.ResourceMemory: resource.MustParse("128Mi")}}}}}}}}, metav1.CreateOptions{})
		if err != nil {
			t.Fatalf("create deployment %s: %v", name, err)
		}
	}
	createDeployment(t, "app")
	saveExisting := func(kind, name string, req model.AutoscalingRequest, dry bool) (*unstructured.Unstructured, error) {
		var saved *unstructured.Unstructured
		err := retry.OnError(wait.Backoff{Steps: 8, Duration: 100 * time.Millisecond, Factor: 1.7, Jitter: 0.1}, apierrors.IsConflict, func() error {
			current, err := svc.Get(ctx, 1, namespace, kind, name)
			if err != nil {
				return err
			}
			req.ResourceVersion = current.GetResourceVersion()
			saved, err = svc.Save(ctx, 1, namespace, kind, name, req, false, dry)
			return err
		})
		return saved, err
	}
	deleteExisting := func(kind, name string, dry bool) error {
		return retry.OnError(wait.Backoff{Steps: 8, Duration: 100 * time.Millisecond, Factor: 1.7, Jitter: 0.1}, apierrors.IsConflict, func() error {
			current, err := svc.Get(ctx, 1, namespace, kind, name)
			if err != nil {
				return err
			}
			_, err = svc.Delete(ctx, 1, namespace, kind, name, current.GetResourceVersion(), dry)
			return err
		})
	}
	hpaCreated := t.Run("HPA_dry_run_CRUD_and_stale_version", func(t *testing.T) {
		req := model.AutoscalingRequest{Name: "horizontal", HPA: basicHPA()}
		checked, err := svc.Save(ctx, 1, namespace, "hpa", "", req, true, true)
		if err != nil {
			t.Fatalf("HPA dry-run: %v", err)
		}
		if checked.GetAPIVersion() != "autoscaling/v2" {
			t.Errorf("unexpected API version %s", checked.GetAPIVersion())
		}
		if _, err = svc.Get(ctx, 1, namespace, "hpa", req.Name); !apierrors.IsNotFound(err) {
			t.Fatalf("dry-run created a persisted HPA: %v", err)
		}
		created, err := svc.Save(ctx, 1, namespace, "hpa", "", req, true, false)
		if err != nil {
			t.Fatalf("create HPA: %v", err)
		}
		if created.GetResourceVersion() == "" {
			t.Fatal("created HPA lacks resourceVersion")
		}
		list, err := svc.List(ctx, 1, namespace, "hpa")
		if err != nil || len(list) != 1 {
			t.Fatalf("HPA list: count=%d err=%v", len(list), err)
		}
		req.HPA.MaxReplicas = 4
		if _, err = saveExisting("hpa", req.Name, req, true); err != nil {
			t.Fatalf("HPA update dry-run: %v", err)
		}
		unchanged, err := svc.Get(ctx, 1, namespace, "hpa", req.Name)
		if err != nil {
			t.Fatal(err)
		}
		maximum, _, _ := unstructured.NestedInt64(unchanged.Object, "spec", "maxReplicas")
		if maximum != 5 {
			t.Fatalf("dry-run update persisted maxReplicas=%d", maximum)
		}
		if _, err = saveExisting("hpa", req.Name, req, false); err != nil {
			t.Fatalf("HPA update: %v", err)
		}
		// Use an actual earlier API version, not merely an invalid sentinel.
		req.ResourceVersion = created.GetResourceVersion()
		if _, err = svc.Save(ctx, 1, namespace, "hpa", req.Name, req, false, false); !apierrors.IsConflict(err) {
			t.Fatalf("stale HPA update accepted: %v", err)
		}
		if _, err = svc.Delete(ctx, 1, namespace, "hpa", req.Name, created.GetResourceVersion(), false); !apierrors.IsConflict(err) {
			t.Fatalf("stale HPA delete accepted: %v", err)
		}
		if err = deleteExisting("hpa", req.Name, true); err != nil {
			t.Fatalf("HPA delete dry-run: %v", err)
		}
		if _, err = svc.Get(ctx, 1, namespace, "hpa", req.Name); err != nil {
			t.Fatalf("dry-run delete removed HPA: %v", err)
		}
	})
	if hpaCreated && selected["vpa"] {
		t.Run("VPA_recommendation_policy_and_target_conflicts", func(t *testing.T) {
			req := model.AutoscalingRequest{Name: "vertical", VPA: &model.VPASpec{TargetRef: basicHPA().ScaleTargetRef, UpdatePolicy: model.VPAUpdatePolicy{UpdateMode: "Off"}, ResourcePolicy: model.VPAResourcePolicy{ContainerPolicies: []model.VPAContainerPolicy{{ContainerName: "*", ControlledResources: []string{"cpu", "memory"}, ControlledValues: "RequestsOnly", MinAllowed: map[string]string{"cpu": "25m", "memory": "16Mi"}, MaxAllowed: map[string]string{"cpu": "100m", "memory": "128Mi"}}}}}}
			if _, err := svc.Save(ctx, 1, namespace, "vpa", "", req, true, true); err != nil {
				t.Fatalf("VPA dry-run: %v", err)
			}
			if _, err := svc.Get(ctx, 1, namespace, "vpa", req.Name); !apierrors.IsNotFound(err) {
				t.Fatalf("VPA dry-run persisted: %v", err)
			}
			if _, err := svc.Save(ctx, 1, namespace, "vpa", "", req, true, false); err != nil {
				t.Fatalf("create Off VPA beside resource HPA: %v", err)
			}
			list, err := svc.List(ctx, 1, namespace, "vpa")
			if err != nil || len(list) != 1 {
				t.Fatalf("VPA list: count=%d err=%v", len(list), err)
			}
			for _, mode := range []string{"Auto", "Recreate"} {
				req.VPA.UpdatePolicy.UpdateMode = mode
				if _, err = saveExisting("vpa", req.Name, req, false); err == nil || !strings.Contains(err.Error(), "冲突") {
					t.Fatalf("%s VPA/HPA conflict not blocked: %v", mode, err)
				}
			}
			duplicate := model.AutoscalingRequest{Name: "duplicate-hpa", HPA: basicHPA()}
			if _, err = svc.Save(ctx, 1, namespace, "hpa", "", duplicate, true, false); err == nil || !strings.Contains(err.Error(), "同类型") {
				t.Fatalf("duplicate HPA target not blocked: %v", err)
			}
			req.VPA.UpdatePolicy.UpdateMode = "Off"
			req.VPA.ResourcePolicy.ContainerPolicies[0].MaxAllowed["cpu"] = "150m"
			if _, err = saveExisting("vpa", req.Name, req, false); err != nil {
				t.Fatalf("edit VPA policy: %v", err)
			}
			if err = deleteExisting("vpa", req.Name, true); err != nil {
				t.Fatalf("VPA delete dry-run: %v", err)
			}
			if _, err = svc.Get(ctx, 1, namespace, "vpa", req.Name); err != nil {
				t.Fatalf("VPA dry-run delete persisted: %v", err)
			}
			if err = deleteExisting("vpa", req.Name, false); err != nil {
				t.Fatalf("delete VPA: %v", err)
			}
		})
	}
	if hpaCreated {
		t.Run("HPA_delete", func(t *testing.T) {
			if err := deleteExisting("hpa", "horizontal", false); err != nil {
				t.Fatal(err)
			}
			if _, err := svc.Get(ctx, 1, namespace, "hpa", "horizontal"); !apierrors.IsNotFound(err) {
				t.Fatalf("deleted HPA still exists: %v", err)
			}
		})
	}
	t.Run("concurrent_HPA_target_serialization", func(t *testing.T) {
		createDeployment(t, "race-target")
		var wg sync.WaitGroup
		outcomes := make(chan error, 2)
		start := make(chan struct{})
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()
				<-start
				spec := basicHPA()
				spec.ScaleTargetRef.Name = "race-target"
				_, err := svc.Save(ctx, 1, namespace, "hpa", "", model.AutoscalingRequest{Name: fmt.Sprintf("race-%d", index), HPA: spec}, true, false)
				outcomes <- err
			}(i)
		}
		close(start)
		wg.Wait()
		close(outcomes)
		succeeded, blocked := 0, 0
		for err := range outcomes {
			if err == nil {
				succeeded++
			} else if strings.Contains(err.Error(), "同类型") {
				blocked++
			} else {
				t.Errorf("unexpected concurrent result: %v", err)
			}
		}
		if succeeded != 1 || blocked != 1 {
			t.Fatalf("expected one success and one target conflict, got success=%d blocked=%d", succeeded, blocked)
		}
	})
	if selected["kpa"] {
		t.Run("KPA_sparse_update_dry_run_and_reset", func(t *testing.T) {
			fixture := &unstructured.Unstructured{Object: map[string]interface{}{
				"apiVersion": "serving.knative.dev/v1", "kind": "Service",
				"metadata": map[string]interface{}{"name": "knative-http", "namespace": namespace, "labels": map[string]interface{}{"networking.knative.dev/visibility": "cluster-local"}},
				"spec":     map[string]interface{}{"template": map[string]interface{}{"metadata": map[string]interface{}{"annotations": map[string]interface{}{kpaPrefix + "class": "kpa.autoscaling.knative.dev", kpaPrefix + "min-scale": "0", kpaPrefix + "max-scale": "1", "autoops.io/preserve": "yes"}}, "spec": map[string]interface{}{"nodeSelector": map[string]interface{}{"kubernetes.io/role": "node"}, "containers": []interface{}{map[string]interface{}{"name": "http", "image": image, "ports": []interface{}{map[string]interface{}{"containerPort": int64(8080)}}, "resources": map[string]interface{}{"requests": map[string]interface{}{"cpu": "25m", "memory": "32Mi"}, "limits": map[string]interface{}{"cpu": "100m", "memory": "128Mi"}}}}}}},
			}}
			if len(httpCommand) > 0 {
				containers, _, _ := unstructured.NestedSlice(fixture.Object, "spec", "template", "spec", "containers")
				values := make([]interface{}, len(httpCommand))
				for i, arg := range httpCommand {
					values[i] = arg
				}
				containers[0].(map[string]interface{})["command"] = values
				if err = unstructured.SetNestedSlice(fixture.Object, containers, "spec", "template", "spec", "containers"); err != nil {
					t.Fatal(err)
				}
			}
			initial, err := dyn.Resource(autoscalingResources["kpa"]).Namespace(namespace).Create(ctx, fixture, metav1.CreateOptions{})
			if err != nil {
				t.Fatalf("create Knative fixture: %v", err)
			}
			t.Cleanup(func() {
				cleanupCtx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
				defer cancel()
				uid := initial.GetUID()
				foreground := metav1.DeletePropagationForeground
				resource := dyn.Resource(autoscalingResources["kpa"]).Namespace(namespace)
				if err := resource.Delete(cleanupCtx, initial.GetName(), metav1.DeleteOptions{Preconditions: &metav1.Preconditions{UID: &uid}, PropagationPolicy: &foreground}); err != nil && !apierrors.IsNotFound(err) {
					t.Errorf("cleanup Knative fixture: %v", err)
					return
				}
				if err := wait.PollUntilContextTimeout(cleanupCtx, time.Second, 145*time.Second, true, func(ctx context.Context) (bool, error) {
					_, err := resource.Get(ctx, initial.GetName(), metav1.GetOptions{})
					return apierrors.IsNotFound(err), ignoreAutoscalingNotFound(err)
				}); err != nil {
					t.Errorf("Knative fixture foreground deletion incomplete: %v", err)
				} else {
					t.Log("Knative fixture foreground cleanup confirmed")
				}
			})
			req := model.AutoscalingRequest{Name: initial.GetName(), KPA: &model.KPAConfig{Metric: "rps", Target: scalingTestPtr(10.0), MaxScale: scalingTestPtr(int64(2)), WindowSeconds: scalingTestPtr(int64(6))}}
			if _, err = saveExisting("kpa", req.Name, req, true); err != nil {
				t.Fatalf("KPA dry-run: %v", err)
			}
			current, err := svc.Get(ctx, 1, namespace, "kpa", req.Name)
			if err != nil {
				t.Fatal(err)
			}
			a, _, _ := unstructured.NestedStringMap(current.Object, "spec", "template", "metadata", "annotations")
			if a[kpaPrefix+"metric"] == "rps" {
				t.Fatal("KPA dry-run persisted template annotations")
			}
			saved, err := saveExisting("kpa", req.Name, req, false)
			if err != nil {
				t.Fatalf("KPA save: %v", err)
			}
			a, _, _ = unstructured.NestedStringMap(saved.Object, "spec", "template", "metadata", "annotations")
			if a[kpaPrefix+"metric"] != "rps" || a[kpaPrefix+"target"] != "10" || a["autoops.io/preserve"] != "yes" {
				t.Fatalf("KPA annotation merge failed: %v", a)
			}
			if a[kpaPrefix+"panic-window-percentage"] != "" {
				t.Fatal("sparse KPA update pinned inherited defaults")
			}
			containers, _, _ := unstructured.NestedSlice(saved.Object, "spec", "template", "spec", "containers")
			if len(containers) != 1 || containers[0].(map[string]interface{})["image"] != image {
				t.Fatal("KPA save changed application containers")
			}
			req.ResourceVersion = initial.GetResourceVersion()
			if _, err = svc.Save(ctx, 1, namespace, "kpa", req.Name, req, false, false); !apierrors.IsConflict(err) {
				t.Fatalf("stale KPA update accepted: %v", err)
			}
			if err = deleteExisting("kpa", req.Name, true); err != nil {
				t.Fatalf("KPA reset dry-run: %v", err)
			}
			current, err = svc.Get(ctx, 1, namespace, "kpa", req.Name)
			if err != nil {
				t.Fatal(err)
			}
			a, _, _ = unstructured.NestedStringMap(current.Object, "spec", "template", "metadata", "annotations")
			if a[kpaPrefix+"metric"] != "rps" {
				t.Fatal("KPA reset dry-run persisted")
			}
			if err = deleteExisting("kpa", req.Name, false); err != nil {
				t.Fatalf("KPA reset: %v", err)
			}
			current, err = svc.Get(ctx, 1, namespace, "kpa", req.Name)
			if err != nil {
				t.Fatalf("KPA reset deleted Service: %v", err)
			}
			a, _, _ = unstructured.NestedStringMap(current.Object, "spec", "template", "metadata", "annotations")
			for _, key := range managedKPAKeys {
				if _, found := a[kpaPrefix+key]; found {
					t.Errorf("KPA reset retained %s", key)
				}
			}
			if a["autoops.io/preserve"] != "yes" {
				t.Fatal("KPA reset removed unrelated annotation")
			}
		})
	}
}
func ignoreAutoscalingNotFound(err error) error {
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}
