# 直播竞拍全栈系统

Go 后端 + React 前端 monorepo，对应课题任务清单见 [TASKS.md](./TASKS.md)。

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.22+、Gin |
| 前端 | React 19、TypeScript、Vite |
| 数据 | MySQL 8、Redis 7（Docker Compose） |

## 目录结构

```
zhibo/
├── backend/          # Go API（api / service / domain / infra）
├── frontend/         # React 应用
├── docker-compose.yml
└── TASKS.md
```

## 快速启动

### 1. 基础设施（可选，后续接库时需要）

```bash
docker compose up -d
```

### 2. 后端

```bash
cd backend
cp ../.env.example ../.env   # 可选
go run ./cmd/server
```

默认监听 `http://localhost:8081`（若 8080 被占用可在 `.env` 中改 `PORT`），健康检查：`GET /api/v1/health`

管理端 API（阶段 2A 已完成）见 [docs/api-spec.md](./docs/api-spec.md)。请求头示例：`X-Mock-Open-Id: anchor_001`

主要接口：管理端商品/场次/订单；用户端竞拍列表、快照、出价（`X-Mock-Open-Id: buyer_001`）。

WebSocket 实时通信（阶段 4）：`ws://localhost:8081/api/v1/ws?roomId=room_sess_1&openId=buyer_001`，协议见 [docs/ws-protocol.md](./docs/ws-protocol.md)。

### 3. 前端

```bash
cd frontend
npm install
npm run dev
```

浏览器打开 `http://localhost:5173`，开发环境通过 Vite 代理访问后端 `/api`。

| 入口 | 路径 | 说明 |
|------|------|------|
| 管理后台 | `/admin` | 手机号登录 / 主播注册 |
| 用户端 | `/app` | 竞拍列表、登录注册 |
| 用户直播间 | `/live` | 跳转默认房间 |

演示账号密码均为 `123456`：主播 `13800000001`，买家 `13800000002`。

首次使用需执行 `backend/migrations/003_auth.sql`（为 users 表增加手机号与密码字段）。

## 数据模型（阶段 1）

- 设计文档：[docs/data-model/README.md](./docs/data-model/README.md)（ER、状态机、Redis Key）
- **MySQL 与 Redis 协作**：[docs/mysql-redis.md](./docs/mysql-redis.md)（读写路径、一致性、降级、代码索引）
- DDL / 种子：`backend/migrations/001_schema.sql`、`002_seed.sql`
- 领域模型：`backend/internal/domain/`

```bash
# 首次 docker compose up -d 会自动执行 migrations；已有数据卷时需手动导入：
mysql -h127.0.0.1 -uzhibo -pzhibo zhibo < backend/migrations/001_schema.sql
mysql -h127.0.0.1 -uzhibo -pzhibo zhibo < backend/migrations/002_seed.sql
```

## 常用命令

```bash
# 后端测试
cd backend && go test ./...

# 前端构建
cd frontend && npm run build
```

## 环境变量

复制根目录 `.env.example` 为 `.env` 并按需修改。后端启动时会自动加载项目根目录或 `backend` 目录下的 `.env`（通过 godotenv）。

## 阿里云部署

完整步骤见 **[docs/deploy-aliyun.md](./docs/deploy-aliyun.md)**。

生产演示用 **公网 IP** 访问（免备案）：`http://<IP>/app`、`http://<IP>/admin`。域名上线见 [docs/icp-filing.md](docs/icp-filing.md)。

快速启动（服务器上）：

```bash
cp .env.example .env   # 密码、JWT、FRONTEND_URL=http://你的公网IP
./scripts/manual-deploy.sh
```
