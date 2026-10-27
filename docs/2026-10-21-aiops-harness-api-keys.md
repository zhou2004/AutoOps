# AI 运维会话、DeepSeek Harness 与平台 API Key

本次修改包括工作负载、弹性伸缩、Docker 管理及 AI 运维操作按钮的图标补齐；诊断工作台升级为会话列表、对话、资产范围、限时授权和工具执行记录；基础管理新增「API Key 管理」。原一次性工作流诊断由工作台右上角「工作流诊断」进入，历史记录保留。

## 工作台

1. 在「AI 运维 → 模型配置」创建模型，支持 OpenAI Compatible、Ollama、DeepSeek Harness。
2. 在工作台选择模型、可选 Skill 和资产连接，发送问题后创建会话。资产范围与模型在会话中固定，切换范围请新建会话。
3. 默认只读。Harness 可调用 `docker_info`、`docker_list`、`docker_logs`；操作工具为 `docker_start`、`docker_stop`、`docker_restart`，须在当前会话权限面板明确授权。停止/重启使用完整容器名称，沿用现有 Docker 服务的名称确认约束。
4. 授权按资产连接和操作类型生效，页面提供 15 分钟、1 小时、4 小时，可随时撤销；API 最大 24 小时。每次工具调用均重新检查用户启用状态、会话归属、当前授权、资产权限、凭据权限及连接只读状态。撤销阻止后续调用，已开始的远端操作不能假定已回滚。
5. 工具结果显示在执行记录，支持展开参数和结果。普通模型支持持续对话与业务汇总分析，不调用操作工具。首批 Harness 操作工具仅覆盖 Docker，不包含任意 shell、文件读写或 Kubernetes 写操作。

对话、授权和工具结果存入数据库。模型上下文回放最近 40 条成功消息，消息文本预算 128 KiB，并附加当前用户有权看到的业务汇总和有限的历史工具结果。SDK 每轮在临时独立目录运行，对话持久化由 AutoOps 负责；不依赖持久 shell 状态。中断会记录状态并释放会话锁；崩溃后的锁到期自动释放。工具调用以会话 + callId 去重，遇到执行中/结果未知的记录应先核实远端状态，不能改 ID 盲目重试。

## 官方 Harness 接入

