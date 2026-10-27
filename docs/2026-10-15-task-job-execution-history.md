# 任务作业：多语言、Cron、执行历史与回滚

## 功能范围

任务中心的“任务模板”和“任务作业”现支持以下执行类型：

| 类型值 | 类型 | 远端执行方式 |
| --- | --- | --- |
| 1 | Shell | `bash script.sh` |
| 2 | Python | `python3 script.py` |
| 3 | Ansible Playbook | `ansible-playbook playbook.yml` |
| 4 | JavaScript | `node script.js` |
| 5 | PowerShell | `pwsh -NoProfile -File script.ps1` |
| 6 | Go | `go run script.go` |
| 7 | Ruby | `ruby script.rb` |
| 8 | PHP | `php script.php` |
| 9 | Perl | `perl script.pl` |
| 10 | 直接命令 | `bash -lc '<command>'` |

目标服务器必须安装对应运行时，并提供 `setsid`（通常由 util-linux 提供）以支持安全的进程组暂停和停止。脚本内容通过 Base64 写入唯一临时文件，避免脚本正文参与远端 Shell 拼接；执行结束后临时脚本、PID、退出码和远端日志文件会被清理。

## 任务模式

- 手动任务：仅在用户点击“立即执行”时运行。
- 定时任务：使用标准 5 位 Cron（分、时、日、月、周）；定时开关控制自动调度。
- 定时任务也可点击“立即执行”，该操作不会改变原有定时开关状态。
- 执行内容可来自一个或多个任务模板，也可直接在任务作业中填写命令或脚本。

## 执行历史

每次手动或定时触发都会创建一条 `task_execution_history`，并为每个“主机 × 模板”创建 `task_execution_log`。记录包含：

- 触发方式和执行人；
- 当次任务名称、主机、Cron 和备注；
- 当次实际执行的模板/命令快照；
- 每个主机的状态、开始/结束时间、耗时和完整日志。

`max_history_keep` 控制每个任务最大保留记录数，范围为 1–200，默认 20。一次执行汇总完成后会自动删除超限的最旧历史、子日志和对应本地日志文件。

## 暂停、恢复与停止

- “暂停执行”向该次执行的远端进程组发送 `SIGSTOP`，状态和执行历史显示为“已暂停”。
- “恢复执行”发送 `SIGCONT`，队列中尚未启动的工作项也会遵守暂停状态。
- “停止执行”先发送 `SIGTERM`，进程未退出时再发送 `SIGKILL`；历史状态记录为“已停止”，已产生的日志继续保留。
- 定时调度开关与当前执行状态相互独立：停止某一次定时执行不会关闭后续 Cron 调度；暂停 Cron 调度也不会隐式杀死已经开始的命令。

## 回滚语义

历史列表中的“回滚”恢复该次执行时的任务配置和模板内容，包括命令、语言、模板、主机、Cron 与备注。回滚只恢复配置，不会自动执行。

运行中的普通任务不能回滚；已启用自动调度的定时任务需先暂停，以避免调度器继续使用旧配置。若历史中的模板已删除，回滚会重建模板并更新任务关联。

## API

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/api/v1/task/:id/run` | 手动立即触发一次 |
| POST | `/api/v1/task/:id/schedule` | 启用 Cron 调度 |
| POST | `/api/v1/task/:id/execution/pause` | 暂停当前执行 |
| POST | `/api/v1/task/:id/execution/resume` | 恢复当前执行 |
| POST | `/api/v1/task/:id/execution/stop` | 停止当前执行 |
| GET | `/api/v1/task/:id/history` | 分页查询执行历史 |
| GET | `/api/v1/task/:id/history/:history_id` | 查询命令快照和逐主机日志 |
| POST | `/api/v1/task/:id/history/:history_id/rollback` | 恢复该次执行配置 |

## Ansible 任务记录

Ansible 任务列表的任务名称已作为可点击入口，路由到隐藏菜单页 `/task/ansible/history?id=<taskID>&name=<taskName>`，复用现有 `AnsibleTaskHistory.vue` 页面。
