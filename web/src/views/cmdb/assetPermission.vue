<template>
  <div class="asset-permission-management asset-page">
    <header v-if="!embedded" class="asset-page-header">
      <div>
        <span class="asset-eyebrow">ACCESS CONTROL</span>
        <h1>{{ $tr("资产授权") }}</h1>
        <p>{{ $tr("按资产范围、访问协议、文件操作与配置变更分配权限，并设置有效期。") }}</p>
      </div>
      <div class="asset-header-count">
        <strong>{{ total }}</strong
        ><span>{{ $tr("授权规则") }}</span>
      </div>
    </header>
    <el-card shadow="never" class="main-card asset-surface">
      <!-- Search -->
      <div class="search-section asset-filters">
        <el-form :inline="true" :model="query" size="small">
          <el-form-item :label='$tr("规则名称")'>
            <el-input
              v-model="query.name"
              :placeholder='$tr("规则名称")'
              clearable
              style="width: 180px"
              @keyup.enter="search"
            />
          </el-form-item>
          <el-form-item :label='$tr("授权对象")'>
            <el-select
              v-model="query.subjectType"
              :placeholder='$tr("全部")'
              clearable
              style="width: 120px"
              @change="search"
            >
              <el-option :label='$tr("用户")' value="user" />
              <el-option :label='$tr("用户组")' value="group" />
            </el-select>
          </el-form-item>
          <el-form-item v-if="query.subjectType" :label='$tr("授权主体")'>
            <el-select
              v-model="query.subjectId"
              filterable
              clearable
              :placeholder='$tr("选择用户/用户组")'
              style="width: 200px"
              @change="search"
              v-if="query.subjectType === 'user'"
            >
              <el-option
                v-for="u in userList"
                :key="u.id"
                :label="`${u.username || ''} (${u.nickname || ''})`"
                :value="u.id"
              />
            </el-select>
            <el-select
              v-model="query.subjectId"
              filterable
              clearable
              :placeholder='$tr("选择用户组")'
              style="width: 200px"
              @change="search"
              v-if="query.subjectType === 'group'"
            >
              <el-option
                v-for="g in userGroupList"
                :key="g.id"
                :label="$tr(g.name)"
                :value="g.id"
              />
            </el-select>
          </el-form-item>
          <el-form-item :label='$tr("状态")'>
            <el-select
              v-model="query.isActive"
              :placeholder='$tr("全部")'
              clearable
              style="width: 120px"
            >
              <el-option :label='$tr("启用")' :value="1" />
              <el-option :label='$tr("禁用")' :value="0" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" size="small" @click="search"
              >{{ $tr("查询") }}</el-button
            >
            <el-button size="small" @click="reset">{{ $tr("重置") }}</el-button>
          </el-form-item>
        </el-form>
      </div>

      <div class="action-section asset-toolbar">
        <el-button type="primary" size="small" @click="showCreate"
          >{{ $tr("+ 新增授权规则") }}</el-button
        >
      </div>

      <el-table
        :data="list"
        v-loading="loading"
        class="asset-table"
        row-key="id"
        style="width: 100%"
      >
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="name" :label='$tr("授权名称")' min-width="160" />
        <el-table-column :label='$tr("授权主体")' min-width="220">
          <template #default="{ row }">
            <div>
              <el-tag
                size="small"
                type="primary"
                v-if="row.userIds && row.userIds != '[]'"
              >
                {{ $tr("用户:") }} {{ resolveUserNames(row.userIds) }}
              </el-tag>
            </div>
            <div style="margin-top: 2px">
              <el-tag
                size="small"
                type="success"
                v-if="row.groupIds && row.groupIds != '[]'"
              >
                {{ $tr("用户组:") }} {{ resolveGroupNames(row.groupIds) }}
              </el-tag>
            </div>
            <span
              v-if="
                (!row.userIds || row.userIds == '[]') &&
                (!row.groupIds || row.groupIds == '[]')
              "
              style="color: #999"
              >-</span
            >
          </template>
        </el-table-column>
        <el-table-column :label='$tr("资产类型")' min-width="140">
          <template #default="{ row }">
            <el-tag
              size="small"
              v-for="t in safeJSON(row.assetTypes, [])"
              :key="t"
              style="margin: 2px"
              >{{ $tr(assetTypeLabel(t)) }}</el-tag
            >
            <span
              v-if="!row.assetTypes || row.assetTypes == '[]'"
              style="color: #999"
              >-</span
            >
          </template>
        </el-table-column>
        <el-table-column :label='$tr("权限操作")' min-width="200">
          <template #default="{ row }">
            <el-tag
              size="small"
              v-for="a in safeJSON(row.permissionActions, [])"
              :key="a"
              :type="actionTag(a)"
              style="margin: 2px"
              >{{ actionText(a) }}</el-tag
            >
          </template>
        </el-table-column>
        <el-table-column :label='$tr("状态")' width="80" align="center">
          <template #default="{ row }">
            <el-tag
              :type="row.isActive === 1 ? 'success' : 'info'"
              size="small"
              >{{ row.isActive === 1 ? $tr("启用") : $tr("禁用") }}</el-tag
            >
          </template>
        </el-table-column>
        <el-table-column :label='$tr("有效期")' width="200">
          <template #default="{ row }">
            <span v-if="row.dateStart || row.dateExpired"
              >{{ row.dateStart || "永久" }} ~
              {{ row.dateExpired || "永久" }}</span
            >
            <span v-else style="color: #999">{{ $tr("永久") }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="createdAt" :label='$tr("创建时间")' width="170" />
        <el-table-column :label='$tr("操作")' width="160" align="center" fixed="right">
          <template #default="{ row }">
            <el-button type="warning" size="small" @click="showEdit(row)"
              >{{ $tr("编辑") }}</el-button
            >
            <el-button type="danger" size="small" @click="handleDelete(row)"
              >{{ $tr("删除") }}</el-button
            >
          </template>
        </el-table-column>
      </el-table>

      <div class="pagination-section asset-pagination">
        <el-pagination
          @size-change="
            s => {
              query.size = s;
              fetch();
            }
          "
          @current-change="
            p => {
              query.page = p;
              fetch();
            }
          "
          :current-page="query.page"
          :page-sizes="[10, 20, 50, 100]"
          :page-size="query.size"
          layout="total, sizes, prev, pager, next, jumper"
          :total="total"
        />
      </div>
    </el-card>

    <!-- Create/Edit Dialog -->
    <el-dialog
      :title="$tr(formTitle)"
      v-model="formVisible"
      width="900px"
      @close="resetForm"
      top="3vh"
    >
      <el-form :model="form" :rules="rules" ref="formRef" label-width="100px">
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item :label='$tr("规则名称")' prop="name">
              <el-input
                v-model="form.name"
                :placeholder='$tr("如: 基础运维-主机管理权限")'
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label='$tr("启用状态")'>
              <el-switch
                v-model="form.isActive"
                :active-value="1"
                :inactive-value="0"
              />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item :label='$tr("描述")'>
          <el-input
            v-model="form.description"
            type="textarea"
            :rows="2"
            :placeholder='$tr("规则描述")'
          />
        </el-form-item>
        <el-row :gutter="20">
          <el-col :span="12">
            <el-form-item :label='$tr("授权用户")'>
              <el-select
                v-model="form.userIds"
                multiple
                filterable
                :placeholder='$tr("选择用户")'
                style="width: 100%"
              >
                <el-option
                  v-for="u in userList"
                  :key="u.id"
                  :label="`${u.username || ''} (${u.nickname || ''})`"
                  :value="u.id"
                />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label='$tr("授权用户组")'>
              <el-select
                v-model="form.groupIds"
                multiple
                filterable
                :placeholder='$tr("选择用户组")'
                style="width: 100%"
              >
                <el-option
                  v-for="g in userGroupList"
                  :key="g.id"
                  :label="$tr(g.name)"
                  :value="g.id"
                />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>

        <el-form-item :label='$tr("资产类型")'>
          <el-checkbox-group v-model="form.assetTypes">
            <el-checkbox label="host">{{ $tr("主机") }}</el-checkbox>
            <el-checkbox label="physical">{{ $tr("物理机") }}</el-checkbox>
            <el-checkbox label="network">{{ $tr("网络设备") }}</el-checkbox>
            <el-checkbox label="database">{{ $tr("数据源") }}</el-checkbox>
 <el-checkbox label="database_cluster">数据库集群</el-checkbox>
          </el-checkbox-group>
        </el-form-item>

        <el-form-item :label='$tr("资产范围")'>
          <el-tabs type="border-card" size="small" style="width: 100%">
 <el-tab-pane v-if="form.assetTypes.includes('database_cluster')" label="数据库集群授权">
  <el-form-item label="集群"><el-select v-model="form.databaseClusterIds" multiple filterable style="width:100%"><el-option v-for="c in databaseClusters" :key="c.id" :value="c.id" :label="c.name + ' · ' + c.engine"/></el-select></el-form-item>
  <el-form-item label="业务分组（含子分组）"><el-tree-select v-model="form.hostGroupIds" :data="groupList" :props="{label:'name',value:'id',children:'children'}" multiple check-strictly style="width:100%"/></el-form-item>
 </el-tab-pane>
            <el-tab-pane
              :label='$tr("主机授权")'
              v-if="form.assetTypes.includes('host')"
            >
              <div style="margin-bottom: 10px">
                <el-button size="mini" type="primary" @click="showHostSelector"
                  >{{ $tr("选择主机") }}</el-button
                >
                <el-tag size="mini" type="info" style="margin-left: 8px"
                  >{{ $tr("已选") }} {{ form.hostIds.length }} {{ $tr("台主机") }}</el-tag
                >
                <el-tag size="mini" type="warning" style="margin-left: 4px"
                  >{{ $tr("已选") }} {{ form.hostGroupIds.length }} {{ $tr("个分组") }}</el-tag
                >
              </div>
              <el-select
                v-model="form.hostGroupIds"
                multiple
                filterable
                :placeholder='$tr("选择主机分组(业务线)")'
                style="width: 100%"
              >
                <el-option
                  v-for="g in groupList"
                  :key="g.id"
                  :label="$tr(g.name)"
                  :value="g.id"
                />
              </el-select>
              <div
                style="margin-top: 8px; max-height: 150px; overflow-y: auto"
                v-if="form.hostIds.length"
              >
                <el-tag
                  closable
                  size="mini"
                  v-for="id in form.hostIds"
                  :key="id"
                  style="margin: 2px"
                  @close="removeHost(id)"
                >
                  {{ hostMap[id] || "主机#" + id }}
                </el-tag>
              </div>
            </el-tab-pane>
            <el-tab-pane
              :label='$tr("物理机授权")'
              v-if="form.assetTypes.includes('physical')"
            >
              <div style="margin-bottom: 10px">
                <el-button
                  size="mini"
                  type="primary"
                  @click="showPhysicalSelector"
                  >{{ $tr("选择物理机") }}</el-button
                >
                <el-tag size="mini" type="info" style="margin-left: 8px"
                  >{{ $tr("已选") }} {{ form.physicalIds.length }} {{ $tr("台") }}</el-tag
                >
              </div>
              <el-row :gutter="10">
                <el-col :span="8">
                  <el-select
                    v-model="physicalFilter.idcId"
                    filterable
                    clearable
                    :placeholder='$tr("选择机房")'
                    style="width: 100%"
                    size="mini"
                    @change="loadPhysicalList"
                  >
                    <el-option
                      v-for="d in idcList"
                      :key="d.id"
                      :label="$tr(d.name)"
                      :value="d.id"
                    />
                  </el-select>
                </el-col>
                <el-col :span="8">
                  <el-select
                    v-model="physicalFilter.cabinetId"
                    filterable
                    clearable
                    :placeholder='$tr("选择机柜")'
                    style="width: 100%"
                    size="mini"
                    :disabled="!physicalFilter.idcId"
                    @change="loadPhysicalList"
                  >
                    <el-option
                      v-for="c in cabinetList"
                      :key="c.id"
                      :label="$tr(c.name)"
                      :value="c.id"
                    />
                  </el-select>
                </el-col>
                <el-col :span="8">
                  <el-input
                    v-model="physicalFilter.keyword"
                    :placeholder='$tr("搜索名称/SN")'
                    size="mini"
                    clearable
                    @input="loadPhysicalList"
                  />
                </el-col>
              </el-row>
              <el-table
                :data="physicalList"
                border
                stripe
                size="mini"
                style="width: 100%; margin-top: 8px"
                max-height="200"
                @selection-change="onPhysicalSelect"
              >
                <el-table-column type="selection" width="40" />
                <el-table-column prop="sn" label="SN" width="120" />
                <el-table-column prop="hostName" :label='$tr("名称")' width="120" />
                <el-table-column prop="brand" :label='$tr("品牌")' width="80" />
                <el-table-column prop="manageIp" :label='$tr("管理IP")' width="120" />
              </el-table>
              <div style="margin-top: 4px">
                <el-tag
                  closable
                  size="mini"
                  v-for="id in form.physicalIds"
                  :key="id"
                  style="margin: 2px"
                  @close="removePhysical(id)"
                >
                  {{ physicalMap[id] || "物理机#" + id }}
                </el-tag>
              </div>
            </el-tab-pane>
            <el-tab-pane
              :label='$tr("网络设备授权")'
              v-if="form.assetTypes.includes('network')"
            >
              <div style="margin-bottom: 10px">
                <el-button
                  size="mini"
                  type="primary"
                  @click="showNetworkSelector"
                  >{{ $tr("选择网络设备") }}</el-button
                >
                <el-tag size="mini" type="info" style="margin-left: 8px"
                  >{{ $tr("已选") }} {{ form.networkIds.length }} {{ $tr("台") }}</el-tag
                >
              </div>
              <el-row :gutter="10">
                <el-col :span="6">
                  <el-select
                    v-model="networkFilter.idcId"
                    filterable
                    clearable
                    :placeholder='$tr("选择机房")'
                    style="width: 100%"
                    size="mini"
                    @change="loadNetworkList"
                  >
                    <el-option
                      v-for="d in idcList"
                      :key="d.id"
                      :label="$tr(d.name)"
                      :value="d.id"
                    />
                  </el-select>
                </el-col>
                <el-col :span="6">
                  <el-select
                    v-model="networkFilter.deviceType"
                    clearable
                    :placeholder='$tr("设备类型")'
                    style="width: 100%"
                    size="mini"
                    @change="loadNetworkList"
                  >
                    <el-option :label='$tr("路由器")' :value="1" />
                    <el-option :label='$tr("交换机")' :value="2" />
                    <el-option :label='$tr("防火墙")' :value="3" />
                    <el-option :label='$tr("负载均衡")' :value="4" />
                    <el-option :label='$tr("其他")' :value="5" />
                  </el-select>
                </el-col>
                <el-col :span="6">
                  <el-input
                    v-model="networkFilter.keyword"
                    :placeholder='$tr("搜索名称")'
                    size="mini"
                    clearable
                    @input="loadNetworkList"
                  />
                </el-col>
              </el-row>
              <el-table
                :data="networkList"
                border
                stripe
                size="mini"
                style="width: 100%; margin-top: 8px"
                max-height="200"
                @selection-change="onNetworkSelect"
              >
                <el-table-column type="selection" width="40" />
                <el-table-column prop="name" :label='$tr("名称")' width="120" />
                <el-table-column :label='$tr("类型")' width="80">
                  <template #default="{ row }">{{
                    $tr(deviceTypeLabel(row.deviceType))
                  }}</template>
                </el-table-column>
                <el-table-column prop="manageIp" :label='$tr("管理IP")' width="120" />
                <el-table-column prop="brand" :label='$tr("品牌")' width="80" />
              </el-table>
              <div style="margin-top: 4px">
                <el-tag
                  closable
                  size="mini"
                  v-for="id in form.networkIds"
                  :key="id"
                  style="margin: 2px"
                  @close="removeNetwork(id)"
                >
                  {{ networkMap[id] || "设备#" + id }}
                </el-tag>
              </div>
            </el-tab-pane>
            <el-tab-pane :label='$tr("机房授权")'>
              <el-select
                v-model="form.idcIds"
                multiple
                filterable
                :placeholder='$tr("选择机房（含其下所有资产）")'
                style="width: 100%"
              >
                <el-option
                  v-for="d in idcList"
                  :key="d.id"
                  :label="$tr(d.name)"
                  :value="d.id"
                />
              </el-select>
            </el-tab-pane>
          </el-tabs>
        </el-form-item>

        <el-form-item :label='$tr("权限操作")' prop="permissionActions">
          <div class="permission-action-groups">
            <section
              v-for="group in actionGroups"
              :key="group.title"
              class="permission-action-group"
            >
              <h4>{{ $tr(group.title) }}</h4>
              <p>{{ $tr(group.description) }}</p>
              <el-checkbox-group v-model="form.permissionActions"
                ><el-checkbox
                  v-for="action in group.actions"
                  :key="action"
                  :value="action"
                  >{{ actionText(action) }}</el-checkbox
                ></el-checkbox-group
              >
            </section>
            <el-alert
              v-if="form.permissionActions.includes('admin')"
              :title='$tr("管理权限包含该规则范围内的所有操作，请确认资产范围和有效期。")'
              type="warning"
              :closable="false"
              show-icon
            />
          </div>
        </el-form-item>

        <el-row :gutter="20">
          <el-col :span="8">
            <el-form-item :label='$tr("生效日期")'
              ><el-input v-model="form.dateStart" placeholder="YYYY-MM-DD"
            /></el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item :label='$tr("失效日期")'
              ><el-input v-model="form.dateExpired" placeholder="YYYY-MM-DD"
            /></el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="formVisible = false">{{ $tr("取消") }}</el-button>
        <el-button type="primary" @click="handleSubmit" :loading="submitting"
          >{{ $tr("确定") }}</el-button
        >
      </template>
    </el-dialog>

    <!-- Host Selector Dialog -->
    <el-dialog
      :title='$tr("选择主机")'
      v-model="hostSelectorVisible"
      width="min(920px, 96vw)"
      top="5vh"
    >
      <el-row :gutter="10" style="margin-bottom: 10px">
        <el-col :span="8">
          <el-select
            v-model="hostFilter.groupId"
            filterable
            clearable
            :placeholder='$tr("选择业务线/分组")'
            style="width: 100%"
            size="small"
            @change="loadHostList"
          >
            <el-option
              v-for="g in flatGroupList"
              :key="g.id"
              :label="$tr(g.name)"
              :value="g.id"
            />
          </el-select>
        </el-col>
        <el-col :span="8">
          <el-input
            v-model="hostFilter.keyword"
            :placeholder='$tr("搜索主机名/IP")'
            size="small"
            clearable
            @input="loadHostList"
          />
        </el-col>
        <el-col :span="4">
          <el-select
            v-model="hostFilter.status"
            clearable
            :placeholder='$tr("状态")'
            size="small"
            style="width: 100%"
            @change="loadHostList"
          >
            <el-option :label='$tr("认证成功")' :value="1" />
            <el-option :label='$tr("未认证")' :value="2" />
            <el-option :label='$tr("认证失败")' :value="3" />
          </el-select>
        </el-col>
        <el-col :span="4">
          <el-button size="small" @click="selectAllHosts">{{ $tr("全选本页") }}</el-button>
        </el-col>
      </el-row>
      <el-table
        :data="hostList"
        border
        stripe
        size="mini"
        style="width: 100%"
        max-height="350"
        ref="hostTable"
        @selection-change="onHostSelect"
      >
        <el-table-column type="selection" width="40" />
        <el-table-column prop="hostName" :label='$tr("主机名")' width="140" />
        <el-table-column prop="sshIp" label="SSH-IP" width="130" />
        <el-table-column prop="privateIp" :label='$tr("私网IP")' width="130" />
        <el-table-column :label='$tr("状态")' width="80">
          <template #default="{ row }">{{
            { 1: $tr("正常"), 2: $tr("未认证"), 3: $tr("失败") }[row.status] || "未知"
          }}</template>
        </el-table-column>
        <el-table-column :label='$tr("分组")' min-width="120">
          <template #default="{ row }">{{ row.group?.name || "-" }}</template>
        </el-table-column>
      </el-table>
      <div style="text-align: center; margin-top: 10px">
        <el-pagination
          small
          @current-change="
            p => {
              hostPage = p;
              loadHostList();
            }
          "
          :current-page="hostPage"
          :page-size="10"
          layout="prev, pager, next"
          :total="hostTotal"
        />
      </div>
      <template #footer>
        <el-button @click="hostSelectorVisible = false">{{ $tr("关闭") }}</el-button>
        <el-button type="primary" @click="confirmHostSelect"
          >{{ $tr("确定选择 (") }}{{ selectedHostIds.length }})</el-button
        >
      </template>
    </el-dialog>
  </div>
