# LMS Developer Handbook

> Version 2.0 | Last updated: 2026-06-05

本文档按当前仓库代码编写，优先反映实际目录、依赖、运行方式和架构边界。

## 1. 项目概览

LMS（Lightweight Management System）是一个文件管理与社区论坛一体化平台，采用 Go + React 前后端分离架构，并通过 Docker Compose 编排用户端、管理端、后端 API 和 PostgreSQL。

核心能力：

- 用户注册、登录、JWT 鉴权、管理员角色校验
- 文件上传、目录浏览、下载、删除、分享链接、视频文件状态标记
- 论坛板块、帖子、嵌套回复、点赞/取消点赞
- 管理后台 Dashboard、用户/文件/论坛管理、数据库浏览器
- 命令式运行时配置：日志等级、JWT 过期时间、上传限制、登录封禁策略、CORS 白名单
- 登录失败保护：验证码、限流、按账号或 IP 封禁
- 结构化日志：stdout + 文件输出，支持 lumberjack 轮转

## 2. 技术栈

| 层 | 技术 |
| --- | --- |
| 后端 | Go 1.25, Gin 1.12, GORM 1.31, PostgreSQL driver, Viper, slog |
| 认证与安全 | golang-jwt/jwt v5, bcrypt, gin-contrib/cors |
| 存储 | 本地文件系统 `storage.LocalDriver` |
| 日志 | `log/slog` + `gopkg.in/natefinch/lumberjack.v2` |
| 用户端 | React 19, Vite 8, React Router 7, Lucide React |
| 管理端 | React 19, Vite 7, React Router 7, Lucide React |
| 部署 | Docker Compose, Nginx, PostgreSQL 16 |

## 3. 目录结构

```text
.
|-- server/                         Go 后端
|   |-- cmd/lms/main.go             启动入口、AutoMigrate、管理员初始化
|   |-- config.yaml                 默认静态配置
|   |-- Dockerfile                  运行镜像，依赖预构建 bin/lms
|   |-- internal/
|   |   |-- config/                 Viper 配置加载
|   |   |-- dci/
|   |   |   |-- context/             业务交互上下文：auth/file/forum/admin
|   |   |   |-- data/                仓储接口与 GORM 实现
|   |   |   `-- tx/                  Unit of Work + Saga 补偿栈
|   |   |-- handler/                Gin HTTP Handler
|   |   |-- log/                    slog 封装与请求日志中间件
|   |   |-- loginprotect/           登录失败保护、验证码、封禁策略
|   |   |-- middleware/             JWT、Admin、CORS 中间件
|   |   |-- model/                  GORM 数据模型
|   |   |-- router/                 依赖装配与路由注册
|   |   |-- runtimecfg/             运行时配置引擎
|   |   `-- storage/                文件存储 Driver 接口与本地实现
|-- web/                            React 用户端
|   |-- src/api/client.js           fetch 封装、用户 token 注入
|   |-- src/contexts/AuthContext.jsx
|   |-- src/components/AppLayout.jsx
|   `-- src/pages/                  登录、注册、文件、分享、论坛、视频播放
|-- admin/                          React 管理端
|   |-- src/api/client.js           fetch 封装、admin token 注入
|   |-- src/components/AdminLayout.jsx
|   `-- src/pages/                  Dashboard、用户、文件、论坛、配置、DB 管理
|-- docker-compose.yml              web/admin/server/db 编排
|-- README.md
|-- DESIGN.md
`-- DEVELOPER.md
```

注意：当前代码已经从旧的 `internal/service`、`internal/repository` 目录迁移到 `internal/dci`，不要再按旧目录新增业务代码。

## 4. 后端启动流程

入口为 `server/cmd/lms/main.go`：

1. `config.Load()` 读取 `config.yaml`、默认值和 `LMS_*` 环境变量。
2. 初始化日志：debug 模式使用 text handler，release 模式使用 JSON handler。
3. 连接 PostgreSQL。
4. 执行 `AutoMigrate`：`User`、`File`、`FileShare`、`Board`、`Post`、`PostLike`、`VideoTranscode`、`RuntimeConfig`。
5. 如果存在 `LMS_SETUP_ADMIN=username:password`，确保管理员用户存在或提升为 admin。
6. 初始化本地存储驱动。
7. `router.Setup()` 装配运行时配置、登录保护、仓储、Handler 和路由。
8. 监听 `server.port`，默认 `:8080`。

还支持交互式创建管理员：

```bash
cd server
go run ./cmd/lms setup-admin
```

## 5. 后端架构边界

当前后端采用 DCI 风格组织业务：

- `handler/`：只处理 HTTP 细节，例如参数绑定、路径参数、状态码和 JSON 响应。
- `dci/context/<domain>/`：每个业务交互一个 Context，例如注册、登录、上传、删除文件、发帖、点赞、管理用户。
- `dci/data/`：定义仓储接口并提供 GORM 实现，Handler 不直接写复杂查询。
- `dci/tx/Unit`：包装 GORM 事务，并提供补偿动作栈。适用于“先写外部存储，再写数据库”这类需要失败回滚清理的流程。
- `model/`：GORM 表模型，不承载 HTTP 逻辑。
- `runtimecfg/`：命令解析、持久化、缓存和变更通知。

典型调用链：

```text
HTTP request
  -> handler.<Domain>Handler
  -> dci/context/<domain>.<Interaction>Context.Execute()
  -> dci/data.<Repo interface>
  -> GORM / storage.Driver / runtimecfg.Engine
  -> JSON response
