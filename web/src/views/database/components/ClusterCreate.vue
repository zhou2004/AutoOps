<template>
  <el-dialog
    :model-value="modelValue"
    :title="$tr('新建 / 接管数据库集群')"
    width="900px"
    destroy-on-close
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <el-steps :active="step" finish-status="success" align-center
      ><el-step :title="$tr('基本信息')" /><el-step
        :title="$tr('节点与连接')" /><el-step :title="$tr('确认登记')"
    /></el-steps>
    <el-alert v-if="error" :title="$tr(error)" type="error" :closable="false" />
    <el-form label-position="top" class="cluster-form">
      <template v-if="step === 0"
        ><el-radio-group v-model="form.provision"
          ><el-radio-button :value="true">{{ $tr("新建集群") }}</el-radio-button
          ><el-radio-button :value="false">{{
            $tr("发现并接管")
          }}</el-radio-button></el-radio-group
        >
        <div class="form-grid">
          <el-form-item :label="$tr('集群名称')"
            ><el-input
              v-model="form.name"
              :placeholder="$tr('例如 saas-mysql，字母、数字及连字符')"
              maxlength="64" /></el-form-item
          ><el-form-item :label="$tr('业务分组')"
            ><el-tree-select
              v-model="form.groupId"
              :data="groups"
              :props="{ label: 'name', value: 'id', children: 'children' }"
              check-strictly
              filterable /></el-form-item
          ><el-form-item :label="$tr('数据库类型')"
            ><el-select
              v-model="form.engine"
              @change="form.topology = 'standalone'"
              ><el-option
                v-for="value in ['mysql', 'postgresql', 'redis']"
                :key="value"
                :value="value"
                :label="value" /></el-select></el-form-item
          ><el-form-item :label="$tr('拓扑')"
            ><el-select v-model="form.topology"
              ><el-option
                v-for="value in topologies"
                :key="value"
                :value="value"
                :label="topologyNames[value]" /></el-select></el-form-item
          ><el-form-item :label="$tr('环境')"
            ><el-select v-model="form.environment" allow-create filterable
              ><el-option
                v-for="value in ['test', 'staging', 'production']"
                :key="value"
                :value="value" /></el-select
          ></el-form-item></div
      ></template>
      <template v-if="step === 1 && form.provision"
        ><el-alert
          type="info"
          :closable="false"
          :title="
            $tr(
              '安装会先检查端口、目录、资源和时间同步；登记后需预检并提交安装任务。'
            )
          "
        /><el-form-item
          :label="
            form.topology === 'standalone'
              ? $tr('CMDB 主机')
              : $tr('三个不同故障域的 CMDB 主机')
          "
          ><el-select v-model="hostIds" multiple filterable style="width: 100%"
            ><el-option
              v-for="h in hosts"
              :key="h.id"
              :value="h.id"
              :label="`${h.hostname || h.hostName || h.name || h.id} · ${h.ipAddr || h.ip || h.privateIp || ''}`" /></el-select
        ></el-form-item>
        <div class="form-grid">
          <el-form-item :label="$tr('数据服务起始端口')"
            ><el-input-number
              v-model="port"
              :min="1024"
              :max="55530" /></el-form-item
          ><el-form-item
            :label="$tr('Redis 每实例内存配额 / 主机预检基线（MB）')"
            ><el-input-number
              v-model="form.policy.memoryMB"
              :min="32"
              :max="1048576" /></el-form-item
          ><el-form-item :label="$tr('管理密码')"
            ><el-input
              v-model="form.password"
              type="password"
              show-password
              autocomplete="new-password"
              :placeholder="$tr('留空则自动生成并加密保管')" /></el-form-item
          ><el-form-item
            v-if="form.topology === 'redis-sentinel'"
            label="Sentinel masterName"
            ><el-input
              v-model="form.policy.masterName"
              :placeholder="$tr('默认使用集群名称')"
          /></el-form-item>
        </div>
        <p v-if="form.topology === 'redis-cluster'">
          {{
            $tr(
              "将创建 9 个数据实例，每个分片的主节点与两个副本分布在三台主机。默认 noeviction，并要求至少一个有效副本。"
            )
          }}
        </p>
        <p v-if="form.topology === 'postgresql-patroni'">
          {{
            $tr(
              "默认严格同步。未验证 watchdog 与业务入口前，保持“可运维”，不标记高可用已验收。"
            )
          }}
        </p></template
      >
      <template v-if="step === 1 && !form.provision"
        ><el-alert
          type="info"
          :closable="false"
          :title="
            $tr(
              '发现仅查询拓扑和现有配置；接管后默认为观察模式，不修改账号、复制关系或选主机制。'
            )
          "
        /><el-form-item :label="$tr('已有数据源（作为种子节点）')"
          ><el-select
            v-model="sourceIds"
            multiple
            filterable
            style="width: 100%"
            ><el-option
              v-for="s in sources.filter(s => s.type === engineType)"
              :key="s.id"
              :value="s.id"
              :label="s.name" /></el-select></el-form-item
        ><el-button :loading="busy" @click="discover">{{
          $tr("只读发现")
        }}</el-button
        ><el-table v-if="discovered" :data="discovered.members"
          ><el-table-column prop="name" :label="$tr('成员')" /><el-table-column
            prop="address"
            :label="$tr('地址')"
          /><el-table-column prop="port" :label="$tr('端口')" /><el-table-column
            :label="$tr('角色')"
            ><template #default="{ row }">{{
              label(row.role)
            }}</template></el-table-column
          ><el-table-column :label="$tr('状态')"
            ><template #default="{ row }"
              ><el-tag :type="tagType(row.state)">{{
                label(row.state)
              }}</el-tag></template
            ></el-table-column
          ></el-table
        ></template
      >
      <template v-if="step === 2"
        ><el-descriptions :column="2" border
          ><el-descriptions-item :label="$tr('集群')">{{
            form.name
          }}</el-descriptions-item
          ><el-descriptions-item :label="$tr('模式')">{{
            form.provision ? $tr("新建") : $tr("只读接管")
          }}</el-descriptions-item
          ><el-descriptions-item :label="$tr('拓扑')">{{
            topologyNames[form.topology]
          }}</el-descriptions-item
          ><el-descriptions-item :label="$tr('数据实例')">{{
            members.length
          }}</el-descriptions-item></el-descriptions
        ><el-table :data="members"
          ><el-table-column :label="$tr('主机 / 数据源')"
            ><template #default="{ row }">{{
              row.hostId
                ? `CMDB #${row.hostId}`
                : $tr("数据源 #{value0}", { value0: row.sourceId })
            }}</template></el-table-column
          ><el-table-column prop="port" :label="$tr('端口')" /><el-table-column
            prop="role"
            :label="$tr('初始角色')" /><el-table-column
            prop="shard"
            :label="$tr('分片')"
        /></el-table>
        <p>
          {{
            $tr(
              "访问入口使用独立代理服务。VIP / 已有负载均衡和独立备份仓库可在具备资源后配置；安装成功不会自动标记高可用验收。"
            )
          }}
        </p></template
      >
    </el-form>
    <template #footer
      ><el-button @click="$emit('update:modelValue', false)">{{
        $tr("取消")
      }}</el-button
      ><el-button v-if="step > 0" @click="step--">{{ $tr("上一步") }}</el-button
      ><el-button v-if="step < 2" type="primary" @click="next">{{
        $tr("下一步")
      }}</el-button
      ><el-button v-else type="primary" :loading="busy" @click="create">{{
        $tr("登记集群")
      }}</el-button></template
    >
  </el-dialog>
