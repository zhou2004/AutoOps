# Ansible 自动化运维平台核心技术面试题库 & 架构解析

本文档基于项目背景：“主导 Ansible 自动化任务分发模块的设计与开发，实现 CLI 向 RESTful API 平台化转型，解决高并发与日志实时推送问题”，整理了相关的深度面试题及技术实现思路。

---

## 一、 项目架构与设计思路 (Architecture & Design)

### Q1: 从传统 CLI 脚本向 RESTful API 平台化转型中，你做了哪些关键的架构设计？
**考察点：** 系统分层、解耦、API 设计规范。
**参考回答思路：**
1.  **分层架构**：采用了典型的 Controller (Gin) -> Service (业务逻辑) -> DAO (数据持久化) 分层结构。
    *   `router/` 层负责参数校验和 HTTP 协议处理。
    *   `service/` (如 `taskansible.go`, `taskwork.go`) 封装 Ansible 的核心调用逻辑，屏蔽底层命令行的复杂性。
2.  **异步化设计**：CLI 是同步阻塞的，API 必须是异步的。设计了“任务提交接口” (立即返回 TaskID) 和 “任务查询接口” 分离的模式。
3.  **数据模型抽象**：将 Ansible 的 Playbook、Inventory、Vars 抽象为数据库模型 (`model.TaskAnsible`, `model.ConfigAnsible`)，实现了配置的可视化管理和复用。

### Q2: 既然提到“平台化”，你是如何规划整个自动化任务分发模块的？
**考察点：** 业务规划能力、模块划分。
**参考回答思路：**
1.  **核心执行引擎**：基于 Ansible Ad-hoc 和 Playbook 的封装 (见 `taskansible.go`)。
2.  **配置中心**：将 Inventory (主机清单)、Global Vars (全局变量) 配置化管理 (见 `configansible.go`)。
3.  **调度中心**：支持 Cron 表达式的定时任务调度，集成 `robfig/cron` (见 `globalScheduler.go`)。
4.  **监控与反馈**：实时日志流 (SSE) 和 任务状态生命周期管理。

---

## 二、 Ansible 异步调用与高并发处理 (Concurrency & Async)

### Q3: 你是如何使用 Go 实现 Ansible 的异步调用的？如何避免通过 API 调用时阻塞主线程？
**考察点：** Go 协程、`os/exec`、非阻塞 I/O。
**参考回答思路：**
1.  **Goroutine 启动**：在 Service 层接收到任务请求后，通过 `go func() { ... }` 启动后台协程执行具体的 `ExecuteTask` 逻辑，主线程立即返回 TaskID 给前端。
2.  **命令封装**：底层使用 `os/exec` 或 `ssh` (如 `taskwork.go` 中的 `executeSSHTask`) 调用 Ansible 命令。
3.  **状态流转**：
    *   初始状态为 `Pending` (等待中)。
    *   协程开始执行时更新为 `Running` (运行中)。
    *   执行结束根据 Exit Code 更新为 `Success` 或 `Failed`。
    *   这一状态变化持久化到 MySQL 数据库中，供前端轮询或查看。

### Q4: 面对“海量节点并发调度”，你是如何利用 Go 协程机制突破性能瓶颈的？如果同时有 1000 台机器需要执行，如何控制并发度？
**考察点：** 协程池、Channel 缓冲、Redis 队列、系统负载保护。
**参考回答思路：**
1.  **Ansible 自身的 Fork**：首先利用 Ansible 配置文件中的 `forks` 参数，控制 Ansible 进程内部的 SSH 并发数。
2.  **任务队列 (Task Queue)**：在代码中引入了队列机制 (如 `taskwork.go` 中提到的 `TaskQueue` 和 Redis)。
    *   API 接收请求后，不是直接 `exec`，而是将任务 Push 到 Redis List 或 Go Channel 中。
    *   **Worker Pool 模式**：后台启动固定数量的 Worker 协程 (Dispatcher) 消费队列。
    *   **流控**：通过限制 Worker 的数量和 Channel 的 Buffer 大小，防止瞬间启动过多 `ansible-playbook` 进程导致管理节点 CPU/内存耗尽（惊群效应）。

### Q5: 任务执行过程中，如果服务进程（API Server）重启或崩溃，如何保证任务状态的一致性？(Crash Recovery)
**考察点：** 容灾恢复、状态机设计。
**参考回答思路：**
1.  **启动时恢复机制**：参考 `globalScheduler.go` 中的 `LoadScheduledTasks` 逻辑。在服务启动时，扫描数据库中状态为 `Running` 的任务。
    *   因为进程崩溃，这些 `Running` 的任务实际上已经丢失上下文。
    *   策略：将这些任务标记为 `Failed` (异常终止)，或者根据业务逻辑重新加入调度队列。
