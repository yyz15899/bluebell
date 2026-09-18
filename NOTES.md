# bluebell 项目笔记

> 记录代码审查中发现的问题、修复方案，以及相关 Go 语言知识点。
> 持续更新。

---

## 一、已修复的问题

### 1.1 路由与鉴权

| 位置 | 问题 | 修复 |
|---|---|---|
| `main.go:16` | 目录重命名后 import 路径未改，编译失败 | `routes "web_app/00-routes"` |
| `main.go` | 雪花算法 `Init` 失败未 `return`，后续 `node` 为 nil 导致 panic | 补 `return` |
| `00-routes/routes.go` | `GET /signout`、`PATCH /update` 注册在 `v1.Use(JWTAuthMiddleware())` **之前**，导致无鉴权 | 移到 `Use` 之后 |

> **知识点**：Gin 在**注册路由时**就把当时的 handler 链拷贝进节点，之后再加中间件对已注册路由**无效**。

### 1.2 Controller 层

| 位置 | 问题 | 修复 |
|---|---|---|
| `01-controller/user.go` | `SignoutHandler` 用 `ParseInt("userID")` 取 ID，必然失败 | `c.GetInt64(middlewares.ContextUserIDKey)` |
| `01-controller/user.go` | `UpdateHandler` 错误分支缺 `return`，且用户身份从请求体取（可越权） | 补 `return`；身份改从 JWT 上下文取 |
| `01-controller/post.go` | `GetPost` 错误分支缺 `return`，造成**双响应** | 补 `return` |
| `01-controller/post.go` | `CreatePost` 错误用写死的 `CodeServerBusy` | 改用 `ResponseWithError(c, err)` |

### 1.3 Logic 层

| 位置 | 问题 | 修复 |
|---|---|---|
| `02-logic/post.go` | `DeletePost` 错误分支缺 `return`，错误被丢弃 | 补 `return` |
| `02-logic/post.go` | `GetPostListByHot` 未检查 `GetPostListByIDs` 的 err | 补 err 检查 |
| `02-logic/post.go` | `GetPostList` 签名不合理、空列表未早退 | 改签名 + 早退 |
| `02-logic/post.go` | `CreatePost` 未校验 community_id 是否存在 | 增加社区存在性校验 |
| `02-logic/post.go` | `CreatePost` 第一版把业务代码塞进 `if err != nil` 分支内 | 改为平铺结构 |
| `02-logic/user.go` | `SignUp` 中 `zap.Error(err)` 在 err 必为 nil 的分支里 | 改 `zap.Warn` + `zap.String` |
| `02-logic/user.go` | `UpdateInfo` 明文存密码、未校验旧密码 | 重写为调用 `mysql.UpdatePassword` + 错误码翻译 |

### 1.4 DAO 层

| 位置 | 问题 | 修复 |
|---|---|---|
| `03-dao/mysql/user.go` | `UpdateInfo` 明文存密码 | 重写为 `UpdatePassword`：查 → 验旧密码 → 加密 → 更新 |
| `03-dao/mysql/community.go` | `CheckCommunityid` 用 `Find` 判断存在性（永不返回 `ErrRecordNotFound`） | 改用 `Count`，返回 `(bool, error)` |
| `03-dao/mysql/post.go` | `GetPostList` 不支持社区过滤、查了 `content` 大字段 | 支持 `community_id` + `Select` 排除 `content` |

### 1.5 Models / Middlewares / pkg

| 位置 | 问题 | 修复 |
|---|---|---|
| `05-models/params.go` | `ParamUpdate` 嵌入 `*User`，改密码时 panic | 去掉嵌入，改 `OldPassword` + `NewPassword` |
| `05-models/post.go` | `ApiPostDetail` 嵌入 `*Community`，导致 `community_id` 在 JSON 中**静默消失** | 改扁平 `CommunityName` 字段 |
| `06-middlewares/auth.go` | `ParseToken` 失败返回 `CodeServerBusy` | 改为 `CodeInvalidToken` |

---

## 二、核心知识点

### 2.1 map 取值缺 key 返回零值

```go
communityMap := make(map[int64]*models.Community)

c := communityMap[999]        // c == nil（指针的零值）
name := c.CommunityName       // PANIC: nil pointer dereference
```

**这是 `AssemblePostDetails` 里加保护的原因**：

```go
communityName := ""
if c, ok := communityMap[p.CommunityID]; ok && c != nil {
    communityName = c.CommunityName
}
```

- `map[K]V` 缺 key → 返回 `V` 的**零值**
- 值为**指针** → 零值是 `nil` → 解引用 panic
- 值为 **string** → 零值是 `""` → 无害（所以 `userMap[p.AuthorID]` 不用判断）
- `ok` 与 `c != nil` 是两件事：`ok` 判断 key 在不在，`c != nil` 判断值有没有意义

