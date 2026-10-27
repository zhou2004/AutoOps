# K8s 存储管理高阶功能测试指南

## 测试环境准备

### 1. 创建 test_pv 命名空间

在 K8s 存储管理页面：
1. 进入「K8s 管理」→「命名空间」页面
2. 选择已连接的集群
3. 点击「创建命名空间」
4. 输入名称: `test_pv`
5. 点击确认

或通过 API 调用：
```bash
curl -X POST http://localhost:8080/api/v1/k8s/cluster/{clusterId}/namespaces \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{"name": "test_pv"}'
```

---

## 功能测试用例

### 用例 1: PVC 扩容测试

**步骤：**
1. 进入「K8s 存储管理」页面
2. 选择集群和 `test_pv` 命名空间
3. 在 PVC 标签页，先创建一个测试 PVC：
   - 点击「创建 PVC」
   - YAML 内容：
   ```yaml
   apiVersion: v1
   kind: PersistentVolumeClaim
   metadata:
     name: test-expand-pvc
     namespace: test_pv
   spec:
     accessModes:
       - ReadWriteOnce
     resources:
       requests:
         storage: 1Gi
     storageClassName: standard
   ```
4. 等待 PVC 状态变为 Bound
5. 点击 PVC 行的「扩容」按钮（橙色图标）
6. 输入新容量: `2Gi`
7. 点击「确认扩容」
8. **预期结果**: 提示"PVC 扩容请求已提交"，PVC 容量更新为 2Gi

### 用例 2: PVC 克隆/备份测试

**步骤：**
1. 在 PVC 列表中，找到 `test-expand-pvc`
2. 点击「备份/克隆」按钮（灰色图标）
3. 在弹出的对话框中：
   - 目标 PVC 名称: `test-expand-pvc-backup`
   - 目标命名空间: `test_pv`（默认）
   - 新容量: 留空（使用原容量）
4. 点击「确认克隆」
5. **预期结果**: 提示"PVC 克隆请求已提交"，列表中新增 `test-expand-pvc-backup`

### 用例 3: StorageClass 设为默认测试

**步骤：**
1. 切换到 StorageClass 标签页
2. 找到目标 StorageClass（如 `standard`）
3. 点击「设为默认」按钮（绿色图标）
4. 确认操作
5. **预期结果**: 提示"存储类已设为默认"，其他已默认的 SC 被取消

### 用例 4: 存储统计测试

通过 API 验证：
```bash
curl http://localhost:8080/api/v1/k8s/cluster/{clusterId}/storage/stats \
  -H "Authorization: Bearer {token}"
```

**预期响应：**
```json
{
  "code": 200,
  "data": {
    "totalPVCs": 2,
    "totalPVs": 1,
    "totalStorage": "2Gi",
    "usedStorage": "1Gi",
    "availableStorage": "1Gi",
    "storageClasses": ["standard"],
    "defaultStorageClass": "standard"
  }
}
```

---

## 新增 API 端点汇总

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/k8s/cluster/:id/namespaces/:ns/pvcs/:name/expand` | PVC 扩容 |
| POST | `/k8s/cluster/:id/namespaces/:ns/pvcs/:name/clone` | PVC 克隆（备份） |
| POST | `/k8s/cluster/:id/storageclasses/:name/set-default` | 设置默认 SC |
| GET | `/k8s/cluster/:id/storage/stats` | 存储统计 |

---

## 新增前端功能

| 功能 | 位置 | 说明 |
|------|------|------|
| PVC 扩容 | PVC 操作列 → 橙色按钮 | 输入新容量即可在线扩容 |
| PVC 克隆 | PVC 操作列 → 灰色按钮 | 创建同规格 PVC 副本 |
| 设置默认 SC | SC 操作列 → 绿色按钮 | 一键设置/取消默认存储类 |
| 存储统计 | API only | 通过 API 获取集群存储概览 |
