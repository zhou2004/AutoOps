<template>
  <div class="asset-page docker-page">
    <header class="asset-page-header">
      <div>
        <span class="asset-eyebrow">CONTAINER OPERATIONS</span>
        <h1>{{ $tr("Docker 管理") }}</h1>
        <p>{{ $tr("通过资产已有的 SSH 连接档案管理 Docker，支持容器、镜像、数据卷与 Compose。") }}</p>
      </div>
      <div class="asset-actions">
        <el-select
          v-model="selectedAsset"
          :placeholder='$tr("选择主机或物理机")'
          filterable
          style="width: 240px"
          @change="loadConnections"
        >
          <el-option-group
            v-for="group in assetGroups"
            :key="group.label"
            :label="$tr(group.label)"
          >
            <el-option
              v-for="item in group.options"
              :key="item.value"
              :label="$tr(item.label)"
              :value="item.value"
            />
          </el-option-group>
        </el-select>
        <el-select
          v-model="connectionId"
          :placeholder='$tr("SSH 连接档案")'
          style="width: 190px"
        >
          <el-option
            v-for="item in connections"
            :key="item.id"
            :label="`${item.name} · ${item.address}`"
            :value="item.id"
          />
        </el-select>
        <el-button
          type="primary"
          :loading="loading"
          :disabled="!connectionId"
          @click="refreshAll"
          >{{ $tr("连接并刷新") }}</el-button
        >
      </div>
    </header>

    <el-alert
      v-if="error"
      :title="$tr(error)"
      type="error"
      :closable="false"
      show-icon
    />

    <section v-if="info" class="docker-summary">
      <article>
        <span>Docker</span><strong>{{ info.ServerVersion || "-" }}</strong>
      </article>
      <article>
        <span>{{ $tr("运行中") }}</span><strong>{{ info.ContainersRunning ?? 0 }}</strong>
      </article>
      <article>
        <span>{{ $tr("已停止") }}</span><strong>{{ info.ContainersStopped ?? 0 }}</strong>
      </article>
      <article>
        <span>{{ $tr("镜像") }}</span><strong>{{ info.Images ?? 0 }}</strong>
      </article>
      <article>
        <span>{{ $tr("节点") }}</span><strong>{{ info.Name || "-" }}</strong>
      </article>
      <article>
        <span>{{ $tr("系统") }}</span><strong>{{ info.OperatingSystem || "-" }}</strong>
      </article>
    </section>

    <el-card class="asset-table-card" shadow="never">
      <el-tabs v-model="tab" @tab-change="loadTab">
        <!-- 容器 -->
        <el-tab-pane :label='$tr("容器")' name="containers">
          <div class="table-toolbar">
            <el-input
              v-model="keyword"
              clearable
              :placeholder='$tr("搜索名称、镜像、端口或 ID")'
              style="width: 280px"
            />
            <el-radio-group v-model="statusFilter">
              <el-radio-button value="all">{{ $tr("全部") }}</el-radio-button>
              <el-radio-button value="running">{{ $tr("运行中") }}</el-radio-button>
              <el-radio-button value="exited">{{ $tr("已停止") }}</el-radio-button>
              <el-radio-button value="paused">{{ $tr("已暂停") }}</el-radio-button>
            </el-radio-group>
            <div style="flex: 1"></div>
            <el-button
              type="primary"
              :disabled="!connectionId"
              @click="openCreate"
              >{{ $tr("创建容器") }}</el-button
            >
          </div>
          <el-table :data="filteredContainers" v-loading="loading" stripe>
            <el-table-column :label='$tr("容器")' min-width="200"
              ><template #default="{ row }"
                ><div class="asset-primary">{{ row.Names }}</div>
                <div class="asset-secondary">{{ shortId(row.ID) }}</div></template
              ></el-table-column
            >
            <el-table-column
              prop="Image"
              :label='$tr("镜像")'
              min-width="200"
              show-overflow-tooltip
            />
            <el-table-column :label='$tr("状态")' width="120"
              ><template #default="{ row }"
                ><el-tag :type="stateType(row.State)">{{
                  row.State
                }}</el-tag></template
              ></el-table-column
            >
            <el-table-column
              prop="Ports"
              :label='$tr("端口")'
              min-width="190"
              show-overflow-tooltip
            />
            <el-table-column prop="Status" :label='$tr("运行信息")' min-width="160" />
            <el-table-column :label='$tr("操作")' width="470" fixed="right"
              ><template #default="{ row }"
                ><div class="row-actions">
                  <el-button link type="primary" @click="showLogs(row)"
                    >{{ $tr("日志") }}</el-button
                  ><el-button
                    link
                    type="primary"
                    :disabled="row.State !== 'running'"
                    @click="openTerminal(row)"
                    >{{ $tr("终端") }}</el-button
                  ><el-button
                    link
                    type="primary"
                    :disabled="row.State !== 'running'"
                    @click="openFiles(row)"
                    >{{ $tr("文件") }}</el-button
                  ><el-button
                    v-if="row.State !== 'running'"
                    link
                    type="success"
                    @click="operate(row, 'start')"
                    >{{ $tr("启动") }}</el-button
                  ><el-button
                    v-if="row.State === 'running'"
                    link
                    type="warning"
                    @click="operate(row, 'restart')"
                    >{{ $tr("重启") }}</el-button
                  ><el-button
                    v-if="row.State === 'running'"
                    link
                    type="warning"
                    @click="operate(row, 'pause')"
                    >{{ $tr("暂停") }}</el-button
                  ><el-button
                    v-if="row.State === 'paused'"
                    link
                    type="success"
                    @click="operate(row, 'unpause')"
                    >{{ $tr("恢复") }}</el-button
                  ><el-button
                    v-if="row.State === 'running'"
                    link
                    type="danger"
                    @click="operate(row, 'stop')"
                    >{{ $tr("停止") }}</el-button
                  ><el-button
                    v-if="row.State !== 'running'"
                    link
                    type="danger"
                    @click="removeContainer(row)"
                    >{{ $tr("删除") }}</el-button
                  >
                </div></template
              ></el-table-column
            >
          </el-table>
        </el-tab-pane>

        <!-- 镜像 -->
        <el-tab-pane :label='$tr("镜像")' name="images">
          <div class="table-toolbar">
            <el-button type="primary" @click="openRegistry"
              >{{ $tr("镜像仓库") }}</el-button
            ><el-upload
              class="inline-upload"
              action="#"
              :http-request="importImage"
              :show-file-list="false"
              :disabled="!connectionId"
            >
              <el-button type="info" :disabled="!connectionId">{{ $tr("导入镜像") }}</el-button>
            </el-upload>
            <el-button
              type="warning"
              :disabled="!selectedImages.length"
              @click="exportImages()"
              >{{ $tr("导出选中") }}</el-button
            >
            <el-button
              type="danger"
              :disabled="!selectedImages.length"
              @click="removeImages()"
              >{{ $tr("删除选中") }}</el-button
            >
          </div>
          <el-table
            :data="images"
            v-loading="loading"
            stripe
            @selection-change="onImageSelection"
          >
            <el-table-column type="selection" width="46" />
            <el-table-column prop="Repository" :label='$tr("仓库")' min-width="230" />
            <el-table-column prop="Tag" :label='$tr("标签")' width="140" />
            <el-table-column :label='$tr("镜像 ID")' width="130"
              ><template #default="{ row }">{{
                shortId(row.ID)
              }}</template></el-table-column
            >
            <el-table-column prop="Size" :label='$tr("大小")' width="120" />
            <el-table-column prop="CreatedSince" :label='$tr("创建时间")' width="150" />
            <el-table-column :label='$tr("操作")' width="380" fixed="right"
              ><template #default="{ row }"
                ><div class="row-actions">
                  <el-button link type="primary" @click="openImageOp('tag', row)"
                    >{{ $tr("打标签") }}</el-button
                  ><el-button
                    link
                    type="primary"
                    @click="openImageOp('rename', row)"
                    >{{ $tr("重命名") }}</el-button
                  ><el-button
                    link
                    type="primary"
                    @click="openTransfer(row, false)"
                    >{{ $tr("复制") }}</el-button
                  ><el-button
                    link
                    type="warning"
                    @click="openTransfer(row, true)"
                    >{{ $tr("迁移") }}</el-button
                  ><el-button link type="success" @click="exportImages([row])"
                    >{{ $tr("导出") }}</el-button
                  ><el-button link type="danger" @click="removeImages([row])"
                    >{{ $tr("删除") }}</el-button
                  >
                </div></template
              ></el-table-column
            >
          </el-table>
        </el-tab-pane>

        <!-- 数据卷 -->
        <el-tab-pane :label='$tr("数据卷")' name="volumes">
          <div class="table-toolbar">
            <el-input
              v-model="volumeKeyword"
              clearable
              :placeholder='$tr("搜索数据卷名称")'
              style="width: 260px"
            />
            <div style="flex: 1"></div>
            <el-button type="success" :disabled="!connectionId" @click="openVolumeCreate"
              >{{ $tr("创建数据卷") }}</el-button
            >
          </div>
          <el-table :data="filteredVolumes" v-loading="loading" stripe>
            <el-table-column prop="Name" :label='$tr("名称")' min-width="240" />
            <el-table-column prop="Driver" :label='$tr("驱动")' width="130" />
            <el-table-column prop="Scope" :label='$tr("范围")' width="120" />
            <el-table-column
              prop="Mountpoint"
              :label='$tr("挂载位置")'
              min-width="320"
              show-overflow-tooltip
            />
            <el-table-column :label='$tr("操作")' width="230" fixed="right"
              ><template #default="{ row }"
                ><div class="row-actions">
                  <el-button link type="primary" @click="openVolumeDetail(row)"
                    >{{ $tr("详情") }}</el-button
                  ><el-button
                    link
                    type="primary"
                    @click="openVolumeContainers(row)"
                    >{{ $tr("映射容器") }}</el-button
                  ><el-button link type="danger" @click="removeVolume(row)"
                    >{{ $tr("删除") }}</el-button
                  >
                </div></template
              ></el-table-column
            >
          </el-table>
        </el-tab-pane>

        <!-- Compose -->
        <el-tab-pane label="Compose" name="compose">
          <div class="table-toolbar">
            <el-input
              v-model="composeKeyword"
              clearable
              :placeholder='$tr("搜索项目名称")'
              style="width: 260px"
            />
            <div style="flex: 1"></div>
            <el-button
              type="success"
              :disabled="!connectionId"
              @click="openComposeEditor()"
              >{{ $tr("新建项目") }}</el-button
            >
          </div>
          <el-table :data="filteredCompose" v-loading="composeLoading" stripe>
            <el-table-column prop="Name" :label='$tr("项目")' min-width="200" />
            <el-table-column prop="Status" :label='$tr("状态")' width="200" />
            <el-table-column
              prop="ConfigFiles"
              :label='$tr("配置文件")'
              min-width="300"
              show-overflow-tooltip
            />
            <el-table-column :label='$tr("操作")' width="340" fixed="right"
              ><template #default="{ row }"
                ><div class="row-actions">
                  <el-button link type="primary" @click="openComposeEditor(row.Name)"
                    >{{ $tr("编辑 YML") }}</el-button
                  ><el-button
                    link
                    type="success"
                    @click="composeAction(row.Name, 'up')"
                    >{{ $tr("启动") }}</el-button
                  ><el-button
                    link
                    type="warning"
                    @click="composeAction(row.Name, 'restart')"
                    >{{ $tr("重启") }}</el-button
                  ><el-button
                    link
                    type="warning"
                    @click="composeAction(row.Name, 'down')"
                    >{{ $tr("停止") }}</el-button
                  ><el-button link type="danger" @click="removeCompose(row.Name)"
                    >{{ $tr("删除") }}</el-button
                  >
                </div></template
              ></el-table-column
            >
          </el-table>
        </el-tab-pane>

        <!-- 网络 -->
        <el-tab-pane :label='$tr("网络")' name="networks">
          <div class="table-toolbar">
            <el-input
              v-model="networkKeyword"
              clearable
              :placeholder='$tr("搜索网络名称")'
              style="width: 260px"
            />
            <div style="flex: 1"></div>
            <el-button type="success" :disabled="!connectionId" @click="openNetworkCreate"
              >{{ $tr("创建网络") }}</el-button
            >
          </div>
          <el-table :data="filteredNetworks" v-loading="loading" stripe>
            <el-table-column prop="Name" :label='$tr("名称")' min-width="220" />
            <el-table-column prop="Driver" :label='$tr("驱动")' width="130" />
            <el-table-column prop="Scope" :label='$tr("范围")' width="120" />
            <el-table-column :label='$tr("网络 ID")' width="130"
              ><template #default="{ row }">{{
                shortId(row.ID)
              }}</template></el-table-column
            >
            <el-table-column :label='$tr("操作")' width="300" fixed="right"
              ><template #default="{ row }"
                ><div class="row-actions">
                  <el-button link type="primary" @click="openNetworkDetail(row)"
                    >{{ $tr("详情") }}</el-button
                  ><el-button
                    link
                    type="primary"
                    @click="openNetworkConnect(row)"
                    >{{ $tr("连接容器") }}</el-button
                  ><el-button
                    link
                    type="warning"
                    @click="openNetworkDisconnect(row)"
                    >{{ $tr("断开容器") }}</el-button
                  ><el-button link type="danger" @click="removeNetwork(row)"
                    >{{ $tr("删除") }}</el-button
                  >
                </div></template
              ></el-table-column
            >
          </el-table>
        </el-tab-pane>

        <!-- 审计 -->
        <el-tab-pane :label='$tr("操作审计")' name="audits">
          <el-table :data="audits" v-loading="loading" stripe>
            <el-table-column prop="createdAt" :label='$tr("时间")' min-width="190" />
            <el-table-column prop="userId" :label='$tr("用户")' width="90" />
            <el-table-column prop="action" :label='$tr("权限动作")' width="150" />
            <el-table-column
              prop="target"
              :label='$tr("目标")'
              min-width="220"
              show-overflow-tooltip
            />
            <el-table-column :label='$tr("结果")' width="100"
              ><template #default="{ row }"
                ><el-tag
                  :type="row.status === 'success' ? 'success' : 'danger'"
                  >{{ row.status }}</el-tag
                ></template
              ></el-table-column
            >
            <el-table-column prop="durationMs" :label='$tr("耗时(ms)")' width="110" />
            <el-table-column prop="clientIp" :label='$tr("客户端 IP")' width="150" />
            <el-table-column
              prop="message"
              :label='$tr("错误")'
              min-width="240"
              show-overflow-tooltip
            />
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- 日志抽屉 -->
    <el-drawer
      v-model="logsVisible"
      :title="`${currentContainer?.Names || ''} · 日志`"
      size="60%"
      ><div class="logs-toolbar">
        <el-input-number v-model="logTail" :min="1" :max="2000" /><el-button
          type="primary"
          @click="loadLogs"
          >{{ $tr("刷新日志") }}</el-button
        >
      </div>
      <pre class="docker-logs">{{ logs || "暂无日志" }}</pre>
    </el-drawer>

    <!-- 创建容器 -->
    <el-dialog v-model="createVisible" :title='$tr("创建容器")' width="min(720px,94vw)">
      <el-alert
        :title='$tr("仅允许使用目标主机已有镜像，不开放 privileged、宿主机网络和任意目录挂载。")'
        type="info"
        :closable="false"
        show-icon
      />
      <el-form :model="createForm" label-width="100px" class="create-form">
        <el-form-item :label='$tr("容器名称")'
          ><el-input v-model="createForm.name"
        /></el-form-item>
        <el-form-item :label='$tr("镜像")'
          ><el-select
            v-model="createForm.image"
            filterable
            allow-create
            style="width: 100%"
            ><el-option
              v-for="item in imageChoices"
              :key="item"
              :label="$tr(item)"
              :value="item" /></el-select
        ></el-form-item>
        <el-form-item :label='$tr("重启策略")'
          ><el-select v-model="createForm.restart"
            ><el-option :label='$tr("不自动重启")' value="no" /><el-option
              :label='$tr("始终重启")'
              value="always" /><el-option
              :label='$tr("异常时重启")'
              value="on-failure" /><el-option
              :label='$tr("手动停止前重启")'
              value="unless-stopped" /></el-select
        ></el-form-item>
        <el-form-item :label='$tr("端口映射")'
          ><el-input
            v-model="createForm.ports"
            type="textarea"
            :rows="2"
            :placeholder='$tr("每行一个，如 8080:80/tcp")'
        /></el-form-item>
        <el-form-item :label='$tr("数据卷")'
          ><el-input
            v-model="createForm.volumes"
            type="textarea"
            :rows="2"
            :placeholder='$tr("每行一个，如 卷名:/容器路径 或 卷名:/容器路径:ro")'
        /></el-form-item>
        <el-form-item :label='$tr("环境变量")'
          ><el-input
            v-model="createForm.environment"
            type="textarea"
            :rows="3"
            :placeholder='$tr("每行一个 KEY=VALUE")'
        /></el-form-item>
        <el-form-item :label='$tr("启动参数")'
          ><el-input
            v-model="createForm.command"
            :placeholder='$tr("按空格分隔；复杂参数建议在镜像中定义")'
        /></el-form-item>
      </el-form>
      <template #footer
        ><el-button @click="createVisible = false">{{ $tr("取消") }}</el-button
        ><el-button type="primary" :loading="creating" @click="createContainer"
          >{{ $tr("创建并启动") }}</el-button
        ></template
      >
    </el-dialog>

    <!-- 文件管理 -->
    <el-dialog
      v-model="filesVisible"
      :title="`文件管理 - ${filesContainer?.Names || ''}`"
      width="900px"
      destroy-on-close
    >
      <docker-file-manager
        v-if="filesVisible"
        :asset-type="selection.type"
        :asset-id="selection.id"
        :container="filesContainer?.ID"
        :connection-id="connectionId"
      />
    </el-dialog>

    <!-- 镜像仓库 -->
    <el-dialog v-model="registryVisible" :title='$tr("镜像仓库")' width="min(860px,96vw)">
      <div class="table-toolbar">
        <el-input
          v-model="registryKeyword"
          clearable
          :placeholder='$tr("输入关键词筛选，如 nginx")'
          style="width: 300px"
          @keyup.enter="loadRegistryPage(1)"
        />
        <el-button type="primary" :loading="registryLoading" @click="loadRegistryPage(1)"
          >{{ $tr("查询") }}</el-button
        >
      </div>
      <el-table
        :data="registryList"
        v-loading="registryLoading"
        stripe
        height="460"
      >
        <el-table-column
          prop="name"
          :label='$tr("镜像名称")'
          min-width="230"
          show-overflow-tooltip
        />
        <el-table-column prop="category" :label='$tr("分类")' width="110" />
        <el-table-column
          prop="description"
          :label='$tr("描述")'
          min-width="220"
          show-overflow-tooltip
        />
        <el-table-column prop="starCount" label="Stars" width="90" />
        <el-table-column :label='$tr("操作")' width="110" fixed="right"
          ><template #default="{ row }"
            ><el-button link type="success" @click="openDownload(row)"
              >{{ $tr("下载") }}</el-button
            ></template
          ></el-table-column
        >
      </el-table>
      <div class="pager-row">
        <el-pagination
          background
          layout="total, prev, pager, next, sizes"
          :total="registryTotal"
          :page-size="registryPageSize"
          :current-page="registryPage"
          :page-sizes="[10, 20, 50]"
          @current-change="loadRegistryPage"
          @size-change="onRegistrySizeChange"
        />
      </div>
    </el-dialog>

    <!-- 下载镜像 -->
    <el-dialog v-model="downloadVisible" :title='$tr("下载镜像")' width="min(620px,94vw)">
      <el-form label-width="100px">
        <el-form-item :label='$tr("镜像站前缀")'
          ><el-select v-model="downloadRegistry" style="width: 100%">
            <el-option :label='$tr("Docker Hub（不添加前缀）")' value="" />
            <el-option
              v-for="r in registries"
              :key="r.value"
              :label="`${r.label} (${r.value}/)`"
              :value="r.value + '/'"
            />
          </el-select></el-form-item
        >
        <el-form-item :label='$tr("镜像名称")'
          ><el-input
            v-model="downloadImage"
            :placeholder='$tr("如 nginx:latest")'
            @keyup.enter="submitDownload"
        /></el-form-item>
      </el-form>
      <el-alert
        :title='$tr("下载即拉取到当前选中的 Docker 资产；建议通过镜像站前缀拉取（Docker Hub 直连可能不可用）。")'
        type="info"
        :closable="false"
        show-icon
      />
      <template #footer
        ><el-button @click="downloadVisible = false">{{ $tr("取消") }}</el-button
        ><el-button type="primary" :loading="downloading" @click="submitDownload"
          >{{ $tr("下载") }}</el-button
        ></template
      >
    </el-dialog>

    <!-- 打标签 / 重命名 / 迁移 -->
    <el-dialog
      v-model="imageOpVisible"
      :title="$tr(imageOpTitle)"
      width="min(620px,94vw)"
    >
      <el-form label-width="100px">
        <el-form-item :label='$tr("源镜像")'
          ><el-input v-model="imageOpForm.source" disabled /></el-form-item
        >
        <el-form-item :label='$tr("目标")'
          ><el-input
            v-model="imageOpForm.target"
            :placeholder="$tr(imageOpPlaceholder)"
            @keyup.enter="submitImageOp"
        /></el-form-item>
      </el-form>
      <template #footer
        ><el-button @click="imageOpVisible = false">{{ $tr("取消") }}</el-button
        ><el-button type="primary" :loading="imageOpLoading" @click="submitImageOp"
          >{{ $tr("确定") }}</el-button
        ></template
      >
    </el-dialog>

    <!-- 跨机迁移 / 复制镜像 -->
    <el-dialog
      v-model="transferVisible"
      :title="$tr(transferForm.migrate ? '迁移镜像到其他机器' : '复制镜像到其他机器')"
      width="min(620px,94vw)"
    >
      <el-alert
        :title="`将镜像「${transferForm.image}」导出并导入到目标机器的 Docker。${transferForm.migrate ? '完成后将删除源机器上的镜像。' : '源机器镜像将保留。'}`"
        type="info"
        :closable="false"
        show-icon
      />
      <el-form label-width="100px" class="create-form">
        <el-form-item :label='$tr("源镜像")'
          ><el-input v-model="transferForm.image" disabled /></el-form-item
        >
        <el-form-item :label='$tr("目标机器")'
          ><el-select
            v-model="transferForm.targetAsset"
            filterable
            style="width: 100%"
            :placeholder='$tr("选择平台中的主机或物理机")'
          >
            <el-option-group
              v-for="group in assetGroups"
              :key="group.label"
              :label="$tr(group.label)"
            >
              <el-option
                v-for="item in group.options"
                :key="item.value"
                :label="$tr(item.label)"
                :value="item.value"
              />
            </el-option-group>
          </el-select></el-form-item
        >
      </el-form>
      <template #footer
        ><el-button @click="transferVisible = false">{{ $tr("取消") }}</el-button
        ><el-button
          type="primary"
          :loading="transferLoading"
          @click="submitTransfer"
          >{{ transferForm.migrate ? "迁移" : $tr("复制") }}</el-button
        ></template
      >
    </el-dialog>

    <!-- 创建数据卷 -->
    <el-dialog v-model="volumeCreateVisible" :title='$tr("创建数据卷")' width="min(480px,94vw)">
      <el-form label-width="100px">
        <el-form-item :label='$tr("名称")'
          ><el-input v-model="volumeForm.name" :placeholder='$tr("数据卷名称")'
        /></el-form-item>
        <el-form-item :label='$tr("驱动")'
          ><el-input v-model="volumeForm.driver" :placeholder='$tr("默认 local")'
        /></el-form-item>
      </el-form>
      <template #footer
        ><el-button @click="volumeCreateVisible = false">{{ $tr("取消") }}</el-button
        ><el-button type="primary" :loading="volumeCreating" @click="createVolume"
          >{{ $tr("创建") }}</el-button
        ></template
      >
    </el-dialog>

    <!-- 数据卷详情 -->
    <el-dialog
      v-model="volumeDetailVisible"
      :title="`数据卷 - ${volumeDetail?.Name || ''}`"
      width="min(720px,94vw)"
    >
      <el-descriptions v-if="volumeDetail" :column="2" border>
        <el-descriptions-item :label='$tr("名称")'>{{ volumeDetail.Name }}</el-descriptions-item>
        <el-descriptions-item :label='$tr("驱动")'>{{ volumeDetail.Driver }}</el-descriptions-item>
        <el-descriptions-item :label='$tr("范围")'>{{ volumeDetail.Scope }}</el-descriptions-item>
        <el-descriptions-item :label='$tr("创建时间")'>{{ volumeDetail.CreatedAt }}</el-descriptions-item>
        <el-descriptions-item :label='$tr("挂载位置")' :span="2">{{ volumeDetail.Mountpoint }}</el-descriptions-item>
      </el-descriptions>
      <h4 class="sub-title">{{ $tr("使用该数据卷的容器") }}</h4>
      <el-table :data="volumeDetailContainers" stripe max-height="260">
        <el-table-column prop="Names" :label='$tr("名称")' min-width="200" />
        <el-table-column prop="Image" :label='$tr("镜像")' min-width="200" show-overflow-tooltip />
        <el-table-column prop="State" :label='$tr("状态")' width="120" />
      </el-table>
    </el-dialog>

    <!-- 创建网络 -->
    <el-dialog v-model="networkCreateVisible" :title='$tr("创建网络")' width="min(480px,94vw)">
      <el-form label-width="100px">
        <el-form-item :label='$tr("名称")'
          ><el-input v-model="networkForm.name" :placeholder='$tr("网络名称")'
        /></el-form-item>
        <el-form-item :label='$tr("驱动")'
          ><el-select v-model="networkForm.driver" style="width: 100%">
            <el-option :label='$tr("bridge（默认）")' value="bridge" />
            <el-option label="overlay" value="overlay" />
            <el-option label="host" value="host" />
            <el-option label="macvlan" value="macvlan" />
          </el-select></el-form-item
        >
      </el-form>
      <template #footer
        ><el-button @click="networkCreateVisible = false">{{ $tr("取消") }}</el-button
        ><el-button type="primary" :loading="networkCreating" @click="createNetwork"
          >{{ $tr("创建") }}</el-button
        ></template
      >
    </el-dialog>

    <!-- 网络详情 -->
    <el-dialog
      v-model="networkDetailVisible"
      :title="`网络 - ${networkDetail?.Name || ''}`"
      width="min(720px,94vw)"
    >
      <el-descriptions v-if="networkDetail" :column="2" border>
        <el-descriptions-item :label='$tr("名称")'>{{ networkDetail.Name }}</el-descriptions-item>
        <el-descriptions-item :label='$tr("驱动")'>{{ networkDetail.Driver }}</el-descriptions-item>
        <el-descriptions-item :label='$tr("范围")'>{{ networkDetail.Scope }}</el-descriptions-item>
        <el-descriptions-item :label='$tr("子网")'>{{ networkDetail.IPAM?.Config?.[0]?.Subnet || "-" }}</el-descriptions-item>
        <el-descriptions-item :label='$tr("网关")' :span="2">{{ networkDetail.IPAM?.Config?.[0]?.Gateway || "-" }}</el-descriptions-item>
      </el-descriptions>
      <h4 class="sub-title">{{ $tr("已连接的容器") }}</h4>
      <el-table :data="networkDetailContainers" stripe max-height="260">
        <el-table-column prop="Names" :label='$tr("名称")' min-width="200" />
        <el-table-column prop="Image" :label='$tr("镜像")' min-width="200" show-overflow-tooltip />
        <el-table-column prop="State" :label='$tr("状态")' width="120" />
      </el-table>
    </el-dialog>

    <!-- 连接 / 断开容器 -->
    <el-dialog
      v-model="networkLinkVisible"
      :title="networkLinkMode === 'connect' ? '连接容器到网络' : '将容器从网络断开'"
      width="min(520px,94vw)"
    >
      <el-form label-width="100px">
        <el-form-item :label='$tr("网络")'
          ><el-input :model-value="networkLinkForm.network" disabled /></el-form-item
        >
        <el-form-item :label='$tr("容器")'
          ><el-select
            v-model="networkLinkForm.container"
            filterable
            style="width: 100%"
            :placeholder='$tr("选择容器")'
          >
            <el-option
              v-for="c in runningContainers"
              :key="c.ID"
              :label="`${c.Names} (${shortId(c.ID)})`"
              :value="c.ID"
            />
          </el-select></el-form-item
        >
      </el-form>
      <template #footer
        ><el-button @click="networkLinkVisible = false">{{ $tr("取消") }}</el-button
        ><el-button
          type="primary"
          :loading="networkLinking"
          @click="submitNetworkLink"
          >{{ $tr("确定") }}</el-button
        ></template
      >
    </el-dialog>

    <!-- Compose 编辑 -->
    <el-dialog
      v-model="composeEditorVisible"
      :title="composeForm.project ? `编辑 Compose - ${composeForm.project}` : '新建 Compose 项目'"
      width="min(820px,96vw)"
      top="4vh"
    >
      <el-form label-width="100px">
        <el-form-item :label='$tr("项目名称")'
          ><el-input
            v-model="composeForm.project"
            :placeholder='$tr("项目名称（小写字母、数字、-、_）")'
        /></el-form-item>
        <el-form-item :label='$tr("YML 配置")'
          ><el-input
            v-model="composeForm.content"
            type="textarea"
            :rows="20"
            spellcheck="false"
            placeholder="version: '3'\nservices:\n  web:\n    image: nginx:latest\n    ports:\n      - 80:80"
        /></el-form-item>
      </el-form>
      <template #footer
        ><el-button @click="composeEditorVisible = false">{{ $tr("取消") }}</el-button
        ><el-button :loading="composeSaving" @click="saveCompose(false)"
          >{{ $tr("仅保存") }}</el-button
        ><el-button type="primary" :loading="composeSaving" @click="saveCompose(true)"
          >{{ $tr("保存并启动") }}</el-button
        ></template
      >
    </el-dialog>
  </div>
