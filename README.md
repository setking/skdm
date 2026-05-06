# SKDM — 跨平台桌面下载器

基于 **Wails v3 + Vue 3 + Go + aria2** 的跨平台桌面下载管理工具，当前处于活跃开发阶段。

## 技术架构

```
┌─────────────────────────────────────────────────────────────────┐
│                        Frontend (Vue 3)                         │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌───────────────┐  │
│  │  Header  │  │  Router  │  │  Pinia   │  │  Naive UI     │  │
│  │ 标题栏    │  │ 路由管理  │  │ 状态管理  │  │ 组件库        │  │
│  └──────────┘  └──────────┘  └──────────┘  └───────────────┘  │
│                         │                                       │
│               @wailsio/runtime (IPC)                            │
└─────────────────────────┼───────────────────────────────────────┘
                          │  Wails v3 Bindings (auto-generated)
┌─────────────────────────┼───────────────────────────────────────┐
│                   Backend (Go)                                   │
│  ┌──────────────────────┴──────────────────────────────────┐   │
│  │  api/aria2server/Aria2Service  ← Wails Service 接口层    │   │
│  │  - AddURI(uris, options) (arigo.GID, error)              │   │
│  │  - PauseAll() error                                       │   │
│  │  - ServiceStartup() / ServiceShutdown()                  │   │
│  └──────────────────────────┬───────────────────────────────┘   │
│                             │ 委托                              │
│  ┌──────────────────────────┴───────────────────────────────┐   │
│  │  pkg/aria2/Aria2Service  ← 核心引擎层                      │   │
│  │  - aria2c 进程生命周期管理                                  │   │
│  │  - WebSocket RPC 客户端 (arigo)                           │   │
│  └──────────────────────────┬───────────────────────────────┘   │
│                             │  WebSocket (localhost:6800)       │
└─────────────────────────────┼───────────────────────────────────┘
                              │
┌─────────────────────────────┴───────────────────────────────────┐
│                    aria2c (独立进程)                              │
│  - HTTP / FTP / BitTorrent / Magnet 多协议下载                  │
│  - JSON-RPC 接口，通过 WebSocket 通信                            │
└─────────────────────────────────────────────────────────────────┘
```

### 架构分层

| 层 | 技术 | 职责 |
|---|------|------|
| **UI 层** | Vue 3 + Naive UI + UnoCSS | 界面渲染、用户交互、窗口控制 |
| **绑定层** | `@wailsio/runtime` (自动生成) | 前后端 IPC 通信，类型安全的方法调用 |
| **服务接口层** | `api/aria2server` (Go) | Wails Service 实现，暴露 API 给前端 |
| **核心引擎层** | `pkg/aria2` (Go) | aria2c 进程管理、WebSocket RPC 连接 |
| **下载引擎** | aria2c (C++) | 实际下载执行，支持多协议、多线程 |

### 数据流

```
用户操作 → Vue组件 → Wails Binding (JS) → IPC → Go Service → arigo RPC Client
                                                                    │
                                                              WebSocket
                                                                    │
                                                              aria2c 进程
```

1. 用户在 Vue 界面输入下载链接，点击"立即下载"
2. `download-dialog` 调用自动生成的 JS binding `AddURI([url], options)`
3. Wails runtime 通过 IPC 将调用转发到 Go 后端的 `Aria2Service.AddURI()`
4. Go 层通过 `arigo` 库的 WebSocket 客户端发送 JSON-RPC 请求到 aria2c
5. aria2c 开始下载，返回 GID 标识任务

## 技术栈

### 前端

| 类别 | 选型 | 版本 |
|------|------|------|
| 框架 | Vue 3 (Composition API) | 3.5 |
| 语言 | TypeScript | 6.0 |
| 构建 | Vite | 8.0 |
| UI 组件库 | Naive UI | 2.44 |
| CSS 方案 | UnoCSS + Sass | 66.6 / 1.98 |
| 路由 | Vue Router | 5.0 |
| 状态管理 | Pinia | 3.0 |
| 图标 | Vicons (Ionicons 5) | 0.13 |
| 测试 | Vitest + jsdom | 4.1 |
| 代码检查 | ESLint + Oxlint | 10.1 / 1.57 |

### 后端

| 类别 | 选型 | 版本 |
|------|------|------|
| 语言 | Go | 1.25 |
| 桌面框架 | Wails v3 | 3.0.0-alpha.74 |
| RPC 客户端 | arigo | 0.3.0 |
| 下载引擎 | aria2c | 1.36+ |
| 构建 | Task (Taskfile) | - |

### 平台支持

| 平台 | 状态 | 打包方式 |
|------|------|---------|
| Windows | 原生构建 | NSIS / MSIX 安装包 |
| macOS | 原生构建 | .app 包 |
| Linux | 原生构建 | AppImage |
| Docker | 跨平台编译 / 服务模式部署 | Docker 镜像 |

## 项目结构

