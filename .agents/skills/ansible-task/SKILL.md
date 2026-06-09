---
name: ansible-task
description: "AutoOps Ansible 任务管理模块实现了 Ansible 运维任务的创建、执行、历史追踪、定时调度和视图分组管理。支持手动任务和 Git 自动任务两种模式。"
---

# Ansible 任务管理模块 (ansible-task)

## 概述

Ansible 任务管理模块是 AutoOps 的自动化运维核心，支持通过 Ansible Playbook 对目标主机执行批量操作。

### 任务类型

| 类型 | 值 | 说明 |
|------|-----|------|
| 手动任务 | 1 | 手动上传 Playbook 文件（.yml/.yaml）和 Roles 压缩包（.zip） |
| 自动任务 | 2 | 指定 Git 仓库地址，自动拉取并解析 Playbook |
| K8s 任务 | 3 | 专门用于 K8s 集群部署的自动化任务 |

### 核心功能

- **任务 CRUD**：创建、编辑、删除 Ansible 任务
- **任务执行**：支持 Ansible-playbook 命令执行，实时 SSE 日志推送
- **定时调度**：支持 Cron 表达式配置周期性执行
- **历史追踪**：保存每次执行的详细日志和结果
- **视图分组**：通过 `TaskAnsibleView` 对任务进行分组管理
- **配置中心**：支持 Inventory、全局变量、额外变量、命令行参数的配置管理

## 核心架构

### 数据模型

#### 1. **task_ansible 表** - 任务主表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键ID |
| name | string | 任务名称（唯一） |
| description | text | 任务描述 |
| type | int | 任务类型:1-手动,2-Git,3-K8s |
| git_repo | string | Git仓库地址 |
| host_groups | text | 主机分组JSON |
| all_host_ids | text | 所有主机ID JSON数组 |
| global_vars | text | 全局变量JSON |
| extra_vars | text | 额外参数YAML/JSON |
| cli_args | text | CLI命令行参数 |
| status | int | 任务状态:1-等待中,2-运行中,3-成功,4-异常 |
| task_count | int | 任务数量 |
| total_duration | int | 总耗时(秒) |
| use_config | int | 是否使用配置中心:0-否,1-是 |
| inventory_config_id | uint | Inventory配置ID (FK → config_ansible) |
| global_vars_config_id | uint | 全局变量配置ID (FK → config_ansible) |
| extra_vars_config_id | uint | 额外变量配置ID (FK → config_ansible) |
| cli_args_config_id | uint | 命令行参数配置ID (FK → config_ansible) |
| max_history_keep | int | 最大保留历史记录数（默认3） |
| cron_expr | string | 定时表达式 |
| is_recurring | int | 是否周期性任务:0-否,1-是 |
| view_id | uint | 视图ID (FK → task_ansible_view) |
| created_at | datetime | 创建时间 |
| updated_at | datetime | 更新时间 |

#### 2. **task_ansiblework 表** - 子任务表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键ID |
| task_id | uint | 父任务ID (FK → task_ansible) |
| entry_file_name | string | Playbook文件名 |
| entry_file_path | string | Playbook文件路径 |
| log_path | string | 日志文件路径 |
| status | int | 状态:1-等待中,2-运行中,3-成功,4-异常 |
| start_time | datetime | 开始时间 |
| end_time | datetime | 结束时间 |
| duration | int | 执行耗时(秒) |
| exit_code | int | 退出代码 |
| error_msg | text | 错误信息 |

#### 3. **task_ansible_view 表** - 视图分组表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键ID |
| name | string | 视图名称（唯一） |
| created_at | datetime | 创建时间 |
| updated_at | datetime | 更新时间 |

#### 4. **task_ansible_history 表** - 执行历史主表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键ID |
| task_id | uint | 任务ID |
| uniq_id | string | 执行唯一标识 |
| status | int | 执行状态 |
| total_duration | int | 总耗时(秒) |
| trigger | int | 触发方式:1-手动,2-定时,3-API |
| started_at | datetime | 开始时间 |
| finished_at | datetime | 完成时间 |

#### 5. **task_ansiblework_history 表** - 历史子任务详情表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键ID |
| history_id | uint | 历史记录ID |
| task_id | uint | 任务ID |
| work_id | uint | 子任务ID |
| host_name | string | 主机名/IP |
| status | int | 状态 |
| log_path | string | 日志文件路径 |
| duration | int | 耗时(秒) |

