# 生产环境操作手册

## 前置条件

### GCP 凭证文件（Vertex AI 必须）

在生产服务器上提前放好凭证文件，路径必须与 `docker-compose.prod.yml` 中的挂载路径一致：

```bash
# 在生产服务器上执行
sudo mkdir -p /secrets
sudo cp gcp-credentials.json /secrets/gcp-credentials.json
sudo chmod 600 /secrets/gcp-credentials.json
```

---

## 快捷脚本（推荐）

脚本位于 `scripts/` 目录（不在 git 中，需手动上传到服务器）。

首次使用先赋予执行权限：

```bash
chmod +x scripts/*.sh
```

| 脚本 | 说明 |
|------|------|
| `scripts/deploy.sh` | 拉取代码 + 全量重建 + 启动 |
| `scripts/restart.sh` | 重启所有服务（不重建） |
| `scripts/restart-svc.sh backend` | 重建并重启指定服务，附带实时日志 |
| `scripts/migrate.sh` | 单独运行数据库迁移 |
| `scripts/stop.sh` | 停止所有服务（保留容器） |
| `scripts/stop.sh --down` | 停止并删除容器（数据卷保留） |

---

## 首次部署

```bash
# 1. 克隆代码
git clone <repo-url> /opt/wcstransfer
cd /opt/wcstransfer

# 2. 准备环境变量
cp .env.prod.example .env.prod   # 按实际值填写
# 必填项：POSTGRES_PASSWORD, AUTH_TOKEN_SECRET, DOMAIN,
#         ACME_EMAIL, ADMIN_UI_USER, ADMIN_UI_PASSWORD_HASH

# 3. 启动全栈（构建镜像 + 运行迁移 + 启动所有服务）
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build
```

---

## 日常更新部署

```bash
cd /opt/wcstransfer

# 1. 拉取最新代码
git pull

# 2. 重新构建并滚动重启（含数据库迁移）
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build
```

> `--build` 会重新构建有变化的镜像；`migrate` 服务会自动跑新迁移后退出。

---

## 只重建并重启某个服务

```bash
# 推荐用脚本
scripts/restart-svc.sh backend
scripts/restart-svc.sh frontend

# 或直接用 docker compose
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build <服务名>
```

---

## 查看日志

```bash
# 实时跟踪 backend 日志
docker compose -f docker-compose.prod.yml logs -f backend

# 查看最近 200 行
docker compose -f docker-compose.prod.yml logs --tail=200 backend

# 查看所有服务日志
docker compose -f docker-compose.prod.yml logs -f
```

---

## 查看服务状态

```bash
docker compose -f docker-compose.prod.yml ps
```

---

## 停止全栈

```bash
# 停止但保留容器和数据卷
docker compose -f docker-compose.prod.yml stop

# 停止并删除容器（数据卷保留）
docker compose -f docker-compose.prod.yml down
```

---

## 回滚

```bash
# 切到上一个稳定 tag/commit
git checkout <stable-tag>

# 重新构建并启动
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build
```

---

## 数据库手动备份 / 恢复

```bash
# 手动触发一次备份
docker compose -f docker-compose.prod.yml exec postgres-backup \
  /bin/sh /scripts/backup-postgres.sh

# 查看备份文件列表
docker compose -f docker-compose.prod.yml exec postgres-backup ls /backups

# 恢复（按提示操作）
docker compose -f docker-compose.prod.yml exec postgres-backup \
  /bin/sh /scripts/restore-postgres.sh /backups/<backup-file>.sql.gz
```

---

## 常见问题

| 现象 | 排查命令 |
|------|---------|
| backend 启动失败 | `docker compose -f docker-compose.prod.yml logs backend` |
| 迁移失败 | `docker compose -f docker-compose.prod.yml logs migrate` |
| GCP 凭证报错 | 确认 `/secrets/gcp-credentials.json` 存在且权限为 `600` |
| 健康检查一直失败 | `docker compose -f docker-compose.prod.yml ps` 查看 health 列 |
| Caddy 证书申请失败 | 确认域名 DNS 已指向本机，`ACME_EMAIL` 已正确设置 |
