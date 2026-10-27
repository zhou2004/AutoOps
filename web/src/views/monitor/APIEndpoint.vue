<template>
  <div class="api-monitor">
    <header class="monitor-page-header">
      <div>
        <span class="monitor-eyebrow">MONITORING</span>
        <h1>{{ $tr("API 监控") }}</h1>
        <p>{{ $tr("按分类管理接口，关注响应状态、耗时与异常。") }}</p>
      </div>
      <div class="monitor-total">
        <strong>{{ total }}</strong
        ><span>{{ $tr("匹配目标") }}</span>
      </div>
    </header>
    <el-card shadow="never" class="api-card">
      <div v-show="showSearch" class="search-section">
        <el-form :model="q" :inline="true">
          <el-form-item :label="$tr('分类')">
            <el-select
              v-model="q.categoryId"
              clearable
              filterable
              :placeholder="$tr('全部分类')"
              style="width: 180px"
              @change="handleQuery"
            >
              <el-option :value="0" :label="$tr('未分类')" />
              <el-option
                v-for="c in categories"
                :key="c.id"
                :value="c.id"
                :label="c.name"
              />
            </el-select>
          </el-form-item>
          <el-form-item :label="$tr('名称')"
            ><el-input
              v-model="q.name"
              :placeholder="$tr('搜索名称或 URL')"
              clearable
              size="small"
              style="width: 200px"
              @keyup.enter="handleQuery"
          /></el-form-item>
          <el-form-item :label="$tr('状态')"
            ><el-select
              v-model="q.status"
              :placeholder="$tr('状态')"
              clearable
              size="small"
              style="width: 150px"
            >
              <el-option :label="$tr('正常')" :value="1" /><el-option
                :label="$tr('异常')"
                :value="2"
              /><el-option :label="$tr('超时')" :value="3" /><el-option
                :label="$tr('失败')"
                :value="4"
              /> </el-select
          ></el-form-item>
          <el-form-item
            ><el-button
              type="primary"
              icon="Search"
              size="small"
              @click="handleQuery"
              >{{ $tr("搜索") }}</el-button
            ><el-button icon="Refresh" size="small" @click="resetQuery">{{
              $tr("重置")
            }}</el-button></el-form-item
          >
        </el-form>
      </div>
      <div class="action-section">
        <el-button type="primary" icon="Plus" size="small" @click="handleAdd">{{
          $tr("添加API")
        }}</el-button>
        <el-button
          icon="Refresh"
          size="small"
          :loading="checkingAll"
          @click="handleCheckAll"
          >{{ $tr("检查全部") }}</el-button
        >
        <el-button
          type="danger"
          plain
          icon="Delete"
          size="small"
          :disabled="multiple"
          @click="handleBatchDelete"
          >{{ $tr("批量删除") }}</el-button
        >
        <MonitorCategoryTools
          scope="api_endpoint"
          :ids="ids"
          @loaded="onCategoriesLoaded"
          @changed="getList"
        />
        <el-button icon="Refresh" @click="getList">{{ $tr("刷新") }}</el-button>
      </div>
      <div class="table-section">
        <el-table
          v-loading="loading"
          :data="list"
          row-key="id"
          @selection-change="handleSelectionChange"
        >
          <el-table-column type="selection" width="45" />
          <el-table-column :label="$tr('分类')" min-width="130"
            ><template #default="{ row }"
              ><el-tag type="info" effect="plain">{{
                categoryName(row.categoryId)
              }}</el-tag></template
            ></el-table-column
          >
          <el-table-column :label="$tr('名称')" prop="name" min-width="140" />
          <el-table-column
            label="URL"
            prop="url"
            min-width="250"
            show-overflow-tooltip
          />
          <el-table-column :label="$tr('方法')" prop="method" width="80"
            ><template #default="s"
              ><el-tag size="small">{{ s.row.method }}</el-tag></template
            ></el-table-column
          >
          <el-table-column
            :label="$tr('状态码')"
            prop="lastStatusCode"
            width="80"
            align="center"
          />
          <el-table-column :label="$tr('响应时间')" width="100" align="center">
            <template #default="s"
              ><span>{{ s.row.lastResponseTime }}ms</span></template
            >
          </el-table-column>
          <el-table-column
            :label="$tr('期望码')"
            prop="expectedCode"
            width="80"
            align="center"
          />
          <el-table-column :label="$tr('状态')" width="80" align="center">
            <template #default="s"
              ><el-tag
                :type="getStatusType(s.row.status)"
                size="small"
                effect="dark"
                >{{ $tr(getStatusText(s.row.status)) }}</el-tag
              ></template
            >
          </el-table-column>
          <el-table-column
            :label="$tr('检查时间')"
            prop="checkTime"
            min-width="150"
          />
          <el-table-column :label="$tr('操作')" width="200" fixed="right">
            <template #default="{ row }">
              <el-button
                link
                type="primary"
                :loading="row._checking"
                @click="handleCheck(row)"
                >{{ $tr("检查") }}</el-button
              >
              <el-button link type="primary" @click="handleEdit(row)">{{
                $tr("编辑")
              }}</el-button>
              <el-dropdown
                trigger="click"
                @command="
                  command =>
                    command === 'detail' ? handleDetail(row) : handleDelete(row)
                "
              >
                <el-button link>{{ $tr("更多") }}</el-button>
                <template #dropdown
                  ><el-dropdown-menu
                    ><el-dropdown-item command="detail">{{
                      $tr("详情")
                    }}</el-dropdown-item
                    ><el-dropdown-item command="delete" divided>{{
                      $tr("删除")
                    }}</el-dropdown-item></el-dropdown-menu
                  ></template
                >
              </el-dropdown>
            </template>
          </el-table-column>
          <template #empty
            ><el-empty
              :description="$tr('暂无匹配目标，请调整筛选条件或添加监控')"
          /></template>
        </el-table>
      </div>
      <div class="pagination-section">
        <el-pagination
          :current-page="q.page"
          :page-sizes="[10, 20, 50, 100]"
          :page-size="q.pageSize"
          layout="total, sizes, prev, pager, next, jumper"
          :total="total"
          @size-change="
            s => {
              q.pageSize = s;
              q.page = 1;
              getList();
            }
          "
          @current-change="
            p => {
              q.page = p;
              getList();
            }
          "
        />
      </div>
    </el-card>

    <el-dialog
      v-model="dialogVisible"
      class="monitor-overlay"
      :title="isEdit ? '编辑API' : $tr('添加API')"
      width="min(720px, 94vw)"
    >
      <el-form ref="formRef" :model="form" :rules="rules" label-width="110px">
        <el-form-item :label="$tr('分类')">
          <el-select v-model="form.categoryId" filterable
            ><el-option :value="0" :label="$tr('未分类')" /><el-option
              v-for="c in categories"
              :key="c.id"
              :value="c.id"
              :label="c.name"
          /></el-select>
        </el-form-item>
        <el-form-item :label="$tr('名称')" prop="name"
          ><el-input v-model="form.name" :placeholder="$tr('如: 生产环境API')"
        /></el-form-item>
        <el-form-item label="URL" prop="url"
          ><el-input
            v-model="form.url"
            placeholder="https://api.example.com/health"
        /></el-form-item>
        <el-form-item :label="$tr('请求方法')"
          ><el-select v-model="form.method" style="width: 150px"
            ><el-option label="GET" value="GET" /><el-option
              label="POST"
              value="POST" /><el-option label="PUT" value="PUT" /><el-option
              label="DELETE"
              value="DELETE" /></el-select
        ></el-form-item>
        <el-form-item :label="$tr('请求头(JSON)')"
          ><el-input
            v-model="form.headers"
            placeholder='{"Authorization":"Bearer xxx"}'
        /></el-form-item>
        <el-form-item :label="$tr('请求体')"
          ><el-input
            v-model="form.body"
            type="textarea"
            :rows="2"
            placeholder='{"key":"value"}'
        /></el-form-item>
        <el-form-item :label="$tr('检查间隔(秒)')"
          ><el-input-number v-model="form.checkInterval" :min="10" :max="86400"
        /></el-form-item>
        <el-form-item :label="$tr('超时(秒)')"
          ><el-input-number v-model="form.timeout" :min="1" :max="120"
        /></el-form-item>
        <el-form-item :label="$tr('期望状态码')"
          ><el-input-number v-model="form.expectedCode" :min="100" :max="599"
        /></el-form-item>
        <el-form-item :label="$tr('期望响应体')"
          ><el-input
            v-model="form.expectedBody"
            :placeholder="$tr('留空则不检查响应体')"
        /></el-form-item>
      </el-form>
      <template #footer
        ><el-button @click="dialogVisible = false">{{ $tr("取消") }}</el-button
        ><el-button
          type="primary"
          :loading="submitLoading"
          @click="handleSubmit"
          >{{ isEdit ? $tr("保存") : $tr("确定") }}</el-button
        ></template
      >
    </el-dialog>

    <el-dialog
      v-model="detailVisible"
      class="monitor-overlay"
      :title="$tr('检查详情')"
      width="min(680px, 94vw)"
    >
      <div v-if="detail" class="cert-detail">
        <el-descriptions :column="1" border>
          <el-descriptions-item :label="$tr('名称')">{{
            detail.name
          }}</el-descriptions-item>
          <el-descriptions-item label="URL">{{
            detail.url
          }}</el-descriptions-item>
          <el-descriptions-item :label="$tr('方法')">{{
            detail.method
          }}</el-descriptions-item>
          <el-descriptions-item :label="$tr('状态码')">{{
            detail.lastStatusCode
          }}</el-descriptions-item>
          <el-descriptions-item :label="$tr('响应时间')"
            >{{ detail.lastResponseTime }}ms</el-descriptions-item
          >
          <el-descriptions-item :label="$tr('状态')"
            ><el-tag :type="getStatusType(detail.status)" size="small">{{
              $tr(getStatusText(detail.status))
            }}</el-tag></el-descriptions-item
          >
          <el-descriptions-item :label="$tr('检查时间')">{{
            detail.checkTime
          }}</el-descriptions-item>
          <el-descriptions-item :label="$tr('错误信息')"
            ><span v-if="detail.errorMsg" style="color: #f56c6c">{{
              detail.errorMsg
            }}</span
            ><span v-else>-</span></el-descriptions-item
          >
        </el-descriptions>
      </div>
    </el-dialog>
  </div>