</template>

<script setup>
import { tr } from "@/locales";

import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { ElMessage, ElMessageBox } from "element-plus";
import api from "@/api/cmdb";
import DockerFileManager from "./DockerFileManager.vue";

const router = useRouter();

const loading = ref(false),
  error = ref(""),
  selectedAsset = ref(""),
  connectionId = ref(),
  connections = ref([]),
  assets = ref({ hosts: [], physical: [] }),
  info = ref(null),
  tab = ref("containers"),
  containers = ref([]),
  images = ref([]),
  volumes = ref([]),
  networks = ref([]),
  keyword = ref(""),
  statusFilter = ref("all");
const audits = ref([]);
const logsVisible = ref(false),
  currentContainer = ref(null),
  logs = ref(""),
  logTail = ref(200);
const createVisible = ref(false),
  creating = ref(false),
  createForm = ref({
    name: "",
    image: "",
    restart: "no",
    ports: "",
    volumes: "",
    environment: "",
    command: ""
  });
const filesVisible = ref(false),
  filesContainer = ref(null);
const selectedImages = ref([]);
const registryVisible = ref(false),
  registryKeyword = ref(""),
  registryList = ref([]),
  registryTotal = ref(0),
  registryPage = ref(1),
  registryPageSize = ref(10),
  registryLoading = ref(false);
