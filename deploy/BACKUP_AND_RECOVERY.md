# PostgreSQL 备份与恢复

本项目生产环境通过 `docker-compose.prod.yml` 内的 `postgres-backup` 服务执行周期性逻辑备份。

## 自动备份

备份服务使用：

- 镜像：`postgres:16-alpine`
- 脚本：`deploy/backup/backup-postgres.sh`
- 默认输出目录：容器内 `/backups`
- 挂载卷：`postgres_backup_data`

默认策略：

- 每 `24` 小时执行一次
- 保留最近 `7` 天
- 文件格式：`*.sql.gz`
- 如容器内存在 `sha256sum`，同时生成 `*.sha256`

## 相关环境变量

在生产 compose 使用的 `.env.prod` 中配置：

- `BACKUP_INTERVAL_HOURS`
- `BACKUP_RETENTION_DAYS`
- `BACKUP_PREFIX`

## 手动查看备份文件

```powershell
docker volume inspect wcstransfer_postgres_backup_data
```

或进入备份容器：

```powershell
docker compose --env-file .env.prod -f docker-compose.prod.yml exec postgres-backup ls -lah /backups
```

## 手动恢复

恢复脚本：

- `deploy/backup/restore-postgres.sh`

恢复前建议：

- 停掉会写数据库的应用容器
- 确认目标库和备份时间点
- 先保留当前数据库的新备份

示例：

```powershell
docker compose --env-file .env.prod -f docker-compose.prod.yml stop backend
docker compose --env-file .env.prod -f docker-compose.prod.yml exec postgres-backup `
  /bin/sh /scripts/restore-postgres.sh /backups/wcstransfer_wcstransfer_20260421T120000Z.sql.gz
docker compose --env-file .env.prod -f docker-compose.prod.yml start backend
```

## 恢复验收

恢复后至少检查：

- `GET /healthz` 正常
- 管理员可登录
- 最近租户、client key、模型、日志数据可查询
- 钱包余额和流水随机抽样核对

## 最近一次演练

- 时间：`2026-04-21`
- 环境：本地 PostgreSQL `wcstransfer`
- 方式：
  - 使用 `pg_dump` 导出当前库
  - 恢复到临时库 `wcstransfer_restore_verify`
  - 校验核心表和基础行数
- 校验结果：
  - `admin_users=1`
  - `providers=2`
  - `models=4`
  - `client_api_keys=4`
  - `request_logs=13`
  - `tenants=1`
  - `tenant_wallet_ledger=4`

## 当前边界

当前方案是逻辑备份，不是物理热备，不覆盖：

- 秒级 RPO
- 主从切换
- 跨地域灾备

对当前试运营阶段足够，但不等价于高可用数据库方案。