</template>
<script setup lang="ts">
import { tr } from "@/locales";

import { ref, reactive, computed, watch } from "vue";
import cmdb from "@/api/cmdb";
import {
  dbRequest,
  label,
  tagType,
  type Cluster
} from "@/api/databasePlatform";
const props = defineProps<{ modelValue: boolean; groups: any[] }>();
const emit = defineEmits(["update:modelValue", "created"]);
const step = ref(0),
  hosts = ref<any[]>([]),
  sources = ref<any[]>([]),
  hostIds = ref<number[]>([]),
  sourceIds = ref<number[]>([]),
  port = ref(14306),
  busy = ref(false),
  error = ref(""),
  discovered = ref<Cluster>();
const form = reactive({
  name: "",
  groupId: undefined as number | undefined,
  engine: "mysql",
  topology: "standalone",
  environment: "test",
  provision: true,
  password: "",
  policy: { memoryMB: 64, masterName: "" }
});
const topologyNames: Record<string, string> = {
  get standalone() {
    return tr("单机");
  },
  "mysql-gtid": "MySQL GTID + Orchestrator",
  "postgresql-patroni": "PostgreSQL + Patroni",
  "redis-sentinel": "Redis Sentinel",
  get "redis-cluster"() {
    return tr("Redis Cluster（三主六从）");
  }
};
const topologies = computed(() => [
  "standalone",
  ...({
    mysql: ["mysql-gtid"],
    postgresql: ["postgresql-patroni"],
    redis: ["redis-sentinel", "redis-cluster"]
  }[form.engine] || [])
]);
const engineType = computed(
  () => ({ mysql: 1, postgresql: 2, redis: 3 })[form.engine]
);
const members = computed(() => {
  if (!form.provision)
    return (
      discovered.value?.members?.map(m => ({
        sourceId: m.sourceId,
        hostId: m.hostId,
        accountId: m.accountId,
        address: m.address,
        port: m.port,
        kind: m.kind,
        name: m.name
      })) || sourceIds.value.map(sourceId => ({ sourceId, kind: "data" }))
    );
  const count =
    form.topology === "standalone"
      ? 1
      : form.topology === "redis-cluster"
        ? 9
        : 3;
  return Array.from({ length: count }, (_, i) => {
    const hostIndex =
      form.topology === "redis-cluster" ? ((i % 3) + Math.floor(i / 3)) % 3 : i;
    return {
      hostId: hostIds.value[hostIndex],
      kind: "data",
      name: `data-${i + 1}`,
      port:
        port.value +
        (form.topology === "redis-cluster" ? Math.floor(i / 3) : 0),
      role:
        i === 0 || (form.topology === "redis-cluster" && i < 3)
          ? "primary"
          : "replica",
      shard: form.topology === "redis-cluster" ? `shard-${(i % 3) + 1}` : ""
    };
  });
});
function payload() {
  const policy: Record<string, any> = { memoryMB: form.policy.memoryMB };
  if (form.policy.masterName) policy.masterName = form.policy.masterName;
  return { ...form, policy, members: members.value };
}
function next() {
  error.value = "";
  if (step.value === 0 && (!form.name || !form.groupId)) {
    error.value = tr("请填写集群名称和业务分组");
    return;
  }
  if (step.value === 1) {
    if (
      form.provision &&
      hostIds.value.length !== (form.topology === "standalone" ? 1 : 3)
    ) {
      error.value = tr("请选择对应数量的不同主机");
      return;
    }
    if (!form.provision && !sourceIds.value.length) {
      error.value = tr("请选择种子数据源");
      return;
    }
  }
  step.value++;
}
async function discover() {
  busy.value = true;
  error.value = "";
  try {
    discovered.value = await dbRequest("discovery", "post", payload());
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    busy.value = false;
  }
}
async function create() {
  busy.value = true;
  error.value = "";
  try {
    const row = await dbRequest<Cluster>("clusters", "post", payload());
    emit("created", row.id);
    emit("update:modelValue", false);
    form.password = "";
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    busy.value = false;
  }
}
watch(
  () => form.engine,
  value => {
    port.value =
      { mysql: 14306, postgresql: 15433, redis: 17379 }[value] || 14306;
    discovered.value = undefined;
    sourceIds.value = [];
  }
);
watch(
  () => [sourceIds.value.join(","), form.topology, form.provision],
  () => {
    discovered.value = undefined;
  }
);
watch(
  () => props.modelValue,
  async value => {
    if (!value) return;
    step.value = 0;
    error.value = "";
    try {
      const [h, s] = await Promise.all([
        cmdb.getCmdbHostList({ page: 1, pageSize: 100 }),
        cmdb.listDatabases({ page: 1, size: 100 })
      ]);
      hosts.value = h.data?.data?.list || h.data?.data?.records || [];
      sources.value = s.data?.data?.list || [];
    } catch (e) {
      error.value = (e as Error).message;
    }
  }
);
</script>
<style scoped>
.cluster-form {
  margin-top: 24px;
}
.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0 24px;
  margin-top: 20px;
}
.el-alert {
  margin: 20px 0;
}
.el-select,
.el-tree-select {
  width: 100%;
}
p {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  line-height: 1.8;
}
.el-descriptions {
  margin-bottom: 20px;
}
</style>
