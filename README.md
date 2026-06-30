# weoa-cli

> Manage WeChat Official Account from the terminal.

`weoa-cli` 是一个命令行工具，用于在终端管理微信公众号后台，支持扫码登录、查看账号信息、同步与查询已发表文章等操作。

## 已实现功能（v0.1）

```
weoa-cli
├── auth
│   ├── login        扫码登录，保存会话
│   ├── logout       登出，清除会话
│   └── status       查看登录状态
├── whoami           查看公众号详细信息
└── publish
    ├── list         列出已发表文章
    └── sync         增量同步文章到本地数据库
```

- **auth login** — 启动浏览器打开 mp.weixin.qq.com，扫码登录后自动保存 Cookie + Token
- **auth login --qr** — 在终端直接打印二维码扫码，适合 SSH/服务器环境
- **auth logout** — 清除本地保存的登录会话
- **auth status** — 检查当前是否已登录，显示账号名和微信号
- **whoami** — 展示公众号名称、微信号、简介、邮箱、粉丝数、分类、所在地、视频号、管理员等信息
- **publish list** — 默认从本地 SQLite 列出未删除文章，支持 `--all`、`--json`；使用 `--remote` 联网从微信后台接口获取
- **publish sync** — 同步未删除文章索引到本地 SQLite，按 `appmsgid` 去重；首次全量，后续增量

## 安装

```bash
git clone https://github.com/mudongliang/weoa-cli.git
cd weoa-cli
go build -o weoa-cli .
```

首次运行时 Playwright 浏览器会自动安装，无需手动操作。

## 使用方法

```bash
# 登录（打开浏览器，用微信扫码）
weoa-cli auth login
weoa-cli auth login --qr              # 终端直接打印二维码

# 查看登录状态
weoa-cli auth status

# 登出
weoa-cli auth logout

# 查看账号信息
weoa-cli whoami

# 首次同步：拉取全部已发表文章到本地
weoa-cli publish sync

# 后续同步：仅拉取新增文章
weoa-cli publish sync

# 从本地数据库列出（默认，无需网络；先运行 publish sync）
weoa-cli publish list
weoa-cli publish list -n 10            # 显示 10 条
weoa-cli publish list --limit 10       # 等同于 -n 10
weoa-cli publish list --all            # 本地全部
weoa-cli publish list --search 内核    # 本地搜索标题、摘要、URL
weoa-cli publish list --json           # JSON 格式
weoa-cli publish list --json | jq '.[] | .title'

# 联网从 API 列出
weoa-cli publish list --remote
weoa-cli publish list --remote -n 10
weoa-cli publish list --remote --all
weoa-cli publish list --remote --json
```

## 技术架构

```
weoa-cli/
├── main.go
├── cmd/                        # Cobra 命令行定义
│   ├── root.go                 # 根命令 & 子命令注册
│   ├── whoami.go               # weoa-cli whoami
│   ├── auth/login.go           # weoa-cli auth login
│   └── publish/
│       ├── list.go             # weoa-cli publish list
│       └── sync.go             # weoa-cli publish sync
├── internal/
│   ├── auth/
│   │   ├── session.go          # Cookie/Token 持久化 (~/.config/weoa-cli/session.json)
│   │   └── login.go            # Playwright 扫码登录流程
│   ├── client/
│   │   ├── client.go           # HTTP 客户端（CookieJar + Token 注入）
│   │   └── settings.go         # 账号详情接口 & 数据模型
│   ├── config/config.go        # 配置路径 (~/.config/weoa-cli/)
│   ├── publish/
│   │   ├── model.go            # 文章数据模型 & API 响应解析
│   │   └── api.go              # PublishService（列表、分页、全量拉取）
│   └── storage/
│       └── sqlite.go           # SQLite 存储（modernc.org/sqlite，纯 Go）
└── go.mod
```

## 关键设计

### 登录 & 会话

```
weoa-cli auth login
  → Playwright 启动 Chromium
  → 打开 mp.weixin.qq.com
  → 用户扫码
  → 拦截跳转 URL，提取 token
  → 保存 Cookie + Token + UA → ~/.config/weoa-cli/session.json
```

之后所有命令直接复用 session，无需再次扫码。Session 过期时重新 `weoa-cli auth login` 即可。

### 文章同步与列表

使用微信后台内部接口 `GET /cgi-bin/appmsgpublish?sub=list`，返回数据结构为双重 JSON 编码：

- `publish_page` 字段是 JSON 字符串（第一层）
- 每条记录内的 `publish_info` 也是 JSON 字符串（第二层）

同步与列表都只保留未删除文章：

- 接口返回 `is_deleted: true` 的文章会被跳过
- 本地列表只查询 `is_deleted = 0` 的记录
- API 列表会过滤删除记录，并按 `appmsgid` 去重

同步策略：

- 本地未删除文章数为 0 时，`publish sync` 执行首次全量同步
- 本地已有未删除文章时，执行增量同步
- 增量同步分页遍历，遇到本地已存在的历史 `appmsgid` 后停止
- 同一次同步中刚插入后又遇到的重复 `appmsgid` 不会触发提前停止
- SQLite 使用 `UNIQUE(appmsgid)` 和 `INSERT OR IGNORE` 保证本地去重

### 本地存储

```
~/.config/weoa-cli/
├── session.json      # 登录会话
└── cache.db          # SQLite 文章索引
```

`articles` 表以 `appmsgid` 为唯一键，存储标题、URL、发布时间、封面、摘要、阅读数、点赞数、删除状态等。`publish list` 默认读取本地 SQLite，并按 `appmsgid` 倒序展示，因此日常查询无需联网；需要查看微信后台实时数据时使用 `--remote`。

## 依赖

| 库 | 用途 |
|---|---|
| `github.com/spf13/cobra` | CLI 框架 |
| `github.com/go-resty/resty/v2` | HTTP 客户端 |
| `github.com/mxschmitt/playwright-go` | 浏览器自动化（登录） |
| `modernc.org/sqlite` | 纯 Go SQLite（无 CGO） |
| `github.com/PuerkitoBio/goquery` | HTML 解析 |

## License

MIT
