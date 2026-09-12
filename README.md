# CIGC 预售系统

Go 1.27.1 + Kratos v3 后端。数据库以 `scripts/schema.sql` 为准，**禁止** GORM AutoMigrate。

日结与金额精度对齐金牛协议：Asia/Shanghai 自然日 `settle_runs` 占位防重；业务金额 `DECIMAL(36,8)`，计算后 `Round(8)`。

前后端分离：用户端仓库 `dapp`（`/api/app_server`），管理端仓库 `dapp-admin`（本仓 `/api/admin_cigc`；nginx 把 `/api/admin_dhb/` 转到同一套接口）。

创世地址必须配置（默认 `0x8Bd86ad98D9fA366E52CFB5c08a3e33f5f412Fb1`），否则无法注册第一个用户。

日结：每天 00:00 Asia/Shanghai 回写 `cap_effective`。本地可：

```bash
curl -s -X POST http://127.0.0.1:8000/api/admin_cigc/settle \
  -H "Authorization: Bearer $ADMIN_TOKEN"
# 同日再跑需 force=1，且 CIGC_ALLOW_FORCE_SETTLE=true
```

## 环境

- Go 1.27.1
- Docker + Compose（MySQL 26.7.0、API、nginx）

```bash
cp .env.example .env
docker compose up --build -d
curl -s http://127.0.0.1:8000/health
curl -s http://127.0.0.1/api/health
```

只起数据库、宿主机跑 API：

```bash
docker compose up -d mysql
export CIGC_DATABASE_DSN='root:root@tcp(127.0.0.1:3306)/cigc?charset=utf8mb4&parseTime=True&loc=Asia%2FShanghai&time_zone=%27%2B08%3A00%27'
export CIGC_JWT_KEY=change-me-jwt-key
export CIGC_ADMIN_USERNAME=admin
export CIGC_ADMIN_PASSWORD=admin123
export CIGC_GENESIS_ADDRESS=0x8Bd86ad98D9fA366E52CFB5c08a3e33f5f412Fb1
make run
```

密钥只走环境变量：`CIGC_JWT_KEY`、`CIGC_ADMIN_PASSWORD`、`CIGC_DATABASE_DSN`、`CIGC_HOT_WALLET_KEY`。

挂载已构建前端：

```bash
DAPP_DIST=../dapp/dist ADMIN_DIST=../dapp-admin/dist docker compose up -d
```

领域术语见 `CONTEXT.md`。

已有 MySQL 数据卷需手工跑增量：

```bash
mysql -h127.0.0.1 -uroot -proot cigc < scripts/migrate_user_placements.sql
mysql -h127.0.0.1 -uroot -proot cigc < scripts/migrate_user_match.sql
```