</template>

<script>
import { tr } from "@/locales";

import cmdbAPI from "@/api/cmdb";
import systemApi from "@/api/system";
import {dbPlatform} from "@/api/databasePlatform";

const DEVICE_TYPE_MAP = {
  1: "路由器",
  2: "交换机",
  3: "防火墙",
  4: "负载均衡",
  5: "其他"
};
const ASSET_TYPE_MAP = {
  host: "主机",
  physical: "物理机",
  network: "网络设备",
  database: "数据源",
  database_cluster: "数据库集群"
};
const ACTION_MAP = {
  get: "查看详情",
  list: "查看列表",
  connect: "SSH 连接（旧规则）",
  connect_ssh: "SSH 终端",
  connect_vnc: "VNC 桌面",
  connect_rdp: "RDP 桌面",
  file_list: "浏览文件",
  file_upload: "上传文件",
  file_download: "下载文件",
  file_delete: "删除文件",
  command: "执行命令",
  audit: "操作审计",
  config_view: "查看配置",
  config_backup: "备份配置",
  config_restore: "恢复配置",
  db_query: "数据库查询",
  db_execute: "数据库写操作",
  db_cluster_view: "查看集群",
  db_cluster_manage: "维护集群",
  db_cluster_switch: "高可用切换",
  db_cluster_backup: "集群备份",
  db_cluster_restore: "隔离恢复",
  db_cluster_config: "修改集群参数",
  db_cluster_audit: "查看运维审计",

  docker_view: "查看 Docker",
  docker_logs: "查看容器日志",
  docker_operate: "操作容器",
  docker_remove: "删除容器",
  create: "创建资产",
  update: "修改资产与连接",
  delete: "删除资产",
  admin: "全部管理权限"
};
const ACTION_GROUPS = [
 {title:"数据库集群",description:"数据源查询权限不包含集群维护；按集群与动作分别授权。",actions:["db_cluster_view","db_cluster_manage","db_cluster_switch","db_cluster_backup","db_cluster_restore","db_cluster_config","db_cluster_audit"]},
  {
    title: "资产信息",
    description: "决定可以查看和维护哪些资产信息。",
    actions: ["get", "list", "create", "update", "delete"]
  },
  {
    title: "远程访问",
    description: "不同协议分别授权；终端与桌面访问均可操作远端系统。",
    actions: ["connect_ssh", "connect_vnc", "connect_rdp", "command", "audit"]
  },
  {
    title: "文件管理",
    description: "文件的查看、上传、下载与删除分别控制。",
    actions: ["file_list", "file_upload", "file_download", "file_delete"]
  },
  {
    title: "网络配置",
    description: "恢复会变更设备持久配置，应单独授权。",
    actions: ["config_view", "config_backup", "config_restore"]
  },
  {
    title: "数据管理",
    description: "查询和写操作分开授权；写操作还需要名称确认。",
    actions: ["db_query", "db_execute"]
  },
  {
    title: "Docker 管理",
    description: "容器查看、日志、生命周期操作和删除分别授权。",
    actions: ["docker_view", "docker_logs", "docker_operate", "docker_remove"]
  },
  {
    title: "完整管理",
    description: "包括该规则所选资产的全部操作。",
    actions: ["admin"]
  }
];
const ACTION_TAG = {
  get: "",
  list: "info",
  connect: "primary",
  create: "success",
  update: "warning",
  delete: "danger",
  admin: "danger"
};

