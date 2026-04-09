#!/bin/bash

# ========================================
# 监控告警群组 & 子规则 & 分类(Style) API 完整测试脚本
# ========================================

BASE_URL="http://172.22.107.76:8000/api/v1"
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6ODksInVzZXJuYW1lIjoiYWRtaW4iLCJuaWNrbmFtZSI6ImFkbWluIiwiaWNvbiI6Imh0dHA6Ly8xOTIuMTY4LjMuNzo4MDgwL2FwaS92MS91cGxvYWQvMjAyNTEyMTMvODYyMzI4MDAwLnBuZyIsImVtYWlsIjoiMTIzNDU2Nzg5QHFxLmNvbSIsInBob25lIjoiMTM3NTQzNTQ1MzYiLCJub3RlIjoi5ZCO56uv56CU5Y-RIiwiZXhwIjoxNzc1NzE5OTU0LCJpc3MiOiJhZG1pbiJ9.fScdJkkKnS1rhDuODHj5udUREFgnyw-4keBGnDHS1Gg"

HEADER_JSON="Content-Type: application/json"
HEADER_AUTH="Authorization: Bearer ${TOKEN}"
# DS_ID=1

# echo "========================================"
# echo "0. 创建告警分类 (POST /monitor/alert/style)"
# echo "========================================"
# CREATE_STYLE_RES=$(curl -s -X POST "${BASE_URL}/monitor/alert/style" -H "${HEADER_JSON}" -H "${HEADER_AUTH}" -d '{"name": "CPU", "description": "CPU指标告警规则"}')
# echo "新建分类结果: ${CREATE_STYLE_RES}"
# STYLE_ID=$(echo "${CREATE_STYLE_RES}" | jq -r '.data.ID // .data.id')

# CREATE_STYLE_RES2=$(curl -s -X POST "${BASE_URL}/monitor/alert/style" -H "${HEADER_JSON}" -H "${HEADER_AUTH}" -d '{"name": "Memory", "description": "内存指标告警规则"}')
# STYLE_ID2=$(echo "${CREATE_STYLE_RES2}" | jq -r '.data.ID // .data.id')

# echo "========================================"
# echo "0.1 查询告警分类列表 (GET /monitor/alert/styles)"
# echo "========================================"
# curl -s -X GET "${BASE_URL}/monitor/alert/styles" -H "${HEADER_AUTH}" | jq .

# echo ""
# echo "========================================"
# echo "1. 创建告警群组 (POST /monitor/alert/group)"
# echo "========================================"
# YAML_RULE="apiVersion: monitoring.coreos.com/v1\nkind: PrometheusRule\nmetadata:\n  name: node-cpu-usage\n  namespace: monitor\n  labels:\n    release: prometheus\nspec:\n  groups:\n    - name: node.cpu.usage.rules\n      rules: []"

# CREATE_GROUP_PAYLOAD=$(cat <<PAYLOAD
# {
#   "data_source_id": ${DS_ID},
#   "group_name": "node-system-usage",
#   "labels": "{\"cluster\": \"beijing-core\"}",
#   "rule_content": "${YAML_RULE}"
# }
# PAYLOAD
# )

# CREATE_GROUP_RES=$(curl -s -X POST "${BASE_URL}/monitor/alert/group" -H "${HEADER_JSON}" -H "${HEADER_AUTH}" -d "${CREATE_GROUP_PAYLOAD}")
# echo "告警群组创建返回结果: ${CREATE_GROUP_RES}"
# GROUP_ID=$(echo "${CREATE_GROUP_RES}" | jq -r '.data.ID // .data.id')
# echo "Group ID: ${GROUP_ID}"

# echo ""
# echo "========================================"
# echo "2. 直接添加两条子规则 (测试 Style 与 Enabled 字段) (POST /monitor/alert/rule)"
# echo "========================================"
# CREATE_RULE_PAYLOAD1=$(cat <<PAYLOAD
# {
#   "group_id": ${GROUP_ID},
#   "alert": "NodeCPUUsage",
#   "expr": "100 - (avg by (instance) (rate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100) > 90",
#   "for_duration": "5m",
#   "severity": "warning",
#   "summary": "CPU使用率高",
#   "description": "节点CPU持续超过 90%",
#   "constraints": "{\"service\": \"nginx\", \"env\": \"prod\"}",
#   "labels": "{\"team\": \"devops\"}",
#   "style": "CPU",
#   "enabled": 1
# }
# PAYLOAD
# )

# CREATE_RULE_RES1=$(curl -s -X POST "${BASE_URL}/monitor/alert/rule" -H "${HEADER_JSON}" -H "${HEADER_AUTH}" -d "${CREATE_RULE_PAYLOAD1}")
# echo "新增子规则1(CPU, 启用)返回结果: ${CREATE_RULE_RES1}"
# RULE_ID1=$(echo "${CREATE_RULE_RES1}" | jq -r '.data.ID // .data.id')
GROUP_ID=9
CREATE_RULE_PAYLOAD2=$(cat <<PAYLOAD
{
  "group_id": ${GROUP_ID},
  "alert": "NodeMemoryUsage",
  "expr": "100 - (node_memory_MemAvailable_bytes / node_memory_MemTotal_bytes) * 100 > 90",
  "for_duration": "5m",
  "severity": "warning",
  "summary": "内存使用率高",
  "description": "节点内存持续超过 90%",
  "constraints": "{\"service\": \"mysql\", \"env\": \"prod\"}",
  "labels": "{\"team\": \"devops\"}",
  "style": "Memory",
  "enabled": 1
}
PAYLOAD
)

