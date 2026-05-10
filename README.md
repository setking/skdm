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
│  │  internal/pkg/server/Config  ← Wails Service 桥接层      │   │
│  │  暴露 ~20 个方法给前端，委托到 GenericARIA2Server          │   │
│  └──────────────────────────┬───────────────────────────────┘   │
│                             │                                    │
│  ┌──────────────────────────┴───────────────────────────────┐   │
│  │  internal/pkg/server/GenericARIA2Server  ← 核心引擎层     │   │
│  │  - aria2c 进程生命周期管理（启动/监控/重启）                 │   │
│  │  - WebSocket RPC 连接管理 (arigo)                         │   │
│  │  - 事件订阅 + 状态轮询 + 前端推送                           │   │
│  └───────┬──────────────────────────────┬───────────────────┘   │
│          │ 注入 Aria2ClientProvider      │ 注入 store.Factory   │
│  ┌───────┴──────────────────────────────┴───────────────────┐   │
│  │         internal/apiserver/controller/v1/  ← Controller 层│   │
│  │  ┌──────────┐ ┌────────┐ ┌──────────┐ ┌───────────────┐ │   │
│  │  │ Download │ │ Event  │ │ Settings │ │     Sys       │ │   │
│  │  │Controller│ │Control.│ │Controller│ │  Controller   │ │   │
│  │  └────┬─────┘ └───┬────┘ └────┬─────┘ └───────┬───────┘ │   │
│  └───────┼───────────┼───────────┼─────────────────┼─────────┘   │
│          │           │           │                 │              │
│  ┌───────┴───────────┴───────────┴─────────────────┴─────────┐   │
│  │     internal/apiserver/service/v1/  ← Service 层（透传）    │   │
│  └────────────────────────┬──────────────────────────────────┘   │
│                           │                                      │
│  ┌────────────────────────┴──────────────────────────────────┐   │
│  │     internal/apiserver/store/  ← Store 接口 + SQLite 实现  │   │
│  │     SQLite (modernc.org/sqlite, 纯 Go, 无需 CGO)          │   │
│  └────────────────────────────────────────────────────────────┘   │
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐   │
│  │                RPC (WebSocket) localhost:6800               │   │
│  └────────────────────────┬───────────────────────────────────┘   │
└────────────────────────────┼──────────────────────────────────────┘
                             │
┌────────────────────────────┴──────────────────────────────────────┐
│                    aria2c (独立进程)                               │
│  HTTP / FTP / BitTorrent / Magnet 多协议下载                      │
│  JSON-RPC 接口，通过 WebSocket 通信                                │
└───────────────────────────────────────────────────────────────────┘
```

### 架构分层

| 层             | 技术                                                  | 职责                                                           |
| -------------- | ----------------------------------------------------- | -------------------------------------------------------------- |
| **UI 层**      | Vue 3 + Naive UI + UnoCSS                             | 界面渲染、用户交互、窗口控制                                   |
| **绑定层**     | `@wailsio/runtime` (自动生成)                         | 前后端 IPC 通信，类型安全的方法调用                            |
| **桥接层**     | `internal/pkg/server/Config` (Go)                     | Wails Service 实现，委托到 GenericARIA2Server，nil-safety 守卫 |
| **核心引擎层** | `internal/pkg/server/GenericARIA2Server` (Go)         | aria2c 进程管理、WebSocket RPC 连接、事件订阅、状态轮询        |
| **控制器层**   | `internal/apiserver/controller/v1/` (Go)              | 业务逻辑编排，RPC 调用 + 数据库读写 + 前端事件推送             |
| **服务层**     | `internal/apiserver/service/v1/` (Go)                 | Store 接口透传（薄层）                                         |
| **持久化层**   | `internal/apiserver/store/` + `pkg/db/` (Go + SQLite) | 下载记录、事件日志、用户设置持久化，启动自动建表               |
| **下载引擎**   | aria2c (C++)                                          | 实际下载执行，支持多协议、多线程                               |

### 数据流

```
用户操作 → Vue 组件 → Wails Binding (JS) → IPC → Config 桥接 → GenericARIA2Server → Controller → Service → Store → SQLite
                                                                        ↓
                                                                  arigo RPC Client → WebSocket → aria2c

主动推送（事件驱动）:
  aria2c 状态变更 → GenericARIA2Server 事件回调 → Controller → SQLite 写入（持久化） → Wails Event → Pinia Store → UI 响应式更新
  aria2c 进度更新 → GenericARIA2Server 轮询（3s，不写 SQLite） → Wails Event → Pinia Store → UI 响应式更新

全量兜底同步（60s 间隔）:
  GenericARIA2Server → aria2 全量拉取 → Controller → SQLite 同步 + Wails Event 全量推送

