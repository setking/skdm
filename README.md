# SKDM — 跨平台桌面下载器

基于 **Wails v3 + Vue 3 + Go + aria2** 的跨平台桌面下载管理工具。

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
│  │  api/apiserver/Aria2Service  ← Wails Service 接口层      │   │
│  │  暴露 ~50 个方法给前端调用                                  │   │
│  └──────────────────────────┬───────────────────────────────┘   │
│                             │ 委托                              │
│  ┌──────────────────────────┴───────────────────────────────┐   │
│  │  pkg/aria2/Aria2Service  ← 核心引擎层                      │   │
│  │  - aria2c 进程生命周期管理                                  │   │
│  │  - WebSocket RPC 客户端 (arigo)                           │   │
│  │  - 事件订阅 + 定时轮询 → SQLite 持久化                      │   │
│  └──────────┬────────────────────┬──────────────────────────┘   │
│             │                    │                              │
│  ┌──────────┴──────┐  ┌─────────┴──────────┐                  │
│  │  pkg/store/     │  │  RPC (WebSocket)   │                  │
│  │  SQLite 持久化   │  │  localhost:6800    │                  │
│  └─────────────────┘  └─────────┬──────────┘                  │
└─────────────────────────────────┼───────────────────────────────┘
                                  │
┌─────────────────────────────────┴───────────────────────────────┐
│                    aria2c (独立进程)                              │
│  HTTP / FTP / BitTorrent / Magnet 多协议下载                     │
│  JSON-RPC 接口，通过 WebSocket 通信                               │
└─────────────────────────────────────────────────────────────────┘
```

### 架构分层

| 层 | 技术 | 职责 |
|---|------|------|
| **UI 层** | Vue 3 + Naive UI + UnoCSS | 界面渲染、用户交互、窗口控制 |
| **绑定层** | `@wailsio/runtime` (自动生成) | 前后端 IPC 通信，类型安全的方法调用 |
| **服务接口层** | `api/apiserver` (Go) | Wails Service 实现，暴露 ~50 个方法给前端 |
| **核心引擎层** | `pkg/aria2` (Go) | aria2c 进程管理、WebSocket RPC 连接、事件订阅、状态轮询 |
| **持久化层** | `pkg/store` (Go + SQLite) | 下载记录、事件日志、用户设置持久化 |
| **下载引擎** | aria2c (C++) | 实际下载执行，支持多协议、多线程 |

### 数据流

```
用户操作 → Vue 组件 → Wails Binding (JS) → IPC → Go Service → arigo RPC Client → WebSocket → aria2c