#### 6. **config_ansible 表** - 配置中心表

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键ID |
| name | string | 配置名称（唯一） |
| type | int | 类型:1-inventory,2-global_vars,3-extra_vars,4-cli_args |
| content | longtext | 配置内容 |
| remark | string | 备注 |

### 关联关系

```
task_ansible_view (1) ──→ (N) task_ansible (view_id)
task_ansible (1) ──→ (N) task_ansiblework (task_id)
task_ansible (1) ──→ (N) task_ansible_history (task_id)
task_ansible_history (1) ──→ (N) task_ansiblework_history (history_id)
config_ansible (1) ──→ (N) task_ansible (inventory_config_id / global_vars_config_id / extra_vars_config_id / cli_args_config_id)
```

### 任务状态流转

```
创建 → 等待中(1) → 运行中(2) → 成功(3)
                             → 异常(4)
```

### 执行流程

1. 用户点击启动任务 → `StartJob` → `ExecuteTask`
2. 更新任务状态为运行中
3. 异步 goroutine 执行任务：
   - 检查任务目录和 hosts 文件
   - 若启用配置中心，覆盖对应配置
   - 遍历子任务，对每个 Playbook 执行 `ansible-playbook -i hosts <playbook> -v`
   - 实时将日志写入文件，同时通过 SSE 推送给前端
4. 所有子任务完成后，更新状态
5. 保存历史记录，清理旧历史

## 后端实现

### 文件结构

```
api/api/task/
├── model/
│   ├── taskansible.go              # 任务主表模型
│   ├── taskansiblework.go          # 子任务模型
│   ├── taskansiblehistory.go       # 历史记录模型
│   ├── taskansibleview.go          # 视图分组模型
│   └── configansible.go            # 配置中心模型
├── dao/
│   ├── taskansible.go              # 任务DAO（含缓存）
│   ├── taskansibleview.go          # 视图DAO
│   └── configansible.go            # 配置中心DAO
├── service/
│   ├── taskansible.go              # 任务服务（核心逻辑）
│   ├── taskansibleview.go          # 视图服务
│   └── configansible.go            # 配置中心服务
└── controller/
    ├── taskansible.go              # 任务控制器
    ├── taskansibleview.go          # 视图控制器
    └── configansible.go            # 配置中心控制器

api/router/task/task.go             # 路由注册
```

### API 接口

#### 任务管理 API
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/task/ansiblelist | 获取任务列表（分页，page/size） |
| POST | /api/v1/task/ansible | 创建Ansible任务（multipart/form-data） |
| PUT | /api/v1/task/ansible/:id | 修改任务（JSON body） |
| DELETE | /api/v1/task/ansible/:id | 删除任务（级联删除子任务） |
| GET | /api/v1/task/ansible/:id | 获取任务详情 |
| POST | /api/v1/task/ansible/:id/start | 启动任务 |
| GET | /api/v1/task/ansible/query | 多条件查询任务（name/type/viewName/page/size） |
| GET | /api/v1/task/ansible/query/name | 按名称模糊查询 |
| GET | /api/v1/task/ansible/query/type | 按类型查询 |
| POST | /api/v1/task/k8s | 创建K8s部署任务 |

#### 历史记录 API
| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/task/ansible/:id/history | 获取历史列表 |
| GET | /api/v1/task/ansible/:id/history/:history_id | 获取历史详情 |
| DELETE | /api/v1/task/ansible/:id/history/:history_id | 删除历史记录 |
| GET | /api/v1/task/ansible/:id/log/:work_id | 获取实时日志(SSE) |
| GET | /api/v1/task/ansible/history/work/:work_history_id/log | 获取历史日志 |
| GET | /api/v1/task/ansible/history/detail/task/:task_id/work/:work_id/history/:history_id/log | 获取历史日志(通过详细信息) |

#### 实时日志 (SSE)
- 路径: `GET /api/v1/task/ansible/:id/log/:work_id`
- 响应格式: `text/event-stream`
- 事件类型: `data:`（日志行）, `event: complete`（完成信号）, `event: error`（错误信号）
- 支持实时追踪文件变化（200ms 轮询间隔）
- 支持心跳机制（5秒间隔）

