# Lumina 实施计划（前端 + 多用户扩展 + 契约）

> 从零搭建 React 18 + Vite 前端，并把后端从「单用户」升级为「多用户带登录」的自部署阅读服务。单 Go 二进制交付（`embed.FS`），SPA 同源，HTTP-only Cookie Session 认证。

---

## 0. 设计总纲（替代原 demo.html 的通用化）

原 `demo.html` 仅验证了阅读容器本身的排印。它作为"氛围锚点"保留，但下列项在此计划中**覆盖**它：

| 维度 | demo 原方案 | 新方案 | 原因 |
|:---|:---|:---|:---|
| UI 字体 | Inter | **Geist**（可变字重，CJK 回退到 Noto Sans SC） | Inter 是 AI 生成最常见默认字，和"氛围阅读器"调性冲突 |
| Display / Latin 正文 | 无 | **Fraunces**（variable serif，软 / soft 轴） | 提供一致的"书卷感"延伸到 UI 标题 |
| CJK 阅读字体 | Noto Serif SC | **Noto Serif SC**（保留） | — |
| Accent（深色） | `hsl(260,100%,75%)` 霓虹紫 | **`hsl(38, 72%, 64%)` 暖金** | 紫色 100% 饱和在纸质氛围里过赛博朋克 |
| Accent（晨曦） | 未定义 | **`hsl(218, 62%, 42%)` 墨蓝** | 白底克制 |
| Accent（羊皮纸） | 未定义 | **`hsl(12, 48%, 48%)` 陶土红** | 与纸色同色温、印章感 |
| 章节标题颜色 | accent 色 | `--text-main` 加粗 + 章节编号 eyebrow（小号字距大写字母）用 accent | 大块彩色破坏沉浸；accent 当点缀 |
| 阅读容器投影 | `box-shadow: 0 0 40px rgba(0,0,0,.2)` | 删除，改 `border-inline: 1px solid rgba(255,255,255,.04)` | 深色底上看不见，纯粹费性能 |
| 正文对齐 | `justify` + `text-indent: 2em` | 默认 `left` + 首行缩进 2em，设置里可切 justify | CJK justify 易出稀疏行 |
| 噪点层 | 无 | 全局 3% opacity 的 SVG noise 覆层 | 消除深色毛玻璃的塑料感 |
| Toast 漂浮动画 | `floatUpDown 3s infinite` | 删除，仅首次引导时用一次 | 永续动画是廉价感来源 |
| 左侧 TOC 按钮 | 吸附 `left: 20px` | 吸附在阅读容器左外侧 `calc(50% - 400px - 60px)` | 大屏上不孤悬 |

### 设计代币分层（Design Tokens）

两层结构，主题切换只改颜色，不影响 motion / space：

```
Tier 1 (不变原子)：--space-*, --radius-*, --z-*, --ease-*, --dur-*
Tier 2 (主题色)：--bg-body, --bg-reader, --text-main, --text-muted, --accent, --glass-bg, --glass-border
```

### Motion 清单

| 场景 | 实现 |
|:---|:---|
| 书架卡片入场 | stagger 50ms，`translateY(12px) → 0`，opacity `0 → 1`，`cubic-bezier(0.22, 1, 0.36, 1)` |
| 进入阅读器 | **View Transitions API**（书名从卡片 morph 到 reader 标题位置） |
| 章节切换 | 旧章 `translateY(-12px) + blur(4px) + opacity 0`，新章反向，300ms |
| 沉浸淡出 | 3 秒无操作，UI 所有元素 `opacity 0 + translateY(-4px)`，600ms；鼠标 hover 在 UI 元素上时**阻止**隐藏 |
| 进度数字 | 翻牌式切换（`overflow: hidden` + 数字堆叠竖向位移） |
| 章节切换后 HUD | 1.5 秒短暂显示"第 N 章 / 共 M 章"，自动淡出 |

### 可访问性（A11y）基线

- 正文对比度 ≥ 7:1（AAA），UI 文字 ≥ 4.5:1（AA）
- 所有交互元素有 `:focus-visible`（accent 色 2px outline + 2px offset）
- 键盘：`← / →` 翻章，`J / K` 滚动，`B` 书签，`/` 搜索，`Esc` 关闭浮层，`G` 跳转目录
- `prefers-reduced-motion: reduce` 时禁用 View Transitions 和位移动画

