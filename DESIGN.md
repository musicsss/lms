# LMS — Lightweight Media System 软件设计文档

> 版本：v1.0 | 日期：2026-06-05

## 1. 概要

LMS 是一个面向个人/小团队的私有云平台，以**网盘**为核心，集成**在线视频播放**和**轻量论坛**，支持用户注册登录与权限管理。后端采用 Go 单体架构（Gin + PostgreSQL），前端为 React SPA（Vite + React Router），通过 Docker Compose 一键部署。

**核心设计原则**：单体优先、接口抽象、存储可替换、简洁务实。

## 2. 系统架构

```
┌─────────────────────────────────────────────────┐
│                   Nginx (反向代理)                 │
├──────────────────────┬──────────────────────────┤
│   React SPA (Vite)   │   Go API Server (Gin)     │
│   localhost:5173      │   localhost:8080          │
│   (dev) / static/     │                           │
│                       │  ┌─────────────────────┐ │
│                       │  │  Auth Module         │ │
│                       │  │  File Module         │ │
│                       │  │  Video Module        │ │
│                       │  │  Forum Module        │ │
│                       │  └─────────────────────┘ │
│                       │           │               │
│                       │     ┌─────▼─────┐         │
│                       │     │ PostgreSQL │         │
│                       │     └───────────┘         │
│                       │           │               │
│                       │  ┌────────▼────────┐      │
│                       │  │ 本地文件系统      │      │
│                       │  │ (S3接口抽象层)    │      │
│                       │  └─────────────────┘      │
└──────────────────────┴──────────────────────────┘
```

## 3. 后端设计

### 3.1 目录结构（参考 Gitea/Cloudreve 的 Go 项目布局）

```
server/
├── cmd/
│   └── lms/              # 主入口
│       └── main.go
├── internal/
│   ├── config/           # 配置加载（Viper）
│   ├── middleware/       # Gin 中间件（JWT、CORS、日志、限流）
│   ├── model/            # GORM 数据模型
│   ├── repository/       # 数据访问层
│   ├── service/          # 业务逻辑层
│   │   ├── auth/
│   │   ├── file/
│   │   ├── video/
│   │   └── forum/
│   ├── handler/          # HTTP 处理器（薄层，调用 service）
│   │   ├── auth.go
│   │   ├── file.go
│   │   ├── video.go
│   │   └── forum.go
│   ├── router/           # 路由注册
│   └── storage/          # 存储抽象层（本地/S3）
│       ├── driver.go     # StorageDriver 接口
│       ├── local.go
│       └── s3.go
├── migrations/           # SQL 迁移脚本
├── go.mod
└── go.sum
```

### 3.2 存储抽象层（参考 MinIO 的接口设计）

```go
type StorageDriver interface {
    Put(ctx context.Context, key string, reader io.Reader, size int64) error
    Get(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
    Range(ctx context.Context, key string, offset, length int64) (io.ReadCloser, error)
    List(ctx context.Context, prefix string) ([]FileInfo, error)
}
```

开发期默认使用 `LocalDriver`（本地文件系统），生产可切换到 `S3Driver`（MinIO/AWS S3/阿里云OSS）。

### 3.3 数据库设计（核心表）

```
┌──────────────┐     ┌─────────────────┐     ┌───────────────────┐
│    users     │     │     files       │     │   file_shares     │
├──────────────┤     ├─────────────────┤     ├───────────────────┤
│ id (PK)      │◄──┐ │ id (PK)         │     │ id (PK)           │
│ username     │   │ │ user_id (FK)   ─┼────►│ file_id (FK)      │
│ password_hash│   │ │ parent_id       │     │ token             │
│ email        │   │ │ name            │     │ password (可选)    │
│ role         │   │ │ is_dir          │     │ expire_at         │
│ avatar_url   │   │ │ size            │     │ created_at        │
│ created_at   │   │ │ mime_type       │     └───────────────────┘
│ updated_at   │   │ │ storage_key     │
└──────────────┘   │ │ md5             │
                    │ │ is_video        │     ┌───────────────────┐
                    │ │ video_status   ─┼────►│  video_transcodes │
                    │ │ created_at      │     ├───────────────────┤
                    │ │ updated_at      │     │ id (PK)           │
                    │ └─────────────────┘     │ file_id (FK)      │
                    │                         │ resolution        │
                    │                         │ hls_path          │
┌──────────────┐    │                         │ status            │
│   boards     │    │                         │ created_at        │
├──────────────┤    │                         └───────────────────┘
│ id (PK)      │    │  ┌─────────────────┐
│ name         │    │  │     posts       │     ┌──────────────┐
│ slug         │    │  ├─────────────────┤     │  post_likes  │
│ description  │    │  │ id (PK)         │     ├──────────────┤
│ sort_order   │    │  │ board_id (FK)   │     │ post_id (FK) │
│ created_at   │    └──│ user_id (FK)    │     │ user_id (FK) │
└──────────────┘       │ title           │     │ created_at   │
        │               │ content         │     └──────────────┘
        │               │ parent_id(回复) │
        ▼               │ view_count      │
   ┌───────────┐        │ created_at      │
   │   posts   │        │ updated_at      │
   └───────────┘        └─────────────────┘
```

