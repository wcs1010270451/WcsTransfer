# 试运营上线 Runbook

本文档面向当前 WcsTransfer 版本的一次小规模、白名单、自测型试运营上线。

## 1. 上线前准备

1. 复制根目录 `.env.prod.example` 为 `.env.prod`
2. 至少替换以下字段：
   - `DOMAIN`
   - `PUBLIC_BASE_URL`
   - `POSTGRES_PASSWORD`
   - `AUTH_TOKEN_SECRET`
   - `ADMIN_AUTH_TOKEN_SECRET`
   - `ADMIN_BOOTSTRAP_PASSWORD`
   - `ALERT_WEBHOOK_URL`
3. 如果使用飞书或企微告警，确认 webhook 已可用
4. 运行预检：

```sh
sh deploy/release/preflight-prod.sh .env.prod docker-compose.prod.yml
```

## 2. 首次启动

```sh
docker compose --env-file .env.prod -f docker-compose.prod.yml up -d --build
```

启动后检查：

```sh
docker compose --env-file .env.prod -f docker-compose.prod.yml ps
docker compose --env-file .env.prod -f docker-compose.prod.yml logs --tail=100 backend
docker compose --env-file .env.prod -f docker-compose.prod.yml logs --tail=50 healthz-monitor
```

## 3. 上线后烟雾测试

至少验证：
- `/healthz`
- `/version`
- `/console/`

可直接使用：

```sh
BASE_URL="https://your-domain" \
ADMIN_USER="admin" \
ADMIN_PASSWORD="your-admin-ui-password" \
sh deploy/release/smoke-test.sh
```

## 4. 业务验收最小集

至少完成一次：
- 管理端登录
- 一个租户状态为 `active`
- 一个 `client key` 可正常调用
- 一次 `chat` 请求成功
- 请求日志落库
- 钱包流水扣费落库
- 至少一条告警链路验证成功

## 5. 试运营控制策略

试运营阶段建议保持：
- 仅白名单租户开放
- 不开放自动支付
- 保留人工审核
- 保留人工充值
- 保留人工修账
- 首周每日人工检查一次钱包、账单和错误日志

## 6. 回滚原则

如果出现严重问题，优先级如下：

1. 暂停新租户接入
2. 保留现有数据库数据
3. 回滚应用版本
4. 如需恢复数据库，按 `deploy/BACKUP_AND_RECOVERY.md` 执行

## 7. 当前可上线边界

当前版本适合：
- 小规模试运营
- 邀请制租户
- 白名单用户

当前版本不建议：
- 公开放量
- 自动支付商用
- 承诺正式 SLA