---

## 1. 架构决策记录（ADR）

| # | 决策 | 选择 | 理由 |
|:---|:---|:---|:---|
| ADR-1 | 认证方式 | **HTTP-only Cookie Session**，抽象为 `AuthProvider` 接口 | 同源 SPA，Cookie 自动携带；避免前端管理 token 的 XSS 风险；接口化以便未来替换为用户自有的鉴权 SDK，见 §2.6 |
| ADR-2 | 多端同步粒度 | 全部资源按 `user_id` 隔离；设置 / 进度 / 书签 / 书籍 | 满足多端 + 多用户 |
| ADR-3 | 进度数据模型 | `{chapter_idx, char_offset}`，百分比仅派生 | 字号 / 窗口 / 设备变了也能准确还原 |
| ADR-4 | 章节正文传输 | `{paragraphs: string[]}`（后端切段） | 避免前端 / 后端重复实现段落切分逻辑 |
| ADR-5 | 设置存储位置 | 后端（随 user_id） + 前端 localStorage 作引导期缓存 | 多端同步的前提是以后端为事实源 |
| ADR-6 | 部署形态 | Go `embed.FS` 打包 `web/dist`，同源 + NoRoute → `index.html` | 单二进制；SPA history 模式可用 |
| ADR-7 | 密码存储 | **bcrypt**（cost=12） | 标准做法 |
| ADR-8 | 会话存储 | 数据库 `sessions` 表（session_id → user_id + expires_at） | 简单、可撤销；不引入 Redis |
| ADR-9 | CSRF 防护 | Cookie `SameSite=Lax` + 所有写操作走 JSON（非 form） | 同源 SPA 下足够；避开 CSRF token 复杂度 |
| ADR-10 | 封面色生成 | 前端 hash（`BookCard` 用），仅作无封面时 fallback | 未来支持上传封面时不破坏数据 |
| ADR-11 | 用户名字符集 | **不做白名单**，允许任意 Unicode（含中文 / 符号 / emoji），长度 1–64 字符 | 用户体验优先；SQLi 由参数化查询兜底（见 ADR-12） |
| ADR-12 | SQLi 防护 | **一律使用 `$N` 占位符参数化查询**，禁止任何 `fmt.Sprintf` / 字符串拼接进 SQL | 从根源杜绝注入；与用户名 / 内容字符集无关 |
| ADR-13 | 鉴权可插拔 | 定义 `AuthProvider` 接口，当前实现 `LocalAuthProvider`，未来可替换为用户自研 SDK Provider | 业务代码只依赖接口，不依赖具体认证实现 |
| ADR-14 | 开放注册 | 第一版完全开放；预留 `REGISTRATION_ENABLED` 环境变量开关 | 自部署早期阶段；接入 SDK 后由 SDK 控制 |

---

## 2. 后端扩展（多用户 + 契约化）

### 2.1 新增：User + Session

**`internal/model/user.go`**
```go
type User struct {
    ID           int       `json:"id"`
    Username     string    `json:"username"`
    PasswordHash string    `json:"-"`
    CreatedAt    time.Time `json:"createdAt"`
}

type Session struct {
    ID        string    `json:"-"` // 128-bit random，base64url
    UserID    int       `json:"-"`
    ExpiresAt time.Time `json:"-"`
    CreatedAt time.Time `json:"-"`
}
```

**DB 迁移（新增）**
```sql
CREATE TABLE users (
    id            SERIAL PRIMARY KEY,
    username      VARCHAR(256) UNIQUE NOT NULL,   -- 允许 64 字符 × 最大 4 字节 UTF-8
    password_hash VARCHAR(255) NOT NULL,
    created_at    TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE sessions (
    id         VARCHAR(64) PRIMARY KEY,
    user_id    INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);
```

**用户名校验规则（ADR-11 / ADR-12）**：

- **允许**：任意 Unicode 字符（中文、日文、韩文、emoji、标点、符号）
- **长度**：1–64 个 Unicode code point（`utf8.RuneCountInString(name) in [1, 64]`）
- **清洗**（服务端强制）：
  1. NFC 归一化（`golang.org/x/text/unicode/norm`）
  2. 去首尾空白（`strings.TrimSpace`）
  3. 拒绝包含控制字符 / `\x00` / 零宽字符（ZWSP/ZWJ/ZWNJ）/ 方向覆写字符（LRO/RLO）
  4. 拒绝纯空白串
