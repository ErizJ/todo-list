# Todo List

一个基于 **Go + Gin + GORM** 的轻量待办事项 Web 应用，前端使用 Vue 3 实现。

## 技术栈

| 层级 | 技术 |
|------|------|
| Web 框架 | [Gin](https://github.com/gin-gonic/gin) |
| ORM | [GORM](https://gorm.io) + MySQL 驱动 |
| 数据库 | MySQL |
| 前端 | Vue 3 (CDN)，无需打包 |

## 快速启动

### 1. 准备数据库

确保本地 MySQL 已启动，并创建数据库：

```sql
CREATE DATABASE person_practice CHARACTER SET utf8mb4;
```

### 2. 配置数据库连接（可选）

默认连接参数：`root:root@localhost:3306/person_practice`

如需修改，通过环境变量覆盖：

```bash
export DB_USER=your_user
export DB_PASSWORD=your_password
export DB_HOST=localhost
export DB_PORT=3306
export DB_NAME=person_practice
```

### 3. 启动服务

```bash
go run main.go
```

服务默认监听 `http://localhost:9090`，可通过环境变量 `PORT` 修改端口：

```bash
PORT=8080 go run main.go
```

## 使用说明

打开浏览器访问 `http://localhost:9090`：

- **添加任务**：在输入框输入内容，按 Enter 或点击「+ 添加」
- **完成/撤销**：点击任务左侧的圆圈勾选/取消
- **删除任务**：鼠标悬停在任务上，点击右侧 × 按钮
- **筛选任务**：切换「全部 / 待完成 / 已完成」标签
- **批量清除**：点击「清除已完成」一键删除所有已完成项

## API 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/v1/todo` | 获取所有待办 |
| POST | `/v1/todo` | 新增待办 `{"title": "..."}` |
| PUT | `/v1/todo/:id` | 切换完成状态 |
| DELETE | `/v1/todo/:id` | 删除待办 |