const downloadVisible = ref(false),
  downloading = ref(false),
  downloadRegistry = ref("docker.1ms.run/"),
  downloadImage = ref("");
const imageOpVisible = ref(false),
  imageOpMode = ref("tag"),
  imageOpLoading = ref(false),
  imageOpForm = ref({ source: "", target: "" });
const volumeKeyword = ref("");
const volumeCreateVisible = ref(false),
  volumeCreating = ref(false),
  volumeForm = ref({ name: "", driver: "" });
const volumeDetailVisible = ref(false),
  volumeDetail = ref(null),
  volumeDetailContainers = ref([]);
const compose = ref([]),
  composeLoading = ref(false),
  composeKeyword = ref("");
const composeEditorVisible = ref(false),
  composeSaving = ref(false),
  composeForm = ref({ project: "", content: "" });
const transferVisible = ref(false),
  transferLoading = ref(false),
  transferForm = ref({
    image: "",
    targetAsset: "",
    migrate: false
  });
const networkKeyword = ref("");
const networkCreateVisible = ref(false),
  networkCreating = ref(false),
  networkForm = ref({ name: "", driver: "bridge" });
const networkDetailVisible = ref(false),
  networkDetail = ref(null),
  networkDetailContainers = ref([]);
const networkLinkVisible = ref(false),
  networkLinkMode = ref("connect"),
  networkLinking = ref(false),
  networkLinkForm = ref({ network: "", container: "" });