- **唯一性**：大小写敏感（Alice ≠ alice，Unicode 下去 case fold 复杂度不值得），但在清洗后比对
- **SQLi 防护不靠字符白名单**。Go 的 `database/sql` 配合 PostgreSQL `$N` 占位符已完全免疫 SQL 注入。计划中 **严禁** 以下写法：
  ```go
  // ❌ 禁止
  db.Query(fmt.Sprintf("SELECT ... WHERE username='%s'", name))
  // ✅ 只允许
  db.Query("SELECT ... WHERE username = $1", name)
  ```
  Phase 0 会对整个 service 层做一次 grep 扫描确认 0 处字符串拼接进 SQL。

**现有表迁移**（给所有用户资源表加 `user_id`）：
```sql
ALTER TABLE books       ADD COLUMN user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE chapters    ... -- 通过 book_id 间接关联，无需加字段
ALTER TABLE progress    ADD COLUMN user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE bookmarks   ADD COLUMN user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE;
ALTER TABLE settings    ADD COLUMN user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE;

CREATE INDEX idx_books_user     ON books(user_id);
CREATE INDEX idx_progress_user  ON progress(user_id, book_id);
CREATE INDEX idx_bookmarks_user ON bookmarks(user_id, book_id);
CREATE UNIQUE INDEX idx_settings_user_key ON settings(user_id, key);
```

> **迁移策略**：若 DB 已有数据，迁移时先创建 `user_id=1` 的 `admin` 用户，给所有历史数据回填 `user_id=1`，再加 `NOT NULL`。

### 2.2 上传目录按用户分隔

```
uploads/
  1/                  # user_id=1
    book-xxx.txt
  2/
    book-yyy.txt
```

避免不同用户同名文件冲突，也便于未来导出 / 备份。

### 2.3 新增 API（Auth）

| 方法 | 路径 | 请求 | 响应 | 说明 |
|:---|:---|:---|:---|:---|
| POST | `/api/auth/register` | `{username, password}` | `{user: {...}}` + Set-Cookie | 注册即登录；用户名 1–64 Unicode 字符（见 §2.1），密码 ≥ 8；`REGISTRATION_ENABLED=false` 时返回 403 |
| POST | `/api/auth/login` | `{username, password}` | `{user: {...}}` + Set-Cookie | 失败统一 401，不暴露用户名是否存在 |
| POST | `/api/auth/logout` | — | 204 + 清 Cookie | 服务端删除 session 记录 |
| GET  | `/api/auth/me` | — | `{user: {...}}` 或 401 | 前端启动时调用决定路由 |

Cookie 规则：`HttpOnly; Secure（生产）; SameSite=Lax; Path=/; Max-Age=30d`。

### 2.4 Auth 中间件

- 所有 `/api/*`（除 `/api/auth/register` / `/api/auth/login`）必须经过 `RequireAuth`。
- 中间件从 Cookie 读 `session_id` → 查表 → 将 `user_id` 注入 `c.Set("userID", ...)`。
- 未登录返回 `401 {"error": "unauthorized"}`。
- 所有 service 方法签名增加 `userID int` 参数，SQL 查询全部按 `WHERE user_id = $1` 过滤。**切勿依赖 URL 里的 id 做授权**——所有 `book_id/bookmark_id` 的访问都要复核归属权。

### 2.5 契约化响应 DTO

**`GET /api/books/:id/chapters/:idx`**（改动）
```json
{
  "chapterIdx": 3,
  "title": "第三章 ...",
  "paragraphs": ["段落1", "段落2", "..."],
  "charCount": 5234,
  "prevIdx": 2,
  "nextIdx": 4
}
```

**`GET /api/books/:id/progress`** / **`PUT /api/books/:id/progress`**
```json
{
  "bookId": 12,
  "chapterIdx": 3,
  "charOffset": 1820,
  "percentage": 0.4531,
  "updatedAt": "2026-04-21T10:30:00Z"
}
```
后端校验 `chapterIdx` 和 `charOffset` 合法性（越界则 400）。

