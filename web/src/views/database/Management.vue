<template>
  <div class="db-management" :class="{ 'embedded-management': embedded }">
    <header class="db-hero">
      <div v-if="!embedded">
        <small>DATABASE OPERATIONS</small>
        <h1>{{ title }}</h1>
        <p>{{ descriptions[section] }}</p>
      </div>
      <div class="hero-actions">
        <DatabaseEngineFilter v-model="filters.engine" @change="changeEngine" />
        <el-button
          v-if="section === 'clusters' && platform.isAdmin"
          type="primary"
          @click="createOpen = true"
          >{{ $tr("新建 / 接管集群") }}</el-button
        >
        <el-button
          v-if="section === 'storage' && platform.isAdmin"
          @click="thresholdsOpen = true"
          >{{ $tr("容量阈值") }}</el-button
        >
        <el-button
          v-if="section === 'audit' && platform.isAdmin"
          @click="openAuditPolicy"
          >{{ $tr("日志保存周期") }}</el-button
        >
        <div v-if="section === 'backups'">
          <el-button
            @click="
              settingsKind = 'policy';
              editingPolicy = undefined;
              settingsOpen = true;
            "
            >{{ $tr("创建备份策略") }}</el-button
          ><el-button
            v-if="platform.isAdmin"
            @click="
              settingsKind = 'repository';
              settingsOpen = true;
            "
            >{{ $tr("登记备份仓库") }}</el-button
          >
        </div>
      </div>
    </header>
    <BackupGuide v-if="section === 'backups'" :engine="filters.engine" />
    <el-card shadow="never"
      ><div class="db-filters">
        <el-input
          v-model="filters.name"
          clearable
          :placeholder="
            recordTab === 'repositories' && section === 'backups'
              ? $tr('仓库名称')
              : section === 'backups' && isBackupTab
                ? $tr('集群或备份名称')
                : $tr('集群名称')
          "
          @keyup.enter="search"
        /><el-tree-select
          v-model="filters.groupId"
          :data="groups"
          :props="{ label: 'name', value: 'id', children: 'children' }"
          check-strictly
          clearable
          filterable
          :placeholder="$tr('业务分组（含子分组）')"
        /><el-input
          v-model="filters.environment"
          clearable
          :placeholder="$tr('环境')"
        /><el-input
          v-if="section === 'backups' && recordTab === 'repositories'"
          v-model="filters.location"
          clearable
          :placeholder="$tr('位置（目录 / Endpoint / Bucket）')"
          @keyup.enter="search"
        /><el-select
          v-if="filterStatuses.length"
          v-model="filters.status"
          clearable
          :placeholder="$tr('状态')"
          ><el-option
            v-for="status in filterStatuses"
            :key="status"
            :value="status"
            :label="label(status)" /></el-select
        ><el-button type="primary" :loading="loading" @click="search">{{
          $tr("查询")
        }}</el-button>
      </div>
      <el-tag
        v-if="filters.clusterId"
        closable
        class="cluster-scope"
        @close="clearClusterScope"
        >{{ $tr("当前集群：") }}{{ clusterName(filters.clusterId) }}</el-tag
      >
      <el-alert
        v-if="error"
        :title="$tr(error)"
        type="error"
        :closable="false"
        show-icon
      />
      <el-table
        v-if="section === 'clusters'"
        :data="clusters"
        v-loading="loading"
        row-key="id"
        class="asset-table"
        @row-click="openCluster"
      >
        <el-table-column :label="$tr('集群')" min-width="280"
          ><template #default="{ row }"
            ><div class="cluster-identity">
              <DatabaseEngineIcon :engine="row.engine" />
              <div>
                <el-tooltip :content="row.name" placement="top"
                  ><el-button
                    link
                    type="primary"
                    @click.stop="openCluster(row)"
                    >{{ row.name }}</el-button
                  ></el-tooltip
                >
                <small class="db-muted"
                  >{{ row.topology }} ·
                  {{ row.environment || $tr("未指定环境") }}</small
                >
              </div>
            </div></template
          ></el-table-column
        >
        <el-table-column :label="$tr('业务分组')" min-width="130"
          ><template #default="{ row }">{{
            groupName(row.groupId)
          }}</template></el-table-column
        ><el-table-column prop="engine" :label="$tr('引擎')" width="115" />
        <el-table-column :label="$tr('管理状态')" width="120"
          ><template #default="{ row }">{{
            label(row.management)
          }}</template></el-table-column
        ><el-table-column :label="$tr('成员')" width="80"
          ><template #default="{ row }">{{
            row.members?.length || 0
          }}</template></el-table-column
        >
        <el-table-column :label="$tr('健康')" width="120"
          ><template #default="{ row }"
            ><el-tag :type="tagType(row.status)">{{
              label(row.status)
            }}</el-tag></template
          ></el-table-column
        >
        <el-table-column :label="$tr('访问入口')" min-width="200"
          ><template #default="{ row }"
            ><div v-for="ep in (row.endpoints || []).slice(0, 2)" :key="ep.id">
              <el-tooltip
                :content="ep.addresses?.join(', ')"
                placement="top"
                :disabled="(ep.addresses?.length || 0) < 2"
              >
                <span class="db-muted"
                  >{{ ep.addresses?.[0]
                  }}<span v-if="ep.addresses?.length > 1">
                    {{ $tr("· 共") }} {{ ep.addresses.length }}
                    {{ $tr("个地址") }}</span
                  ></span
                >
              </el-tooltip>
            </div>
            <span v-if="!row.endpoints?.length">{{
              $tr("未配置")
            }}</span></template
          ></el-table-column
        >
        <el-table-column
          v-if="section === 'clusters'"
          :label="$tr('最近备份')"
          min-width="165"
          ><template #default="{ row }"
            ><template v-if="row.latestBackup"
              ><el-tag size="small" :type="tagType(row.latestBackup.status)">{{
                label(row.latestBackup.status)
              }}</el-tag
              ><small class="db-muted">{{
                date(row.latestBackup.createdAt)
              }}</small></template
            ><span v-else>{{ $tr("暂无备份") }}</span></template
          ></el-table-column
        >
        <el-table-column
          v-if="section === 'clusters'"
          :label="$tr('当前任务')"
          min-width="145"
          ><template #default="{ row }"
            ><el-button
              v-if="row.currentOperation"
              link
              type="primary"
              @click.stop="
                taskId = row.currentOperation.id;
                taskOpen = true;
              "
              >#{{ row.currentOperation.id }} ·
              {{ operationStatusLabel(row.currentOperation.status) }}</el-button
            ><span v-else>{{ $tr("无") }}</span></template
          ></el-table-column
        >
        <el-table-column :label="$tr('最后采集')" min-width="180"
          ><template #default="{ row }">{{
            date(row.collectedAt)
          }}</template></el-table-column
        >
        <el-table-column :label="$tr('操作')" width="160" fixed="right">
          <template #default="{ row }"
            ><span class="row-actions" @click.stop>
              <el-button link type="primary" @click="openCluster(row)">{{
                $tr("详情")
              }}</el-button>
              <ClusterActions
                @detail="tab => openCluster(row, tab)"
                :cluster-id="row.id"
                :cluster-name="row.name"
                :context="section"
                @task="showTask"
                @changed="load"
              /> </span
          ></template>
        </el-table-column>
      </el-table>
      <div
        v-if="
          section === 'storage' ||
          (section === 'backups' && recordTab !== 'repositories')
        "
        class="module-toolbar"
      >
        <el-select
          v-model="operationClusterId"
          filterable
          :placeholder="$tr('选择操作集群')"
          clearable
          @change="selectOperationCluster"
          @visible-change="loadOperationClusters"
        >
          <el-option
            v-for="c in operationClusters"
            :key="c.id"
            :value="c.id"
            :label="c.name"
          />
        </el-select>
        <BackupLauncher
          v-if="section === 'backups'"
          :cluster-id="operationClusterId"
          @task="showTask"
        />
        <ClusterActions
          v-if="operationClusterId && section === 'storage'"
          :key="`${section}-${operationClusterId}`"
          :cluster-id="operationClusterId"
          :context="section"
          @task="showTask"
          @changed="load"
        />
      </div>
      <template v-if="section === 'storage'"
        ><el-tabs v-model="storageTab" @tab-change="search"
          ><el-tab-pane :label="$tr('容量总览')" name="all" /><el-tab-pane
            :label="$tr('实例存储')"
            name="instance" /><el-tab-pane
            :label="$tr('文件系统')"
            name="filesystem"
        /></el-tabs>
        <p class="db-note">
          {{
            $tr(
              "逻辑数据量与副本占用分别统计；共享文件系统按主机与设备去重。采集过期不显示正常状态。"
            )
          }}
        </p>
        <el-table :data="storageRows" v-loading="loading"
          ><el-table-column type="expand"
            ><template #default="{ row }"
              ><StorageDetails
                :detail="row.details || {}" /></template></el-table-column
          ><el-table-column :label="$tr('集群 / 实例')" min-width="190"
            ><template #default="{ row }"
              ><div class="cluster-identity">
                <DatabaseEngineIcon :engine="row.engine" />
                <div>
                  <el-button
                    link
                    type="primary"
                    @click="openById(row.clusterId)"
                    >{{
                      row.clusterName || clusterName(row.clusterId)
                    }}</el-button
                  ><small class="db-muted" :title="row.name">{{
                    row.name
                  }}</small>
                </div>
              </div></template
            ></el-table-column
          ><el-table-column :label="$tr('类型')" width="120"
            ><template #default="{ row }">{{
              row.kind === "filesystem" ? $tr("文件系统") : $tr("实例占用")
            }}</template></el-table-column
          ><el-table-column :label="$tr('已用 / 总量')" min-width="200"
            ><template #default="{ row }"
              >{{ bytes(row.usedBytes) }} /
              {{ row.totalBytes ? bytes(row.totalBytes) : $tr("未配置上限")
              }}<el-progress
                v-if="row.totalBytes"
                :percentage="
                  Math.min(
                    100,
                    Math.round((row.usedBytes / row.totalBytes) * 100)
                  )
                "
                :status="
                  (row.usedBytes / row.totalBytes) * 100 >=
                  (row.details?.criticalPercent || 90)
                    ? 'exception'
                    : (row.usedBytes / row.totalBytes) * 100 >=
                        (row.details?.warningPercent || 80)
                      ? 'warning'
                      : undefined
                " /></template></el-table-column
          ><el-table-column :label="$tr('状态')" min-width="180"
            ><template #default="{ row }"
              ><span v-if="row.details?.error">{{ row.details.error }}</span
              ><el-tag
                v-else
                :type="row.details?.stale ? 'warning' : 'success'"
                >{{
                  row.details?.stale
                    ? $tr("状态未知（采集过期）")
                    : $tr("已采集")
                }}</el-tag
              ></template
            ></el-table-column
          ><el-table-column :label="$tr('采集时间')" min-width="180"
            ><template #default="{ row }">{{
              date(row.collectedAt)
            }}</template></el-table-column
          ><el-table-column :label="$tr('操作')" width="100" fixed="right"
            ><template #default="{ row }">
              <ClusterActions
                :cluster-id="row.clusterId"
                :member-id="row.memberId"
                context="storage"
                @task="showTask"
                @changed="load"
              /> </template></el-table-column></el-table
        ><el-empty
          v-if="!records.length && !loading"
          :description="$tr('选择集群执行容量采集后显示真实数据')"
      /></template>
      <template v-if="section === 'backups'"
        ><el-tabs v-model="recordTab" @tab-change="search"
          ><el-tab-pane :label="$tr('备份集')" name="backups" /><el-tab-pane
            :label="$tr('恢复演练')"
            name="verification" /><el-tab-pane
            :label="$tr('备份策略')"
            name="backup-policies" /><el-tab-pane
            :label="$tr('备份仓库')"
            name="repositories"
        /></el-tabs>
        <p v-if="recordTab === 'repositories'" class="db-note">
          {{
            $tr(
              "仓库按业务分组共享，不区分数据库类型。登记后先执行连接检查，再在备份或策略中选择。"
            )
          }}
        </p>
        <p v-else-if="recordTab === 'backup-policies'" class="db-note">
          {{
            $tr(
              "策略按 Worker 所在时区调度；下一次执行时间按当前浏览器时区显示。创建策略不会立即执行备份。"
            )
          }}
        </p>
        <p v-else class="db-note">
          {{
            recordTab === "verification"
              ? $tr(
                  "这里展示生成完成、等待恢复验证的备份。通过行操作执行恢复验证，成功后自动进入备份集。失败任务请在基础管理 → 数据日志 → 数据库操作审计查看。"
                )
              : $tr(
                  "仅展示完整且通过实际恢复验证的备份。新备份请到恢复演练验证；清理备份可删除尚未过期的指定备份。"
                )
          }}
        </p>
        <el-table
          v-if="recordTab === 'repositories'"
          :data="records"
          v-loading="loading"
          ><el-table-column :label="$tr('仓库')"
            ><template #default="{ row }"
              ><div class="cluster-identity">
                <el-icon :size="26"><FolderOpened /></el-icon
                ><span>{{ row.name }}</span>
              </div></template
            ></el-table-column
          ><el-table-column prop="kind" :label="$tr('类型')" /><el-table-column
            :label="$tr('位置')"
            ><template #default="{ row }">{{
              row.kind === "local" ? row.path : `${row.endpoint}/${row.bucket}`
            }}</template></el-table-column
          ><el-table-column :label="$tr('业务分组')"
            ><template #default="{ row }">{{
              groupName(row.groupId)
            }}</template></el-table-column
          ><el-table-column :label="$tr('状态')"
            ><template #default="{ row }"
              ><el-tag :type="tagType(row.status)">{{
                label(row.status)
              }}</el-tag></template
            ></el-table-column
          ><el-table-column :label="$tr('操作')" min-width="200"
            ><template #default="{ row }"
              ><el-button
                link
                type="primary"
                @click="
                  repositoryId = row.id;
                  repositoryDetailOpen = true;
                "
                >{{ $tr("详情") }}</el-button
              >
              <el-button
                v-if="platform.isAdmin"
                link
                type="primary"
                @click="checkRepository(row.id)"
                >{{ $tr("连接检查") }}</el-button
              ><el-button
                v-if="platform.isAdmin"
                link
                type="danger"
                @click="deleteRepository(row)"
                >{{ $tr("删除") }}</el-button
              ></template
            ></el-table-column
          ></el-table
        ><el-table v-else :data="records" v-loading="loading"
          ><el-table-column
            v-if="isBackupTab"
            :label="$tr('备份名称')"
            min-width="240"
            show-overflow-tooltip
            ><template #default="{ row }">{{
              backupDisplayName(row)
            }}</template></el-table-column
          >
          <el-table-column :label="$tr('集群')" min-width="180"
            ><template #default="{ row }"
              ><div class="cluster-identity">
                <DatabaseEngineIcon :engine="row.engine" /><el-button
                  link
                  type="primary"
                  @click="openById(row.clusterId)"
                  >{{
                    row.clusterName || clusterName(row.clusterId)
                  }}</el-button
                >
              </div></template
            ></el-table-column
          ><el-table-column
            v-if="recordTab === 'backup-policies'"
            prop="name"
            :label="$tr('策略名称')"
            min-width="170"
            show-overflow-tooltip
          /><el-table-column :label="$tr('备份方式')" min-width="170"
            ><template #default="{ row }">{{
              backupMethodLabel(row.method)
            }}</template></el-table-column
          ><el-table-column v-if="isBackupTab" :label="$tr('状态')"
            ><template #default="{ row }"
              ><el-tag :type="tagType(row.status)">{{
                label(row.status)
              }}</el-tag></template
            ></el-table-column
          ><el-table-column
            v-if="isBackupTab"
            :label="$tr('副本位置')"
            min-width="150"
            ><template #default="{ row }">{{
              row.manifest?.remoteComplete
                ? $tr("独立仓库已核验")
                : row.manifest?.localComplete
                  ? $tr("本地暂存")
                  : $tr("未完成")
            }}</template></el-table-column
          ><el-table-column
            v-if="isBackupTab"
            :label="$tr('恢复验证')"
            min-width="180"
            ><template #default="{ row }">{{
              row.verifiedAt ? date(row.verifiedAt) : $tr("尚未验证")
            }}</template></el-table-column
          ><el-table-column
            v-else
            prop="cron"
            :label="$tr('调度（Cron）')"
            min-width="120"
          /><el-table-column
            v-if="recordTab === 'backup-policies'"
            :label="$tr('下次执行')"
            min-width="180"
            ><template #default="{ row }">{{
              row.enabled ? date(row.nextRunAt) : $tr("已暂停")
            }}</template></el-table-column
          ><el-table-column
            v-if="recordTab === 'backup-policies'"
            :label="$tr('保留')"
            width="90"
            ><template #default="{ row }"
              >{{ row.retentionDays }} {{ $tr("天") }}</template
            ></el-table-column
          ><el-table-column
            v-if="recordTab === 'backup-policies'"
            :label="$tr('启用')"
            ><template #default="{ row }"
              ><el-switch
                :model-value="row.enabled"
                @change="
                  value => togglePolicy(row, value)
                " /></template></el-table-column
          ><el-table-column :label="$tr('创建时间')" min-width="180"
            ><template #default="{ row }">{{
              date(row.createdAt)
            }}</template></el-table-column
          ><el-table-column
            v-if="isBackupTab"
            :label="$tr('操作')"
            width="180"
            fixed="right"
          >
            <template #default="{ row }"
              ><span class="row-actions">
                <el-button
                  link
                  type="primary"
                  @click="
                    selectedBackup = row;
                    backupDetailOpen = true;
                  "
                  >{{ $tr("详情") }}</el-button
                >
                <ClusterActions
                  :cluster-id="row.clusterId"
                  :backup-id="row.id"
                  :backup-status="row.status"
                  context="backups"
                  button-label="操作"
                  @task="showTask"
                  @changed="load"
                /> </span
            ></template>
          </el-table-column>
          <el-table-column
            v-if="recordTab === 'backup-policies'"
            :label="$tr('操作')"
            width="100"
            fixed="right"
            ><template #default="{ row }"
              ><el-button
                link
                type="primary"
                @click="
                  editingPolicy = row;
                  settingsKind = 'policy';
                  settingsOpen = true;
                "
                >{{ $tr("编辑") }}</el-button
              ></template
            ></el-table-column
          >
        </el-table>
      </template>
      <template v-if="section === 'audit'"
        ><el-tabs v-model="auditTab" @tab-change="search"
          ><el-tab-pane
            :label="$tr('运维任务')"
            name="operations" /><el-tab-pane
            :label="$tr('事件与变更')"
            name="events" /></el-tabs
        ><el-table :data="records" v-loading="loading"
          ><el-table-column prop="id" label="ID" width="80" /><el-table-column
            :label="$tr('集群')"
            min-width="170"
            ><template #default="{ row }">{{
              row.clusterName || clusterName(row.clusterId)
            }}</template></el-table-column
          ><el-table-column :label="$tr('操作 / 事件')" min-width="150"
            ><template #default="{ row }"
              ><el-button
                v-if="auditTab === 'operations'"
                link
                type="primary"
                @click="showTask(row.id)"
                >{{ row.action }}</el-button
              ><span v-else>{{ row.kind }}</span></template
            ></el-table-column
          ><el-table-column :label="$tr('状态')" width="120"
            ><template #default="{ row }"
              ><el-tag :type="tagType(row.status)">{{
                auditTab === "operations"
                  ? operationStatusLabel(row.status)
                  : label(row.status || row.severity)
              }}</el-tag></template
            ></el-table-column
          ><el-table-column
            :label="$tr('结果 / 原因')"
            min-width="280"
            show-overflow-tooltip
            ><template #default="{ row }">{{
              operationMessage(row.message)
            }}</template></el-table-column
          ><el-table-column :label="$tr('时间')" min-width="180"
            ><template #default="{ row }">{{
              date(row.createdAt)
            }}</template></el-table-column
          ><el-table-column :label="$tr('操作')" width="180" fixed="right"
            ><template #default="{ row }"
              ><span class="row-actions">
                <el-button
                  v-if="auditTab === 'operations'"
                  link
                  type="primary"
                  @click="showTask(row.id)"
                  >{{ $tr("步骤与日志") }}</el-button
                >
                <el-button
                  v-if="
                    platform.isAdmin &&
                    (auditTab === 'events' ||
                      ['succeeded', 'failed', 'cancelled'].includes(row.status))
                  "
                  link
                  type="danger"
                  @click="deleteAudit(row)"
                  >{{ $tr("删除") }}</el-button
                >
                <ClusterActions
                  v-if="row.status === 'unknown'"
                  :cluster-id="row.clusterId"
                  context="audit"
                  @task="showTask"
                  @changed="load"
                /> </span></template></el-table-column></el-table
      ></template>
      <el-pagination
        v-model:current-page="filters.page"
        v-model:page-size="filters.size"
        :page-sizes="[10, 20, 50, 100]"
        :total="section === 'clusters' ? total : recordTotal"
        layout="total, sizes, prev, pager, next, jumper"
        @current-change="load"
        @size-change="search"
      />
    </el-card>
    <ClusterDetail
      v-if="selected"
      v-model="detailOpen"
      :cluster-id="selected"
      :initial-tab="detailTab"
      @task="showTask"
    />
    <BackupDetail
      v-model="backupDetailOpen"
      :backup="selectedBackup"
      @task="showTask"
    />
    <ClusterCreate v-model="createOpen" :groups="groups" @created="created" />
    <TaskDetail v-model="taskOpen" :operation-id="taskId" @finished="load" />
    <StorageThresholds
      v-model="thresholdsOpen"
      :groups="groups"
      @saved="load"
    />
    <el-dialog
      v-model="auditPolicyOpen"
      :title="$tr('日志保存周期')"
      width="min(480px, 94vw)"
    >
      <el-form label-width="100px"
        ><el-form-item :label="$tr('保存天数')"
          ><el-input-number
            v-model="retentionDays"
            :min="0"
            :max="3650" /></el-form-item
      ></el-form>
      <p class="db-note">
        {{
          $tr(
            "0 表示永久保存。到期清理已结束任务的日志与事件，运行中及待核验任务保留；备份文件不受影响。"
          )
        }}
      </p>
      <template #footer
        ><el-button @click="auditPolicyOpen = false">{{
          $tr("取消")
        }}</el-button
        ><el-button type="primary" @click="saveAuditPolicy">{{
          $tr("确认保存")
        }}</el-button></template
      >
    </el-dialog>
    <RepositoryDetail
      v-model="repositoryDetailOpen"
      :repository-id="repositoryId"
      @changed="load"
    />
    <BackupSettings
      v-model="settingsOpen"
      :kind="settingsKind"
      :engine="filters.engine"
      :cluster-id="operationClusterId"
      :policy="editingPolicy"
      :groups="groups"
      @saved="load"
    />
  </div>