const registries = [
  { label: "渡渡鸟镜像站", value: "docker.1ms.run" },
  { label: "DaoCloud 镜像站", value: "docker.m.daocloud.io" },
  { label: "dockerproxy", value: "dockerproxy.com" },
  { label: "hub.rat.dev", value: "hub.rat.dev" }
];

function unpack(res) {
  if (res.data?.code !== 200) throw new Error(res.data?.message || tr("操作失败"));
  return res.data.data;
}
const selection = computed(() => {
  const [type, id] = String(selectedAsset.value).split(":");
  return { type, id: Number(id) };
});
const assetGroups = computed(() => [
  {
    label: "主机",
    options: assets.value.hosts.map(x => ({
      label: x.hostName,
      value: `host:${x.id}`
    }))
  },
  {
    label: "物理机",
    options: assets.value.physical.map(x => ({
      label: x.hostName || x.sn,
      value: `physical:${x.id}`
    }))
  }
]);
const filteredContainers = computed(() => {
  const q = keyword.value.trim().toLowerCase();
  return containers.value.filter(x => {
    if (statusFilter.value !== "all" && x.State !== statusFilter.value) return false;
    if (!q) return true;
    return [x.Names, x.Image, x.State, x.Status, x.Ports, x.ID].some(v =>
      String(v || "")
        .toLowerCase()
        .includes(q)
    );
  });
});
const filteredVolumes = computed(() => {
  const q = volumeKeyword.value.trim().toLowerCase();
  return q
    ? volumes.value.filter(x => String(x.Name || "").toLowerCase().includes(q))
    : volumes.value;
});
const filteredCompose = computed(() => {
  const q = composeKeyword.value.trim().toLowerCase();
  return q
    ? compose.value.filter(x => String(x.Name || "").toLowerCase().includes(q))
    : compose.value;
});
const filteredNetworks = computed(() => {
  const q = networkKeyword.value.trim().toLowerCase();
  return q
    ? networks.value.filter(x => String(x.Name || "").toLowerCase().includes(q))
    : networks.value;
});
const runningContainers = computed(() =>
  containers.value.filter(x => x.State === "running")
);
const imageChoices = computed(() =>
  images.value
    .map(x => `${x.Repository}:${x.Tag}`)
    .filter(x => !x.includes("<none>"))
);
const imageOpTitle = computed(
  () => ({ tag: "打标签", rename: "重命名" }[imageOpMode.value])
);
const imageOpPlaceholder = computed(() => "目标名称，如 newname:tag");