**`GET /api/books/:id/search?q=`**
```json
{
  "query": "黑暗森林",
  "hits": [
    {
      "chapterIdx": 5,
      "chapterTitle": "第五章 ...",
      "paragraphIdx": 12,
      "charOffset": 340,
      "preview": "...前后 40 字片段 **黑暗森林** 前后..."
    }
  ],
  "total": 23
}
```

**`POST /api/books/upload`**（响应改动）
```json
{ "book": { /* 完整 Book 对象 */ } }
```

**统一错误体**
```json
{ "error": { "code": "UNAUTHORIZED", "message": "请先登录" } }
```
code 枚举：`UNAUTHORIZED / FORBIDDEN / NOT_FOUND / VALIDATION / CONFLICT / INTERNAL`。前端可按 code 分流处理。

### 2.6 AuthProvider 接口（可插拔鉴权）

业务代码（中间件 / handler）**只依赖接口**，不依赖具体认证实现。当前实现 `LocalAuthProvider`；未来替换为你自研的鉴权 SDK 时，只需新实现一个 `ExternalAuthProvider` 注入即可，业务层零改动。

**`internal/auth/provider.go`**
```go
package auth

import "context"

// Principal 是认证成功后返回的"用户主体"，业务层只认它
type Principal struct {
    UserID   int
    Username string
}

// AuthProvider 定义认证能力。任何实现（本地 / 外部 SDK / OIDC）都要满足它
type AuthProvider interface {
    // Register 注册并创建会话；REGISTRATION_ENABLED=false 时实现应返回 ErrRegistrationDisabled
    Register(ctx context.Context, username, password string) (*Principal, SessionToken, error)

    // Login 校验凭据并创建会话；失败统一返回 ErrInvalidCredentials
    Login(ctx context.Context, username, password string) (*Principal, SessionToken, error)

    // Logout 销毁指定会话
    Logout(ctx context.Context, token SessionToken) error

    // Authenticate 由中间件调用，把 token 解析成 Principal；失败返回 ErrUnauthenticated
    Authenticate(ctx context.Context, token SessionToken) (*Principal, error)
}

type SessionToken string

var (
    ErrInvalidCredentials   = errors.New("invalid credentials")
    ErrUnauthenticated      = errors.New("unauthenticated")
    ErrRegistrationDisabled = errors.New("registration disabled")
    ErrUsernameTaken        = errors.New("username taken")
    ErrInvalidUsername      = errors.New("invalid username")
    ErrInvalidPassword      = errors.New("invalid password")
)
```

**`internal/auth/local_provider.go`**（第一版唯一实现）
- bcrypt 密码哈希（cost=12）
- `sessions` 表持久化，`crypto/rand` 生成 32 字节 session_id → base64url
- 执行 ADR-11 用户名校验（NFC / 去空白 / 拒零宽字符 / 长度 1–64）

**`main.go` 依赖注入**
```go
var authProvider auth.AuthProvider = auth.NewLocalProvider(db)
handler.RegisterAuthRoutes(api, authProvider)
api.Use(middleware.RequireAuth(authProvider))
```

**未来接入外部 SDK**：
```go
// 以后的某一天
var authProvider auth.AuthProvider = sdkauth.NewProvider(sdkConfig)
```
业务代码一行不改。

**重要边界**：外部 SDK 的 `user_id` 命名空间可能与现有 `users.id`（SERIAL）冲突。为此预留迁移方案——给 `users` 表加 `external_id VARCHAR NULL UNIQUE`（本地登录时为空，外部 SDK 登录时填 SDK 返回的用户标识）。这条迁移**现在就加**，省得未来改表麻烦：

```sql
ALTER TABLE users ADD COLUMN provider    VARCHAR(32) NOT NULL DEFAULT 'local';
ALTER TABLE users ADD COLUMN external_id VARCHAR(128);
CREATE UNIQUE INDEX idx_users_provider_external ON users(provider, external_id)
    WHERE external_id IS NOT NULL;
```
当前 `LocalAuthProvider` 只写入 `provider='local'` 且 `external_id=NULL` 的记录。

---

## 3. 前端计划

### 3.1 技术栈

