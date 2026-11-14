<template>
  <div class="web-management" :class="{ 'logs-page': section === 'logs' }">
    <header v-if="!embedded" class="web-heading">
      <div>
        <h2>{{ $tr(names[section] || "Web 管理") }}</h2>
        <p>{{ $tr("连接服务、管理配置与证书，追踪每一次发布") }}</p>
      </div>
      <div v-if="section === 'logs'" class="log-scope-actions">
        <el-button
          :aria-expanded="logScopeExpanded"
          @click="logScopeExpanded = !logScopeExpanded"
        >
          {{ $tr("范围筛选")
          }}<el-tag v-if="logScopeCount" size="small">{{
            logScopeCount
          }}</el-tag>
        </el-button>
        <el-button @click="refreshPage">{{ $tr("刷新") }}</el-button>
      </div>
      <WebResourceIcon v-else />
    </header>
    <el-tabs
      v-if="submenus.length > 1"
      v-model="subsection"
      :before-leave="confirmTabChange"
      @tab-change="reset"
      ><el-tab-pane
        v-for="s in submenus"
        :key="s"
        :name="s"
        :label="$tr(s === 'configurations' ? '发布草稿' : names[s])"
    /></el-tabs>
    <div v-show="section !== 'logs' || logScopeExpanded" class="toolbar">
      <el-input
        v-model="filters.name"
        clearable
        :placeholder="$tr('搜索名称')"
        @change="reset"
      />
      <el-select
        v-model="filters.groupId"
        clearable
        :placeholder="$tr('业务分组')"
        @change="reset"
        ><el-option
          v-for="g in groups"
          :key="g.id"
          :label="g.name"
          :value="g.id"
      /></el-select>
      <el-select
        v-if="kind === 'configurations'"
        v-model="filters.configGroupId"
        clearable
        :placeholder="$tr('配置分组')"
        @change="reset"
        ><el-option
          v-for="g in configGroupOptions"
          :key="g.id"
          :label="g.name"
          :value="g.id"
      /></el-select>
      <el-select
        v-if="
          ![
            'clusters',
            'acme-accounts',
            'dns-accounts',
            'renewal-policies'
          ].includes(kind)
        "
        v-model="filters.clusterId"
        clearable
        :placeholder="$tr('集群')"
        @change="reset"
        ><el-option
          v-for="c in options.clusters || []"
          :key="c.id"
          :label="c.name"
          :value="c.id"
      /></el-select>
      <el-select
        v-model="filters.status"
        clearable
        :placeholder="$tr('状态')"
        @change="reset"
        ><el-option
          v-for="s in statusOptions"
          :key="s"
          :value="s"
          :label="
            $tr(
              kind === 'audit' && s === 'unknown'
                ? '结果待核验'
                : statusNames[s]
            )
          " /></el-select
      ><EnvironmentSelect
        v-model="filters.environment"
        clearable
        :placeholder="$tr('环境')"
        @change="reset"
      />
      <el-button v-if="section === 'certificates'" @click="setupGuide = true">{{
        $tr("证书配置教程")
      }}</el-button>
      <el-button @click="refreshPage">{{ $tr("刷新") }}</el-button
      ><el-button
        v-if="
          canEditKind &&
          !['audit', 'terminal', 'logs', 'native-configs'].includes(kind)
        "
        :type="kind === 'certificates' ? 'default' : 'primary'"
        @click="edit()"
        >{{ $tr(kind === "certificates" ? "高级配置" : "新增") }}</el-button
      >
      <el-button
        v-if="kind === 'certificates' && canWrite && platformAccess.isAdmin"
        type="primary"
        @click="requestCertificateDialog = true"
        >{{ $tr("通过 Cloudflare 申请证书") }}</el-button
      >
      <el-button
        v-if="kind === 'certificates' && canWrite"
        @click="createImportDialog = true"
        >{{ $tr("导入证书") }}</el-button
      >
    </div>
    <el-alert
      v-if="linkedDraft && kind === 'configurations'"
      :title="$tr('已定位配置草稿') + '：' + linkedDraft.name"
      type="info"
      show-icon
      @close="clearLinkedDraft"
    />
    <el-alert
      v-if="workerOnline === false && canExecute"
      :title="$tr('Web Worker 当前不在线，提交的任务会保留在队列中')"
      type="warning"
      :closable="false"
    /><el-alert
      v-if="encryptionReady === false && canExecute"
      :title="$tr('Web 加密密钥尚未配置，请联系管理员完成部署配置')"
      type="warning"
      :closable="false"
    /><el-alert
      v-if="error"
      :title="error"
      type="error"
      show-icon
      :closable="false"
    />
    <el-alert
      v-if="optionError"
      :title="optionError"
      type="error"
      :closable="false"
    />
    <WebNativeConfigurations
      ref="nativeEditor"
      v-if="kind === 'native-configs'"
      :instances="filteredInstances"
      :all-instances="options.instances || []"
      :config-groups="options['config-groups'] || []"
      :initial-instance-id="routeInstanceId"
      :refresh-key="nativeRefreshKey"
      @saved="nativeSaved"
    />
    <WebLogs
      v-else-if="section === 'logs'"
      :sites="options.sites || []"
      :initial-site-id="Number(route.query.siteId) || undefined"
      :layout-key="logScopeExpanded"
      :instances="filteredInstances"
      :initial-instance-id="routeInstanceId"
    />
    <template v-else>
      <el-table
        :key="kind"
        v-loading="loading"
        :data="rows"
        row-key="id"
        class="resource-table asset-table"
      >
        <el-table-column
          prop="name"
          :label="$tr('名称')"
          min-width="180"
          show-overflow-tooltip
          ><template #default="{ row }"
            ><div class="resource-identity">
              <WebResourceIcon
                :kind="kind"
                :provider="row.spec?.provider"
                :engine="row.engine"
              /><span>{{ row.name || $tr(row.action) + " #" + row.id }}</span>
            </div></template
          ></el-table-column
        >
        <el-table-column
          v-if="kind !== 'audit'"
          :label="$tr('业务分组')"
          min-width="130"
          ><template #default="{ row }">{{
            groupName(row.groupId)
          }}</template></el-table-column
        >
        <el-table-column
          v-if="kind !== 'audit'"
          prop="environment"
          :label="$tr('环境')"
          width="120"
        />
        <el-table-column
          v-if="kind === 'configurations'"
          :label="$tr('配置分组')"
          min-width="140"
          ><template #default="{ row }">
            <el-tag
              v-if="configGroupName(row.spec?.configGroupId)"
              size="small"
              >{{ configGroupName(row.spec?.configGroupId) }}</el-tag
            >
            <span v-else>—</span>
            <small class="muted">{{
              $tr("共 {value0} 个文件", {
                value0: row.spec?.files?.length || 0
              })
            }}</small>
          </template></el-table-column
        >
        <el-table-column
          v-if="kind === 'certificates'"
          :label="$tr('到期')"
          width="150"
          ><template #default="{ row }">
            <el-tag
              v-if="row.observed?.notAfter"
              size="small"
              :type="expiryType(row.observed.days)"
            >
              {{
                row.observed.days < 0
                  ? $tr("已过期")
                  : $tr("剩余 {value0} 天", { value0: row.observed.days })
              }}
            </el-tag>
            <span v-else>—</span>
            <div v-if="row.observed?.notAfter" class="muted">
              {{ new Date(row.observed.notAfter).toLocaleDateString() }}
            </div>
          </template></el-table-column
        >
        <el-table-column
          v-if="kind === 'dns-accounts'"
          :label="$tr('DNS 服务商')"
          min-width="150"
          ><template #default="{ row }"
            ><strong>{{ dnsProviderName(row.spec?.provider) }}</strong>
            <div class="muted">
              {{ $tr(row.spec?.credentialId ? "配置中心凭据" : "专用凭据")
              }}<span v-if="row.spec?.credentialId">
                #{{ row.spec.credentialId }}</span
              >
            </div></template
          ></el-table-column
        >
        <el-table-column
          v-if="kind === 'dns-accounts'"
          :label="$tr('最近检查')"
          min-width="200"
          ><template #default="{ row }"
            ><template v-if="row.observed?.checkedAt"
              ><el-tag :type="row.observed.valid ? 'success' : 'warning'">{{
                $tr(row.observed.valid ? "读取检查通过" : "检查未通过")
              }}</el-tag>
              <div class="muted">
                {{ new Date(row.observed.checkedAt).toLocaleString() }}
              </div>
              <div v-if="row.observed.domain" class="muted">
                {{ row.observed.domain }}
              </div></template
            ><span v-else>{{ $tr("尚未检查") }}</span></template
          ></el-table-column
        >
        <el-table-column
          v-if="kind === 'acme-accounts'"
          :label="$tr('联系邮箱')"
          min-width="220"
          ><template #default="{ row }"
            >{{ row.spec.email }}
            <div class="muted">
              {{
                $tr(
                  row.spec.directoryUrl.includes("acme-staging")
                    ? "测试环境"
                    : "正式或自定义 CA"
                )
              }}
            </div></template
          ></el-table-column
        >
        <el-table-column
          v-if="kind === 'renewal-policies'"
          :label="$tr('下次续期')"
          min-width="170"
          ><template #default="{ row }">{{
            row.spec?.nextRunAt
              ? new Date(row.spec.nextRunAt).toLocaleString()
              : "—"
          }}</template></el-table-column
        >
        <el-table-column :label="$tr('状态')" width="140"
          ><template #default="{ row }"
            ><el-tag :type="tagType(row.status)">{{
              $tr(
                (kind === "audit" && row.status === "unknown"
                  ? "结果待核验"
                  : statusNames[row.status]) ||
                  row.status ||
                  "未知"
              )
            }}</el-tag></template
          ></el-table-column
        >
        <el-table-column
          v-if="kind === 'sources'"
          :label="$tr('管理连接')"
          min-width="210"
          ><template #default="{ row }"
            ><div>
              {{ connectionView(row)?.address || "—" }}:{{
                connectionView(row)?.port || 22
              }}
            </div>
            <small>{{ connectionView(row)?.alias || "" }}</small></template
          ></el-table-column
        >
        <el-table-column
          v-if="kind === 'sources'"
          :label="$tr('最近发现')"
          min-width="220"
          ><template #default="{ row }"
            ><div>
              {{
                row.collectedAt
                  ? new Date(row.collectedAt).toLocaleString()
                  : $tr("尚未检查")
              }}
            </div>
            <el-text
              v-if="row.observed?.message"
              type="warning"
              :title="row.observed.message"
              truncated
              >{{ row.observed.message }}</el-text
            ></template
          ></el-table-column
        >
        <el-table-column
          v-if="kind === 'audit'"
          :label="$tr('操作')"
          min-width="140"
          ><template #default="{ row }"
            >{{ $tr(actionLabel(row.action)) }} #{{ row.id }}</template
          ></el-table-column
        >
        <el-table-column
          v-if="kind === 'sources'"
          :label="$tr('运行环境')"
          width="110"
          ><template #default="{ row }">{{
            $tr(statusNames[row.spec?.runtime] || row.spec?.runtime)
          }}</template></el-table-column
        >
        <el-table-column
          v-if="kind === 'clusters'"
          :label="$tr('管理方式')"
          width="130"
          ><template #default="{ row }">{{
            $tr(statusNames[row.management])
          }}</template></el-table-column
        >
        <el-table-column
          v-if="['certificates', 'sites'].includes(kind)"
          :label="$tr('域名')"
          min-width="200"
          show-overflow-tooltip
          ><template #default="{ row }">{{
            row.spec?.domains?.join(", ")
          }}</template></el-table-column
        >
        <el-table-column v-if="kind === 'clusters'" label="VIP" min-width="190"
          ><template #default="{ row }"
            ><template v-if="row.spec?.vip"
              ><div>{{ row.spec.vip.cidr }}</div>
              <el-tag size="small" :type="tagType(vipRowStatus(row))">{{
                $tr(statusNames[vipRowStatus(row)] || vipRowStatus(row))
              }}</el-tag></template
            ><span v-else>—</span></template
          ></el-table-column
        >
        <el-table-column
          v-if="kind === 'sites'"
          :label="$tr('监听 / HTTPS')"
          min-width="150"
          ><template #default="{ row }"
            ><div>{{ row.spec?.ports?.join(", ") || "—" }}</div>
            <el-tag :type="row.spec?.tls ? 'success' : 'info'" size="small">{{
              row.spec?.tls ? "HTTPS" : "HTTP"
            }}</el-tag
            ><span class="muted">
              · {{ $tr(statusNames[row.spec?.mode] || row.spec?.mode) }}</span
            ></template
          ></el-table-column
        >
        <el-table-column
          v-if="['sites', 'upstreams'].includes(kind)"
          :label="$tr('配置文件路径')"
          min-width="230"
          show-overflow-tooltip
          ><template #default="{ row }">{{
            row.spec?.path || "—"
          }}</template></el-table-column
        >
        <el-table-column
          v-if="kind === 'upstreams'"
          :label="$tr('上游成员')"
          min-width="170"
          show-overflow-tooltip
          ><template #default="{ row }">{{
            row.spec?.members?.map(m => m.address).join(", ") || "—"
          }}</template></el-table-column
        >
        <el-table-column :label="$tr('操作')" width="220" fixed="right">
          <template #default="{ row }">
            <div class="asset-actions resource-actions">
              <el-button link type="primary" @click="detail(row)">{{
                $tr("详情")
              }}</el-button>
              <el-button
                v-if="canEditKind && kind !== 'audit'"
                link
                type="primary"
                @click="edit(row)"
                >{{ $tr("编辑") }}</el-button
              >
              <el-dropdown
                :key="kind + '-' + row.id + '-' + filters.page"
                :persistent="false"
                v-if="rowActions(row).length"
                trigger="click"
                placement="bottom-end"
                @visible-change="
                  v =>
                    v &&
                    kind === 'clusters' &&
                    canExecute &&
                    loadCapabilities(row)
                "
              >
                <el-button text size="small" :aria-label="$tr('更多操作')">
                  {{ $tr("更多")
                  }}<el-icon class="more-icon"><ArrowDown /></el-icon>
                </el-button>
                <template #dropdown>
                  <el-dropdown-menu>
                    <el-dropdown-item
                      v-for="action in rowActions(row)"
                      :key="action.key"
                      :disabled="action.disabled"
                      :divided="action.divided"
                      @click="action.run?.()"
                    >
                      <span
                        :title="action.reason && $tr(action.reason)"
                        :class="{ 'resource-action-danger': action.danger }"
                        >{{ $tr(action.label) }}</span
                      >
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </template>
              </el-dropdown>
            </div>
          </template>
        </el-table-column>
      </el-table>
      <div class="resource-pagination">
        <el-pagination
          :current-page="filters.page"
          :page-size="filters.size"
          :page-sizes="[10, 20, 50, 100]"
          :pager-count="5"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          background
          @size-change="changePageSize"
          @current-change="changePage"
        />
      </div>
    </template>
    <template v-if="section === 'audit'"
      ><el-divider>{{ $tr("连接与查询审计") }}</el-divider
      ><EventJournal :cluster-id="filters.clusterId"
    /></template>
    <SourceDialog
      v-model="sourceDialog"
      :record="editingSource"
      :groups="groups"
      :hosts="options.hosts || []"
      :connections="options.connections || []"
      :instances="options.instances || []"
      :initial-group-id="filters.groupId"
      :environment="filters.environment"
      @saved="refreshPage"
    />
    <SiteDialog
      v-model="siteDialog"
      :record="editingSite"
      :groups="groups"
      :clusters="options.clusters || []"
      :instances="options.instances || []"
      :certificates="options.certificates || []"
      :bindings="options['certificate-bindings'] || []"
      @saved="refreshPage"
      @render="render"
    />
    <ClusterDialog
      v-model="clusterDialog"
      :record="editingCluster"
      :groups="groups"
      :initial-group-id="filters.groupId"
      :initial-environment="filters.environment"
      :can-execute="canExecute"
      @saved="refreshPage"
    />
    <ClusterDetail
      v-model="clusterDetailDialog"
      :cluster-id="detailClusterId"
      :initial-tab="detailClusterTab"
      @action="handleClusterAction"
      @navigate="jump"
    />
    <el-dialog
      append-to-body
      :lock-scroll="false"
      class="web-resource-dialog"
      v-model="editing"
      :title="$tr(names[editKind])"
      width="820px"
      destroy-on-close
    >
      <el-form label-position="top" @submit.prevent>
        <div class="form-grid">
          <el-form-item :label="$tr('名称')" required
            ><el-input v-model="form.name" maxlength="100"
          /></el-form-item>
          <el-form-item :label="$tr('业务分组')" required
            ><el-select v-model="form.groupId" :disabled="!!form.id" filterable
              ><el-option
                v-for="g in groups"
                :key="g.id"
                :label="g.name"
                :value="g.id" /></el-select
          ></el-form-item>
          <el-form-item :label="$tr('环境')"
            ><EnvironmentSelect v-model="form.environment"
          /></el-form-item>
          <el-form-item
            v-if="needsCluster(editKind)"
            :label="$tr('集群')"
            required
            ><el-select
              v-model="form.clusterId"
              :disabled="!!form.id"
              filterable
              @change="selectCluster"
              ><el-option
                v-for="c in options.clusters || []"
                :key="c.id"
                :label="c.name"
                :value="c.id" /></el-select
          ></el-form-item>
          <el-form-item
            v-for="f in visibleFields"
            :key="f.key"
            :label="$tr(f.label)"
            :required="f.required"
          >
            <el-switch v-if="f.type === 'boolean'" v-model="form.spec[f.key]" />
            <el-input-number
              v-else-if="f.type === 'number'"
              v-model="form.spec[f.key]"
              :min="0"
            />
            <el-select
              v-else-if="f.type === 'select' || f.type === 'multiple'"
              v-model="form.spec[f.key]"
              :multiple="f.type === 'multiple'"
              clearable
              filterable
              ><el-option
                v-for="o in fieldOptions(f)"
                :key="o.value"
                :value="o.value"
                :label="o.label"
            /></el-select>
            <el-input
              v-else-if="['csv', 'numbers', 'lines'].includes(f.type || '')"
              :model-value="
                (form.spec[f.key] || []).join(f.type === 'lines' ? '\n' : ', ')
              "
              :type="f.type === 'lines' ? 'textarea' : 'text'"
              @update:model-value="v => setArray(f, v)"
            />
            <el-input v-else v-model="form.spec[f.key]" />
            <small v-if="f.tip">{{ $tr(f.tip) }}</small>
          </el-form-item>
        </div>
        <el-alert
          v-if="editKind === 'sources'"
          :title="
            $tr(
              '新建部署使用独立目录 /etc/autoops-web/名称/current/nginx.conf；主机服务名使用 autoops-web- 前缀。'
            )
          "
          type="info"
          :closable="false"
        />
        <template v-if="editKind === 'sites'">
          <el-input
            v-if="form.spec.mode === 'raw'"
            v-model="form.spec.raw"
            type="textarea"
            :rows="16"
            class="code-input"
          />
          <template v-else
            ><h4>{{ $tr("路径规则") }}</h4>
            <div
              v-for="(l, index) in form.spec.locations"
              :key="index"
              class="location-row"
            >
              <el-input
                v-model="l.path"
                :placeholder="$tr('匹配路径')"
              /><el-select v-model="l.kind"
                ><el-option :label="$tr('反向代理')" value="proxy" /><el-option
                  :label="$tr('静态目录')"
                  value="static" /><el-option
                  :label="$tr('重定向')"
                  value="redirect" /></el-select
              ><el-input
                v-model="l.target"
                :placeholder="$tr('目标地址或目录')"
              /><el-input-number
                v-model="l.timeout"
                :min="1"
                :max="3600"
                :title="$tr('代理超时（秒）')"
              /><el-input
                :model-value="headerText(l.headers)"
                :placeholder="$tr('请求头，每行 名称: 值')"
                type="textarea"
                @update:model-value="v => setHeaders(l, v)"
              /><el-switch
                v-model="l.websocket"
                active-text="WebSocket"
              /><el-button
                link
                type="danger"
                @click="form.spec.locations.splice(index, 1)"
                >{{ $tr("移除") }}</el-button
              >
            </div>
            <el-button
              @click="
                form.spec.locations.push({
                  path: '/',
                  kind: 'proxy',
                  target: '',
                  headers: {},
                  timeout: 60
                })
              "
              >{{ $tr("添加路径规则") }}</el-button
            ></template
          >
        </template>
        <template v-if="editKind === 'upstreams'"
          ><h4>{{ $tr("上游成员") }}</h4>
          <div
            v-for="(m, index) in form.spec.members"
            :key="index"
            class="location-row"
          >
            <el-input
              v-model="m.address"
              :placeholder="$tr('地址:端口')"
            /><el-input-number
              v-model="m.weight"
              :min="1"
              :max="1000"
            /><el-switch
              v-model="m.backup"
              :active-text="$tr('备用')"
            /><el-button @click="form.spec.members.splice(index, 1)">{{
              $tr("移除")
            }}</el-button>
          </div>
          <el-button
            @click="
              form.spec.members.push({ address: '', weight: 1, backup: false })
            "
            >{{ $tr("添加成员") }}</el-button
          ></template
        >
        <template v-if="editKind === 'configurations'"
          ><el-alert
            :title="
              $tr('变更会先在每个节点上执行 nginx -t，校验失败不会写入配置')
            "
            type="info"
            :closable="false" />
          <div class="form-grid">
            <el-form-item :label="$tr('目标实例')">
              <el-select
                v-model="form.spec.instanceIds"
                multiple
                clearable
                :placeholder="$tr('留空表示整个集群')"
              >
                <el-option
                  v-for="instance in options.instances?.filter(
                    i => i.clusterId === form.clusterId
                  ) || []"
                  :key="instance.id"
                  :value="instance.id"
                  :label="instance.name"
                />
              </el-select>
            </el-form-item>
            <el-form-item :label="$tr('配置分组')"
              ><el-select
                v-model="form.spec.configGroupId"
                clearable
                filterable
                :placeholder="$tr('未分组')"
                ><el-option
                  v-for="g in formConfigGroups"
                  :key="g.id"
                  :label="g.name"
                  :value="g.id" /></el-select
            ></el-form-item>
          </div>
          <div class="toolbar">
            <el-select
              v-model="importInstance"
              :placeholder="$tr('从实例读取配置')"
              ><el-option
                v-for="i in options.instances?.filter(
                  i => i.clusterId === form.clusterId
                ) || []"
                :key="i.id"
                :label="i.name"
                :value="i.id" /></el-select
            ><el-button :disabled="!importInstance" @click="readConfig">{{
              $tr("读取配置")
            }}</el-button
            ><el-button
              @click="
                form.spec.files.push({ path: '', content: '', mode: 420 })
              "
              >{{ $tr("添加文件") }}</el-button
            ><el-button
              :disabled="!form.clusterId || !changedFiles.length"
              :loading="validating"
              @click="validateFormConfig"
              >{{ $tr("校验语法") }}</el-button
            ><el-checkbox v-model="onlyChangedFiles">{{
              $tr("仅显示已修改")
            }}</el-checkbox
            ><el-select
              v-model="fileGroupFilter"
              clearable
              :placeholder="$tr('分组筛选')"
              ><el-option
                v-for="g in fileGroupNames"
                :key="g"
                :label="g"
                :value="g"
            /></el-select>
          </div>
          <el-input
            v-model="fileSearch"
            clearable
            :placeholder="$tr('搜索文件路径或配置内容')" />
          <el-alert
            v-if="validationResult"
            :title="
              validationResult.valid
                ? $tr('Nginx 语法校验通过')
                : $tr('Nginx 语法校验未通过，配置不会被更新')
            "
            :type="validationResult.valid ? 'success' : 'error'"
            :closable="false"
            show-icon
          >
            <div
              v-for="line in validationResult.lines"
              :key="line"
              class="validation-line"
            >
              {{ line }}
            </div>
          </el-alert>
          <div
            v-for="(f, index) in visibleConfigFiles"
            :key="configFileKey(f)"
            class="config-file"
          >
            <div class="toolbar">
              <el-input
                v-model="f.path"
                :placeholder="$tr('配置文件路径')"
              /><el-select
                v-model="f.group"
                clearable
                filterable
                allow-create
                default-first-option
                class="file-group"
                :placeholder="$tr('文件分组')"
                ><el-option
                  v-for="g in fileGroupNames"
                  :key="g"
                  :label="g"
                  :value="g"
              /></el-select>
              <el-tag
                v-if="changedFilePaths.has(f.path)"
                type="warning"
                size="small"
                >{{ $tr("已修改") }}</el-tag
              >
              <el-button link type="danger" @click="removeFile(f)">{{
                $tr("移除")
              }}</el-button>
            </div>
            <el-checkbox
              :model-value="!!f.deleted"
              @change="v => markFileDeletion(f, Boolean(v))"
              >{{ $tr("发布时删除该文件") }}</el-checkbox
            ><el-input
              v-model="f.content"
              :disabled="f.deleted"
              type="textarea"
              :rows="14"
              spellcheck="false"
              class="code-input"
            />
          </div>
          <el-divider>{{ $tr("节点变量") }}</el-divider>
          <div
            v-for="i in options.instances?.filter(
              i => i.clusterId === form.clusterId
            ) || []"
            :key="i.id"
            class="location-row"
          >
            <span>{{ i.name }}</span
            ><el-input
              type="textarea"
              :model-value="headerText(form.spec.variables?.[i.id])"
              :placeholder="$tr('每行 变量名: 值，配置中使用 ${变量名}')"
              @update:model-value="v => setVariables(i.id, v)"
            /></div
        ></template>
        <template v-if="editKind === 'dns-accounts'"
          ><el-divider>{{ $tr("专用 DNS 凭据") }}</el-divider>
          <div class="form-grid">
            <el-form-item
              :label="
                $tr(
                  form.spec.provider === 'cloudflare' ? 'API Token' : '密钥 ID'
                )
              "
              ><el-input
                v-model="dnsKey"
                show-password
                autocomplete="new-password" /></el-form-item
            ><el-form-item
              v-if="form.spec.provider !== 'cloudflare'"
              :label="$tr('密钥 Secret')"
              ><el-input
                v-model="dnsSecret"
                show-password
                autocomplete="new-password"
            /></el-form-item>
          </div>
          <small>{{
            $tr("编辑时留空保留原凭据，凭据不会回显")
          }}</small></template
        >
      </el-form>
      <template #footer
        ><el-button @click="editing = false">{{ $tr("取消") }}</el-button
        ><el-button type="primary" :loading="saving" @click="save">{{
          $tr("保存")
        }}</el-button></template
      >
    </el-dialog>
    <el-dialog
      append-to-body
      :lock-scroll="false"
      class="web-resource-dialog"
      v-model="planDialog"
      :title="$tr('操作预检')"
      width="720px"
      @closed="clearPlan"
    >
      <p>{{ planTarget?.name }} · {{ $tr(actionLabel(planAction)) }}</p>
      <el-select
        v-if="['publish', 'validate'].includes(planAction)"
        v-model="parameters.configurationId"
        :placeholder="$tr('配置草稿')"
        @change="plan = null"
        ><el-option
          v-for="r in options.configurations?.filter(
            r => r.clusterId === planTarget?.id
          ) || []"
          :key="r.id"
          :label="r.name"
          :value="r.id"
      /></el-select>
      <el-select
        v-if="planAction === 'deploy-certificate'"
        v-model="parameters.bindingId"
        :placeholder="$tr('证书部署绑定')"
        @change="plan = null"
        ><el-option
          v-for="r in options['certificate-bindings']?.filter(
            r => r.clusterId === planTarget?.id
          ) || []"
          :key="r.id"
          :label="r.name"
          :value="r.id"
      /></el-select>
      <el-input-number
        v-if="planAction === 'rollback'"
        v-model="parameters.revisionId"
        :min="1"
        @change="plan = null"
      />
      <el-input-number
        v-if="planAction === 'reconcile'"
        v-model="parameters.operationId"
        :min="1"
        @change="plan = null"
      />
      <el-table v-if="plan" :data="plan.checks"
        ><el-table-column prop="name" :label="$tr('检查项')" /><el-table-column
          :label="$tr('结果')"
          width="90"
          ><template #default="{ row }"
            ><el-tag :type="row.passed ? 'success' : 'danger'">{{
              $tr(row.passed ? "通过" : "失败")
            }}</el-tag></template
          ></el-table-column
        ><el-table-column prop="message" :label="$tr('说明')"
      /></el-table>
      <el-collapse v-if="planPreview.length"
        ><el-collapse-item
          v-for="node in planPreview"
          :key="node.instanceId"
          :name="node.instanceId"
          :title="$tr('节点渲染结果') + ' · ' + node.name"
          ><div v-for="file in node.files" :key="file.path">
            <strong>{{ file.path }}</strong>
            <pre class="detail-code">{{
              file.deleted ? $tr("发布时删除该文件") : file.content
            }}</pre>
          </div></el-collapse-item
        ></el-collapse
      >
      <template #footer
        ><el-button @click="planDialog = false">{{ $tr("取消") }}</el-button
        ><el-button :loading="planning" @click="preflight">{{
          $tr("重新预检")
        }}</el-button
        ><el-button
          type="primary"
          :disabled="!plan?.executable || executing"
          :loading="executing"
          @click="execute"
          >{{ $tr("确认执行") }}</el-button
        ></template
      >
    </el-dialog>
    <el-dialog
      append-to-body
      :lock-scroll="false"
      class="web-resource-dialog"
      v-model="operationDialog"
      :title="$tr('操作详情')"
      width="900px"
      ><template v-if="operation"
        ><p>
          <el-button
            v-if="
              ['queued', 'preflight', 'running', 'verifying'].includes(
                operation.operation.status
              )
            "
            @click="cancelOperation"
            >{{ $tr("取消后续执行") }}</el-button
          ><el-button
            v-if="operation.operation.status === 'unknown' && canExecute"
            @click="reconcileOperation"
            >{{ $tr("核验中断任务") }}</el-button
          ><el-tag :type="tagType(operation.operation.status)">{{
            $tr(
              operation.operation.status === "unknown"
                ? "结果待核验"
                : statusNames[operation.operation.status]
            )
          }}</el-tag>
          {{ operation.operation.message }}
        </p>
        <el-table :data="operation.steps"
          ><el-table-column
            prop="instanceId"
            :label="$tr('实例')"
            width="90" /><el-table-column
            prop="name"
            :label="$tr('步骤')" /><el-table-column
            prop="status"
            :label="$tr('状态')" /><el-table-column
            prop="message"
            :label="$tr('说明')"
            min-width="250" /></el-table></template
    ></el-dialog>
    <el-drawer
      append-to-body
      :lock-scroll="false"
      destroy-on-close
      v-model="detailsVisible"
      :title="selected?.name"
      size="min(900px,95vw)"
      ><template v-if="selected"
        ><el-descriptions :column="2" border
          ><el-descriptions-item :label="$tr('状态')">{{
            $tr(statusNames[selected.status] || selected.status)
          }}</el-descriptions-item
          ><el-descriptions-item :label="$tr('最近采集')">{{
            selected.collectedAt || "—"
          }}</el-descriptions-item
          ><el-descriptions-item :label="$tr('版本')">{{
            selected.revision
          }}</el-descriptions-item
          ><el-descriptions-item :label="$tr('业务分组')">{{
            groupName(selected.groupId)
          }}</el-descriptions-item></el-descriptions
        >
        <div class="toolbar" v-if="selected.clusterId || kind === 'clusters'">
          <el-button
            v-for="s in ['sites', 'configurations', 'certificates', 'logs']"
            :key="s"
            @click="jump(s, selected!.clusterId || selected!.id)"
            >{{ $tr(names[s]) }}</el-button
          ><el-button @click="jumpAudit(selected!.clusterId || selected!.id)">{{
            $tr("操作记录")
          }}</el-button>
        </div>
        <el-alert
          v-if="selected.observed?.message"
          :title="selected.observed.message"
          type="warning"
          :closable="false"
        />
        <el-descriptions v-if="selected.spec?.runtime" :column="1" border
          ><el-descriptions-item :label="$tr('运行环境')">{{
            selected.spec.runtime
          }}</el-descriptions-item
          ><el-descriptions-item :label="$tr('管理连接')"
            >{{ connectionView(selected)?.address || "—" }}:{{
              connectionView(selected)?.port || 22
            }}</el-descriptions-item
          >
          <el-descriptions-item :label="$tr('最近发现')">{{
            selected.collectedAt
              ? new Date(selected.collectedAt).toLocaleString()
              : $tr("尚未检查")
          }}</el-descriptions-item>
          <el-descriptions-item :label="$tr('主配置入口')">{{
            selected.spec.configPath
          }}</el-descriptions-item
          ><el-descriptions-item :label="$tr('Nginx 版本')">{{
            selected.observed?.version || "—"
          }}</el-descriptions-item
          ><el-descriptions-item :label="$tr('健康检查地址')">{{
            selected.spec.healthUrl || "—"
          }}</el-descriptions-item></el-descriptions
        >
        <el-descriptions v-if="selected.spec?.domains" :column="1" border
          ><el-descriptions-item :label="$tr('域名')">{{
            selected.spec.domains.join(", ")
          }}</el-descriptions-item
          ><el-descriptions-item :label="$tr('证书版本')">{{
            selected.spec.currentVersionId || "—"
          }}</el-descriptions-item
          ><el-descriptions-item :label="$tr('生效时间')">{{
            selected.observed?.notBefore
              ? new Date(selected.observed.notBefore).toLocaleString()
              : "—"
          }}</el-descriptions-item
          ><el-descriptions-item :label="$tr('到期时间')">{{
            selected.observed?.notAfter
              ? new Date(selected.observed.notAfter).toLocaleString()
              : "—"
          }}</el-descriptions-item
          ><el-descriptions-item :label="$tr('建议续期')">{{
            selected.observed?.renewAt
              ? new Date(selected.observed.renewAt).toLocaleString()
              : "—"
          }}</el-descriptions-item
          ><el-descriptions-item :label="$tr('证书指纹')"
            ><span class="mono">{{
              selected.observed?.fingerprint || "—"
            }}</span>
          </el-descriptions-item>
          <el-descriptions-item :label="$tr('导出')"
            ><el-button
              link
              type="primary"
              @click="exportCertificate(selected!, false)"
              >{{ $tr("导出证书") }}</el-button
            >
            <el-button
              v-if="canWrite && platformAccess.isAdmin"
              link
              type="warning"
              @click="exportCertificate(selected!, true)"
              >{{ $tr("导出私钥") }}</el-button
            ></el-descriptions-item
          ></el-descriptions
        >
        <el-table
          v-if="selected.observed?.entries?.length"
          :data="selected.observed.entries"
          ><el-table-column
            prop="name"
            :label="$tr('业务入口')" /><el-table-column
            prop="url"
            label="URL" /><el-table-column prop="status" :label="$tr('状态')"
        /></el-table>
        <el-descriptions v-if="selected.observed?.metrics" :column="2" border
          ><el-descriptions-item
            v-for="(value, metric) in selected.observed.metrics"
            :key="metric"
            :label="String(metric)"
            >{{ value }}</el-descriptions-item
          ></el-descriptions
        >
        <el-table
          v-if="selected.observed?.certificates?.length"
          :data="selected.observed.certificates"
          ><el-table-column
            prop="path"
            :label="$tr('证书部署路径')" /><el-table-column
            prop="fingerprint"
            :label="$tr('证书指纹')"
            show-overflow-tooltip /><el-table-column
            prop="details"
            :label="$tr('证书信息')"
            show-overflow-tooltip
        /></el-table>
        <el-table v-if="selected.observed?.nodes" :data="deploymentNodes"
          ><el-table-column
            prop="instanceId"
            :label="$tr('实例')" /><el-table-column
            prop="versionId"
            :label="$tr('实际证书版本')" /><el-table-column
            prop="fingerprint"
            :label="$tr('证书指纹')"
            show-overflow-tooltip /><el-table-column
            prop="verifiedAt"
            :label="$tr('核验时间')"
        /></el-table>
        <el-collapse v-if="selected.observed?.files?.length"
          ><el-collapse-item
            v-for="file in selected.observed.files"
            :key="file.path"
            :name="file.path"
            :title="file.path"
          >
            <pre class="detail-code">{{ file.content }}</pre>
          </el-collapse-item></el-collapse
        >
        <el-descriptions v-if="kind === 'renewal-policies'" :column="1" border
          ><el-descriptions-item :label="$tr('下次续期检查')">{{
            selected.spec.nextRunAt
          }}</el-descriptions-item
          ><el-descriptions-item :label="$tr('签发任务')"
            ><el-button
              v-if="selected.spec.pendingOperationId"
              link
              @click="showOperation(selected!.spec.pendingOperationId)"
              >#{{ selected.spec.pendingOperationId }}</el-button
            ></el-descriptions-item
          ><el-descriptions-item :label="$tr('部署任务')"
            ><el-button
              v-for="oid in Object.values(
                selected.spec.deployOperationIds || {}
              )"
              :key="String(oid)"
              link
              @click="showOperation(Number(oid))"
              >#{{ oid }}</el-button
            ></el-descriptions-item
          ></el-descriptions
        ></template
      ></el-drawer
    >
    <el-dialog
      append-to-body
      :lock-scroll="false"
      class="web-resource-dialog"
      v-model="membersDialog"
      :title="$tr('集群成员')"
      width="760px"
      ><div class="toolbar">
        <el-select v-model="addInstance" :placeholder="$tr('选择未分组实例')"
          ><el-option
            v-for="i in options.instances?.filter(
              i => !i.clusterId && i.groupId === memberCluster?.groupId
            ) || []"
            :key="i.id"
            :label="i.name"
            :value="i.id" /></el-select
        ><el-button
          :disabled="!addInstance || !canExecute"
          @click="bindMember(false, addInstance)"
          >{{ $tr("加入集群") }}</el-button
        >
      </div>
      <el-table :data="members"
        ><el-table-column prop="name" :label="$tr('名称')" /><el-table-column
          prop="status"
          :label="$tr('状态')"
        /><el-table-column :label="$tr('操作')"
          ><template #default="{ row }"
            ><el-button
              link
              :disabled="!canExecute"
              @click="bindMember(true, row.id)"
              >{{ $tr("移出管理") }}</el-button
            ></template
          ></el-table-column
        ></el-table
      ></el-dialog
    >
    <SetupGuide v-model="setupGuide" kind="certificate" />
    <DnsAccountDialog
      v-model="dnsAccountDialog"
      :account="editingDNSAccount"
      :groups="groups"
      :initial-group-id="filters.groupId"
      :initial-environment="filters.environment"
      @saved="dnsAccountSaved"
    />
    <DnsAccountInspection
      v-model="dnsInspectionDialog"
      :account="inspectedDNSAccount"
      :initial-action="dnsInspectionAction"
      @checked="refreshAccounts"
    />
    <AcmeAccountDialog
      v-model="acmeAccountDialog"
      :account="editingAcmeAccount"
      :groups="groups"
      :initial-group-id="filters.groupId"
      :initial-environment="filters.environment"
      @saved="refreshAccounts"
    />
    <CertificateRequestDialog
      v-model="requestCertificateDialog"
      :groups="groups"
      :dns-accounts="options['dns-accounts'] || []"
      :acme-accounts="options['acme-accounts'] || []"
      :initial-group-id="filters.groupId"
      :initial-environment="filters.environment"
      :can-execute="canExecute"
      @saved="certificateRequestSaved"
      @configure-dns="configureDNS"
      @account-created="loadOptions"
    />
    <CertificateImportDialog
      v-model="createImportDialog"
      :groups="groups"
      :initial-group-id="filters.groupId"
      :initial-environment="filters.environment"
      @saved="certificateImported"
    />
    <el-dialog
      append-to-body
      :lock-scroll="false"
      class="web-resource-dialog"
      v-model="importDialog"
      :title="$tr('导入证书')"
      width="760px"
      ><el-form label-position="top"
        ><el-form-item :label="$tr('PEM 证书链')"
          ><div class="toolbar">
            <label class="file-picker"
              >{{ $tr("选择证书文件") }}
              <input
                type="file"
                accept=".pem,.crt,.cer,.txt"
                @change="loadCertificateFile($event, 'cert')" /></label
            ><small>{{
              $tr("支持 fullchain.pem 或 crt 文本，直接粘贴亦可")
            }}</small>
          </div>
          <el-input
            v-model="certificatePEM"
            type="textarea"
            :rows="8" /></el-form-item
        ><el-form-item :label="$tr('PEM 私钥')"
          ><div class="toolbar">
            <label class="file-picker"
              >{{ $tr("选择私钥文件") }}
              <input
                type="file"
                accept=".pem,.key,.txt"
                @change="loadCertificateFile($event, 'key')" /></label
            ><small>{{ $tr("私钥只在服务端加密保存，不会回显") }}</small>
          </div>
          <el-input
            v-model="privatePEM"
            type="textarea"
            :rows="8"
            autocomplete="off" /></el-form-item></el-form
      ><template #footer
        ><el-button @click="importDialog = false">{{ $tr("取消") }}</el-button
        ><el-button
          type="primary"
          :loading="saving"
          @click="importCertificate"
          >{{ $tr("校验并导入") }}</el-button
        ></template
      ></el-dialog
    >
    <el-dialog
      append-to-body
      :lock-scroll="false"
      class="web-resource-dialog"
      v-model="syntaxDialog"
      :title="$tr('Nginx 语法校验') + ' · ' + syntaxTarget"
      width="720px"
      ><el-table :data="syntaxResults" size="small"
        ><el-table-column
          prop="name"
          :label="$tr('节点')"
          min-width="150"
        /><el-table-column :label="$tr('结果')" width="110"
          ><template #default="{ row }"
            ><el-tag :type="row.valid ? 'success' : 'danger'">{{
              row.valid ? $tr("通过") : $tr("未通过")
            }}</el-tag></template
          ></el-table-column
        ><el-table-column :label="$tr('输出')" min-width="320"
          ><template #default="{ row }">
            <pre class="detail-code">{{ row.message || "nginx: -t ok" }}</pre>
          </template></el-table-column
        ></el-table
      ></el-dialog
    >
    <el-dialog
      append-to-body
      :lock-scroll="false"
      class="web-resource-dialog"
      v-model="historyDialog"
      :title="$tr('版本历史')"
      width="900px"
      ><el-table v-loading="historyLoading" :data="revisions"
        ><el-table-column
          prop="id"
          :label="$tr('版本')"
          width="100"
        /><el-table-column
          prop="createdAt"
          :label="$tr('创建时间')"
        /><el-table-column
          prop="digest"
          :label="$tr('摘要')"
          show-overflow-tooltip
        /><el-table-column :label="$tr('操作')"
          ><template #default="{ row }"
            ><el-button link @click="showRevision(row)">{{
              $tr(row.kind === "certificate" ? "证书信息" : "查看配置")
            }}</el-button
            ><el-button
              v-if="row.kind === 'certificate' && historyCertificate"
              link
              @click="exportCertificate(historyCertificate, false, row.id)"
              >{{ $tr("导出证书") }}</el-button
            >
            <el-button
              v-if="
                row.kind === 'certificate' &&
                historyCertificate &&
                canWrite &&
                platformAccess.isAdmin
              "
              link
              @click="exportCertificate(historyCertificate, true, row.id)"
              >{{ $tr("导出私钥") }}</el-button
            >
            <el-button
              v-if="row.kind === 'configuration'"
              link
              @click="compareRevision(row)"
              >{{ $tr("版本对比") }}</el-button
            ><el-button
              v-if="canExecute && row.kind === 'configuration'"
              link
              @click="rollback(row)"
              >{{ $tr("回滚") }}</el-button
            ></template
          ></el-table-column
        ></el-table
      ><template v-if="revisionView?.kind === 'certificate'"
        ><el-descriptions :column="2" border
          ><el-descriptions-item :label="$tr('签发者')">{{
            revisionView.content.issuer
          }}</el-descriptions-item
          ><el-descriptions-item :label="$tr('有效期至')">{{
            revisionView.content.notAfter
          }}</el-descriptions-item
          ><el-descriptions-item :label="$tr('证书指纹')">{{
            revisionView.content.fingerprint
          }}</el-descriptions-item
          ><el-descriptions-item :label="$tr('建议续期时间')">{{
            revisionView.content.renewAt
          }}</el-descriptions-item></el-descriptions
        ></template
      ><template v-else-if="revisionView"
        ><el-collapse
          ><el-collapse-item
            v-for="(files, node) in revisionView.content.nodes"
            :key="node"
            :title="$tr('实例') + ' ' + node"
            ><div v-for="file in files" :key="file.path">
              <strong>{{ file.path }}</strong>
              <pre class="detail-code">{{ file.content }}</pre>
            </div></el-collapse-item
          ></el-collapse
        ></template
      >
      <div v-for="file in revisionDiff" :key="file.instanceId + file.path">
        <h4>{{ file.instanceId }} · {{ file.path }}</h4>
        <pre class="detail-code">{{ file.diff }}</pre>
      </div></el-dialog
    >
  </div>
