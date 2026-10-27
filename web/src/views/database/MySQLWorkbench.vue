<template>
  <div v-loading="loading" class="mysql-workbench">
    <header class="workbench-heading">
      <div class="source-heading">
        <el-button
          circle
          aria-label="返回数据源管理"
          @click="router.push('/database/sources')"
          >←</el-button
        ><span class="mysql-mark">MY</span>
        <div>
          <p>{{ $tr("数据库管理") }} / {{ $tr("数据源管理") }}</p>
          <h1>{{ source.name }}</h1>
          <span
            >{{ overview?.address || "MySQL" }} ·
            {{ overview?.version || "—" }}</span
          >
        </div>
      </div>
      <div class="heading-actions">
        <el-tag :type="overview ? 'success' : 'info'">{{
          overview ? $tr("已连接") : $tr("未连接")
        }}</el-tag
        ><el-tag v-if="overview?.metrics?.readOnly" type="warning">{{
          $tr("只读实例")
        }}</el-tag
        ><el-button @click="refresh">{{ $tr("刷新") }}</el-button>
      </div>
    </header>
    <el-alert
      v-if="error"
      :title="error"
      type="error"
      show-icon
      :closable="false"
      class="workbench-error"
    />
    <div class="browser-layout">
      <aside class="object-sidebar">
        <div class="sidebar-title">
          <strong>{{ $tr("数据库对象") }}</strong
          ><el-button
            v-if="canWrite"
            link
            type="primary"
            @click="openDefinition('create-database')"
            >＋ {{ $tr("新建库") }}</el-button
          >
        </div>
        <el-input
          v-model="objectSearch"
          clearable
          :placeholder="$tr('搜索库和表')"
        />
        <el-tree
          ref="tree"
          :data="treeData"
          node-key="id"
          :default-expanded-keys="expanded"
          :current-node-key="currentKey"
          :filter-node-method="filterNode"
          highlight-current
          :expand-on-click-node="false"
          @node-click="chooseNode"
          @node-expand="expandNode"
        >
          <template #default="{ data }"
            ><span class="object-node"
              ><span :class="['object-icon', data.kind]">{{
                data.kind === "database" ? "▱" : "▦"
              }}</span
              ><span>{{ data.label }}</span
              ><small
                v-if="data.kind === 'database' && tableCache[data.database]"
                >{{ tableCache[data.database].length }}</small
              ></span
            ></template
          >
        </el-tree>
        <el-empty
          v-if="!catalog.length && !loading"
          :description="$tr('暂无可访问的数据库')"
          :image-size="60"
        />
        <div class="sidebar-foot">
          {{ catalog.length }} {{ $tr("个数据库") }} ·
          {{ $tr("按需加载表目录") }}
        </div>
      </aside>
      <main class="object-main">
        <div class="object-heading">
          <div>
            <p>MYSQL WORKBENCH</p>
            <h2>{{ table || database || $tr("选择数据库") }}</h2>
            <span
              >{{ database
              }}<template v-if="metadata">
                · {{ metadata.engine || metadata.type }} ·
                {{ metadata.collation }}</template
              ></span
            >
          </div>
          <div class="object-actions">
            <el-button
              v-if="canWrite && database"
              @click="openDefinition('create-table')"
              >{{ $tr("新建表") }}</el-button
            ><el-dropdown v-if="canWrite && database" @command="openDefinition"
              ><el-button>{{ $tr("结构操作") }} ▾</el-button
              ><template #dropdown
                ><el-dropdown-menu
                  ><el-dropdown-item command="alter-database">{{
                    $tr("数据库字符集")
                  }}</el-dropdown-item
                  ><el-dropdown-item
                    v-if="table && !systemDatabase"
                    command="drop-table"
                    divided
                    >{{ $tr("删除当前表") }}</el-dropdown-item
                  ></el-dropdown-menu
                ></template
              ></el-dropdown
            >
          </div>
        </div>
        <el-tabs v-model="tab" @tab-change="loadTab">
          <el-tab-pane :label="$tr('表数据')" name="data">
            <template v-if="table">
              <div class="data-tools">
                <el-select
                  v-model="filter.column"
                  clearable
                  :placeholder="$tr('筛选字段')"
                  ><el-option
                    v-for="column in metadata?.columns || []"
                    :key="column.name"
                    :label="column.name"
                    :value="column.name" /></el-select
                ><el-select v-model="filter.operator"
                  ><el-option
                    v-for="op in operators"
                    :key="op[0]"
                    :label="$tr(op[1])"
                    :value="op[0]" /></el-select
                ><el-input
                  v-model="filter.value"
                  :disabled="['null', 'not-null'].includes(filter.operator)"
                  :placeholder="$tr('筛选值')"
                  clearable
                  @keyup.enter="searchRows"
                /><el-button @click="searchRows">{{ $tr("筛选") }}</el-button
                ><el-button @click="resetFilter">{{ $tr("重置") }}</el-button>
              </div>
              <div class="data-actions">
                <div>
                  <strong>{{ rowData.total }}</strong> {{ $tr("条匹配记录")
                  }}<span
                    v-if="metadata && !metadata.primaryKey.length"
                    class="muted"
                  >
                    · {{ $tr("无主键，行编辑不可用") }}</span
                  >
                </div>
                <div>
                  <el-button v-if="canEditTable" @click="openRow()"
                    >＋ {{ $tr("新增记录") }}</el-button
                  ><el-button
                    v-if="canEditTable"
                    @click="importVisible = true"
                    >{{ $tr("导入数据") }}</el-button
                  ><el-dropdown @command="exportData"
                    ><el-button :loading="exporting"
                      >{{ $tr("导出数据") }} ▾</el-button
                    ><template #dropdown
                      ><el-dropdown-menu
                        ><el-dropdown-item command="csv">CSV</el-dropdown-item
                        ><el-dropdown-item command="json"
                          >JSON</el-dropdown-item
                        ></el-dropdown-menu
                      ></template
                    ></el-dropdown
                  >
                </div>
              </div>
              <el-table
                v-loading="dataLoading"
                :data="rowData.rows"
                border
                class="data-grid"
                height="440"
                @sort-change="sortRows"
                @row-dblclick="
                  row =>
                    canEditTable && metadata?.primaryKey.length && openRow(row)
                "
              >
                <el-table-column
                  type="index"
                  width="55"
                  :index="index => (page - 1) * pageSize + index + 1"
                />
                <el-table-column
                  v-for="column in metadata?.columns || []"
                  :key="column.name"
                  :prop="column.name"
                  :label="column.name"
                  min-width="160"
                  sortable="custom"
                  show-overflow-tooltip
                  ><template #header
                    ><span
                      >{{ column.key === "PRI" ? "⌑ " : ""
                      }}{{ column.name }}</span
                    ><small class="column-type">{{
                      column.type
                    }}</small></template
                  ><template #default="{ row }"
                    ><span v-if="row[column.name] === null" class="null-value"
                      >NULL</span
                    ><span v-else>{{ row[column.name] }}</span></template
                  ></el-table-column
                >
                <el-table-column
                  v-if="canEditTable && metadata?.primaryKey.length"
                  fixed="right"
                  :label="$tr('操作')"
                  width="130"
                  ><template #default="{ row }"
                    ><el-button link type="primary" @click="openRow(row)">{{
                      $tr("编辑")
                    }}</el-button
                    ><el-button link type="danger" @click="deleteRow(row)">{{
                      $tr("删除")
                    }}</el-button></template
                  ></el-table-column
                >
                <template #empty
                  ><el-empty
                    :description="$tr('没有匹配的数据')"
                    :image-size="60"
                /></template>
              </el-table>
              <div class="data-pagination">
                <span>{{
                  $tr("双击记录可编辑；NULL 与空字符串分别显示。")
                }}</span
                ><el-pagination
                  v-model:current-page="page"
                  v-model:page-size="pageSize"
                  :page-sizes="[25, 50, 100, 200, 500]"
                  :total="rowData.total"
                  layout="sizes, prev, pager, next"
                  @current-change="loadRows"
                  @size-change="searchRows"
                />
              </div>
            </template>
            <template v-else
              ><el-table
                :data="tables"
                class="table-catalog"
                @row-click="item => selectTable(item.name)"
                ><el-table-column
                  prop="name"
                  :label="$tr('表 / 视图')" /><el-table-column
                  prop="type"
                  :label="$tr('类型')" /><el-table-column
                  prop="rows"
                  :label="$tr('估算行数')" /><el-table-column width="120"
                  ><template #default="{ row }"
                    ><el-button
                      link
                      type="primary"
                      @click.stop="selectTable(row.name)"
                      >{{ $tr("浏览数据") }}</el-button
                    ></template
                  ></el-table-column
                ><template #empty
                  ><el-empty
                    :description="
                      $tr('从左侧选择数据库，或新建数据库和表')
                    " /></template></el-table
            ></template>
          </el-tab-pane>
          <el-tab-pane :label="$tr('表结构')" name="structure">
            <template v-if="metadata"
              ><el-table :data="metadata.columns"
                ><el-table-column
                  prop="name"
                  :label="$tr('字段')"
                  min-width="140" /><el-table-column
                  prop="type"
                  :label="$tr('类型')"
                  min-width="160" /><el-table-column
                  :label="$tr('允许空值')"
                  width="100"
                  ><template #default="{ row }">{{
                    row.nullable ? "YES" : "NO"
                  }}</template></el-table-column
                ><el-table-column
                  prop="key"
                  :label="$tr('键')"
                  width="80" /><el-table-column :label="$tr('默认值')"
                  ><template #default="{ row }">{{
                    row.default === null ? "NULL" : row.default
                  }}</template></el-table-column
                ><el-table-column
                  prop="extra"
                  :label="$tr('属性')" /><el-table-column
                  prop="comment"
                  :label="$tr('注释')"
              /></el-table>
              <h3>{{ $tr("索引") }}</h3>
              <el-table :data="metadata.indexes"
                ><el-table-column
                  prop="name"
                  :label="$tr('索引名称')"
                /><el-table-column
                  prop="column"
                  :label="$tr('字段')"
                /><el-table-column
                  prop="sequence"
                  :label="$tr('顺序')"
                /><el-table-column
                  prop="type"
                  :label="$tr('类型')"
                /><el-table-column :label="$tr('唯一')"
                  ><template #default="{ row }">{{
                    row.unique ? "YES" : "NO"
                  }}</template></el-table-column
                ></el-table
              >
              <div class="ddl-title">
                <h3>CREATE TABLE</h3>
                <el-button size="small" @click="copyDDL">{{
                  $tr("复制建表语句")
                }}</el-button>
              </div>
              <pre class="ddl-code">{{ metadata.ddl }}</pre></template
            ><el-empty v-else :description="$tr('请选择数据表')" />
          </el-tab-pane>
          <el-tab-pane :label="$tr('控制台终端')" name="console"
            ><MySQLConsole
              v-if="consoleReady"
              ref="consoleView"
              :source="source"
              :database="database"
              :catalog="catalog"
              :tables="tables"
              :columns="metadata?.columns || []"
              :can-write="canWrite"
              :active="tab === 'console'"
              @database-change="selectDatabase"
              @executed="afterExecution"
          /></el-tab-pane>
          <el-tab-pane :label="$tr('基础配置')" name="settings">
            <el-descriptions :column="2" border
              ><el-descriptions-item :label="$tr('数据源')">{{
                source.name
              }}</el-descriptions-item
              ><el-descriptions-item :label="$tr('地址')">{{
                overview?.address
              }}</el-descriptions-item
              ><el-descriptions-item :label="$tr('默认数据库')">{{
                source.defaultDb || "—"
              }}</el-descriptions-item
              ><el-descriptions-item label="TLS">{{
                source.tlsMode
              }}</el-descriptions-item></el-descriptions
            >
            <p class="settings-note">
              {{
                $tr(
                  "连接账号、默认库和 TLS 在数据源管理中编辑。以下参数只修改当前实例的运行值，重启后以服务器配置为准。"
                )
              }}
            </p>
            <el-table v-loading="settingsLoading" :data="variables"
              ><el-table-column
                prop="name"
                :label="$tr('参数')"
                min-width="200"
              /><el-table-column
                prop="label"
                :label="$tr('说明')"
                min-width="250"
              /><el-table-column
                prop="value"
                :label="$tr('当前值')"
                width="150"
              /><el-table-column
                v-if="platformAccess.isAdmin"
                :label="$tr('操作')"
                width="100"
                ><template #default="{ row }"
                  ><el-button link type="primary" @click="editVariable(row)">{{
                    $tr("修改")
                  }}</el-button></template
                ></el-table-column
              ></el-table
            >
          </el-tab-pane>
          <el-tab-pane
            v-if="capabilities.audit"
            :label="$tr('操作记录')"
            name="audit"
            ><el-table :data="audits"
              ><el-table-column
                prop="action"
                :label="$tr('操作')" /><el-table-column
                prop="target"
                :label="$tr('对象')" /><el-table-column
                prop="status"
                :label="$tr('结果')" /><el-table-column
                prop="message"
                :label="$tr('说明')" /><el-table-column
                prop="createdAt"
                :label="$tr('时间')"
                min-width="180" /></el-table
          ></el-tab-pane>
        </el-tabs>
      </main>
    </div>
    <el-dialog
      v-model="rowVisible"
      :title="$tr(originalRow ? '编辑记录' : '新增记录')"
      width="min(760px, 94vw)"
      destroy-on-close
    >
      <p class="muted">
        {{ database }}.{{ table }} ·
        {{ $tr("未勾选的字段使用数据库默认值；NULL 开关表示空值。") }}
      </p>
      <div v-for="field in rowFields" :key="field.name" class="record-field">
        <el-checkbox
          v-if="!originalRow"
          v-model="field.include"
          :disabled="field.disabled"
        />
        <div class="field-name">
          <strong>{{ field.name }}</strong
          ><small>{{ field.type }}</small>
        </div>
        <el-input
          v-model="field.value"
          :type="/text|json/i.test(field.type) ? 'textarea' : 'text'"
          :disabled="
            field.disabled || field.isNull || (!originalRow && !field.include)
          "
        /><el-checkbox
          v-model="field.isNull"
          :disabled="!field.nullable || field.disabled"
          >NULL</el-checkbox
        >
      </div>
      <template #footer
        ><el-button @click="rowVisible = false">{{ $tr("取消") }}</el-button
        ><el-button type="primary" :loading="saving" @click="saveRow">{{
          $tr("保存记录")
        }}</el-button></template
      >
    </el-dialog>
    <el-dialog
      v-model="importVisible"
      :title="$tr('导入表数据')"
      width="min(660px, 94vw)"
      destroy-on-close
    >
      <el-alert
        :title="
          $tr(
            '以新增记录方式导入，不覆盖现有记录；发生冲突整批回滚。单次最多 10000 行 / 8 MiB。'
          )
        "
        type="info"
        :closable="false"
      />
      <p>{{ database }}.{{ table }}</p>
      <el-radio-group v-model="importFormat"
        ><el-radio-button value="csv">CSV</el-radio-button
        ><el-radio-button value="json">JSON</el-radio-button></el-radio-group
      >
      <p class="muted">
        {{
          $tr(
            "CSV 首行为字段名，\\N 表示 NULL，反斜杠开头的文本多写一个反斜杠；JSON 使用记录对象数组。省略自增和生成字段。"
          )
        }}
      </p>
      <input
        ref="fileInput"
        type="file"
        :accept="importFormat === 'csv' ? '.csv' : '.json'"
        aria-label="选择导入数据文件"
        @change="importFile = $event.target.files[0]"
      />
      <pre v-if="importFormat === 'json'" class="example-code">
