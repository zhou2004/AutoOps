#!/bin/bash

# ========================================
# 监控告警规则 & 数据源 API 测试脚本
# 使用前请自行修改基础配置中的 URL 和 TOKEN
# ========================================

# --- 基础配置区 ---
BASE_URL="http://172.22.107.76:8000/api/v1" # 请修改为后端实际运行的端口
# 对于 /monitor 相关的接口由于注册在 `monitorGroup.Use(middleware.AuthMiddleware())` 下，需要有效的 JWT Token
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6ODksInVzZXJuYW1lIjoiYWRtaW4iLCJuaWNrbmFtZSI6ImFkbWluIiwiaWNvbiI6Imh0dHA6Ly8xOTIuMTY4LjMuNzo4MDgwL2FwaS92MS91cGxvYWQvMjAyNTEyMTMvODYyMzI4MDAwLnBuZyIsImVtYWlsIjoiMTIzNDU2Nzg5QHFxLmNvbSIsInBob25lIjoiMTM3NTQzNTQ1MzYiLCJub3RlIjoi5ZCO56uv56CU5Y-RIiwiZXhwIjoxNzc1NjMzMDg5LCJpc3MiOiJhZG1pbiJ9.T4HM5cVA8vpmDj6JVtwGqAApyxwnGNoo6-h9-5VcmYs"     

# 提取 Token 后请求的基础 Headers
HEADER_JSON="Content-Type: application/json"
HEADER_AUTH="Authorization: Bearer ${TOKEN}"

# echo "========================================"
# echo "1. 创建监控数据源 (POST /monitor/datasource)"
# echo "========================================"
# # 由于 k8s 规则需要基于数据源里的 config 来进行鉴权下发，因此先创建数据源
# CREATE_DS_PAYLOAD=$(cat <<EOF
# {
#   "name": "北京核心 K8s 集群 (本机测试)",
#   "type": "Prometheus",
#   "deployMethod": "Kubernetes",
#   "apiUrl": "https://192.168.0.51:6443",
#   "config": "{\"auth_type\":\"service_account\",\"k8s_api_url\":\"https://192.168.0.51:6443\",\"namespace\":\"monitor\",\"token\":\"eyJhbGciOiJSUzI1NiIsImtpZCI6IkRtejVzMmo3b3JkMFFOa0Etby1JdVRBZWZfM2MtcTZMSTExbjZJajJfOUEifQ.eyJhdWQiOlsiYXBpIiwiaXN0aW8tY2EiXSwiZXhwIjoyMDkwOTAyNzY3LCJpYXQiOjE3NzU1NDI3NjcsImlzcyI6Imh0dHBzOi8va3ViZXJuZXRlcy5kZWZhdWx0LnN2YyIsImp0aSI6ImRiNjg4NjViLTRkM2EtNDAxZC1hYzY5LWFmOTkwNGJkMTNiMSIsImt1YmVybmV0ZXMuaW8iOnsibmFtZXNwYWNlIjoibW9uaXRvciIsInNlcnZpY2VhY2NvdW50Ijp7Im5hbWUiOiJtb25pdG9yLW1hbmFnZXItc2EiLCJ1aWQiOiI4NmYwYTAyMS0wNjk1LTQ1ZTYtYmQ0OS00NjI1MTg2MTU2OGQifX0sIm5iZiI6MTc3NTU0Mjc2Nywic3ViIjoic3lzdGVtOnNlcnZpY2VhY2NvdW50Om1vbml0b3I6bW9uaXRvci1tYW5hZ2VyLXNhIn0.ez8h2RjDcMc7-OW55JWcz1h2IOMTQitEv-F8tqqIZNjV8EUbkzfU8CPU4UdqgLeZK26eh4-0Ql_I-7GgmS_Il93Dxow9jHi1ihwrQc7oao9EHCRyjEqDVUuJdECY4SKEolwupeFc_dJYQre0UNGbxPttyqFNfh5-36gekIe4rt96-5F-yDL_U6Jfv58wXEXGPl-ReAowjwHhY9djCrFYtKbA4Fs6hkmycIi_34lqn0w-9RmJ4MWss0XosUEqtPBw41EhO4cJ7xsrQgHEenQBKQOV7RGhtPfA8i8Pyd0OiYvKyb1dzXFbbB96zWOKdMKXIweFjMkD6V4cP6pg0r196A\",\"insecure_skip_tls_verify\":true}"
# }
# EOF
# )