CREATE_RULE_RES2=$(curl -s -X POST "${BASE_URL}/monitor/alert/rule" -H "${HEADER_JSON}" -H "${HEADER_AUTH}" -d "${CREATE_RULE_PAYLOAD2}")
echo "新增子规则2(Memory, 禁用)返回结果: ${CREATE_RULE_RES2}"
RULE_ID2=$(echo "${CREATE_RULE_RES2}" | jq -r '.data.ID // .data.id')

# echo ""
# echo "========================================"
# echo "3. 测试多条件分页组合查询接口 (GET /monitor/alert/rules_list)"
# echo "========================================"
# echo ">> 查询 Style=CPU 且 Enabled=1 的规则:"
# curl -s -X GET "${BASE_URL}/monitor/alert/rules_list?style=CPU&enabled=1&page=1&pageSize=10" -H "${HEADER_AUTH}" | jq .

# echo ">> 查询 Style=Memory 且 Enabled=0 的规则:"
# curl -s -X GET "${BASE_URL}/monitor/alert/rules_list?style=Memory&enabled=0&page=1&pageSize=10" -H "${HEADER_AUTH}" | jq .

# echo ""
# echo "========================================"
# echo "3.1 测试据所属群组直接拉取下属所有规则列表 (GET /monitor/alert/rules/:groupId)"
# echo "========================================"
# echo ">> 拉取 GroupID=${GROUP_ID} 的所有告警(包括已启用和未启用):"
# curl -s -X GET "${BASE_URL}/monitor/alert/rules/${GROUP_ID}?page=1&pageSize=10" -H "${HEADER_AUTH}" | jq .


# echo ""
# echo "========================================"
# echo "4. 获取刚才创建的告警群组，验证群组YAML配置是否只下发了启用的规则"
# echo "========================================"
# curl -s -X GET "${BASE_URL}/monitor/alert/group/${GROUP_ID}" -H "${HEADER_AUTH}" | jq -r '.data.rule_content'

# echo ""
# echo "========================================"
# echo "5. 修改分类 (PUT /monitor/alert/style)"
# echo "========================================"
# curl -s -X PUT "${BASE_URL}/monitor/alert/style" -H "${HEADER_JSON}" -H "${HEADER_AUTH}" -d "{\"id\": ${STYLE_ID}, \"name\": \"CPU-Metrics\", \"description\": \"CPU 相关\"}" | jq .

# echo ""
# echo "========================================"
# echo "6. 单条子规则停用测试 (将 CPU 规则 Enabled 设置为 0) (PUT /monitor/alert/rule)"
# echo "========================================"
# UPDATE_RULE_PAYLOAD=$(cat <<PAYLOAD
# {
#   "id": ${RULE_ID1},
#   "group_id": ${GROUP_ID},
#   "alert": "NodeCPUUsage Updated",
#   "expr": "100 - (avg by (instance) (rate(node_cpu_seconds_total{mode=\"idle\"}[5m])) * 100) > 90",
#   "for_duration": "3m",
#   "severity": "critical",
#   "summary": "CPU使用率高",
#   "description": "节点CPU持续超过 90%",
#   "constraints": "{\"service\": \"nginx\", \"env\": \"prod\"}",
#   "labels": "{\"team\": \"sre\"}",
#   "style": "CPU-Metrics",
#   "enabled": 0
# }
# PAYLOAD
# )
# curl -s -X PUT "${BASE_URL}/monitor/alert/rule" -H "${HEADER_JSON}" -H "${HEADER_AUTH}" -d "${UPDATE_RULE_PAYLOAD}" | jq .

# echo ">> 再次获取 Group YAML，确认未启用规则已被过滤 (此时应该为空 rules):"
# curl -s -X GET "${BASE_URL}/monitor/alert/group/${GROUP_ID}" -H "${HEADER_AUTH}" | jq -r '.data.rule_content'

# echo ""
# echo "========================================"
# echo "7. 资源清理: 删除子规则、群组、分类"
# echo "========================================"
# curl -s -X DELETE "${BASE_URL}/monitor/alert/rule/${RULE_ID1}" -H "${HEADER_AUTH}" | jq .
# curl -s -X DELETE "${BASE_URL}/monitor/alert/rule/${RULE_ID2}" -H "${HEADER_AUTH}" | jq .
# curl -s -X DELETE "${BASE_URL}/monitor/alert/group/${GROUP_ID}" -H "${HEADER_AUTH}" | jq .
# curl -s -X DELETE "${BASE_URL}/monitor/alert/style/${STYLE_ID}" -H "${HEADER_AUTH}" | jq .
# curl -s -X DELETE "${BASE_URL}/monitor/alert/style/${STYLE_ID2}" -H "${HEADER_AUTH}" | jq .

# echo ""
# echo "测试流程执行完毕！"
