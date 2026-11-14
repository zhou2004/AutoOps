# Cloudflare DNS 与证书申请

本次优化复用已有 DNS 账户，将连接检查、域名选择、签发账户和证书预检连成可操作的页面流程。Cloudflare 用于 DNS-01 域名验证；证书由所选 ACME CA 签发。自动续期继续按当前需求暂缓，页面操作不自动创建续期策略或部署证书。

## 使用流程

1. 打开 **Web 管理 → 证书管理 → DNS 账户**。已有 `cloudflare` 账户可直接使用，无需再次输入 Token。
2. 在账户行的“更多”菜单中点击“检查连接”，查看凭据配置、可访问区域、区域状态及 DNS 记录读取结果；可输入具体域名检查授权范围。同一菜单中的“域名区域”用于查询和搜索当前 Token 可见的区域。
3. 回到证书列表，点击“通过 Cloudflare 申请证书”。选择 DNS 账户，选择区域后添加根域名或泛域名，也可逐行输入多个区域的域名。
4. 执行“检查域名与 DNS 权限”。这些操作只读取 Cloudflare 数据，不创建、删除或修改 DNS 记录。读取通过不能证明 Token 具有 DNS 写权限，也不代表公网传播验证已完成。
5. 选择 ACME 账户；没有账户时可在申请弹窗内新增，填写名称、业务分组、联系邮箱并选择 Let's Encrypt 测试或正式环境。默认测试环境；测试证书不受浏览器信任。保存账户本身不会向 CA 注册或申请证书。
6. “保存申请”只保存平台证书资产。“保存并预检签发”进入现有操作预检，用户确认执行后由 Worker 进行 ACME 注册、挑战和签发。签发成功后再通过证书绑定和发布流程部署。

根域名与泛域名分别加入 SAN，例如 `example.com` 与 `*.example.com`；后者不覆盖根域名。域名或账户发生变化后，需要重新检查。

## 本次两个域名如何配置

本次目标为 **`mc.stariverfeel.eu.org`** 和 **`mc2.stariverfeel.eu.org`**，两者属于同一个 Cloudflare Zone：`stariverfeel.eu.org`。Token 授权范围选父级 Zone，不创建名为 `mc2` 的独立 Zone。已有 DNS 账户 ID **1** 可以复用。

### 1. 在 Cloudflare 准备 Token

进入 **个人资料 → API Tokens → Create Token**，使用 Edit zone DNS 模板或自定义 Token：

| 项目 | 值 |
| --- | --- |
| 权限一 | Zone → DNS → Edit |
| 权限二 | Zone → Zone → Read |
| Zone Resources | Include → Specific zone → `stariverfeel.eu.org` |

将 Token 保存到配置中心的 Cloudflare 凭据或 DNS 账户的专用凭据框，二者选一种。普通账户邮箱不是 DNS API Token；ACME 联系邮箱也不能替代 Token。现有账户检查可见区域且目标 Zone 为 active 后，无须重复创建。

DNS-01 在 `_acme-challenge.mc.stariverfeel.eu.org` 与 `_acme-challenge.mc2.stariverfeel.eu.org` 临时创建 TXT。无需关闭业务记录的橙云代理，也不要求 Nginx 公网 80 端口可达；CA 要求验证记录在公网权威 DNS 可见。签发不会自动修改业务 A/AAAA/CNAME 记录。Cloudflare 边缘证书、Origin CA 证书和平台申请的 ACME 证书是不同资产。

### 2. 在页面申请并检查结果

1. 进入 **证书管理 → DNS 账户 → cloudflare → 更多 → 检查连接**，分别输入上述两个完整域名。结果应匹配 `stariverfeel.eu.org`。这里只证明读取及区域匹配，实际 TXT 写入由签发验证。
2. 进入 **ACME 账户**，填写联系邮箱并选测试环境。当前已使用用户提供邮箱创建账户 **1：Cloudflare 域名验收 · 测试 CA**，无需再建同一测试账户。
3. 进入 **证书资产 → 通过 Cloudflare 申请证书**，选择业务分组、DNS 账户 1、ACME 账户 1。域名逐行填写；需要分别管理时建立两个资产，需要同步部署时也可放入一张多 SAN 证书。
4. 点击域名检查，然后保存并预检签发，查看计划后确认执行。Worker 完成真实 DNS 写入、传播验证和 CA 签发；在任务详情查看错误与结果。
5. 本次已经留下两个可见资产，无需为了查看效果再次签发：证书 **3** `mc.stariverfeel.eu.org · 测试 CA（未部署）`，证书 **4** `mc2.stariverfeel.eu.org · 测试 CA（未部署）`。成功操作分别为 **29、30**。
6. 在资产“更多 → 证书版本”查看 SAN、颁发者、有效期并导出证书链；需要私钥时使用单独的私钥导出动作，必须满足管理员和证书管理权限。导出操作记录审计，列表及任务不显示私钥。