</template>
<script setup lang="ts">
import SetupGuide from "./SetupGuide.vue";
import EnvironmentSelect from "./EnvironmentSelect.vue";
import { computed, reactive, ref, watch, onUnmounted } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import { tr } from "@/locales";
import ArrowDown from "~icons/ep/arrow-down";
import { canPlatform, platformAccess } from "@/utils/platformAccess";
import {
  webRequest,
  webList,
  webOptions,
  webQuery,
  webValidateClusterConfig,
  webExportCertificate,
  webInstanceConfig,
  type WebConfigFile,
  type WebConfigValidation,
  type WebResource,
  type WebPlan,
  type WebCapability
} from "@/api/webservice";
import {
  names,
  fieldSets,
  defaults,
  statusNames,
  type Field
} from "./definitions";
import WebLogs from "./Logs.vue";
import WebResourceIcon from "./WebResourceIcon.vue";
import SourceDialog from "./SourceDialog.vue";
import SiteDialog from "./SiteDialog.vue";
import ClusterDetail from "./ClusterDetail.vue";
import ClusterDialog from "./ClusterDialog.vue";
const clusterDialog = ref(false),
  editingCluster = ref<WebResource>();
import EventJournal from "./EventJournal.vue";
import WebNativeConfigurations from "./NativeConfigurations.vue";
import CertificateImportDialog from "./CertificateImportDialog.vue";
import DnsAccountDialog from "./DnsAccountDialog.vue";
import DnsAccountInspection from "./DnsAccountInspection.vue";
import AcmeAccountDialog from "./AcmeAccountDialog.vue";
import CertificateRequestDialog from "./CertificateRequestDialog.vue";
const sourceDialog = ref(false),
  editingSource = ref<WebResource>();
