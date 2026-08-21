# cry-055 实验仪器校准与合格状态管理平台

这是一个离线优先的实验仪器校准管理系统。后端使用 Go 1.24、Gin、pgx、validator 和 zap，PostgreSQL 负责持久化；前端使用 Vue 3、TypeScript、Vite、Pinia 和 `@lucide/vue`。通知、附件、定时扫描和回调均通过本地适配器完成，不依赖第三方 API、CDN、云存储或在线模型。

## 模块职责

- `internal/domain`：实验室、仪器、校准项目、计划、执行结果、证书、不合格处置、告警、审计和使用前校验。
- `internal/application`：构造函数注入的用例服务，负责事务边界、乐观并发控制、状态流转和幂等协作。
- `internal/repository/memory`：并发安全的离线测试仓储，提供事务快照回滚和本地通知记录。
- `internal/repository/postgres`：基于 pgx/pgxpool 的 PostgreSQL 适配器，使用 JSONB 聚合、唯一约束和版本条件更新。
- `internal/transport/http`：`/api/v1` HTTP API、分页/白名单筛选、稳定错误码、字段错误和 `request_id`。
- `internal/platform`：UTC 时钟、受控本地附件存储、本地通知和敏感字段脱敏。
- `web/src/views`：仪器台账、校准日历、执行记录、证书中心、不合格处置、告警中心和追溯历史。

## 本地启动

需要 Go 1.24+、Node.js 22+ 和 PostgreSQL 16+。复制 `.env.example` 后执行 `migrations/001_init.sql` 与 `migrations/002_seed.sql`，再运行 `go run ./cmd/server`。前端目录执行 `npm ci` 和 `npm run dev`，浏览器打开 `http://localhost:5173`。完整离线环境可使用 `docker compose up --build`。

## 配置

`APP_ADDR` 默认 `:8080`；`DATABASE_URL` 是 PostgreSQL 连接串；`ATTACHMENT_ROOT` 默认 `./runtime/attachments`；`REQUEST_TIMEOUT` 默认 `5s`；`SHUTDOWN_TIMEOUT` 默认 `10s`；`MAX_UPLOAD_BYTES` 默认 `10485760`。

## 迁移与状态规则

迁移使用可重复执行的建表语句、唯一索引和部分索引，种子文件只登记迁移版本，不覆盖业务数据。不合格或逾期关键仪器不能通过使用前校验；超差执行会进入不合格处置，依次完成影响批次评估、复检、独立复核和质量负责人恢复确认。校准结果修订生成带 `previous_id` 的新版本，不覆盖旧证据。

## API 与错误

所有业务请求使用 `/api/v1`，列表支持 `page`、`size`、`sort`、`order` 以及白名单 `filter[...]`。错误响应固定包含 `code`、`message`、`field_errors` 和 `request_id`。健康检查为 `/healthz`，数据库就绪检查为 `/readyz`。证书接受 PDF、PNG 和 JPEG，并写入受控本地附件目录。演示鉴权使用 `X-Actor-ID` 和 `X-Actor-Roles` 请求头。

## 测试与实际验证

后端验证命令为 `go build ./...`、`go test ./...`、`go test -race ./...` 和 `go vet ./...`；前端验证命令为进入 `web` 后执行 `npm ci`、`npm test` 和 `npm run build`。当前基线已实际通过后端 build、测试、race、vet 和前端生产构建。