这两个证书是 **Let’s Encrypt 测试 CA** 签发，浏览器不信任，当前均未部署。正式使用时新建正式环境的 ACME 账户和正式证书申请，使用相同 DNS 账户及域名，预检后签发；不要把测试账户直接改为正式 CA 来继续未完成订单。CA 环境依据 [Let’s Encrypt 测试环境文档](https://letsencrypt.org/docs/staging-environment/)。

### 3. 正式证书如何接入 Nginx

以下是部署配置说明，本次未替换现有线上证书，也未修改公网入口：

1. 在 **证书管理 → 证书部署绑定** 新增记录，选择集群和正式证书。版本 `0` 表示最新签发版本；有固定版本要求时填写实际版本 ID。
2. 证书与私钥路径必须位于该集群的受管目录。例如当前平台实例可使用 `/etc/autoops-web/web-main/current/tls/mc/fullchain.pem`、`/etc/autoops-web/web-main/current/tls/mc/key.pem`；第二域名使用 `tls/mc2/`。不要直接沿用表单中与实际实例不符的 `/etc/nginx` 默认值。
3. 设置“节点 TLS 核验地址”为各节点可访问的服务地址（常用 `127.0.0.1`）、TLS 核验端口为实际监听端口、“TLS 核验域名”为对应完整域名。这里是目标 Nginx 的核验地址，不能填 Cloudflare 边缘入口来代替节点验证。
4. 在 **站点管理** 新建或编辑对应域名，开启 HTTPS、选择该集群的证书部署绑定，填写正确监听和业务规则。原文配置则明确写入相同的 `ssl_certificate`、`ssl_certificate_key` 路径及 `server_name`。
5. 生成草稿并检查候选配置，预检、确认发布。首次上线 HTTPS 时通过站点/配置发布一起带上证书材料；既有 HTTPS 站点替换证书可用绑定的证书发布动作。不要只创建绑定就认为服务已更新。
6. 查看每个节点实际 TLS 指纹、配置版本和健康检查，再访问业务入口。若 Cloudflare 开启代理，客户端看到的是边缘证书，应额外核验源站证书；平台签发版本、部署期望版本和节点实际版本分别记录。

自动续期按当前要求暂缓，以上操作不会创建续期策略。

### 常见错误

| 现象 | 处理 |
| --- | --- |
| Token 有效但域名检查找不到 Zone | 核对完整域名和父级 Zone，确认 Token Include 范围及 Zone Read 权限 |
| 读取成功但签发失败 | 检查 DNS Edit 权限，打开任务详情查看 TXT 写入和传播阶段 |
| 传播长时间 NXDOMAIN | 查询实际权威 DNS；本次网络的 UDP 53 返回陈旧否定缓存，TCP 权威查询可见正确 TXT |
| 测试证书提示不受信任 | 属于测试 CA 的预期行为；正式站点另用正式账户签发 |
| 签发成功但网站仍旧证书 | 检查部署绑定、发布任务、实际 SNI/端口；Cloudflare 代理入口还涉及边缘证书 |

传播检查优先沿用 lego；失败时补充 TCP 权威 DNS 核验，要求所有权威 NS 返回带 AA 标记、名称和挑战值完全一致的 TXT，遇到 CNAME、缺失或不一致记录仍拒绝提交 CA。这是网络兼容回退，没有跳过 DNS 验证。运行 API/Worker 的环境应允许 DNS UDP/TCP 53 和到 Cloudflare/ACME 的 HTTPS 443。

## 凭据与区域

DNS 账户提供两种互斥来源：

- **配置中心凭据**：按服务商加载脱敏名称和 ID，下拉选择，不手填数字；不存在匹配凭据时明确提示。
- **专用凭据**：Cloudflare 使用 API Token，加密保存，编辑时默认保留。勾选“更换已保存的凭据”才要求输入新值；切换服务商或凭据来源后不能误用原凭据。

Cloudflare Token 需要覆盖目标区域的 Zone Read 与 DNS Edit 权限，建议将资源范围限定到需要管理的区域。参考 [lego Cloudflare Provider 文档](https://go-acme.github.io/lego/dns/cloudflare/) 与 [Cloudflare API Token 创建说明](https://developers.cloudflare.com/fundamentals/api/get-started/create-token/)。

域名区域最多读取 10 页、500 条，超出时明确标记截断；目标不在结果中时应缩小 Token 范围。匹配使用最长域名后缀和标签边界，支持多级子域区域，不把 `notexample.com` 当成 `example.com` 的子域。签发预检要求区域已激活。

当前接入流程针对直接托管在 Cloudflare 的验证域名。发现 `_acme-challenge` 的 CNAME 委派时，预检会明确拒绝并提示当前流程不支持，不将读取成功误当作委派链路已经验证。

## 签发一致性

签发计划冻结证书及其 ACME、DNS 和引用凭据的依赖摘要。提交和执行前重新检查，排队期间修改账户或轮换凭据会使旧计划失效，需重新预检；摘要不包含可读 Token。

未完成订单绑定签发账户身份与域名范围。只有身份和域名一致才能恢复；更改 DNS Token 后可以通过新计划重试同一订单，不能拿另一个 CA 或账户继续旧订单。没有身份记录的历史未完成订单不会盲目重放。

签发版本与订单完成状态在同一事务中保存，避免状态保存失败后重试产生重复版本。

## 接口

| 接口 | 用途 |
| --- | --- |
| `GET /api/v1/web/dns-credentials?provider=cloudflare` | 管理员可见的脱敏凭据选项 `{id,name,provider}[]` |
| `GET /api/v1/web/dns-accounts/:id/zones` | `{zones,total,truncated,checkedAt}`，每条区域包含 `id,name,status` |
| `POST /api/v1/web/dns-accounts/:id/check` | 输入 `{domain?}`，返回检查项、所检查域名、匹配区域及 `writeAccess: unverified` |

连接检查会保存最近结果和审计，因此要求模块写权限及管理员身份；区域和凭据列表要求读取权限及管理员身份。请求取消、切换账户或编辑域名后，旧结果不能覆盖新选择。Token 不进入列表、响应、审计或错误信息；连接检查和区域查询使用固定 HTTPS API 地址、超时和响应大小限制，并拒绝重定向。

检查结果按 DNS 账户版本及实际凭据摘要核对后保存。检查过程中账户或配置中心凭据发生变化时，返回冲突，避免将旧检查结果挂到新凭据上。并发检查中较早开始、较晚返回的请求不能覆盖较新的结果。

本次真实检查与测试结果记录在 [Web 管理验收记录](web-management-validation.md)。
