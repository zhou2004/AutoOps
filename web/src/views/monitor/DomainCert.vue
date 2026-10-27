<template>
  <div class="domain-cert-monitor">
    <header class="monitor-page-header">
      <div>
        <span class="monitor-eyebrow">MONITORING</span>
        <h1>{{ $tr("域名证书监控") }}</h1>
        <p>{{ $tr("按分类管理域名，跟踪证书有效期与检查结果。") }}</p>
      </div>
      <div class="monitor-total">
        <strong>{{ total }}</strong
        ><span>{{ $tr("匹配目标") }}</span>
      </div>
    </header>
    <el-card shadow="never" class="domain-cert-card">
      <div v-show="showSearch" class="search-section">
        <el-form :model="queryParams" :inline="true" class="search-form">
          <el-form-item :label="$tr('分类')">
            <el-select
              v-model="queryParams.categoryId"
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
          <el-form-item prop="domain" :label="$tr('域名')">
            <el-input
              v-model="queryParams.domain"
              :placeholder="$tr('请输入域名')"
              clearable
              size="small"
              style="width: 200px"
              @keyup.enter="handleQuery"
            />
          </el-form-item>
          <el-form-item prop="status" :label="$tr('状态')">
            <el-select
              v-model="queryParams.status"
              :placeholder="$tr('请选择状态')"
              clearable
              size="small"
              style="width: 150px"
            >
              <el-option :label="$tr('正常')" :value="1" /><el-option
                :label="$tr('即将过期')"
                :value="2"
              />
              <el-option :label="$tr('已过期')" :value="3" /><el-option
                :label="$tr('检查失败')"
                :value="4"
              />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button
              type="primary"
              icon="Search"
              size="small"
              @click="handleQuery"
              >{{ $tr("搜索") }}</el-button
            >
            <el-button icon="Refresh" size="small" @click="resetQuery">{{
              $tr("重置")
            }}</el-button>
          </el-form-item>
        </el-form>
      </div>
      <div class="action-section">
        <el-button type="primary" icon="Plus" size="small" @click="handleAdd">{{
          $tr("添加域名")
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
          scope="domain_cert"
          :ids="ids"
          @loaded="onCategoriesLoaded"
          @changed="getList"
        />
        <el-button icon="Refresh" @click="getList">{{ $tr("刷新") }}</el-button>
      </div>
      <div class="table-section">
        <el-table
          v-loading="Loading"
          :data="domainCertList"
          row-key="id"
          style="width: 100%"
          class="domain-cert-table"
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
          <el-table-column :label="$tr('域名')" prop="domain" min-width="180">
            <template #default="scope">
              <el-link
                type="primary"
                :underline="false"
                @click="handleViewDetail(scope.row)"
                >{{ scope.row.domain }}</el-link
              >
            </template>
          </el-table-column>
          <el-table-column :label="$tr('端口')" prop="port" width="70" />
          <el-table-column
            :label="$tr('颁发者')"
            prop="issuer"
            min-width="150"
            show-overflow-tooltip
          />
          <el-table-column
            :label="$tr('到期日期')"
            prop="notAfter"
            min-width="150"
          />
          <el-table-column :label="$tr('剩余天数')" width="100" align="center">
            <template #default="scope">
              <el-tag
                v-if="scope.row.remainingDays >= 0"
                :type="getRemainingDaysType(scope.row.remainingDays)"
                size="small"
                effect="dark"
                >{{ scope.row.remainingDays }} {{ $tr("天") }}</el-tag
              >
              <el-tag v-else type="info" size="small" effect="dark">{{
                $tr("未知")
              }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column :label="$tr('状态')" width="100" align="center">
            <template #default="scope">
              <el-tag
                :type="getStatusType(scope.row.status)"
                size="small"
                effect="dark"
                >{{ $tr(getStatusText(scope.row.status)) }}</el-tag
              >
            </template>
          </el-table-column>
          <el-table-column
            :label="$tr('最近检查')"
            prop="checkTime"
            min-width="150"
          />
          <el-table-column :label="$tr('操作')" width="200" fixed="right">
            <template #default="{ row }">
              <el-button
                link
                type="primary"
                :loading="row._checking"
                @click="handleCheckSingle(row)"
                >{{ $tr("检查") }}</el-button
              >
              <el-button link type="primary" @click="handleEdit(row)">{{
                $tr("编辑")
              }}</el-button>
              <el-dropdown
                trigger="click"
                @command="
                  command =>
                    command === 'detail'
                      ? handleViewDetail(row)
                      : handleDelete(row)
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
          :current-page="queryParams.page"
          :page-sizes="[10, 20, 50, 100]"
          :page-size="queryParams.pageSize"
          layout="total, sizes, prev, pager, next, jumper"
          :total="total"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>

    <!-- 添加/编辑对话框 -->
    <el-dialog
      v-model="dialogVisible"
      class="monitor-overlay"
      :title="isEdit ? '编辑域名' : $tr('添加域名')"
      width="min(540px, 94vw)"
      :close-on-click-modal="false"
    >
      <el-form
        ref="formRef"
        :model="form"
        :rules="formRules"
        label-width="80px"
      >
        <el-form-item :label="$tr('分类')">
          <el-select v-model="form.categoryId" filterable
            ><el-option :value="0" :label="$tr('未分类')" /><el-option
              v-for="c in categories"
              :key="c.id"
              :value="c.id"
              :label="c.name"
          /></el-select>
        </el-form-item>
        <el-form-item :label="$tr('域名')" prop="domain">
          <el-input
            v-model="form.domain"
            :placeholder="$tr('请输入域名，如: example.com')"
          />
        </el-form-item>
        <el-form-item :label="$tr('端口')" prop="port">
          <el-input-number
            v-model="form.port"
            :min="1"
            :max="65535"
            :placeholder="$tr('默认443')"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ $tr("取消") }}</el-button>
        <el-button
          type="primary"
          :loading="submitLoading"
          @click="handleSubmit"
          >{{ isEdit ? $tr("保存") : $tr("确定") }}</el-button
        >
      </template>
    </el-dialog>

    <!-- 证书详情对话框 -->
    <el-dialog
      v-model="detailVisible"
      class="monitor-overlay"
      :title="$tr('证书详情')"
      width="min(680px, 94vw)"
    >
      <div v-if="detailData" class="cert-detail">
        <el-descriptions :column="1" border>
          <el-descriptions-item :label="$tr('域名')">{{
            detailData.domain
          }}</el-descriptions-item>
          <el-descriptions-item :label="$tr('端口')">{{
            detailData.port
          }}</el-descriptions-item>
          <el-descriptions-item :label="$tr('颁发者')">{{
            detailData.issuer || "-"
          }}</el-descriptions-item>
          <el-descriptions-item :label="$tr('主题')">{{
            detailData.subject || "-"
          }}</el-descriptions-item>
          <el-descriptions-item :label="$tr('起始日期')">{{
            detailData.notBefore || "-"
          }}</el-descriptions-item>
          <el-descriptions-item :label="$tr('到期日期')">{{
            detailData.notAfter || "-"
          }}</el-descriptions-item>
          <el-descriptions-item :label="$tr('剩余天数')">
            <el-tag
              v-if="detailData.remainingDays >= 0"
              :type="getRemainingDaysType(detailData.remainingDays)"
              size="small"
              >{{ detailData.remainingDays }} {{ $tr("天") }}</el-tag
            ><span v-else>-</span>
          </el-descriptions-item>
          <el-descriptions-item :label="$tr('状态')"
            ><el-tag :type="getStatusType(detailData.status)" size="small">{{
              $tr(getStatusText(detailData.status))
            }}</el-tag></el-descriptions-item
          >
          <el-descriptions-item :label="$tr('检查时间')">{{
            detailData.checkTime || "-"
          }}</el-descriptions-item>
          <el-descriptions-item :label="$tr('错误信息')"
            ><span v-if="detailData.errorMsg" style="color: #f56c6c">{{
              detailData.errorMsg
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
  getDomainCertList,
  addDomainCert,
  updateDomainCert,
  deleteDomainCert,
  batchDeleteDomainCert,
  checkDomainCert,
  checkAllDomainCert,
  getDomainCertById
} from "@/api/monitor";
export default {
  components: { MonitorCategoryTools },
  data() {
    return {
      categories: [],
      listRequestId: 0,
      Loading: true,
      multiple: true,
      showSearch: true,
      total: 0,
      ids: [],
      domainCertList: [],
      queryParams: {
        page: 1,
        pageSize: 20,
        domain: "",
        status: undefined,
        categoryId: undefined
      },
      dialogVisible: false,
      isEdit: false,
      submitLoading: false,
      form: { id: 0, domain: "", port: 443, categoryId: 0 },
      formRules: {
        domain: [
          {
            required: true,
            get message() {
              return tr("请输入域名");
            },
            trigger: "blur"
          }
        ]
      },
      detailVisible: false,
      detailData: null,
      checkingAll: false
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
        this.queryParams.categoryId &&
        !categories.some(c => c.id === this.queryParams.categoryId)
      ) {
        this.queryParams.categoryId = undefined;
        this.queryParams.page = 1;
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
      this.Loading = true;
      try {
        const params = {
          ...this.queryParams,
          categoryId:
            this.queryParams.categoryId === ""
              ? undefined
              : this.queryParams.categoryId
        };
        const { data: res } = await getDomainCertList(params);
        if (requestId !== this.listRequestId) return;
        if (res.code !== 200) {
          this.$message.error(res.message);
          return;
        }
        this.domainCertList = res.data.list || [];
        this.total = res.data.total || 0;
        if (
          !this.domainCertList.length &&
          this.total &&
          this.queryParams.page > 1
        ) {
          this.queryParams.page = Math.ceil(
            this.total / this.queryParams.pageSize
          );
          return this.getList();
        }
      } catch {
        if (requestId === this.listRequestId)
          this.$message.error(tr("获取列表失败"));
      } finally {
        if (requestId === this.listRequestId) this.Loading = false;
      }
    },
    handleQuery() {
      this.queryParams.page = 1;
      this.getList();
    },
    resetQuery() {
      this.queryParams = {
        page: 1,
        pageSize: 20,
        domain: "",
        status: undefined,
        categoryId: undefined
      };
      this.getList();
      this.$message.success(tr("重置成功"));
    },
    handleSelectionChange(selection) {
      this.ids = selection.map(i => i.id);
      this.multiple = !selection.length;
    },
    handleSizeChange(s) {
      this.queryParams.pageSize = s;
      this.queryParams.page = 1;
      this.getList();
    },
    handleCurrentChange(p) {
      this.queryParams.page = p;
      this.getList();
    },
    handleAdd() {
      this.isEdit = false;
      this.form = { id: 0, domain: "", port: 443, categoryId: 0 };
      this.dialogVisible = true;
    },
    async handleEdit(row) {
      this.isEdit = true;
      this.form = {
        id: row.id,
        domain: row.domain,
        port: row.port,
        categoryId: row.categoryId || 0
      };
      this.dialogVisible = true;
    },
    handleSubmit() {
      this.$refs.formRef.validate(async valid => {
        if (!valid) return;
        this.submitLoading = true;
        try {
          const fn = this.isEdit ? updateDomainCert : addDomainCert;
          const { data: res } = await fn(this.form);
          if (res.code !== 200) {
            this.$message.error(res.message);
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
    async handleDelete(row) {
      const c = await this.$confirm(
        `是否确认删除 "${row.domain}"？`,
        tr("提示"),
        {
          confirmButtonText: tr("确定"),
          cancelButtonText: tr("取消"),
          type: "warning"
        }
      ).catch(e => e);
      if (c !== "confirm") return;
      try {
        const { data: res } = await deleteDomainCert(row.id);
        if (res.code !== 200) {
          this.$message.error(res.message);
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
        this.$message.warning(tr("请选择要删除的记录"));
        return;
      }
      const c = await this.$confirm(
        `确认删除 ${this.ids.length} 条记录？`,
        tr("提示"),
        { type: "warning" }
      ).catch(e => e);
      if (c !== "confirm") return;
      try {
        const { data: res } = await batchDeleteDomainCert(this.ids);
        if (res.code !== 200) {
          this.$message.error(res.message);
        } else {
          this.$message.success(tr("删除成功"));
          this.ids = [];
          this.getList();
        }
      } catch {
        this.$message.error(tr("批量删除失败"));
      }
    },
    async handleCheckSingle(row) {
      row._checking = true;
      try {
        const { data: res } = await checkDomainCert(row.id);
        if (res.code !== 200) {
          this.$message.error(res.message || tr("检查失败"));
        } else {
          this.$message.success(`检查完成: ${res.data.domain}`);
          this.detailData = { ...row, ...res.data };
          this.detailVisible = true;
          this.getList();
        }
      } catch {
        this.$message.error(tr("检查失败"));
      } finally {
        row._checking = false;
      }
    },
    async handleCheckAll() {
      this.checkingAll = true;
      try {
        const { data: res } = await checkAllDomainCert();
        if (res.code !== 200) {
          this.$message.error(res.message);
        } else {
          this.$message.success(`检查完成，共 ${res.data.total} 个`);
          this.getList();
        }
      } catch {
        this.$message.error(tr("批量检查失败"));
      } finally {
        this.checkingAll = false;
      }
    },
    async handleViewDetail(row) {
      try {
        const { data: res } = await getDomainCertById(row.id);
        if (res.code === 200) {
          this.detailData = res.data;
          this.detailVisible = true;
        } else {
          this.$message.error(res.message);
        }
      } catch {
        this.$message.error(tr("获取详情失败"));
      }
    },
    getStatusType(s) {
      return (
        { 1: "success", 2: "warning", 3: "danger", 4: "info" }[s] || "info"
      );
    },
    getStatusText(s) {
      return (
        { 1: "正常", 2: "即将过期", 3: "已过期", 4: "检查失败" }[s] || "未知"
      );
    },
    getRemainingDaysType(d) {
      if (d <= 7) return "danger";
      if (d <= 30) return "warning";
      return "success";
    }
  }
};
</script>
<style scoped src="./monitor-targets.css"></style>