2.  **优雅停止 (Graceful Shutdown)**：利用 Go 的 `context` 和信号监听，在服务停止前拒绝新请求，并等待当前正在执行的关键步骤完成（或保存 Checkpoint）。

---

## 三、 实时日志流推送 (SSE - Server-Sent Events)

### Q6: 为什么在一个自动化运维平台中选择 SSE 而不是 WebSocket 来做日志推送？
**考察点：** 技术选型对比、协议特性。
**参考回答思路：**
1.  **场景匹配**：日志推送是一个典型的 **单向通信** 场景（服务器 -> 客户端）。WebSocket 是全双工的，对于只读日志来说过重。
2.  **协议轻量**：SSE 基于 HTTP 协议 (`text/event-stream`)，无需像 WebSocket 那样处理复杂的握手及自定义的心跳协议，对防火墙和负载均衡更友好。
3.  **断线重连**：SSE 协议原生支持自动重连 (浏览器端 `EventSource` 会自动尝试)，减少了前端代码复杂度。

### Q7: 请详细描述后端也就是 Go 侧是如何实现 SSE 日志推送的？日志文件很大怎么办？
**考察点：** HTTP Flusher、文件读取 (`tail`)、缓冲区管理。
**参考回答思路：**
1.  **设置响应头**：设置 `Content-Type: text/event-stream`，`Cache-Control: no-cache`，`Connection: keep-alive`。
2.  **Flushing**：在 Gin 中，通过 `c.Writer.Flush()` 将缓冲区的数据立即推送到客户端，而不是等待请求结束。
3.  **日志读取策略 (Tail)**：
    *   **实时性**：实际上是类似于 `tail -f` 的逻辑。代码中会在一个循环中读取日志文件 (`os.Open` + seek)。
    *   **增量读取**：记录上一次读取的文件偏移量 (Offset)，循环检查文件大小变化，只读取新增部分。
4.  **大文件处理**：不一次性读取整个文件放入内存。按行或按块读取，读一部分 Flush 一部分。并且前端只接收“增量”日志。

### Q8: 前端 (Vue) 在处理 SSE 时遇到了哪些坑？如何处理鉴权？
**考察点：** 前端工程化、EventSource 限制。
**参考回答思路：**
1.  **鉴权问题**：原生的 `EventSource(url)` 不支持设置自定义 HTTP Headers (如 `Authorization: Bearer ...`)。
    *   **解决方案**：虽然使用了 `event-source-polyfill` 可以支持 Header，但为了兼容性，本项目采用了 **URL Query Parameter** 传递 Token (如 `?token=xxx`)，并在后端中间件 (`authMiddleware.go`) 中兼容了从 Query 获取 Token 的逻辑。
2.  **生命周期管理**：必须在 Vue 组件销毁 (`onUnmounted`) 或 Dialog 关闭时，显式调用 `.close()` 关闭连接，否则会造成后台连接泄漏和带宽浪费。

---

## 四、 规划与进阶 (Future & Roadmap)

### Q9: 随着节点规模从一百台扩展到一万台，现有的 Ansible 架构会遇到什么瓶颈？你打算如何升级？
**考察点：** 分布式架构、性能瓶颈分析。
**参考回答思路：**
1.  **瓶颈**：
    *   **单点执行瓶颈**：单台 Ansible 控制节点的 CPU/网络带宽有限，无法维持数千个并发 SSH 连接。
    *   **SSH 握手与传输**：SSH 协议本身开销大。
2.  **升级方案 (分布式调度)**：
    *   **Master-Worker 架构**：将目前单体的 API Server 拆分。Master 负责任务分发和状态管理。
    *   **代理节点 (Proxy/Runner)**：部署多个 Runner 节点，Master 将任务通过 RPC/MQ 下发给不同的 Runner，由 Runner 去执行 Ansible 命令，分摊 SSH 连接压力。
    *   **Ansible 优化**：开启 `SSH Pipelining`，使用 `Mitogen` 插件加速 Ansible 执行效率。

### Q10: 既然涉及任务下发，在 Go 中如何设计任务的优先级？
**思路**：参考 `taskwork.go` 中的 `getTaskPriority`。
*   **实现**：使用 Redis 的多 List 实现优先级队列 (如 `queue:high`, `queue:normal`, `queue:low`)。
*   **策略**：
    *   **立即执行任务** -> High 优先级。
    *   **定时任务** -> Normal 优先级。
    *   **批量巡检/报表** -> Low 优先级。
*   **调度**：消费者协程优先 Pop `queue:high` 中的数据。
