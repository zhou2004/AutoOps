<script setup lang="ts">
import * as OperationIcons from "@element-plus/icons-vue";
import { onMounted, ref } from "vue";
import { ElMessage } from "element-plus";
import { getRun, getRuns, statusText, type AIRun } from "@/api/aiops";
import { locale, tr, localizeError } from "@/locales";
import "./style.css";
const rows = ref<AIRun[]>([]),
  total = ref(0),
  page = ref(1),
  loading = ref(false),
  error = ref("");
const detail = ref<AIRun>(),
  open = ref(false),
  detailLoading = ref(false);
function date(value: string | null) {
  return value ? new Date(value).toLocaleString(locale.value) : "—";
}
async function load() {
  loading.value = true;
  error.value = "";
  try {
    const data = await getRuns(page.value);
    rows.value = data.items;
    total.value = data.total;
  } catch (e) {
    error.value = (e as Error).message;
  } finally {
    loading.value = false;
  }
}
async function show(id: number) {
  detailLoading.value = true;
  try {
    detail.value = await getRun(id);
    open.value = true;
  } catch (e) {
    ElMessage.error((e as Error).message);
  } finally {
    detailLoading.value = false;
  }
}
onMounted(load);
</script>
<template>
  <div v-loading="loading || detailLoading" class="ai-page">
    <header class="ai-header">
      <div>
        <div class="ai-eyebrow">AUTOOPS / AI OPERATIONS</div>
        <h1>{{ tr("我的诊断记录") }}</h1>
        <p>{{ tr("仅展示当前账号发起的诊断。") }}</p>
      </div>
      <el-button :icon="OperationIcons.Refresh" @click="load">{{ tr("刷新") }}</el-button>
    </header>
    <el-alert
      v-if="error"
      :title="error"
      type="error"
      class="ai-alert"
      :closable="false"
    />
    <section class="ai-card">
      <el-table :data="rows" row-key="id"
        ><el-table-column prop="id" label="ID" width="80" /><el-table-column
          prop="prompt"
          :label="tr('问题')"
          min-width="260"
          show-overflow-tooltip /><el-table-column
          :label="tr('状态')"
          width="140"
          ><template #default="{ row }"
            ><el-tag
              :type="
                row.status === 'failed'
                  ? 'danger'
                  : row.status === 'succeeded'
                    ? 'success'
                    : 'info'
              "
              >{{ statusText(row.status) }}</el-tag
            ></template
          ></el-table-column
        ><el-table-column :label="tr('创建时间')" min-width="180"
          ><template #default="{ row }">{{
            date(row.createdAt)
          }}</template></el-table-column
        ><el-table-column :label="tr('操作')" width="140" fixed="right"
          ><template #default="{ row }"
            ><el-button :icon="OperationIcons.View" link type="primary" @click="show(row.id)">{{
              tr("查看详情")
            }}</el-button></template
          ></el-table-column
        ><template #empty
          ><el-empty :description="tr('暂无诊断记录')" /></template></el-table
      ><el-pagination
        v-model:current-page="page"
        :page-size="20"
        :total="total"
        layout="prev, pager, next, total"
        class="ai-pagination"
        @current-change="load"
      />
    </section>
    <el-drawer
      v-model="open"
      :title="`${tr('运行详情')} #${detail?.id || ''}`"
      size="min(850px, 95vw)"
      ><template v-if="detail"
        ><el-descriptions :column="1" border
          ><el-descriptions-item :label="tr('问题')">{{
            detail.prompt
          }}</el-descriptions-item
          ><el-descriptions-item :label="tr('状态')">{{
            statusText(detail.status)
          }}</el-descriptions-item
          ><el-descriptions-item :label="tr('创建时间')">{{
            date(detail.createdAt)
          }}</el-descriptions-item
          ><el-descriptions-item :label="tr('完成时间')">{{
            date(detail.finishedAt)
          }}</el-descriptions-item></el-descriptions
        ><el-alert
          v-if="detail.error"
          :title="localizeError(detail.error)"
          type="error"
          :closable="false"
          class="ai-alert"
        />
        <pre class="ai-history-output">{{ detail.output }}</pre>
        <el-collapse v-if="detail.evidence"
          ><el-collapse-item :title="tr('业务快照')">
            <pre class="ai-history-output">{{ detail.evidence }}</pre>
          </el-collapse-item></el-collapse
        ></template
      ></el-drawer
    >
  </div>
</template>