#### 视图管理 API
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/task/ansible/view | 创建视图 |
| PUT | /api/v1/task/ansible/view/:id | 更新视图 |
| DELETE | /api/v1/task/ansible/view/:id | 删除视图（关联任务 view_id 置空） |
| GET | /api/v1/task/ansible/view/all | 获取所有视图 |
| GET | /api/v1/task/ansible/view | 分页获取视图 |

#### 配置中心 API
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/config/ansible | 创建配置 |
| PUT | /api/v1/config/ansible/:id | 更新配置 |
| DELETE | /api/v1/config/ansible/:id | 删除配置 |
| GET | /api/v1/config/ansible/:id | 获取配置详情 |
| GET | /api/v1/config/ansible | 获取配置列表 |

### 关键服务方法

#### 任务创建 `CreateTask`

接收 `multipart/form-data`，处理流程：
1. 解析表单参数（type/name/hostGroups/gitRepo/variables 等）
2. 解析 `*uint` 类型的配置ID字段（inventory_config_id, view_id 等）
3. 处理上传文件（playbooks 和 roles zip）
4. 创建任务数据库记录
5. 根据类型调用 `handleManualTask` 或 `handleGitTask`
6. 创建项目目录 `task/{taskID}/{name}/`
7. 生成 hosts 文件、保存 Playbook、解压 Roles
8. 触发调度器配置变更钩子

#### 任务执行 `ExecuteTask`

异步 goroutine 执行：
1. 获取任务详情和子任务列表
2. 构建 ansible-playbook 命令
3. 执行命令并实时写日志文件
4. 更新子任务状态
5. 全部完成后保存历史记录

#### 综合查询 `GetTasks`

支持多条件查询（DAO 层 `GetTasks`）：
- 支持 `JOIN task_ansible_view` 按视图名称筛选
- 支持 `Preload("View")` 加载视图信息
- 支持任务名称模糊查询（LIKE）
- 支持任务类型精确查询

#### 更新任务 `UpdateTask`

- 运行中的任务不可修改（状态=2时拒绝）
- 支持增量更新：CronExpr、IsRecurring、ViewID 等
- ViewID 支持置空（设为 nil 清除分组）
- 更新后触发调度器配置变更钩子

### DAO 缓存机制

`TaskAnsibleDao` 内置内存缓存（5秒 TTL）：
- 子任务详情缓存：`work_{taskID}_{workID}`
- 任务详情缓存：`task_detail_{taskID}`
- 子任务状态缓存：`work_status_{taskID}_{workID}`
- 任何写操作（创建/更新/删除）清空全部缓存

## 前端实现

### 文件结构

```
web/src/
├── api/task.js                     # 所有任务相关API封装
└── views/task/
    ├── TaskAnsible.vue             # Ansible任务管理主页面
    ├── Job/
    │   ├── AnsibleJobFlow.vue      # 任务执行流程组件
    │   └── CreateTaskHost.vue      # 主机选择组件
    └── ...                         # 其他任务相关组件
```

### 前端 API 封装 (`api/task.js`)

```javascript
// 任务管理
GetAnsibleTaskList(params)        // GET /task/ansiblelist
CreateAnsibleTask(data)            // POST /task/ansible (multipart/form-data)
UpdateAnsibleTask(data)            // PUT /task/ansible/:id
DeleteAnsibleTask(id)              // DELETE /task/ansible/:id
GetAnsibleTaskById(id)             // GET /task/ansible/:id
StartAnsibleTask(id)               // POST /task/ansible/:id/start
GetAnsibleTasksByName(params)      // GET /task/ansible/query/name
GetAnsibleTasksByType(params)      // GET /task/ansible/query/type
GetAnsibleTasksByQuery(params)     // GET /task/ansible/query (多条件)
GetAnsibleTaskLog(id, workId)      // GET /task/ansible/:id/log/:work_id (SSE)

// 历史记录
GetAnsibleTaskHistory(params)      // GET /task/ansible/:id/history
GetAnsibleHistoryDetail(params)    // GET /task/ansible/:id/history/:history_id
GetAnsibleTaskLogByHistory(params) // GET .../history/detail/task/.../work/.../history/.../log

// 视图管理
CreateAnsibleView(data)            // POST /task/ansible/view
UpdateAnsibleView(id, data)        // PUT /task/ansible/view/:id
DeleteAnsibleView(id)              // DELETE /task/ansible/view/:id
GetAllAnsibleViews()               // GET /task/ansible/view/all

// 配置中心
GetAnsibleConfigList(params)       // GET /config/ansible
CreateAnsibleConfig(data)          // POST /config/ansible
UpdateAnsibleConfig(data)          // PUT /config/ansible/:id
DeleteAnsibleConfig(id)            // DELETE /config/ansible/:id
```

