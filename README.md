# Distributed Task Scheduler

一个可以在单机运行的任务调度系统 MVP，用实际案例学习任务状态机、Worker、数据库队列和并发领取。

## 当前功能

- 通过 HTTP API 创建和查询任务
- PostgreSQL 持久化任务状态
- Worker 轮询并执行 `print` 任务
- 使用 `FOR UPDATE SKIP LOCKED` 原子领取任务
- Docker Compose 一键启动

## 快速开始

需要安装 Docker Desktop。

```powershell
docker compose up --build
```

创建任务：

```powershell
$task = Invoke-RestMethod `
  -Method Post `
  -Uri http://localhost:8080/tasks `
  -ContentType application/json `
  -Body '{"type":"print","payload":{"message":"Hello, distributed systems!"}}'

$task
```

查询执行结果：

```powershell
Invoke-RestMethod "http://localhost:8080/tasks/$($task.id)"
```

预期状态最终变为 `succeeded`，Worker 日志会输出消息内容。

停止并删除本地数据：

```powershell
docker compose down --volumes
```

## API

### `POST /tasks`

```json
{
  "type": "print",
  "payload": {
    "message": "Hello"
  }
}
```

可以通过可选的 `scheduled_at` 字段安排未来时间执行：

```json
{
  "type": "print",
  "payload": {
    "message": "Run later"
  },
  "scheduled_at": "2026-10-08T10:00:00Z"
}
```

### `GET /tasks/{id}`

返回任务、状态、执行次数、Worker 和时间信息。

### `GET /healthz`

返回 API 进程的健康状态。

## 项目结构

```text
cmd/api/             HTTP API 入口
cmd/worker/          Worker 入口
internal/httpapi/    HTTP 路由和处理逻辑
internal/task/       任务模型及 PostgreSQL 存储
migrations/          数据库初始化脚本
compose.yaml         本地运行环境
```

## 学习重点

当前版本通过数据库事务避免多个 Worker 同时领取同一任务，但还没有处理 Worker 在执行中宕机的情况。下一阶段将加入租约、超时回收、重试和幂等机制。

## 测试

无需安装 Go，使用 Docker 执行：

```powershell
docker run --rm -v "${PWD}:/src" -w /src golang:1.24-alpine go test ./...
```