| 模块 | 选型 | 理由 |
|:---|:---|:---|
| 构建 | Vite 6 | 极速 HMR |
| 框架 | React 18 | — |
| 样式 | CSS Modules + 全局 tokens | 精确控制，不用 Tailwind |
| 状态 | Zustand | 轻量 |
| 路由 | React Router v6 (data router) | `loader` / `action` 契合 SSR-friendly 结构 |
| 动画 | Motion（React 用）+ CSS / View Transitions API | 高影响时刻用 Motion；其余 CSS |
| 图标 | Lucide React | 线条风 |
| HTTP | 自封装 `fetch` + credentials:include | Cookie 自动携带 |

### 3.2 文件结构

```
web/
├── index.html
├── package.json
├── vite.config.js        # proxy /api → :8080
├── public/
│   ├── favicon.svg
│   └── noise.svg         # 3% opacity 全局纹理
└── src/
    ├── main.jsx
    ├── App.jsx           # 路由 + AuthGate
    ├── styles/
    │   ├── tokens.css    # Tier-1 原子 tokens
    │   ├── themes.css    # Tier-2 三主题
    │   ├── reset.css
    │   ├── typography.css
    │   └── glass.css
    ├── api/
    │   ├── client.js     # fetch 封装（401 → 跳登录；统一错误）
    │   ├── auth.js
    │   ├── books.js
    │   ├── chapters.js
    │   ├── progress.js
    │   ├── bookmarks.js
    │   └── settings.js
    ├── stores/
    │   ├── useAuthStore.js
    │   ├── useBookStore.js
    │   ├── useReaderStore.js
    │   └── useConfigStore.js
    ├── utils/
    │   ├── color-generator.js
    │   ├── reading-time.js
    │   └── throttle.js
    ├── hooks/
    │   ├── useImmersive.js
    │   ├── useKeyboard.js
    │   └── useProgressSync.js
    ├── components/
    │   ├── GlassPanel/
    │   ├── BookCover/          # 封面渐变 + 书名 watermark
    │   ├── BookCard/
    │   ├── UploadZone/
    │   ├── TopBar/
    │   ├── SideTOC/
    │   ├── SettingsPanel/
    │   ├── ChapterHUD/         # 章节切换浮现
    │   ├── SearchOverlay/      # 全书搜索浮层
    │   └── auth/
    │       ├── LoginForm/
    │       └── RegisterForm/
    └── pages/
        ├── Auth/               # 登录 / 注册合并页
        ├── Bookshelf/
        └── Reader/
```

### 3.3 路由与 AuthGate

```jsx
<BrowserRouter>
  <Routes>
    <Route path="/auth" element={<AuthPage />} />
    <Route element={<RequireAuth />}>  {/* 调 /api/auth/me，失败→/auth */}
      <Route path="/" element={<Bookshelf />} />
      <Route path="/read/:bookId" element={<Reader />} />
    </Route>
  </Routes>
</BrowserRouter>
```

应用冷启动：
1. 读 localStorage 缓存的主题立即应用（避免 FOUC 主题闪烁）
2. 调 `/api/auth/me`：
   - 200 → 拉 `/api/settings` 覆盖本地主题
   - 401 → 跳 `/auth`

### 3.4 Auth 页

**单页 tab 切换：登录 / 注册**，不是两个独立路由。
- 毛玻璃卡片居中，背景是大号书名（品牌渲染）+ 噪点
- 登录失败统一文案"账号或密码错误"
- 注册成功即自动登录 → 跳 `/`
- 密码框有显示切换 + 强度指示条（前端仅 UI 提示，后端才是硬规则）

### 3.5 Bookshelf（重新设计，放弃原计划的"Hero + 网格"）

**"书桌隐喻"双栏：**

```
┌─────────────────────────────────────────────────────────────┐
│  TopBar（固定毛玻璃胶囊）                                     │
├──────────────┬──────────────────────────────────────────────┤
│              │  [全部书籍]  [搜索]  [+ 新增]   ← sticky       │
│  近期阅读     │                                              │
│  ┌────────┐  │  ┌──────┐ ┌──────┐ ┌──────┐ ┌──────┐          │
│  │ 大卡 1 │  │  │ BC 1 │ │ BC 2 │ │ BC 3 │ │ BC 4 │          │
│  │ 进度条 │  │  └──────┘ └──────┘ └──────┘ └──────┘          │
│  │ 续读→  │  │  ┌──────┐ ...                                  │
│  └────────┘  │                                                │
│  ┌────────┐  │                                                │
│  │ 大卡 2 │  │                                                │
│  └────────┘  │                                                │
└──────────────┴──────────────────────────────────────────────┘
```