**为什么这里必须区分存在性**：如果 value 用值类型 `models.Community`，缺 key 拿到零值结构体，`CommunityName` 是空串——但你就分不清「社区不存在」还是「社区名恰好为空」。指针 + 双重判断才能区分。

### 2.2 嵌入结构体与 JSON 字段冲突

`ApiPostDetail` 同时嵌入 `*Post`（含 `community_id`）和 `*Community`（含 `community_id`）时，`encoding/json` 的规则是：

> 同一层级出现多个同 JSON tag 的字段 → **全部剔除，不报错**

结果就是 `community_id` 从响应里**静默消失**。更隐蔽的是 panic：如果嵌入的是 `*User` 指针而 JSON 中没触发指针分配，`p.User` 保持 `nil`，访问提升字段直接 panic。

**验证方式**：写最小复现程序，用纯标准库实测，不靠记忆下结论。

### 2.3 `make` 的三个参数

```go
make([]T, len, cap)   // 切片：长度 + 容量
make(map[K]V, hint)   // map：第二参数只是容量预分配，不参与任何过滤
```

- `make([]int64, 0, len(idset))` → len=0，cap=len(idset)，**append 零次扩容**
- `make([]int64, len(idset))` → len 已经是 N，再 append 会变成 2N，前面全是 0 ⚠️
- `make(map[K]V, n)` 的 `n` **不会**过滤数据，只是减少 rehash

### 2.4 索引化组装数据（O(N×M) → O(N+M)）

`AssemblePostDetails` 的标准范式：

```
第一步：把辅助数据索引化（各自扫一遍）
    userMap      := map[int64]string              ← 30 个用户扫 1 遍
    communityMap := map[int64]*models.Community   ← 8 个社区扫 1 遍

第二步：遍历主数据，查表填充（O(1) 查找）
    100 条 posts × 2 次 map 查找
```

对比不建 map 的双层循环：`100 × 30 = 3000` 次比较 vs `30 + 100 = 130` 次操作。

**这不是去重**（去重是副产品）。目的是**建立 ID → 值的索引**。因为 `posts` 里只有 ID 没有名字，而遍历 `userlist` 时又不知道哪些帖子会用哪个用户——必须先备好索引，再遍历主数据。

整体复杂度 **O(P + U + C)**（帖子数 + 涉及到的作者数 + 涉及到的社区数），前提是 DAO 层用 `WHERE id IN (...)` 只查涉及的，而不是全表查。

### 2.5 `map[K]struct{}` 做集合

```go
idset := make(map[int64]struct{})
for _, p := range posts {
    idset[p.AuthorID] = struct{}{}
}
```

- `struct{}` 是**类型**，`struct{}{}` 是**该类型的唯一值**
- 空结构体**不占内存**（`unsafe.Sizeof(struct{}{}) == 0`）
- 去重来自 **map 的 key 唯一性**，不是来自 `struct{}`。写 `map[int64]bool` 也能去重，只是多存了个无意义的 `bool`

三步流程：**set 去重 → 转 slice → DAO 用 IN 查询**

### 2.6 `range` 的取值形式

| 写法 | 拿到 | 类型 |
|---|---|---|
| `for _, u := range userlist` | 元素**副本** | `models.User` |
| `for i := range communitylist` | **索引**（不是元素！） | `int` |
| `for i, c := range communitylist` | 索引 + 元素副本 | `int`, `models.Community` |
| `for id := range idset` | map 的 **key** | `int64` |

**为什么社区那段必须用索引**：

```go
communityMap[...] = &communitylist[i]   // ✅ 稳定
```

```go
for _, c := range communitylist {
    communityMap[...] = &c   // ⚠️ Go 1.22 之前所有 value 指向同一地址
}
```

Go 1.22 起改为每次迭代新变量，`&c` 行为正确了，但 `&communitylist[i]` 仍然更明确、不依赖版本语义。

> 如果循环体完全没用索引，写 `for range xs` 即可，别留无用变量名。

### 2.7 `First` / `Find` / `Count` 的差异

| 方法 | 查不到时 | 用途 |
|---|---|---|
| `First` | 返回 `gorm.ErrRecordNotFound` | 取单条，需判错 |
| `Find` | **不报错**，切片为空 | 批量查询，靠 `len()` 判断 |
| `Count` | 返回 0 | 判断存在性（**推荐**） |

**用 `Find` 判断存在性是错的**——它永远不返回 `ErrRecordNotFound`。

### 2.8 `Select` 的变参陷阱

```go
db.Select([]string{"id", "title"})   // ✅ 传切片，GORM 变参展开
db.Select("id", "title")             // ✅ 等价写法
```

注意 `Select` 是变参函数，传切片时要展开。

---

## 三、待处理清单