const siteDialog = ref(false),
  editingSite = ref<WebResource>();
const clusterDetailDialog = ref(false),
  detailClusterId = ref<number>(),
  detailClusterTab = ref("topology");
function openCluster(row: WebResource, tab = "topology") {
  detailClusterId.value = row.id;
  detailClusterTab.value = tab;
  clusterDetailDialog.value = true;
}
function handleClusterAction(
  cluster: WebResource,
  action: string,
  params: Record<string, any> = {}
) {
  clusterDetailDialog.value = false;
  launch(cluster, action, "clusters", params);
}
const props = defineProps<{ section?: string; embedded?: boolean }>();
const route = useRoute();
const router = useRouter();
const section = computed(
  () => props.section || String(route.meta.webSection || "sources")
);
const subsection = ref("");
const submenus = computed(() =>
  section.value === "sites"
    ? ["sites", "upstreams"]
    : section.value === "configurations"
      ? ["native-configs", "configurations", "config-groups"]
      : section.value === "certificates"
        ? [
            "certificates",
            "dns-accounts",
            "acme-accounts",
            "certificate-bindings",
            "renewal-policies"
          ]
        : [section.value]
);
const kind = computed(() =>
  submenus.value.includes(subsection.value) ? subsection.value : section.value
);
const routeInstanceId = computed(
  () => Number(route.query.instanceId) || undefined
);
const filteredInstances = computed(() => {
  const groupIds = new Set(filters.groupId ? [filters.groupId] : []);
  if (filters.groupId) {
    let changed = true;
    while (changed) {
      changed = false;
      for (const g of groups.value)
        if (groupIds.has(g.parentId) && !groupIds.has(g.id)) {
          groupIds.add(g.id);
          changed = true;
        }
    }
  }
  return (options.instances || []).filter(
    i =>
      (!filters.clusterId || i.clusterId === filters.clusterId) &&
      (!filters.groupId || groupIds.has(i.groupId)) &&
      (!filters.environment || i.environment === filters.environment) &&
      (!filters.status || i.status === filters.status) &&
      (!filters.name ||
        i.name.toLowerCase().includes(filters.name.toLowerCase()))
  );
});
const canWrite = computed(() => canPlatform("web:write"));
const canExecute = computed(() => canPlatform("web:execute"));
const canEditKind = computed(
  () =>
    canWrite.value &&
    (!["sources", "acme-accounts", "dns-accounts"].includes(kind.value) ||
      platformAccess.isAdmin)
);
const statusOptions = computed(() =>
  kind.value === "audit"
    ? [
        "queued",
        "preflight",
        "running",
        "verifying",
        "succeeded",
        "failed",
        "rolled-back",
        "unknown"
      ]
    : [
        "healthy",
        "unknown",
        "drifted",
        "unavailable",
        "issued",
        "draft",
        "configured",
        "published",
        "partial",
        "deployed"
      ]
);
const logScopeExpanded = ref(false);
const logScopeCount = computed(
  () =>
    [
      filters.name,
      filters.groupId,
      filters.clusterId,
      filters.status,
      filters.environment
    ].filter(Boolean).length
);
const filters = reactive({
  name: "",
  groupId: undefined as number | undefined,
  clusterId: route.query.clusterId
    ? Number(route.query.clusterId)
    : (undefined as number | undefined),
  configGroupId: undefined as number | undefined,
  environment: "",
  status: "",
  page: 1,
  size: 20
});
watch(
  () => filters.clusterId,
  () => {
    filters.configGroupId = undefined;
  },
  { flush: "sync" }
);
const rows = ref<any[]>([]),
  total = ref(0),
  loading = ref(false),
  error = ref(""),
  optionError = ref(""),
  groups = ref<any[]>([]),
  options = reactive<Record<string, any[]>>({}),
  capabilities = reactive<Record<number, WebCapability[]>>({}),
  capabilitiesLoading = reactive<Record<number, boolean>>({}),
  capabilitiesErrors = reactive<Record<number, string>>({});