```

带补偿的文件上传流程：

```text
UploadContext.Execute()
  -> 校验运行时上传大小限制
  -> storage.Driver.Put()
  -> tx.Unit.Defer(delete uploaded storage)
  -> tx.Begin()
  -> fileRepo.Create()
  -> tx.Commit()

任一步失败后调用 Rollback 时，会按逆序执行补偿动作。
```

## 6. 数据模型与表

所有表由 GORM AutoMigrate 创建。

| 表 | 模型 | 说明 |
| --- | --- | --- |
| `users` | `model.User` | 用户名、密码哈希、邮箱、角色、头像 |
| `files` | `model.File` | 用户文件和目录，树形 `parent_id`，存储 key，视频状态 |
| `file_shares` | `model.FileShare` | 文件分享 token、密码、过期时间 |
| `boards` | `model.Board` | 论坛板块、slug、描述、排序 |
| `posts` | `model.Post` | 帖子与回复，`parent_id` 表示嵌套回复 |
| `post_likes` | `model.PostLike` | 帖子点赞记录 |
| `video_transcodes` | `model.VideoTranscode` | 预留的视频转码记录 |
| `runtime_configs` | `runtimecfg.RuntimeConfig` | 运行时配置的 SET/ADD 数据 |

管理后台的 Database 页面只允许访问白名单表：`users`、`files`、`file_shares`、`boards`、`posts`、`post_likes`、`video_transcodes`、`runtime_configs`。

## 7. API 路由

所有 API 前缀为 `/api/v1`。

### 7.1 公开路由

| 方法 | 路径 | Handler | 说明 |
| --- | --- | --- | --- |
| POST | `/auth/register` | `authH.Register` | 注册用户 |
| POST | `/auth/login` | `authH.Login` | 登录，可能要求验证码或触发封禁 |
| GET | `/auth/captcha` | `authH.Captcha` | 获取算术验证码 |
| GET | `/share/:token` | `fileH.GetShare` | 访问分享链接 |

### 7.2 登录用户路由

这些路由需要 `Authorization: Bearer <token>`。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/auth/me` | 当前用户信息 |
| GET | `/files?parent_id=` | 当前目录文件列表 |
| POST | `/files/upload` | multipart 上传，字段为 `file`、可选 `parent_id` |
| POST | `/files/mkdir` | 创建目录，JSON: `name`, `parent_id` |
| GET | `/files/:id/download` | 下载文件 |
| DELETE | `/files/:id` | 删除文件或目录 |
| POST | `/files/:id/share` | 生成分享链接，JSON: `password`, `expire_hours` |
| GET | `/boards` | 板块列表 |
| GET | `/boards/:id/posts?page=` | 帖子列表 |
| POST | `/boards/:id/posts` | 发帖，JSON: `title`, `content` |
| GET | `/posts/:id` | 帖子详情并增加浏览量 |
| POST | `/posts/:id/reply` | 回复，JSON: `content` |
| POST | `/posts/:id/like` | 点赞/取消点赞 |