# CREATE_DS_RES=$(curl -s -X POST "${BASE_URL}/monitor/datasource" \
#   -H "${HEADER_JSON}" \
#   -H "${HEADER_AUTH}" \
#   -d "${CREATE_DS_PAYLOAD}")

# echo "数据源返回结果: ${CREATE_DS_RES}"

# # 注意：如果安装了 jq 命令，可以通过以下方式拿到动态产生的 id
# # DS_ID=$(echo "${CREATE_DS_RES}" | jq '.data.id')
# # 此处我们假设接口返回的数据源 ID 为 1，执行时可将其手动修改为实际返回的值



# DS_ID=1

# echo ""
# echo "========================================"
# echo "2. 创建告警规则并应用到 K8s (POST /monitor/alertrule)"
# echo "========================================"
# # 下面的 YAML 代表了我们要下发的 PrometheusRule
# YAML_RULE="apiVersion: monitoring.coreos.com/v1\\nkind: PrometheusRule\\nmetadata:\\n  name: node-cpu-usage\\n  namespace: monitor\\n  labels:\\n    release: prometheus\\nspec:\\n  groups:\\n    - name: node.cpu.usage.rules\\n      rules:\\n        - alert: HighCPUUsage\\n          expr: 100 - (avg by (instance) (rate(node_cpu_seconds_total{mode=\\\"idle\\\"}[5m])) * 100) > 1\\n          for: 1m\\n          labels:\\n            severity: warning\\n          annotations:\\n            summary: \\\"CPU 使用率高\\\"\\n            description: \\\"CPU 使用率持续 > 90%。\\\"\\n        - alert: HighCPUUsageCritical\\n          expr: 100 - (avg by (instance) (rate(node_cpu_seconds_total{mode=\\\"idle\\\"}[5m])) * 100) > 95\\n          for: 3m\\n          labels:\\n            severity: critical\\n          annotations:\\n            summary: \\\"CPU 使用率高\\\"\\n            description: \\\"CPU 使用率持续 > 95%。\\\""

# CREATE_RULE_PAYLOAD=$(cat <<EOF
# {
#   "id": 2,
#   "dataSourceId": ${DS_ID},
#   "name": "test-cpu-alert",
#   "labels": {
#     "instance": "192.168.0.51:9100"
#   },
#   "ruleContent": "${YAML_RULE}",
#   "status": "inactive"
# }
# EOF
# )

# CREATE_RULE_RES=$(curl -s -X PUT "${BASE_URL}/monitor/alertrule" \
#   -H "${HEADER_JSON}" \
#   -H "${HEADER_AUTH}" \
#   -d "${CREATE_RULE_PAYLOAD}")

# echo "告警创建返回结果: ${CREATE_RULE_RES}"











# 假设返回的规则 ID 为 1
RULE_ID=2

echo ""
echo "========================================"
echo "3. 获取告警规则列表 (GET /monitor/alertrules)"
echo "========================================"
curl -s -X GET "${BASE_URL}/monitor/alertrules?page=1&pageSize=10" \
  -H "${HEADER_AUTH}" | jq . || echo "请自行察看由于无jq格式化的打印"

echo ""
echo "========================================"
echo "4. 根据 ID 获取单一条告警规则 (GET /monitor/alertrule/:id)"
echo "========================================"
curl -s -X GET "${BASE_URL}/monitor/alertrule/${RULE_ID}" \
  -H "${HEADER_AUTH}" | jq . || echo "已返回内容"

echo ""
echo "========================================"
echo "5. 删除并在 K8s 中卸载告警规则 (DELETE /monitor/alertrule/:id)"
echo "========================================"
DELETE_RULE_RES=$(curl -s -X DELETE "${BASE_URL}/monitor/alertrule/${RULE_ID}" \
  -H "${HEADER_AUTH}")

echo "删除返回结果: ${DELETE_RULE_RES}"

echo ""
echo "测试流程执行完毕！"