function shortId(id) {
  return String(id || "")
    .replace(/^sha256:/, "")
    .slice(0, 12);
}
function stateType(state) {
  return state === "running"
    ? "success"
    : state === "paused"
      ? "warning"
      : "info";
}
function imageRef(row) {
  if (row.Tag && row.Tag !== "<none>") return `${row.Repository}:${row.Tag}`;
  return row.ID || row.Repository;
}
function params() {
  return { connectionId: connectionId.value };
}

async function loadAssets() {
  const targets = unpack(await api.getDockerTargets());
  assets.value.hosts = targets.hosts || [];
  assets.value.physical = targets.physical || [];
  const physical =
    assets.value.physical.find(
      x => String(x.hostName || "").toLowerCase() === "lenovo"
    ) || assets.value.physical[0];
  const first = physical || assets.value.hosts[0];
  if (first) {
    selectedAsset.value = `${physical ? "physical" : "host"}:${first.id}`;
    await loadConnections();
  }
}
async function loadConnections() {
  info.value = null;
  connectionId.value = undefined;
  connections.value = [];
  if (!selection.value.id) return;
  try {
    const data = unpack(
      await api.getDockerConnections(selection.value.type, selection.value.id)
    );
    connections.value = (data.connections || []).filter(
      x => x.protocol === "ssh" && x.enabled && x.canDocker
    );
    connectionId.value = connections.value[0]?.id;
    if (connectionId.value) await refreshAll();
  } catch (e) {
    error.value = e.message;
  }
}
async function refreshAll() {
  if (!connectionId.value) return;
  loading.value = true;
  error.value = "";
  try {
    const [i, c, im, n, v] = await Promise.all([
      api.getDockerInfo(selection.value.type, selection.value.id, params()),
      api.getDockerContainers(selection.value.type, selection.value.id, params()),
      api.getDockerImages(selection.value.type, selection.value.id, params()),
      api.getDockerNetworks(selection.value.type, selection.value.id, params()),
      api.getDockerVolumes(selection.value.type, selection.value.id, params())
    ]);
    info.value = unpack(i);
    containers.value = unpack(c).list || [];
    images.value = unpack(im).list || [];
    networks.value = unpack(n).list || [];
    volumes.value = unpack(v).list || [];
  } catch (e) {
    error.value = e.message;
  } finally {
    loading.value = false;
  }
}
async function loadCompose() {
  if (!connectionId.value) return;
  composeLoading.value = true;
  try {
    compose.value =
      unpack(
        await api.getDockerCompose(selection.value.type, selection.value.id, params())
      )?.list || [];
  } catch (e) {
    ElMessage.error(e.message);
  } finally {
    composeLoading.value = false;
  }
}
async function loadAudits() {
  try {
    audits.value =
      unpack(
        await api.getDockerAudits(selection.value.type, selection.value.id, {
          ...params(),
          page: 1,
          size: 100
        })
      )?.list || [];
  } catch (e) {
    ElMessage.error(e.message);
  }
}
async function loadTab(name) {
  if (name === "audits" && connectionId.value) await loadAudits();
  else if (name === "compose" && connectionId.value) await loadCompose();
  else if (!info.value && connectionId.value) await refreshAll();
}

