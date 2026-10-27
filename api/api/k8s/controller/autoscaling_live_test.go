package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	k8smodel "dodevops-api/api/k8s/model"
	systemmodel "dodevops-api/api/system/model"
	"dodevops-api/common/constant"
	"dodevops-api/middleware"
	appdb "dodevops-api/pkg/db"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// This opt-in test uses the production controller, namespace middleware, DAO and
// Kubernetes clients, but an isolated SQLite DB and in-process HTTP recorder.
// User identity is injected in this test router only; it does not test JWT/Redis
// login and never creates a production token or edits the running AutoOps DB.
// Requests are reads/rejections unless AUTOOPS_AUTOSCALING_HTTP_MUTATE=1.
// With that separate switch, a fresh reserved test namespace and zero-replica
// Deployment exercise HTTP HPA/VPA CRUD and a Knative Service fixture covers
// KPA updates/reset. Run this sequentially with the
// service suite when using the same AUTOOPS_AUTOSCALING_LIVE_NAMESPACE.
func TestAutoscalingHTTPLive(t *testing.T) {
	configPath := os.Getenv("AUTOOPS_AUTOSCALING_LIVE_KUBECONFIG")
	if configPath == "" {
		t.Skip("set AUTOOPS_AUTOSCALING_LIVE_KUBECONFIG to opt into real-cluster HTTP integration")
	}
	credential, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal("unable to read isolated kubeconfig")
	}
	fixture, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "autoscaling-http.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := fixture.DB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	if err = fixture.AutoMigrate(&k8smodel.KubeCluster{}, &k8smodel.K8sPermission{}, &k8smodel.K8sGroupPermission{}, &k8smodel.K8sUserGroup{}, &k8smodel.K8sUserGroupMember{}, &k8smodel.K8sRbacRole{}, &k8smodel.K8sRbacBinding{}); err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{"CREATE TABLE sys_role (id INTEGER PRIMARY KEY, role_key TEXT, status INTEGER)", "CREATE TABLE sys_admin_role (admin_id INTEGER, role_id INTEGER)", "INSERT INTO sys_role VALUES(1,'admin',1)", "INSERT INTO sys_admin_role VALUES(1,1)"} {
		if err = fixture.Exec(stmt).Error; err != nil {
			t.Fatal(err)
		}
	}
	cluster := k8smodel.KubeCluster{ID: 1, Name: "isolated-live-http", Version: "test", Status: 2, Credential: string(credential)}
	if err = fixture.Create(&cluster).Error; err != nil {
		t.Fatal("unable to store isolated test credential")
	}
	addRole := func(user uint, namespace, group, resource string, verbs ...string) {
		t.Helper()
		rules, _ := json.Marshal([]k8smodel.K8sRule{{APIGroups: []string{group}, Resources: []string{resource}, Verbs: verbs}})
		role := k8smodel.K8sRbacRole{ClusterID: 1, Namespace: namespace, Name: fmt.Sprintf("test-user-%d", user), Rules: string(rules)}
		if err = fixture.Create(&role).Error; err != nil {
			t.Fatal(err)
		}
		if err = fixture.Create(&k8smodel.K8sRbacBinding{ClusterID: 1, Namespace: namespace, RoleID: role.ID, SubjectType: "User", SubjectID: user}).Error; err != nil {
			t.Fatal(err)
		}
	}
	addRole(2, "default", "autoscaling", "horizontalpodautoscalers", "get", "list")
	addRole(3, "default", "", "services", "get", "list", "update")
	addRole(4, "other-namespace", "autoscaling", "horizontalpodautoscalers", "*")
	previous := appdb.Db
	appdb.Db = fixture
	t.Cleanup(func() { appdb.Db = previous })
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		if value := c.GetHeader("X-Isolated-Test-User"); value != "" {
			id, _ := strconv.ParseUint(value, 10, 32)
			c.Set(constant.ContextKeyUserObj, &systemmodel.JwtAdmin{ID: uint(id), Username: "isolated-live-test"})
		}
		c.Next()
	})
	ctrl := NewK8sAutoscalingController(fixture)
	guard := middleware.NewK8sPermissionMiddleware()
	base := "/api/v1/k8s/cluster/:id/namespaces/:namespaceName/autoscaling"
	router.GET(base+"/capabilities", guard.CheckAutoscalingPermission(), ctrl.Handle)
	for _, path := range []string{base + "/:kind", base + "/:kind/:name"} {
		router.GET(path, guard.CheckAutoscalingPermission(), ctrl.Handle)
	}
	router.POST(base+"/:kind", guard.CheckAutoscalingPermission(), ctrl.Handle)
	router.PUT(base+"/:kind/:name", guard.CheckAutoscalingPermission(), ctrl.Handle)
	router.DELETE(base+"/:kind/:name", guard.CheckAutoscalingPermission(), ctrl.Handle)
	call := func(t *testing.T, method, path, body string, user uint) (int, json.RawMessage) {
		t.Helper()
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		if user != 0 {
			req.Header.Set("X-Isolated-Test-User", strconv.Itoa(int(user)))
		}
		writer := httptest.NewRecorder()
		router.ServeHTTP(writer, req)
		if writer.Code != http.StatusOK {
			t.Fatalf("unexpected transport status %d", writer.Code)
		}
		var result struct {
			Code    int             `json:"code"`
			Message string          `json:"message"`
			Data    json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(writer.Body.Bytes(), &result); err != nil {
			t.Fatal("response is not API envelope")
		}
		// Do not log raw response bodies: Kubernetes resources can contain data.
		return result.Code, result.Data
	}
	path := "/api/v1/k8s/cluster/1/namespaces/default/autoscaling"
	tests := []struct {
		name, method, path, body string
		user                     uint
		code                     int
	}{
		{"unauthenticated rejected", "GET", path + "/hpa", "", 0, 401},
		{"admin real HPA list", "GET", path + "/hpa", "", 1, 200},
		{"HPA readonly list", "GET", path + "/hpa", "", 2, 200},
		{"readonly create rejected", "POST", path + "/hpa", `{}`, 2, 403},
		{"readonly delete rejected", "DELETE", path + "/hpa/never-created?resourceVersion=1", "", 2, 403},
		{"core Service cannot edit Knative Service", "PUT", path + "/kpa/never-created", `{}`, 3, 403},
		{"namespace isolation", "GET", path + "/hpa", "", 4, 403},
		{"invalid resource kind", "GET", path + "/unknown", "", 1, 400},
		{"invalid cluster ID", "GET", "/api/v1/k8s/cluster/invalid/namespaces/default/autoscaling/hpa", "", 1, 400},
		{"ambiguous dry-run rejected", "POST", path + "/hpa?dryRun=TRUE", `{}`, 1, 400},
		{"unknown JSON field rejected", "POST", path + "/hpa?dryRun=true", `{"name":"invalid","unknown":true}`, 1, 400},
		{"multiple JSON objects rejected", "POST", path + "/hpa?dryRun=true", `{} {}`, 1, 400},
		{"KPA create rejected", "POST", path + "/kpa?dryRun=true", `{"name":"never-created","kpa":{}}`, 1, 400},
		{"missing version rejected", "PUT", path + "/hpa/never-created?dryRun=true", `{"name":"never-created"}`, 1, 400},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := call(t, tc.method, tc.path, tc.body, tc.user)
			if got != tc.code {
				t.Fatalf("business status=%d want=%d", got, tc.code)
			}
		})
	}
	t.Run("capabilities_and_button_permissions", func(t *testing.T) {
		code, data := call(t, "GET", path+"/capabilities", "", 2)
		if code != 200 {
			t.Fatalf("capabilities code=%d", code)
		}
		var result struct {
			Permissions  map[string]map[string]bool `json:"permissions"`
			Capabilities map[string]struct {
				Available bool `json:"available"`
			} `json:"capabilities"`
		}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatal(err)
		}
		if !result.Permissions["hpa"]["list"] || result.Permissions["hpa"]["delete"] || result.Permissions["kpa"]["update"] {
			t.Fatal("button permissions do not match exact grants")
		}
		if !result.Capabilities["hpa"].Available || !result.Capabilities["metrics"].Available {
			t.Fatal("real cluster HPA/metrics capability unavailable")
		}
	})
	if os.Getenv("AUTOOPS_AUTOSCALING_HTTP_MUTATE") == "1" {
		t.Run("HTTP_autoscaler_real_CRUD", func(t *testing.T) {
			namespace := os.Getenv("AUTOOPS_AUTOSCALING_LIVE_NAMESPACE")
			if !strings.HasPrefix(namespace, "autoops-api-e2e-") {
				t.Fatal("HTTP mutation test requires a reserved autoops-api-e2e- namespace")
			}
			image := os.Getenv("AUTOOPS_AUTOSCALING_LIVE_IMAGE")
			if image == "" {
				t.Fatal("HTTP mutation test requires a fixture image")
			}
			cfg, err := clientcmd.BuildConfigFromFlags("", configPath)
			if err != nil {
				t.Fatal("unable to parse isolated kubeconfig")
			}
			cfg.Timeout = 25 * time.Second
			client, err := kubernetes.NewForConfig(cfg)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
			defer cancel()
			ns, err := client.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace, Labels: map[string]string{"autoops.io/integration-test": "autoscaling-http"}}}, metav1.CreateOptions{})
			if err != nil {
				t.Fatalf("create fresh HTTP fixture namespace: %v", err)
			}
			t.Cleanup(func() {
				cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 180*time.Second)
				defer cleanupCancel()
				uid := ns.UID
				if err := client.CoreV1().Namespaces().Delete(cleanupCtx, namespace, metav1.DeleteOptions{Preconditions: &metav1.Preconditions{UID: &uid}}); err != nil && !apierrors.IsNotFound(err) {
					t.Errorf("HTTP namespace cleanup: %v", err)
					return
				}
				err := wait.PollUntilContextTimeout(cleanupCtx, time.Second, 175*time.Second, true, func(ctx context.Context) (bool, error) {
					_, err := client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
					if apierrors.IsNotFound(err) {
						return true, nil
					}
					return false, err
				})
				if err != nil {
					t.Errorf("HTTP namespace deletion incomplete: %v", err)
				} else {
					t.Logf("HTTP cleanup confirmed: %s removed", namespace)
				}
			})
			zero := int32(0)
			_, err = client.AppsV1().Deployments(namespace).Create(ctx, &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: "http-target"}, Spec: appsv1.DeploymentSpec{Replicas: &zero, Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "http-target"}}, Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "http-target"}}, Spec: corev1.PodSpec{NodeSelector: map[string]string{"kubernetes.io/role": "node"}, Containers: []corev1.Container{{Name: "fixture", Image: image}}}}}}, metav1.CreateOptions{})
			if err != nil {
				t.Fatalf("HTTP Deployment fixture: %v", err)
			}
			resourceName := "http-hpa"
			prefix := "/api/v1/k8s/cluster/1/namespaces/" + namespace + "/autoscaling/hpa"
			input := map[string]interface{}{"name": "http-hpa", "hpa": map[string]interface{}{"scaleTargetRef": map[string]interface{}{"apiVersion": "apps/v1", "kind": "Deployment", "name": "http-target"}, "minReplicas": 1, "maxReplicas": 3, "metrics": []interface{}{map[string]interface{}{"type": "Resource", "resource": map[string]interface{}{"name": "cpu", "target": map[string]interface{}{"type": "Utilization", "averageUtilization": 70}}}}}}
			encode := func() string {
				data, err := json.Marshal(input)
				if err != nil {
					t.Fatal(err)
				}
				return string(data)
			}
			expect := func(method, url, body string, want int) json.RawMessage {
				code, data := call(t, method, url, body, 1)
				if code != want {
					t.Fatalf("%s expected business code %d got %d", method, want, code)
				}
				return data
			}
			expect("POST", prefix+"?dryRun=true", encode(), 200)
			expect("GET", prefix+"/"+resourceName, "", 404)
			expect("POST", prefix, encode(), 200)
			getVersion := func() string {
				data := expect("GET", prefix+"/"+resourceName, "", 200)
				var row struct {
					Metadata struct {
						ResourceVersion string `json:"resourceVersion"`
					} `json:"metadata"`
				}
				if err := json.Unmarshal(data, &row); err != nil {
					t.Fatal(err)
				}
				return row.Metadata.ResourceVersion
			}
			input["resourceVersion"] = "stale-version"
			expect("PUT", prefix+"/"+resourceName, encode(), 409)
			expect("DELETE", prefix+"/http-hpa?resourceVersion=stale-version", "", 409)
			input["hpa"].(map[string]interface{})["maxReplicas"] = 4
			mutateFresh := func(method string, dry bool) {
				err := wait.PollUntilContextTimeout(ctx, 100*time.Millisecond, 8*time.Second, true, func(context.Context) (bool, error) {
					version := getVersion()
					input["resourceVersion"] = version
					suffix := "/" + resourceName + "?dryRun=" + strconv.FormatBool(dry)
					body := encode()
					if method == "DELETE" {
						suffix += "&resourceVersion=" + version
						body = ""
					}
					code, _ := call(t, method, prefix+suffix, body, 1)
					if code == 409 {
						return false, nil
					}
					if code != 200 {
						return false, fmt.Errorf("%s business status %d", method, code)
					}
					return true, nil
				})
				if err != nil {
					t.Fatal(err)
				}
			}
			mutateFresh("PUT", true)
			readMaximum := func() int {
				data := expect("GET", prefix+"/"+resourceName, "", 200)
				var row struct {
					Spec struct {
						MaxReplicas int `json:"maxReplicas"`
					} `json:"spec"`
				}
				if err := json.Unmarshal(data, &row); err != nil {
					t.Fatal(err)
				}
				return row.Spec.MaxReplicas
			}
			if readMaximum() != 3 {
				t.Fatal("HTTP dry-run update persisted")
			}
			mutateFresh("PUT", false)
			if readMaximum() != 4 {
				t.Fatal("HTTP update did not persist")
			}
			mutateFresh("DELETE", true)
			expect("GET", prefix+"/"+resourceName, "", 200)
			mutateFresh("DELETE", false)
			expect("GET", prefix+"/"+resourceName, "", 404)

			t.Log("HPA HTTP create/update/delete, all dry-runs and conflict envelopes passed")
			enabled := func(kind string) bool {
				value := os.Getenv("AUTOOPS_AUTOSCALING_LIVE_KINDS")
				return value == "" || strings.Contains(","+value+",", ","+kind+",")
			}
			if enabled("vpa") {
				resourceName = "http-vpa"
				prefix = strings.TrimSuffix(prefix, "/hpa") + "/vpa"
				policy := map[string]interface{}{"containerName": "*", "controlledValues": "RequestsOnly", "minAllowed": map[string]interface{}{"cpu": "25m"}, "maxAllowed": map[string]interface{}{"cpu": "200m"}}
				input = map[string]interface{}{"name": resourceName, "vpa": map[string]interface{}{"targetRef": map[string]interface{}{"apiVersion": "apps/v1", "kind": "Deployment", "name": "http-target"}, "updatePolicy": map[string]interface{}{"updateMode": "Off"}, "resourcePolicy": map[string]interface{}{"containerPolicies": []interface{}{policy}}}}
				expect("POST", prefix+"?dryRun=true", encode(), 200)
				expect("GET", prefix+"/"+resourceName, "", 404)
				expect("POST", prefix, encode(), 200)
				policy["maxAllowed"].(map[string]interface{})["cpu"] = "300m"
				readCPU := func() string {
					data := expect("GET", prefix+"/"+resourceName, "", 200)
					var row unstructured.Unstructured
					if err := json.Unmarshal(data, &row.Object); err != nil {
						t.Fatal(err)
					}
					policies, _, _ := unstructured.NestedSlice(row.Object, "spec", "resourcePolicy", "containerPolicies")
					return policies[0].(map[string]interface{})["maxAllowed"].(map[string]interface{})["cpu"].(string)
				}
				mutateFresh("PUT", true)
				if readCPU() != "200m" {
					t.Fatal("VPA HTTP dry-run update persisted")
				}
				mutateFresh("PUT", false)
				if readCPU() != "300m" {
					t.Fatal("VPA HTTP JSON resourcePolicy did not persist")
				}
				mutateFresh("DELETE", true)
				expect("GET", prefix+"/"+resourceName, "", 200)
				mutateFresh("DELETE", false)
				expect("GET", prefix+"/"+resourceName, "", 404)
				t.Log("VPA HTTP JSON policy create/update/delete and all dry-runs passed")
			}
			if enabled("kpa") {
				resourceName = "http-knative"
				prefix = "/api/v1/k8s/cluster/1/namespaces/" + namespace + "/autoscaling/kpa"
				dyn, err := dynamic.NewForConfig(cfg)
				if err != nil {
					t.Fatal(err)
				}
				annotations := map[string]interface{}{"autoscaling.knative.dev/class": "kpa.autoscaling.knative.dev", "autoscaling.knative.dev/min-scale": "0", "autoscaling.knative.dev/max-scale": "1", "autoops.io/preserve": "yes"}
				container := map[string]interface{}{"name": "http", "image": image, "ports": []interface{}{map[string]interface{}{"containerPort": int64(8080)}}, "resources": map[string]interface{}{"requests": map[string]interface{}{"cpu": "25m", "memory": "32Mi"}, "limits": map[string]interface{}{"cpu": "100m", "memory": "128Mi"}}}
				if command := os.Getenv("AUTOOPS_AUTOSCALING_LIVE_HTTP_COMMAND_JSON"); command != "" {
					var args []interface{}
					if err := json.Unmarshal([]byte(command), &args); err != nil {
						t.Fatal(err)
					}
					container["command"] = args
				}
				fixture := &unstructured.Unstructured{Object: map[string]interface{}{"apiVersion": "serving.knative.dev/v1", "kind": "Service", "metadata": map[string]interface{}{"name": resourceName, "namespace": namespace, "labels": map[string]interface{}{"networking.knative.dev/visibility": "cluster-local"}}, "spec": map[string]interface{}{"template": map[string]interface{}{"metadata": map[string]interface{}{"annotations": annotations}, "spec": map[string]interface{}{"nodeSelector": map[string]interface{}{"kubernetes.io/role": "node"}, "containers": []interface{}{container}}}}}}
				resource := dyn.Resource(schema.GroupVersionResource{Group: "serving.knative.dev", Version: "v1", Resource: "services"}).Namespace(namespace)
				created, err := resource.Create(ctx, fixture, metav1.CreateOptions{})
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() {
					cleanupCtx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
					defer cancel()
					uid := created.GetUID()
					foreground := metav1.DeletePropagationForeground
					if err := resource.Delete(cleanupCtx, created.GetName(), metav1.DeleteOptions{Preconditions: &metav1.Preconditions{UID: &uid}, PropagationPolicy: &foreground}); err != nil && !apierrors.IsNotFound(err) {
						t.Errorf("HTTP Knative fixture cleanup: %v", err)
						return
					}
					if err := wait.PollUntilContextTimeout(cleanupCtx, time.Second, 145*time.Second, true, func(ctx context.Context) (bool, error) {
						_, err := resource.Get(ctx, created.GetName(), metav1.GetOptions{})
						if apierrors.IsNotFound(err) {
							return true, nil
						}
						return false, err
					}); err != nil {
						t.Errorf("HTTP Knative fixture foreground deletion incomplete: %v", err)
					} else {
						t.Log("HTTP Knative fixture foreground cleanup confirmed")
					}
				})
				input = map[string]interface{}{"name": resourceName, "kpa": map[string]interface{}{"metric": "rps", "target": 9, "maxScale": 2, "windowSeconds": 6}}
				readAnnotations := func() map[string]string {
					data := expect("GET", prefix+"/"+resourceName, "", 200)
					var row unstructured.Unstructured
					if err := json.Unmarshal(data, &row.Object); err != nil {
						t.Fatal(err)
					}
					a, _, _ := unstructured.NestedStringMap(row.Object, "spec", "template", "metadata", "annotations")
					return a
				}
				mutateFresh("PUT", true)
				if readAnnotations()["autoscaling.knative.dev/metric"] == "rps" {
					t.Fatal("KPA HTTP dry-run persisted")
				}
				mutateFresh("PUT", false)
				a := readAnnotations()
				if a["autoscaling.knative.dev/metric"] != "rps" || a["autoscaling.knative.dev/target"] != "9" || a["autoops.io/preserve"] != "yes" {
					t.Fatal("KPA HTTP JSON annotations not persisted/preserved")
				}
				mutateFresh("DELETE", true)
				if readAnnotations()["autoscaling.knative.dev/metric"] != "rps" {
					t.Fatal("KPA HTTP reset dry-run persisted")
				}
				mutateFresh("DELETE", false)
				a = readAnnotations()
				for _, key := range []string{"class", "metric", "target", "min-scale", "max-scale", "window", "panic-window-percentage", "panic-threshold-percentage", "target-utilization-percentage", "scale-down-delay"} {
					if _, found := a["autoscaling.knative.dev/"+key]; found {
						t.Errorf("KPA HTTP reset kept managed annotation %s", key)
					}
				}
				if a["autoops.io/preserve"] != "yes" {
					t.Fatal("KPA HTTP reset removed unrelated annotation")
				}
				t.Log("KPA HTTP JSON update/dry-run/reset passed; Service retained with unrelated annotation")
			}
		})
	}

}
