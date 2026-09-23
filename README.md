# bluebell

一个基于 Go 的论坛 / 社区后端服务，采用经典分层架构（routes → controller → logic → dao），
使用 MySQL 持久化 + Redis 做投票与热榜缓存，JWT 做无状态鉴权；
基于 coder/websocket 实现了 Hub 模式的 WebSocket 实时聊天。

---

## 目录

- [技术栈](#技术栈)
- [项目结构](#项目结构)
- [分层架构](#分层架构)
- [快速开始](#快速开始)
- [配置说明](#配置说明)
- [接口清单](#接口清单)
- [核心设计](#核心设计)
- [WebSocket 聊天（Hub 模式）](#websocket-聊天hub-模式)
- [统一响应与错误处理](#统一响应与错误处理)
- [数据库表结构](#数据库表结构)
- [开发说明](#开发说明)
- [已知问题 / TODO](#已知问题--todo)

---

## 技术栈

| 分类 | 组件 | 版本 | 用途 |
|---|---|---|---|
| Web 框架 | gin | v1.12.0 | HTTP 路由与中间件 |
| ORM | gorm + driver/mysql | v1.31.2 / v1.6.0 | MySQL 数据访问 |
| 缓存 | go-redis | v9.22.0 | 投票、热榜、登录态 |
| 配置 | viper | v1.21.0 | config.yaml 加载与热更新 |
| 日志 | zap + lumberjack | v1.28.0 / v2.0.0 | 结构化日志 + 按大小切割 |
| 鉴权 | golang-jwt | v5.3.1 | HS256 Token 签发与解析 |
| ID 生成 | bwmarrin/snowflake | v0.3.0 | 分布式唯一 user_id / post_id |
| 实时通信 | coder/websocket | v1.8.15 | WebSocket 聊天（Hub 模式） |
| 热重载 | air | — | 开发期自动编译重启 |

Go 版本：`go 1.27.0`（module 名 `web_app`）

---

## 项目结构

```
bluebell/
├── main.go                     程序入口：加载配置 → 初始化组件 → 注册路由 → 优雅关机
├── config.yaml                 本地配置（含密码，已被 .gitignore 排除）
├── .air.toml                   air 热重载配置
│
├── 00-routes/                  路由注册
│   └── routes.go               所有路由 + 中间件挂载点
│
├── 01-controller/              接入层：参数绑定与校验、调用 logic、统一响应
│   ├── common.go               ResponseWithError：BizError → 错误码的翻译中枢
│   ├── user.go                 注册 / 登录 / 登出 / 改密
│   ├── community.go            社区列表 / 社区详情
│   ├── post.go                 发帖 / 帖子详情 / 列表 / 热榜 / 点赞 / 删帖
│   └── ws.go                   WebSocket 握手：验签 → Accept → 构造 Client → 装配双泵
│
├── 02-logic/                   业务逻辑层：编排 dao、组装数据、翻译错误码
│   ├── user.go                 SignUp / SignIn / Signout / UpdateInfo
│   ├── community.go            GetCommunityList / GetCommunity
│   ├── post.go                 CreatePost / GetPost / GetPostList / GetPostListByHot
│   │                           / AssemblePostDetails / PostLike / SyncPostVoteNumSQL / DeletePost
│   └── ws.go                   VerifyToken（聊天握手前的身份验证）
│
├── 03-dao/                     数据访问层
│   ├── mysql/
│   │   ├── mysql.go            GORM 连接池初始化与关闭
│   │   ├── user.go             用户表 CRUD + 密码加解密
│   │   ├── community.go        社区查询 + 存在性检查
│   │   └── post.go             帖子 CRUD + 票数批量同步
│   └── redis/
│       ├── redis.go            Redis 客户端初始化与会话
│       ├── keys.go             Key 定义与构造函数
│       ├── session.go          登录 Token 存取（挤下线机制）
│       └── vote.go             投票记录、票数、发帖时间、HN 热榜分数
│
├── 04-logger/                  日志组件
│   └── logger.go               zap 初始化 + GinLogger + GinRecovery 中间件
│
├── 05-models/                  数据模型与请求参数
│   ├── user.go                 User
│   ├── community.go            Community / CommunityDetail
│   ├── post.go                 Post / ParamCreatePost / ParamPostList / ApiPostDetail
│   ├── params.go               ParamSignUp / ParamSignIn / ParamUpdate
│   ├── hub.go                  Hub（注册表 + 广播）/ Client（连接 + 双泵）/ 全局单例
│   ├── message.go              Message：聊天广播的 JSON 消息体
│   └── creat_table.sql         建表脚本 + 初始社区数据
│
├── 06-middlewares/             中间件
│   └── auth.go                 JWTAuthMiddleware：Token 解析 + 挤下线校验
│
├── 07-pkg/                     公共组件
│   ├── code.go                 ResCode 错误码枚举与文案映射
│   ├── errors.go               BizError + NewBizError / WrapBizError
│   ├── response.go             统一响应体与响应函数
│   ├── jwt/jwt.go              Token 签发与解析
│   └── snowflake/snowflake.go  雪花算法封装
│
├── settings/                   配置加载（viper）
│   └── settings.go
│
└── NOTES.md                    开发过程中的问题记录与知识点笔记
```

> **命名说明**：各层目录带数字前缀（`00-` ~ `07-`），用于在编辑器中固定排序。
> 包名（`package` 声明）保持无前缀原样，因此 import 时统一使用显式别名，例如
> `controller "web_app/01-controller"`。

---

## 分层架构

```
                 HTTP 请求
                     │
                     ▼
        ┌────────────────────────┐
        │  00-routes  路由 + 中间件 │   JWT 鉴权、日志、Recovery
        └───────────┬────────────┘
                    ▼
        ┌────────────────────────┐
        │  01-controller 接入层   │   参数绑定/校验 → 调 logic → 统一响应
        └───────────┬────────────┘
                    ▼
        ┌────────────────────────┐
        │  02-logic  业务逻辑层   │   业务编排、数据组装、错误码翻译
        └───────────┬────────────┘
                    ▼
        ┌────────────────────────┐
        │  03-dao    数据访问层   │   MySQL (GORM)  /  Redis
        └────────────────────────┘

        横切关注点：04-logger（日志）  07-pkg（错误码/响应/JWT/雪花ID）
        数据模型：  05-models
```

**各层职责边界**：

| 层 | 应该做 | 不应该做 |
|---|---|---|
| controller | 参数绑定、基础校验、调用 logic、写响应 | 写业务规则、直接碰数据库 |
| logic | 业务编排、调用多个 dao、组装返回结构、把 dao 错误翻译成业务码 | 写 SQL、读写 HTTP 上下文 |
| dao | 单表 SQL、返回哨兵错误（如 `ErrPostNotExist`） | 判断业务、返回业务码 |
| models | 结构体定义、GORM tag、binding tag | 业务方法 |

---

## 快速开始

### 1. 前置依赖

- Go 1.27+
- MySQL 5.7+ / 8.0
- Redis 5.0+

### 2. 建库建表

```bash
mysql -u root -p -e "CREATE DATABASE db_1 DEFAULT CHARSET utf8mb4 COLLATE utf8mb4_general_ci;"
mysql -u root -p db_1 < 05-models/creat_table.sql
```

`creat_table.sql` 会自动建 `user` / `community` / `post` 三张表，并插入 4 条初始社区数据。

### 3. 配置

复制并按实际环境修改 `config.yaml`（见下一节）。该文件已被 `.gitignore` 排除。

### 4. 运行

```bash
# 方式一：直接运行
go run .

# 方式二：编译后运行
go build -o web_app .
./web_app

# 方式三：air 热重载（推荐开发时使用）
air
```

服务默认监听 `:8080`。

### 5. 验证

```bash
curl http://localhost:8080/api/v1/community
```

---

## 配置说明

`config.yaml` 结构：

```yaml
app:
  name: "web_app"
  mode: "dev"          # dev 时日志同时输出到终端和文件
  port: 8080

log:
  level: "debug"       # debug / info / warn / error
  filename: "web_app.log"
  max_size: 200        # 单个日志文件上限（MB）
  max_backups: 7       # 保留的旧日志文件数
  max_age: 30          # 保留天数

mysql:
  host: "127.0.0.1"
  port: 3306
  user: "root"
  password: "***"
  dbname: "db_1"
  MaxIdleConns: 20     # 空闲连接数
  MaxOpenConns: 200    # 最大连接数

redis:
  host: "127.0.0.1"
  port: 6379
  password: "***"
  db: 0
  pool_size: 30
  min_idle: 15

snowflake:
  starttime: "2026-09-03"   # 雪花算法起始时间
  machineid: 1              # 机器号（多实例部署时必须唯一）
```

> ⚠️ **安全提示**：`config.yaml` 含数据库密码与 Redis 密码，已在 `.gitignore` 中排除。
> 生产环境建议改为环境变量注入，避免密钥入库。

---

## 接口清单

Base URL：`/api/v1`

响应统一为 `{"code": 1000, "msg": "success", "data": ...}`，HTTP 状态码恒为 200。

### 公开接口（无需登录）

| 方法 | 路径 | 说明 | 请求参数 |
|---|---|---|---|
| POST | `/signup` | 用户注册 | Body: `username`, `password`, `re_password` |
| POST | `/signin` | 用户登录，返回 Token | Body: `username`, `password` |

### 需登录接口

需在请求头携带 `Authorization: Bearer <token>`。

| 方法 | 路径 | 说明 | 请求参数 |
|---|---|---|---|
| GET | `/signout` | 退出登录（删除 Redis 中的 Token） | — |
| PATCH | `/update` | 修改密码 | Body: `old_password`, `new_password` |
| GET | `/community` | 社区列表 | — |
| GET | `/community/:id` | 社区详情 | Path: `id` |
| POST | `/post` | 发布帖子 | Body: `community_id`, `title`, `content` |
| GET | `/post/:postid` | 帖子详情 | Path: `postid` |
| DELETE | `/post/:postid` | 删除帖子（仅作者本人） | Path: `postid` |
| GET | `/posts` | 帖子分页列表 | Query: `community_id`, `page`, `size` |
| GET | `/posts/hot` | 热榜分页 | Query: `page`, `size` |
| POST | `/like` | 点赞 / 点踩 | Body: `post_id`, `direction`(1=赞, -1=踩) |

**分页参数默认值与边界**（在 controller 层强制收敛）：

- `page` < 1 → 强制为 `1`
- `size` < 1 或 > 100 → 强制为 `10`
- `community_id` < 0 → 返回 `CodeInvalidParam`

### 请求示例

```bash
# 注册
curl -X POST http://localhost:8080/api/v1/signup \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"123456","re_password":"123456"}'

# 登录
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/signin \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"123456"}' | jq -r '.data')

# 发帖
curl -X POST http://localhost:8080/api/v1/post \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"community_id":1,"title":"hello","content":"first post"}'

# 分页查询某社区帖子
curl "http://localhost:8080/api/v1/posts?community_id=1&page=1&size=10" \
  -H "Authorization: Bearer $TOKEN"

# 点赞
curl -X POST http://localhost:8080/api/v1/like \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"post_id":123456789,"direction":1}'
```

### WebSocket 接口

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/ws?token=<JWT>` | 建立聊天连接（全双工长连接） |

该路由注册在 JWT 中间件**之外**，token 通过 URL query 传递、由 controller 内手动验签——
因为浏览器发起 WebSocket 握手时无法携带自定义 `Authorization` 头。
token 无效返回 `{"code":1006}`；curl 等非 WebSocket 客户端访问会得到 `426 Upgrade Required`（预期现象）。

连接建立后，服务端把每条消息以 JSON 广播给**所有在线用户**（含发送者回显）：

```json
{"uid": 7083544471932928, "username": "xiaobao", "msg": "hello", "send_time": 1790082516}
```

- 客户端只需发送 `{"msg":"正文"}`；`uid` / `username` / `send_time` 由服务端覆盖填充，
  伪造身份字段无效
- 服务端每 30s ping 一次探活，10s 内无 pong 判定掉线并清理
- 非 JSON 内容的消息会被服务端直接丢弃

---

## 核心设计

### 1. 主键策略：业务 ID 与自增 ID 分离

每张表都有两个 ID：

- `id` —— 数据库自增主键，仅内部使用
- `user_id` / `post_id` / `community_id` —— **对外暴露的业务 ID**，由雪花算法生成

好处：不暴露业务量、便于分库分表、避免自增 ID 被遍历爬取。

ID 生成入口：`07-pkg/snowflake`，配置项 `snowflake.starttime` 与 `snowflake.machineid`。

### 2. 鉴权与"挤下线"

```
登录  →  签发 JWT（HS256）→  写入 Redis: bluebell:login:{userid} = token（TTL 2h）
请求  →  中间件解析 JWT →  对比 Redis 中的 token
        ├─ 不一致 → 说明账号已在别处登录，旧 token 被挤掉 → 返回 CodeInvalidToken
        └─ 一致   → 通过，将 userid 写入 gin.Context
登出  →  删除 Redis 中的 token 记录
```

Key 定义见 `03-dao/redis/session.go`，中间件见 `06-middlewares/auth.go`。
用户 ID 通过 `middlewares.ContextUserIDKey`（值 `"userID"`）在上下文传递。

### 3. 投票：Redis 为主，MySQL 异步落库

**Redis 数据结构**（见 `03-dao/redis/keys.go`）：

| Key | 类型 | Member | Score | 用途 |
|---|---|---|---|---|
| `post:time` | ZSet | post_id | 发帖时间戳 | 7 天投票窗口期判断 |
| `post:vote_num` | ZSet | post_id | 总票数 | 票数累加 |
| `post:voted:{post_id}` | ZSet | user_id | 1 / -1 | 记录用户投票方向 |
| `post:hotscore` | ZSet | post_id | HN 热度分 | 热榜排序 |

**投票流程**（`02-logic/post.go::PostLike`）：

```
1. 校验 direction ∈ {1, -1}
2. 从 post:time 取发帖时间 → 超过 7 天拒绝（CodeVoteTimeExpired）
3. 从 post:voted:{pid} 取当前用户投票方向 direction_old
4. 若 direction_old == direction → 重复投票，拒绝
5. delta = direction - direction_old，通过 TxPipeline 原子执行：
     ZAdd(post:voted:{pid}, direction)
     ZIncrBy(post:vote_num, delta, pid)
```

**异步落库**：`SyncPostVoteNumSQL` 由 `main.go` 中 5 分钟一次的 ticker 驱动，
把 `post:vote_num` 全量同步到 MySQL `post.vote_num` 字段（事务包裹）。

### 4. 热榜：Hacker News 热度算法

`RecomputeHNScores` 由 1 分钟一次的 ticker 驱动：

```
score = (votes + 1) / (age_hours + 2) ^ 1.5
```

- 分子 `votes + 1` 保证零票帖子也有基础分
- 分母随发布时长增长，实现**新帖加权、老帖衰减**
- `gravity = 1.5` 为常数（`hnGravity`）
- 时钟回拨导致 `age < 0` 时按 0 处理，防止分数爆炸

结果写入 `post:hotscore` ZSet，热榜接口用 `ZRevRangeWithScores` 分页取。

### 5. 列表数据组装（`AssemblePostDetails`）

列表接口拿到的 `posts` 里只有 `author_id` 和 `community_id`，需要补齐用户名与社区名。
采用**索引化组装**，把复杂度从 O(P×U) 降到 O(P+U)：

```
1. 遍历 posts，用 map[K]struct{} 收集去重后的 author_id / community_id
2. 转成 slice，用 WHERE ... IN (...) 批量查库（只查涉及的，不全表扫）
3. 把查询结果索引化成 map：userMap[id]name、communityMap[id]*Community
4. 再次遍历 posts，O(1) 查表填充 ApiPostDetail
```

**容错处理**：若某帖子的 `community_id` 在 community 表中不存在（悬空引用），
不 panic 也不过滤，而是把 `CommunityName` 留空 —— 保证列表条数与分页一致。

```go
communityName := ""
if c, ok := communityMap[p.CommunityID]; ok && c != nil {
    communityName = c.CommunityName
}
```

### 6. 删除帖子：软删除

`post.status` 字段：`1` = 正常，`0` = 已删除。

删除时执行 `UPDATE post SET status = 0`，所有查询都带 `WHERE status = 1` 过滤。
仅帖子作者本人可删（`post.AuthorID != uid` → `CodePermissionDenied`）。

### 7. 配置热更新与优雅关机

- **配置**：`settings.Init()` 中 `viper.WatchConfig()` 监听 `config.yaml` 变更
- **优雅关机**：监听 `SIGINT` / `SIGTERM`，用 5 秒超时的 context 执行 `srv.Shutdown()`，
  并注册 `defer` 依次关闭 MySQL 连接池与 Redis 客户端

### 8. WebSocket 聊天：Hub 模式

REST 接口是"一请求一函数"，聊天是"一连接两 goroutine + 一个常驻 Hub"。
设计参考 gorilla/websocket 的 chat 示例，API 适配 coder/websocket
（其契约：**同一条连接最多 1 个读 goroutine + 1 个写 goroutine**，恰好对应双泵模型）。

```
                        ┌────────────────────────────────────┐
                        │            Hub.Run (单goroutine)    │
   controller           │   for + select 三分支:              │
 ┌──────────┐  register │                                    │
 │ 握手/验签  │ ────────►│  register:   clients[c] = true     │
 │ 构造Client│           │  unregister: delete + close(send)  │
 │ go 写泵   │           │              (守卫防 double-close)  │
 │ 当前goroutine进读泵   │  broadcast:  遍历 clients 塞 send   │
 └──────────┘           │              (满则踢: select+default)│
                        └────────────────────────────────────┘
   读泵 ReadPump                    │  broadcast 通道
   连接→Unmarshal→服务端补身份→Marshal│
                                  ▼
   写泵 WritePump:  for-select { msg := <-send → Conn.Write
                                 ticker.C   → Conn.Ping(探活) }
```

关键决策：

- **Hub 与库解耦**：Hub 只玩 channel 和 map，不 import websocket——模式可复用到任何长连接服务
- **单 goroutine 持有 map**：`clients` 只被 `Hub.Run` 读写（读也不行），
  注册/注销都走 channel 提交，天然无锁并发安全
- **慢客户端踢出**：广播时 `select + default`，信箱（缓冲 256）塞不下说明消费太慢，
  当场 `delete + close`，避免一个死连接卡死整个 Hub（背压取舍）
- **close 权归 Hub**：`send` channel 只有 Hub 能 close（先删 map 再 close 防往已关闭通道投递），
  写泵用 `for msg := range` 感知关闭并退出；unregister 分支带 `if _, ok` 守卫
  防止"踢出 + 正常退出"双路径重复 close 导致 panic
- **身份不可信**：客户端发来的 `uid` / `username` 一律丢弃，由服务端用 JWT 解析出的身份覆盖
- **gin.Context 不进泵**：HTTP handler 返回后 context 会被回收复用，
  常驻泵自己 `context.Background()` 做父 ctx，**每次 Write/Ping 用 `WithTimeout` 派生子 ctx**
  （超时粒度是"每次操作"而非"整个连接"，父 ctx 挂全局限时会把活连接误杀）
- **分层归属**：Hub/Client/Message 是内存领域对象，放 `05-models`（不碰 HTTP、不碰 dao、只被引用）；
  聊天路由注册在 JWT 中间件之外，验签在 controller 内完成

---

## 统一响应与错误处理

### 响应格式

```go
type ResponseData struct {
    Code ResCode     `json:"code"`
    Msg  string      `json:"msg"`
    Data interface{} `json:"data"`
}
```

### 错误码表（`07-pkg/code.go`）

| 码 | 常量 | 文案 |
|---|---|---|
| 1000 | `CodeSuccess` | success |
| 1001 | `CodeInvalidParam` | 请求参数错误 |
| 1002 | `CodeUserExist` | 用户名已存在 |
| 1003 | `CodeUserNotExist` | 用户名不存在 |
| 1004 | `CodeInvalidPassword` | 用户名或密码错误 |
| 1005 | `CodeServerBusy` | 服务繁忙 |
| 1006 | `CodeInvalidToken` | 无效的 token |
| 1007 | `CodeNeedLogin` | 请先登录 |
| 1008 | `CodeCommunityNotExist` | 未找到相关社区 |
| 1009 | `CodePostNotExist` | 未找到该帖子 |
| 1010 | `CodeVoteRepeat` | 您已经给该帖子投过票了 |
| 1011 | `CodeLikeRepeat` | 您已经给这个帖子点过赞了 |
| 1012 | `CodeUnLikeRepeat` | 您已经给这个帖子点过踩了 |
| 1013 | `CodeVoteDirectionInvalid` | 这个帖子您没有点赞 |
| 1014 | `CodeVoteTimeExpired` | 现在时间不支持点赞 |
| 1015 | `CodePermissionDenied` | 权限不足 |
| 1016 | `CodeUpgradefailed` | WebSocket 协议升级失败 |

### 错误传播模式

核心是 `BizError`（`07-pkg/errors.go`），它同时携带**对外的业务码**和**对内的底层错误**：

```go
type BizError struct {
    Code ResCode  // 给客户端看
    Msg  string
    Err  error    // 只进日志，不返回给客户端
}
```

两个构造函数，语义严格区分：

| 构造 | 场景 | 是否保留错误链 | 日志级别 |
|---|---|---|---|
| `NewBizError(code)` | **业务结果**：用户名不存在、密码错误、重复点赞… | 否 | `Debug`（可预期的正常分支） |
| `WrapBizError(code, err)` | **系统故障**：数据库断连、Redis 抖动… | 是（`%w` / `Unwrap`） | `Error`（真故障，需告警） |

**完整链路**：

```
dao 层    返回哨兵错误（ErrPostNotExist / 原始 SQL err）
   ↓
logic 层  errors.Is 判断 → 翻译成 NewBizError 或 WrapBizError
   ↓
controller ResponseWithError 用 errors.As 提取 BizError：
             ├─ 是 BizError → Debug 日志 + 返回其 Code
             └─ 不是       → Error 日志 + 返回 CodeServerBusy（兜底，不泄露内部细节）
```

**原则**：

1. 业务错误用 `Debug`，系统故障用 `Error` —— 避免正常业务分支污染日志、掩盖真故障
2. 底层错误细节（SQL 语句、连接串）**永不返回给客户端**
3. Redis 缓存写失败时**不打断主流程**（发帖、删帖）：缓存可重建，主数据已落库
4. 日志携带结构化字段（`zap.Int64("post_id", pid)`），便于检索

---

## 数据库表结构

| 表 | 说明 | 关键索引 |
|---|---|---|
| `user` | 用户表 | `UNIQUE idx_username`, `UNIQUE idx_user_id` |
| `community` | 社区表 | `UNIQUE idx_community_id`, `UNIQUE idx_community_name` |
| `post` | 帖子表 | `UNIQUE idx_post_id`, `idx_author_id`, `idx_community_id`, `idx_community_status_id`, `idx_status_id` |

`post` 表关键字段：

- `post_id` bigint —— 雪花生成的业务 ID
- `vote_num` int —— 票数（由 Redis 定时同步）
- `status` tinyint —— 1 正常 / 0 已删除
- `content` varchar(8192) —— 列表查询时通过 `Select` 显式排除，避免大字段拖慢 IO

完整 DDL 见 `05-models/creat_table.sql`。

> 注意：`post.community_id` 只有普通索引，**没有外键约束**，
> 因此需要应用层用 `CheckCommunityID` 保证引用有效性（发帖时校验）。

---

## 开发说明

### 命名与目录约定

- 目录数字前缀用于固定排序，包名不带前缀
- import 跨层时统一使用显式别名：`logic "web_app/02-logic"`
- 日志统一走 `zap.L()`，不直接使用 `fmt.Println`

### 日志规范

| 级别 | 使用场景 |
|---|---|
| `Debug` | 业务错误（预期内的分支）、开发调试 |
| `Info` | 关键流程节点（启动、同步完成、请求日志） |
| `Warn` | 可疑但非故障（用户名占用、密码不匹配、脏数据跳过） |
| `Error` | 系统故障（DB/Redis 异常、panic） |
| `Fatal` | 进程级致命错误（监听失败、关机失败） |

### 构建检查

```bash
go build ./...     # 编译检查
go vet ./...       # 静态检查
gofmt -l .         # 格式检查
```

> **排查提示**：若修改代码后行为"没有生效"，先跑 `go build ./...` 确认编译通过。
> air 编译失败时会继续运行旧二进制，日志只显示 `exit status 1`，
> 真实错误在控制台或 `tmp/build-errors.log`。

---

## 已知问题 / TODO

### 待优化

- [ ] `06-middlewares/auth.go` —— `redis.GetLoginToken` 的错误未区分「Redis 故障」与「Token 失效」，
      前者被误报为 `CodeInvalidToken`（应为 `CodeServerBusy`）
- [ ] `main.go` —— 两个 ticker 协程未监听 `ctx.Done()`，`Shutdown` 后仍在操作已关闭的连接池
- [ ] `01-controller/community.go` —— `GetCommunities` 错误分支写死 `CodeServerBusy`，未走 `ResponseWithError`
- [ ] `GET /posts/hot` —— 未显式拒绝 `community_id` 参数（热榜是全局 ZSet，无法按社区过滤）
- [ ] `07-pkg/code.go` —— `CodeVoteDirectionInvalid` 文案"这个帖子您没有点赞"与语义不符
- [ ] `03-dao/redis/keys.go` —— `keyPostScoreZSet` 为死常量，实际用的是 `post:hotscore`
- [ ] `05-models/post.go` —— `Post` 缺 `UpdateTime` 字段（建表 SQL 中有该列）
- [ ] `07-pkg/jwt/jwt.go` —— `mySecret` 硬编码；`GenToken` 内重复写死过期时间，未复用 `TokenExpireDuration`
- [ ] `02-logic/post.go` —— `SyncPostVoteNumSQL` 每 5 分钟全量 UPDATE，可改为仅同步有变化的帖子
- [ ] `02-logic/post.go::PostLike` —— 并发场景下「检查-写入」非原子，同一用户并发请求可能重复计数

### 功能缺失

- [ ] 帖子按票数排序的接口（`post:vote_num` 已就绪，缺对外接口）
- [ ] 用户信息查询接口（发帖列表中作者名已返回，但无独立用户主页接口）
- [ ] 帖子编辑功能（`update_time` 字段与索引已就绪）
- [ ] 单元测试与集成测试

### 聊天功能后续计划

- [ ] 聊天消息落库（MySQL `chat_message` 表）+ 离线消息补发
- [ ] 广播排除发送者自己（当前发送者会收到自己的回显）
- [ ] 私聊（按 UID 定向投递，Hub 名册可按 UID 索引）
- [ ] 消息内容校验/敏感词过滤（走 logic 层）

---

## 相关文档

- `NOTES.md` —— 开发过程中的问题记录、修复方案与 Go 语言知识点笔记

---

*技术选型参考：Go Web 开发实战（bluebell 项目）*