type ResourceAction = {
  key: string;
  label: string;
  run?: () => unknown;
  disabled?: boolean;
  divided?: boolean;
  danger?: boolean;
  reason?: string;
};
function rowActions(row: WebResource): ResourceAction[] {
  const actions: ResourceAction[] = [];
  const add = (key: string, label: string, run: () => unknown) =>
    actions.push({ key, label, run });
  switch (kind.value) {
    case "sources":
      add("discover", "发现实例", () => discover(row));
      if (options.instances?.some(i => i.sourceId === row.id))
        add("config", "配置文件", () => openSourceConfig(row));
      if (canExecute.value && platformAccess.isAdmin)
        add("provision", "部署实例", () => launch(row, "provision", "sources"));
      break;
    case "clusters":
      add("members", "成员管理", () => openMembers(row));
      if (canWrite.value && canExecute.value && platformAccess.isAdmin)
        add("deployment", "新建部署", () => newDeployment(row));
      if (canExecute.value) {
        if (capabilitiesLoading[row.id]) {
          actions.push({
            key: "loading",
            label: "加载中...",
            disabled: true,
            divided: true
          });
        } else if (capabilitiesErrors[row.id]) {
          actions.push({
            key: "error",
            label: "读取失败，请重试",
            reason: capabilitiesErrors[row.id],
            disabled: true,
            divided: true
          });
        } else {
          (capabilities[row.id] || []).forEach((a, index) =>
            actions.push({
              key: `capability-${a.action}`,
              label: a.label,
              disabled: !a.enabled,
              reason: a.reason,
              divided: index === 0,
              run: () =>
                a.action === "configure-vip"
                  ? openCluster(row, "vip")
                  : launch(row, a.action, "clusters")
            })
          );
        }
      }
      break;
    case "sites":
    case "upstreams":
      if (canWrite.value) add("render", "生成配置草稿", () => render(row));
      add("configurations", "配置管理", () =>
        jump("configurations", row.clusterId)
      );
      if (kind.value === "sites")
        add("logs", "查看日志", () =>
          router.push({
            path: "/web/logs",
            query: { clusterId: row.clusterId, siteId: row.id }
          })
        );
      break;
    case "configurations":
      if (canWrite.value) add("validate", "校验语法", () => validateDraft(row));
      if (canExecute.value)
        add("publish", "发布", () =>
          launchCluster(row, "publish", { configurationId: row.id })
        );
      add("history", "版本历史", () => history(row.clusterId));
      break;
    case "certificates":
      add("history", "证书版本", () => certificateHistory(row));
      if (canWrite.value)
        add("import", "导入证书", () => {
          importFor.value = row;
          importDialog.value = true;
        });
      add("export", "导出证书", () => exportCertificate(row, false));
      if (canWrite.value && platformAccess.isAdmin)
        add("export-key", "导出私钥", () => exportCertificate(row, true));
      if (canExecute.value && row.spec?.owner === "platform") {
        add("issue", "签发证书", () => launch(row, "issue", "certificates"));
        if (row.spec?.currentVersionId)
          add("renew", "立即续期", () => launch(row, "renew", "certificates"));
      }
      break;
    case "certificate-bindings":
      if (canExecute.value)
        add("deploy", "部署证书", () =>
          launchCluster(row, "deploy-certificate", { bindingId: row.id })
        );
      break;
    case "dns-accounts":
      if (row.spec?.provider === "cloudflare") {
        if (canEditKind.value)
          add("check", "检查连接", () => inspectDNS(row, "check"));
        if (platformAccess.isAdmin)
          add("zones", "域名区域", () => inspectDNS(row, "zones"));
      }
      break;
  }
  if (
    canEditKind.value &&
    !["audit", "certificates", "acme-accounts", "dns-accounts"].includes(
      kind.value
    )
  )
    actions.push({
      key: "delete",
      label: "删除",
      danger: true,
      divided: actions.length > 0,
      run: () => removeResource(row)
    });
  return actions;
}
let generation = 0;
async function reload() {
  const gen = ++generation;
  if (["logs", "terminal", "native-configs"].includes(kind.value)) {
    loading.value = false;
    return;
  }
  loading.value = true;
  error.value = "";
  try {
    const value =
      kind.value === "audit"
        ? await webRequest("/audit?" + webQuery(filters))
        : await webList(kind.value, filters);
    if (gen === generation) {
      const lastPage = Math.max(1, Math.ceil(value.total / filters.size));
      if (filters.page > lastPage) {
        filters.page = lastPage;
        await reload();
        return;
      }
      rows.value = value.list;
      total.value = value.total;
    }
  } catch (e) {
    if (gen === generation) {
      rows.value = [];
      total.value = 0;
      error.value = (e as Error).message;
    }
  } finally {
    if (gen === generation) loading.value = false;
  }
}
const setupGuide = ref(false);
const workerOnline = ref<boolean | null>(null);
const encryptionReady = ref<boolean | null>(null);
let optionGeneration = 0;
async function loadOptions() {
  const gen = ++optionGeneration;
  const failures: string[] = [];
  optionError.value = "";
  const scopeKinds: Record<string, string[]> = {
    sources: ["clusters", "instances"],
    clusters: [
      "clusters",
      "instances",
      "configurations",
      "certificate-bindings"
    ],
    sites: [
      "clusters",
      "instances",
      "sites",
      "certificates",
      "certificate-bindings",
      "upstreams"
    ],
    configurations: [
      "clusters",
      "instances",
      "sites",
      "configurations",
      "config-groups"
    ],
    certificates: [
      "clusters",
      "instances",
      "certificates",
      "certificate-bindings",
      "acme-accounts",
      "dns-accounts"
    ],
    logs: ["clusters", "instances", "sites"]
  };
  const kinds = (scopeKinds[section.value] || ["clusters", "instances"]).filter(
    k =>
      platformAccess.isAdmin || !["acme-accounts", "dns-accounts"].includes(k)
  );
  const requests = [
    ...kinds.map(k => ({
      key: k,
      label: names[k] || tr("实例"),
      run: () => webOptions(k),
      apply: (v: any) => (options[k] = v || [])
    })),
    {
      key: "groups",
      label: tr("业务分组"),
      run: () => webRequest("/groups"),
      apply: (v: any) => (groups.value = v || [])
    },
    {
      key: "worker",
      label: "Web Worker",
      run: () => webRequest("/worker-status"),
      apply: (v: any) => {
        workerOnline.value = v.online;
        encryptionReady.value = v.encryptionReady;
      }
    },
    ...(canWrite.value &&
    platformAccess.isAdmin &&
    ["sources", "clusters"].includes(section.value)
      ? [
          {
            key: "targets",
            label: tr("管理连接"),
            run: () => webRequest("/targets"),
            apply: (v: any) => {
              options.hosts = v.hosts || [];
              options.connections = v.connections || [];
            }
          }
        ]
      : [])
  ];
  if (!platformAccess.isAdmin) {
    options.hosts = [];
    options.connections = [];
    options["acme-accounts"] = [];
    options["dns-accounts"] = [];
  }
  await Promise.allSettled(
    requests.map(async request => {
      try {
        const value = await request.run();
        if (gen === optionGeneration) request.apply(value);
      } catch (e) {
        if (gen !== optionGeneration) return;
        if (request.key === "groups") groups.value = [];
        else if (request.key === "worker") {
          workerOnline.value = null;
          encryptionReady.value = null;
        } else if (request.key === "targets") {
          options.hosts = [];
          options.connections = [];
        } else options[request.key] = [];
        failures.push(
          `${tr(request.label)}: ${(e as Error).message || tr("请求失败")}`
        );
        optionError.value = failures.join("；");
      }
    })
  );
}
async function refreshPage() {
  await Promise.all([reload(), loadOptions()]);
  nativeRefreshKey.value++;
}
function connectionView(row: WebResource) {
  return (
    row.connection ||
    options.connections?.find(c => c.id === row.spec?.connectionId)
  );
}
function reset() {
  filters.page = 1;
  reload();
}
function changePageSize(size: number) {
  if (size === filters.size) return;
  filters.size = size;
  reset();
}
function changePage(page: number) {
  if (page === filters.page) return;
  filters.page = page;
  reload();
}
function groupName(id: number) {
  return groups.value.find(g => g.id === id)?.name || "—";
}
const configGroupOptions = computed(() =>
  (options["config-groups"] || []).filter(
    g => !filters.clusterId || g.clusterId === filters.clusterId
  )
);
function configGroupName(id: number) {
  return (options["config-groups"] || []).find(g => g.id === id)?.name || "";
}
function tagType(s: string): "success" | "danger" | "warning" | "info" {
  return ["healthy", "succeeded", "issued"].includes(s)
    ? "success"
    : ["failed", "unavailable"].includes(s)
      ? "danger"
      : ["drifted", "degraded", "unknown"].includes(s)
        ? "warning"
        : "info";
}
function vipRowStatus(row: WebResource) {
  const state = row.observed?.vip;
  const age = Date.now() - Date.parse(state?.checkedAt || "");
  return !Number.isFinite(age) || age > 90000
    ? "unknown"
    : state.status || "unknown";
}
function expiryType(days: number): "success" | "danger" | "warning" | "info" {
  return days < 15 ? "danger" : days < 30 ? "warning" : "success";
}
async function guarded(fn: () => Promise<void>) {
  try {
    await fn();
  } catch (e) {
    if (e !== "cancel" && e !== "close")
      ElMessage.error((e as Error).message || String(e));
  }
}
const editing = ref(false),
  editKind = ref("sources"),
  saving = ref(false),
  form = ref<any>({ spec: {} }),
  dnsKey = ref(""),
  dnsSecret = ref(""),
  importInstance = ref<number>();
