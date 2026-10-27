<h1>vue-pure-admin精简版（非国际化版本）</h1>

[![license](https://img.shields.io/github/license/pure-admin/vue-pure-admin.svg)](LICENSE)

**中文** | [English](./README.en-US.md)

## AutoOps 运维功能

本目录是 AutoOps 的 Vue 3 / Element Plus 前端。K8s 模块包含 Ceph 资源维护、迁移和备份恢复，以及 HPA / VPA / KPA 弹性伸缩管理；Ansible 模块提供内置 kubeasz 集群运维入口。

- [Ceph 基础使用教程](../docs/2026-08-16-ceph-usage-tutorial.md)
- [Ceph 维护中心：配置、权限、迁移和备份恢复](../docs/2026-10-01-ceph-maintenance.md)
- [HPA / VPA / KPA：配置、权限与冲突检查](../docs/2026-10-01-k8s-autoscaling.md)
- [HPA / VPA / KPA：真实集群安装与验收](../docs/2026-10-02-autoscaling-live-validation.md)
- [内置 kubeasz：组件准备、节点维护与执行记录](../docs/2026-10-01-kubeasz-system-operations.md)

本地开发：`pnpm install` 后执行 `pnpm dev`；构建使用 `pnpm build`。后端接口地址按项目已有 `.env.development` 配置。维护动作必须连接已配置权限的真实集群；页面构建成功不代表目标集群的控制器、工具镜像和存储前提均已满足。

## 介绍

精简版是基于 [vue-pure-admin](https://github.com/pure-admin/vue-pure-admin) 提炼出的架子，包含主体功能，更适合实际项目开发，打包后的大小在全局引入 [element-plus](https://element-plus.org) 的情况下仍然低于 `2.3MB`，并且会永久同步完整版的代码。开启 `brotli` 压缩和 `cdn` 替换本地库模式后，打包大小低于 `350kb`

## 版本选择

当前是非国际化版本，如果您需要国际化版本 [请点击](https://github.com/pure-admin/pure-admin-thin/tree/i18n)

## 配套视频

[点我查看 UI 设计](https://www.bilibili.com/video/BV17g411T7rq)  
[点我查看快速开发教程](https://www.bilibili.com/video/BV1kg411v7QT)

## 配套保姆级文档

[点我查看 vue-pure-admin 文档](https://pure-admin.cn/)  
[点我查看 @pureadmin/utils 文档](https://pure-admin-utils.netlify.app)

## 高级服务

[点我查看详情](https://pure-admin.cn/pages/service/)

## 预览

[查看预览](https://pure-admin-thin.netlify.app/#/login)

## 维护者

[xiaoxian521](https://github.com/xiaoxian521)

## ⚠️ 注意

精简版不接受任何 `issues` 和 `pr`，如果有问题请到完整版 [issues](https://github.com/pure-admin/vue-pure-admin/issues/new/choose) 去提，谢谢！

## 许可证

[MIT © 2020-present, pure-admin](./LICENSE)
