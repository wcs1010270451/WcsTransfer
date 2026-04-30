# WcsTransfer 迭代记录

> 记录每次有意义的功能变更。开发者每次迭代完成后在此追加一条记录。

---

## 格式规范

```markdown
## vYYYY-MM-DD  简短标题

**变更内容：**
- 做了什么

**影响范围：**
- 影响的接口 / 文件 / 数据库

**部署注意：**
- 是否有数据库迁移、配置变更、不兼容变更
```

---

## v2026-04-30  Vertex AI 支持 + 超时修复 + 错误请求 Token 统计

**变更内容：**
- 新增 `vertexai` provider_type，使用 `google.golang.org/genai` v1.55.0 SDK + ADC 鉴权，无需 API Key
- 请求进来时提前估算输入 Token，写入日志；上游成功后用真实值覆盖，失败时保留估算值
- `HTTP_WRITE_TIMEOUT` 默认值从 60s 提升至 300s，避免大任务流式传输被截断

**影响范围：**
- `backend/internal/api/openai/gemini.go`：provider_type=vertexai 时绕过 API Key 路由，走 SDK
- `backend/internal/api/openai/gemini_vertexai.go`：新文件，全部 Vertex AI SDK 逻辑
- `backend/internal/api/openai/handler.go`：新增 Vertex AI 客户端缓存（按 project:location 懒初始化）
- `backend/internal/config/config.go`：WriteTimeout 默认值修改
- `go.mod`：新增 `google.golang.org/genai` 依赖

**部署注意：**
- 使用 Vertex AI 时需设置 `GOOGLE_APPLICATION_CREDENTIALS` 环境变量
- Provider extra_config 格式：`{"project_id":"xxx","location":"us-central1"}`
- 无数据库迁移

---

## v2026-04-29  三端→两端架构重构（移除 Tenant 层）

**变更内容：**
- 删除"租户"概念，架构从"管理员 + 租户 + 用户"简化为"管理员 + 用户"
- 管理员直接 CRUD 用户，用户直接拥有钱包和 client key
- 用户端不再有工作区状态、租户激活等概念
- 前端导航"租户"→"用户"，路由 `/tenants` → `/users`
- 恢复首页（`LandingPage`），展示管理员/用户双入口
- `portalAuthStore` 移除 tenant 字段
- `userauth` service 替代 `tenantauth`（JWT Claims 去掉 TenantID）

**影响范围：**
- 数据库：迁移 0017（见下方"部署注意"）
- `backend/internal/entity/gateway.go`：移除所有 Tenant 类型
- `backend/internal/repository/interfaces.go`：UserAuthStore / UserClientKeyStore 替换旧接口
- `backend/internal/repository/postgres/store.go`：全面重写
- `backend/internal/service/userauth/`：新建，替代 tenantauth
- `backend/internal/api/tenant/handler.go`：完全重写
- `backend/internal/api/admin/handler.go`：移除租户 CRUD，添加用户 CRUD
- `backend/internal/router/router.go`：路由更新
- `frontend/src/pages/UsersPage.jsx`：新文件，替代 TenantsPage
- `frontend/src/api/client.js`：Tenant API → User API
- 告警服务：`billingalert`、`walletalert`、`reconciliation` 全改为 User 类型

**部署注意（破坏性变更）：**
- 必须执行迁移 `0017_remove_tenant_layer.up.sql`（生产部署时 migrate 服务自动执行）
- 迁移内容：
  - `tenant_users` 新增 `wallet_balance`、`min_available_balance` 字段
  - `tenant_wallet_ledger.tenant_id` 重命名为 `user_id`，外键改指向 `tenant_users`
  - `client_api_keys.tenant_id` 删除，`created_by_user_id` 重命名为 `user_id`
  - 删除 `tenants` 表
- 已有用户 JWT Token 全部失效（Claims 结构变更），用户需重新登录
- 生产环境老数据中租户 ID ≠ 用户 ID 的钱包流水记录会被清理（孤立记录删除）

---

## v2026-04-22  Gemini Native API 适配

**变更内容：**
- 新增 Gemini Native API 代理接口（`/v1/gemini/generate-content` 和 `/v1/gemini/stream-generate-content`）
- 支持 `gemini` provider_type，使用 `x-goog-api-key` 鉴权
- 新增 Gemini 流式响应解析，从 SSE 事件中提取 `usageMetadata`

**影响范围：**
- `backend/internal/api/openai/gemini.go`：新文件
- `backend/internal/router/router.go`：新增路由

**部署注意：**
- 无数据库迁移
- 需在管理端配置 `gemini` 类型 Provider 和对应 API Key

---

## v2026-04-22  初始生产版本

**包含能力：**
- OpenAI-compatible 代理：`chat/completions`、`embeddings`、`models`
- Anthropic Messages API 原生适配
- Provider Key 多键轮转：`fixed` / `failover` / `round_robin`
- Key 健康管理与冷却机制
- 管理员登录（JWT）+ 操作日志
- 多租户体系（已在后续迭代中移除）
- Client API Key 管理，支持模型授权、RPM/每日限流、费用上限
- 钱包计费：双价格、预留机制、扣费流水
- 账务对账定时任务 + Webhook 告警
- Provider 429/5xx 异常告警
- 依赖（PG/Redis）健康告警
- PostgreSQL 自动备份（pg_dump）+ 恢复脚本
- Docker Compose 生产部署（含 Caddy 反代、TLS）
- 管理端前端（React 19 + Ant Design 5）
- 用户工作台前端

**部署注意：**
- 需执行迁移 `0001`–`0016`

---

## 迭代规范

1. **迭代前**：阅读 `FEATURES.md` 了解当前系统状态
2. **迭代中**：遵循 `FEATURES.md` 中的架构约定（接口层级、鉴权方式、命名规范）
3. **迭代后**：
   - 更新 `FEATURES.md` 中受影响的章节
   - 在本文件顶部追加一条版本记录
   - 如有数据库变更，在"部署注意"中明确说明迁移编号和内容
   - 如有破坏性变更（Token 失效、接口路径变更、不兼容数据迁移），必须单独标注
