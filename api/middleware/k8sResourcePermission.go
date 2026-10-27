package middleware

import (
	"bytes"
	"dodevops-api/pkg/jwt"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func canonicalK8sResource(resource string) string {
	aliases := map[string]string{"deployment": "deployments", "statefulset": "statefulsets", "daemonset": "daemonsets", "job": "jobs", "cronjob": "cronjobs", "pod": "pods", "pv": "persistentvolumes", "pvs": "persistentvolumes", "pvc": "persistentvolumeclaims", "pvcs": "persistentvolumeclaims", "crds": "customresourcedefinitions"}
	if canonical, ok := aliases[resource]; ok {
		return canonical
	}
	return resource
}
func k8sAPIGroup(resource string) string {
	switch strings.Split(resource, "/")[0] {
	case "deployments", "statefulsets", "daemonsets", "replicasets":
		return "apps"
	case "jobs", "cronjobs":
		return "batch"
	case "ingresses", "networkpolicies":
		return "networking.k8s.io"
	case "storageclasses", "csidrivers", "volumeattachments":
		return "storage.k8s.io"
	case "customresourcedefinitions":
		return "apiextensions.k8s.io"
	}
	return ""
}

// Resource checks are shared by read/write/cluster routes. Transport GET does not imply read-only.
func (m *K8sPermissionMiddleware) checkResourcePermission() gin.HandlerFunc {
	return func(c *gin.Context) {
		user, err := jwt.GetAdmin(c)
		if err != nil {
			denyAuth(c, 401, "未授权")
			return
		}
		if admin, err := m.permDao.IsAdmin(user.ID); err == nil && admin {
			c.Next()
			return
		}
		id := c.Param("id")
		if id == "" {
			id = c.Param("clusterId")
		}
		cluster, err := strconv.ParseUint(id, 10, 32)
		if err != nil || cluster == 0 {
			denyAuth(c, 400, "无效的集群 ID")
			return
		}
		ns := c.Param("namespaceName")
		if ns == "all" || ns == "*" {
			denyAuth(c, 403, "请选择已授权的单个命名空间")
			return
		}
		path := c.FullPath()
		// Cluster selectors/overview and namespace list are filtered by existing controllers.
		if c.Request.Method == http.MethodGet && (strings.HasSuffix(path, "/cluster/:id") || strings.HasSuffix(path, "/cluster/:id/status") || strings.HasSuffix(path, "/cluster/:id/detail") || strings.HasSuffix(path, "/cluster/:id/namespaces")) {
			allowed, _ := m.hasAnyClusterPermission(user.ID, id)
			if !allowed {
				denyAuth(c, 403, "没有该集群的访问权限")
				return
			}
			c.Next()
			return
		}
		resource, verb := m.getResourceAndVerbFromCtx(c)
		resource = canonicalK8sResource(resource)
		if strings.Contains(path, "/crds/:crdName/resources") {
			parts := strings.SplitN(c.Param("crdName"), ".", 2)
			if len(parts) != 2 {
				denyAuth(c, 400, "CRD 名称无效")
				return
			}
			if c.Request.Method == "GET" {
				verb = "list"
				if c.Param("crName") != "" {
					verb = "get"
				}
			}
			if !m.autoscalingAccess(user.ID, uint(cluster), ns)(parts[1], parts[0], verb) {
				denyAuth(c, 403, "没有该自定义资源的授权")
				return
			}
			c.Next()
			return
		}

		if strings.HasSuffix(path, "/workload-yaml") && c.Request.Method == http.MethodPut {
			data, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, 2<<20))
			if err != nil {
				denyAuth(c, 400, "YAML 请求过大")
				return
			}
			c.Request.Body = io.NopCloser(bytes.NewReader(data))
			var body struct {
				WorkloadType string `json:"workloadType"`
			}
			if json.Unmarshal(data, &body) != nil {
				denyAuth(c, 400, "请求格式无效")
				return
			}
			resource = canonicalK8sResource(strings.ToLower(body.WorkloadType))
		}
		allowed := m.autoscalingAccess(user.ID, uint(cluster), ns)
		check := func(res, v string) bool { return allowed(k8sAPIGroup(res), res, v) }
		if resource == "workloads" {
			kind := canonicalK8sResource(strings.ToLower(c.DefaultQuery("type", "all")))
			if kind == "all" {
				// Keep partial grants useful without leaking the other workload kinds.
				kinds := []string{}
				for _, r := range []string{"deployments", "statefulsets", "daemonsets", "jobs", "cronjobs", "pods"} {
					if check(r, "list") {
						kinds = append(kinds, r)
					}
				}
				if len(kinds) == 0 {
					denyAuth(c, 403, "没有工作负载查看权限")
					return
				}
				c.Set("k8s_workload_kinds", kinds)
				c.Next()
				return
			}
			resource = kind
		}
		if resource == "" || !check(resource, verb) {
			denyAuth(c, 403, "没有该资源及操作的授权")
			return
		}
		if strings.HasSuffix(path, "/drain") && (!check("pods", "list") || !check("pods", "delete")) {
			denyAuth(c, 403, "排空节点需要集群级 Pod 列表及删除权限")
			return
		}
		c.Next()
	}
}