[{ "name": "example", "description": null }]</pre
      >
      <template #footer
        ><el-button @click="importVisible = false">{{ $tr("取消") }}</el-button
        ><el-button
          type="primary"
          :loading="saving"
          :disabled="!importFile"
          @click="importData"
          >{{ $tr("开始导入") }}</el-button
        ></template
      >
    </el-dialog>
    <el-dialog
      v-model="definitionVisible"
      :title="$tr(definitionTitle)"
      width="min(900px, 94vw)"
      destroy-on-close
    >
      <el-form label-width="105px"
        ><el-form-item :label="$tr('数据库')"
          ><el-input
            v-model="definition.database"
            :disabled="definition.action !== 'create-database'" /></el-form-item
        ><el-form-item
          v-if="definition.action !== 'drop-table'"
          :label="$tr('字符集')"
          ><el-select v-model="definition.charset"
            ><el-option
              v-for="charset in ['utf8mb4', 'utf8', 'latin1']"
              :key="charset"
              :value="charset" /></el-select></el-form-item
        ><el-form-item
          v-if="definition.action.includes('table')"
          :label="$tr('表名')"
          ><el-input
            v-model="definition.table"
            :disabled="definition.action === 'drop-table'" /></el-form-item
      ></el-form>
      <template v-if="definition.action === 'create-table'"
        ><el-table :data="definition.columns"
          ><el-table-column :label="$tr('字段名')" min-width="160"
            ><template #default="{ row }"
              ><el-input v-model="row.name" /></template></el-table-column
          ><el-table-column :label="$tr('类型')" min-width="190"
            ><template #default="{ row }"
              ><el-select v-model="row.type" filterable allow-create
                ><el-option
                  v-for="kind in columnTypes"
                  :key="kind"
                  :value="kind" /></el-select></template></el-table-column
          ><el-table-column label="NULL" width="75"
            ><template #default="{ row }"
              ><el-checkbox
                v-model="row.nullable"
                :disabled="row.primary" /></template></el-table-column
          ><el-table-column :label="$tr('主键')" width="75"
            ><template #default="{ row }"
              ><el-checkbox v-model="row.primary" /></template></el-table-column
          ><el-table-column :label="$tr('自增')" width="75"
            ><template #default="{ row }"
              ><el-checkbox
                v-model="row.autoIncrement" /></template></el-table-column
          ><el-table-column width="60"
            ><template #default="{ $index }"
              ><el-button
                link
                type="danger"
                @click="definition.columns.splice($index, 1)"
                >×</el-button
              ></template
            ></el-table-column
          ></el-table
        ><el-button
          class="add-column"
          @click="
            definition.columns.push({
              name: '',
              type: 'VARCHAR(255)',
              nullable: true,
              primary: false,
              autoIncrement: false
            })
          "
          >＋ {{ $tr("添加字段") }}</el-button
        ></template
      >
      <el-alert
        v-if="definition.action === 'drop-table'"
        type="warning"
        :closable="false"
        :title="$tr('删除表会永久删除该表数据，执行前请确认已有可用备份。')"
      />
      <p v-if="definition.action === 'alter-database'" class="muted">
        {{ $tr("只修改数据库默认字符集，不转换现有表和字段。") }}
      </p>
      <template #footer
        ><el-button @click="definitionVisible = false">{{
          $tr("取消")
        }}</el-button
        ><el-button
          :type="definition.action === 'drop-table' ? 'danger' : 'primary'"
          :loading="saving"
          @click="saveDefinition"
          >{{ $tr("确认执行") }}</el-button
        ></template
      >
    </el-dialog>
  </div>
