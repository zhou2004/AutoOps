<template>
  <div class="backup-guide">
    <div class="guide-heading">
      <strong>{{ $tr("备份与恢复流程") }}</strong
      ><el-button link type="primary" @click="open = true">{{
        $tr("使用说明")
      }}</el-button>
    </div>
    <ol>
      <li><b>1</b>{{ $tr("选择集群并立即备份") }}</li>
      <li><b>2</b>{{ $tr("在恢复演练验证新备份") }}</li>
      <li><b>3</b>{{ $tr("验证通过后进入备份集") }}</li>
      <li><b>4</b>{{ $tr("需要取回数据时隔离恢复") }}</li>
    </ol>
    <p>
      {{
        $tr(
          "首次使用可先做 {value0} 并保存到本地，再验证恢复；需要周期备份时创建策略。",
          {
            value0: engine === "redis" ? $tr("RDB 快照") : $tr("逻辑全量")
          }
        )
      }}
    </p>
    <el-drawer
      v-model="open"
      :title="$tr('备份恢复使用说明')"
      size="min(780px, 94vw)"
      append-to-body
    >
      <h3>{{ $tr("第一次备份") }}</h3>
      <ol class="instructions">
        <li>
          {{
            $tr(
              "选择上方操作集群，点击“立即备份”。MySQL / PostgreSQL 可从逻辑全量开始；Redis 使用 RDB 快照。"
            )
          }}
        </li>
        <li>
          {{
            $tr(
              "选择保存位置。不选择仓库时保存在数据库节点的本地暂存目录；需要独立副本时，先在“备份仓库”登记仓库并完成连接检查。"
            )
          }}
        </li>
        <li>
          {{
            $tr(
              "完成预检并确认执行，任务抽屉展示步骤与日志。完成后进入“恢复演练”，检查文件和副本位置，从行操作执行“恢复验证”。成功后此记录自动进入“备份集”；失败任务在“基础管理 → 数据日志 → 数据库操作审计”排查。"
            )
          }}
        </li>
      </ol>
      <h3>{{ $tr("设置定时备份") }}</h3>
      <p>
        {{
          $tr(
            "点击“创建备份策略”，选择集群、方式、仓库及周期。默认每日 02:00，按服务端时区执行；保存后从“备份策略”查看下一次运行时间，也可暂停调度。保留天数用于记录计划保存周期，手动清理不受此时间限制。"
          )
        }}
      </p>
      <h3>{{ $tr("从备份恢复") }}</h3>
      <ol class="instructions">
        <li>
          {{
            $tr("找到完整备份集，先打开“详情”核对方式、基线、文件和恢复范围。")
          }}
        </li>
        <li>
          {{
            $tr(
              "通过该行“操作”选择“隔离恢复”或“恢复验证”，选择有部署权限的 CMDB 主机和空闲端口。Redis Cluster 会为各分片依次使用多个端口。"
            )
          }}
        </li>
        <li>
          {{
            $tr(
              "有连续归档证据时才可选择目标时间；逻辑备份和 RDB 只能恢复备份内容。MySQL binlog 归档必须指定目标时间。"
            )
          }}
        </li>
        <li>
          {{
            $tr(
              "预检并确认后，系统恢复到新目录，启动隔离实例并校验数据，完成后停止隔离实例、保留文件和报告。在备份详情查看恢复主机、端口、目录及核验结果。"
            )
          }}
        </li>
      </ol>
      <el-alert
        type="info"
        :closable="false"
        show-icon
        :title="
          $tr(
            '当前“隔离恢复”和“恢复验证”都执行实际恢复与校验；完成后停止测试实例，不会自动覆盖现有数据库或切换业务入口。正式投入使用需另行安排切换。'
          )
        "
      />
      <h3>{{ $tr("生产中什么时候用") }}</h3>
      <el-descriptions :column="1" border>
        <el-descriptions-item :label="$tr('日常恢复演练')">{{
          $tr(
            "备份完成后、升级前或定期抽检：选择恢复验证，在隔离主机实际还原，检查校验报告，再由业务核对关键表、订单或键值。验证成功才进入备份集。"
          )
        }}</el-descriptions-item>
        <el-descriptions-item :label="$tr('误删 / 误更新')">{{
          $tr(
            "选择误操作之前的备份，具备连续归档时恢复到误操作之前的时间。先隔离恢复并核对受影响数据，再导出需要的表或记录，审核冲突后按变更流程回灌；避免整库回退覆盖之后的正常写入。"
          )
        }}</el-descriptions-item>
        <el-descriptions-item :label="$tr('整库损坏 / 主机丢失')">{{
          $tr(
            "从可用仓库取回备份，在新主机恢复。核对时间点、应用数据、账号和性能后，单独安排新实例接管、复制重建及业务入口切换。"
          )
        }}</el-descriptions-item>
      </el-descriptions>
      <h3>{{ $tr("正式切换前要完成什么") }}</h3>
      <ol class="instructions">
        <li>
          {{
            $tr(
              "记录故障时间和目标恢复时间，保留原库与日志，确认备份和归档范围。没有连续日志时只能回到快照时点。"
            )
          }}
        </li>
        <li>
          {{
            $tr(
              "选择独立主机、目录与端口，预留恢复所需磁盘空间；测试实例不接入生产写入口。"
            )
          }}
        </li>
        <li>
          {{
            $tr(
              "完成系统校验后，由业务检查关键数据、约束、账号权限和应用连接。逻辑备份不等于完整服务器账号备份。"
            )
          }}
        </li>
        <li>
          {{
            $tr(
              "安排业务停写或增量补齐窗口，审批数据回灌或入口变更。保留旧环境，验证新入口读写后再按计划处理旧数据。"
            )
          }}
        </li>
      </ol>
      <p>
        {{
          $tr(
            "主从高可用切换用于节点故障；误删数据会复制到副本，仍需备份恢复。当前页面提供恢复准备和演练，不自动执行正式生产接管或业务切换。"
          )
        }}
      </p>
      <h3>{{ $tr("清理备份和删除仓库") }}</h3>
      <p>
        {{
          $tr(
            "行操作“清理备份”删除指定备份的本地文件及仓库副本，可清理未过期备份，也不要求保留更新的备份；确认前核对时间与方式。自包含备份包各自携带所需基线，其他备份包不受清理影响。历史任务和报告仍可追溯。"
          )
        }}
      </p>
      <p>
        {{
          $tr(
            "仓库可按位置搜索。管理员“删除”只移除仓库登记，不删除桶、文件或凭据；仍有可用备份、策略或未完成任务引用时，先处理引用。"
          )
        }}
      </p>
      <h3>{{ $tr("如何判断成功") }}</h3>
      <el-descriptions :column="1" border>
        <el-descriptions-item :label="$tr('备份完整')">{{
          $tr("备份文件生成并通过完整性检查；仍需验证恢复。")
        }}</el-descriptions-item>
        <el-descriptions-item :label="$tr('本地暂存')">{{
          $tr("文件仍在数据库节点，原节点不可用时可能无法取回。")
        }}</el-descriptions-item>
        <el-descriptions-item :label="$tr('独立仓库已核验')">{{
          $tr("已完成仓库上传与校验，区别于本地暂存成功。")
        }}</el-descriptions-item>
        <el-descriptions-item :label="$tr('恢复已验证')">{{
          $tr("已有实际恢复与数据核验报告；报告只对应这次演练的备份和时间点。")
        }}</el-descriptions-item>
      </el-descriptions>
      <h3>{{ $tr("失败后从哪里排查") }}</h3>
      <p>
        {{
          $tr(
            "打开备份的“备份任务日志”，或进入基础管理 → 数据日志 → 数据库操作审计查看失败步骤。预检阻断时先处理列出的条件；“结果待核验”任务从集群操作中的“核验中断任务”处理，避免重复执行。"
          )
        }}
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
