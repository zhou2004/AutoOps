---
name: backend-router-layer
description: "后端 API 路由注册代码组织结构规范。适用于新增、修改后端接口以及处理路由映射的任务，强制按照模块化进行路由分离。"
---

# 后端路由注册架构规范 (Backend Router Layer)

为了保持 Go 后端项目路由体系的清晰且可维护，本 Skill 规定了在处理后端（位于 `api/`）的新增接口时的注册与封装规范。

## 1. 核心约束：统一下沉至业务路由模块
- **按模块注册**：后端 API 接口统一定义并注册在 `api/router/` 目录下的相关业务模块里。
- **示例结构**：关于 CMDB 的接口必须要追加注册在 `api/router/cmdb/cmdb.go` 中，并严格对照绑定的 `Controller` 和 `Service`。
- **禁止行为**：严禁将具体的路由定义散落在 `main.go` 或者非 `router` 控制域的代码内；严禁将匿名函数直接挂载到路由上处理复杂业务逻辑（必须转发至 Controller）。

## 2. 标准接驳工作流 (Implementation Workflow)
当在后端新增功能（如添加文件删除接口）时，需严格执行以下流转层次：
1. **服务提供（Service）**：
   在 `api/api/<module>/service/` (例如 `api/api/cmdb/service/`) 下完善对应接口的核心业务逻辑与数据操作。
2. **逻辑控制（Controller）**：
   在 `api/api/<module>/controller/` (例如 `api/api/cmdb/controller/`) 封装请求参数校验、数据返回封装。
3. **路由接管（Router Layer）**：
   最后将这个 Handler 通过对应的 HTTP 方法注册到 **专属的 Router 文件** 中。例如编辑 `api/router/cmdb/cmdb.go` :
   ```go
   import "dodevops-api/api/cmdb/controller"

   func InitCmdbRouter(router *gin.RouterGroup) {
      // 模块分组
      cmdbGroup := router.Group("/cmdb")
      {
          // 关联到对应的 Controller 函数上
          cmdbGroup.GET("/hostssh/files", controller.NewCmdbHostSSHController(...).FileList)
      }
   }
   ```

## 3. 质量验收清单
- [ ] 路由地址及 HTTP 方法映射是否准确对应并添加在合适的 `router/<module>/<module>.go` 文件中？
- [ ] 对应路由是否存在规范的跨域、鉴权相关的中间件包裹？
- [ ] 业务逻辑是否已成功下移到 Controller 和 Service 而没有停留在路由定义体内？