### 7.3 管理员路由

这些路由需要 JWT，且 `role=admin`。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/admin/stats` | Dashboard 统计 |
| GET | `/admin/users?page=&page_size=&search=` | 用户列表 |
| PUT | `/admin/users/:id` | 更新用户角色，JSON: `role` |
| DELETE | `/admin/users/:id` | 删除用户，禁止删除最后一个管理员 |
| GET | `/admin/files?page=&page_size=` | 文件列表 |
| DELETE | `/admin/files/:id` | 删除任意文件或目录 |
| GET | `/admin/boards` | 板块列表 |
| POST | `/admin/boards` | 新建板块 |
| PUT | `/admin/boards/:id` | 更新板块 |
| DELETE | `/admin/boards/:id` | 删除板块 |
| GET | `/admin/boards/:id/posts?page=&page_size=` | 板块帖子列表 |
| DELETE | `/admin/posts/:id` | 删除帖子及直接回复 |
| POST | `/admin/config/exec` | 执行运行时配置命令 |
| GET | `/admin/config/targets` | 配置页面元数据 |
| GET | `/admin/db/tables` | 可浏览表列表 |
| GET | `/admin/db/tables/:name` | 表结构 |
| GET | `/admin/db/tables/:name/rows` | 表数据分页 |
| POST | `/admin/db/tables/:name` | 插入行 |
| PUT | `/admin/db/tables/:name/:id` | 更新行 |
| DELETE | `/admin/db/tables/:name/:id` | 删除行 |

## 8. 运行时配置

运行时配置由 `runtimecfg.Engine` 负责：

- `Store` 将配置持久化到 `runtime_configs`。
- `Cache` 使用内存缓存和锁保护读写。
- `Engine.Exec()` 解析并执行命令。
- `OnChange` 通知日志、登录保护、CORS 等模块热更新。

### 8.1 命令语法

```text
SET <TARGET>: KEY=VALUE [, KEY=VALUE ...]
ADD <TARGET>: KEY=VALUE [, KEY=VALUE ...]
LST <TARGET> [ID=n]
MOD <TARGET>: ID=n, KEY=VALUE [, KEY=VALUE ...]
RMV <TARGET>: ID=n
ACT <ACTION>
```

### 8.2 已注册 Target

| Target | Kind | 字段 | 说明 |
| --- | --- | --- | --- |
| `SYSLOG` | SET | `LEVEL=DEBUG|INFO|WARN|ERROR` | 热切换日志级别 |
| `JWT` | SET | `EXPIRETIME=1..720` | JWT 过期小时数 |
| `FILEUPLD` | SET | `MAXSIZE=1..10240` | 文件上传大小上限，单位 MB |
| `LGFAILFIBPLCY` | ADD | `RANGE`, `BLOCKPLCY` | 登录失败封禁策略 |
| `CORS` | ADD | `ORIGIN` | 动态 CORS 白名单 |

默认 SET 配置在引擎启动时自动补齐：

```text
SYSLOG: LEVEL=INFO
JWT: EXPIRETIME=72
FILEUPLD: MAXSIZE=2048
```

### 8.3 Action

| Action | 说明 |
| --- | --- |
| `SYSTEMRST` | 返回响应后调用 `os.Exit(0)`，Docker 会按 restart policy 拉起 |
| `CLRLIMIT` | 清空登录失败计数和封禁状态 |
| `RELOAD` | 从数据库重载所有运行时配置，并通知所有 target |

示例：

```text
SET SYSLOG: LEVEL=DEBUG
SET JWT: EXPIRETIME=96
SET FILEUPLD: MAXSIZE=4096
ADD CORS: ORIGIN=https://example.com
ADD LGFAILFIBPLCY: RANGE=ALL_USER, BLOCKPLCY=ACCOUNT
ADD LGFAILFIBPLCY: RANGE=SINGLE_USER:alice, BLOCKPLCY=ACCOUNT
ADD LGFAILFIBPLCY: RANGE=IP:10.0.0.5, BLOCKPLCY=IP
LST LGFAILFIBPLCY
MOD LGFAILFIBPLCY: ID=1, BLOCKPLCY=IP
RMV LGFAILFIBPLCY: ID=1
ACT CLRLIMIT
ACT RELOAD
ACT SYSTEMRST
```

`runtimecfg.Store` 会对 ADD 类型做语义去重：

- `LGFAILFIBPLCY`：按 `RANGE` 去重。
- `CORS`：按 `ORIGIN` 去重。

## 9. 登录保护

`loginprotect.Guard` 在内存中维护失败窗口、验证码和封禁状态：

- 失败窗口：5 分钟。
- 3 次失败后要求验证码。
- 5 次失败后封禁 1 小时。
- 验证码为算术加法题，有效期 5 分钟。
- 登录成功后清除对应 IP 和账号的失败记录。

封禁策略来自运行时配置 `LGFAILFIBPLCY`：

- `RANGE=ALL_USER`：全局策略。
- `RANGE=SINGLE_USER:<username>`：指定账号策略。
- `RANGE=IP:<addr>`：指定 IP 策略。
- `BLOCKPLCY=ACCOUNT`：按账号封禁。
- `BLOCKPLCY=IP`：按 IP 封禁。

策略匹配优先级为 `SINGLE_USER > IP > ALL_USER`，默认策略是全局按 IP 封禁。

## 10. 前端应用

### 10.1 用户端 `web`

主要路由：

| 路径 | 页面 |
| --- | --- |
| `/login` | 登录 |
| `/register` | 注册 |
| `/share/:token` | 分享链接访问 |
| `/files` | 文件浏览 |
| `/files/:folderId` | 子目录文件浏览 |
| `/video/:id` | 视频播放页 |
| `/forum` | 论坛首页 |
| `/forum/:boardId` | 帖子列表 |
| `/forum/:boardId/:postId` | 帖子详情 |

`web/src/api/client.js` 支持 `VITE_API_BASE`，默认 `/api/v1`，并从 `localStorage.token` 注入用户 JWT。

### 10.2 管理端 `admin`

主要路由：

| 路径 | 页面 |
| --- | --- |
| `/login` | 管理员登录 |
| `/dashboard` | 统计 |
| `/users` | 用户管理 |
| `/files` | 文件管理 |
| `/forum` | 论坛管理 |
| `/config` | 运行时配置 |
| `/db` | 数据库管理 |

`admin/src/api/client.js` 固定使用 `/api/v1`，并从 `localStorage.admin_token` 注入管理员 JWT。

## 11. 本地开发

### 11.1 Docker 一键运行

```bash
docker compose up -d
```

默认端口：

| 服务 | 地址 |
| --- | --- |
| 用户端 | `http://localhost` |
| 管理端 | `http://localhost:8081` |
| 后端 API | 容器内 `server:8080`，经 Nginx `/api` 反代 |
| PostgreSQL | 容器内 `db:5432` |

