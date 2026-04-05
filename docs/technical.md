# 技术文档

本文档结合源码，逐一讲解项目各主要功能的实现方式。

---

## 目录

1. [项目整体架构](#1-项目整体架构)
2. [启动流程](#2-启动流程)
3. [数据库连接与连接池](#3-数据库连接与连接池)
4. [数据模型与表结构](#4-数据模型与表结构)
5. [路由注册](#5-路由注册)
6. [CRUD 功能实现](#6-crud-功能实现)
   - [新增 Todo](#61-新增-todo)
   - [查询所有 Todo](#62-查询所有-todo)
   - [切换完成状态](#63-切换完成状态)
   - [删除 Todo](#64-删除-todo)
7. [统一响应格式](#7-统一响应格式)
8. [前端实现](#8-前端实现)
   - [整体结构](#81-整体结构)
   - [状态管理](#82-状态管理)
   - [加载列表](#83-加载列表)
   - [新增任务](#84-新增任务)
   - [切换状态](#85-切换状态)
   - [删除与批量清除](#86-删除与批量清除)
   - [筛选与统计](#87-筛选与统计)
   - [Toast 通知](#88-toast-通知)
9. [Mock 模式](#9-mock-模式)

---

## 1. 项目整体架构

项目采用经典的分层架构，每一层职责明确：

```
main.go          ← 程序入口，串联各层初始化
├── dao/         ← 数据访问层：管理数据库连接
├── models/      ← 模型层：定义数据结构 + 封装 DB 操作
├── controller/  ← 控制层：处理 HTTP 请求/响应
├── routers/     ← 路由层：注册 URL 与 Handler 的映射
└── templates/   ← 视图层：单页前端应用
```

请求的完整链路：

```
浏览器 → Gin Router → Controller → Model → GORM → MySQL
                  ↑                              ↓
              响应 JSON  ←─────────────────── 查询结果
```

---

## 2. 启动流程

入口文件 `main.go` 按顺序做三件事：

```go
// main.go
func main() {
    dao.InitMySQL()                        // 1. 建立数据库连接 + 配置连接池
    dao.DB.AutoMigrate(&models.Todo{})     // 2. 按 Todo 结构体自动建表/更新表结构
    routers.SetupRouter()                  // 3. 注册路由，启动 HTTP 服务（阻塞）
}
```

`AutoMigrate` 是 GORM 提供的自动迁移功能：它会检查 `todos` 表是否存在，不存在则创建，已存在则只做增量变更（新增字段），**不会删除已有列**，对开发期间迭代模型非常方便。

---

## 3. 数据库连接与连接池

文件：`dao/mysql.go`

### 3.1 环境变量优先

为了避免把数据库密码硬编码在源码里，连接参数通过 `getEnv` 读取环境变量，没有设置则回退默认值：

```go
func getEnv(key, fallback string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return fallback
}

username := getEnv("DB_USER", "root")
pwd      := getEnv("DB_PASSWORD", "root")
db       := getEnv("DB_NAME", "person_practice")
ip       := getEnv("DB_HOST", "localhost")
port     := getEnv("DB_PORT", "3306")
```

这样在生产环境只需设置环境变量，无需修改代码：

```bash
export DB_USER=app_user
export DB_PASSWORD=s3cr3t
```

### 3.2 DSN 拼接与 GORM 初始化

```go
dsn := fmt.Sprintf("%v:%v@tcp(%v:%v)/%v?charset=utf8mb4&parseTime=True&loc=Local",
    username, pwd, ip, port, db)

DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
```

DSN 参数说明：
- `charset=utf8mb4`：支持完整 Unicode（含 emoji）
- `parseTime=True`：将 MySQL 的 `DATETIME` 自动解析为 Go 的 `time.Time`
- `loc=Local`：时间按本机时区处理

### 3.3 连接池配置

`gorm.Open` 底层使用标准库的 `database/sql`，可以通过 `DB.DB()` 获取底层实例并配置连接池：

```go
sqlDB, _ := DB.DB()
sqlDB.SetMaxOpenConns(50)           // 最多同时保持 50 条连接
sqlDB.SetMaxIdleConns(10)           // 连接池中最多保留 10 条空闲连接
sqlDB.SetConnMaxLifetime(time.Hour) // 每条连接最长存活 1 小时，防止 MySQL 强制断开
```

不配置连接池时，默认最大连接数无限制，高并发下会瞬间打爆 MySQL 的连接上限。

---

## 4. 数据模型与表结构

文件：`models/todo.go`

```go
type Todo struct {
    Id     int    `json:"id"    gorm:"primaryKey"`
    Task   string `json:"title" gorm:"size:500;not null"`
    Status bool   `json:"status"`
}
```

这一个结构体同时承担三个角色：

| 角色 | 说明 |
|------|------|
| **Go 结构体** | 在代码里传递数据 |
| **GORM 映射** | `gorm:"..."` tag 决定表字段的类型和约束，AutoMigrate 据此建表 |
| **JSON 序列化** | `json:"..."` tag 决定返回给前端的字段名，`Task` 字段序列化后叫 `title` |

`gorm` tag 说明：
- `primaryKey`：标记为主键，GORM 会自动处理 AUTO_INCREMENT
- `size:500`：限制字段为 `varchar(500)`，不加则默认 `longtext`（浪费空间）
- `not null`：数据库层面加 NOT NULL 约束，拒绝空值写入

对应生成的 MySQL 建表语句：
```sql
CREATE TABLE `todos` (
  `id`     bigint NOT NULL AUTO_INCREMENT,
  `task`   varchar(500) NOT NULL,
  `status` tinyint(1),
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

---

## 5. 路由注册

文件：`routers/routers.go`

```go
func SetupRouter() {
    r := gin.Default() // 包含 Logger + Recovery 中间件

    r.Static("/static", "statics")   // 托管静态资源（CSS/JS/字体）
    r.LoadHTMLGlob("templates/*")    // 加载 HTML 模板
    r.GET("/", controller.InitIndexHtml) // 首页

    v1 := r.Group("/v1")
    {
        v1.POST("/todo",    controller.CreateTodo)
        v1.GET("/todo",     controller.GetAllTodo)
        v1.PUT("/todo/:id", controller.UpdateTodoStatus)
        v1.DELETE("/todo/:id", controller.DeleteTodo)
    }

    port := os.Getenv("PORT")
    if port == "" { port = "9090" }
    r.Run(":" + port)
}
```

路由设计遵循 RESTful 风格：用 HTTP Method 区分操作类型，URL 表示资源，`:id` 是路径参数，在 Handler 里通过 `c.Param("id")` 取到。

`r.Group("/v1")` 给所有业务路由加上统一前缀，方便未来做版本迭代（不影响老客户端的情况下推出 `/v2`）。

---

## 6. CRUD 功能实现

### 6.1 新增 Todo

**Controller** `controller/controller.go:20`：

```go
func CreateTodo(c *gin.Context) {
    var todo models.Todo
    if err := c.ShouldBindJSON(&todo); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"msg": msgFail, "data": err.Error()})
        return
    }
    if err := models.CreateTodo(&todo); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"msg": msgFail, "data": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"msg": msgSuccess, "data": todo})
}
```

`ShouldBindJSON` 会把请求 Body 的 JSON（`{"title": "买牛奶"}`）自动解析并填充到 `todo` 结构体。因为 `Task` 字段的 json tag 是 `"title"`，前端传的 `title` 会映射到 `todo.Task`。

**Model** `models/todo.go:16`：

```go
func CreateTodo(todo *Todo) (err error) {
    if todo.Task == "" {
        return errors.New("待办事项内容不能为空")
    }
    err = dao.DB.Create(todo).Error
    return
}
```

`dao.DB.Create(todo)` 对应的 SQL：
```sql
INSERT INTO `todos` (`task`, `status`) VALUES ('买牛奶', false);
```

GORM 会把自增后的 `id` 回填到 `todo.Id`，所以 Controller 里直接把 `todo` 返回给前端，前端就能拿到带 `id` 的完整对象，无需再查一次数据库。

---

### 6.2 查询所有 Todo

**Controller** `controller/controller.go:54`：

```go
func GetAllTodo(c *gin.Context) {
    todos, err := models.GetAllTodo()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"msg": msgFail, "data": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"msg": msgSuccess, "data": todos})
}
```

**Model** `models/todo.go:58`：

```go
func GetAllTodo() (todoList []*Todo, err error) {
    err = dao.DB.Find(&todoList).Error
    return
}
```

`dao.DB.Find(&todoList)` 对应：
```sql
SELECT * FROM `todos`;
```

返回 `[]*Todo`（指针切片）而非 `[]Todo`（值切片），可以避免大量数据时的内存复制，GORM 内部也更高效。

---

### 6.3 切换完成状态

这是整个项目里最需要注意的操作。

**Model** `models/todo.go:41`：

```go
func UpdateTodoStatus(id string) (err error) {
    tid, err := strconv.Atoi(id)
    if err != nil || tid <= 0 {
        return errors.New("无效的 id")
    }
    result := dao.DB.Model(&Todo{}).Where("id = ?", tid).
        Update("status", dao.DB.Raw("NOT status"))
    if result.RowsAffected == 0 {
        return errors.New("记录不存在")
    }
    return result.Error
}
```

这里使用 **原子 SQL 更新**，对应的 SQL 是：

```sql
UPDATE `todos` SET `status` = NOT status WHERE id = 1;
```

**为什么不用"先查再改再存"？**

如果写成：
```go
// 错误示范 ——  存在并发问题
dao.DB.First(&todo, id)   // ① 读出 status = false
todo.Status = !todo.Status // ② 取反 → true
dao.DB.Save(&todo)         // ③ 写回
```

当两个请求同时到达，步骤 ① 都读到 `false`，步骤 ② 都变成 `true`，步骤 ③ 各写一次 `true`——最终结果是 `true`，而不是来回切换，状态被破坏。

原子 `NOT status` 在数据库内部完成读-取反-写，整个过程不会被中断，天然解决并发问题。

---

### 6.4 删除 Todo

**Model** `models/todo.go:25`：

```go
func DeleteTodo(id string) (err error) {
    tid, err := strconv.Atoi(id)
    if err != nil || tid <= 0 {
        return errors.New("无效的 id")
    }
    result := dao.DB.Delete(&Todo{}, tid)
    if result.RowsAffected == 0 {
        return errors.New("记录不存在")
    }
    return result.Error
}
```

先用 `strconv.Atoi` 把字符串 id 转成整数，非法值（如 `"abc"`、`"0"`、`"-1"`）直接拦截，不走数据库。

`result.RowsAffected == 0` 检测"删了 0 行"的情况——说明这个 id 根本不存在，对应 400 而非 500。

对应 SQL：
```sql
DELETE FROM `todos` WHERE `id` = 1;
```

---

## 7. 统一响应格式

所有接口（除 500 错误外）返回相同的 JSON 结构：

```json
{
  "msg": "success",
  "data": <payload>
}
```

失败时：

```json
{
  "msg": "fail",
  "data": "错误描述"
}
```

HTTP 状态码策略：

| 场景 | 状态码 |
|------|--------|
| 操作成功 | `200 OK` |
| 请求参数有误 / 记录不存在 | `400 Bad Request` |
| 数据库内部错误 | `500 Internal Server Error` |

这一区分在 Controller 里体现：

```go
// 客户端传了非法数据 → 400
c.JSON(http.StatusBadRequest, gin.H{"msg": msgFail, "data": err.Error()})

// 数据库本身出错 → 500
c.JSON(http.StatusInternalServerError, gin.H{"msg": msgFail, "data": err.Error()})
```

---

## 8. 前端实现

文件：`templates/index.html`

前端是一个不依赖构建工具的单页应用，通过 CDN 引入 Vue 3，所有逻辑写在一个 HTML 文件里。

### 8.1 整体结构

```html
<script src="https://unpkg.com/vue@3/dist/vue.global.prod.js"></script>
```

引入 Vue 3 后，`Vue` 对象挂载在全局，可以直接解构使用：

```js
const { createApp, ref, computed, onMounted } = Vue
```

页面分为五个视觉区域：

```
┌─────────────────────────────┐
│  Header（标题 + MOCK 徽标）   │
│  Stats（全部/待完成/已完成）   │
│  Input（输入框 + 添加按钮）    │
│  Filter Tabs（筛选标签）      │
│  Todo List / Empty State     │
│  Footer Bar（清除已完成）     │
└─────────────────────────────┘
```

---

### 8.2 状态管理

用 Vue 3 的 `ref` 定义响应式状态，用 `computed` 派生只读计算值：

```js
const todos    = ref([])       // 所有 todo 数据
const newTitle = ref('')       // 输入框内容
const filter   = ref('all')    // 当前筛选项：all / open / done
const adding   = ref(false)    // 添加请求进行中（防重复提交）
const clearing = ref(false)    // 清除请求进行中
const toasts   = ref([])       // 当前显示的通知列表

// 派生值，自动随 todos 更新
const pendingCount = computed(() => todos.value.filter(t => !t.status).length)
const doneCount    = computed(() => todos.value.filter(t => t.status).length)
const filtered     = computed(() => {
    if (filter.value === 'open') return todos.value.filter(t => !t.status)
    if (filter.value === 'done') return todos.value.filter(t => t.status)
    return todos.value
})
```

`computed` 的优势：只要 `todos` 或 `filter` 变化，`filtered`、`pendingCount`、`doneCount` 自动重新计算，模板直接绑定，不需要手动维护。

---

### 8.3 加载列表

页面挂载后立即拉取数据：

```js
async function loadTodos() {
    try {
        const json = isMockMode
            ? await mockApi.getAll()
            : await realRequest('GET', '/v1/todo')
        todos.value = json.data || []
    } catch (e) {
        toast('加载失败：' + e.message, 'error')
    }
}

onMounted(loadTodos)
```

`onMounted` 对应 Vue 的生命周期钩子，组件渲染完成后执行一次。`|| []` 是防御性处理，防止后端返回 `null` 时前端报错。

---

### 8.4 新增任务

```js
async function handleAdd() {
    if (!newTitle.value || adding.value) return  // 空内容或正在请求中则跳过
    adding.value = true
    try {
        const title = newTitle.value
        const json = isMockMode
            ? await mockApi.create(title)
            : await realRequest('POST', '/v1/todo', { title })
        todos.value.push(json.data)  // 把后端返回的（含 id 的）新对象追加到列表
        newTitle.value = ''
        toast('已添加')
    } catch (e) {
        toast('添加失败：' + e.message, 'error')
    } finally {
        adding.value = false          // 无论成功失败都解除 loading 状态
    }
}
```

关键点：
- `adding.value = true` 会让按钮进入 loading 状态（`:disabled="adding"`），防止用户连点导致重复提交
- 成功后直接 `push(json.data)` 把新对象加入本地列表，不需要重新拉全量列表，性能更好
- `finally` 保证 `adding` 一定被重置，即使请求抛出异常

模板中按钮的绑定：

```html
<button class="btn-add" :disabled="adding || !newTitle" @click="handleAdd">
    <span v-if="adding" class="spinner"></span>
    <span v-else>+ 添加</span>
</button>
```

`adding` 为 `true` 时按钮禁用，图标切换为旋转动画；`newTitle` 为空时同样禁用，避免提交空内容。

---

### 8.5 切换状态

```js
async function handleToggle(todo) {
    try {
        isMockMode
            ? await mockApi.toggle(todo.id)
            : await realRequest('PUT', `/v1/todo/${todo.id}`)
        todo.status = !todo.status  // 请求成功后本地同步翻转
    } catch (e) {
        toast('更新失败：' + e.message, 'error')
        // 请求失败时不修改本地状态，保持与后端一致
    }
}
```

这是乐观更新模式的变体：**等后端成功再翻转本地状态**。如果失败，本地状态不变，用户看到错误提示，数据不会不一致。

模板中 checkbox 的绑定：

```html
<div :class="['checkbox', todo.status ? 'checked' : '']" @click="handleToggle(todo)">
```

`todo.status` 变化时，Vue 自动更新 class，CSS 中 `.checked` 类改变外观（渐变填充 + 显示勾）。

---

### 8.6 删除与批量清除

**单条删除：**

```js
async function handleDelete(todo) {
    try {
        isMockMode
            ? await mockApi.remove(todo.id)
            : await realRequest('DELETE', `/v1/todo/${todo.id}`)
        todos.value = todos.value.filter(t => t.id !== todo.id)
        toast('已删除')
    } catch (e) {
        toast('删除失败：' + e.message, 'error')
    }
}
```

用 `filter` 生成新数组替换旧数组，Vue 检测到引用变化会重新渲染列表。

**批量清除已完成：**

```js
async function clearDone() {
    const done = todos.value.filter(t => t.status)
    try {
        await Promise.all(done.map(t =>
            isMockMode ? mockApi.remove(t.id) : realRequest('DELETE', `/v1/todo/${t.id}`)
        ))
        todos.value = todos.value.filter(t => !t.status)
        toast(`已清除 ${done.length} 项`)
    } catch (e) {
        toast('清除失败：' + e.message, 'error')
        await loadTodos() // 部分失败时重新拉列表，保证与后端同步
    } finally {
        clearing.value = false
    }
}
```

`Promise.all` 把所有删除请求并发发出，而不是串行等待，速度快 N 倍。只要有一个请求失败，`catch` 会执行 `loadTodos()` 重新同步，避免本地状态与后端不一致。

---

### 8.7 筛选与统计

筛选完全在前端完成，不需要额外请求后端：

```js
const filtered = computed(() => {
    if (filter.value === 'open') return todos.value.filter(t => !t.status)
    if (filter.value === 'done') return todos.value.filter(t => t.status)
    return todos.value
})
```

模板中切换标签时只是修改 `filter` 这个 ref：

```html
<button :class="['tab', filter==='all'?'active':'']" @click="filter='all'">全部</button>
```

`filter` 一变，`filtered` computed 自动重新计算，列表立刻更新，整个过程没有任何网络请求。

---

### 8.8 Toast 通知

Toast 是一个轻量的消息提示系统，不依赖任何 UI 库：

```js
function toast(msg, type = 'success') {
    const id = Date.now()
    toasts.value.push({ id, msg, type })
    setTimeout(() => {
        toasts.value = toasts.value.filter(t => t.id !== id)
    }, 2500)
}
```

每次调用 `toast()` 往数组里 push 一条记录，2.5 秒后自动通过 `id` 精确删除这一条，支持同时显示多条而不互相干扰。

模板：

```html
<div class="toast-wrap">
    <div v-for="t in toasts" :key="t.id" :class="['toast', t.type]">{{ t.msg }}</div>
</div>
```

CSS 中用 `animation: toastIn` 实现入场动画，`pointer-events: none` 保证 Toast 不会遮挡用户点击。

---

## 9. Mock 模式

文件：`templates/index.html:373`

通过读取 URL Query 参数决定是否启用 Mock：

```js
const isMockMode = new URLSearchParams(location.search).get('mock') === '1'
    || new URLSearchParams(location.search).get('mock') === 'true'
```

访问 `http://localhost:9090/?mock=1` 时，所有 API 请求不走网络，转而调用内存中的 `mockApi`：

```js
const mockApi = {
    async getAll()      { await delay(); return { msg: 'success', data: [..._mockStore] } },
    async create(title) { await delay(); const t = { id: ++_mockId, title, status: false }; ... },
    async toggle(id)    { await delay(); const t = _mockStore.find(x => x.id === id); t.status = !t.status; ... },
    async remove(id)    { await delay(); _mockStore.splice(i, 1); ... },
}
```

`delay(300)` 模拟真实网络延迟，让 loading 状态能被观察到。Mock 数据在内存中，刷新页面会重置。

所有业务函数（`handleAdd`、`handleToggle` 等）内部通过 `isMockMode` 三元运算符分流：

```js
const json = isMockMode
    ? await mockApi.create(title)         // Mock 环境
    : await realRequest('POST', '/v1/todo', { title })  // 真实后端
```

这样 Mock 和真实请求走完全相同的业务逻辑路径，前端行为完全可验证。