实现使用 [DeepSeek 官方 Harness](https://github.com/deepseek-ai/deepseek-harness) 的 Python SDK `deepseek-harness-sdk==0.2.1rc2` 和随包 runtime，选择 `sdk-minimal`。`docker/harness/bridge.py` 是本项目的 HTTP 适配层，不是官方 Harness 自带的 HTTP API。

官方适配器使用 DeepSeek Messages 协议：模型配置中的接口地址通常填 `https://api.deepseek.com/anthropic`，模型 ID 填账号实际支持的模型，密钥填 DeepSeek 模型服务密钥。平台 API Key 与模型服务密钥用途不同，不能互换。具体协议见[官方适配器说明](https://github.com/deepseek-ai/deepseek-harness/blob/master/packages/llm/llm-deepseek/README.md)。

桥接层关闭 `persistent-bash`、`persistent-pwsh`、`tool-fs`、`tool-working-directory`，注册六个 AutoOps 工具，并加全局执行守卫，阻止其他工具运行。不会挂载 Docker socket、宿主目录或集群凭据。每轮工具回调使用仅存哈希的随机临时票据，限定用户、会话、有效期及发起 API Key；回调仍需通过业务权限检查。

### Docker Compose

```bash
# 在部署环境生成，不提交到仓库；API 与桥接服务必须使用同一个值。
export AIOPS_HARNESS_SECRET="$(openssl rand -hex 32)"
cd docker
docker compose -f docker-compose.yml -f docker-compose.harness.yml up -d --build
```

覆盖文件为 API 设置 `AIOPS_HARNESS_URL=http://harness:8091`、`AIOPS_CALLBACK_URL=http://devops-api:8000`。桥接设置 `AUTOOPS_CALLBACK_URL` 为同一个 API 根地址。桥接不发布宿主端口，以非 root 用户运行，根文件系统只读，临时状态在 `/tmp`，单实例最多四个并发 SDK 任务。

开发环境可使用：

```bash
python3 -m venv .venv-harness
.venv-harness/bin/pip install -r docker/harness/requirements.txt
export AUTOOPS_CALLBACK_URL=http://127.0.0.1:8000
.venv-harness/bin/uvicorn bridge:app --app-dir docker/harness --host 127.0.0.1 --port 8091 --no-access-log
```

API 进程需另设 `AIOPS_HARNESS_URL=http://127.0.0.1:8091`、`AIOPS_CALLBACK_URL=http://127.0.0.1:8000` 和相同共享密钥。配置完成后，在模型页执行「连接测试」，这会实际调用 SDK 和模型。工作台的「桥接已配置」仅表示配置项齐备，不代替连接测试。

### Kubernetes

构建并推送 `docker/harness/Dockerfile` 镜像，修改 `k8s/api/harness-deployment.yaml` 的镜像地址。创建 `autoops-harness` Secret 的 `shared-secret` 键，然后应用部署，并用 `harness-api.patch.yaml` 补充 API 环境变量：

```bash
kubectl create secret generic autoops-harness --from-literal=shared-secret="$AIOPS_HARNESS_SECRET"
kubectl apply -f k8s/api/harness-deployment.yaml
kubectl patch deployment devops-api --type strategic --patch-file k8s/api/harness-api.patch.yaml
```

示例假设 API Service 为 `devops-api:8000`；不同命名空间或端口需同时修改 API 回调根地址和桥接允许的根地址。内部链路应位于受信任网络；跨网络部署应配置 HTTPS。会话锁和回调票据保存在数据库，可通过同一数据库支撑多副本 API。

## 平台 API Key

基础管理 → API Token 管理：仅管理员可以向指定用户分配和撤销 Token。普通用户只能查看自己的记录。API Key 即用户 API Token，默认实时继承所属用户的模块、资产、凭据和容器权限，不再只开放 AI 接口。名称与 1–365 天有效期可配置，明文仅分配时显示一次，数据库存储 SHA-256 摘要。

完整配置和升级说明见 [统一权限与用户 API Token](2026-10-21-platform-permissions-tokens.md)。此前的 `aiops:read/chat/operate` Key 保留原有范围，由管理员重新分配后才可调用其他模块。Harness 工具操作仍取用户权限、有效 Token 与会话限时授权的交集；每次回调重新核验权限。推荐通过 `Authorization: Bearer <API_KEY>` 请求头传递。

API 返回 `{code, message, data}`，失败须同时检查 HTTP 状态和 `code`。会话消息为 SSE：`start / step / tool / delta / done / error`。除消息 SSE 外，接口如下：

| 方法 | `/api/v1` 下路径 | 说明 |
| --- | --- | --- |
| GET / POST | `/system/api-keys` | 用户自己的记录（管理员可查所有用户） / 管理员为指定用户分配 |
| DELETE | `/system/api-keys/:id` | 管理员撤销指定密钥 |
| GET | `/aiops/catalog`、`/aiops/targets` | 模型目录、可访问连接 |
| GET / POST | `/aiops/sessions` | 最近会话 / 创建会话 |
| GET | `/aiops/sessions/:id` | 会话、消息和执行记录 |
| POST | `/aiops/sessions/:id/messages` | `{ "prompt": "检查容器状态" }`，SSE 回复 |
| PUT | `/aiops/sessions/:id/grants` | `{ "actions": ["docker_restart"], "minutes": 60 }`；空 actions 撤销 |
| POST | `/aiops/sessions/:id/tools` | 外部 Harness 的受控工具入口 |

创建会话示例：

```json
{"title":"容器故障分析","modelId":1,"skillId":0,"targets":[7]}
```

`targets` 使用 `/aiops/targets` 返回的连接 ID，不是主机 ID。调用工具示例：

```json
{"callId":"unique-request-20261021-001","tool":"docker_logs","connectionId":7,"container":"web-api"}
```

外部 Harness 也可加载 `docker/harness/autoops-tools.mjs` 插件，把 `AUTOOPS_CALLBACK_URL` 设置为完整的 `/api/v1/aiops/sessions/<id>/tools` 地址，把 `AUTOOPS_CALLBACK_TOKEN` 设置为平台 API Key；会话必须由同一 Key 所属用户创建并授权。插件安装和 overlay 配置使用[官方 SDK 的 patches 参数](https://github.com/deepseek-ai/deepseek-harness/blob/master/python/sdk/README.md)，参考 `bridge.patch_file()` 的关闭内置工具和加载插件配置。不要把平台密钥放进模型提示词。

## 数据库与验证

启动自动迁移新增 `sys_api_key`、`aiops_session`、`aiops_message`、`aiops_tool_call`、`aiops_harness_ticket`，原模型配置和一次性诊断表保持兼容。

验证命令：

```bash
(cd api && go test ./api/aiops/... ./pkg/apikey ./middleware ./router/aiops -count=1)
(cd web && pnpm build && pnpm i18n:test && pnpm i18n:audit)
python3 -m unittest discover -s docker/harness -p test_runtime.py -v
```

SDK 集成测试使用真实已安装的官方 SDK/runtime，模型服务与工具回调使用本地 HTTP 测试服务，覆盖实际 Messages SSE、工具注册、调用、结果回传和工具白名单；不消耗外部模型额度。生产 DeepSeek 调用仍需部署桥接并配置真实模型密钥。

本次验证结果：

- AI service/controller、API Key、系统控制器、权限中间件及全部 router 包测试通过。
- `pnpm build`、`pnpm i18n:test`、`pnpm i18n:audit` 通过，静态未国际化文案为 0。
- 实际官方 SDK 和本项目 HTTP 桥接服务均完成本地 Messages SSE → 工具调用 → 回调 → 回复测试；SDK 0.2.1rc2 关闭时有上游 `ResourceWarning` 提示，未影响测试结果。
- 浏览器验证多轮对话、资产范围、操作授权、工具结果、密钥创建/撤销、明暗主题及 390px 窄屏。Docker 复制/跨节点迁移入口与工作负载弹性伸缩回归通过；检查本次主要页面的按钮与菜单均带图标。
- 全仓 `vue-tsc` 受原有 tsconfig `ignoreDeprecations: "6.0"` 与安装的 TypeScript 版本不匹配影响。临时命令行覆盖后仍存在原有 cmdb/k8s 等文件的类型错误，本次新增页面/API 文件未出现在错误列表中。

界面预览（全部使用浏览器测试数据）：[明亮工作台](images/aiops-harness-light.png)、[深色会话](images/aiops-harness-dark.png)、[API Key 管理](images/platform-api-keys-dark.png)。
