---
name: k8s-crd
description: 当用户要求增加、修改或调试 Kubernetes 自定义资源 (CRD & CR) 相关的后端接口和服务逻辑时使用此技能。
---
# Kubernetes CRD 与自定义资源管理 (CR) 开发规范

## 1. 核心概念与技术选型
在 AutoOps 运维平台项目中，我们需要操作集群中的自定义资源定义 (CRD) 及其具体实例 (CR, Custom Resource)。
- **技术局限性**：由于 CRD 类型不是 Kubernetes 原生内置对象（不可提前通过 Golang Struct 声明），无法直接通过 `client-go` 的常规 `Clientset` 操作。
- **标准解决方案**：必须使用 `client-go/dynamic` 提供的 `dynamic.Interface` 与 `unstructured.Unstructured`（无类型数据结构）对 CR 做动态的增删改查。

## 2. GVR (GroupVersionResource) 发现机制
自定义资源的操作依赖于正确的 GVR：
- **解析资源名与组**：前端传入的 `crdName`（例如: `prometheusrules.monitoring.coreos.com`），需根据第一个 `.` 切分为 resource (`prometheusrules`) 和 group (`monitoring.coreos.com`)。
- **版本推断**：由于版本(Version)动态多变，系统采用 `Discovery Client` (`clientset.Discovery().ServerGroups()`) 进行 API 组发现，找到对应的 `PreferredVersion.Version`。如果遇到异常或未找到，兜底降级使用 `v1` 或 `v1beta1`。

## 3. 核心 API 路由与作用域
- **CRD 定义 (Cluster 级别)**：
  `CRD` 本身是集群作用域（不存在命名空间隔离），所以在拉取定义时无需传入 Namespace 参数。
  - `GET /k8s/cluster/:id/crds/groups`
  - `GET /k8s/cluster/:id/crds`
- **CR 实例 (Namespace 级别或 Cluster 级别)**：
  `CR` 资源的具体实例可能带有 Namespace。系统需根据传入的 `namespaceName` 是否为 `""` 或者是 `"all"` 来判断是否需要调用 `.Namespace(namespaceName)`。
  - `GET /k8s/cluster/:id/namespaces/:namespaceName/crds/:crdName/resources`
  - `GET /k8s/cluster/:id/namespaces/:namespaceName/crds/:crdName/resources/:crName`
  - `POST /k8s/cluster/:id/namespaces/:namespaceName/crds/:crdName/resources`
  - `DELETE /k8s/cluster/:id/namespaces/:namespaceName/crds/:crdName/resources/:crName`
  - `GET /k8s/cluster/:id/namespaces/:namespaceName/crds/:crdName/resources/:crName/yaml`
  - `PUT /k8s/cluster/:id/namespaces/:namespaceName/crds/:crdName/resources/:crName/yaml`

## 4. 易错点与开发踩坑指南 (坑点清单)
1. **YAML 提交报 "Object 'Kind' is missing" 错误**：
   - **现象**：前端在配置/编辑 YAML 创建实例时，通常包裹在一层 JSON 中：`{"yamlContent": "apiVersion: xxx\nkind: xxx"}`。
   - **解决**：K8s API 需要的是直接映射出的 `map[string]interface{}`。后端接收到字典对象后，必须检测是否带有 `yamlContent` 键。如果是，需要使用 `yaml.Unmarshal` 提取出字符串文本并解析为新的 `map[string]interface{}`，然后再裹上 `&unstructured.Unstructured{Object: finalData}` 去请求集群。
2. **Get / List 列表庞大与多余元数据**：
   - 转换对象并返回给前端前，**务必移除 `metadata.managedFields`** 等无用的状态元数据。
   - 否则 `YAML` 将极其庞大，严重影响前端渲染与解析性能。可通过 `unstructured.RemoveNestedField(item.Object, "metadata", "managedFields")` 处理。
3. **Update (PUT) 更新操作**：
   - 更新 CR 实例必须显式传递当前的 `resourceVersion`。如果前端通过纯 YAML 内容更新，后端必须先进行一次 `Get` 请求，将查出的 `existing.GetResourceVersion()` 通过 `SetResourceVersion()` 塞回新的 `unstructured` 结构体，然后再执行 `.Update()`。

## 5. 代码层面的职责划分
- **Controller (`k8sCRDController`)**：负责解析路由中的 `clusterId`、`namespaceName`、`crdName` 等，统一包装处理分页相关的参数，并向外暴露 RESTful 接口。
- **Service (`k8sCRDService`)**：负责衔接 Dao 的 Client，处理核心业务逻辑，包括 GVR 解析、Namespace 过滤、YAML/JSON 数据转换处理，更新时做 ResourceVersion 注入。
- **Dao (`k8sCRDDao`)**：专注基础连接库的生成，负责将数据库里的集群凭证 (Credential) 实例化为 `DynamicClient` 或 `Clientset` 供上层使用。
