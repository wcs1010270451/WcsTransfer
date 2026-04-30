# WcsTransfer 功能说明

> 本文档是所有开发者迭代前的必读文件。每次功能变更后同步更新本文档。

---

## 项目定位

WcsTransfer 是一个自托管的 AI 模型网关，部署在 API 客户端与上游 LLM 提供方之间。

**核心职责：**
- 暴露 OpenAI-compatible 接口，屏蔽上游差异
- 管理上游 Provider Key 的健康状态与轮转
- 对用户计费（双价格：成本价 + 售价）
- 提供管理控制台和用户工作台

**当前定位：** 小规模私有部署，白名单用户，人工运营。不适合公开注册和自动商业化。

---

## 架构概览

```
客户端
  │
  ├── /v1/*          公开代理接口（client_api_key 鉴权）
  ├── /portal/*      用户工作台（用户 JWT 鉴权）
  └── /admin/*       管理控制台（管理员 JWT 鉴权）

后端（Go + Gin）
  ├── PostgreSQL     持久化存储（自动迁移）
  └── Redis          限流计数器

前端（React 19 + Vite + Ant Design 5）
  ├── /              首页（管理员/用户双入口）
  ├── /admin/login   管理员登录
  ├── /dashboard     管理控制台
  └── /portal/login  用户登录 → /portal/keys
```

---

## 两端权限模型

| 端 | 路由 | 说明 |
|---|---|---|
| **管理员** | `/admin/*` | 系统全局管理，由 bootstrap 账号登录 |
| **用户** | `/portal/*` | 普通用户，由管理员创建账号 |

> 不支持用户自注册，不支持多管理员体系。

---

## 代理接口（`/v1/*`）

所有接口使用 `X-API-Key` 或 `Authorization: Bearer <client_api_key>` 鉴权。

| 接口 | 说明 |
|---|---|
| `GET /v1/models` | 列出可用模型 |
| `POST /v1/chat/completions` | OpenAI Chat（流式/非流式） |
| `POST /v1/embeddings` | OpenAI Embeddings |
| `POST /v1/messages` | Anthropic Messages API（流式/非流式） |
| `POST /v1/gemini/generate-content` | Gemini Native API（非流式） |
| `POST /v1/gemini/stream-generate-content` | Gemini Native API（流式） |

**请求链路：**

1. `PublicAPIAuth` — 验证 client_api_key，检查用户钱包余额
2. `PublicAPIQuota` — Redis 滑动窗口限流（RPM / 每日请求数 / 每日 Token 数）
3. Handler — 解析模型路由，选 Provider Key，转发上游，记录日志，扣费

---

## 支持的 Provider 类型

| provider_type | 鉴权方式 | 说明 |
|---|---|---|
| `openai` / `openai_compatible` | API Key | OpenAI 及兼容接口（阿里百炼等） |
| `anthropic` | API Key | Anthropic 官方 Messages API |
| `gemini` | API Key（`x-goog-api-key`） | Google AI Studio |
| `vertexai` | ADC（Application Default Credentials） | Google Cloud Vertex AI，无需 API Key |

**Vertex AI 配置（extra_config）：**
```json
{
  "project_id": "your-gcp-project-id",
  "location": "us-central1"
}
```
Provider Key 的 api_key 字段对 vertexai 无效，填占位符即可。ADC 凭据通过 `GOOGLE_APPLICATION_CREDENTIALS` 环境变量或 GCP 运行时自动注入。

---

## 路由策略

每个 Model 可配置 `route_strategy`：

| 策略 | 说明 |
|---|---|
| `fixed` | 始终使用第一个可用 Key |
| `failover` | 主 Key 失败时切换下一个 |
| `round_robin` | 在可用 Key 间轮转 |

Key 健康管理：
- 请求失败 → 短暂冷却（10s–10min，视错误类型）
- 上游 429 → 冷却 30s
- 上游 401 → 冷却 10min
- 冷却状态内存维持，重启后重置

---

## 超时配置

| 参数 | 默认值 | 说明 |
|---|---|---|
| `HTTP_READ_TIMEOUT` | 15s | 读取请求体超时 |
| `HTTP_WRITE_TIMEOUT` | 300s | 写响应超时（含流式传输） |
| Model `timeout_seconds` | 120s | 单个上游请求超时，管理端按模型配置 |

