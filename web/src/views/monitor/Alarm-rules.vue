<template>
  <div class="alarm-rules-management">
    <el-row :gutter="20" style="height: 100%">
      <!-- 左侧：业务线与群组列表 -->
      <el-col :xs="24" :md="7" :lg="6">
        <el-card class="modern-card menu-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <span class="header-title">
                <el-icon><FolderOpened /></el-icon> {{ $tr("告警群组") }}
              </span>
              <el-tooltip :content="$tr('新建群组')" placement="top"
                ><span class="action-icon-trigger"
                  ><el-button
                    type="primary"
                    link
                    icon="Plus"
                    class="action-icon"
                    :aria-label="$tr('新建群组')"
                    @click="openGroupDialog()" /></span
              ></el-tooltip>
            </div>
          </template>

          <div class="group-select-wrap">
            <div class="ds-selector" style="margin-bottom: 10px">
              <el-select
                v-model="activeDataSourceId"
                :placeholder="$tr('选择数据源')"
                style="width: 100%"
                class="ds-select"
                @change="handleDataSourceChange"
              >
                <template #prefix>
                  <el-icon v-if="!activeDataSourceId"><Monitor /></el-icon>
                  <el-icon v-else
                    ><component :is="getDataSourceIcon(activeDataSourceId)"
                  /></el-icon>
                </template>
                <el-option
                  v-for="ds in dataSources"
                  :key="ds.id || ds.ID"
                  :label="$tr(ds.name)"
                  :value="ds.id || ds.ID"
                  class="ds-option"
                >
                  <div class="ds-option-content">
                    <el-icon
                      ><component :is="getDsTypeIcon(ds.type)"
                    /></el-icon>
                    <span class="ds-name">{{ ds.name }}</span>
                    <el-tag
                      v-if="ds.deployMethod"
                      size="small"
                      type="info"
                      class="ds-deploy-tag"
                    >
                      <el-icon v-if="getDeployIcon(ds.deployMethod)" size="14"
                        ><component :is="getDeployIcon(ds.deployMethod)"
                      /></el-icon>
                      {{ ds.deployMethod }}
                    </el-tag>
                  </div>
                </el-option>
              </el-select>
            </div>
            <el-select
              v-model="filterCmdbGroup"
              filterable
              :placeholder="$tr('筛选业务分组')"
              clearable
              style="width: 100%; margin-bottom: 10px"
              @change="handleBusinessFilter"
            >
              <el-option
                v-for="c in cmdbGroups"
                :key="c.id"
                :label="c.path"
                :value="c.id"
              />
            </el-select>
          </div>

          <!-- 群组菜单 -->
          <el-menu
            v-loading="loadingGroups"
            :default-active="activeGroupId"
            class="group-menu transparent-bg"
            @select="handleGroupSelect"
          >
            <el-menu-item index="all">
              <el-icon class="group-colored-icon" size="20"><Grid /></el-icon>
              <template #title>
                <div class="menu-item-content">
                  <span class="group-name">{{ $tr("全部规则") }}</span>
                </div>
              </template>
            </el-menu-item>
            <el-menu-item
              v-for="g in filteredGroupList"
              :key="g.ID || g.id"
              :index="(g.ID || g.id) + ''"
            >
              <el-icon class="group-colored-icon" size="20"><Folder /></el-icon>
              <template #title>
                <div class="menu-item-content">
                  <span class="group-name" :title="$tr(g.group_name)">
                    {{ g.group_name }}
                    <el-tag
                      v-if="getBusinessGroupLabel(g)"
                      size="small"
                      class="cluster-tag"
                      disable-transitions
                      >{{ $tr(getBusinessGroupLabel(g)) }}</el-tag
                    >
                  </span>
                  <div class="group-actions">
                    <el-tooltip :content="$tr('编辑群组')" placement="top"
                      ><span class="action-icon-trigger"
                        ><el-button
                          link
                          type="primary"
                          icon="EditPen"
                          class="action-icon"
                          :aria-label="$tr('编辑群组')"
                          @click.stop="openGroupDialog(g)" /></span
                    ></el-tooltip>
                    <el-tooltip :content="$tr('删除群组')" placement="top"
                      ><span class="action-icon-trigger"
                        ><el-button
                          link
                          type="danger"
                          icon="Delete"
                          class="action-icon"
                          :aria-label="$tr('删除群组')"
                          @click.stop="deleteGroup(g.ID || g.id)" /></span
                    ></el-tooltip>
                  </div>
                </div>
              </template>
            </el-menu-item>
            <el-empty
              v-if="filteredGroupList.length === 0"
              :description="$tr('暂无群组数据')"
              :image-size="60"
            />
          </el-menu>
        </el-card>
      </el-col>

      <!-- 右侧：规则列表 -->
      <el-col :xs="24" :md="17" :lg="18">
        <el-card class="modern-card main-card" shadow="hover">
          <template #header>
            <div class="card-header">
              <span class="header-title">
                <el-icon><Warning /></el-icon>
                {{
                  activeGroupId === "all"
                    ? "所有群组告警规则"
                    : activeGroupName
                      ? `【${activeGroupName}】告警规则`
                      : $tr("告警规则")
                }}
              </span>
              <div class="right-actions">
                <el-button
                  type="info"
                  plain
                  icon="Collection"
                  @click="openStyleDialog"
                  >{{ $tr("分类(Style)管理") }}</el-button
                >
                <el-button
                  type="primary"
                  icon="Plus"
                  :disabled="!activeGroupId"
                  @click="openRuleDialog()"
                  >{{ $tr("新增规则") }}</el-button
                >
              </div>
            </div>
          </template>

          <!-- 搜索过滤 -->
          <div class="search-section">
            <el-form :inline="true" :model="query" class="search-form">
              <el-form-item :label="$tr('名称')">
                <el-input
                  v-model="query.alert"
                  :placeholder="$tr('告警名称')"
                  clearable
                  style="width: 150px"
                  @keyup.enter="fetchRules"
                />
              </el-form-item>
              <el-form-item :label="$tr('优先级')">
                <el-select
                  v-model="query.severity"
                  clearable
                  :placeholder="$tr('优先级')"
                  style="width: 120px"
                  @change="fetchRules"
                >
                  <el-option :label="$tr('Critical (严重)')" value="critical" />
                  <el-option :label="$tr('Warning (警告)')" value="warning" />
                  <el-option :label="$tr('Info (通知)')" value="info" />
                </el-select>
              </el-form-item>
              <el-form-item :label="$tr('状态')">
                <el-select
                  v-model="query.status"
                  clearable
                  :placeholder="$tr('告警状态')"
                  style="width: 120px"
                  @change="fetchRules"
                >
                  <el-option :label="$tr('触发中 (Firing)')" value="firing" />
                  <el-option :label="$tr('等待中 (Pending)')" value="pending" />
                  <el-option
                    :label="$tr('未触发 (Inactive)')"
                    value="inactive"
                  />
                </el-select>
              </el-form-item>
              <el-form-item :label="$tr('规则分类')">
                <el-select
                  v-model="query.style"
                  clearable
                  :placeholder="$tr('选择分类')"
                  style="width: 150px"
                  @change="fetchRules"
                >
                  <el-option
                    v-for="s in styleList"
                    :key="s.name"
                    :label="$tr(s.name)"
                    :value="s.name"
                  />
                </el-select>
              </el-form-item>
              <el-form-item :label="$tr('启用状态')">
                <el-select
                  v-model="query.enabled"
                  clearable
                  :placeholder="$tr('状态')"
                  style="width: 120px"
                  @change="fetchRules"
                >
                  <el-option :label="$tr('已启用')" :value="1" />
                  <el-option :label="$tr('已禁用')" :value="0" />
                </el-select>
              </el-form-item>
              <el-form-item>
                <el-button type="primary" icon="Search" @click="fetchRules">{{
                  $tr("查询")
                }}</el-button>
                <el-button type="warning" icon="Refresh" @click="resetQuery">{{
                  $tr("重置")
                }}</el-button>
              </el-form-item>
            </el-form>
          </div>

          <!-- 规则表格 -->
          <el-table
            v-loading="loadingRules"
            :data="ruleList"
            border
            stripe
            style="width: 100%"
            :header-cell-style="{
              background: 'rgba(102, 126, 234, 0.1)',
              color: 'var(--text-primary)',
              fontWeight: '600'
            }"
          >
            <el-table-column
              prop="alert"
              :label="$tr('告警名称 (Alert)')"
              min-width="150"
              show-overflow-tooltip
            />
            <el-table-column
              v-if="activeGroupId === 'all'"
              :label="$tr('所属群组')"
              min-width="140"
              show-overflow-tooltip
            >
              <template #default="{ row }">
                <el-tag size="small" type="warning" plain>{{
                  getGroupName(row.group_id)
                }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column
              prop="severity"
              :label="$tr('优先级')"
              width="100"
              align="center"
            >
              <template #default="{ row }">
                <el-tag :type="getSeverityType(row.severity)" effect="dark">{{
                  row.severity || "未知"
                }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column
              prop="style"
              :label="$tr('分类')"
              width="120"
              align="center"
            >
              <template #default="{ row }">
                <el-tag type="info">{{ row.style || "暂无分类" }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column
              prop="expr"
              :label="$tr('PromQL 表达式 (Expr)')"
              min-width="250"
              show-overflow-tooltip
            >
              <template #default="{ row }">
                <code class="promql-code">{{ row.expr }}</code>
              </template>
            </el-table-column>
            <el-table-column
              prop="for_duration"
              :label="$tr('持续时间(For)')"
              width="120"
              align="center"
            />
            <el-table-column
              prop="status"
              :label="$tr('告警状态')"
              width="100"
              align="center"
            >
              <template #default="{ row }">
                <el-tag
                  :type="
                    row.status === 'firing'
                      ? 'danger'
                      : row.status === 'pending'
                        ? 'warning'
                        : 'success'
                  "
                >
                  {{
                    row.status === "firing"
                      ? "触发中"
                      : row.status === "pending"
                        ? $tr("等待中")
                        : "未触发"
                  }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column
              prop="enabled"
              :label="$tr('启用状态')"
              width="100"
              align="center"
            >
              <template #default="{ row }">
                <el-switch
                  v-model="row.enabled"
                  :active-value="1"
                  :inactive-value="0"
                  style="--el-switch-on-color: #13ce66"
                  @change="toggleRuleStatus(row)"
                />
              </template>
            </el-table-column>
            <el-table-column
              :label="$tr('操作')"
              width="144"
              align="center"
              fixed="right"
            >
              <template #default="{ row }">
                <div class="operation-buttons">
                  <el-tooltip :content="$tr('检查')" placement="top"
                    ><span class="action-icon-trigger"
                      ><el-button
                        link
                        type="success"
                        icon="CircleCheck"
                        class="action-icon"
                        :aria-label="$tr('检查')"
                        @click="checkRule(row)" /></span
                  ></el-tooltip>
                  <el-tooltip :content="$tr('编辑')" placement="top"
                    ><span class="action-icon-trigger"
                      ><el-button
                        link
                        type="primary"
                        icon="EditPen"
                        class="action-icon"
                        :aria-label="$tr('编辑')"
                        @click="openRuleDialog(row)" /></span
                  ></el-tooltip>
                  <el-tooltip :content="$tr('删除')" placement="top"
                    ><span class="action-icon-trigger"
                      ><el-button
                        link
                        type="danger"
                        icon="Delete"
                        class="action-icon"
                        :aria-label="$tr('删除')"
                        @click="deleteRule(row.ID || row.id)" /></span
                  ></el-tooltip>
                </div>
              </template>
            </el-table-column>
          </el-table>

          <!-- 分页 -->
          <div class="pagination-section">
            <el-pagination
              v-model:current-page="query.page"
              v-model:page-size="query.pageSize"
              background
              layout="total, sizes, prev, pager, next, jumper"
              :total="totalRules"
              :page-sizes="[10, 20, 50, 100]"
              @current-change="fetchRules"
              @size-change="fetchRules"
            />
          </div>
        </el-card>
      </el-col>
    </el-row>

    <!-- 群组表单弹窗 -->
    <el-dialog
      v-model="groupDialogVisible"
      :title="$tr(groupForm.id ? '编辑告警群组' : '新建告警群组')"
      width="min(650px,94vw)"
      class="modern-dialog"
      :append-to-body="true"
    >
      <el-form
        ref="groupFormRef"
        :model="groupForm"
        :rules="groupRules"
        label-position="top"
        class="rule-form"
        :disabled="submitting"
      >
        <!-- Section 1: 基本配置 -->
        <div class="form-section-title">
          <el-icon><Setting /></el-icon> {{ $tr("基本配置") }}
        </div>
        <el-row :gutter="20">
          <el-col :xs="24" :sm="12">
            <el-form-item :label="$tr('群组名称')" prop="group_name">
              <el-input
                v-model="groupForm.group_name"
                maxlength="255"
                :placeholder="$tr('例如：node-system-usage')"
              />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12">
            <el-form-item :label="$tr('数据源')" prop="data_source_id">
              <el-select
                v-model="groupForm.data_source_id"
                :placeholder="$tr('请选择数据源')"
                style="width: 100%"
                :disabled="!!groupForm.id"
              >
                <el-option
                  v-for="ds in dataSources"
                  :key="ds.id || ds.ID"
                  :label="$tr(ds.name)"
                  :value="ds.id || ds.ID"
                >
                  <div class="ds-option-content" style="margin-left: 0">
                    <el-icon
                      ><component :is="getDsTypeIcon(ds.type)"
                    /></el-icon>
                    <span>{{ ds.name }}</span>
                  </div>
                </el-option>
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>

        <el-form-item :label="$tr('业务线')" prop="cmdb_group_id">
          <el-select
            v-model="groupForm.cmdb_group_id"
            filterable
            :loading="loadingBusinessGroups"
            :placeholder="$tr('请选择资产管理中的业务分组')"
            style="width: 100%"
            :disabled="!!businessGroupsError"
          >
            <el-option
              v-for="g in cmdbGroups"
              :key="g.id"
              :label="g.path"
              :value="g.id"
            />
          </el-select>
          <p class="help-text">
            {{ $tr("业务线与资产管理的业务分组关联，可搜索分组名称选择。") }}
          </p>
          <el-alert
            v-if="legacyBusinessLabel"
            :title="
              $tr('原业务线') +
              '：' +
              legacyBusinessLabel +
              '。' +
              $tr('请选择对应业务分组后保存。')
            "
            type="info"
            :closable="false"
          />
          <el-alert
            v-if="businessGroupsError"
            :title="businessGroupsError"
            type="error"
            :closable="false"
          />
          <el-button
            v-if="businessGroupsError"
            link
            type="primary"
            @click="fetchBusinessGroups"
            >{{ $tr("重新加载") }}</el-button
          >
          <el-empty
            v-if="
              !loadingBusinessGroups &&
              !businessGroupsError &&
              !cmdbGroups.length
            "
            :image-size="50"
            :description="
              $tr('暂无可选业务分组，请先在资产管理中创建或获取授权。')
            "
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="groupDialogVisible = false">{{
            $tr("取消")
          }}</el-button>
          <el-button
            type="primary"
            :loading="submitting"
            :disabled="
              loadingBusinessGroups ||
              !!businessGroupsError ||
              !cmdbGroups.length
            "
            @click="submitGroup"
            >{{ $tr("保存") }}</el-button
          >
        </span>
      </template>
    </el-dialog>

    <!-- 规则表单弹窗 -->
    <el-dialog
      v-model="ruleDialogVisible"
      :title="$tr(ruleForm.id ? '编辑告警规则' : '新增告警规则')"
      width="760px"
      class="modern-dialog"
      :append-to-body="true"
      :modal="false"
    >
      <el-form
        ref="ruleFormRef"
        :model="ruleForm"
        :rules="ruleRules"
        label-width="120px"
        class="rule-form"
      >
        <!-- Section 1: 基本配置 -->
        <div class="form-section-title">
          <el-icon><Setting /></el-icon> {{ $tr("基本配置") }}
        </div>
        <el-row :gutter="20">
          <el-col :xs="24" :sm="12">
            <el-form-item :label="$tr('所属群组')" prop="group_id">
              <el-select
                v-model="ruleForm.group_id"
                :placeholder="$tr('选择群组')"
                style="width: 100%"
              >
                <el-option
                  v-for="g in groupList"
                  :key="g.ID || g.id"
                  :label="$tr(g.group_name)"
                  :value="parseInt(g.ID || g.id)"
                />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12">
            <el-form-item :label="$tr('告警分类')" prop="style">
              <el-select
                v-model="ruleForm.style"
                :placeholder="$tr('选择分类')"
                style="width: 100%"
              >
                <el-option
                  v-for="s in styleList"
                  :key="s.name"
                  :label="$tr(s.name)"
                  :value="s.name"
                />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>

        <el-row :gutter="20">
          <el-col :xs="24" :sm="12">
            <el-form-item :label="$tr('告警名称(Alert)')" prop="alert">
              <el-input
                v-model="ruleForm.alert"
                :placeholder="$tr('例如: NodeCPUUsage')"
              />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12">
            <el-form-item :label="$tr('告警级别')" prop="severity">
              <el-select v-model="ruleForm.severity" style="width: 100%">
                <el-option :label="$tr('Critical (严重)')" value="critical" />
                <el-option :label="$tr('Warning (警告)')" value="warning" />
                <el-option :label="$tr('Info (通知)')" value="info" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>

        <!-- Section 2: 规则定义 -->
        <div class="form-section-title">
          <el-icon><DataLine /></el-icon> {{ $tr("规则定义") }}
        </div>
        <el-form-item
          :label="$tr('表达式(Expr)')"
          prop="expr"
          class="expr-form-item"
        >
          <el-input
            v-model="ruleForm.expr"
            type="textarea"
            :rows="4"
            class="promql-input"
            :placeholder="$tr(promqlExample)"
          />
        </el-form-item>

        <el-row :gutter="20">
          <el-col :xs="24" :sm="12">
            <el-form-item :label="$tr('持续时间(For)')" prop="for_duration">
              <el-input
                v-model="ruleForm.for_duration"
                :placeholder="$tr('例如: 5m, 1h')"
              />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12">
            <el-form-item :label="$tr('是否启用')">
              <el-switch
                v-model="ruleForm.enabled"
                :active-value="1"
                :inactive-value="0"
              />
            </el-form-item>
          </el-col>
        </el-row>

        <!-- Section 3: 附加信息 -->
        <div class="form-section-title">
          <el-icon><Document /></el-icon> {{ $tr("附加信息") }}
        </div>
        <el-form-item :label="$tr('摘要(Summary)')" prop="summary">
          <el-input
            v-model="ruleForm.summary"
            :placeholder="$tr('告警摘要，短描述')"
          />
        </el-form-item>

        <el-form-item :label="$tr('描述(Desc)')" prop="description">
          <el-input
            v-model="ruleForm.description"
            type="textarea"
            :rows="2"
            :placeholder="$tr('详细的告警描述 (支持模板变量)')"
          />
        </el-form-item>

        <el-row :gutter="20">
          <el-col :xs="24" :sm="12">
            <el-form-item :label="$tr('约束条件')" prop="constraints">
              <el-input
                v-model="ruleForm.constraints"
                :placeholder="
                  $tr('JSON格式，如 {&quot;env&quot;: &quot;prod&quot;}')
                "
              />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12">
            <el-form-item :label="$tr('附加标签')" prop="labels">
              <el-input
                v-model="ruleForm.labels"
                :placeholder="
                  $tr('JSON格式，如 {&quot;team&quot;: &quot;devops&quot;}')
                "
              />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="ruleDialogVisible = false">{{
            $tr("取消")
          }}</el-button>
          <el-button type="primary" :loading="submitting" @click="submitRule">{{
            $tr("保存")
          }}</el-button>
        </span>
      </template>
    </el-dialog>

    <!-- 告警分类管理弹窗 -->
    <el-dialog
      v-model="styleDialogVisible"
      :title="$tr('管理告警分类 (Styles)')"
      width="650px"
      class="modern-dialog"
      :append-to-body="true"
      :modal="false"
    >
      <div style="margin-bottom: 10px">
        <el-button
          type="primary"
          size="small"
          icon="Plus"
          @click="openStyleForm()"
          >{{ $tr("新增分类") }}</el-button
        >
      </div>
      <el-table
        :data="styleList"
        border
        size="small"
        :header-cell-style="{
          background: 'rgba(102, 126, 234, 0.1)',
          color: 'var(--text-primary)',
          fontWeight: '600'
        }"
      >
        <el-table-column prop="name" :label="$tr('分类名称')" width="120" />
        <el-table-column
          prop="description"
          :label="$tr('描述')"
          min-width="150"
        />
        <el-table-column
          prop="CreatedAt"
          :label="$tr('创建时间')"
          width="160"
          align="center"
        >
          <template #default="{ row }">
            {{ row.CreatedAt ? new Date(row.CreatedAt).toLocaleString() : "-" }}
          </template>
        </el-table-column>
        <el-table-column :label="$tr('操作')" width="140" align="center">
          <template #default="{ row }">
            <div class="operation-buttons">
              <el-tooltip :content="$tr('编辑')" placement="top"
                ><span class="action-icon-trigger"
                  ><el-button
                    link
                    type="primary"
                    size="small"
                    icon="EditPen"
                    class="action-icon"
                    :aria-label="$tr('编辑')"
                    @click="openStyleForm(row)" /></span
              ></el-tooltip>
              <el-tooltip :content="$tr('删除')" placement="top"
                ><span class="action-icon-trigger"
                  ><el-button
                    link
                    type="danger"
                    size="small"
                    icon="Delete"
                    class="action-icon"
                    :aria-label="$tr('删除')"
                    @click="deleteStyle(row.ID || row.id)" /></span
              ></el-tooltip>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <!-- 新建/编辑分类内部弹窗 -->
    <el-dialog
      v-model="styleFormVisible"
      :title="styleForm.id ? '编辑分类' : $tr('新增分类')"
      width="400px"
      class="modern-dialog"
      append-to-body
    >
      <el-form ref="styleFormRef" :model="styleForm" label-width="80px">
        <el-form-item :label="$tr('分类名称')" required>
          <el-input v-model="styleForm.name" />
        </el-form-item>
        <el-form-item :label="$tr('描述')">
          <el-input v-model="styleForm.description" type="textarea" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="styleFormVisible = false">{{
          $tr("取消")
        }}</el-button>
        <el-button type="primary" @click="submitStyle">{{
          $tr("保存")
        }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { tr } from "@/locales";

import { ref, reactive, computed, onMounted, nextTick } from "vue";
import { ElMessage, ElMessageBox } from "element-plus";
import {
  getDataSources,
  getAlertGroupList,
  getAlertBusinessGroups,
  createAlertGroup,
  updateAlertGroup,
  deleteAlertGroup,
  getAlertRulesList,
  getRuleListByGroup,
  createAlertRule,
  updateAlertRule,
  deleteAlertRule,
  checkAlertRule,
  getAlertStyles,
  createAlertStyle,
  updateAlertStyle,
  deleteAlertStyle
} from "@/api/monitor";

// ---- 数据状态 ----
const promqlExample =
  "PromQL, 比如: 100 - (avg by (instance) (rate(node_cpu_seconds_total{mode='idle'}[5m])) * 100) > 90";
const dataSources = ref([]);
const activeDataSourceId = ref(null);

const groupList = ref([]);
const ruleList = ref([]);
const styleList = ref([]);
const filterCmdbGroup = ref("");

const businessGroups = ref([]);
const loadingBusinessGroups = ref(false);
const businessGroupsError = ref("");
const legacyBusinessLabel = ref("");
const cmdbGroups = computed(() => {
  const byId = new Map(businessGroups.value.map(g => [g.id, g]));
  return businessGroups.value.map(g => {
    const names = [g.name],
      seen = new Set([g.id]);
    let parent = byId.get(g.parentId);
    while (parent && !seen.has(parent.id)) {
      names.unshift(parent.name);
      seen.add(parent.id);
      parent = byId.get(parent.parentId);
    }
    return { ...g, path: names.join(" / ") };
  });
});
const fetchBusinessGroups = async () => {
  loadingBusinessGroups.value = true;
  businessGroupsError.value = "";
  try {
    const { data } = await getAlertBusinessGroups();
    if (data.code !== 200)
      throw new Error(data.message || tr("读取业务分组失败"));
    businessGroups.value = data.data || [];
  } catch (e) {
    businessGroupsError.value = e.message || tr("读取业务分组失败");
  } finally {
    loadingBusinessGroups.value = false;
  }
};
const getBusinessGroupLabel = group =>
  cmdbGroups.value.find(g => g.id === group.cmdb_group_id)?.name ||
  getClusterLabel(group.labels);
const handleBusinessFilter = () => {
  activeGroupId.value = "all";
  query.page = 1;
  fetchRules();
};

const activeGroupId = ref("");
const activeGroupName = computed(() => {
  const g = groupList.value.find(
    item => (item.ID || item.id) + "" === activeGroupId.value
  );
  return g ? g.group_name : "";
});

const loadingGroups = ref(false);
const loadingRules = ref(false);
const submitting = ref(false);

const query = reactive({
  alert: "",
  severity: "",
  status: "",
  style: "",
  enabled: "",
  page: 1,
  pageSize: 10
});
const totalRules = ref(0); // 总数

// 根据下拉框筛选 CMDB 业务线和数据源 (群组级别)
const filteredGroupList = computed(() => {
  let list = groupList.value;

  if (activeDataSourceId.value) {
    list = list.filter(g => g.data_source_id === activeDataSourceId.value);
  }

  if (!filterCmdbGroup.value) return list;

  return list.filter(g => g.cmdb_group_id === filterCmdbGroup.value);
});

// 工具：取群组 cluster 名称用于渲染 Tag
const getClusterLabel = labelsStr => {
  try {
    const l = JSON.parse(labelsStr);
    return l.cluster || "";
  } catch (e) {
    return "";
  }
};

const parseId = data => data.ID || data.id;

// ---- 初始化生命周期 ----
onMounted(async () => {
  await Promise.all([fetchDataSources(), fetchStyles(), fetchBusinessGroups()]);
  await fetchGroups();
});

// ---- HTTP/API 调用 ----
const fetchDataSources = async () => {
  try {
    const res = await getDataSources({ page: 1, pageSize: 100 });
    if (res.code === 200 || res.data) {
      console.log("数据源成功:");
      dataSources.value = res.data.data?.list || [];
      if (dataSources.value.length > 0 && !activeDataSourceId.value) {
        activeDataSourceId.value =
          dataSources.value[0].id || dataSources.value[0].ID;
      }
    }
  } catch (error) {
    console.error("获取数据源失败", error);
  }
};

const fetchStyles = async () => {
  try {
    const res = await getAlertStyles();
    if (res.code === 200 || res.data) {
      styleList.value = Array.isArray(res.data.data) ? res.data.data : [];
    }
  } catch (error) {
    console.error("获取分类失败", error);
  }
};

const fetchGroups = async () => {
  loadingGroups.value = true;
  try {
    const groups = [];
    let page = 1;
    while (true) {
      const { data } = await getAlertGroupList({ page, pageSize: 200 });
      if (data.code !== 200)
        throw new Error(data.message || tr("获取群组失败"));
      const items = data.data?.list || [];
      groups.push(...items);
      if (!items.length || groups.length >= data.data.total) break;
      page++;
    }
    groupList.value = groups;
    if (!activeGroupId.value) {
      activeGroupId.value = "all";
      await fetchRules();
    }
  } catch (error) {
    ElMessage.error(error.message || tr("获取群组失败"));
  } finally {
    loadingGroups.value = false;
  }
};

let rulesRequest = 0;
const fetchRules = async () => {
  const request = ++rulesRequest;
  loadingRules.value = true;
  try {
    const params = {
      ...query,
      dataSourceId: activeDataSourceId.value || undefined,
      cmdbGroupId: filterCmdbGroup.value || undefined
    };
    if (params.alert === "") delete params.alert;
    if (params.severity === "") delete params.severity;
    if (params.status === "") delete params.status;
    if (params.style === "") delete params.style;
    if (params.enabled === "") delete params.enabled;

    let res = null;
    if (activeGroupId.value && activeGroupId.value !== "all") {
      res = await getRuleListByGroup(activeGroupId.value, params);
    } else {
      res = await getAlertRulesList(params);
    }

    if (request !== rulesRequest) return;
    if (res.data.code !== 200)
      throw new Error(res.data.message || tr("获取规则失败"));
    ruleList.value = res.data.data?.list || [];
    totalRules.value = res.data.data?.total ?? 0;
  } catch (error) {
    if (request === rulesRequest) {
      ruleList.value = [];
      totalRules.value = 0;
      ElMessage.error(error.message || tr("获取规则失败"));
    }
  } finally {
    if (request === rulesRequest) loadingRules.value = false;
  }
};

// 侧边栏菜单切换组
const handleGroupSelect = index => {
  activeGroupId.value = index;
  resetQuery();
};

const resetQuery = () => {
  query.alert = "";
  query.severity = "";
  query.status = "";
  query.style = "";
  query.enabled = "";
  query.page = 1;
  fetchRules();
};

// ---- 群组管理 ----
const groupDialogVisible = ref(false);
const groupFormRef = ref(null);
const groupForm = reactive({
  id: null,
  group_name: "",
  data_source_id: undefined,
  cmdb_group_id: undefined
});
const groupRules = {
  group_name: [
    { required: true, message: tr("请输入群组名称"), trigger: "blur" }
  ],
  data_source_id: [
    { required: true, message: tr("请选择数据源"), trigger: "change" }
  ],
  cmdb_group_id: [
    { required: true, message: tr("请选择业务分组"), trigger: "change" }
  ]
};
const openGroupDialog = async (row = null) => {
  Object.assign(groupForm, {
    id: row ? parseId(row) : null,
    group_name: row?.group_name || "",
    data_source_id:
      row?.data_source_id || activeDataSourceId.value || undefined,
    cmdb_group_id: row?.cmdb_group_id || undefined
  });
  legacyBusinessLabel.value =
    row && !row.cmdb_group_id ? getClusterLabel(row.labels) : "";
  groupDialogVisible.value = true;
  await nextTick();
  groupFormRef.value?.clearValidate();
  await fetchBusinessGroups();
  if (
    row?.cmdb_group_id &&
    !businessGroupsError.value &&
    !cmdbGroups.value.some(g => g.id === row.cmdb_group_id)
  ) {
    groupForm.cmdb_group_id = undefined;
    legacyBusinessLabel.value = getClusterLabel(row.labels);
  }
};

const submitGroup = async () => {
  if (!groupFormRef.value) return;
  await groupFormRef.value.validate(async valid => {
    if (valid) {
      submitting.value = true;
      try {
        const action = groupForm.id ? updateAlertGroup : createAlertGroup;
        const payload = { ...groupForm };
        if (groupForm.id) payload.id = groupForm.id;

        const { data } = await action(payload);
        if (data.code !== 200)
          throw new Error(data.message || tr("保存群组失败"));
        ElMessage.success(tr(groupForm.id ? "编辑群组成功" : "新增群组成功"));
        groupDialogVisible.value = false;
        await fetchGroups();
        await fetchRules();
      } catch (error) {
        console.error(error);
        ElMessage.error(error.message || tr("保存群组失败"));
      } finally {
        submitting.value = false;
      }
    }
  });
};

const deleteGroup = id => {
  ElMessageBox.confirm(tr("确认删除该群组及其下属所有规则吗?"), tr("警告"), {
    type: "warning"
  })
    .then(async () => {
      try {
        await deleteAlertGroup(id);
        ElMessage.success(tr("删除组成功"));
        if (activeGroupId.value == id) {
          activeGroupId.value = "";
          ruleList.value = [];
        }
        fetchGroups();
      } catch (err) {
        ElMessage.error(tr("删除失败"));
      }
    })
    .catch(() => {});
};

// ---- 规则管理 ----
const ruleDialogVisible = ref(false);
const ruleFormRef = ref(null);
const ruleForm = reactive({
  id: null,
  group_id: null,
  alert: "",
  expr: "",
  for_duration: "5m",
  severity: "warning",
  summary: "",
  description: "",
  constraints: "{}",
  labels: "{}",
  style: "",
  enabled: 1
});

const validateExpr = (rule, value, callback) => {
  if (!value) return callback(new Error(tr("请输入告警表达式")));
  const stack = [];
  const pairs = { ")": "(", "}": "{", "]": "[" };
  let inString = false;
  let stringChar = "";

  for (let i = 0; i < value.length; i++) {
    const char = value[i];
    if (!inString && (char === '"' || char === "'" || char === "`")) {
      inString = true;
      stringChar = char;
      continue;
    } else if (inString && char === stringChar && value[i - 1] !== "\\") {
      inString = false;
      continue;
    }
    if (inString) continue;

    if (["(", "{", "["].includes(char)) {
      stack.push(char);
    } else if ([")", "}", "]"].includes(char)) {
      if (stack.length === 0 || stack.pop() !== pairs[char])
        return callback(new Error(`表达式格式有误: 括号 '${char}' 不匹配`));
    }
  }

  if (inString)
    return callback(new Error(tr("表达式格式有误: 存在未闭合的引号")));
  if (stack.length > 0)
    return callback(new Error(`表达式格式有误: 存在未闭合的括号`));

  callback();
};

const ruleRules = {
  alert: [
    {
      required: true,
      get message() {
        return tr("请输入规则名称");
      },
      trigger: "blur"
    }
  ],
  group_id: [
    {
      required: true,
      get message() {
        return tr("请选择所属群组");
      },
      trigger: "change"
    }
  ],
  expr: [{ required: true, validator: validateExpr, trigger: "blur" }],
  style: [
    {
      required: true,
      get message() {
        return tr("请选择分类");
      },
      trigger: "change"
    }
  ]
};

const openRuleDialog = (row = null) => {
  if (!activeGroupId.value)
    return ElMessage.warning(tr("请先选择一个告警群组"));
  if (row) {
    Object.keys(ruleForm).forEach(k => {
      ruleForm[k] = row[k] !== undefined ? row[k] : ruleForm[k];
    });
    ruleForm.id = parseId(row);
    ruleForm.group_id = parseInt(row.group_id || activeGroupId.value);
  } else {
    ruleForm.id = null;
    ruleForm.group_id =
      activeGroupId.value === "all" ? null : parseInt(activeGroupId.value);
    ruleForm.alert = "";
    ruleForm.expr = "";
    ruleForm.for_duration = "5m";
    ruleForm.severity = "warning";
    ruleForm.summary = "";
    ruleForm.description = "";
    ruleForm.constraints = '{"env": "prod"}';
    ruleForm.labels = '{"team": "devops"}';
    ruleForm.style = "";
    ruleForm.enabled = 1;
  }
  ruleDialogVisible.value = true;
};

const submitRule = async () => {
  if (!ruleFormRef.value) return;
  await ruleFormRef.value.validate(async valid => {
    if (valid) {
      submitting.value = true;
      try {
        const action = ruleForm.id ? updateAlertRule : createAlertRule;
        const payload = { ...ruleForm };
        if (ruleForm.id) payload.id = ruleForm.id;

        await action(payload);
        ElMessage.success(ruleForm.id ? "编辑规则成功" : "新增规则成功");
        ruleDialogVisible.value = false;
        fetchRules(); // 刷新列表
      } catch (error) {
        ElMessage.error(tr("保存规则失败"));
      } finally {
        submitting.value = false;
      }
    }
  });
};

const toggleRuleStatus = async row => {
  try {
    const payload = { ...row };
    payload.id = parseId(row);
    await updateAlertRule(payload);
    ElMessage.success(`告警规则已${row.enabled ? tr("启用") : tr("禁用")}`);
  } catch (error) {
    row.enabled = row.enabled ? 0 : 1; // 恢复状态
    ElMessage.error(tr("状态修改失败"));
  }
};

const checkRule = async row => {
  if (!row.expr) return ElMessage.warning(tr("规则表达式为空，无法检查"));
  try {
    const group = groupList.value.find(
      item => parseInt(item.ID || item.id) === parseInt(row.group_id)
    );
    const dataSourceId = group
      ? group.data_source_id
      : activeDataSourceId.value;

    if (!dataSourceId)
      return ElMessage.warning(tr("未找到对应的数据源，无法检查"));

    const res = await checkAlertRule({
      data_source_id: parseInt(dataSourceId),
      expr: row.expr
    });

    if (res.code === 200 || (res.data && res.data.code === 200)) {
      ElMessage.success(tr("检查通过，规则生效中"));
    } else {
      ElMessage.warning(
        "检查返回失败: " + (res.data?.message || "请确认表达式")
      );
    }
  } catch (error) {
    ElMessage.error(error.message || "检查发生错误，可能是由于表达式不合法");
  }
};

const deleteRule = id => {
  ElMessageBox.confirm(tr("确定删除此告警规则吗？"), tr("警告"), {
    type: "warning"
  })
    .then(async () => {
      try {
        await deleteAlertRule(id);
        ElMessage.success(tr("删除成功"));
        fetchRules();
      } catch (err) {
        ElMessage.error(tr("删除规则失败"));
      }
    })
    .catch(() => {});
};

// ---- 分类管理 ----
const styleDialogVisible = ref(false);
const styleFormVisible = ref(false);
const styleFormRef = ref(null);
const styleForm = reactive({ id: null, name: "", description: "" });

const openStyleDialog = () => {
  styleDialogVisible.value = true;
  fetchStyles();
};

const openStyleForm = (row = null) => {
  if (row) {
    styleForm.id = parseId(row);
    styleForm.name = row.name;
    styleForm.description = row.description;
  } else {
    styleForm.id = null;
    styleForm.name = "";
    styleForm.description = "";
  }
  styleFormVisible.value = true;
};

const submitStyle = async () => {
  if (!styleForm.name) return ElMessage.warning(tr("分类名称不能为空"));
  try {
    const action = styleForm.id ? updateAlertStyle : createAlertStyle;
    const payload = { ...styleForm };
    if (styleForm.id) payload.id = styleForm.id;

    await action(payload);
    ElMessage.success(tr("保存分类成功"));
    styleFormVisible.value = false;
    fetchStyles();
  } catch (error) {
    ElMessage.error(tr("保存分类失败"));
  }
};

const deleteStyle = id => {
  ElMessageBox.confirm(tr("确定删除该分类吗？"), tr("警告"), {
    type: "warning"
  })
    .then(async () => {
      try {
        await deleteAlertStyle(id);
        ElMessage.success(tr("删除成功"));
        fetchStyles();
      } catch (err) {}
    })
    .catch(() => {});
};

const getDataSourceIcon = id => {
  const ds = dataSources.value.find(item => (item.id || item.ID) === id);
  return ds ? getDsTypeIcon(ds.type) : "";
};

const getDsTypeIcon = type => {
  const t = (type || "").toLowerCase();
  if (t.includes("prometheus")) return "Monitor";
  if (t.includes("zabbix")) return "Odometer";
  if (t.includes("loki")) return "DataLine";
  return "Coin";
};

const getDeployIcon = method => {
  const m = (method || "").toLowerCase();
  if (m.includes("docker")) return "Coin";
  if (m.includes("kubernetes") || m.includes("kubenetes")) return "Cpu";
  return "";
};

const handleDataSourceChange = () => {
  query.page = 1;
  activeGroupId.value = "all";
  fetchRules();
};

// ---- 辅助函数 ----
const getSeverityType = severity => {
  switch (severity?.toLowerCase()) {
    case "critical":
      return "danger";
    case "warning":
      return "warning";
    case "info":
      return "info";
    default:
      return "";
  }
};

const getGroupName = id => {
  const g = groupList.value.find(item => (item.ID || item.id) + "" === id + "");
  return g ? g.group_name : `ID: ${id}`;
};
</script>

<style scoped>
.alarm-rules-management {
  padding: 20px;
  min-height: calc(100vh - 120px);
  background: var(--bg-page);
}

.modern-card {
  background: var(--bg-card);
  backdrop-filter: blur(10px);
  border-radius: 16px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.2);
  display: flex;
  flex-direction: column;
}

.menu-card {
  height: 100%;
}

.main-card {
  height: 100%;
}

:deep(.modern-card .el-card__body) {
  flex: 1;
  overflow: auto;
  padding: 15px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
  display: flex;
  align-items: center;
  gap: 8px;
}

.transparent-bg {
  background-color: transparent !important;
  --el-menu-bg-color: transparent;
  --el-menu-text-color: var(--text-primary);
  --el-menu-hover-text-color: var(--text-primary);
  --el-menu-hover-bg-color: var(--bg-card-alt);
  --el-menu-active-color: var(--primary);
  --el-menu-item-bg-color: transparent;
}

.transparent-bg :deep(.el-menu-item) {
  color: var(--text-primary) !important;
  background-color: transparent !important;
  transition: all 0.2s ease;
  padding-right: 8px !important;
}

.transparent-bg :deep(.el-menu-item:hover) {
  background-color: rgba(102, 126, 234, 0.1) !important;
  color: var(--primary) !important;
}

.transparent-bg :deep(.el-menu-item.is-active) {
  color: var(--primary) !important;
  background-color: var(--bg-card-alt) !important;
  font-weight: 600;
}

.group-menu {
  border-right: none;
}
.menu-item-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
  min-width: 0;
}
.group-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: flex;
  align-items: center;
}

.cluster-tag {
  margin-left: 6px;
  transform: scale(0.85);
  transform-origin: left center;
}

.group-actions {
  display: flex;
  opacity: 0;
  pointer-events: none;
  align-items: center;
  gap: 2px;
  margin-left: auto;
  flex-shrink: 0;
  padding-left: 8px;
}
.el-menu-item:hover .group-actions,
.el-menu-item:focus-within .group-actions {
  opacity: 1;
  pointer-events: auto;
}

/* 搜索表单部分 */
.search-section {
  margin-bottom: 16px;
  padding: 16px;
  background: rgba(103, 126, 234, 0.05);
  border-radius: 12px;
  border: 1px solid rgba(103, 126, 234, 0.1);
}

.search-form .el-form-item {
  margin-bottom: 12px;
  margin-right: 16px;
}

.search-form .el-form-item__label {
  color: var(--text-regular);
  font-weight: 500;
}

/* 表格美化 */
:deep(.el-table) {
  background: rgba(255, 255, 255, 0.9);
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);
}