</template>
<script setup lang="ts">
import { tr } from "@/locales";

import { ElMessageBox, ElMessage } from "element-plus";
import { FolderOpened } from "@element-plus/icons-vue";
import RepositoryDetail from "./components/RepositoryDetail.vue";
import BackupGuide from "./components/BackupGuide.vue";
import BackupDetail from "./components/BackupDetail.vue";
import BackupLauncher from "./components/BackupLauncher.vue";
import { backupDisplayName, backupMethodLabel } from "@/utils/databaseBackup";
import StorageDetails from "./components/StorageDetails.vue";
import { computed, onMounted, onUnmounted, ref, reactive, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { platformAccess as platform } from "@/utils/platformAccess";
import {
  dbPlatform,
  dbRequest,
  label,
  operationStatusLabel,
  operationMessage,
  tagType,
  bytes,
  type Cluster
} from "@/api/databasePlatform";
import DatabaseEngineIcon from "./components/DatabaseEngineIcon.vue";
import ClusterActions from "./components/ClusterActions.vue";
import DatabaseEngineFilter from "./components/DatabaseEngineFilter.vue";
import ClusterDetail from "./components/ClusterDetail.vue";
import ClusterCreate from "./components/ClusterCreate.vue";
import TaskDetail from "./components/TaskDetail.vue";
import BackupSettings from "./components/BackupSettings.vue";
import StorageThresholds from "./components/StorageThresholds.vue";
const props = defineProps<{ section?: string; embedded?: boolean }>();
const repositoryId = ref<number>(),
  repositoryDetailOpen = ref(false);
const thresholdsOpen = ref(false);
const selectedBackup = ref<any>(),
  backupDetailOpen = ref(false);
const auditPolicyOpen = ref(false),
  retentionDays = ref(90);
async function openAuditPolicy() {
  try {
    retentionDays.value = (await dbRequest("audit-policy")).retentionDays;
    auditPolicyOpen.value = true;
  } catch (e) {
    error.value = (e as Error).message;
  }
}
async function saveAuditPolicy() {
  try {
    await dbRequest("audit-policy", "put", {
      retentionDays: retentionDays.value
    });
    auditPolicyOpen.value = false;
    ElMessage.success(tr("日志保存周期已更新"));
  } catch (e) {
    error.value = (e as Error).message;
  }
}
async function deleteAudit(row: any) {
  try {
    await ElMessageBox.confirm(
      tr("确认删除这条{value0}？", {
        value0:
          auditTab.value === "operations"
            ? tr("任务的步骤与日志")
            : tr("事件日志")
      }),
      tr("确认删除"),
      {
        get confirmButtonText() {
          return tr("确认删除");
        },
        get cancelButtonText() {
          return tr("取消");
        },
        type: "warning"
      }
    );
    await dbRequest(`audit/${auditTab.value}/${row.id}`, "delete", {
      confirmed: true
    });
    await load();
  } catch (e) {
    if (e !== "cancel" && e !== "close") error.value = (e as Error).message;
  }
}
const operationClusters = ref<Cluster[]>([]),
  operationClusterId = ref<number>();
async function loadOperationClusters(visible: boolean) {
  if (!visible) return;
  try {
    operationClusters.value = (
      await dbPlatform.clusters({
        size: 100,
        engine: filters.engine,
        groupId: filters.groupId
      })
    ).list;
  } catch (e) {
    error.value = (e as Error).message;
  }
}
const route = useRoute();
const router = useRouter();
function selectOperationCluster(id?: number) {
  const query = { ...route.query };
  if (id) query.clusterId = String(id);
  else delete query.clusterId;
  router.replace({ query });
}
function clearClusterScope() {
  const query = { ...route.query };
  delete query.clusterId;
  router.replace({ query });
}
const section = computed(() =>
  String(props.section || route.meta.databaseSection || "clusters")
);
const titles: Record<string, string> = {
  get clusters() {
    return tr("集群管理");
  },
  get storage() {
    return tr("存储管理");
  },
  get backups() {
    return tr("备份恢复");
  },
  get audit() {
    return tr("操作审计");
  }
};
const title = computed(() => titles[section.value]);
const descriptions: Record<string, string> = {
  get clusters() {
    return tr("统一查看实例、复制与访问入口，通过预检计划执行运维操作。");
  },
  get storage() {
    return tr("查看数据库、持久化日志与主机文件系统的真实容量。");
  },
  get backups() {
    return tr("管理备份策略与备份集，通过隔离恢复核验可恢复性。");
  },
  get audit() {
    return tr("追踪人工操作、原生拓扑变化、任务步骤和失败原因。");
  }
};
const filters = reactive({
  clusterId: Number(route.query.clusterId) || (undefined as number | undefined),
  name: "",
  location: "",
  groupId: undefined as number | undefined,
  engine: ["mysql", "postgresql", "redis", ""].includes(
    String(route.query.engine)
  )
    ? String(route.query.engine)
    : "mysql",
  environment: "",
  status: "",
  page: 1,
  size: 20
});
const clusters = ref<Cluster[]>([]),
  groups = ref<any[]>([]),
  records = ref<any[]>([]),
  total = ref(0),
  recordTotal = ref(0),
  loading = ref(false),
  error = ref("");
const detailOpen = ref(false),
  detailTab = ref("overview"),
  selected = ref(0),
  createOpen = ref(false),
  taskOpen = ref(false),
  taskId = ref(0),
  storageTab = ref("all"),
  recordTab = ref("backups"),
  auditTab = ref("operations");
const editingPolicy = ref<any>();
const settingsOpen = ref(false),
  settingsKind = ref<"repository" | "policy">("policy");
async function togglePolicy(row: any, enabled: boolean | string | number) {
  try {
    await dbRequest(`backup-policies/${row.id}`, "patch", { enabled });
    await load();
  } catch (e) {
    error.value = (e as Error).message;
  }
}
async function checkRepository(id: number) {
  try {
    await dbRequest(`repositories/${id}/check`, "post");
    await load();
  } catch (e) {
    error.value = (e as Error).message;
  }
}
const isBackupTab = computed(() =>
  ["backups", "verification"].includes(recordTab.value)
);
const filterStatuses = computed(() =>
  section.value === "backups"
    ? recordTab.value === "repositories"
      ? ["healthy", "unchecked", "unavailable"]
      : []
    : section.value === "audit"
      ? ["queued", "running", "verifying", "unknown", "succeeded", "failed"]
      : ["healthy", "degraded", "unavailable", "stale", "planned"]
);
async function deleteRepository(row: any) {
  try {
    await ElMessageBox.confirm(
      tr(
        "确认删除仓库「{value0}」的登记？仅移除平台配置，保留存储桶、已有文件和配置中心凭据。仍被备份、策略或任务使用的仓库需要先解除引用。",
        { value0: row.name }
      ),
      tr("删除备份仓库"),
      {
        get confirmButtonText() {
          return tr("确认删除");
        },
        get cancelButtonText() {
          return tr("取消");
        },
        type: "warning"
      }
    );
    await dbRequest(`repositories/${row.id}`, "delete", { confirmed: true });
    ElMessage.success(tr("仓库登记已删除"));
    await load();
  } catch (e) {
    if (e !== "cancel" && e !== "close") error.value = (e as Error).message;
  }
}
const storageRows = computed(() => records.value);
function groupName(id: number) {
  const find = (items: any[]): string => {
    for (const g of items) {
      if (g.id === id) return g.name;
      const child = find(g.children || []);
      if (child) return child;
    }
    return "";
  };
  return find(groups.value) || tr("分组 #{value0}", { value0: id });
}
function clusterName(id: number) {
  return (
    operationClusters.value.find(c => c.id === id)?.name ||
    clusters.value.find(c => c.id === id)?.name ||
    tr("集群 #{value0}", { value0: id })
  );
}
const date = (v?: string) =>
  v ? new Date(v).toLocaleString() : tr("尚未采集");
let loadGeneration = 0;
async function load() {
  const generation = ++loadGeneration;
  const query = {
    ...filters,
    status: filterStatuses.value.includes(filters.status) ? filters.status : ""
  };
  const currentSection = section.value;
  const currentTab = recordTab.value;
  loading.value = true;
  error.value = "";
  try {
    const data = await dbPlatform.clusters({
      ...query,
      name:
        currentSection === "backups" &&
        ["backups", "verification"].includes(currentTab)
          ? undefined
          : query.name,
      page: currentSection === "clusters" ? filters.page : 1
    });
    if (generation !== loadGeneration) return;
    clusters.value = data.list || [];
    if (filters.clusterId) {
      operationClusterId.value = filters.clusterId;
      const selectedCluster = clusters.value.find(
        c => c.id === filters.clusterId
      );
      if (
        selectedCluster &&
        !operationClusters.value.some(c => c.id === selectedCluster.id)
      )
        operationClusters.value.push(selectedCluster);
    }
    total.value = data.total;
    const resource =
      currentSection === "storage"
        ? "storage"
        : currentSection === "backups"
          ? currentTab === "verification"
            ? "backups"
            : currentTab
          : currentSection === "audit"
            ? auditTab.value
            : "";
    if (resource) {
      const result = await dbRequest(resource, "get", undefined, {
        ...query,
        kind: storageTab.value,
        view:
          resource === "backups"
            ? currentTab === "verification"
              ? "verification"
              : "verified"
            : undefined,
        engine: resource === "repositories" ? undefined : query.engine
      });
      if (generation !== loadGeneration) return;
      records.value = result.list || [];
      if (resource === "backups" && selectedBackup.value) {
        const fresh = records.value.find(b => b.id === selectedBackup.value.id);
        if (fresh) selectedBackup.value = fresh;
      }
      recordTotal.value = result.total ?? records.value.length;
    }
  } catch (e) {
    if (generation === loadGeneration) error.value = (e as Error).message;
  } finally {
    if (generation === loadGeneration) loading.value = false;
  }
}
function search() {
  filters.page = 1;
  load();
}
function openCluster(c: Cluster, tab: unknown = "overview") {
  detailTab.value = typeof tab === "string" ? tab : "overview";
  selected.value = c.id;
  detailOpen.value = true;
}
function openById(id: number) {
  detailTab.value = "overview";
  selected.value = id;
  detailOpen.value = true;
}
function showTask(id: number) {
  taskId.value = id;
  taskOpen.value = true;
}
function created(id: number) {
  load();
  openById(id);
}
function changeEngine() {
  operationClusterId.value = undefined;
  operationClusters.value = [];
  const query: Record<string, any> = { ...route.query, engine: filters.engine };
  delete query.clusterId;
  delete query.detail;
  router.replace({ query });
}
let scopeGeneration = 0;
watch(
  recordTab,
  (tab, previous) => {
    filters.status = "";
    filters.location = "";
    if (
      !["backups", "verification"].includes(tab) ||
      !["backups", "verification"].includes(previous)
    )
      filters.name = "";
  },
  { flush: "sync" }
);
watch(
  () => [
    section.value,
    route.query.clusterId,
    route.query.engine,
    route.query.detail
  ],
  async () => {
    const generation = ++scopeGeneration;
    ++loadGeneration;
    filters.clusterId = Number(route.query.clusterId) || undefined;
    filters.engine = ["mysql", "postgresql", "redis", ""].includes(
      String(route.query.engine)
    )
      ? String(route.query.engine)
      : "mysql";
    filters.page = 1;
    records.value = [];
    operationClusterId.value = filters.clusterId;
    operationClusters.value = [];
    if (filters.clusterId && route.query.engine == null) {
      try {
        const { cluster } = await dbPlatform.cluster(filters.clusterId);
        if (generation !== scopeGeneration) return;
        filters.engine = cluster.engine;
      } catch (e) {
        error.value = (e as Error).message;
      }
    }
    if (generation !== scopeGeneration) return;
    if (route.query.detail === "parameters" && filters.clusterId) {
      selected.value = filters.clusterId;
      detailTab.value = "parameters";
      detailOpen.value = true;
    }
    await load();
  },
  { immediate: true }
);
let refreshTimer: ReturnType<typeof setInterval>;
onUnmounted(() => clearInterval(refreshTimer));
onMounted(async () => {
  refreshTimer = setInterval(() => {
    if (
      !loading.value &&
      !document.hidden &&
      !detailOpen.value &&
      !taskOpen.value &&
      !auditPolicyOpen.value
    )
      load();
  }, 15000);
  try {
    groups.value = await dbRequest("groups");
  } catch {}
});
</script>
<style scoped>
.embedded-management .db-hero {
  justify-content: flex-end;
  margin-bottom: 16px;
}
.cluster-identity {
  display: flex;
  align-items: center;
  gap: 10px;
  height: 44px;
  min-width: 0;
}
.row-actions {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  white-space: nowrap;
}
.db-management :deep(.database-engine-icon) {
  width: 26px;
  height: 26px;
  flex: 0 0 26px;
}
.db-management :deep(.el-table__row > td) {
  height: 64px;
  padding: 7px 0;
  box-sizing: border-box;
}
.db-management :deep(.el-table__row > td .cell) {
  max-height: 48px;
  overflow: hidden;
}
.db-management :deep(.el-table__row > td:not(.el-table__expand-column) .cell) {
  line-height: 20px;
}
.db-muted {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.el-pagination {
  flex-wrap: wrap;
  gap: 8px;
}

.cluster-identity,
.row-actions,
.cluster-identity > div {
  flex: 1;
  min-width: 0;
}
.cluster-identity .el-button {
  max-width: 100%;
}
.cluster-identity .el-button :deep(span) {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
}
.module-toolbar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
}
.module-toolbar {
  margin: 16px 0;
}
.module-toolbar .el-select {
  width: 280px;
}
.cluster-scope {
  margin-bottom: 16px;
}
.db-management {
  padding: 20px;
  color: var(--el-text-color-primary);
}
.db-hero {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}
.hero-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  flex-wrap: wrap;
}
.db-hero small {
  color: var(--el-color-primary);
  letter-spacing: 2px;
  font-size: 11px;
}
.db-hero h1 {
  font-size: 26px;
  margin: 8px 0;
}
.db-hero p,
.db-note {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
.db-filters {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
  margin-bottom: 20px;
}
.db-filters .el-input,
.db-filters .el-select {
  width: 170px;
}
.db-muted {
  display: block;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
}
.el-pagination {
  margin-top: 20px;
  justify-content: flex-end;
}
.el-alert {
  margin-bottom: 14px;
}
@media (max-width: 900px) {
  .db-hero {
    align-items: flex-start;
    flex-wrap: wrap;
    gap: 16px;
  }
}
</style>