---

## 限流（per client_api_key）

| 维度 | 字段 | 默认 |
|---|---|---|
| 每分钟请求数 | `rpm_limit` | 0（不限） |
| 每日请求次数 | `daily_request_limit` | 0（不限） |
| 每日 Token 数 | `daily_token_limit` | 0（不限） |
| 每日费用上限 | `daily_cost_limit` | 0（不限，仅统计） |
| 每月费用上限 | `monthly_cost_limit` | 0（不限，仅统计） |

---

## 钱包与计费

每个用户持有一个钱包（`wallet_balance`）。

**计费流程：**
1. 请求进入时，用 Token 估算预留金额（`reserved_amount`）
2. 余额 < 预留金额 → 拒绝请求，返回 402
3. 请求成功后，按实际 Token 扣费：`billable_amount = 输入Token × 售价 + 输出Token × 售价`
4. 扣费写入 `tenant_wallet_ledger`，关联 `trace_id`、`model_public_name`、token 明细

**模型双价格配置（per 1M tokens）：**

| 字段 | 说明 |
|---|---|
| `cost_input_per_1m` | 上游成本价（输入） |
| `cost_output_per_1m` | 上游成本价（输出） |
| `sale_input_per_1m` | 对客售价（输入） |
| `sale_output_per_1m` | 对客售价（输出） |
| `reserve_multiplier` | 预留倍率（避免流式低估） |
| `reserve_min_amount` | 最低预留金额 |

**管理员钱包操作：**
- 充值（`/admin/users/:id/wallet/adjust`）
- 账务修正（`/admin/users/:id/wallet/correct`，支持正负数）

---

## Token 统计策略

- 请求进来时 → 用 `estimatePromptTokens` 估算输入 Token，写入日志
- 请求成功 → 用上游返回的 `usage` 真实值覆盖
- 请求失败（429/超时/上下文取消） → 保留估算值，日志中标注 `estimated_prompt_tokens`

---

## 管理控制台（`/admin/*`）

| 功能 | 路由 | 说明 |
|---|---|---|
| 登录 | `POST /admin/auth/login` | 管理员登录，返回 JWT |
| Provider 管理 | `GET/POST/PUT /admin/providers` | CRUD Provider |
| Provider Key 管理 | `GET/POST/PUT /admin/keys` | CRUD Provider Key |
| 模型管理 | `GET/POST/PUT /admin/models` | CRUD 模型映射 |
| 用户管理 | `GET/POST /admin/users` | 创建用户、查看用户列表 |
| 用户状态 | `PUT /admin/users/:id/status` | 启用/停用 |
| 重置密码 | `POST /admin/users/:id/reset-password` | 管理员重置用户密码 |
| 钱包充值 | `POST /admin/users/:id/wallet/adjust` | 充值（正数） |
| 账务修正 | `POST /admin/users/:id/wallet/correct` | 人工修账（正负均可） |
| 钱包流水 | `GET /admin/users/:id/wallet/ledger` | 分页查看流水 |
| 账单导出 | `GET /admin/users/:id/billing/export` | 按条件导出 CSV |
| Client Key 管理 | `GET/POST/PUT /admin/client-keys` | CRUD client_api_key |
| 请求日志 | `GET /admin/logs` | 分页查询日志 |
| 日志详情 | `GET /admin/logs/:id` | 查看完整请求/响应 payload |
| 日志导出 | `GET /admin/logs/export` | 按条件导出 CSV |
| 统计看板 | `GET /admin/stats` | 请求量、Token、成本、毛利等 |
| 对账 | `GET /admin/reconciliation/users` | 钱包与账本一致性核对 |
| 调试（dev） | `POST /admin/debug/*` | 调试代理，仅 `ENABLE_ADMIN_DEBUG=true` 时可用 |

---

## 用户工作台（`/portal/*`）

