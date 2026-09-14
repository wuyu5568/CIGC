# CIGC 预售系统（单体仓库）

本仓库包含三部分代码：

| 目录 | 说明 |
|------|------|
| **仓库根目录** | 后端：Go 1.27.1 + Kratos v3 + MySQL（`/api/app_server`、`/api/admin_cigc`） |
| [`dapp/`](./dapp) | 用户端：Vite + Vue3（Web3 商城 / 社区 / 钱包等） |
| [`dapp-admin/`](./dapp-admin) | 管理后台：Vue CLI + Ant Design Vue（`publicPath=/admin`） |

数据库以 `scripts/schema.sql` 为准，**禁止** GORM AutoMigrate。领域术语见 [`CONTEXT.md`](./CONTEXT.md)。

## 后端

日结与金额精度：Asia/Shanghai 自然日 `settle_runs` 占位防重；业务金额 `DECIMAL(36,8)`，计算后 `Round(8)`。

创世地址必须配置（默认 `0x8Bd86ad98D9fA366E52CFB5c08a3e33f5f412Fb1`），否则无法注册第一个用户。

```bash
cp .env.example .env
docker compose up --build -d
make smoke
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

已有 MySQL 数据卷按顺序跑增量：

```bash
make migrate-existing
```

## 用户端（dapp）

```bash
cd dapp
cp .env.cigc .env.cigc.local   # 可选
npm ci
npm run dev:cigc               # http://127.0.0.1:5185 ，代理到本地 :8000
npm run build -- --mode cigc   # 产物 dist/，由 nginx 挂载为站点根
```

生产构建使用同源 `/api`（`VITE_API=""`）。

## 管理后台（dapp-admin）

```bash
cd dapp-admin
npm ci
# 开发：同源 /api 由 vue-cli 代理到 :8000
NODE_OPTIONS=--openssl-legacy-provider ADMIN_API= npm run serve
# 生产
NODE_OPTIONS=--openssl-legacy-provider ADMIN_API= npm run build
```

后台静态资源挂在 `/admin/`。nginx 将 `/api/admin_dhb/` 转到后端 `/api/admin_cigc/`。

## 一体部署（Docker）

```bash
# 先分别构建前端，再挂载
cd dapp && npm ci && npx vite build --mode cigc && cd ..
cd dapp-admin && npm ci && NODE_OPTIONS=--openssl-legacy-provider ADMIN_API= npx vue-cli-service build && cd ..
DAPP_DIST=./dapp/dist ADMIN_DIST=./dapp-admin/dist docker compose up -d
```

生产覆盖见 `docker-compose.prod.yml` 与 `scripts/aws-push.sh`。