### 页面数据映射

后端接口返回字段为大写（Go 结构体字段名），前端 `fetchTasks` 中做驼峰映射：

```javascript
tasks.value = taskList.map(item => ({
  id: item.ID,
  name: item.Name,
  type: item.Type,
  status: item.status,
  is_recurring: item.IsRecurring,
  cron_expr: item.CronExpr,
  view_name: item.View ? item.View.name : (item.ViewName || ''),
  createdAt: formatTime(item.CreatedAt),
  // ...
}))
```

### 搜索模式

`TaskAnsible.vue` 支持 4 种搜索模式：
- **全部(all)**: 默认，使用 `GetAnsibleTasksByQuery` 多条件查询
- **按名称(name)**: 使用 `GetAnsibleTasksByName`
- **按类型(type)**: 使用 `GetAnsibleTasksByType`
- **按视图(view)**: 使用 `GetAnsibleTasksByQuery({ viewName })`

## 测试方法

### 环境准备

```bash
# 启动后端
cd api && go run main.go -c config.yaml

# 获取 token
# 通过登录接口或直接生成
```

### 测试场景

#### 场景 1：创建手动任务

```bash
# 上传 playbook 创建任务
curl -X POST "http://localhost:8000/api/v1/task/ansible" \
  -H "Authorization: Bearer $TOKEN" \
  -F "name=test-task" \
  -F "type=1" \
  -F 'hostGroups={"web":[444,445]}' \
  -F "playbooks=@test.yml" \
  -F 'variables={"version":"1.0"}'
```

#### 场景 2：创建 Git 任务

```bash
curl -X POST "http://localhost:8000/api/v1/task/ansible" \
  -H "Authorization: Bearer $TOKEN" \
  -F "name=git-task" \
  -F "type=2" \
  -F "gitRepo=git@gitee.com:xxx/ansible-playbook.git" \
  -F 'hostGroups={"web":[444]}'
```

#### 场景 3：启动任务并查看实时日志

```bash
# 启动
curl -X POST "http://localhost:8000/api/v1/task/ansible/1/start" \
  -H "Authorization: Bearer $TOKEN"

# SSE 日志（浏览器或 curl）
curl -N "http://localhost:8000/api/v1/task/ansible/1/log/1" \
  -H "Authorization: Bearer $TOKEN"
```

#### 场景 4：视图分组管理

```bash
# 创建视图
curl -X POST "http://localhost:8000/api/v1/task/ansible/view" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"生产环境"}'

# 查询视图
curl "http://localhost:8000/api/v1/task/ansible/view/all" \
  -H "Authorization: Bearer $TOKEN"

# 按视图搜索任务
curl "http://localhost:8000/api/v1/task/ansible/query?viewName=生产环境" \
  -H "Authorization: Bearer $TOKEN"
```

#### 场景 5：定时任务

```bash
# 创建带 Cron 的任务
curl -X POST "http://localhost:8000/api/v1/task/ansible" \
  -H "Authorization: Bearer $TOKEN" \
  -F "name=scheduled-task" \
  -F "type=1" \
  -F "is_recurring=1" \
  -F "cron_expr=0 0 * * *" \
  -F 'hostGroups={"web":[444]}'
```

## 路由注册

```go
package task

func RegisterTaskRoutes(router *gin.RouterGroup) {
    // 任务路由...
    taskAnsibleCtrl := controller.NewTaskAnsibleController(service.NewTaskAnsibleService(common.GetDB()))
    router.GET("/task/ansiblelist", middleware.AuthMiddleware(), taskAnsibleCtrl.List)
    router.POST("/task/ansible", middleware.AuthMiddleware(), taskAnsibleCtrl.CreateTask)
    // ... 其他路由

    // 视图路由
    taskAnsibleViewCtrl := controller.NewTaskAnsibleViewController(service.NewTaskAnsibleViewService(common.GetDB()))
    router.POST("/task/ansible/view", middleware.AuthMiddleware(), taskAnsibleViewCtrl.Create)
    // ... 其他路由
}
```