| 功能 | 路由 | 说明 |
|---|---|---|
| 登录 | `POST /portal/auth/login` | 用户登录，返回 JWT |
| 我的信息 | `GET /portal/me` | 当前用户信息（含钱包余额） |
| 可用模型 | `GET /portal/models` | 列出已授权模型 |
| 统计 | `GET /portal/stats` | 本用户请求量/Token/费用统计 |
| 钱包流水 | `GET /portal/wallet/ledger` | 分页查看自己的流水 |
| 账单导出 | `GET /portal/billing/export` | 导出 CSV |
| 请求日志 | `GET /portal/logs` | 分页查看自己的日志 |
| 日志详情 | `GET /portal/logs/:id` | 查看自己的请求详情 |
| Client Key 列表 | `GET /portal/client-keys` | 查看自己的 client key |
| 创建 Client Key | `POST /portal/client-keys` | 创建新 key |
| 禁用 Client Key | `POST /portal/client-keys/:id/disable` | 停用 key |

---

## 告警体系

所有告警通过 `ALERT_WEBHOOK_URL` 推送，支持 `generic`、`wecom`（企微）、`feishu`（飞书）格式。

| 告警类型 | 触发条件 |
|---|---|
| Provider 429/5xx 激增 | 滑动窗口内超阈值，单 Provider 只告一次直至恢复 |
| 用户钱包拦截激增 | 短窗口内 `wallet_empty/wallet_below_minimum` 超阈值 |
| 预留金额拦截激增 | 短窗口内 `wallet_reserve_insufficient` 超阈值 |
| 账单扣费异常 | 有成功请求但无对应账本扣费记录 |
| 依赖不可用 | PostgreSQL 或 Redis 连接失败 |
| 服务健康 | `/healthz` 外部 watcher 失联 |
| 账务对账异常 | 钱包余额与账本净额不符 |

---

## 数据库迁移

迁移文件在 `backend/migrations/`，格式为顺序编号 SQL 文件。

**生产环境**：由 docker-compose.prod.yml 中的 `migrate` 服务在后端启动前自动执行。

**本地开发**：手动执行对应 `.up.sql` 文件即可。

当前最新迁移：`0017_remove_tenant_layer`

---

## 生产部署

```bash
# 首次或更新部署
git pull
docker compose --env-file .env.prod -f docker-compose.prod.yml up -d --build

# 查看日志
docker compose --env-file .env.prod -f docker-compose.prod.yml logs --tail=50 backend
docker compose --env-file .env.prod -f docker-compose.prod.yml logs migrate
```

**关键环境变量（.env.prod 必须配置）：**

| 变量 | 说明 |
|---|---|
| `AUTH_TOKEN_SECRET` | JWT 签名密钥（管理员和用户共用） |
| `ADMIN_BOOTSTRAP_USERNAME` | 初始管理员用户名 |
| `ADMIN_BOOTSTRAP_PASSWORD` | 初始管理员密码 |
| `DATABASE_URL` | PostgreSQL 连接串 |
| `REDIS_ADDR` | Redis 地址 |
| `CORS_ALLOWED_ORIGINS` | 前端域名，不能为 `*` |
| `ALERT_WEBHOOK_URL` | 告警 Webhook（可选） |
| `GOOGLE_APPLICATION_CREDENTIALS` | GCP 服务账号 JSON 路径（使用 Vertex AI 时必须） |

---

## 备份与恢复

- **自动备份**：`postgres-backup` 容器，默认每 24 小时 pg_dump，保留 7 天
- **恢复脚本**：`deploy/backup/restore-postgres.sh`
- **恢复流程**：停 backend → 执行恢复脚本 → 启 backend → 验证 `/healthz`

---

## 初始数据配置（部署后人工操作）

| 数据 | 说明 |
|---|---|
| Provider | 至少 1 个，配置 base_url 和 provider_type |
| Provider Key | 至少 1 个（vertexai 类型填占位符） |
| Model | 至少 1 个启用，配置双价格和预留策略 |
| 用户 | 管理员在控制台创建，手动充值 |
| Client Key | 用户在工作台自行创建 |

---

## 已知取舍（刻意不做的事）

- 单管理员，不做管理员 CRUD
- 用户不能自助改密码（管理员重置）
- 钱包修正为纯人工操作
- 监控以 Webhook 告警为主，不集成 Prometheus/Grafana
- 部署形态固定为 PostgreSQL + Redis + Docker Compose
- 不支持自动支付/充值
- 不支持多角色/RBAC