// ================= 容器操作 =================
async function confirmName(row, action) {
  if (action === "start") return "";
  const labels = {
    stop: "停止",
    restart: "重启",
    pause: "暂停",
    unpause: "恢复",
    remove: "删除"
  };
  const out = await ElMessageBox.prompt(
    `输入容器名称 ${row.Names} 确认${labels[action]}`,
    tr("操作确认"),
    {
      inputValidator: v => v === row.Names || "名称不匹配",
      type: action === "remove" ? "error" : "warning"
    }
  );
  return out.value;
}
async function operate(row, action) {
  try {
    const confirmNameValue = await confirmName(row, action);
    unpack(
      await api.operateDockerContainer(
        selection.value.type,
        selection.value.id,
        row.ID,
        {
          connectionId: connectionId.value,
          action,
          confirmName: confirmNameValue
        }
      )
    );
    ElMessage.success(tr("操作成功"));
    await refreshAll();
  } catch (e) {
    if (e !== "cancel" && e !== "close" && e?.message)
      ElMessage.error(e.message);
  }
}
async function removeContainer(row) {
  try {
    const value = await confirmName(row, "remove");
    unpack(
      await api.removeDockerContainer(
        selection.value.type,
        selection.value.id,
        row.ID,
        { connectionId: connectionId.value, confirmName: value }
      )
    );
    ElMessage.success(tr("容器已删除"));
    await refreshAll();
  } catch (e) {
    if (e?.message) ElMessage.error(e.message);
  }
}
function openTerminal(row) {
  router.push({
    path: `/k8s/docker/terminal/${selection.value.type}/${selection.value.id}/${encodeURIComponent(row.ID)}`,
    query: { connectionId: connectionId.value, name: row.Names }
  });
}
function openFiles(row) {
  filesContainer.value = row;
  filesVisible.value = true;
}
async function showLogs(row) {
  currentContainer.value = row;
  logsVisible.value = true;
  await loadLogs();
}
async function loadLogs() {
  if (!currentContainer.value) return;
  try {
    logs.value = unpack(
      await api.getDockerLogs(
        selection.value.type,
        selection.value.id,
        currentContainer.value.ID,
        { ...params(), tail: logTail.value }
      )
    ).logs;
  } catch (e) {
    ElMessage.error(e.message);
  }
}
function openCreate() {
  createForm.value = {
    name: "",
    image: "",
    restart: "no",
    ports: "",
    volumes: "",
    environment: "",
    command: ""
  };
  createVisible.value = true;
}
function parseEnvironment(raw) {
  const output = {};
  for (const line of raw
    .split("\n")
    .map(x => x.trim())
    .filter(Boolean)) {
    const i = line.indexOf("=");
    if (i < 1) throw new Error(`环境变量格式错误：${line}`);
    output[line.slice(0, i)] = line.slice(i + 1);
  }
  return output;
}
function parsePorts(raw) {
  return raw
    .split("\n")
    .map(x => x.trim())
    .filter(Boolean)
    .map(line => {
      const [mapping, protocol = "tcp"] = line.split("/");
      const values = mapping.split(":").map(Number);
      const [host, container] = values.length === 2 ? values : [0, values[0]];
      if (!container) throw new Error(`端口格式错误：${line}`);
      return { host, container, protocol };
    });
}
function parseVolumes(raw) {
  return raw
    .split("\n")
    .map(x => x.trim())
    .filter(Boolean)
    .map(line => {
      const parts = line.split(":");
      if (parts.length < 2 || parts.length > 3)
        throw new Error(`数据卷格式错误：${line}`);
      return {
        name: parts[0],
        containerPath: parts[1],
        readOnly: parts[2] === "ro"
      };
    });
}
async function createContainer() {
  creating.value = true;
  try {
    const body = {
      connectionId: connectionId.value,
      name: createForm.value.name.trim(),
      image: createForm.value.image.trim(),
      restart: createForm.value.restart,
      environment: parseEnvironment(createForm.value.environment),
      ports: parsePorts(createForm.value.ports),
      volumes: parseVolumes(createForm.value.volumes),
      command: createForm.value.command.trim()
        ? createForm.value.command.trim().split(/\s+/)
        : []
    };
    unpack(
      await api.createDockerContainer(
        selection.value.type,
        selection.value.id,
        body
      )
    );
    ElMessage.success(tr("容器已创建并启动"));
    createVisible.value = false;
    await refreshAll();
  } catch (e) {
    ElMessage.error(e.message);
  } finally {
    creating.value = false;
  }
}