**设计要点**：

- `files` 表用 `parent_id` 实现树形目录结构，`is_dir` 区分文件和文件夹
- `files.is_video` + `files.video_status` 标记视频文件及其转码状态（pending/processing/done/failed）
- `posts.parent_id` 为 NULL 表示主题帖，非 NULL 表示回复，支持单层嵌套回复
- 文件分享通过 `file_shares` 实现，生成唯一 token，支持可选密码和过期时间

### 3.4 API 设计（RESTful）

| 模块 | 方法 | 路径 | 说明 |
|------|------|------|------|
| Auth | POST | `/api/v1/auth/register` | 用户注册 |
| Auth | POST | `/api/v1/auth/login` | 登录，返回 JWT |
| Auth | GET | `/api/v1/auth/me` | 获取当前用户信息 |
| File | GET | `/api/v1/files?parent_id=` | 列出目录内容 |
| File | POST | `/api/v1/files/upload` | 上传文件（multipart） |
| File | GET | `/api/v1/files/:id/download` | 下载/流式传输 |
| File | DELETE | `/api/v1/files/:id` | 删除文件/文件夹 |
| File | POST | `/api/v1/files/mkdir` | 创建文件夹 |
| File | POST | `/api/v1/files/:id/share` | 生成分享链接 |
| File | GET | `/api/v1/share/:token` | 访问分享（公开接口） |
| Video | GET | `/api/v1/videos/:id/playlist.m3u8` | HLS 播放列表 |
| Video | GET | `/api/v1/videos/:id/segment/:seg` | HLS 分片 |
| Forum | GET | `/api/v1/boards` | 获取板块列表 |
| Forum | GET | `/api/v1/boards/:id/posts?page=` | 板块帖子列表（分页） |
| Forum | POST | `/api/v1/boards/:id/posts` | 发帖 |
| Forum | GET | `/api/v1/posts/:id` | 帖子详情+回复列表 |
| Forum | POST | `/api/v1/posts/:id/reply` | 回复帖子 |
| Forum | POST | `/api/v1/posts/:id/like` | 点赞/取消点赞 |

### 3.5 视频处理流水线（参考 PeerTube）

```
上传视频 → 保存原始文件 → 标记 video_status=pending
    → 异步FFmpeg转码(goroutine池)
        → 转HLS (1080p/720p/480p三档)
        → 生成 .m3u8 + .ts 分片
    → 标记 video_status=done
    → 前端 video.js / hls.js 播放
```

- 使用 `os/exec` 调用 FFmpeg，非阻塞转码
- 转码任务通过 channel + worker pool 控制并发（默认并发数=2）
- 转码进度写入 `video_transcodes` 表，前端轮询获取进度

## 4. 前端设计

### 4.1 路由结构

```
/                         → 首页/网盘（需登录）
/login                    → 登录页
/register                 → 注册页
/files/:path*             → 网盘文件浏览/管理
/files/share/:token       → 分享页（公开访问）
/video/:id                → 视频播放页
/forum                    → 论坛首页（板块列表）
/forum/:boardSlug         → 板块帖子列表
/forum/:boardSlug/:postId → 帖子详情
```

### 4.2 组件树（关键组件）