aria2c 事件 / 轮询 → arigo Client → Go Service → SQLite (pkg/store) → 前端查询展示
```

## 功能

### 已实现

- [x] aria2c 进程内嵌，应用启动自动拉起
- [x] WebSocket JSON-RPC 连接管理
- [x] 自定义无边框标题栏（拖拽、最小化/最大化/关闭）
- [x] 新建下载任务对话框（仿迅雷风格）
  - [x] 剪贴板自动粘贴链接
  - [x] 从 URL / magnet / ed2k 自动提取文件名
  - [x] 自定义保存路径和文件名
- [x] 侧边导航路由：下载中 / 未完成 / 已完成 / 回收站 / 设置
- [x] 下载中视图：active/waiting/paused 任务列表，实时进度/速度展示
- [x] 未完成视图：error/paused 任务，支持重试/恢复/移入回收站/永久删除
- [x] 已完成视图：下载完成的任务列表，支持重新下载和删除
- [x] 回收站视图：已删除任务，支持重新下载/永久删除/清空
- [x] 任务操作：暂停 / 恢复 / 继续 / 移除 / 永久删除
- [x] 设置页面：下载目录、并发数、连接数、分片数、断点续传、文件覆盖、限速
- [x] SQLite 持久化：下载记录、事件日志、用户设置
- [x] aria2 状态同步：启动时从 aria2 恢复活跃任务、完成/停止任务同步入库
- [x] 应用重启后记忆下载目录和设置
- [x] 后端服务进程优雅关闭
- [x] GitHub Actions CI/CD：自动构建 + Release 发布
- [x] 跨平台构建流水线（Windows NSIS/MSIX、macOS .app、Linux AppImage、Docker）

### 待实现

- [ ] BT 种子文件解析与下载
- [ ] 批量下载（导入链接列表）
- [ ] 全局状态统计（aria2.getGlobalStat）
- [ ] 剪贴板监听自动弹出下载对话框
- [ ] 托盘图标最小化到系统托盘

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
| 数据库 | SQLite (modernc.org/sqlite) | 1.50 |
| 下载引擎 | aria2c | 1.36+ |
| 构建 | Task (Taskfile) | 3.x |

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
├── Taskfile.yml                     # 顶层构建任务定义
│
├── backed/                          # Go 后端
│   ├── api/
│   │   └── apiserver/app.go         # Wails Service（接口层，~50 个方法）
│   ├── pkg/
│   │   ├── aria2/
│   │   │   ├── aria2.go             # aria2c 进程管理、事件订阅、状态轮询、DB 同步
│   │   │   ├── api.go               # RPC 方法封装（AddURI, Pause, TellStatus 等）
│   │   │   └── third_party/
│   │   │       ├── aria2c.exe       # 嵌入的 aria2 可执行文件
│   │   │       └── aria2.conf       # aria2 配置文件
│   │   └── store/
│   │       ├── store.go             # SQLite 数据库初始化、表迁移
│   │       ├── models.go            # 数据模型（DownloadRecord, Settings, EventRecord）
│   │       ├── download.go          # 下载记录 CRUD
│   │       ├── settings.go          # 设置读写（key-value + 类型化存取）
│   │       └── event.go             # 事件日志
│
├── frontend/                        # Vue 前端
│   ├── src/
│   │   ├── main.ts                  # Vue 入口
│   │   ├── App.vue                  # 根组件（Naive UI 配置）
│   │   ├── router/index.ts          # 路由表（/, /run, /complete, /trash, /settings）
│   │   ├── layout/index.vue         # 主布局（header + sidebar + main）
│   │   ├── components/
│   │   │   ├── header/index.vue     # 自定义标题栏 + 窗口控制 + 新建任务按钮
│   │   │   ├── menu/index.vue       # 侧边导航菜单（RouterLink + ionicons）
│   │   │   └── download-dialog/     # 新建下载任务对话框
│   │   ├── views/
│   │   │   ├── runtask/             # 下载中（active/waiting/paused）
│   │   │   ├── alltask/             # 未完成（error/paused，可恢复/重试/删除）
│   │   │   ├── completetask/        # 已完成
│   │   │   ├── trashtask/           # 回收站（可清空/恢复/永久删除）
│   │   │   ├── settings/            # 设置（下载目录、连接、限速等）
│   │   │   └── error/               # 403 / 404 错误页
│   │   └── assets/                  # 样式、字体等
│   ├── bindings/                    # Wails 自动生成的 JS binding
│   └── dist/                        # 前端构建产物（嵌入 Go 二进制）
│
├── build/                           # 构建配置
│   ├── config.yml                   # Wails 构建元数据
│   ├── appicon.png                  # 应用图标
│   ├── Taskfile.yml                 # 公共构建任务
│   └── {windows,darwin,linux}/      # 平台构建任务 + 打包配置
│
├── .github/workflows/               # CI/CD
│   └── build.yml                    # 自动构建 + Release 发布
│
└── bin/                             # 编译输出 (skdm.exe)
```

## 快速开始

### 前置条件

- Go 1.25+
- Node.js 20.19+ 或 22.12+
- pnpm
- [Task](https://taskfile.dev/)（推荐）

### 开发模式

```bash
# 安装前端依赖
cd frontend && pnpm install && cd ..

# 启动开发模式（热重载）
task dev
# 或：wails3 dev
```

### 构建

```bash
# 本地构建（Windows 上构建 exe）
task build

# Windows NSIS 安装包
task windows:package

# Docker 跨平台编译
task setup:docker
task build:docker

# 服务模式（无 GUI，仅 HTTP API）
task build:server
```

### 前端单独开发

```bash
cd frontend
pnpm dev          # Vite 开发服务器，端口 9245
pnpm build        # 生产构建
pnpm test:unit    # 运行单元测试
pnpm lint         # 代码检查
```

## 数据库

应用自动在系统缓存目录创建 SQLite 数据库（Windows：`%LocalAppData%/skdm/skdm.db`）。

### 表结构

| 表 | 用途 |
|----|------|
| `downloads` | 下载任务记录（gid, url, dir, filename, 进度, 状态, 错误信息等） |
| `download_events` | 下载事件日志（事件类型, 事件数据 JSON, 时间） |
| `settings` | 用户设置（key-value 存储） |

### 状态同步机制

- **aria2 事件订阅**：监听 Start/Pause/Complete/Error/Stop 事件，实时更新 DB
- **定时轮询**：每 3 秒调用 TellActive 更新下载进度和速度
- **启动同步**：应用启动时从 aria2 拉取所有任务状态，合并到 SQLite

## 开发说明

### RPC 通信

- aria2c 监听 `localhost:6800`，WebSocket 端点 `/jsonrpc`
- RPC Secret 硬编码为 `my-strong-secret-token-2026`
- Go 端使用 `github.com/siku2/arigo` 库封装 RPC 调用
- 前端通过 Wails 自动生成的 binding 调用，不直接连接 aria2

### 添加新的下载操作方法

1. 在 `backed/pkg/aria2/api.go` 中添加 RPC 方法实现
2. 在 `backed/api/apiserver/app.go` 中添加 Wails Service 包装方法
3. 重新运行 `wails3 dev` 或 `wails3 generate bindings` 更新前端 binding
4. 在前端组件中导入生成的 binding 函数调用

### 无边框窗口

- 窗口控制通过 `@wailsio/runtime` 的 `Window` API
- 标题栏拖拽：CSS `-webkit-app-region: drag`，按钮区域 `no-drag`