function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value));
}
const nativeRefreshKey = ref(0);
const nativeEditor = ref<InstanceType<typeof WebNativeConfigurations>>();
async function confirmTabChange() {
  return nativeEditor.value ? nativeEditor.value.confirmLeave() : true;
}
async function nativeSaved(draft: WebResource, publish: boolean) {
  await loadOptions();
  if (publish)
    await launchCluster(draft, "publish", { configurationId: draft.id });
}
function openSourceConfig(source: WebResource) {
  const instance = options.instances?.find(i => i.sourceId === source.id);
  if (instance)
    router.push({
      path: "/web/configurations",
      query: {
        instanceId: instance.id,
        clusterId: instance.clusterId || undefined
      }
    });
}
const configFileKeys = new WeakMap<object, number>();
let configFileSequence = 0;
function configFileKey(file: object) {
  if (!configFileKeys.has(file)) configFileKeys.set(file, ++configFileSequence);
  return configFileKeys.get(file)!;
}
const fileSearch = ref("");
const provisionAfterSave = ref(false);
function edit(row?: WebResource, override?: string) {
  const targetKind = override || kind.value;
  if (targetKind === "sources" && !override) {
    editingSource.value = row;
    sourceDialog.value = true;
    return;
  }
  if (targetKind === "clusters") {
    editingCluster.value = row;
    clusterDialog.value = true;
    return;
  }
  if (targetKind === "sites") {
    editingSite.value = row;
    siteDialog.value = true;
    return;
  }
  if (targetKind === "dns-accounts") {
    editingDNSAccount.value = row;
    dnsAccountDialog.value = true;
    return;
  }
  if (targetKind === "acme-accounts") {
    editingAcmeAccount.value = row;
    acmeAccountDialog.value = true;
    return;
  }
  provisionAfterSave.value = false;
  fileSearch.value = "";
  configBaseline.value = {};
  validationResult.value = null;
  onlyChangedFiles.value = false;
  fileGroupFilter.value = "";
  editKind.value = override || kind.value;
  form.value = row
    ? clone(row)
    : {
        name: "",
        groupId: filters.groupId,
        clusterId: filters.clusterId || undefined,
        engine: "nginx",
        environment: filters.environment,
        revision: 0,
        spec: defaults(editKind.value)
      };
  dnsKey.value = "";
  dnsSecret.value = "";
  if (editKind.value === "configurations") {
    importInstance.value = (
      options.instances?.find(i => i.clusterId === form.value.clusterId) || {}
    ).id;
  }
  editing.value = true;
}
function newDeployment(cluster: WebResource) {
  edit(undefined, "sources");
  const name = `c${cluster.id}-${Math.floor(Date.now() / 1000)}`;
  const root = `/etc/autoops-web/${name}/current`;
  Object.assign(form.value, {
    name: `${cluster.name} · Nginx`,
    groupId: cluster.groupId,
    clusterId: 0,
    environment: cluster.environment
  });
  Object.assign(form.value.spec, {
    configPath: `${root}/nginx.conf`,
    prefix: root,
    managedPaths: [root],
    service: `autoops-web-${name}`,
    container: `autoops-web-${name}`
  });
  provisionAfterSave.value = true;
}
const visibleFields = computed(() =>
  (fieldSets[editKind.value] || []).filter(f => {
    const spec = form.value.spec;
    if (editKind.value === "sources") {
      if (["container", "image"].includes(f.key))
        return spec.runtime === "docker";
      if (["service", "packageVersion"].includes(f.key))
        return spec.runtime === "host";
    }
    if (editKind.value === "sites" && f.key === "certificateId")
      return spec.tls;
    if (editKind.value === "certificates") {
      if (f.key === "dnsAccountId") return spec.challenge === "dns-01";
      if (["instanceIds", "challengeRoot"].includes(f.key))
        return spec.challenge === "http-01";
    }
    if (f.key === "bindingIds") return spec.autoDeploy;
    return true;
  })
);
function headerText(value: Record<string, string> = {}) {
  return Object.entries(value)
    .map(([k, v]) => k + ": " + v)
    .join("\n");
}
function parsePairs(value: string) {
  return Object.fromEntries(
    value
      .split("\n")
      .filter(x => x.includes(":"))
      .map(x => {
        const i = x.indexOf(":");
        return [x.slice(0, i).trim(), x.slice(i + 1).trim()];
      })
  );
}
function setHeaders(location: any, value: string) {
  location.headers = parsePairs(value);
}
function setVariables(id: number, value: string) {
  form.value.spec.variables ||= {};
  form.value.spec.variables[id] = parsePairs(value);
}
const deploymentNodes = computed(() =>
  Object.entries(selected.value?.observed?.nodes || {}).map(([id, value]) => ({
    instanceId: Number(id),
    ...(value as object)
  }))
);
async function removeResource(row: WebResource) {
  await guarded(async () => {
    await ElMessageBox.confirm(
      tr("删除管理记录将保留远端服务和文件，确定继续？"),
      tr("删除"),
      { type: "warning" }
    );
    await webRequest("/" + kind.value + "/" + row.id, "DELETE");
    await Promise.all([reload(), loadOptions()]);
  });
}
function jump(section: string, cid: number) {
  detailsVisible.value = false;
  clusterDetailDialog.value = false;
  router.push({ path: "/web/" + section, query: { clusterId: cid } });
}
function jumpAudit(cid: number) {
  router.push({
    path: "/monitor/dblog",
    query: { tab: "web-audit", clusterId: cid }
  });
}
async function cancelOperation() {
  await guarded(async () => {
    await webRequest(
      "/operations/" + operation.value.operation.id + "/cancel",
      "POST"
    );
  });
}
async function reconcileOperation() {
  await guarded(async () => {
    const op = operation.value.operation;
    const target = await webRequest<WebResource>(
      "/" + op.targetType + "/" + op.targetId
    );
    operationDialog.value = false;
    launch(target, "reconcile", op.targetType, { operationId: op.id });
  });
}
function needsCluster(k: string) {
  return [
    "sites",
    "upstreams",
    "configurations",
    "config-groups",
    "certificate-bindings"
  ].includes(k);
}
function selectCluster(id: number) {
  const c = options.clusters?.find(c => c.id === id);
  if (c) {
    form.value.groupId = c.groupId;
    form.value.environment = c.environment;
  }
  const instance = options.instances?.find(i => i.clusterId === id);
  if (editKind.value === "configurations") {
    importInstance.value = instance?.id;
    form.value.spec.configGroupId = undefined;
    form.value.spec.instanceIds = [];
  }
  // Site and upstream defaults live under /etc/nginx; adopted instances keep
  // their configuration under their own prefix, so follow the member layout.
  const configPath = instance?.spec?.configPath;
  if (
    ["sites", "upstreams"].includes(editKind.value) &&
    configPath &&
    String(form.value.spec.path || "").startsWith("/etc/nginx/")
  ) {
    const root = configPath.replace(/\/[^/]+$/, "");
    form.value.spec.path = `${root}/conf.d/${String(form.value.spec.path).split("/").pop()}`;
  }
}
function fieldOptions(f: Field) {
  if (f.values)
    return f.values.map(v => ({ value: v, label: tr(statusNames[v] || v) }));
  let all = options[f.options || ""] || [];
  if (
    f.options === "certificate-bindings" &&
    editKind.value === "renewal-policies"
  )
    all = all.filter(
      b => b.spec?.certificateId === form.value.spec.certificateId
    );
  if (f.options === "connections")
    all = all.filter(c => c.assetId === form.value.spec.hostId);
  return all.map(v => ({
    value: v.id,
    label: v.hostName || v.name || v.alias || String(v.id)
  }));
}
function setArray(f: Field, v: string) {
  const arr = v
    .split(f.type === "lines" ? /\n/ : /[,，]/)
    .map(x => x.trim())
    .filter(Boolean);
  form.value.spec[f.key] = f.type === "numbers" ? arr.map(Number) : arr;
}
async function save() {
  await guarded(async () => {
    if (!form.value.name || !form.value.groupId)
      throw Error(tr("请填写名称和业务分组"));
    if (needsCluster(editKind.value) && !form.value.clusterId)
      throw Error(tr("请选择集群"));
    saving.value = true;
    try {
      const payload = clone(form.value);
      if (editKind.value === "configurations") {
        payload.spec.configGroupId = Number(payload.spec.configGroupId) || 0;
        if (
          (payload.spec.files || []).some(
            (f: any) => !String(f.path || "").trim()
          )
        )
          throw Error(tr("配置文件路径不能为空"));
      }
      if (editKind.value === "dns-accounts" && dnsKey.value)
        payload.secretInput = JSON.stringify(
          payload.spec.provider === "cloudflare"
            ? { token: dnsKey.value }
            : { key: dnsKey.value, secret: dnsSecret.value }
        );
      const saved = await webRequest<WebResource>(
        "/" + editKind.value + (payload.id ? "/" + payload.id : ""),
        payload.id ? "PUT" : "POST",
        payload
      );
      editing.value = false;
      ElMessage.success(tr("保存成功"));
      await Promise.all([reload(), loadOptions()]);
      if (provisionAfterSave.value) launch(saved, "provision", "sources");
    } finally {
      saving.value = false;
    }
  });
}
async function discover(row: WebResource) {
  await guarded(async () => {
    loading.value = true;
    try {
      await webRequest("/sources/" + row.id + "/discovery", "POST");
      ElMessage.success(tr("实例发现完成"));
    } finally {
      loading.value = false;
      await refreshPage();
    }
  });
}
async function readConfig() {
  await guarded(async () => {
    if (!importInstance.value) throw Error(tr("请选择要读取配置的实例"));
    const currentForm = form.value;
    const instanceId = importInstance.value;
    const r = await webInstanceConfig(instanceId);
    if (
      !editing.value ||
      form.value !== currentForm ||
      importInstance.value !== instanceId
    )
      return;
    const managed = r.managedPaths || [];
    const files = (r.files || []).filter(
      f =>
        !managed.length ||
        managed.some(m => f.path === m || f.path.startsWith(m + "/"))
    );
    if (!files.length) throw Error(tr("未发现可读取配置或缺少配置查看权限"));
    configBaseline.value = Object.fromEntries(
      files.map(f => [f.path, f.content])
    );
    form.value.spec.files = clone(files);
    form.value.spec.baselines = { [r.instanceId]: r.digest };
    form.value.spec.instanceIds = [r.instanceId];
    validationResult.value = null;
  });
}
const configBaseline = ref<Record<string, string>>({});
const validating = ref(false);
const validationResult = ref<{
  valid: boolean;
  lines: string[];
} | null>(null);
const onlyChangedFiles = ref(false);
const fileGroupFilter = ref("");
const fileGroupNames = computed(() => {
  const names = new Set<string>();
  for (const g of options["config-groups"] || [])
    if (!form.value.clusterId || g.clusterId === form.value.clusterId)
      names.add(g.name);
  for (const f of form.value.spec?.files || []) if (f.group) names.add(f.group);
  return [...names].filter(Boolean);
});
const formConfigGroups = computed(() =>
  (options["config-groups"] || []).filter(
    g => !form.value.clusterId || g.clusterId === form.value.clusterId
  )
);
function isChanged(f: any) {
  if (!f.path) return true;
  if (f.deleted) return true;
  if (!Object.keys(configBaseline.value).length) return true;
  if (!(f.path in configBaseline.value)) return true;
  return configBaseline.value[f.path] !== f.content;
}
const changedFiles = computed<WebConfigFile[]>(() =>
  (form.value.spec?.files || []).filter((f: any) => f.path && isChanged(f))
);
const changedFilePaths = computed(
  () => new Set(changedFiles.value.map(f => f.path))
);
const visibleConfigFiles = computed(() =>
  (form.value.spec?.files || []).filter((f: any) => {
    if (onlyChangedFiles.value && !isChanged(f)) return false;
    if (fileGroupFilter.value && (f.group || "") !== fileGroupFilter.value)
      return false;
    if (
      fileSearch.value &&
      !(f.path || "").includes(fileSearch.value) &&
      !(f.content || "").includes(fileSearch.value)
    )
      return false;
    return true;
  })
);
const pendingFileDeletes = new WeakSet<object>();
async function confirmFileDeletion(target: any, removeNew = false) {
  if (pendingFileDeletes.has(target)) return;
  const currentForm = form.value;
  const saved = Object.hasOwn(configBaseline.value, target.path);
  pendingFileDeletes.add(target);
  try {
    await ElMessageBox.confirm(
      tr(
        saved || !removeNew
          ? "确认将文件「{value0}」标记为删除？只有完整配置校验通过并确认发布后，远端文件才会被删除。"
          : "确认移除尚未保存的新文件「{value0}」？此文件的编辑内容将丢失。",
        { value0: target.path || tr("未命名") }
      ),
      tr("删除文件"),
      { type: "warning", closeOnClickModal: false }
    );
    if (!editing.value || form.value !== currentForm) return;
    const index = form.value.spec.files.indexOf(target);
    if (index < 0) return;
    if (saved || !removeNew) target.deleted = true;
    else form.value.spec.files.splice(index, 1);
  } catch (e) {
    if (e !== "cancel" && e !== "close") ElMessage.error((e as Error).message);
  } finally {
    pendingFileDeletes.delete(target);
  }
}
function removeFile(target: any) {
  return confirmFileDeletion(target, true);
}
function markFileDeletion(target: any, deleted: boolean) {
  if (!deleted) target.deleted = false;
  else return confirmFileDeletion(target);
}

