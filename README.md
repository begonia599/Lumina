# Lumina · 沉浸阅读

> 一间属于你的书房 —— 自部署的深色毛玻璃风格 TXT 阅读器，支持多设备同步、全书搜索、自动滑动、书签备注。Go + React 单仓。

![stack](https://img.shields.io/badge/stack-Go%201.25%20%7C%20React%2019%20%7C%20Postgres%2015-gold?style=flat-square)

## 特性

- 🌙 **三主题**：深渊 / 晨曦 / 羊皮纸，每主题独立衬线阅读色板
- 🔐 **多用户 · Cookie Session**：HTTP-only、SameSite=Lax、bcrypt(12)、IP 级注册/登录限流
- 📖 **章节解析**：自动识别 `第N章` / `序章` / `Chapter N`，自动探测 GBK/UTF-8 编码，解码 HTML 实体
- 🔖 **书签 / 进度跨端同步**：进度按 `{chapterIdx, charOffset}` 落库，字号 / 设备 / 窗口尺寸变化都能准确恢复
- 🔍 **全书搜索**：`/` 键唤起 overlay，结构化 DTO，rune 级预览高亮
- ⏵ **自动滑动**：10–240 px/s 连续滚动，章末可自动翻页，手动操作即暂停
- 🖼️ **自定义封面**：JPEG/PNG/WebP，MIME 字节级校验，无封面自动按书名 Hash 生成渐变
- 🏷️ **书籍元数据 + 标签**：书名 / 作者 / 简介 / Unicode 标签 GIN 索引
- ⌨️ **全键盘操作**：`← → T L B / P [ ] , ?` 全覆盖，按 `?` 查完整表
- 📱 **移动端**：抽屉支持 swipe-to-close，触控目标≥ 40px，底部栏常驻
- ✨ **View Transitions**：书架卡片 → 阅读器，流畅 morph
- 🛡️ **SQLi 防护**：0 处字符串拼接进 SQL，全部 `$N` 占位符

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.25 · Gin · pgx/v5 · bcrypt · gabriel-vasile/mimetype |
| 前端 | React 19 · Vite 8 · React Router 7 · Zustand 5 · Motion · Lucide React |
| 字体 | Fraunces Variable · Geist Variable · Noto Serif SC（fontsource 自托管） |
| 数据 | PostgreSQL 15 |
| 部署 | Docker Compose |

## 快速开始

### 依赖
- Docker + Docker Compose（或自备 PostgreSQL 15）
- Go 1.25+
- Node.js 20+

### 本地开发

```bash
# 1. 数据库
docker compose up -d postgres

# 2. 环境变量
cp .env.example .env   # 按需改 DATABASE_URL / PORT

# 3. 后端
go run .

# 4. 前端（另开终端）
cd web
npm install
npm run dev
```

浏览器打开 http://localhost:5173，注册账号即可使用。

### 局域网访问（手机 / 平板）

Vite 已配置 `host: true`，启动时会打印 LAN 地址：

```
➜  Network: http://192.168.x.x:5173/
```

手机连同一 Wi-Fi，浏览器直接打开该地址。首次 Windows 会弹防火墙提示，允许"专用网络"。

### 生产部署（Docker）

```bash
docker compose up -d --build
```

然后反向代理（Caddy / Nginx）把 80/443 转发到 8081，并在 `.env` 里设 `SESSION_COOKIE_SECURE=true`。

> ⚠️ 生产环境**必须**修改 `docker-compose.yml` 中的 postgres 密码和 `DATABASE_URL`。默认的 `lumina:lumina` 是本地开发用。

## 配置项

| 环境变量 | 默认 | 说明 |
|---|---|---|
| `DATABASE_URL` | — | 必填，pgx DSN |
| `PORT` | `8080` | Go 服务端口 |
| `UPLOAD_DIR` | `./uploads` | 上传根目录，下分 `{userID}/` |
| `REGISTRATION_ENABLED` | `true` | 是否开放注册 |
| `SESSION_COOKIE_SECURE` | `false` | HTTPS 下设 true |
| `CORS_ORIGINS` | `localhost:5173,localhost:3000` | 逗号分隔 |

## 架构要点

- **鉴权可插拔** — `internal/auth.Provider` 接口，当前实现 `LocalAuthProvider`；未来切自研 SDK 只需换一个 provider 注入（见 ADR-13）
- **资源按 userID 隔离** — 所有 service 方法签名含 `userID`，SQL 一律 `WHERE user_id = $N`
- **进度模型** — `{chapter_idx, char_offset}`，percentage 只是派生 UI 值
- **章节正文** — 后端返回 `paragraphs: string[]`，避免前后端双实现段落切分
- **统一错误** — `{error: {code, message}}`，前端按 code 分流

更多设计决策见 [implementation_plan.md](implementation_plan.md) 里的 ADR 章节。

## 目录

```
.
├── main.go                      # Gin 路由 + 中间件装配
├── internal/
│   ├── auth/                    # Provider 接口 + LocalAuthProvider + 用户名校验
│   ├── database/                # pgx 连接 + 幂等 migration
│   ├── handler/                 # HTTP 处理器
│   ├── httpx/                   # 统一错误信封
│   ├── middleware/              # RequireAuth + IP 限流
│   ├── model/                   # 领域对象
│   └── service/                 # 业务逻辑
└── web/
    └── src/
        ├── api/                 # fetch 封装 + 按资源分片
        ├── components/          # UI 原子 + 组合
        ├── hooks/               # useAutoScroll 等
        ├── pages/               # Auth / Bookshelf / Reader
        ├── stores/              # Zustand
        ├── styles/              # tokens / themes / reset / glass
        └── utils/               # 封面 hash / 节流 / view-transition 等
```

## 快捷键

| 键 | 作用 |
|---|---|
| `← / →` | 上 / 下一章 |
| `T` | 目录 |
| `L` | 书签列表 |
| `B` | 当前位置加书签 |
| `/` | 全书搜索 |
| `P` | 自动滑动 ⏵⏸ |
| `[` / `]` | 自动滑动减速 / 加速 |
| `,` | 设置 |
| `?` | 快捷键帮助 |
| `Esc` | 关闭任意浮层 |

## 许可

MIT
