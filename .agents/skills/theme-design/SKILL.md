---
name: theme-design
description: "AutoOps 企业级运维管理平台 UI 主题设计系统 — 双主题(经典白/深邃黑) + CSS 变量驱动 + 毛玻璃/渐变/动画。"
---

# 主题设计系统 (theme-design)

## 2026-10-24 全站控件规范

公共控件的样式以 `web/src/style/design-system.css` 为准；该文件在 Element Plus CSS 后导入，通过 `body.autoops-ui` 同时作用于页面和传送弹窗。本节优先于下文历史按钮渐变/悬浮效果示例。

- 按钮统一高度：默认 36px、小号/表格操作 32px、大号 44px，圆角 6px；禁止页面单独增加渐变、阴影和悬浮位移。
- 主动作使用 `type="primary"`，普通动作默认样式，删除使用 `danger`；表格操作统一轻量显示，危险动作保留红色。原生操作按钮优先改用 `el-button`，统计卡片和选项卡按钮保留其交互结构。
- 面板及弹窗圆角 12px；筛选区、表格、状态色由共享主题变量控制，支持浅色与深色。
- 页面通过布局的 `.app-page-content` 统一留白，桌面 24px、窄屏 14px。远程桌面与终端标记 `meta.uiLayout = "canvas"`，不套页面留白。
- 窄屏表格内横向滚动，固定操作列取消固定，避免遮挡；弹窗宽度不超过视口减 32px。
- 新模块直接复用公共规范，历史覆盖集中在此文件维护，不继续扩散到页面的 scoped CSS。

## 2026-11-04 菜单图标与对齐规范

- 可见路由通过 `web/src/utils/menuIcons.ts` 的 `menuIcon(path)` 配置本地 Element Plus 图标，内网显示不依赖 Iconify 在线服务；已有 SVG 组件与自定义 Iconfont 保持兼容。
- 所有模块和子菜单均保留图标占位，尺寸 18px、文字间距 10px；侧栏和弹出菜单行高统一 44px，同级文字起点一致。层级缩进继续由 Element Plus 管理，折叠后图标居中，横向导航保持顶栏高度。
- SidebarItem/NavMix 对动态菜单使用同一解析器；未知菜单使用 Menu 图标，不能因缺图标让文字左移。
- 图标类型使用 Vue Component，兼容函数式 SVG 与 Element Plus 组件。公共样式位于 design-system.css，不在各页面重复覆盖。
- API 元数据的默认名称在 `api/pkg/db/menu_icons.go`，与前端路径映射保持一致；迁移只补图标，保留自选图标、URL、排序、状态、权限值和角色关联。对照测试由 `go -C api test ./pkg/db` 执行。

## 概述

AutoOps 前端采用 **CSS 变量 + data-theme 属性** 驱动的双主题系统，所有组件通过 CSS 变量自动适配深浅色模式，无需修改组件代码。

### 核心文件

| 文件 | 说明 |
|------|------|
| `web/src/utils/theme.ts` | 主题工具函数（`initTheme()`/`setTheme(key)`） |
| `web/src/style/theme.css` | 全局 CSS 变量定义 + 组件覆盖样式 |
| `web/src/main.ts` | 启动时导入 theme.css 并调用 initTheme() |

### 使用方式

```js
// 切换主题
import { setTheme } from '@/utils/theme'
setTheme('dark')   // 深邃黑
setTheme('light')  // 经典白

// 初始化（已内嵌在 main.js）
```

## 主题配置

### 经典白 (light) — 默认

```css
--bg-page: #f0f2f5;
--bg-card: #ffffff;
--bg-sidebar: #001529;
--bg-header: #ffffff;
--primary: #1677ff;
--text-primary: #1d2129;
--text-regular: #4e5969;
--text-secondary: #86909c;
--border: #e5e6eb;
```

### 深邃黑 (dark)

```css
--bg-page: #0f1419;
--bg-card: #1e2937;
--bg-sidebar: #0d1b2a;
--bg-header: #1a2332;
--primary: #36a3ff;
--text-primary: #f5f7fa;
--text-regular: #c9cdd4;
--text-secondary: #86909c;
--border: #333f4e;
```

## 全局可用的 CSS 类

| 类名 | 说明 | 使用场景 |
|------|------|---------|
| `data-card` | 数据卡片容器 | 面板/区块 |
| `glass-card` | 毛玻璃卡片 | 需浮层效果的区域 |
| `stat-card` | 统计数字卡片 | 指标展示 |
| `dashboard-grid` / `-4`/`-3`/`-2` | CSS Grid 布局 | 多列等宽网格 |
| `chart-container` / `-sm`/`-lg` | ECharts 图表容器 | 图表外层 |
| `section-title` + `title-bar` | 区域标题 | 模块标题行 |
| `animate-fade-in-up` | 入场动画 | 模块首次加载 |
| `status-dot` / `.success`/`.warning`/`.danger`/`.info` | 状态圆点 | 健康状态指示 |

## CSS 变量覆盖范围

所有 Element Plus 组件通过 CSS 变量自动适配主题：

| 组件 | 关键变量 |
|------|---------|
| `el-table` | `--bg-card-alt`, `--text-regular`, `--border` |
| `el-card` | `--bg-card`, `--border` |
| `el-dialog` | `--bg-card`, `--border` |
| `el-form` / `el-input` | `--text-regular`, `--primary` |
| `el-menu` | `--menu-text`, `--menu-active`, `--menu-sub-bg` |
| `el-pagination` | `--text-regular` |
| `el-tag` | `--bg-tag`, `--text-regular` |