watch(
  () => JSON.stringify(form.value.spec),
  () => {
    validationResult.value = null;
  }
);
const syntaxDialog = ref(false),
  syntaxResults = ref<WebConfigValidation[]>([]),
  syntaxTarget = ref("");
async function validateDraft(row: WebResource) {
  await guarded(async () => {
    syntaxTarget.value = row.name;
    syntaxResults.value = [];
    syntaxDialog.value = true;
    syntaxResults.value = await webValidateClusterConfig(
      row.clusterId,
      row.spec?.files || [],
      {
        instanceIds: row.spec?.instanceIds,
        baselines: row.spec?.baselines,
        variables: row.spec?.variables
      }
    );
  });
}
async function validateFormConfig() {
  await guarded(async () => {
    if (!changedFiles.value.length) throw Error(tr("没有需要校验的配置变更"));
    validating.value = true;
    const snapshot = JSON.stringify(form.value);
    try {
      const rows = await webValidateClusterConfig(
        form.value.clusterId,
        form.value.spec.files || [],
        {
          instanceIds: form.value.spec.instanceIds,
          baselines: form.value.spec.baselines,
          variables: form.value.spec.variables
        }
      );
      if (!editing.value || JSON.stringify(form.value) !== snapshot) return;
      const failed = rows.filter(r => !r.valid);
      validationResult.value = {
        valid: failed.length === 0,
        lines: rows.map(
          r => `${r.name}: ${r.valid ? tr("通过") : r.message || tr("未通过")}`
        )
      };
      if (failed.length)
        ElMessage.error(tr("Nginx 语法校验未通过，配置不会被更新") as string);
      else ElMessage.success(tr("Nginx 语法校验通过") as string);
    } finally {
      validating.value = false;
    }
  });
}
function downloadText(name: string, text: string) {
  const url = URL.createObjectURL(
    new Blob([text], { type: "application/x-pem-file" })
  );
  const link = document.createElement("a");
  link.href = url;
  link.download = name;
  link.click();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}
