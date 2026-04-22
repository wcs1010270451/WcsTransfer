# 生产加固待办

本文档按优先级整理当前版本进入正式生产前建议补齐的事项。

标记说明：
- `√` 已完成
- `-` 未完成

## P0

### 管理端与调试接口收口

- √ 生产环境默认关闭或限制：
  - √ `/admin/debug/chat/completions`
  - √ `/admin/debug/embeddings`
  - √ `/admin/debug/messages`
- √ `/docs`、`/redoc`、`/openapi.json` 已支持通过配置显式关闭
- - 如需公网保留文档入口，增加鉴权或网关层白名单

### CORS 与生产配置检查

- √ 生产环境默认拒绝 `localhost`、`127.0.0.1` 和 `*` CORS 来源
- √ 启动时校验高风险生产配置：
  - √ `ADMIN_TOKEN`
  - √ `AUTH_TOKEN_SECRET`
  - √ `ADMIN_AUTH_TOKEN_SECRET`
  - √ `ENABLE_DOCS`
  - √ `ENABLE_ADMIN_DEBUG`
  - √ `CORS_ALLOWED_ORIGINS`
- √ 代理层基础安全头已补齐
- √ 浏览器侧管理端/租户端敏感响应已禁用缓存
- √ 租户端 Token 已从 `localStorage` 收敛到 `sessionStorage`
- - 如通过浏览器访问管理端，补充更严格的 Token 生命周期检查

### 认证与密钥

- √ 管理员认证从固定 Token 升级到正式登录体系
- √ 管理端操作记录操作者身份
- - 租户用户密码策略加强
  - - 长度要求
  - - 重试限制
  - - 异常登录告警
- - 明确 Token 过期、刷新、撤销策略

### 数据备份与恢复

- √ 给 PostgreSQL 增加定时 `pg_dump`
- √ 形成恢复演练脚本
- - 记录恢复 RTO / RPO 目标

## P1

### 监控与告警

- - 接入 Prometheus / Grafana 或等价方案
- - 增加关键指标
  - - 请求量
  - - 成功率
  - √ 上游 `429` 激增告警（Webhook）
  - √ 上游 `5xx` 激增告警（Webhook）
  - - 钱包拦截次数
  - - 预留金额拦截次数
  - - Provider 切换次数
- √ 钱包余额拦截激增告警（Webhook）
- √ 预留金额拦截激增告警（Webhook）
- √ 账单扣费异常告警（Webhook）
- √ 数据库连接异常告警（Webhook）
- √ Redis 连接异常告警（Webhook）
- √ `/healthz` 外部可用性监控（Webhook）
- - 接入错误聚合平台

### 账务一致性

- √ 增加定时账务对账任务
- - 检查：
  - - `request_logs`
  - - `tenant_wallet_ledger`
  - - `tenants.wallet_balance`
- √ 发现异常时自动告警（Webhook）

### 迁移流程自动化

- - 将 SQL 迁移正式接入部署脚本
- - 发布过程固定执行 migrate
- - 禁止线上临时手工跑结构变更

## P2

### 运营化

- - 租户审核通过邮件通知
- - 钱包余额不足邮件通知
- - 预算预警邮件通知
- - Provider 故障或模型不可用时通知管理员

### 用户体验

- - 继续收尾管理端与租户端中文文案
- - 完善空状态、错误状态、加载状态
- - 优化移动端和窄屏布局

### 性能与前端构建

- - 前端做代码分包，降低主包体积
- - 对大表格页面做懒加载或分页优化

## P3

### 更完整的商业闭环

- - 自动支付 / 充值
- - 发票或账单对账单
- - 套餐、折扣、阶梯价
- - 多角色租户权限

### 更高可靠性

- - 灾备环境
- - 灰度发布
- - 回滚自动化
- - 压测基线和容量评估
