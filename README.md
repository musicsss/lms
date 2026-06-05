# LMS — Lightweight Management System

文件管理与社区论坛一体化平台，采用 Go + React 全栈架构，Docker 一键部署。

## 功能概览

| 模块 | 说明 |
|------|------|
| 用户系统 | 注册、登录、JWT 鉴权、角色管理（user/admin） |
| 文件管理 | 上传、下载、目录浏览、一键分享链接 |
| 社区论坛 | 板块、帖子、回复、点赞 |
| 管理后台 | Dashboard、用户管理、文件管理、论坛管理 |
| 运行时配置 | 命令驱动的配置系统，支持热更新（见下方） |
| 登录保护 | 5 分钟窗口内 3 次失败触发验证码，5 次触发封禁 |
| 结构化日志 | JSON 格式，stdout + 文件双输出，自动轮转与压缩 |

## 技术栈

| 层 | 技术 |
|----|------|
| 后端 | Go 1.25 · Gin · GORM · PostgreSQL · JWT · Viper |
| 前端（用户端） | React 19 · Vite · React Router · Lucide Icons |
| 前端（管理端） | React 19 · Vite · React Router · Lucide Icons |
| 基础设施 | Docker Compose · Nginx · PostgreSQL 16 |

## 项目结构

```
├── server/                 # Go 后端
│   ├── cmd/lms/main.go     # 入口
│   ├── internal/
│   │   ├── config/         # 静态配置加载（Viper）
│   │   ├── handler/        # HTTP 处理器（auth, file, forum, admin, config）
│   │   ├── log/            # 结构化日志 + Gin 中间件
│   │   ├── loginprotect/   # 登录限流与封禁
│   │   ├── middleware/      # JWT、CORS、Admin 中间件
│   │   ├── model/          # GORM 数据模型
│   │   ├── repository/     # 数据访问层
│   │   ├── router/         # 路由注册
│   │   ├── runtimecfg/     # 运行时配置引擎
│   │   ├── service/        # 业务逻辑层
│   │   └── storage/        # 文件存储驱动
│   ├── config.yaml         # 默认配置
│   ├── Dockerfile
│   └── go.mod
├── web/                    # React 用户端
│   ├── src/pages/          # AuthPage, FileExplorer, Forum, SharePage
│   ├── Dockerfile
│   └── vite.config.js
├── admin/                  # React 管理端
│   ├── src/pages/          # Dashboard, Users, Files, Forum, Config
│   ├── Dockerfile
│   └── vite.config.js
├── docker-compose.yml
└── README.md
```

## 快速开始

### 前置要求

- Docker & Docker Compose
- （开发模式额外需要）Go 1.25+ · Node.js 18+

### 1. 克隆并启动

```bash
git clone <repo-url> lms
cd lms
docker compose up -d
```

首次启动会自动执行数据库迁移并创建管理员账户。

### 2. 访问

| 服务 | 地址 |
|------|------|
| 用户端 | http://localhost |
| 管理后台 | http://localhost:8081 |
| API（内部） | :8080 |

### 3. 管理员登录

默认管理员凭据由环境变量 `LMS_SETUP_ADMIN` 控制（见 `docker-compose.yml`）：

- 用户名：`admin`
- 密码：`123456`

登录后可在管理后台 → Configuration 动态修改系统参数。

## 运行时配置

管理后台 Configuration 页面提供一套命令驱动的热配置系统，无需重启即可调整系统行为。

### 命令参考

```
SET SYSLOG: LEVEL=DEBUG           # 切换日志等级（DEBUG/INFO/WARN/ERROR）
SET JWT: EXPIRETIME=96            # JWT 过期时间（小时，1-720）
SET FILEUPLD: MAXSIZE=4096        # 文件上传大小上限（MB）

ADD LGFAILFIBPLCY: RANGE=ALL_USER, BLOCKPLCY=ACCOUNT   # 新增登录封禁策略
MOD LGFAILFIBPLCY: ID=1, BLOCKPLCY=IP                  # 修改策略
RMV LGFAILFIBPLCY: ID=1                                # 删除策略
LST LGFAILFIBPLCY                                      # 查看所有策略

ADD CORS: ORIGIN=https://example.com                   # 添加 CORS 白名单

ACT SYSTEMRST         # 重启系统
ACT CLRLIMIT          # 清除所有登录限流计数
ACT RELOAD            # 重载运行时配置
```

配置修改即时生效，无需重启。

## 配置项

### 静态配置（config.yaml）

```yaml
server:
  port: 8080
  mode: debug          # debug | release

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
  maxsizemb: 20        # 单个日志文件上限
  maxbackups: 20       # 最多保留文件数
```

### 环境变量

所有配置项均可通过环境变量覆盖，格式为 `LMS_<SECTION>_<KEY>`：

| 变量 | 说明 |
|------|------|
| `LMS_DATABASE_HOST` | 数据库主机 |
| `LMS_DATABASE_PASSWORD` | 数据库密码 |
| `LMS_JWT_SECRET` | JWT 签名密钥 |
| `LMS_SERVER_MODE` | 运行模式（debug/release） |
| `LMS_SETUP_ADMIN` | 初始管理员（`username:password`） |
| `LMS_STORAGE_ROOT` | 文件存储根目录 |

## API 概览

所有接口前缀 `/api/v1`。

### 公开

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/auth/register` | 注册 |
| POST | `/auth/login` | 登录 |
| GET | `/auth/captcha` | 获取验证码 |
| GET | `/share/:token` | 访问分享链接 |

### 需认证

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/auth/me` | 当前用户信息 |
| GET | `/files` | 文件列表 |
| POST | `/files/upload` | 上传文件 |
| POST | `/files/mkdir` | 创建目录 |
| GET | `/files/:id/download` | 下载文件 |
| DELETE | `/files/:id` | 删除文件 |
| POST | `/files/:id/share` | 生成分享链接 |
| GET | `/boards` | 板块列表 |
| GET | `/boards/:id/posts` | 帖子列表 |
| POST | `/boards/:id/posts` | 发帖 |
| GET | `/posts/:id` | 帖子详情 |
| POST | `/posts/:id/reply` | 回复 |
| POST | `/posts/:id/like` | 点赞 |

### 管理员

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/admin/stats` | Dashboard 统计 |
| GET/PUT/DELETE | `/admin/users` | 用户管理 |
| GET/DELETE | `/admin/files` | 文件管理 |
| GET/POST/PUT/DELETE | `/admin/boards` | 板块管理 |
| GET/DELETE | `/admin/boards/:id/posts` | 帖子管理 |
| POST | `/admin/config/exec` | 执行配置命令 |
| GET | `/admin/config/targets` | 获取可配置目标 |

## 开发

```bash
# 后端（需要本地 PostgreSQL）
cd server
go run ./cmd/lms

# 用户端
cd web
npm install && npm run dev

# 管理端
cd admin
npm install && npm run dev -- --port 8081
```

## 日志

日志以 JSON 格式输出，同时写入 stdout 和 `logs/server.log`。单个文件不超过 20MB，最多保留 20 个历史文件，旧文件自动 gzip 压缩。

```bash
# 查看实时日志
docker compose logs -f server

# 或查看日志文件
tail -f logs/server.log
```
