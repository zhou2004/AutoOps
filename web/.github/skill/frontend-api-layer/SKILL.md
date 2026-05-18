---
name: frontend-api-layer
description: "前端网络请求与 API 接口层代码组织规范。适用于为前端新增功能或修改现有接口请求的任务，强制要求分离接口定义与组件视图代码。"
---

# 前端 API 接口层架构规范 (Frontend API Layer)

为了保持前端代码的复用性和整洁性，本 Skill 规定了在处理前端（位于 `web/`）的数据请求时，必须遵循的集中式管理的开发工作流。

## 1. 核心约束：拒绝在 Vue 原生组件中直书 Request
- **必须分离**：所有的业务请求必须统一定义在 `src/api/` 目录下，按业务模块进行文件拆分。
- **示例结构**：CMDB 相关的接口请求都必须被定义在 `web/src/api/cmdb.js` 或者 `web/src/api/cmdb/index.js` 等专用文件中。
- **禁止行为**：严禁在 `*.vue` 等组件内部随意直接内联编写或散落使用 `request({ ... })` 或 `axios(...)` 请求。

## 2. 标准接驳工作流 (Implementation Workflow)
1. **API 层定义**：
   在 `src/api/<相关模块>.js`（例如 `cmdb.js`、`user.js`）文件中定义并导出请求函数。
   ```javascript
   import request from '@/utils/request'

   export function getCmdbHostFiles(params) {
     return request({
       url: '/api/v1/cmdb/hostssh/files',
       method: 'get',
       params
     })
   }
   ```
2. **UI 组件层引入与调用**：
   在需要使用该请求的 Vue 组件 (`*.vue`) 中，通过 `import` 引入封装好的接口函数，进行异步调用处理视图状态。
   ```javascript
   import { getCmdbHostFiles } from '@/api/cmdb'
   
   // ... 
   const res = await getCmdbHostFiles({ hostId: 1, path: '/' })
   ```

## 3. 质量验收清单
- [ ] 当前涉及的请求是否已经从 Vue 视图层抽离抽离到了 `web/src/api/` 目录下对应的业务文件？
- [ ] 对应模块文件的相对路径定义是否符合标准要求（如 cmdb 接口对应 `src/api/cmdb.js`）？
- [ ] 在调用层，异常处理（catch/finally）和基础参数封装是否遵循标准？