export default {
  name: "AssetPermission",
  props: { embedded: Boolean },
  data() {
    return {
      actionGroups: ACTION_GROUPS,
      loading: false,
      submitting: false,
      list: [],
      total: 0,
      userList: [],
      userGroupList: [],
      groupList: [],
 databaseClusters: [],
      idcList: [],
      cabinetList: [],
      flatGroupList: [],
      query: {
        name: "",
        subjectType: undefined,
        subjectId: undefined,
        isActive: undefined,
        page: 1,
        size: 10
      },
      formVisible: false,
      formTitle: "新增授权规则",
      isEdit: false,
      editId: null,
      hostSelectorVisible: false,
      hostList: [],
      hostTotal: 0,
      hostPage: 1,
      selectedHostIds: [],
      hostFilter: { groupId: undefined, keyword: "", status: undefined },
      hostMap: {},
      physicalFilter: { idcId: undefined, cabinetId: undefined, keyword: "" },
      physicalList: [],
      physicalMap: {},
      networkFilter: { idcId: undefined, deviceType: undefined, keyword: "" },
      networkList: [],
      networkMap: {},
      form: {
        name: "",
        description: "",
        userIds: [],
        groupIds: [],
        assetTypes: ["host"],
        hostGroupIds: [],
        hostIds: [],
        physicalIds: [],
        networkIds: [],
        databaseIds: [],
 databaseClusterIds: [],
        idcIds: [],
        permissionActions: ["get", "list"],
        isActive: 1,
        dateStart: "",
        dateExpired: ""
      },
      rules: {
        name: [{ required: true, get message() { return tr("请输入规则名称"); }, trigger: "blur" }],
        permissionActions: [
          {
            type: "array",
            min: 1,
            required: true,
            get message() { return tr("至少选择一项权限操作"); },
            trigger: "change"
          }
        ]
      }
    };
  },
  created() {
    this.fetch();
    this.loadOptions();
  },
  methods: {
    async fetch() {
      this.loading = true;
      try {
        const res = await cmdbAPI.getAssetPermissionList(this.query);
        if (res.data.code === 200) {
          this.list = res.data.data.list || [];
          this.total = res.data.data.total || 0;
        }
      } catch (e) {
        this.$message.error(tr("获取授权列表失败"));
      } finally {
        this.loading = false;
      }
    },
    safeJSON(str, def) {
      if (Array.isArray(str)) return str;
      try {
        return JSON.parse(str) || def;
      } catch (e) {
        return def;
      }
    },
    assetTypeLabel(t) {
      return ASSET_TYPE_MAP[t] || t;
    },
    actionText(a) {
      return ACTION_MAP[a] || a;
    },
    actionTag(a) {
      return ACTION_TAG[a] || "info";
    },
    deviceTypeLabel(t) {
      return DEVICE_TYPE_MAP[t] || "未知";
    },
    async loadOptions() {
 try { const clusters = await dbPlatform.clusters({size:100}); this.databaseClusters = clusters.list || []; } catch (_) {this.databaseClusters=[];}
      try {
        const [res1, res2, res3, res4, res5] = await Promise.all([
          systemApi.queryAdminList({ page: 1, size: 1000 }),
          cmdbAPI.getAllCmdbGroups(),
          cmdbAPI.getIdcAll(),
          cmdbAPI.getCmdbUserGroupAll(),
          cmdbAPI.getHostList({ page: 1, pageSize: 1000 })
        ]);
        if (res1.data.code === 200) {
          this.userList = res1.data.data.list || [];
          console.log("用户列表", this.userList);
        }
        if (res2.data && res2.data.data) {
          const groups = Array.isArray(res2.data.data)
            ? res2.data.data
            : res2.data.data.list || [];
          this.groupList = groups;
          this.flatGroupList = this.flattenGroups(groups);
        }
        if (res3.data.code === 200) this.idcList = res3.data.data.list || [];
        if (res4.data && res4.data.data)
          this.userGroupList = res4.data.data.list || [];
        if (res5.data && res5.data.data) {
          const hosts = res5.data.data.list || res5.data.data || [];
          hosts.forEach(h => {
            this.hostMap[h.id] = h.hostName || h.name;
          });
        }
      } catch (_) {}
    },
    flattenGroups(groups, prefix = "") {
      let result = [];
      for (const g of groups) {
        result.push({ ...g, name: prefix + g.name });
        if (g.children)
          result = result.concat(
            this.flattenGroups(g.children, prefix + g.name + "/")
          );
      }
      return result;
    },
    async loadHostList() {
      try {
        const params = { page: this.hostPage, pageSize: 10 };
        if (this.hostFilter.groupId) params.groupId = this.hostFilter.groupId;
        if (this.hostFilter.keyword) params.name = this.hostFilter.keyword;
        if (this.hostFilter.status) params.status = this.hostFilter.status;
        const res = await cmdbAPI.getCmdbHostList(params);
        if (res.data.code === 200) {
          this.hostList = res.data.data.list || [];
          this.hostTotal = res.data.data.total || 0;
        }
      } catch (_) {
        this.hostList = [];
        this.hostTotal = 0;
      }
    },
    async loadPhysicalList() {
      try {
        const params = { page: 1, size: 200 };
        if (this.physicalFilter.idcId) params.idcId = this.physicalFilter.idcId;
        if (this.physicalFilter.cabinetId)
          params.cabinetId = this.physicalFilter.cabinetId;
        if (this.physicalFilter.keyword)
          params.keyword = this.physicalFilter.keyword;
        const res = await cmdbAPI.getPhysicalMachineList(params);
        if (res.data.code === 200) {
          this.physicalList = res.data.data.list || [];
          this.physicalList.forEach(p => {
            this.physicalMap[p.id] = p.hostName || p.sn;
          });
        }
      } catch (_) {
        this.physicalList = [];
      }
    },
    async loadNetworkList() {
      try {
        const params = { page: 1, size: 200 };
        if (this.networkFilter.idcId) params.idcId = this.networkFilter.idcId;
        if (this.networkFilter.deviceType)
          params.deviceType = this.networkFilter.deviceType;
        if (this.networkFilter.keyword)
          params.keyword = this.networkFilter.keyword;
        const res = await cmdbAPI.getNetworkDeviceList(params);
        if (res.data.code === 200) {
          this.networkList = res.data.data.list || [];
          this.networkList.forEach(n => {
            this.networkMap[n.id] = n.name;
          });
        }
      } catch (_) {
        this.networkList = [];
      }
    },
    search() {
      this.query.page = 1;
      this.fetch();
    },
    reset() {
      this.query = {
        name: "",
        subjectType: undefined,
        subjectId: undefined,
        isActive: undefined,
        page: 1,
        size: 10
      };
      this.fetch();
    },
    showCreate() {
      this.isEdit = false;
      this.editId = null;
      this.formTitle = "新增授权规则";
      this.form = {
        name: "",
        description: "",
        userIds: [],
        groupIds: [],
        assetTypes: ["host"],
        hostGroupIds: [],
        hostIds: [],
        physicalIds: [],
        networkIds: [],
        databaseIds: [],
 databaseClusterIds: [],
        idcIds: [],
        permissionActions: ["get", "list"],
        isActive: 1,
        dateStart: "",
        dateExpired: ""
      };
      this.formVisible = true;
    },
    showEdit(row) {
      this.isEdit = true;
      this.editId = row.id;
      this.formTitle = "编辑授权规则";
      this.form = {
        name: row.name,
        description: row.description || "",
        userIds: this.safeJSON(row.userIds, []),
        groupIds: this.safeJSON(row.groupIds, []),
        assetTypes: this.safeJSON(row.assetTypes, ["host"]),
        hostGroupIds: this.safeJSON(row.hostGroupIds, []),
        hostIds: this.safeJSON(row.hostIds, []),
        physicalIds: this.safeJSON(row.physicalIds, []),
        networkIds: this.safeJSON(row.networkIds, []),
        databaseIds: this.safeJSON(row.databaseIds, []),
 databaseClusterIds: this.safeJSON(row.databaseClusterIds, []),
        idcIds: this.safeJSON(row.idcIds, []),
        permissionActions: this.safeJSON(row.permissionActions, [
          "get",
          "list"
        ]).map(a => (a === "connect" ? "connect_ssh" : a)),
        isActive: row.isActive,
        dateStart: row.dateStart || "",
        dateExpired: row.dateExpired || ""
      };
      this.formVisible = true;
    },
    resetForm() {
      this.$refs.formRef?.resetFields();
    },
    async handleSubmit() {
      this.$refs.formRef.validate(async v => {
        if (!v) return;
        this.submitting = true;
        try {
          let res;
          if (this.isEdit)
            res = await cmdbAPI.updateAssetPermission(this.editId, this.form);
          else res = await cmdbAPI.createAssetPermission(this.form);
          if (res.data.code === 200) {
            this.$message.success(this.isEdit ? "更新成功" : tr("创建成功"));
            this.formVisible = false;
            this.fetch();
          } else {
            this.$message.error(res.data.message);
          }
        } catch (e) {
          this.$message.error(
            "操作失败: " + (e.response?.data?.message || e.message)
          );
        } finally {
          this.submitting = false;
        }
      });
    },
    handleDelete(row) {
      this.$confirm(`确定删除授权规则 "${row.name}" 吗？`, tr("确认"), {
        type: "warning"
      })
        .then(async () => {
          try {
            const res = await cmdbAPI.deleteAssetPermission(row.id);
            if (res.data.code === 200) {
              this.$message.success(tr("删除成功"));
              this.fetch();
            } else this.$message.error(res.data.message);
          } catch (e) {
            this.$message.error(tr("删除失败"));
          }
        })
        .catch(() => {});
    },
    // Host Selection
    showHostSelector() {
      this.hostSelectorVisible = true;
      this.hostFilter = { groupId: undefined, keyword: "", status: undefined };
      this.hostPage = 1;
      this.selectedHostIds = [];
      this.loadHostList();
    },
    onHostSelect(rows) {
      this.selectedHostIds = rows.map(r => r.id);
    },
    selectAllHosts() {
      this.$nextTick(() => {
        if (this.$refs.hostTable) {
          this.hostList.forEach(row => {
            this.$refs.hostTable.toggleRowSelection(row, true);
          });
        }
      });
    },
    confirmHostSelect() {
      const existing = new Set(this.form.hostIds);
      this.selectedHostIds.forEach(id => existing.add(id));
      this.form.hostIds = Array.from(existing);
      this.hostSelectorVisible = false;
      this.$message.success(`已选择 ${this.selectedHostIds.length} 台主机`);
    },
    removeHost(id) {
      this.form.hostIds = this.form.hostIds.filter(i => i !== id);
    },
    showPhysicalSelector() {
      this.physicalFilter = {
        idcId: undefined,
        cabinetId: undefined,
        keyword: ""
      };
      this.loadPhysicalList();
    },
    onPhysicalSelect(rows) {
      rows.forEach(r => {
        if (!this.form.physicalIds.includes(r.id))
          this.form.physicalIds.push(r.id);
      });
    },
    removePhysical(id) {
      this.form.physicalIds = this.form.physicalIds.filter(i => i !== id);
    },
    showNetworkSelector() {
      this.networkFilter = {
        idcId: undefined,
        deviceType: undefined,
        keyword: ""
      };
      this.loadNetworkList();
    },
    onNetworkSelect(rows) {
      rows.forEach(r => {
        if (!this.form.networkIds.includes(r.id))
          this.form.networkIds.push(r.id);
      });
    },
    removeNetwork(id) {
      this.form.networkIds = this.form.networkIds.filter(i => i !== id);
    },
    async loadCabinetByIdc(idcId) {
      if (!idcId) {
        this.cabinetList = [];
        return;
      }
      try {
        const res = await cmdbAPI.getCabinetByIdc(idcId);
        if (res.data.code === 200) this.cabinetList = res.data.data.list || [];
      } catch (_) {
        this.cabinetList = [];
      }
    },
    // 将JSON数组格式的userIds解析为用户名称字符串
    resolveUserNames(userIdsJSON) {
      const ids = this.safeJSON(userIdsJSON, []);
      return (
        ids
          .map(id => {
            const u = this.userList.find(u => u.id === id);
            return u ? u.nickname || u.username : "用户#" + id;
          })
          .join(", ") || "-"
      );
    },
    // 将JSON数组格式的groupIds解析为用户组名称字符串
    resolveGroupNames(groupIdsJSON) {
      const ids = this.safeJSON(groupIdsJSON, []);
      return (
        ids
          .map(id => {
            const g = this.userGroupList.find(g => g.id === id);
            return g ? g.name : "组#" + id;
          })
          .join(", ") || "-"
      );
    }
  },
  watch: {
    "physicalFilter.idcId"(val) {
      if (val) this.loadCabinetByIdc(val);
    },
    "networkFilter.idcId"(val) {
      if (val) this.loadCabinetByIdc(val);
    }
  }
};
</script>

<style scoped>
.asset-permission-management {
  padding: 20px;
  min-height: 100vh;
  background: var(--bg-page);
}
.main-card {
  border-radius: var(--radius-lg);
}
.card-header .title {
  font-size: 18px;
  font-weight: 600;
  color: var(--text-primary);
}
.search-section {
  background: var(--bg-card-alt);
  padding: 16px 20px;
  border-radius: var(--radius);
  margin-bottom: 16px;
  border: 1px solid var(--border-light);
}
.action-section {
  margin-bottom: 16px;
}
.pagination-section {
  display: flex;
  justify-content: center;
  margin-top: 20px;
}
</style>