```
App
├── AuthLayout（登录/注册页）
│   ├── LoginForm
│   └── RegisterForm
├── MainLayout（需登录的页面，含顶栏+侧边栏）
│   ├── Navbar（用户信息、退出）
│   ├── Sidebar（功能导航：网盘/视频/论坛）
│   └── Content
│       ├── FileExplorer（网盘页面）
│       │   ├── Breadcrumb（目录面包屑）
│       │   ├── FileGrid / FileList（文件视图切换）
│       │   ├── UploadDialog（拖拽/点击上传）
│       │   └── ShareDialog（生成分享链接）
│       ├── VideoPlayer（视频播放页）
│       │   └── video.js / hls.js 播放器
│       └── Forum（论坛页面）
│           ├── BoardList
│           ├── PostList（分页列表）
│           └── PostDetail
│               ├── ReplyList
│               ├── ReplyEditor
│               └── LikeButton
└── ShareLayout（分享页，无登录框）
    └── SharedFileView（预览/下载）
```

### 4.3 状态管理

不引入重量级状态库，使用 React Context + `useReducer` 管理全局状态：

- `AuthContext`：JWT token、当前用户信息、登录/退出方法
- `FileContext`（网盘页面内）：当前目录路径、文件列表、选中项
- 组件内局部状态用 `useState` 处理

API 请求封装在 `src/api/` 目录，使用 `fetch` + 拦截器自动附加 JWT header 和处理 401 刷新。

## 5. 部署架构

Docker Compose 编排文件结构：

```yaml
services:
  nginx:
    image: nginx:alpine
    ports: ["80:80", "443:443"]
    volumes: [./nginx.conf:/etc/nginx/nginx.conf]

  server:
    build: ./server
    environment: [DB_DSN, JWT_SECRET, STORAGE_ROOT, ...]
    volumes: [./data:/data]

  db:
    image: postgres:16-alpine
    volumes: [./pgdata:/var/lib/postgresql/data]
    environment: [POSTGRES_DB, POSTGRES_USER, POSTGRES_PASSWORD]

  redis:  # 可选：用于转码队列、限流
    image: redis:7-alpine
```

## 6. 测试计划

| 层级 | 范围 | 工具 |
|------|------|------|
| 单元测试 | 各 service 包核心逻辑 | Go `testing` + `testify` |
| 集成测试 | Repository 层（真实测试DB） | `testcontainers-go` 启动 PostgreSQL |
| API 测试 | Handler 层端到端 | `httptest` + 表驱动测试 |
| 前端测试 | 关键交互组件 | Vitest + React Testing Library |
| E2E | 核心用户流程 | Playwright（登录→上传→播放→论坛发帖） |

## 7. 关键假设

- 单用户/小团队场景，不做多租户隔离
- 文件存储量级在 TB 以下，不需要分片集群
- FFmpeg 作为系统依赖预装在 Docker 镜像中
- 视频转码仅转 HLS 格式，不做多格式兼容（MP4/DASH/WebM）
- 论坛不包含 @提醒、私信、管理员面板等进阶功能
- 前端仅支持现代浏览器（Chrome/Firefox/Edge 最新两个大版本）
- 生产部署前需替换默认 JWT secret、数据库密码等敏感配置

## 8. 项目分期建议

**一期（MVP）**：用户注册登录 + 网盘（上传/下载/目录管理/分享） + 基础视频转码播放

**二期**：论坛（板块+发帖+回复+点赞） + 视频转码进度展示 + 文件预览（图片/文档缩略图）

**三期**：S3 存储后端切换 + 视频多清晰度 + 分享密码保护 + 管理后台

## 9. 技术选型汇总

| 维度 | 选择 |
|------|------|
| 定位 | 个人/小团队自用 |
| 核心 | 网盘优先，视频+论坛配套 |
| 后端语言 | Go 1.23+ |
| Web 框架 | Gin |
| 数据库 | PostgreSQL 16 |
| ORM | GORM |
| 配置管理 | Viper |
| 前端框架 | React 18+ |
| 构建工具 | Vite |
| 路由 | React Router v6+ |
| UI 组件库 | 待定（推荐 shadcn/ui 或 Ant Design） |
| 视频播放器 | video.js + hls.js |
| 存储 | 本地文件系统 + S3 接口抽象 |
| 视频转码 | FFmpeg (HLS) |
| 认证 | JWT (golang-jwt) |
| 部署 | Docker Compose |
| 反向代理 | Nginx |
