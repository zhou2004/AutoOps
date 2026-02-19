# Dodevops-Web 项目架构说明文档

## 1. 项目概述

Dodevops-Web 是一个基于 Vue 3 的前端项目，旨在提供 DevOps 相关功能的 Web 界面。该项目为 DevOps 工程师、系统管理员和开发人员提供了一个统一的可视化平台，用于管理配置、监控、Kubernetes、工单、CMDB 等 DevOps 相关操作。

## 2. 技术栈

- **核心框架**: Vue 3.5.17
- **状态管理**: Vuex 4.0.2
- **路由管理**: Vue Router 4.5.1
- **UI 框架**: Element Plus 2.10.2
- **图标库**: @element-plus/icons-vue 2.3.1
- **图表库**: ECharts 5.1.2
- **HTTP 客户端**: Axios 0.27.2
- **终端模拟器**: xterm 5.5.0 及相关插件
- **构建工具**: Vue CLI Service 5.0.0
- **样式预处理器**: Less 4.1.1, Sass 1.89.2
- **代码规范**: ESLint 7.32.0, Babel 7.12.16

## 3. 项目结构

```
src/
├── api/              # API 接口管理
├── assets/           # 静态资源文件
├── components/       # 全局公共组件
├── permission/       # 权限控制模块
├── router/           # 路由配置
├── store/            # Vuex 状态管理
├── utils/            # 工具类函数
├── views/            # 页面视图组件
├── App.vue           # 根组件
└── main.js           # 入口文件
```

### 3.1 目录详解

#### 3.1.1 api 目录
该目录用于管理所有后端 API 接口，按照功能模块划分:
- [system.js](file:///Users/apple/Desktop/git/dodevops-web/src/api/system.js) - 系统管理相关接口
- [cmdb.js](file:///Users/apple/Desktop/git/dodevops-web/src/api/cmdb.js) - CMDB 管理相关接口
- [config.js](file:///Users/apple/Desktop/git/dodevops-web/src/api/config.js) - 配置中心相关接口
- [index.js](file:///Users/apple/Desktop/git/dodevops-web/src/api/index.js) - API 入口文件，整合各模块接口

#### 3.1.2 assets 目录
存放静态资源文件，如 CSS 样式文件、图片等。

#### 3.1.3 components 目录
存放全局可复用的 Vue 组件:
- [HeadImage.vue](file:///Users/apple/Desktop/git/dodevops-web/src/components/HeadImage.vue) - 头像组件
- [Menu.vue](file:///Users/apple/Desktop/git/dodevops-web/src/components/Menu.vue) - 菜单组件
- [Tags.vue](file:///Users/apple/Desktop/git/dodevops-web/src/components/Tags.vue) - 标签组件

#### 3.1.4 permission 目录
实现前端权限控制:
- [Authority.js](file:///Users/apple/Desktop/git/dodevops-web/src/permission/Authority.js) - 权限指令实现
- [index.js](file:///Users/apple/Desktop/git/dodevops-web/src/permission/index.js) - 权限插件入口

#### 3.1.5 router 目录
路由配置管理，采用模块化组织:
- [router.js](file:///Users/apple/Desktop/git/dodevops-web/src/router/router.js) - 主路由配置文件
- [system.js](file:///Users/apple/Desktop/git/dodevops-web/src/router/system.js) - 系统管理路由
- [cmdb.js](file:///Users/apple/Desktop/git/dodevops-web/src/router/cmdb.js) - CMDB 管理路由
- [k8s.js](file:///Users/apple/Desktop/git/dodevops-web/src/router/k8s.js) - Kubernetes 管理路由
- [config.js](file:///Users/apple/Desktop/git/dodevops-web/src/router/config.js) - 配置中心路由

#### 3.1.6 store 目录
Vuex 状态管理配置:
- [index.js](file:///Users/apple/Desktop/git/dodevops-web/src/store/index.js) - Vuex store 配置
- [mutations.js](file:///Users/apple/Desktop/git/dodevops-web/src/store/mutations.js) - 状态变更处理

#### 3.1.7 utils 目录
通用工具函数:
- [authority.js](file:///Users/apple/Desktop/git/dodevops-web/src/utils/authority.js) - 权限检查工具
- [common.js](file:///Users/apple/Desktop/git/dodevops-web/src/utils/common.js) - 通用工具函数
- [request.js](file:///Users/apple/Desktop/git/dodevops-web/src/utils/request.js) - Axios 封装
- [storage.js](file:///Users/apple/Desktop/git/dodevops-web/src/utils/storage.js) - 本地存储工具

#### 3.1.8 views 目录
页面组件按照功能模块组织:
- **dashboard/** - 仪表盘模块
- **system/** - 系统管理模块（用户、角色、菜单等）
- **cmdb/** - CMDB 管理模块（主机、数据库等）
- **K8s/** - Kubernetes 管理模块
- **configcenter/** - 配置中心模块
- **monitor/** - 监控模块
- **work/** - 工单系统模块
- **app/** - 应用管理模块

## 4. 核心功能模块

### 4.1 权限控制系统
项目实现了基于指令的权限控制机制:
1. 通过 [Authority.js](file:///Users/apple/Desktop/git/dodevops-web/src/permission/Authority.js) 实现自定义权限指令 `v-authority`
2. 权限检查通过 [authority.js](file:///Users/apple/Desktop/git/dodevops-web/src/utils/authority.js) 工具函数完成
3. 路由级别权限控制在 [router.js](file:///Users/apple/Desktop/git/dodevops-web/src/router/router.js) 中实现

### 4.2 HTTP 请求封装
通过 [request.js](file:///Users/apple/Desktop/git/dodevops-web/src/utils/request.js) 对 Axios 进行统一封装:
1. 统一设置请求头和基础 URL
2. 实现请求拦截和响应拦截
3. 自动处理认证 Token
4. 统一错误处理和用户登出逻辑

### 4.3 路由系统
采用模块化路由配置:
1. 各功能模块独立配置路由
2. 在主路由文件中合并各模块路由
3. 实现导航守卫进行权限验证

### 4.4 状态管理
使用 Vuex 进行全局状态管理:
1. 用户信息存储
2. 权限列表管理
3. 全局配置状态

## 5. 构建与部署

### 5.1 构建配置
项目使用 Vue CLI 进行构建，配置文件为 [vue.config.js](file:///Users/apple/Desktop/git/dodevops-web/vue.config.js):
- 关闭生产环境 sourceMap 以提升性能
- 设置开发服务器端口为 8080
- 配置 API 代理，将 `/api/v1` 请求代理到 `http://127.0.0.1:8000`

### 5.2 开发环境
- Node.js 12+
- npm 11.4.2
- Vue CLI Service 5.0.0

### 5.3 常用命令
- `npm run serve` - 启动开发服务器
- `npm run build` - 构建生产环境版本
- `npm run lint` - 代码检查

## 6. 设计模式与架构特点

### 6.1 模块化设计
项目按照功能模块划分目录结构，便于维护和扩展。

### 6.2 组件化开发
采用 Vue 组件化开发模式，提高代码复用性和可维护性。

### 6.3 单一职责原则
各文件和模块职责明确，如 API 管理、路由配置、状态管理等分离。

### 6.4 权限控制
实现多层级权限控制，包括路由级和组件级权限验证。

## 7. 扩展性建议

1. 可以进一步完善 TypeScript 支持以提高代码可维护性
2. 可以引入单元测试框架提高代码质量
3. 可以考虑使用微前端架构支持更大规模的团队开发