// ================= 镜像操作 =================
function onImageSelection(rows) {
  selectedImages.value = rows;
}
function openRegistry() {
  registryKeyword.value = "";
  registryPage.value = 1;
  registryList.value = [];
  registryTotal.value = 0;
  registryVisible.value = true;
  loadRegistryPage(1);
}
async function loadRegistryPage(page) {
  registryPage.value = page || 1;
  registryLoading.value = true;
  try {
    const data = unpack(
      await api.getDockerRegistryRepositories({
        q: registryKeyword.value.trim(),
        page: registryPage.value,
        pageSize: registryPageSize.value
      })
    );
    registryList.value = data.list || [];
    registryTotal.value = data.total || 0;
  } catch (e) {
    ElMessage.error(e.message);
  } finally {
    registryLoading.value = false;
  }
}
function onRegistrySizeChange(size) {
  registryPageSize.value = size;
  loadRegistryPage(1);
}
function openDownload(row) {
  if (!connectionId.value) {
    ElMessage.warning(tr("请先选择并连接 Docker 资产"));
    return;
  }
  downloadRegistry.value = "docker.1ms.run/";
  downloadImage.value = (row.name || row.pullRef) + ":latest";
  downloadVisible.value = true;
}
async function submitDownload() {
  if (!downloadImage.value.trim()) {
    ElMessage.warning(tr("请输入镜像名称"));
    return;
  }
  downloading.value = true;
  try {
    const image = downloadRegistry.value + downloadImage.value.trim();
    unpack(
      await api.pullDockerImage(selection.value.type, selection.value.id, {
        connectionId: connectionId.value,
        image
      })
    );
    ElMessage.success(tr("镜像下载成功"));
    downloadVisible.value = false;
    await refreshAll();
  } catch (e) {
    ElMessage.error(e.message);
  } finally {
    downloading.value = false;
  }
}
async function removeImages(rows) {
  const list = rows || selectedImages.value;
  if (!list.length) return;
  try {
    await ElMessageBox.confirm(
      `确认删除选中的 ${list.length} 个镜像？`,
      tr("删除镜像"),
      { type: "warning", confirmButtonText: tr("删除"), confirmButtonClass: "el-button--danger" }
    );
    unpack(
      await api.removeDockerImages(selection.value.type, selection.value.id, {
        connectionId: connectionId.value,
        images: list.map(imageRef)
      })
    );
    ElMessage.success(tr("镜像已删除"));
    await refreshAll();
  } catch (e) {
    if (e !== "cancel" && e !== "close" && e?.message)
      ElMessage.error(e.message);
  }
}
async function exportImages(rows) {
  const list = rows || selectedImages.value;
  if (!list.length) return;
  try {
    const res = await api.exportDockerImages(
      selection.value.type,
      selection.value.id,
      { ...params(), images: list.map(imageRef).join(",") }
    );
    downloadBlob(res.data, "docker-images.tar");
  } catch (e) {
    ElMessage.error(e.message);
  }
}
async function importImage({ file }) {
  const formData = new FormData();
  formData.append("file", file);
  try {
    unpack(
      await api.importDockerImage(
        selection.value.type,
        selection.value.id,
        formData,
        params()
      )
    );
    ElMessage.success(tr("镜像导入成功"));
    await refreshAll();
  } catch (e) {
    ElMessage.error(e.message);
  }
}
function openImageOp(mode, row) {
  imageOpMode.value = mode;
  imageOpForm.value = { source: imageRef(row), target: "" };
  imageOpVisible.value = true;
}
async function submitImageOp() {
  if (!imageOpForm.value.target.trim()) {
    ElMessage.warning(tr("请输入目标"));
    return;
  }
  imageOpLoading.value = true;
  try {
    const call =
      imageOpMode.value === "tag" ? api.tagDockerImage : api.renameDockerImage;
    unpack(
      await call(selection.value.type, selection.value.id, {
        connectionId: connectionId.value,
        source: imageOpForm.value.source,
        target: imageOpForm.value.target.trim()
      })
    );
    ElMessage.success(tr("操作成功"));
    imageOpVisible.value = false;
    await refreshAll();
  } catch (e) {
    ElMessage.error(e.message);
  } finally {
    imageOpLoading.value = false;
  }
}
function openTransfer(row, migrate) {
  transferForm.value = {
    image: imageRef(row),
    targetAsset: "",
    migrate
  };
  transferVisible.value = true;
}
async function submitTransfer() {
  if (!transferForm.value.targetAsset) {
    ElMessage.warning(tr("请选择目标主机或物理机"));
    return;
  }
  const [targetAssetType, targetAssetId] = String(transferForm.value.targetAsset).split(":");
  transferLoading.value = true;
  try {
    unpack(
      await api.transferDockerImage(selection.value.type, selection.value.id, {
        connectionId: connectionId.value,
        image: transferForm.value.image,
        targetAssetType,
        targetAssetId: Number(targetAssetId),
        migrate: transferForm.value.migrate
      })
    );
    ElMessage.success(transferForm.value.migrate ? "镜像已迁移" : "镜像已复制");
    transferVisible.value = false;
    await refreshAll();
  } catch (e) {
    ElMessage.error(e.message);
  } finally {
    transferLoading.value = false;
  }
}
function downloadBlob(blob, filename) {
  const url = window.URL.createObjectURL(new Blob([blob]));
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  window.URL.revokeObjectURL(url);
}

// ================= 数据卷操作 =================
function openVolumeCreate() {
  volumeForm.value = { name: "", driver: "" };
  volumeCreateVisible.value = true;
}
async function createVolume() {
  if (!volumeForm.value.name.trim()) {
    ElMessage.warning(tr("请输入数据卷名称"));
    return;
  }
  volumeCreating.value = true;
  try {
    unpack(
      await api.createDockerVolume(selection.value.type, selection.value.id, {
        connectionId: connectionId.value,
        name: volumeForm.value.name.trim(),
        driver: volumeForm.value.driver.trim()
      })
    );
    ElMessage.success(tr("数据卷创建成功"));
    volumeCreateVisible.value = false;
    await refreshAll();
  } catch (e) {
    ElMessage.error(e.message);
  } finally {
    volumeCreating.value = false;
  }
}
async function openVolumeDetail(row) {
  volumeDetailVisible.value = true;
  volumeDetail.value = null;
  volumeDetailContainers.value = [];
  try {
    const data = unpack(
      await api.getDockerVolume(selection.value.type, selection.value.id, row.Name, params())
    );
    volumeDetail.value = data.info || null;
    volumeDetailContainers.value = data.containers || [];
  } catch (e) {
    ElMessage.error(e.message);
  }
}
async function openVolumeContainers(row) {
  try {
    const data = unpack(
      await api.getDockerVolumeContainers(
        selection.value.type,
        selection.value.id,
        row.Name,
        params()
      )
    );
    volumeDetail.value = { Name: row.Name, Driver: row.Driver, Scope: row.Scope, Mountpoint: row.Mountpoint };
    volumeDetailContainers.value = data.list || [];
    volumeDetailVisible.value = true;
  } catch (e) {
    ElMessage.error(e.message);
  }
}
async function removeVolume(row) {
  try {
    const out = await ElMessageBox.prompt(
      `输入数据卷名称 ${row.Name} 确认删除`,
      tr("删除数据卷"),
      {
        inputValidator: v => v === row.Name || "名称不匹配",
        type: "error"
      }
    );
    unpack(
      await api.removeDockerVolume(selection.value.type, selection.value.id, row.Name, {
        connectionId: connectionId.value,
        confirmName: out.value
      })
    );
    ElMessage.success(tr("数据卷已删除"));
    await refreshAll();
  } catch (e) {
    if (e !== "cancel" && e !== "close" && e?.message)
      ElMessage.error(e.message);
  }
}

