<template>
  <el-drawer
    :model-value="modelValue"
    :title="
      $tr(kind === 'vip' ? '高可用 / VIP 配置教程' : 'Cloudflare 证书配置教程')
    "
    size="min(820px, 96vw)"
    append-to-body
    :lock-scroll="false"
    destroy-on-close
    @update:model-value="value => emit('update:modelValue', value)"
  >
    <template v-if="kind === 'vip'">
      <el-alert
        type="info"
        :closable="false"
        :title="
          $tr('主备切换保持 VIP 地址不变；更换 VIP 地址是另一项迁移操作。')
        "
      />
      <ol class="setup-steps">
        <li>
          <strong>{{ $tr("准备成员与入口") }}</strong>
          <p>
            {{
              $tr(
                "选择同一网段内 2–10 台不同主机上的 Nginx 实例，完成发现和接管。每台 SSH 档案需启用、可执行命令；Nginx 健康端口应监听 0.0.0.0 或业务网卡地址。"
              )
            }}
          </p>
        </li>
        <li>
          <strong>{{ $tr("登记集群成员") }}</strong>
          <p>
            {{
              $tr(
                "在集群管理添加实例，确认节点健康、配置一致，设置本机健康检查 URL。成员之间需放通 VRRP 协议 112，网卡名称保持一致。"
              )
            }}
          </p>
        </li>
        <li>
          <strong>{{ $tr("填写高可用参数") }}</strong>
          <p>
            {{
              $tr(
                "打开集群详情 → 高可用 / VIP。填写已预留的 VIP/CIDR、网卡、同网段唯一 VRID，选择成员和首选节点。示例：192.168.x.110/24、ens33、151。"
              )
            }}
          </p>
        </li>
        <li>
          <strong>{{ $tr("预检并部署") }}</strong>
          <p>
            {{
              $tr(
                "点击配置 VIP，检查地址冲突、网卡、端口归属、Keepalived 依赖和成员权限。所有检查通过后确认执行，任务成功后查看拓扑与业务入口。每个集群使用独立服务，不重启 Nginx。"
              )
            }}
          </p>
        </li>
        <li>
          <strong>{{ $tr("切换当前持有者") }}</strong>
          <p>
            {{
              $tr(
                "点击核验 VIP，确保所有备份健康，再点击切换 VIP → 预检 → 确认执行。当前持有者让出地址，健康备份经 VRRP 选举接管；原节点恢复备用，不立即抢回。当前不支持指定某台节点强制接管。"
              )
            }}
          </p>
        </li>
        <li>
          <strong>{{ $tr("更换 VIP 地址、网卡或成员") }}</strong>
          <p>
            {{
              $tr(
                "当前版本尚未提供已部署 VIP 的改址、退役和扩缩容操作，因此这些字段会锁定。切换 VIP 仅改变持有节点。不要直接改数据库、删服务或解绑地址；改址需安排包含旧地址回退和业务入口更新的迁移。"
              )
            }}
          </p>
        </li>
        <li>
          <strong>{{ $tr("处理异常任务") }}</strong>
          <p>
            {{
              $tr(
                "结果待核验时，使用核验中断任务读取原执行证据。不要重复部署或手工删除锁。拓扑读取后台快照，超过 90 秒显示未知；核验 VIP 会执行实时检查。"
              )
            }}
          </p>
        </li>
      </ol>
    </template>
    <template v-else>
      <el-alert
        type="info"
        :closable="false"
        :title="
          $tr(
            'DNS 账户、ACME 账户和证书资产分别配置；签发成功后还需要单独部署。'
          )
        "
      />
      <ol class="setup-steps">
        <li>
          <strong>{{ $tr("准备 Cloudflare API Token") }}</strong>
          <p>
            {{
              $tr(
                "在 Cloudflare 个人资料 → API Tokens 创建 Token，权限为 Zone / Zone / Read 与 Zone / DNS / Edit，资源范围选择域名所属 Zone。填写子域名证书时，Token 仍需覆盖其父级 Zone。"
              )
            }}
          </p>
        </li>
        <li>
          <strong>{{ $tr("连接 DNS 账户") }}</strong>
          <p>
            {{
              $tr(
                "在证书管理 → DNS 账户新增 Cloudflare，选择配置中心凭据或加密保存专用 Token。已有账户可直接复用。更多 → 检查连接和查看区域，随后输入完整域名进行检查。读取通过只证明读取权限，DNS 写入由实际挑战验证。"
              )
            }}
          </p>
        </li>
        <li>
          <strong>{{ $tr("创建 ACME 账户") }}</strong>
          <p>
            {{
              $tr(
                "在 ACME 账户填写联系邮箱，首次测试选择 Let’s Encrypt 测试环境。测试与正式环境使用不同账户；测试证书不受浏览器信任，不应部署到对外站点。"
              )
            }}
          </p>
        </li>
        <li>
          <strong>{{ $tr("创建证书申请") }}</strong>
          <p>
            {{
              $tr(
                "在证书资产点击申请证书，选择业务分组、Cloudflare DNS 账户、ACME 账户，添加完整域名。可以逐域名单独申请，也可以在一张证书中添加多个域名。泛域名不自动覆盖根域名。"
              )
            }}
          </p>
        </li>
        <li>
          <strong>{{ $tr("检查并签发") }}</strong>
          <p>
            {{
              $tr(
                "检查域名通过后保存并签发，查看预检结果并确认任务。平台自动创建本次 _acme-challenge TXT，确认传播后提交 CA，结束时清理本次记录。DNS-01 不要求业务服务器开放公网 80 端口。"
              )
            }}
          </p>
        </li>
        <li>
          <strong>{{ $tr("查看、导出和部署") }}</strong>
          <p>
            {{
              $tr(
                "签发成功后查看 SAN、有效期和版本。公开证书与私钥分开导出；私钥导出需要额外权限。部署关系绑定证书、集群、远端证书/私钥路径及真实 TLS 验证地址，再预检并发布。"
              )
            }}
          </p>
        </li>
        <li>
          <strong>{{ $tr("续期与常见问题") }}</strong>
          <p>
            {{
              $tr(
                "接入 DNS 账户或签发证书不会自动启用续期。Zone 不匹配时检查域名拼写和 Token 范围；传播失败时检查权威 DNS 与网络；签发成功不等于线上 TLS 已切换。"
              )
            }}
          </p>
        </li>
      </ol>
      <el-link
        href="https://go-acme.github.io/lego/dns/cloudflare/"
        target="_blank"
        rel="noopener noreferrer"
        type="primary"
        >{{ $tr("Cloudflare 权限说明") }}</el-link
      >
    </template>
  </el-drawer>
</template>
<script setup lang="ts">
defineProps<{ modelValue: boolean; kind: "vip" | "certificate" }>();
const emit = defineEmits<{
  (event: "update:modelValue", value: boolean): void;
}>();
</script>
<style scoped>
.setup-steps {
  padding-left: 24px;
  margin: 24px 0;
}
.setup-steps li {
  padding: 0 0 20px 8px;
  line-height: 1.8;
}
.setup-steps strong {
  font-size: 15px;
  color: var(--el-text-color-primary);
}
.setup-steps p {
  margin: 8px 0 0;
  color: var(--el-text-color-regular);
  overflow-wrap: anywhere;
}
</style>