- 左栏（约 30%）：最多 3 本"最近阅读"大卡，显示封面 + 进度条 + "继续第 N 章（剩余约 X 分钟）"
- 右栏：所有书网格，响应式 `repeat(auto-fill, minmax(180px, 1fr))`
- 窄屏（< 900px）：左栏折叠为顶部横滑列表
- 空状态：书架只显示大号拖拽区 + 引导文字，带一次性 toast

**BookCover（新组件）**：
- 无封面时用书名 hash 生成 180° 线性渐变（两色），角度 Hash 决定
- 叠一个装饰性 SVG 形状（Hash 决定：圆 / 斜线 / 点阵 / 正方形 / 三角）
- 右下角压薄荷色 2px 水平细线 + 小号衬线体书名 watermark
- Hover：形状缓慢旋转 8°

**UploadZone**：
- 空书架时占满主区域
- 有书时折叠为右上角"+ 拖入 TXT"小徽标
- 拖入时整个视口出现暗金色虚框 + 放大"释放即上传"提示

### 3.6 Reader（核心）

**布局**：
- 阅读容器：`max-width: 800px; padding: 80px 60px;` 居中
- 章节标题：`--text-main` 加粗 2.2rem，上方 eyebrow "CHAPTER 03" 小号大写字母 accent 色
- 正文：`Noto Serif SC` 1.15rem、行高 1.8、字距 0.02em、首行缩进 2em、默认左对齐
- 顶部毛玻璃胶囊：返回 / 书名 / 搜索 / 书签 / 设置
- 左外侧 TOC 按钮：吸附阅读容器外侧（`calc(50% - 400px - 60px)`）
- 底部进度条：4px，hover 12px，accent 填充；`Ctrl+Click` 跳转百分比
- 右下角固定"阅读元信息"：本章剩余时间（字数/300 估算）+ 章节编号
- 章节切换 HUD：切换时中央 1.5s 浮现 → 淡出

**沉浸模式**（升级版）：
- 3 秒无操作 → UI 淡出
- 鼠标 hover 在 UI 元素上 → 取消隐藏计时
- 进入阅读区域 → 2 秒积极隐藏
- 任意键盘 / 滚轮 / 触控 → 重置计时
- `prefers-reduced-motion` → 禁用淡出动画但保留事件

**进度同步**（ADR-3）：
- `IntersectionObserver` 监听所有段落元素
- 记录视口顶端第一个可见段落的 `paragraphIdx` 与该段起始 `charOffset`
- 滚动时 `throttle(1000ms)` + 停止滚动后 `debounce(500ms)` 调 `PUT /progress`
- 进入阅读器时按 `charOffset` 找到对应段 → `scrollIntoView({block: 'start'})`

**翻页 vs 滚动**：第一版只做滚动；预留 `useReaderStore.mode` 字段（`'scroll' | 'paginated'`）供后续扩展。

**键盘**：
- `← / →` 上 / 下一章
- `J / K` 向下 / 向上滚动一屏
- `B` 在当前进度加书签
- `/` 打开搜索
- `T` 打开目录
- `,` 打开设置
- `Esc` 关闭任意浮层

### 3.7 SettingsPanel（重新设计）

毛玻璃面板从右侧滑入，内含：

1. **主题**：三张"氛围卡片"（不是色块），每张卡片是真正的排版小样：
   - 深渊 · 深夜的书房
   - 晨曦 · 窗边初醒
   - 羊皮纸 · 旧图书馆
2. **阅读预设**：三键切换（紧凑 / 标准 / 宽松），一键调好字号+行距+宽度
3. **精细调节**：字号 / 行距 / 阅读区宽度 / 首行缩进 开关 / 对齐方式（left / justify）
4. **字体**：CJK 阅读字体选择（Noto Serif SC / Songti SC / 系统衬线）
5. **实时预览**：面板顶部固定一段真实正文节选，随设置实时变化

所有改动即时写入本地 + debounce 1s 写后端。

### 3.8 SearchOverlay

- `/` 触发；全屏毛玻璃
- 输入节流 300ms → 调 `/api/books/:id/search`
- 结果列表：章节标题 + 预览（高亮命中）
- 回车跳转：路由到 `/read/:id?chapter=X&offset=Y`，reader 读参数后定位