async function exportCertificate(
  row: WebResource,
  includeKey: boolean,
  versionId = 0
) {
  await guarded(async () => {
    if (includeKey)
      await ElMessageBox.confirm(
        tr("私钥属于敏感材料，导出会写入审计。确认继续？"),
        tr("导出私钥"),
        { type: "warning" }
      );
    const data = await webExportCertificate(row.id, versionId, includeKey);
    downloadText(
      `${row.name}-v${data.versionId}-${includeKey ? "private-key" : "certificate"}.pem`,
      includeKey ? data.privateKey : data.certificate
    );
  });
}
const certificateFileReads = { cert: 0, key: 0 };
async function loadCertificateFile(event: Event, target: "cert" | "key") {
  const input = event.target as HTMLInputElement;
  const file = input.files?.[0];
  input.value = "";
  if (!file) return;
  if (file.size > 1024 * 1024) {
    ElMessage.error(tr("单个证书或私钥文件不能超过 1 MiB"));
    return;
  }
  const request = ++certificateFileReads[target];
  try {
    const value = await file.text();
    if (!importDialog.value || request !== certificateFileReads[target]) return;
    if (target === "cert") certificatePEM.value = value;
    else privatePEM.value = value;
  } catch (e) {
    ElMessage.error((e as Error).message);
  }
}
async function render(row: WebResource) {
  await guarded(async () => {
    const spec = await webRequest("/" + kind.value + "/" + row.id + "/render");
    edit(undefined, "configurations");
    form.value = {
      name: row.name,
      groupId: row.groupId,
      clusterId: row.clusterId,
      engine: row.engine,
      environment: row.environment,
      revision: 0,
      spec
    };
  });
}
const detailsVisible = ref(false),
  selected = ref<WebResource>();
const linkedDraft = ref<WebResource>();
let linkedDraftGeneration = 0;
async function focusLinkedDraft() {
  const gen = ++linkedDraftGeneration;
  const id = Number(route.query.configurationId);
  linkedDraft.value = undefined;
  if (
    section.value !== "configurations" ||
    !Number.isSafeInteger(id) ||
    id <= 0
  )
    return;
  try {
    const draft = await webRequest<WebResource>("/configurations/" + id);
    if (gen !== linkedDraftGeneration || section.value !== "configurations")
      return;
    if (!(await confirmTabChange()) || gen !== linkedDraftGeneration) return;
    subsection.value = "configurations";
    filters.name = draft.name;
    filters.clusterId = draft.clusterId;
    filters.groupId = draft.groupId;
    filters.environment = draft.environment;
    filters.status = "";
    filters.configGroupId = undefined;
    filters.page = 1;
    linkedDraft.value = draft;
    await reload();
    if (gen !== linkedDraftGeneration || kind.value !== "configurations")
      return;
    selected.value = draft;
    detailsVisible.value = true;
  } catch (e) {
    if (gen === linkedDraftGeneration) error.value = (e as Error).message;
  }
}
function clearLinkedDraft() {
  if (filters.name === linkedDraft.value?.name) filters.name = "";
  linkedDraft.value = undefined;
  const query = { ...route.query };
  delete query.configurationId;
  router.replace({ query });
  reset();
}
let detailGeneration = 0;
async function detail(row: any) {
  const gen = ++detailGeneration;
  const requestedKind = kind.value;
  if (kind.value === "clusters") {
    openCluster(row);
    return;
  }
  if (kind.value === "audit") {
    await showOperation(row.id);
    return;
  }
  await guarded(async () => {
    const value = await webRequest<WebResource>(
      "/" + requestedKind + "/" + row.id
    );
    if (gen !== detailGeneration || requestedKind !== kind.value) return;
    selected.value = value;
    detailsVisible.value = true;
  });
}
const membersDialog = ref(false),
  memberCluster = ref<WebResource>(),
  members = ref<WebResource[]>([]),
  addInstance = ref<number>();
async function openMembers(row: WebResource) {
  await guarded(async () => {
    memberCluster.value = row;
    members.value = [];
    const value = await webRequest<WebResource[]>(
      "/clusters/" + row.id + "/members"
    );
    if (memberCluster.value?.id !== row.id) return;
    members.value = value;
    membersDialog.value = true;
  });
}
async function bindMember(remove: boolean, iid?: number) {
  if (!iid || !memberCluster.value) return;
  await guarded(async () => {
    await webRequest(
      "/clusters/" + memberCluster.value!.id + "/members",
      "POST",
      { instanceId: iid, remove }
    );
    members.value = await webRequest(
      "/clusters/" + memberCluster.value!.id + "/members"
    );
    addInstance.value = undefined;
    await loadOptions();
  });
}
const planPreview = ref<any[]>([]);
const planDialog = ref(false),
  planTarget = ref<WebResource>(),
  planType = ref("clusters"),
  planAction = ref(""),
  parameters = ref<Record<string, any>>({}),
  plan = ref<WebPlan | null>(null),
  planning = ref(false),
  executing = ref(false);
let planGeneration = 0;
function clearPlan() {
  planPreview.value = [];
  plan.value = null;
  planGeneration++;
}
function actionLabel(action: string) {
  const found = Object.values(capabilities)
    .flat()
    .find(c => c.action === action);
  return (
    found?.label ||
    (
      {
        provision: "部署实例",
        issue: "签发证书",
        publish: "发布配置",
        rollback: "回滚配置",
        "deploy-certificate": "部署证书",
        "configure-vip": "配置虚拟 IP",
        "verify-vip": "核验虚拟 IP",
        "switch-vip": "切换虚拟 IP"
      } as Record<string, string>
    )[action] ||
    action
  );
}
async function loadCapabilities(row: WebResource) {
  if (capabilitiesLoading[row.id]) return;
  capabilitiesLoading[row.id] = true;
  delete capabilities[row.id];
  delete capabilitiesErrors[row.id];
  try {
    capabilities[row.id] = await webRequest(
      "/clusters/" + row.id + "/capabilities"
    );
  } catch (e) {
    capabilitiesErrors[row.id] = (e as Error).message;
    ElMessage.error(capabilitiesErrors[row.id]);
  } finally {
    capabilitiesLoading[row.id] = false;
  }
}
function launch(
  row: WebResource,
  action: string,
  targetType: string,
  p: Record<string, any> = {}
) {
  clearPlan();
  planTarget.value = row;
  planAction.value = action;
  planType.value = targetType;
  parameters.value = { ...p };
  planDialog.value = true;
  if (
    ![
      "publish",
      "validate",
      "rollback",
      "deploy-certificate",
      "reconcile"
    ].includes(action) ||
    Object.keys(p).length
  )
    preflight();
}
async function launchCluster(
  row: WebResource,
  action: string,
  p: Record<string, any>
) {
  await guarded(async () => {
    const cluster = await webRequest<WebResource>("/clusters/" + row.clusterId);
    launch(cluster, action, "clusters", p);
  });
}
async function preflight() {
  const gen = ++planGeneration;
  plan.value = null;
  planning.value = true;
  await guarded(async () => {
    const value = await webRequest<WebPlan>("/operation-plans", "POST", {
      targetType: planType.value,
      targetId: planTarget.value?.id,
      action: planAction.value,
      parameters: parameters.value
    });
    if (gen === planGeneration && planDialog.value) {
      plan.value = value;
      try {
        const preview = await webRequest(
          "/operation-plans/" + value.id + "/preview"
        );
        if (gen === planGeneration) planPreview.value = preview;
      } catch {
        planPreview.value = [];
      }
    }
  });
  if (gen === planGeneration) planning.value = false;
}
async function execute() {
  const frozen = plan.value;
  if (!frozen?.executable || executing.value) return;
  executing.value = true;
  await guarded(async () => {
    await ElMessageBox.confirm(
      tr("确认对 {value0} 执行 {value1}？", {
        value0: planTarget.value?.name,
        value1: tr(actionLabel(planAction.value))
      }),
      tr("确认执行"),
      { type: "warning" }
    );
    if (!planDialog.value || plan.value?.id !== frozen.id) return;
    const op = await webRequest("/operations", "POST", {
      planId: frozen.id,
      idempotencyKey: frozen.idempotencyKey,
      confirmed: true
    });
    planDialog.value = false;
    await showOperation(op.id);
  });
  executing.value = false;
}
const operationDialog = ref(false),
  operation = ref<any>();