```
skdm/
├── main.go                          # Go 应用入口，Wails 服务注册
├── Taskfile.yml                     # 构建任务定义
│
├── backed/                          # Go 后端
│   ├── api/
│   │   ├── aria2server/api.go       # Aria2 Wails Service（接口层）
│   │   └── apiserver/app.go         # GreetService 示例
│   ├── pkg/
│   │   ├── aria2/
│   │   │   ├── aria2.go             # aria2c 进程生命周期管理
│   │   │   ├── api.go               # 下载操作（AddURI, PauseAll）
│   │   │   └── third_party/
│   │   │       ├── aria2c.exe       # 嵌入的 aria2 可执行文件
│   │   │       └── aria2.conf       # aria2 配置文件
│   │   └── configs/aria2.go         # 配置结构体定义
│   ├── cmd/                         # CLI 命令（预留）
│   └── internal/                    # 内部包（预留）
│
├── frontend/                        # Vue 前端
│   ├── src/
│   │   ├── main.ts                  # Vue 入口
│   │   ├── App.vue                  # 根组件（Naive UI 配置）
│   │   ├── router/index.ts          # 路由表
│   │   ├── layout/index.vue         # 主布局（header + sidebar + main）
│   │   ├── components/
│   │   │   ├── header/index.vue     # 自定义标题栏 + 窗口控制
│   │   │   ├── menu/index.vue       # 侧边导航菜单
│   │   │   └── download-dialog/     # 新建下载任务对话框
│   │   ├── views/
│   │   │   ├── alltask/             # 所有任务（待实现）
│   │   │   ├── runtask/             # 下载中（待实现）
│   │   │   ├── completetask/        # 已完成（待实现）
│   │   │   ├── trashtask/           # 回收站（待实现）
│   │   │   ├── settings/            # 设置（待实现）
│   │   │   └── error/               # 403 / 404 错误页
│   │   └── stores/                  # Pinia 状态管理
│   ├── bindings/                    # Wails 自动生成的 JS binding
│   │   └── changeme/backed/api/aria2server/
│   │       └── aria2service.js      # AddURI, PauseAll 类型声明
│   └── dist/                        # 构建产物（嵌入 Go 二进制）
│
├── build/                           # 构建配置
│   ├── config.yml                   # Wails 构建元数据
│   ├── appicon.png                  # 应用图标
│   └── {windows,darwin,linux}/      # 平台构建任务 + 打包配置
│
└── bin/                             # 编译输出
```

## 功能概览

### 已实现

- [x] aria2c 进程内嵌，应用启动自动拉起
- [x] WebSocket JSON-RPC 连接管理
- [x] 自定义无边框标题栏（拖拽、最小化/最大化/关闭）
- [x] 新建下载任务对话框（仿迅雷风格）
  - [x] 剪贴板自动粘贴链接
  - [x] 从 URL / magnet 自动提取文件名
  - [x] 自定义保存路径和文件名
- [x] 侧边导航路由（全部 / 下载中 / 已完成 / 回收站 / 设置）
- [x] 后端服务进程优雅关闭（Windows Kill / Unix SIGINT）
- [x] 跨平台构建流水线（Windows NSIS/MSIX、macOS .app、Linux AppImage、Docker）

### 待实现

- [ ] 任务列表视图（运行中 / 已完成 / 回收站）
- [ ] 下载进度、速度、剩余时间实时展示
- [ ] 任务操作（暂停 / 恢复 / 删除 / 重试）
- [ ] 批量下载（导入链接列表）
- [ ] BT 种子文件解析与下载
- [ ] 下载限速设置
- [ ] 全局状态统计（aria2.getGlobalStat）
- [ ] 应用设置页（下载目录、并发数、连接数等 aria2 配置）
- [ ] 剪贴板监听自动弹出下载对话框
- [ ] 托盘图标最小化到系统托盘

## 快速开始

### 前置条件

- Go 1.25+
- Node.js 20.19+ 或 22.12+
- pnpm（推荐）或 npm
- [Task](https://taskfile.dev/)（推荐，用于运行构建脚本）

### 开发模式

```bash
# 安装前端依赖
cd frontend && pnpm install && cd ..

# 启动开发模式（热重载）
task dev
# 或直接运行
wails3 dev
```

### 构建

```bash
# 本地构建
task build

# Windows NSIS 安装包
task windows:package

# Docker 跨平台编译
task setup:docker
task build:docker

# 服务模式（无 GUI，仅 HTTP API）
task build:server
```

### 前端开发（浏览器中调试）

```bash
cd frontend
pnpm dev          # Vite 开发服务器，端口 9245
pnpm build        # 生产构建
pnpm test:unit    # 运行单元测试
pnpm lint         # 代码检查
```

## 开发说明

### aria2 RPC 通信

- aria2c 默认监听 `localhost:6800`，通过 WebSocket 在 `/jsonrpc` 端点通信
- RPC Secret 硬编码为 `my-strong-secret-token-2026`（后续应改为可配置）
- Go 端使用 `github.com/siku2/arigo` 库封装 RPC 调用
- 前端通过 Wails 自动生成的 binding 调用，不直接连接 aria2

### 添加新的下载操作方法

1. 在 `backed/pkg/aria2/api.go` 中添加方法实现（调用 arigo client）
2. 在 `backed/api/aria2server/api.go` 中添加包装方法
3. 重新运行 `wails3 dev` 或 `wails3 generate` 更新前端 binding
4. 在前端组件中导入生成的 binding 函数调用

### 无边框窗口

- 窗口控制（最小化/最大化/关闭）通过 `@wailsio/runtime` 的 `Window` API
- 标题栏拖拽通过 CSS `-webkit-app-region: drag` 实现
- 按钮区域设置 `no-drag` 确保可点击