自动更新检查（启动时执行一次）:
  SysController.CheckForUpdateOnStartup() → GitHub API → Wails Event("update-check") → Pinia updateStore → 设置页面展示
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
  - [x] 重复 URL 检测（队列中阻止 / 已完成确认重下 / 失败自动覆盖）
  - [x] 下载错误中文友好提示（403/404/500/超时/DNS 等）
  - [x] 提交中取消（可中止正在进行的添加请求）
- [x] 侧边导航路由：下载中 / 未完成 / 已完成 / 回收站 / 设置
- [x] 下载中视图：active/waiting/paused 任务列表，实时进度/速度展示
- [x] 未完成视图：error/paused 任务，支持重试/恢复/移入回收站/永久删除
- [x] 已完成视图：下载完成的任务列表，支持重新下载和删除
- [x] 回收站视图：已删除任务，支持重新下载/永久删除/清空
- [x] 任务操作：暂停 / 恢复 / 继续 / 移除 / 永久删除
- [x] 设置页面：下载目录、并发数、连接数、分片数、断点续传、文件覆盖、限速
- [x] SQLite 持久化：下载记录、事件日志、用户设置
- [x] aria2 状态同步（事件驱动）：状态变更写 SQLite + 推送前端，进度直接推送前端
- [x] Pinia Store 响应式状态管理，替代前端轮询
- [x] 应用重启后记忆下载目录和设置
- [x] 后端服务进程优雅关闭
- [x] GitHub Actions CI/CD：自动构建 + NSIS 安装器打包 + Release 发布
- [x] 跨平台构建流水线（Windows NSIS/MSIX、macOS .app、Linux AppImage、Docker）
- [x] 系统托盘：图标 + 右键菜单（新建任务/打开主面板/退出程序）
  - [x] 左键单击托盘图标：窗口居中显示/隐藏
  - [x] 主窗口关闭时最小化到托盘，不退出程序
  - [x] 主窗口隐藏时新建任务弹窗独立居中展示
- [x] aria2c 后台静默运行（隐藏命令行窗口）

### 待实现

- [ ] BT 种子文件解析与下载
- [ ] 批量下载（导入链接列表）
- [ ] 全局状态统计（aria2.getGlobalStat）
- [ ] 剪贴板监听自动弹出下载对话框

## 技术栈

### 前端

| 类别      | 选型                    | 版本        |
| --------- | ----------------------- | ----------- |
| 框架      | Vue 3 (Composition API) | 3.5         |
| 语言      | TypeScript              | 6.0         |
| 构建      | Vite                    | 8.0         |
| UI 组件库 | Naive UI                | 2.44        |
| CSS 方案  | UnoCSS + Sass           | 66.6 / 1.98 |
| 路由      | Vue Router              | 5.0         |
| 状态管理  | Pinia                   | 3.0         |
| 图标      | Vicons (Ionicons 5)     | 0.13        |
| 测试      | Vitest + jsdom          | 4.1         |
| 代码检查  | ESLint + Oxlint         | 10.1 / 1.57 |

### 后端

| 类别       | 选型                        | 版本           |
| ---------- | --------------------------- | -------------- |
| 语言       | Go                          | 1.25           |
| 桌面框架   | Wails v3                    | 3.0.0-alpha.74 |
| RPC 客户端 | arigo                       | 0.3.0          |
| 数据库     | SQLite (modernc.org/sqlite) | 1.50           |
| 下载引擎   | aria2c                      | 1.36+          |
| 构建       | Task (Taskfile)             | 3.x            |

### 平台支持

| 平台    | 状态                      | 打包方式           |
| ------- | ------------------------- | ------------------ |
| Windows | 原生构建                  | NSIS / MSIX 安装包 |
| macOS   | 原生构建                  | .app 包            |
| Linux   | 原生构建                  | AppImage           |
| Docker  | 跨平台编译 / 服务模式部署 | Docker 镜像        |

## 项目结构