`docker-compose.yml` 当前默认管理员：

```text
username: admin
password: admin123
```

### 11.2 分服务开发

后端：

```bash
cd server
go run ./cmd/lms
```

用户端：

```bash
cd web
npm install
npm run dev
```

管理端：

```bash
cd admin
npm install
npm run dev
```

两个 Vite dev server 都会把 `/api` 代理到 `http://localhost:8080`。管理端默认端口 `8081`，用户端默认端口通常为 `5173`。

### 11.3 构建 Docker 镜像

后端 Dockerfile 只复制 `server/bin/lms`，因此构建镜像前需要先编译二进制：

```powershell
cd server
$env:GOOS='linux'
$env:GOARCH='amd64'
$env:CGO_ENABLED='0'
go build -o bin/lms ./cmd/lms
cd ..
docker compose build
```

前端 Dockerfile 只复制 `dist`，因此如单独构建镜像，也需要先运行对应目录的 `npm run build`。

## 12. 配置项

`server/config.yaml` 默认值：

```yaml
server:
  port: 8080
  mode: debug

database:
  host: db
  port: 5432
  user: lms
  password: lms123
  dbname: lms
  sslmode: disable

jwt:
  secret: change-me-in-production
  expirehour: 72

storage:
  driver: local
  root: ./data

log:
  dir: ./logs
  maxsizemb: 20
  maxbackups: 20
```