### 3.9 错误与离线

- `api/client.js` 遇 401 → 清 authStore → 跳 `/auth`
- 5xx → 全局 Toast，内容用 `error.message`
- 网络错误 → Toast "网络连接失败，请稍后重试"
- 阅读器进度同步失败 → 静默重试 3 次，失败入本地队列，下次成功同步时 flush

### 3.10 字体加载策略

- **自托管** Geist / Fraunces / Noto Serif SC（fontsource 包或本地 woff2），避免 Google Fonts 国内延迟
- `font-display: swap`
- `<link rel="preload" as="font" crossorigin>` 首屏关键字重

---

## 4. 完整 API 清单（含新增）

| 动作 | 方法 | 路径 | Auth |
|:---|:---|:---|:---|
| 注册 | POST | `/api/auth/register` | — |
| 登录 | POST | `/api/auth/login` | — |
| 登出 | POST | `/api/auth/logout` | ✓ |
| 当前用户 | GET | `/api/auth/me` | ✓ |
| 书架 | GET | `/api/books` | ✓ |
| 上传 | POST | `/api/books/upload` | ✓ |
| 书籍详情 | GET | `/api/books/:id` | ✓ |
| 删除 | DELETE | `/api/books/:id` | ✓ |
| 章节列表 | GET | `/api/books/:id/chapters` | ✓ |
| 章节正文 | GET | `/api/books/:id/chapters/:idx` | ✓ |
| 搜索 | GET | `/api/books/:id/search?q=` | ✓ |
| 进度读 | GET | `/api/books/:id/progress` | ✓ |
| 进度写 | PUT | `/api/books/:id/progress` | ✓ |
| 书签列表 | GET | `/api/books/:id/bookmarks` | ✓ |
| 建书签 | POST | `/api/books/:id/bookmarks` | ✓ |
| 删书签 | DELETE | `/api/bookmarks/:id` | ✓ |
| 设置读 | GET | `/api/settings` | ✓ |
| 设置写 | PUT | `/api/settings` | ✓ |

---

## 5. 实施阶段

### Phase 0：后端多用户改造
1. 新增 `user` / `session` model + migration（含 `provider` / `external_id` 预留字段，见 §2.6）
2. 现有 4 张资源表 migration：加 `user_id`（首次迁移创建 `admin` 并回填）
3. **`internal/auth/` 包**：定义 `AuthProvider` 接口 + `LocalAuthProvider` 实现 + 用户名清洗工具（NFC / 去空白 / 零宽字符过滤）
4. `internal/handler/auth_handler.go`（依赖 `AuthProvider` 接口，不依赖具体实现）
5. `internal/middleware/auth.go`（`RequireAuth(provider AuthProvider)`）
6. 所有 service 方法增加 `userID` 参数；SQL 加 `WHERE user_id = $1`；handler 从 `c.MustGet("userID")` 取值
7. **SQLi 扫描**：grep 整个 `internal/` 目录确认 0 处 `fmt.Sprintf` / 字符串拼接进 SQL，全部用 `$N` 占位符
8. 上传目录按 `uploads/{user_id}/` 分隔
9. `chapter_service` 改为返回 `paragraphs[]`
10. 搜索 / 进度 DTO 按 §2.5 契约返回
11. `main.go` 依赖注入 `AuthProvider`；挂 `embed.FS` + NoRoute 回落 `index.html`
12. 环境变量：`REGISTRATION_ENABLED=true`（默认开放）、`SESSION_COOKIE_SECURE=true`（生产）

### Phase 1：前端脚手架
`npm create vite@latest web -- --template react`，加 `react-router-dom motion zustand lucide-react`。
字体用 `@fontsource-variable/geist` / `@fontsource-variable/fraunces` / `@fontsource/noto-serif-sc`。
配置 `vite.config.js` proxy `/api → http://localhost:8080`。

### Phase 2：设计系统
`tokens.css` + `themes.css` + `reset.css` + `typography.css` + `glass.css` + `noise.svg` 叠加层。