## 资产与权限列表标准

主机、物理机、网络设备及授权列表使用同一套资产列表样式。参考实现：

- `web/src/views/cmdb/networkDevice.vue`
- `web/src/views/cmdb/physicalMachine.vue`
- `web/src/views/cmdb/Host/CmdbHostTable.vue`
- `web/src/views/cmdb/assetPermission.vue`
- `web/src/views/cmdb/components/AssetConnections.vue`

### 页面层级

1. 外层使用 `asset-page`，桌面边距 24px、手机 14px。
2. 顶部 `asset-page-header` 包含小型分类 `asset-eyebrow`、唯一 h1、简短用途说明。右侧 `asset-header-count` 只展示 API 返回的实际总数，并注明统计范围，如「匹配资产」。不可把当前页数据伪装成全局统计。
3. 列表容器使用 `el-card` 的 `shadow="never" class="asset-surface"`。使用 16px 圆角、细边框和轻阴影，避免彩色重阴影和多层嵌套卡片。
4. 筛选区 `asset-filters` 使用次级背景；新增、刷新等列表操作置于 `asset-toolbar`。只给主操作使用实色主色按钮。

### 表格内容

- 使用 `class="asset-table" row-key="id"`。以轻水平分隔线和悬停背景分隔行，避免默认 `border stripe` 产生密集网格。
- 表头字号 12px，单元格上下 15px、左右 16px。以两层信息减少横向列数：资产名 + 序列号，地址 + 用途，品牌型号 + 配置。
- 资产主体使用 `asset-identity`、`asset-avatar`、`asset-cell`。主信息字体加重，次信息使用 `asset-muted`。地址和标识使用 `asset-mono`。
- 名称列可固定在左边，操作列固定在右边；其余内容按最小宽度滚动，不压缩到无法辨认。
- 状态必须带文字。监控心跳、SSH 认证和各协议可达性分别展示，不能用 SSH 离线状态禁用 VNC 或 RDP。
- 空态使用 `el-empty`，说明没有匹配数据并给出用户能采取的下一步。加载使用 `v-loading`，请求失败必须给出反馈。
- 行操作最多两个直接入口和一个「更多」菜单。连接用浅主色按钮，编辑用文字按钮；删除收进菜单并要求明确确认。
- 分页使用 `asset-pagination`，变更每页数量时重置页码。不要用 CSS 隐藏数据列作为手机适配。

### 抽屉与表单

- 连接、配置等资产上下文功能优先使用抽屉，宽度使用 `min(760px, 100vw)`。
- 弹窗宽度使用 `min(720px, 94vw)`，两列表单使用 `asset-form-grid`，768px 以下自动变为单列。
- `connection-card` 展示协议、独立地址、账户、启用和只读状态。协议名称必须可读，不仅依赖颜色。
- 权限操作使用 `permission-action-group` 分组展示：资产信息、远程访问、文件管理、网络配置和完整管理。接口权限是实际约束，UI 应使用逐资产权限结果展示入口。
- 代码、配置及差异使用 `asset-code` 等宽滚动容器；将接口文本作为文本渲染，不使用 `v-html`。
- 密码、私钥和会话票据不写入页面 URL、日志或示例。凭据用 ID 引用；新 SSH 连接需获取并核对主机密钥指纹。

### 主题与可访问性

- 颜色优先使用 `--bg-page`、`--bg-card`、`--bg-card-alt`、`--text-primary`、`--text-secondary`、`--border`、`--primary` 等变量，同时兼容 `data-theme="dark"` 与 `html.dark`。
- 远程终端/桌面显示区域可固定深色，这是协议画布的独立显示需要；外围工具栏继续适配全局主题。
- 小于 1200px 时主机分组栏改为顶部区域；小于 768px 时页面与表单缩小间距。表格保留横向滚动。
- 图标按钮必须提供 tooltip 或 aria-label；键盘焦点清晰可见。遵循 `prefers-reduced-motion`，禁用不必要动画。

### 新增页面骨架

```vue
<div class="asset-page">
  <header class="asset-page-header">
    <div>
      <span class="asset-eyebrow">ASSET INVENTORY</span>
      <h1>资产名称</h1>
      <p>说明此页面的用途。</p>
    </div>
    <div class="asset-header-count"><strong>{{ total }}</strong><span>匹配资产</span></div>
  </header>
  <el-card shadow="never" class="asset-surface">
    <div class="asset-filters"><!-- 搜索条件 --></div>
    <div class="asset-toolbar"><!-- 列表操作 --></div>
    <el-table :data="list" row-key="id" class="asset-table" v-loading="loading">
      <!-- 用 asset-cell 展示主次信息，操作列最多两个直接入口 -->
      <template #empty><el-empty description="暂无匹配资产" /></template>
    </el-table>
    <div class="asset-pagination"><!-- 分页 --></div>
  </el-card>
</div>
```

### 预览参考

以下截图使用模拟资产数据，展示列表标准在明暗主题下的实际浏览器效果：

- [经典白列表](../../../docs/images/cmdb-asset-table-light.png)
- [深邃黑列表](../../../docs/images/cmdb-asset-table-dark.png)

### 验证

修改后执行 `cd web && pnpm build`。有浏览器环境时检查明暗主题、长名称、空数据、加载失败、键盘操作、窄屏和横向滚动；远程功能应检查关闭/重连是否释放 WebSocket、终端与键盘监听。