```
skdm/
├── main.go                          # Go 应用入口，Wails 服务注册
├── Taskfile.yml                     # 顶层构建任务定义
│
├── backed/                          # Go 后端
│   ├── cmd/
│   │   └── apiserver/app.go         # Wails Service 启动入口
│   ├── api/apiserver/v1/            # API 类型定义（DownloadRecord, Settings, EventRecord 等）
│   ├── internal/
│   │   ├── apiserver/
│   │   │   ├── app.go               # 应用初始化：SQLite + ARIA2 Server + PrepareRun
│   │   │   ├── server.go            # 服务器编排（7 步启动流程）
│   │   │   ├── router.go            # 依赖注入：创建 Controller 实例并注入 Store + RPC
│   │   │   ├── run.go               # Wails 启动钩子
│   │   │   ├── config/config.go     # 配置加载（viper + yaml）
│   │   │   ├── options/options.go   # 命令行选项
│   │   │   ├── controller/v1/       # Controller 层 — 业务逻辑编排
│   │   │   │   ├── aria2_client.go  # Aria2ClientProvider 接口定义
│   │   │   │   ├── download/        # 下载方法（AddURI, Pause, TellStatus 等 30+ 个）
│   │   │   │   ├── event/           # 事件查询
│   │   │   │   ├── settings/        # 设置读写 + aria2 全局选项同步
│   │   │   │   └── sys/             # 系统方法（版本、更新检查、全局统计）
│   │   │   ├── service/v1/          # Service 层 — Store 接口透传
│   │   │   └── store/               # Store 接口 + SQLite 实现
│   │   │       ├── store.go         # Factory 接口 + 全局单例
│   │   │       └── sqlite/          # SQLite 实现（download/event/settings CRUD）
│   │   └── pkg/
│   │       ├── server/
│   │       │   ├── GenericARIA2Server.go  # aria2c 进程管理 + RPC 重连 + 事件轮询
│   │       │   └── config.go              # Wails Service 桥接（Config 委托 + nil-safety）
│   │       └── options/
│   │           └── server_run_options.go  # 服务器运行选项
│   ├── pkg/
│   │   ├── db/
│   │   │   └── sqlite.go           # SQLite 数据库初始化（modernc.org/sqlite）+ 自动建表
│   │   └── version/
│   │       └── version.go           # 版本号常量
│   ├── configs/apiserver.yaml       # 默认配置文件
│   └── third_party/                 # aria2c 可执行文件 + 配置
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
│   │   ├── stores/
│   │   │   ├── download.ts          # Pinia Store（事件驱动的下载状态管理）
│   │   │   └── update.ts            # Pinia Store（自动更新检查结果缓存）
│   │   ├── views/
│   │   │   ├── runtask/             # 下载中（active/waiting/paused）
│   │   │   ├── alltask/             # 未完成（error/paused，可恢复/重试/删除）
│   │   │   ├── completetask/        # 已完成
│   │   │   ├── trashtask/           # 回收站（可清空/恢复/永久删除）
│   │   │   ├── tray-task/           # 托盘新建任务独立弹窗页面
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

| 表                | 用途                                                            |
| ----------------- | --------------------------------------------------------------- |
| `downloads`       | 下载任务记录（gid, url, dir, filename, 进度, 状态, 错误信息等） |
| `download_events` | 下载事件日志（事件类型, 事件数据 JSON, 时间）                   |
| `settings`        | 用户设置（key-value 存储）                                      |

### 状态同步机制

采用 **事件驱动 + 分级持久化** 架构，避免不必要的数据库写入：

- **aria2 事件订阅**：监听 Start/Pause/Complete/Error/Stop 事件，状态变更时写入 SQLite 并通过 Wails Events 推送到前端
- **进度轮询**：每 3 秒从 aria2 拉取活跃任务进度和速度，直接通过 Wails Events 推送到前端 Pinia Store（不写 SQLite，减少 ~90% 写入）
- **全量兜底同步**：每 300 秒从 aria2 全量拉取，同步到 SQLite 并推送到前端
- **启动同步**：应用启动时从 aria2 拉取所有任务状态，合并到 SQLite，初始化前端 Store

## 开发说明

### RPC 通信

- aria2c 监听 `localhost:6800`，WebSocket 端点 `/jsonrpc`
- RPC Secret 配置为 `my-strong-secret-token-2026`（后续版本将支持自定义）
- Go 端使用 `github.com/siku2/arigo` 库封装 RPC 调用
- `GenericARIA2Server` 持有 `arigo.Client` 并暴露 `Client()` 方法供 Controller 调用
- Controller 通过 `Aria2ClientProvider` 接口获取 RPC 客户端，实现松耦合
- 前端通过 Wails 自动生成的 binding → Config 桥接 → GenericARIA2Server → Controller → arigo RPC，不直接连接 aria2

### 添加新的下载操作方法

1. 在 `backed/internal/apiserver/controller/v1/download/download.go` 的 `DownloadController` 中添加业务方法
2. 业务方法通过 `d.rpc.Client()` 调用 aria2 RPC，通过 `d.srv.xxx()` 操作数据库
3. 在 `backed/internal/pkg/server/GenericARIA2Server.go` 中添加对外暴露的包装方法
4. 在 `backed/internal/pkg/server/config.go` 中添加 Config 桥接方法（含 nil-safety 守卫）
5. 重新运行 `wails3 dev` 或 `wails3 generate bindings` 更新前端 binding
6. 在前端组件中导入生成的 binding 函数调用

### 无边框窗口与系统托盘

- 窗口控制通过 `@wailsio/runtime` 的 `Window` API
- 标题栏拖拽：CSS `-webkit-app-region: drag`，按钮区域 `no-drag`
- 系统托盘：`application.SystemTray` + `build/windows/icon.ico`
- 窗口关闭拦截：`RegisterHook(events.Common.WindowClosing)` 隐藏窗口而非销毁
- 托盘 "新建任务" 弹窗：独立 WebviewWindow 加载 `/tray-task` 路由