环境变量格式为 `LMS_<SECTION>_<KEY>`，例如：

| 环境变量 | 说明 |
| --- | --- |
| `LMS_SERVER_MODE` | `debug` 或 `release` |
| `LMS_DATABASE_HOST` | 数据库主机 |
| `LMS_DATABASE_PORT` | 数据库端口 |
| `LMS_DATABASE_USER` | 数据库用户 |
| `LMS_DATABASE_PASSWORD` | 数据库密码 |
| `LMS_DATABASE_DBNAME` | 数据库名 |
| `LMS_DATABASE_SSLMODE` | SSL 模式 |
| `LMS_JWT_SECRET` | JWT 签名密钥 |
| `LMS_STORAGE_ROOT` | 文件存储根目录 |
| `LMS_SETUP_ADMIN` | 初始化管理员，格式 `username:password` |

当前 `config.Load()` 显式绑定了 database 相关环境变量；其他键依赖 Viper `AutomaticEnv` 与结构体反序列化行为，修改配置加载时需要注意覆盖是否实际生效。

## 13. 日志

- debug 模式：text 格式。
- release 模式：JSON 格式。
- 输出位置：stdout 和 `${log.dir}/server.log`。
- 轮转策略：单文件默认 20 MB，最多 20 个备份，旧文件压缩。
- 请求日志会记录 method、path、status、ip、query 和已鉴权的 user_id。
- 可通过运行时配置热切换等级：`SET SYSLOG: LEVEL=DEBUG`。

查看容器日志：

```bash
docker compose logs -f server
```

## 14. 编码与变更约定

后端：

- 新增业务流程优先在 `internal/dci/context/<domain>` 增加 Context，不要把业务逻辑堆进 Handler。
- 需要数据访问时，先在 `internal/dci/data` 的 Repo 接口补方法，再实现 GORM 查询。
- 需要事务时使用 `tx.NewUnit(db)`，外部副作用要注册补偿动作。
- Handler 负责错误映射和 HTTP 状态码，Context 返回明确 error。
- 新增表模型后，在 `cmd/lms/main.go` 的 `AutoMigrate` 中注册。
- 管理端 DB 浏览器若要开放新表，需要同步更新 `handler/db.go` 的 `safeTables()`。

前端：

- 页面放在 `src/pages/`，公共布局和组件放在 `src/components/`。
- API 请求统一走 `src/api/client.js`。
- 用户端使用 `localStorage.token` / `localStorage.user`，管理端使用 `admin_token` / `admin_user`。
- 路由变更需要同时检查 Nginx SPA 回退配置是否仍然适用。

运行时配置：

- 新增 target 时，至少需要更新 `runtimecfg/const.go`、`runtimecfg/engine.go` 默认值或校验、`handler/config.go` targets 元数据。
- 如果 target 需要热生效，在 `router.Setup()` 的 `rtEngine.OnChange` 回调中接入。
- ADD 类型如需去重，在 `runtimecfg/store.go` 中扩展去重键逻辑。

## 15. 测试

当前测试集中在 DCI 文件上下文和事务补偿：

```bash
cd server
go test ./...
```

已有测试：

- `internal/dci/context/file/file_test.go`：上传大小校验、补偿删除、列表、递归删除收集。
- `internal/dci/tx/unit_test.go`：补偿顺序、补偿错误、Commit 清理、Defer 顺序。

新增涉及数据库、登录保护、运行时配置或路由权限的代码时，建议补充对应单元测试或集成测试，尤其覆盖失败路径和权限边界。
