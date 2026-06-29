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
- **auth logout** — 清除本地保存的登录会话
- **auth status** — 检查当前是否已登录，显示账号名和微信号
- **whoami** — 展示公众号名称、微信号、简介、邮箱、粉丝数、分类、所在地、视频号、管理员等信息
- **publish list** — 从微信后台接口获取已发表文章，支持 `--all` 遍历全部、`--json` 输出、`--from-db` 本地查询
- **publish sync** — 增量同步文章索引到本地 SQLite，自动去重，重复运行只拉取新文章

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

# 从 API 列出最近文章
weoa-cli publish list
weoa-cli publish list -n 10            # 每页 10 条
weoa-cli publish list --all            # 遍历全部
weoa-cli publish list --json           # JSON 格式

# 从本地数据库列出（秒级，无需网络）
weoa-cli publish list --from-db
weoa-cli publish list --from-db --all
weoa-cli publish list --from-db --json | jq '.[] | .title'
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

### 文章同步

使用微信后台内部接口 `GET /cgi-bin/appmsgpublish?sub=list`，返回数据结构为双重 JSON 编码：

- `publish_page` 字段是 JSON 字符串（第一层）
- 每条记录内的 `publish_info` 也是 JSON 字符串（第二层）

增量同步策略：分页遍历，遇到本地已存在的 `appmsgid` 即停止。

### 本地存储

```
~/.config/weoa-cli/
├── session.json      # 登录会话
└── cache.db          # SQLite 文章索引
```

`articles` 表以 `appmsgid` 为主键，存储标题、URL、发布时间、封面、摘要、阅读数、点赞数等。

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
