---
name: theme-design
description: "AutoOps 企业级运维管理平台 UI 主题设计系统 — 双主题(经典白/深邃黑) + CSS 变量驱动 + 毛玻璃/渐变/动画。"
---

# 主题设计系统 (theme-design)

## 概述

AutoOps 前端采用 **CSS 变量 + data-theme 属性** 驱动的双主题系统，所有组件通过 CSS 变量自动适配深浅色模式，无需修改组件代码。

### 核心文件

| 文件 | 说明 |
|------|------|
| `web/src/utils/theme.js` | 主题工具函数（`initTheme()`/`setTheme(key)`） |
| `web/src/styles/theme.css` | 全局 CSS 变量定义 + 组件覆盖样式 |
| `web/src/main.js` | 启动时导入 theme.css 并调用 initTheme() |

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

## CMDB 页面样式规范

所有 CMDB 页面需遵循以下原则：

1. **不使用硬编码背景色** — 用 `var(--bg-page)`
2. **不使用固定颜色值** — 用 `var(--text-primary)` 等
3. **页面容器统一用法**：
   ```vue
   <div class="page-container">...</div>
   ```
4. **搜索区域**：用 `data-card` 类包裹
5. **卡片标题**：用 `.card-title-text` 或 `.section-title`
6. **表格**：保持 `border stripe` 即可，颜色由全局 CSS 控制

## 新增页面步骤

1. 在模板容器上不需要加背景色（全局 `--bg-page` 已处理）
2. 如需数据卡片，用 `data-card` 或 `stat-card` 类
3. 如需网格布局，用 `dashboard-grid dashboard-grid-{n}`