| # | 位置 | 问题 |
|---|---|---|
| 1 | `06-middlewares/auth.go:42` | `if err != nil \|\| curToken != parts[1]` 未区分 Redis 故障与 token 失效 |
| 2 | `01-controller/post.go` | `GetPostList` / `GetPostListByHot` 错误分支写死 `CodeServerBusy` |
| 3 | `main.go` | ticker 协程未监听 `ctx.Done()`，`Shutdown` 后仍在操作已关闭的连接池 |
| 4 | `02-logic/post.go` | `GetPostListByHot` 未显式拒绝 `community_id` |
| 5 | `07-pkg/code.go` | `CodeVoteDirectionInvalid` 文案错配 |
| 6 | `03-dao/redis/keys.go` | 死常量 `keyPostScoreZSet` |
| 7 | `05-models/post.go` | `Post` 缺 `UpdateTime` 字段 |
| 8 | `07-pkg/jwt/jwt.go` | 硬编码 secret、重复过期时间 |
| 9 | `02-logic/post.go` | `SyncPostVoteNumSQL` 每 5 分钟全量 UPDATE |
| 10 | `02-logic/vote.go` | 点赞并发竞态（暂缓） |

---

## 四、方法论

1. **「改了但没生效」→ 先确认编译通过**。用 `go build ./...` 和 `go vet ./...`，再怀疑逻辑。
2. **语言行为假设用最小复现程序实测**，不靠记忆下结论。
3. **`go build` / `go vet` 发现不了**的逻辑错误：不可达代码、错误分支包裹主流程、双响应、错误码语义错配。
4. **企业级规范要点**：错误响应走统一封装（`ResponseWithError`）；日志分级（`Warn` 记业务拒绝、`Error` 记系统故障）；业务错误用 `NewBizError`（不上抛细节），系统故障用 `WrapBizError`（保留错误链）。

### 错误处理模式

```go
pkg.NewBizError(pkg.CodeCommunityNotExist)        // 业务错误：可预期的拒绝
pkg.WrapBizError(pkg.CodeServerBusy, err)         // 系统故障：保留底层错误链
```

判断时用 `errors.Is(err, mysql.ErrPasswordMismatch)` / `errors.As` 做错误码翻译。

---

## 五、关键代码片段

### `AssemblePostDetails` 完整结构

```go
func AssemblePostDetails(posts []*models.Post) ([]*models.ApiPostDetail, error) {
    // 1. 收集 ID 并去重
    idset := make(map[int64]struct{})
    cidset := make(map[int64]struct{})
    for _, p := range posts {
        idset[p.AuthorID] = struct{}{}
        cidset[p.CommunityID] = struct{}{}
    }

    ids := make([]int64, 0, len(idset))
    cids := make([]int64, 0, len(cidset))
    for id := range idset {
        ids = append(ids, id)
    }
    for cid := range cidset {
        cids = append(cids, cid)
    }

    // 2. 批量查库（IN 查询，只查涉及的）
    userlist, err := mysql.GetUserListByIDs(ids)
    if err != nil { return nil, err }
    communitylist, err := mysql.GetCommunityListByIDs(cids)
    if err != nil { return nil, err }

    // 3. 索引化
    userMap := make(map[int64]string, len(userlist))
    for _, u := range userlist {
        userMap[u.UserID] = u.Username
    }
    communityMap := make(map[int64]*models.Community, len(communitylist))
    for i := range communitylist {
        communityMap[int64(communitylist[i].CommunityID)] = &communitylist[i]
    }

    // 4. 填充
    data := make([]*models.ApiPostDetail, 0, len(posts))
    for _, p := range posts {
        communityName := ""
        if c, ok := communityMap[p.CommunityID]; ok && c != nil {
            communityName = c.CommunityName
        }
        data = append(data, &models.ApiPostDetail{
            AuthorName:    userMap[p.AuthorID],
            Post:          p,
            CommunityName: communityName,
        })
    }
    return data, nil
}
```

### `CreatePost` 社区校验（logic 层）

```go
func CreatePost(p *models.Post) error {
    count, err := mysql.CheckCommunityID(p.CommunityID)
    if err != nil {
        zap.L().Error("sql link err", zap.Error(err))
        return pkg.WrapBizError(pkg.CodeServerBusy, err)
    }
    if !count {
        return pkg.NewBizError(pkg.CodeCommunityNotExist)
    }
    p.PostID = snowflake.GenID()
    if err := mysql.CreatePost(p); err != nil {
        zap.L().Error("CreatePost logic failed", zap.Error(err))
        return pkg.WrapBizError(pkg.CodeServerBusy, err)
    }
    if err := redis.CreatePostRecord(p.PostID, p.CreateTime); err != nil {
        zap.L().Error("redis.CreatePostRecord failed", zap.Error(err))
    }
    return nil
}
```

---

*最后更新：2026-09-18*
