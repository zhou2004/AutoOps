<template>
  <div class="backup-guide">
    <div class="guide-heading">
      <strong>备份与恢复流程</strong
      ><el-button link type="primary" @click="open = true">使用说明</el-button>
    </div>
    <ol>
      <li><b>1</b>选择集群并立即备份</li>
      <li><b>2</b>检查备份集与保存位置</li>
      <li><b>3</b>选择备份进行隔离恢复</li>
      <li><b>4</b>查看校验报告</li>
    </ol>
    <p>
      首次使用可先做{{
        engine === "redis" ? " RDB 快照" : "逻辑全量"
      }}并保存到本地，再验证恢复；需要周期备份时创建策略。
    </p>
    <el-drawer
      v-model="open"
      title="备份恢复使用说明"
      size="min(780px, 94vw)"
      append-to-body
    >
      <h3>第一次备份</h3>
      <ol class="instructions">
        <li>
          选择上方操作集群，点击“立即备份”。MySQL / PostgreSQL
          可从逻辑全量开始；Redis 使用 RDB 快照。
        </li>
        <li>
          选择保存位置。不选择仓库时保存在数据库节点的本地暂存目录；需要独立副本时，先在“备份仓库”登记仓库并完成连接检查。
        </li>
        <li>
          完成预检并确认执行，任务抽屉展示步骤与日志。完成后回到“备份集”，检查状态和副本位置。
        </li>
      </ol>
      <h3>设置定时备份</h3>
      <p>
        点击“创建备份策略”，选择集群、方式、仓库及周期。默认每日
        02:00，按服务端时区执行；保存后从“备份策略”查看下一次运行时间，也可暂停调度。保留天数不代表可以立即删除依赖中的备份。
      </p>
      <h3>从备份恢复</h3>
      <ol class="instructions">
        <li>找到完整备份集，先打开“详情”核对方式、基线、文件和恢复范围。</li>
        <li>
          通过该行“恢复操作”选择“隔离恢复”或“恢复验证”，选择有部署权限的 CMDB
          主机和空闲端口。Redis Cluster 会为各分片依次使用多个端口。
        </li>
        <li>
          有连续归档证据时才可选择目标时间；逻辑备份和 RDB
          只能恢复备份内容。MySQL binlog 归档必须指定目标时间。
        </li>
        <li>
          预检并确认后，系统恢复到新目录，启动隔离实例并校验数据，完成后停止隔离实例、保留文件和报告。在备份详情查看恢复主机、端口、目录及核验结果。
        </li>
      </ol>
      <el-alert
        type="info"
        :closable="false"
        show-icon
        title="当前“隔离恢复”和“恢复验证”都执行实际恢复与校验；完成后停止测试实例，不会自动覆盖现有数据库或切换业务入口。正式投入使用需另行安排切换。"
      />
      <h3>如何判断成功</h3>
      <el-descriptions :column="1" border>
        <el-descriptions-item label="备份完整"
          >备份文件生成并通过完整性检查；仍需验证恢复。</el-descriptions-item
        >
        <el-descriptions-item label="本地暂存"
          >文件仍在数据库节点，原节点不可用时可能无法取回。</el-descriptions-item
        >
        <el-descriptions-item label="独立仓库已核验"
          >已完成仓库上传与校验，区别于本地暂存成功。</el-descriptions-item
        >
        <el-descriptions-item label="恢复已验证"
          >已有实际恢复与数据核验报告；报告只对应这次演练的备份和时间点。</el-descriptions-item
        >
      </el-descriptions>
      <h3>失败后从哪里排查</h3>
      <p>
        打开备份的“备份任务日志”，或进入操作审计查看失败步骤。预检阻断时先处理列出的条件；“结果待核验”任务从集群操作中的“核验中断任务”处理，避免重复执行。
      </p>
    </el-drawer>
  </div>
</template>
<script setup lang="ts">
import { ref } from "vue";
defineProps<{ engine: string }>();
const open = ref(false);
</script>
<style scoped>
.backup-guide {
  margin-bottom: 20px;
  padding: 16px 20px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-light);
}
.guide-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
ol {
  display: flex;
  flex-wrap: wrap;
  gap: 12px 28px;
  padding: 0;
  margin: 14px 0;
  list-style: none;
}
li {
  font-size: 13px;
  color: var(--el-text-color-regular);
}
b {
  color: var(--el-color-primary);
  margin-right: 7px;
}
p {
  color: var(--el-text-color-secondary);
  font-size: 13px;
  line-height: 1.8;
}
h3 {
  margin: 22px 0 12px;
}
.instructions {
  display: block;
  padding-left: 20px;
  list-style: decimal;
}
.instructions li {
  line-height: 1.8;
  margin: 10px 0;
}
.el-descriptions {
  margin-top: 16px;
}
</style>