// ================= 网络操作 =================
function openNetworkCreate() {
  networkForm.value = { name: "", driver: "bridge" };
  networkCreateVisible.value = true;
}
async function createNetwork() {
  if (!networkForm.value.name.trim()) {
    ElMessage.warning(tr("请输入网络名称"));
    return;
  }
  networkCreating.value = true;
  try {
    unpack(
      await api.createDockerNetwork(selection.value.type, selection.value.id, {
        connectionId: connectionId.value,
        name: networkForm.value.name.trim(),
        driver: networkForm.value.driver
      })
    );
    ElMessage.success(tr("网络创建成功"));
    networkCreateVisible.value = false;
    await refreshAll();
  } catch (e) {
    ElMessage.error(e.message);
  } finally {
    networkCreating.value = false;
  }
}
async function openNetworkDetail(row) {
  networkDetailVisible.value = true;
  networkDetail.value = null;
  networkDetailContainers.value = [];
  try {
    const data = unpack(
      await api.getDockerNetwork(selection.value.type, selection.value.id, row.Name, params())
    );
    networkDetail.value = data.info || null;
    networkDetailContainers.value = data.containers || [];
  } catch (e) {
    ElMessage.error(e.message);
  }
}
function openNetworkConnect(row) {
  networkLinkMode.value = "connect";
  networkLinkForm.value = { network: row.Name, container: "" };
  networkLinkVisible.value = true;
}
function openNetworkDisconnect(row) {
  networkLinkMode.value = "disconnect";
  networkLinkForm.value = { network: row.Name, container: "" };
  networkLinkVisible.value = true;
}
async function submitNetworkLink() {
  if (!networkLinkForm.value.container) {
    ElMessage.warning(tr("请选择容器"));
    return;
  }
  networkLinking.value = true;
  try {
    const call =
      networkLinkMode.value === "connect"
        ? api.connectDockerNetwork
        : api.disconnectDockerNetwork;
    unpack(
      await call(selection.value.type, selection.value.id, networkLinkForm.value.network, {
        connectionId: connectionId.value,
        container: networkLinkForm.value.container
      })
    );
    ElMessage.success(tr("操作成功"));
    networkLinkVisible.value = false;
    await refreshAll();
  } catch (e) {
    ElMessage.error(e.message);
  } finally {
    networkLinking.value = false;
  }
}
async function removeNetwork(row) {
  try {
    const out = await ElMessageBox.prompt(
      `输入网络名称 ${row.Name} 确认删除`,
      tr("删除网络"),
      {
        inputValidator: v => v === row.Name || "名称不匹配",
        type: "error"
      }
    );
    unpack(
      await api.removeDockerNetwork(selection.value.type, selection.value.id, row.Name, {
        connectionId: connectionId.value,
        confirmName: out.value
      })
    );
    ElMessage.success(tr("网络已删除"));
    await refreshAll();
  } catch (e) {
    if (e !== "cancel" && e !== "close" && e?.message)
      ElMessage.error(e.message);
  }
}

// ================= Compose 操作 =================
async function openComposeEditor(project) {
  composeForm.value = { project: project || "", content: "" };
  composeEditorVisible.value = true;
  if (!project) return;
  try {
    const data = unpack(
      await api.getDockerComposeProject(
        selection.value.type,
        selection.value.id,
        project,
        params()
      )
    );
    composeForm.value.content = data.content || "";
  } catch (e) {
    ElMessage.error(e.message);
  }
}
async function saveCompose(up) {
  if (!composeForm.value.project.trim()) {
    ElMessage.warning(tr("请输入项目名称"));
    return;
  }
  composeSaving.value = true;
  try {
    unpack(
      await api.saveDockerCompose(selection.value.type, selection.value.id, {
        connectionId: connectionId.value,
        project: composeForm.value.project.trim(),
        content: composeForm.value.content,
        up
      })
    );
    ElMessage.success(up ? "项目已启动" : tr("配置已保存"));
    composeEditorVisible.value = false;
    await loadCompose();
  } catch (e) {
    ElMessage.error(e.message);
  } finally {
    composeSaving.value = false;
  }
}
async function composeAction(project, action) {
  try {
    unpack(
      await api.actionDockerCompose(selection.value.type, selection.value.id, project, {
        connectionId: connectionId.value,
        action
      })
    );
    ElMessage.success(tr("操作成功"));
    await loadCompose();
  } catch (e) {
    ElMessage.error(e.message);
  }
}
async function removeCompose(project) {
  try {
    const out = await ElMessageBox.prompt(
      `输入项目名称 ${project} 确认删除`,
      tr("删除 Compose 项目"),
      {
        inputValidator: v => v === project || "名称不匹配",
        type: "error"
      }
    );
    unpack(
      await api.removeDockerCompose(selection.value.type, selection.value.id, project, {
        connectionId: connectionId.value,
        confirmName: out.value
      })
    );
    ElMessage.success(tr("项目已删除"));
    await loadCompose();
  } catch (e) {
    if (e !== "cancel" && e !== "close" && e?.message)
      ElMessage.error(e.message);
  }
}

onMounted(() => loadAssets().catch(e => (error.value = e.message)));
</script>

<style scoped>
.docker-page {
  padding: 22px;
}
.docker-summary {
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  gap: 12px;
  margin: 16px 0;
}
.docker-summary article {
  padding: 16px;
  border: 1px solid var(--border-light);
  border-radius: var(--radius);
  background: var(--bg-card);
}
.docker-summary span {
  display: block;
  color: var(--text-secondary);
  font-size: 12px;
}
.docker-summary strong {
  display: block;
  margin-top: 7px;
  color: var(--text-primary);
  font-size: 18px;
  overflow: hidden;
  text-overflow: ellipsis;
}
.table-toolbar,
.logs-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 14px;
  flex-wrap: wrap;
}
.inline-upload {
  display: inline-block;
}
.asset-primary {
  font-weight: 650;
  color: var(--text-primary);
}
.asset-secondary {
  font: 12px monospace;
  color: var(--text-secondary);
  margin-top: 4px;
}
.row-actions {
  display: flex;
  flex-wrap: wrap;
}
.docker-logs {
  min-height: 520px;
  margin: 0;
  padding: 18px;
  border-radius: 10px;
  background: #07101f;
  color: #d7e3f4;
  white-space: pre-wrap;
  overflow: auto;
  font:
    13px/1.6 ui-monospace,
    monospace;
}
.create-form {
  margin-top: 18px;
}
.sub-title {
  margin: 16px 0 10px;
  color: var(--text-primary);
}
.pager-row {
  display: flex;
  justify-content: flex-end;
  margin-top: 14px;
}
@media (max-width: 1100px) {
  .docker-summary {
    grid-template-columns: repeat(3, 1fr);
  }
}
@media (max-width: 720px) {
  .docker-summary {
    grid-template-columns: repeat(2, 1fr);
  }
}
</style>