:deep(.el-table__header th) {
  border: none;
}

:deep(.el-table__body tr:hover > td) {
  background-color: rgba(102, 126, 234, 0.1) !important;
}

:deep(.el-table td) {
  border: none;
}

:deep(.el-table::before) {
  display: none;
}
:deep(.el-table--border::after) {
  display: none;
}

.operation-buttons {
  display: flex;
  gap: 8px;
  justify-content: center;
}

.operation-buttons .el-button {
  transition: all 0.3s ease;
}

.operation-buttons .el-button:hover {
  transform: translateY(-2px);
}

/* 按钮美化 */
:deep(.el-button--primary) {
  background: var(--bg-card-alt);
  border: none;
  border-radius: 8px;
  transition: all 0.3s ease;
}

:deep(.el-button--primary:hover) {
  background: linear-gradient(135deg, #5a6fd8, #6a4190);
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(102, 126, 234, 0.4);
}

:deep(.el-button--warning) {
  background: linear-gradient(135deg, #f39c12, #e67e22);
  border: none;
  border-radius: 8px;
  color: white;
  transition: all 0.3s ease;
}

:deep(.el-button--warning:hover) {
  background: linear-gradient(135deg, #e67e22, #d35400);
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(243, 156, 18, 0.4);
}

.promql-code {
  background-color: rgba(103, 126, 234, 0.1);
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 13px;
  color: #c7254e;
  font-family: Menlo, Monaco, Consolas, "Courier New", monospace;
  word-break: break-all;
  white-space: normal;
}

.pagination-section {
  display: flex;
  justify-content: center;
  margin-top: 20px;
}

/* 对话框美化 */
:deep(.modern-dialog .el-dialog) {
  background: var(--bg-card);
  backdrop-filter: blur(20px);
  border-radius: 16px;
  border: 1px solid rgba(255, 255, 255, 0.2);
  box-shadow: 0 8px 32px rgba(31, 38, 135, 0.37);
}

:deep(.modern-dialog .el-dialog__header) {
  background: linear-gradient(
    135deg,
    rgba(102, 126, 234, 0.1),
    rgba(118, 75, 162, 0.1)
  );
  border-radius: 16px 16px 0 0;
  padding: 20px 24px 16px;
  border-bottom: 1px solid rgba(102, 126, 234, 0.2);
}

:deep(.modern-dialog .el-dialog__title) {
  color: var(--text-primary);
  font-weight: 600;
  font-size: 18px;
}

:deep(.modern-dialog .el-dialog__body) {
  padding: 24px;
}

:deep(.modern-dialog .el-dialog__footer) {
  padding: 16px 24px 24px;
  background: rgba(248, 249, 250, 0.8);
  border-radius: 0 0 16px 16px;
}

.help-text {
  font-size: 12px;
  color: #999;
  line-height: 1.2;
  margin-top: 5px;
}

/* 告警表单美化 */
.form-section-title {
  font-size: 15px;
  font-weight: 600;
  color: #5a6fd8;
  margin: 10px 0 20px 0;
  display: flex;
  align-items: center;
  gap: 8px;
  padding-bottom: 8px;
  border-bottom: 1px solid rgba(102, 126, 234, 0.2);
}

.promql-input :deep(textarea) {
  font-family: Menlo, Monaco, Consolas, "Courier New", monospace;
  background-color: #1e1e1f;
  color: #d4d4d4;
  border: 1px solid #333;
  padding: 12px;
  font-size: 14px;
  line-height: 1.6;
  border-radius: 8px;
  box-shadow: inset 0 2px 4px rgba(0, 0, 0, 0.2);
  transition: border 0.3s;
}

.promql-input :deep(textarea):focus {
  border-color: var(--primary);
  box-shadow:
    inset 0 2px 4px rgba(0, 0, 0, 0.2),
    0 0 0 2px rgba(102, 126, 234, 0.2);
}

.expr-form-item :deep(.el-form-item__label) {
  font-weight: 600;
  color: #e74c3c;
}

/* 数据源下拉框样式 */
.ds-selector {
  margin-bottom: 15px !important;
  background: rgba(255, 255, 255, 0.4);
  border-radius: 10px;
  padding: 4px;
  border: 1px solid rgba(102, 126, 234, 0.2);
}

.ds-select {
  --el-select-input-focus-border-color: transparent;
}

.ds-select :deep(.el-input__wrapper) {
  box-shadow: none !important;
  background-color: transparent;
}

.ds-icon-small {
  width: 20px;
  height: 20px;
  margin-right: 6px;
  object-fit: contain;
  filter: drop-shadow(0 2px 4px rgba(0, 0, 0, 0.1));
}

.ds-option {
  height: auto !important;
  padding: 10px 12px;
}

.ds-option-content {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
}

.ds-icon {
  width: 26px;
  height: 26px;
  object-fit: contain;
  filter: drop-shadow(0 2px 5px rgba(0, 0, 0, 0.15));
}

.ds-name {
  font-weight: 600;
  color: var(--text-primary);
  font-size: 14px;
  flex: 1;
}

.group-colored-icon {
  width: 22px;
  height: 22px;
  margin-right: 12px;
  object-fit: contain;
  filter: drop-shadow(0 2px 4px rgba(0, 0, 0, 0.15));
  transition: transform 0.3s ease;
}

.el-menu-item:hover .group-colored-icon {
  transform: scale(1.15) translateY(-2px);
}

.ds-deploy-tag {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  border-radius: 20px;
  padding: 0 10px;
  height: 24px;
  border: none;
  background: rgba(102, 126, 234, 0.1);
  color: #5a6fd8;
  font-weight: 500;
}

.deploy-icon {
  width: 14px;
  height: 14px;
  object-fit: contain;
}

/* 代码编辑器包装器 */
.code-editor-container {
  background: #1e1e1f;
  border-radius: 8px;
  overflow: hidden;
  border: 1px solid #333;
  margin-top: 10px;
}

.code-editor-header {
  height: 30px;
  background: #252526;
  border-bottom: 1px solid #333;
  display: flex;
  align-items: center;
  padding: 0 12px;
  position: relative;
}

.mac-dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  margin-right: 6px;
  display: inline-block;
}

.mac-dot.close {
  background: #ff5f56;
}
.mac-dot.minimize {
  background: #ffbd2e;
}
.mac-dot.maximize {
  background: #27c93f;
}

.file-name {
  position: absolute;
  left: 50%;
  transform: translateX(-50%);
  color: #9cdcfe;
  font-size: 13px;
  font-family: Menlo, Monaco, Consolas, "Courier New", monospace;
  opacity: 0.8;
}

.code-editor-container .promql-input :deep(textarea) {
  border: none;
  border-radius: 0 0 8px 8px;
  resize: vertical;
  padding-top: 12px;
  background: #1e1e1f !important;
  box-shadow: none;
}
.code-editor-container .promql-input :deep(textarea):focus {
  box-shadow: none;
}
@media (hover: none) {
  .group-actions {
    opacity: 1;
    pointer-events: auto;
  }
}
@media (max-width: 991px) {
  .alarm-rules-management {
    padding: 0;
  }
  .menu-card {
    margin-bottom: 20px;
    height: auto;
  }
  .card-header {
    flex-wrap: wrap;
    gap: 12px;
  }
  .group-menu {
    max-height: 320px;
    overflow-y: auto;
  }
}
</style>