</template>
<script setup>
import { ref, reactive, computed, onMounted, watch } from "vue";
import { useRouter } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import api from "@/api/cmdb";
import { mysqlWorkbench as mysql, databaseResult } from "@/api/mysqlWorkbench";
import { canPlatform, platformAccess } from "@/utils/platformAccess";
import { tr } from "@/locales";
import MySQLConsole from "./components/MySQLConsole.vue";
const props = defineProps({ source: { type: Object, required: true } }),
  router = useRouter();
const consoleReady = ref(false);
const loading = ref(false),
  error = ref(""),
  overview = ref(null),
  capabilities = ref({ write: false, audit: false });
const catalog = ref([]),
  tableCache = reactive({}),
  database = ref(""),
  table = ref(""),
  tree = ref(),
  objectSearch = ref(""),
  expanded = ref([]),
  metadata = ref(null),
  tab = ref("data");
const page = ref(1),
  pageSize = ref(50),
  sort = ref(""),
  desc = ref(false),
  filter = reactive({ column: "", operator: "contains", value: "" }),
  rowData = ref({ rows: [], total: 0 }),
  dataLoading = ref(false);
const saving = ref(false),
  exporting = ref(false),
  rowVisible = ref(false),
  originalRow = ref(null),
  rowFields = ref([]),
  importVisible = ref(false),
  importFormat = ref("csv"),
  importFile = ref(null),
  fileInput = ref(),
  consoleView = ref();