</template>
<script>
import "./monitor-theme.css";
import MonitorCategoryTools from "./components/MonitorCategoryTools.vue";
import { tr } from "@/locales";

import {
  getAPIEndpointList,
  addAPIEndpoint,
  updateAPIEndpoint,
  deleteAPIEndpoint,
  batchDeleteAPIEndpoint,
  checkAPIEndpoint,
  checkAllAPIEndpoint
} from "@/api/monitor";
export default {
  components: { MonitorCategoryTools },
  data() {
    return {
      categories: [],
      listRequestId: 0,
      loading: true,
      multiple: true,
      showSearch: true,
      total: 0,
      ids: [],
      list: [],
      q: {
        page: 1,
        pageSize: 20,
        name: "",
        status: undefined,
        categoryId: undefined
      },
      checkingAll: false,
      dialogVisible: false,
      isEdit: false,
      submitLoading: false,
      detailVisible: false,
      detail: null,
      form: {
        id: 0,
        categoryId: 0,
        name: "",
        url: "",
        method: "GET",
        headers: "",
        body: "",
        checkInterval: 300,
        timeout: 10,
        expectedCode: 200,
        expectedBody: ""
      },
      rules: {
        name: [
          {
            required: true,
            get message() {
              return tr("请输入名称");
            },
            trigger: "blur"
          }
        ],
        url: [
          {
            required: true,
            get message() {
              return tr("请输入URL");
            },
            trigger: "blur"
          }
        ]
      }
    };
  },
  created() {
    this.getList();
  },
  methods: {
    categoryName(id) {
      return id
        ? this.categories.find(c => c.id === id)?.name || `${tr("分类")} #${id}`
        : tr("未分类");
    },
    onCategoriesLoaded(categories) {
      this.categories = categories;
      if (
        this.q.categoryId &&
        !categories.some(c => c.id === this.q.categoryId)
      ) {
        this.q.categoryId = undefined;
        this.q.page = 1;
        this.getList();
      }
      if (
        this.form.categoryId &&
        !categories.some(c => c.id === this.form.categoryId)
      )
        this.form.categoryId = 0;
    },
    async getList() {
      const requestId = ++this.listRequestId;
      this.loading = true;
      try {
        const params = {
          ...this.q,
          categoryId: this.q.categoryId === "" ? undefined : this.q.categoryId
        };
        const { data: res } = await getAPIEndpointList(params);
        if (requestId !== this.listRequestId) return;
        if (res.code !== 200) {
          this.$message.error(res.message);
          return;
        }
        this.list = res.data.list || [];
        this.total = res.data.total || 0;
        if (!this.list.length && this.total && this.q.page > 1) {
          this.q.page = Math.ceil(this.total / this.q.pageSize);
          return this.getList();
        }
      } catch {
        if (requestId === this.listRequestId)
          this.$message.error(tr("获取列表失败"));
      } finally {
        if (requestId === this.listRequestId) this.loading = false;
      }
    },
    handleQuery() {
      this.q.page = 1;
      this.getList();
    },
    resetQuery() {
      this.q = {
        page: 1,
        pageSize: 20,
        name: "",
        status: undefined,
        categoryId: undefined
      };
      this.getList();
    },
    handleSelectionChange(s) {
      this.ids = s.map(i => i.id);
      this.multiple = !s.length;
    },
    handleAdd() {
      this.isEdit = false;
      this.form = {
        id: 0,
        categoryId: 0,
        name: "",
        url: "",
        method: "GET",
        headers: "",
        body: "",
        checkInterval: 300,
        timeout: 10,
        expectedCode: 200,
        expectedBody: ""
      };
      this.dialogVisible = true;
    },
    handleEdit(r) {
      this.isEdit = true;
      this.form = {
        id: r.id,
        categoryId: r.categoryId || 0,
        name: r.name,
        url: r.url,
        method: r.method,
        headers: r.headers,
        body: r.body,
        checkInterval: r.checkInterval,
        timeout: r.timeout,
        expectedCode: r.expectedCode,
        expectedBody: r.expectedBody
      };
      this.dialogVisible = true;
    },
    handleSubmit() {
      this.$refs.formRef.validate(async v => {
        if (!v) return;
        this.submitLoading = true;
        try {
          const fn = this.isEdit ? updateAPIEndpoint : addAPIEndpoint;
          const { data: r } = await fn(this.form);
          if (r.code !== 200) {
            this.$message.error(r.message);
          } else {
            this.$message.success(this.isEdit ? "更新成功" : tr("添加成功"));
            this.dialogVisible = false;
            this.getList();
          }
        } catch {
          this.$message.error(tr("操作失败"));
        } finally {
          this.submitLoading = false;
        }
      });
    },
    async handleDelete(r) {
      const c = await this.$confirm(`确认删除 "${r.name}"?`, tr("提示"), {
        type: "warning"
      }).catch(e => e);
      if (c !== "confirm") return;
      try {
        const { data: r2 } = await deleteAPIEndpoint(r.id);
        if (r2.code !== 200) {
          this.$message.error(r2.message);
        } else {
          this.$message.success(tr("删除成功"));
          this.getList();
        }
      } catch {
        this.$message.error(tr("删除失败"));
      }
    },
    async handleBatchDelete() {
      if (!this.ids.length) {
        this.$message.warning(tr("请选择记录"));
        return;
      }
      const c = await this.$confirm(
        `确认删除 ${this.ids.length} 条?`,
        tr("提示"),
        { type: "warning" }
      ).catch(e => e);
      if (c !== "confirm") return;
      try {
        const { data: r } = await batchDeleteAPIEndpoint(this.ids);
        if (r.code !== 200) {
          this.$message.error(r.message);
        } else {
          this.$message.success(tr("删除成功"));
          this.ids = [];
          this.getList();
        }
      } catch {
        this.$message.error(tr("批量删除失败"));
      }
    },
    async handleCheck(r) {
      r._checking = true;
      try {
        const { data: r2 } = await checkAPIEndpoint(r.id);
        if (r2.code !== 200) {
          this.$message.error(r2.message);
        } else {
          this.$message.success(tr("检查完成"));
          this.detail = r2.data;
          this.detailVisible = true;
          this.getList();
        }
      } catch {
        this.$message.error(tr("检查失败"));
      } finally {
        r._checking = false;
      }
    },
    async handleCheckAll() {
      this.checkingAll = true;
      try {
        const { data: r } = await checkAllAPIEndpoint();
        if (r.code !== 200) {
          this.$message.error(r.message);
        } else {
          this.$message.success(`检查完成,共 ${r.data.total} 个`);
          this.getList();
        }
      } catch {
        this.$message.error(tr("批量检查失败"));
      } finally {
        this.checkingAll = false;
      }
    },
    handleDetail(r) {
      this.detail = r;
      this.detailVisible = true;
    },
    getStatusType(s) {
      return (
        { 1: "success", 2: "danger", 3: "warning", 4: "info" }[s] || "info"
      );
    },
    getStatusText(s) {
      return { 1: "正常", 2: "异常", 3: "超时", 4: "失败" }[s] || "未知";
    }
  }
};
</script>
<style scoped src="./monitor-targets.css"></style>
