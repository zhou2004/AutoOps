---
name: understand-project
description: "初始化或理解 AutoOps 项目架构规范。强制遵守前后端一体同步执行的工作流，适用于任何新增功能、修改逻辑的指令。"
---

# AutoOps Full-Stack Synchronization Workflow

This skill enforces the project's monorepo structure and a strict synchronized development workflow for the AutoOps project.

## 1. Project Structure
Always respect the following root-level directories and their separation of concerns:
- **`api/`**: 后端项目根目录 (Backend Project Root)
- **`web/`**: 前端项目根目录 (Frontend Project Root)
- **`docs/`**: 文档目录 (Documentation)
- **`docker/`**: Docker 部署目录 (Docker Deployment)
- **`k8s/`**: K8s 部署目录 (Kubernetes Deployment)

## 2. Synchronous Development Workflow
Whenever executing instructions, implementing features, or fixing bugs, you MUST follow this synchronized full-stack workflow:

- **Step 1: 评估双端影响 (Analyze Both Ends)**
  Assess how the user's request impacts both the frontend (`web/`) and the backend (`api/`).
- **Step 2: 同步修改 (Synchronous Modification)**
  You must edit both frontend and backend synchronously to ensure they match. Do not edit only the frontend or only the backend unless explicitly instructed to ignore one side.
  - Modify backend code (routes, controllers, models, etc. in `api/`).
  - Modify frontend code (views, components, API integration, etc. in `web/`).
- **Step 3: 一致性保证 (Ensure Consistency)**
  Verify that API requests and responses, data models, and business logic match perfectly between `api/` and `web/`.
- **Step 4: 基础设施同步 (Infrastructure & Docs Sync)**
  If the feature requires configuration or deployment updates, update `docker/`, `k8s/`, and `docs/` accordingly.