# 生产上线检查清单

本清单面向当前 WcsTransfer 版本的小规模试运营上线。

标记说明：
- `√` 已确认
- `-` 未确认

## 1. 环境与密钥

- - 生产环境 `ADMIN_TOKEN` 已替换为高强度随机值
- - 生产环境 `AUTH_TOKEN_SECRET` 已替换为高强度随机值
- - 生产环境 `ADMIN_AUTH_TOKEN_SECRET` 已替换为高强度随机值
- - 数据库密码、Redis 密码、上游 Provider Key 不使用开发环境值
- - `.env.prod` 与本地开发环境完全隔离
- - 前端 `VITE_API_BASE_URL` 指向正式地址
- - 不存在 `change-me`、本地地址、测试密钥等默认值

## 2. 域名、HTTPS 与入口

- - 前端和后端通过正式域名访问
- - HTTPS 已启用并验证证书自动续期
- - 管理端入口与租户端入口路径已确认
- - `/healthz`、`/version` 可访问
- - 反向代理转发 `/v1/*`、`/portal/*`、`/admin/*`、`/docs`、`/redoc` 行为符合预期

## 3. 管理端与调试接口收口

- √ `POST /admin/debug/chat/completions` 已限制访问
- √ `POST /admin/debug/embeddings` 已限制访问
- √ `POST /admin/debug/messages` 已限制访问
- √ `/docs`、`/redoc`、`/openapi.json` 已支持关闭
- - 如果保留公网访问，已增加鉴权或白名单

## 4. CORS 与安全头

- √ CORS 已支持仅允许正式前端域名
- √ 已拒绝 `localhost`、`127.0.0.1`、`*` 等开发来源用于生产
- √ 代理层已补基础安全头
- - 已确认浏览器侧不会错误缓存或暴露管理 Token

## 5. 数据库迁移

- - 当前数据库已执行至最新迁移
- - 最新迁移至少包含：
  - - `0012_tenant_min_available_balance.up.sql`
  - - `0013_reserved_amount_tracking.up.sql`
  - - `0014_model_reserve_policy.up.sql`
  - - `0015_admin_action_logs.up.sql`
- - 部署脚本或发布流程中固定执行迁移
- - 生产环境不再手工临时跑 SQL

## 6. 数据与配置核对

- - 至少存在一个可用 Provider
- - 至少存在一组可用 Provider Key
- - 至少存在一个启用中的 Model 映射
- - 所有对外模型均已配置：
  - - 成本价
  - - 售价
  - - 预留倍率
  - - 最低预留金额
- - 至少一个租户已审核为 `active`
- - 试运营租户已设置：
  - - `max_client_keys`
  - - 钱包余额
  - - `min_available_balance`

## 7. 业务链路验收

- - `GET /v1/models` 返回符合预期
- - `POST /v1/chat/completions` 非流式调用成功
- - `POST /v1/chat/completions` 流式调用成功
- - `POST /v1/embeddings` 调用成功
- - `POST /v1/messages` 调用 Anthropic 成功
- - `client key` 模型授权可正常生效
- - 配额超限返回正确错误
- - 钱包余额不足时返回正确错误
- - 预留金额不足时返回 `wallet_reserve_insufficient`

## 8. 账单与钱包一致性

- - 请求成功后 `request_logs` 有记录
- - 钱包流水 `tenant_wallet_ledger` 有对应扣费记录
- - 定时账务对账任务已启用并确认运行参数
- - 钱包流水包含：
  - - `trace_id`
  - - `model_public_name`
  - - `total_tokens`
  - - `reserved_amount`
  - - `cost_amount`
  - - `billable_amount`
- - `tenants.wallet_balance` 与流水一致
- - 管理端导出账单正常
- - 租户端导出账单正常

## 9. 备份与恢复

- √ PostgreSQL 已配置自动备份
- √ Redis 持久化策略已确认
- √ 已有数据库恢复脚本
- √ 备份文件落盘位置已确认
- √ 已实际验证一次数据库恢复
- √ 已明确回滚版本和恢复数据的操作步骤

## 10. 监控与告警

- - `/healthz` 已接入监控并完成一次验证
- - 容器退出/重启已接告警
- - 数据库连接异常已接告警
- - Redis 连接异常已接告警
- - `ALERT_WEBHOOK_URL` 已配置为正式企微/飞书/通用 Webhook 地址
- - Provider `429`、`5xx` 激增已接告警并完成一次验证
- - 钱包余额拦截激增已接告警
- - 预留金额拦截激增已接告警
- - 账单扣费异常已接告警

## 11. 试运营建议

- - 仅对白名单租户开放
- - 暂不开放自动支付
- - 暂不承诺正式 SLA
- - 保留人工审核、人工充值、人工客服流程
- - 上线首周至少每日核对一次钱包、账单和错误日志