const variables = ref([]),
  settingsLoading = ref(false),
  audits = ref([]),
  definitionVisible = ref(false),
  definition = reactive({
    action: "create-table",
    database: "",
    table: "",
    charset: "utf8mb4",
    columns: []
  });
let dataRevision = 0,
  selectionRevision = 0;
const tables = computed(() => tableCache[database.value] || []),
  canWrite = computed(
    () =>
      capabilities.value.write &&
      canPlatform("database:execute") &&
      !overview.value?.metrics?.readOnly
  ),
  systemDatabase = computed(() =>
    ["mysql", "sys", "information_schema", "performance_schema"].includes(
      database.value.toLowerCase()
    )
  ),
  canEditTable = computed(
    () =>
      canWrite.value &&
      !systemDatabase.value &&
      metadata.value?.engine === "InnoDB" &&
      metadata.value?.type === "BASE TABLE"
  );
const treeData = computed(() =>
  catalog.value.map(db => ({
    id: "db:" + db,
    label: db,
    kind: "database",
    database: db,
    children: tableCache[db]?.map(t => ({
      id: db + "." + t.name,
      label: t.name,
      kind: "table",
      database: db,
      table: t.name
    })) || [
      {
        id: "load:" + db,
        label: tr("点击展开加载"),
        kind: "loading",
        database: db
      }
    ]
  }))
);
const currentKey = computed(() =>
  table.value ? database.value + "." + table.value : "db:" + database.value
);
const operators = [
  ["contains", "包含"],
  ["eq", "等于"],
  ["ne", "不等于"],
  ["gt", "大于"],
  ["gte", "大于等于"],
  ["lt", "小于"],
  ["lte", "小于等于"],
  ["null", "为空"],
  ["not-null", "不为空"]
];
const columnTypes = [
  "BIGINT",
  "INT",
  "VARCHAR(255)",
  "TEXT",
  "DATETIME",
  "TIMESTAMP",
  "DATE",
  "DECIMAL(12,2)",
  "JSON",
  "BOOLEAN",
  "DOUBLE"
];
const definitionTitle = computed(
  () =>
    ({
      "create-database": "新建数据库",
      "create-table": "新建数据表",
      "alter-database": "数据库字符集",
      "drop-table": "删除数据表"
    })[definition.action]
);
function filterNode(value, node) {
  return !value || node.label.toLowerCase().includes(value.toLowerCase());
}
watch(objectSearch, value => tree.value?.filter(value));
async function tableList(db, force = false) {
  if (!tableCache[db] || force) {
    const data = await databaseResult(
      api.getDatabaseTables(props.source.id, db)
    );
    tableCache[db] = data.list || [];
  }
  return tableCache[db];
}
async function expandNode(node) {
  if (node.kind === "database")
    try {
      await tableList(node.database);
    } catch (e) {
      error.value = e.message;
    }
}
async function chooseNode(node) {
  if (node.kind === "table") {
    if (database.value !== node.database) await selectDatabase(node.database);
    await selectTable(node.table);
  } else await selectDatabase(node.database);
}
async function selectDatabase(db) {
  selectionRevision++;
  dataRevision++;
  database.value = db;
  table.value = "";
  metadata.value = null;
  rowData.value = { rows: [], total: 0 };
  page.value = 1;
  sort.value = "";
  filter.column = "";
  expanded.value = [...new Set([...expanded.value, "db:" + db])];
  try {
    await tableList(db);
    error.value = "";
  } catch (e) {
    error.value = e.message;
  }
}
async function selectTable(name) {
  const version = ++selectionRevision;
  dataRevision++;
  table.value = name;
  metadata.value = null;
  rowData.value = { rows: [], total: 0 };
  page.value = 1;
  sort.value = "";
  filter.column = "";
  tab.value = "data";
  dataLoading.value = true;
  try {
    const info = await mysql.table(props.source.id, {
      database: database.value,
      table: name
    });
    if (version !== selectionRevision) return;
    metadata.value = info;
    await loadRows();
  } catch (e) {
    if (version === selectionRevision) error.value = e.message;
  } finally {
    if (version === selectionRevision) dataLoading.value = false;
  }
}
function dataParams() {
  return {
    database: database.value,
    table: table.value,
    page: page.value,
    size: pageSize.value,
    sort: sort.value,
    desc: desc.value,
    ...filter
  };
}
async function loadRows() {
  if (!table.value || !metadata.value) return;
  const version = ++dataRevision;
  dataLoading.value = true;
  error.value = "";
  try {
    const data = await mysql.data(props.source.id, dataParams());
    if (version === dataRevision) rowData.value = data;
  } catch (e) {
    if (version === dataRevision) error.value = e.message;
  } finally {
    if (version === dataRevision) dataLoading.value = false;
  }
}
function searchRows() {
  page.value = 1;
  void loadRows();
}
function resetFilter() {
  Object.assign(filter, { column: "", operator: "contains", value: "" });
  searchRows();
}
function sortRows({ prop, order }) {
  sort.value = order ? prop : "";
  desc.value = order === "descending";
  searchRows();
}
async function refresh() {
  loading.value = true;
  error.value = "";
  try {
    const [info, dbs, perms] = await Promise.all([
      databaseResult(api.getDatabaseOverview(props.source.id)),
      databaseResult(api.getDatabaseCatalog(props.source.id)),
      mysql.capabilities(props.source.id)
    ]);
    overview.value = info;
    catalog.value = dbs.list || [];
    capabilities.value = perms;
    const chosen = catalog.value.includes(database.value)
      ? database.value
      : catalog.value.includes(props.source.defaultDb) &&
          ![
            "mysql",
            "sys",
            "information_schema",
            "performance_schema"
          ].includes(props.source.defaultDb)
        ? props.source.defaultDb
        : catalog.value.find(
            db =>
              ![
                "mysql",
                "sys",
                "information_schema",
                "performance_schema"
              ].includes(db)
          ) || catalog.value[0];
    if (chosen) {
      await tableList(chosen, true);
      if (database.value !== chosen) await selectDatabase(chosen);
      else if (
        table.value &&
        tableCache[chosen].some(t => t.name === table.value)
      )
        await selectTable(table.value);
      else await selectDatabase(chosen);
    }
  } catch (e) {
    error.value = e.message;
  } finally {
    loading.value = false;
  }
}
async function confirm(title = tr("确认数据操作")) {
  const { value } = await ElMessageBox.prompt(
    `${tr("请输入完整数据源名称确认操作")}：${props.source.name}`,
    title,
    { inputValidator: value => value === props.source.name || tr("名称不匹配") }
  );
  return value;
}
function report(e) {
  if (e instanceof Error) ElMessage.error(e.message);
}
function openRow(row = null) {
  if (!canEditTable.value) return;
  originalRow.value = row ? { ...row } : null;
  rowFields.value = metadata.value.columns.map(c => {
    const value = row ? row[c.name] : c.default;
    return {
      ...c,
      value: value == null ? "" : String(value),
      isNull: row ? value === null : false,
      include:
        !c.generated &&
        !c.binary &&
        !c.extra.includes("auto_increment") &&
        c.default === null,
      disabled: c.generated || c.binary
    };
  });
  rowVisible.value = true;
}
async function saveRow() {
  saving.value = true;
  try {
    const values = {};
    for (const f of rowFields.value) {
      if (f.disabled || (!originalRow.value && !f.include)) continue;
      const value = f.isNull ? null : f.value;
      if (!originalRow.value || value !== originalRow.value[f.name])
        values[f.name] = value;
    }
    if (!Object.keys(values).length) {
      ElMessage.info(tr("没有需要保存的字段"));
      return;
    }
    const confirmName = await confirm();
    await mysql.row(props.source.id, {
      database: database.value,
      table: table.value,
      action: originalRow.value ? "update" : "insert",
      values,
      original: originalRow.value || {},
      confirmName
    });
    rowVisible.value = false;
    ElMessage.success(tr("记录已保存"));
    await loadRows();
  } catch (e) {
    report(e);
  } finally {
    saving.value = false;
  }
}
async function deleteRow(row) {
  try {
    const confirmName = await confirm(tr("删除当前记录"));
    await mysql.row(props.source.id, {
      database: database.value,
      table: table.value,
      action: "delete",
      original: row,
      confirmName
    });
    ElMessage.success(tr("记录已删除"));
    await loadRows();
  } catch (e) {
    report(e);
  }
}
async function exportData(format) {
  exporting.value = true;
  try {
    const data = await mysql.export(props.source.id, {
      ...dataParams(),
      format
    });
    const blob = new Blob([data.content], {
      type: format === "csv" ? "text/csv;charset=utf-8" : "application/json"
    });
    const url = URL.createObjectURL(blob),
      a = document.createElement("a");
    a.href = url;
    a.download = `${database.value}.${table.value}.${format}`;
    a.click();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
    ElMessage.success(`${tr("已导出")} ${data.rows} ${tr("行")}`);
  } catch (e) {
    report(e);
  } finally {
    exporting.value = false;
  }
}
async function importData() {
  if (!importFile.value) return;
  if (importFile.value.size > 8 * 1024 * 1024) {
    ElMessage.error(tr("导入文件不能超过 8 MiB"));
    return;
  }
  saving.value = true;
  try {
    const confirmName = await confirm(tr("导入表数据")),
      data = new FormData();
    data.append("file", importFile.value);
    for (const [key, value] of Object.entries({
      database: database.value,
      table: table.value,
      format: importFormat.value,
      confirmName
    }))
      data.append(key, value);
    const result = await mysql.import(props.source.id, data);
    importVisible.value = false;
    importFile.value = null;
    ElMessage.success(`${tr("已导入")} ${result.affectedRows} ${tr("行")}`);
    await loadRows();
  } catch (e) {
    report(e);
  } finally {
    saving.value = false;
  }
}
function openDefinition(action) {
  Object.assign(definition, {
    action,
    database: action === "create-database" ? "" : database.value,
    table: action === "drop-table" ? table.value : "",
    charset: "utf8mb4",
    columns: [
      {
        name: "id",
        type: "BIGINT",
        primary: true,
        nullable: false,
        autoIncrement: true
      },
      {
        name: "name",
        type: "VARCHAR(255)",
        primary: false,
        nullable: true,
        autoIncrement: false
      }
    ]
  });
  definitionVisible.value = true;
}
async function saveDefinition() {
  saving.value = true;
  try {
    const confirmName = await confirm(definitionTitle.value);
    await mysql.schema(props.source.id, { ...definition, confirmName });
    definitionVisible.value = false;
    ElMessage.success(tr("结构操作完成"));
    const selected = definition.database,
      newTable = definition.action === "create-table" ? definition.table : "";
    await refresh();
    if (catalog.value.includes(selected)) await selectDatabase(selected);
    if (newTable) await selectTable(newTable);
  } catch (e) {
    report(e);
  } finally {
    saving.value = false;
  }
}
async function loadTab() {
  if (tab.value === "console") consoleReady.value = true;
  if (tab.value === "settings") {
    settingsLoading.value = true;
    try {
      variables.value = await mysql.variables(props.source.id);
    } catch (e) {
      report(e);
    } finally {
      settingsLoading.value = false;
    }
  } else if (tab.value === "audit") {
    try {
      audits.value =
        (
          await databaseResult(
            api.getDatabaseAudits(props.source.id, { page: 1, size: 50 })
          )
        ).list || [];
    } catch (e) {
      report(e);
    }
  }
}
async function editVariable(item) {
  try {
    const { value } = await ElMessageBox.prompt(
      `${item.label} (${item.min}–${item.max})`,
      item.name,
      {
        inputValue: item.value,
        inputValidator: v =>
          (/^\d+(\.\d+)?$/.test(v) &&
            Number(v) >= item.min &&
            Number(v) <= item.max) ||
          tr("数值超出范围")
      }
    );
    const confirmName = await confirm(tr("修改实例运行参数"));
    await mysql.setVariable(props.source.id, {
      name: item.name,
      value,
      confirmName
    });
    ElMessage.success(tr("运行参数已更新"));
    await loadTab();
  } catch (e) {
    report(e);
  }
}
async function copyDDL() {
  try {
    await navigator.clipboard.writeText(metadata.value.ddl);
    ElMessage.success(tr("已复制"));
  } catch {
    ElMessage.error(tr("复制失败，请手动选择建表语句"));
  }
}
async function afterExecution() {
  if (table.value) await loadRows();
}
onMounted(refresh);
</script>
<style scoped>
.mysql-workbench {
  padding: 22px;
  color: var(--text-primary);
}
.workbench-heading,
.source-heading,
.heading-actions,
.sidebar-title,
.object-heading,
.object-actions,
.data-actions,
.data-pagination,
.ddl-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
}
.workbench-heading {
  margin-bottom: 22px;
}
.source-heading {
  justify-content: flex-start;
  min-width: 0;
}
.source-heading > div {
  min-width: 0;
  overflow-wrap: anywhere;
}
.source-heading h1 {
  font-size: 23px;
  margin: 5px 0;
}
.source-heading p,
.object-heading p {
  font-size: 11px;
  letter-spacing: 1px;
  color: var(--text-secondary);
  margin: 0;
}
.source-heading span,
.object-heading > div > span {
  font-size: 12px;
  color: var(--text-secondary);
}
.source-heading .mysql-mark {
  background: color-mix(in srgb, var(--primary) 12%, transparent);
  border: 1px solid color-mix(in srgb, var(--primary) 20%, transparent);
  color: var(--primary);
  font-weight: 700;
  font-size: 18px;
  border-radius: 13px;
  padding: 15px 12px;
}
.browser-layout {
  display: grid;
  grid-template-columns: 250px minmax(0, 1fr);
  min-height: 680px;
  border: 1px solid var(--border);
  background: var(--bg-card);
  border-radius: 16px;
  overflow: hidden;
}
.object-sidebar {
  min-width: 0;
  background: var(--bg-card-alt);
  border-right: 1px solid var(--border);
  padding: 18px 12px;
  position: relative;
}
.sidebar-title {
  padding: 0 3px 16px;
  font-size: 14px;
}
.object-sidebar .el-input {
  margin-bottom: 16px;
}
.object-sidebar :deep(.el-tree) {
  background: transparent;
  font-size: 13px;
  max-height: 620px;
  overflow: auto;
}
.object-sidebar :deep(.el-tree-node__content) {
  height: 35px;
  border-radius: 6px;
}
.object-node {
  display: flex;
  align-items: center;
  gap: 9px;
  min-width: 0;
  width: 100%;
  padding-right: 8px;
}
.object-node > span:nth-child(2) {
  overflow: hidden;
  text-overflow: ellipsis;
}
.object-node small {
  margin-left: auto;
  color: var(--text-secondary);
}
.object-icon {
  color: var(--text-secondary);
  font-size: 18px;
}
.object-icon.database {
  color: var(--primary);
}
.sidebar-foot {
  font-size: 11px;
  color: var(--text-secondary);
  margin-top: 25px;
}
.object-main {
  padding: 22px;
  min-width: 0;
}
.object-heading {
  margin-bottom: 16px;
}
.object-heading h2 {
  margin: 6px 0;
  font-size: 21px;
}
.object-heading p {
  color: var(--primary);
  font-weight: 600;
}
.data-tools {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  padding: 12px;
  background: var(--bg-card-alt);
  border: 1px solid var(--border);
  border-radius: 9px;
  margin: 6px 0 16px;
}
.data-tools .el-select {
  width: 165px;
}
.data-tools .el-input {
  width: 210px;
  flex: 1;
  min-width: 140px;
}
.data-actions {
  margin-bottom: 12px;
  font-size: 12px;
  color: var(--text-secondary);
}
.data-actions strong {
  color: var(--text-primary);
  font-size: 16px;
}
.data-actions > div:last-child {
  display: flex;
  gap: 8px;
}
.column-type {
  display: block;
  font-weight: 400;
  font-size: 10px;
  color: var(--text-secondary);
  line-height: 16px;
}
.data-grid :deep(.cell) {
  font-family: "SFMono-Regular", Consolas, monospace;
  font-size: 12px;
}
.null-value {
  font-style: italic;
  color: var(--text-secondary);
  font-size: 11px;
}
.data-pagination {
  margin-top: 16px;
  flex-wrap: wrap;
}
.data-pagination > span {
  font-size: 11px;
  color: var(--text-secondary);
}
.muted,
.settings-note {
  color: var(--text-secondary);
  font-size: 12px;
  line-height: 1.8;
}
.workbench-error {
  margin-bottom: 16px;
}
.ddl-code,
.example-code {
  background: var(--bg-card-alt);
  padding: 18px;
  border-radius: 9px;
  border: 1px solid var(--border);
  font-size: 12px;
  overflow: auto;
  white-space: pre-wrap;
  line-height: 1.7;
}
.record-field {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 0;
  border-bottom: 1px solid var(--border);
}
.field-name {
  width: 150px;
  flex-shrink: 0;
}
.field-name strong,
.field-name small {
  display: block;
  overflow-wrap: anywhere;
}
.field-name small {
  color: var(--text-secondary);
  font-size: 11px;
}
.add-column {
  margin: 16px 0;
}
.table-catalog :deep(.el-table__row) {
  cursor: pointer;
}
@media (max-width: 1100px) {
  .browser-layout {
    grid-template-columns: 205px minmax(0, 1fr);
  }
  .object-main {
    padding: 15px;
  }
  .object-heading,
  .data-actions {
    flex-wrap: wrap;
  }
}
@media (max-width: 768px) {
  .mysql-workbench {
    padding: 12px;
  }
  .workbench-heading {
    align-items: flex-start;
    flex-wrap: wrap;
  }
  .browser-layout {
    grid-template-columns: minmax(0, 1fr);
  }
  .source-heading {
    gap: 8px;
  }
  .source-heading h1 {
    font-size: 18px;
  }
  .data-actions > div:last-child,
  .data-pagination {
    flex-wrap: wrap;
  }
  .object-sidebar {
    border-right: 0;
    border-bottom: 1px solid var(--border);
  }
  .object-sidebar :deep(.el-tree) {
    max-height: 220px;
  }
  .sidebar-foot {
    display: none;
  }
  .record-field {
    flex-wrap: wrap;
  }
  .field-name {
    width: 100%;
  }
}
</style>