let poll: ReturnType<typeof setTimeout> | undefined;
let operationGeneration = 0;
async function showOperation(id: number) {
  const gen = ++operationGeneration;
  operation.value = undefined;
  operationDialog.value = true;
  async function refresh() {
    if (!operationDialog.value) return;
    try {
      const value = await webRequest("/operations/" + id);
      if (gen !== operationGeneration || !operationDialog.value) return;
      operation.value = value;
      if (
        ["queued", "preflight", "running", "verifying"].includes(
          operation.value.operation.status
        )
      )
        poll = setTimeout(refresh, 2000);
      else {
        await reload();
        await loadOptions();
        nativeRefreshKey.value++;
      }
    } catch (e) {
      if (gen === operationGeneration) error.value = (e as Error).message;
    }
  }
  if (poll) clearTimeout(poll);
  await refresh();
}
const dnsAccountDialog = ref(false),
  editingDNSAccount = ref<WebResource>();
const dnsInspectionDialog = ref(false),
  inspectedDNSAccount = ref<WebResource>(),
  dnsInspectionAction = ref<"check" | "zones">("check");
const acmeAccountDialog = ref(false),
  editingAcmeAccount = ref<WebResource>();
const requestCertificateDialog = ref(false);
function dnsProviderName(provider: string) {
  return provider === "cloudflare"
    ? "Cloudflare"
    : tr(
        provider === "alidns"
          ? "阿里云 DNS"
          : provider === "tencentcloud"
            ? "腾讯云 DNSPod"
            : "未知"
      );
}
function inspectDNS(account: WebResource, action: "check" | "zones") {
  inspectedDNSAccount.value = account;
  dnsInspectionAction.value = action;
  dnsInspectionDialog.value = true;
}
async function refreshAccounts() {
  await Promise.all([reload(), loadOptions()]);
}
async function dnsAccountSaved(account: WebResource) {
  const currentKind = kind.value;
  await refreshAccounts();
  if (
    section.value === "certificates" &&
    kind.value === currentKind &&
    account.spec.provider === "cloudflare"
  )
    inspectDNS(account, "check");
}
function configureDNS() {
  requestCertificateDialog.value = false;
  subsection.value = "dns-accounts";
  reset();
  edit(undefined, "dns-accounts");
}
async function certificateRequestSaved(
  certificate: WebResource,
  issue: boolean
) {
  const currentKind = kind.value;
  await refreshAccounts();
  if (issue && section.value === "certificates" && kind.value === currentKind)
    launch(certificate, "issue", "certificates");
}
const createImportDialog = ref(false);
async function certificateImported() {
  await Promise.all([reload(), loadOptions()]);
}
const importDialog = ref(false),
  importFor = ref<WebResource>(),
  certificatePEM = ref(""),
  privatePEM = ref("");
async function importCertificate() {
  await guarded(async () => {
    saving.value = true;
    try {
      await webRequest(
        "/certificates/" + importFor.value?.id + "/import",
        "POST",
        { certificate: certificatePEM.value, privateKey: privatePEM.value }
      );
      importDialog.value = false;
      await Promise.all([reload(), loadOptions()]);
      ElMessage.success(tr("证书校验通过并已导入"));
    } finally {
      saving.value = false;
    }
  });
}
watch(importDialog, v => {
  certificateFileReads.cert++;
  certificateFileReads.key++;
  if (!v) {
    certificatePEM.value = "";
    privatePEM.value = "";
  }
});
const revisionDiff = ref<any[]>([]);
const historyCertificate = ref<WebResource>();
const historyDialog = ref(false),
  historyLoading = ref(false),
  revisions = ref<any[]>([]),
  revisionView = ref<any>();
let historyGeneration = 0;
function showRevision(row: any) {
  historyGeneration++;
  revisionDiff.value = [];
  revisionView.value = row;
}
async function compareRevision(row: any) {
  const gen = ++historyGeneration;
  await guarded(async () => {
    const previous = revisions.value.find(r => r.id < row.id);
    const result = await webRequest(
      "/config-revisions/" + row.id + "/diff?compareTo=" + (previous?.id || 0)
    );
    if (gen !== historyGeneration || !historyDialog.value) return;
    revisionDiff.value = result.files;
    revisionView.value = null;
  });
}
async function loadHistory(query: string, certificate?: WebResource) {
  const gen = ++historyGeneration;
  historyCertificate.value = certificate;
  revisions.value = [];
  revisionView.value = null;
  revisionDiff.value = [];
  historyLoading.value = true;
  historyDialog.value = true;
  try {
    const result = await webRequest<any[]>("/config-revisions?" + query);
    if (gen !== historyGeneration || !historyDialog.value) return;
    revisions.value = result;
    if (certificate) revisionView.value = result[0];
  } catch (e) {
    if (gen === historyGeneration && historyDialog.value)
      ElMessage.error((e as Error).message);
  } finally {
    if (gen === historyGeneration) historyLoading.value = false;
  }
}
function certificateHistory(row: WebResource) {
  return loadHistory("certificateId=" + row.id, row);
}
function history(cid: number) {
  return loadHistory("clusterId=" + cid);
}
watch(historyDialog, open => {
  if (!open) {
    historyGeneration++;
    historyLoading.value = false;
  }
});
watch(detailsVisible, open => {
  if (!open) detailGeneration++;
});
watch(kind, () => {
  detailsVisible.value = false;
  historyDialog.value = false;
});

async function rollback(row: any) {
  await guarded(async () => {
    const c = await webRequest<WebResource>("/clusters/" + row.clusterId);
    historyDialog.value = false;
    launch(c, "rollback", "clusters", { revisionId: row.id });
  });
}
watch(
  () => route.query.clusterId,
  v => {
    filters.clusterId = v ? Number(v) : undefined;
    reset();
  }
);
watch(
  section,
  () => {
    subsection.value =
      section.value === "configurations" ? "native-configs" : section.value;
    filters.page = 1;
    reload();
    loadOptions();
  },
  { immediate: true }
);
watch(() => [section.value, route.query.configurationId], focusLinkedDraft, {
  immediate: true
});
watch(operationDialog, v => {
  if (!v) {
    operationGeneration++;
    if (poll) clearTimeout(poll);
  }
});
onUnmounted(() => {
  linkedDraftGeneration++;
  generation++;
  optionGeneration++;
  detailGeneration++;
  historyGeneration++;
  operationGeneration++;
  planGeneration++;
  if (poll) clearTimeout(poll);
});
</script>
<style scoped>
.web-management {
  min-width: 0;
  width: 100%;
  box-sizing: border-box;
  padding: 20px;
  color: var(--el-text-color-primary);
}
.log-scope-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}
.log-scope-actions .el-button {
  margin: 0;
}
.log-scope-actions .el-tag {
  margin-left: 6px;
}
.logs-page .web-heading {
  margin-bottom: 12px;
}
.logs-page .web-heading p {
  display: none;
}
.logs-page .web-heading h2 {
  margin: 0;
  font-size: 20px;
}
.web-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 24px;
}
.web-heading h2 {
  font-size: 24px;
  margin: 0 0 8px;
}
.web-heading p {
  margin: 0;
  color: var(--el-text-color-secondary);
}
.toolbar {
  display: flex;
  gap: 12px;
  align-items: center;
  flex-wrap: wrap;
  margin: 16px 0;
}
.toolbar > .el-input,
.toolbar > .el-select {
  width: 190px;
}
.toolbar > .el-button {
  margin-left: 0;
}
.resource-table {
  width: 100%;
  margin: 16px 0;
}
.resource-identity {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.resource-identity span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.resource-actions {
  flex-wrap: nowrap;
  gap: 8px;
}
.resource-actions > * {
  flex-shrink: 0;
}
.more-icon {
  margin-left: 4px;
}
.resource-action-danger {
  color: var(--el-color-danger);
}
.resource-pagination {
  display: flex;
  justify-content: center;
  padding: 20px 0 4px;
}
.resource-pagination :deep(.el-pagination) {
  justify-content: center;
  flex-wrap: wrap;
  gap: 10px 8px;
}
.resource-pagination :deep(.el-pagination > *) {
  margin: 0;
}
.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 0 20px;
}
.form-grid .el-select {
  width: 100%;
}
.form-grid small {
  display: block;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
}
.location-row {
  display: flex;
  gap: 10px;
  margin: 12px 0;
  align-items: center;
}
.location-row .el-input {
  flex: 1;
  min-width: 100px;
}
.location-row .el-select {
  width: 150px;
}
.config-file {
  margin: 16px 0;
}
.validation-line {
  font-family: ui-monospace, SFMono-Regular, monospace;
  white-space: pre-wrap;
  font-size: 12px;
}
.muted {
  color: var(--el-text-color-secondary);
}
.mono {
  font-family: ui-monospace, SFMono-Regular, monospace;
  font-size: 12px;
  overflow-wrap: anywhere;
}
.file-picker {
  display: inline-flex;
  align-items: center;
  height: 32px;
  padding: 0 15px;
  border: 1px solid var(--el-border-color);
  border-radius: var(--el-border-radius-base);
  cursor: pointer;
  color: var(--el-color-primary);
  font-size: 13px;
}
.file-picker input {
  display: none;
}
.file-group {
  width: 170px;
}
.code-input :deep(textarea),
.detail-code {
  font-family: ui-monospace, SFMono-Regular, monospace;
  font-size: 13px;
}
.detail-code {
  overflow: auto;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  padding: 16px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
}
:deep(.el-dialog) {
  max-width: 95vw;
}
@media (max-width: 700px) {
  .web-management {
    padding: 12px;
  }
  .form-grid {
    grid-template-columns: 1fr;
  }
  .location-row {
    flex-wrap: wrap;
  }
  .toolbar > .el-input,
  .toolbar > .el-select {
    width: 100%;
  }
}
</style>

<style>
.web-resource-dialog.el-dialog {
  display: flex;
  flex-direction: column;
  max-height: calc(100dvh - 48px);
  max-width: calc(100vw - 32px);
  margin-top: 24px;
  margin-bottom: 24px;
}
.web-resource-dialog .el-dialog__body {
  min-height: 0;
  overflow: auto;
  overscroll-behavior: contain;
}
.web-resource-dialog .el-dialog__footer {
  flex-shrink: 0;
  display: flex;
  justify-content: flex-end;
  flex-wrap: wrap;
  gap: 8px;
}
.web-resource-dialog .el-dialog__footer .el-button {
  margin-left: 0;
}
</style>