### Phase 3：API + 状态层
`api/client.js`（fetch 封装 + credentials:include + 401 路由）+ 所有 api/*.js + Zustand stores。

### Phase 4：通用组件
GlassPanel / BookCover / BookCard / UploadZone / TopBar / SideTOC / SettingsPanel / ChapterHUD / SearchOverlay。

### Phase 5：Auth 页 + AuthGate
登录 / 注册 tab 切换 + 启动 `/api/auth/me` 探测。

### Phase 6：Bookshelf 页
双栏"书桌隐喻"布局 + 响应式折叠 + 封面生成。

### Phase 7：Reader 页
像素级还原 + 沉浸 + 进度同步 + 键盘 + 章节 HUD + 搜索。

### Phase 8：生产构建与 embed
```bash
cd web && npm run build         # 产出 web/dist
cd .. && go build -o lumina     # embed web/dist 进二进制
```
Go 端：
```go
//go:embed web/dist
var webFS embed.FS
// Gin: StaticFS("/", ...) + NoRoute serve index.html
```

---

## 6. 验证计划

### 自动化
```bash
cd web && npm run dev       # :5173
cd web && npm run build     # 构建无警告
go test ./...               # 后端单测（auth 中间件 / 授权隔离必测）
```

### 手动核对
1. 注册 `userA`、`userB`，各自上传书籍，互相不可见（授权隔离）
2. `userA` 在设备 1 改字号 → 设备 2 刷新后继承
3. `userA` 在设备 1 读到第 5 章某段 → 设备 2 打开同书自动定位
4. 未登录直接访问 `/read/1` → 跳 `/auth`
5. 浏览器关闭 30 天后 Cookie 过期 → 再访问跳 `/auth`
6. 上传大 TXT（> 10 MB），章节 `paragraphs` 响应时间 < 1s
7. 键盘快捷键全链路走一遍
8. 三主题切换 + 两种对齐方式 + 三个阅读预设 + 两种字体组合跑完
9. 沉浸模式在 hover UI 时不隐藏，离开后 3s 正确淡出
10. `prefers-reduced-motion` 开启时动画禁用
11. 深色模式正文对比度 AAA 通过（Chrome DevTools 检查）
12. `npm run build` 后 Go 二进制单独跑，`localhost:8080` 可访问全部功能（embed 链路通）
13. **安全扫描**：用名如 `admin' OR '1'='1` / `"; DROP TABLE users; --` / `你好👋` / 含零宽字符 `a\u200Bb` 的用户名注册，验证：
    - 正常 Unicode 被接受
    - 零宽 / 控制字符被拒
    - SQL 特殊字符**被正常接受**（用户可以自由取名）但**不触发注入**（参数化查询兜底）
14. `REGISTRATION_ENABLED=false` 启动，注册端点返回 403，登录正常

---

## 7. 风险与未决

| 风险 | 缓解 |
|:---|:---|
| bcrypt cost=12 在低配 VPS 注册 / 登录慢（~200ms） | 可接受；若成瓶颈再降 cost=10 |
| 超长章节（网文几十万字一章）`paragraphs` 体积爆炸 | 后端在 parser 阶段拆 sub-chapter（单章 ≤ 5 万字切分），chapter_idx 保持连续 |
| View Transitions API 非全浏览器支持 | 做特性检测，不支持时回退到简单 fade |
| Cookie `SameSite=Lax` 在第三方嵌入场景失效 | 本产品自部署同源，不支持嵌入；若需要再切 `None + Secure` |
| 章节分段算法在不同 TXT 格式差异大 | 现状够用；未来加"手动调整章节"功能（见第 8 节） |
| 开放注册 + 无速率限制 → 注册爆破 / 批量占用 | Phase 0 加 IP 级 `X per minute` 限流中间件（登录 5/min、注册 3/min） |
| 用户名允许 emoji / 零宽字符被拿去钓鱼（假冒他人） | NFC 归一化 + 拒零宽 + 拒方向覆写已拦住最常见的同形攻击；后续若需严格可加 Unicode confusables 检测（`unicode/security` 包） |
| 未来切 SDK 时 `LocalAuthProvider` 的已有用户迁移 | `users.provider` 字段已预留；迁移脚本标记老用户为 `local`，SDK 用户走 `external_id` 路径，互不干扰 |

---

## 8. 路线图（本计划外，后续）

- 书籍元信息编辑（书名 / 作者 / 封面上传）
- 手动章节切分修正
- 导入 EPUB
- 朗读（TTS）
- 导出 / 备份（SQLite + uploads 打包下载）
- 被动统计（总阅读时长 / 每日阅读曲线）
