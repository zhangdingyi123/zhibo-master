# 阿里云 ECS 部署完整教程

> **当前推荐（免备案）**：公网 IP 直访  
> - 用户端：`http://47.97.176.185/app`  
> - 主播端：`http://47.97.176.185/admin`  
>
> 域名 `jj520.xyz` / `mgongchang.xyz` 须 ICP 备案，见 [icp-filing.md](./icp-filing.md)

---

## 一、架构概览

```
                    ┌─────────────────────────────────┐
  http://公网IP ───►│  Nginx (80)                     │
                    │  ├─ /api/*  → zhibo-backend     │
                    │  └─ /*      → zhibo-frontend    │
                    │       backend → mysql / redis     │
                    └─────────────────────────────────┘
```

| 组件 | 说明 |
|------|------|
| ECS | 大陆节点，公网 IP 解析域名 |
| 访问 | **IP + 路径**（免备案）；域名见 [icp-filing.md](./icp-filing.md) |
| MySQL | Docker 容器，首次启动自动执行 `backend/migrations/*.sql` |
| Redis | Docker 容器，缓存 + 分布式锁 |
| Nginx | 双域名反代 API / 前端 |

---

## 二、购买与准备

### 2.1 购买 ECS

登录 [阿里云 ECS 控制台](https://ecs.console.aliyun.com/) → 创建实例：

| 配置项 | 建议 |
|--------|------|
| 地域 | 离用户近的（如华东） |
| 镜像 | **Ubuntu 22.04** 或 Alibaba Cloud Linux 3 |
| 规格 | **2 核 4G**（演示够用） |
| 系统盘 | 40GB+ |
| 带宽 | 3–5 Mbps 或按量 |
| 登录 | 密钥对（推荐）或密码 |

创建后记下 **公网 IP**。

### 2.2 安全组

入方向放行：

| 端口 | 用途 |
|------|------|
| 22 | SSH 登录 |
| 80 | HTTP（证书申请 + 跳转 HTTPS） |
| 443 | HTTPS |

**不要**对 `0.0.0.0/0` 开放 3306、6379（数据库只在 Docker 内网访问）。

### 2.3 域名解析

在阿里云 **云解析 DNS** 为两个域名添加记录：

| 主机记录 | 记录类型 | 记录值 |
|----------|----------|--------|
| `@` | A | ECS 公网 IP |
| `www` | A | ECS 公网 IP |

两个域名各配一条（`jj520.xyz`、`mgongchang.xyz`）。

解析生效通常 5–30 分钟，可用 `ping jj520.xyz` 检查是否指向 ECS IP。

---

## 三、登录服务器并安装 Docker

```bash
# 本地 Mac/Linux 连接（把 IP 和密钥路径换成你的）
ssh -i ~/.ssh/your-key.pem root@你的ECS公网IP
```

在 ECS 上执行：

```bash
# Ubuntu 22.04
apt update && apt upgrade -y
apt install -y git curl

# 安装 Docker（官方脚本）
curl -fsSL https://get.docker.com | sh
systemctl enable docker
systemctl start docker

# 验证
docker --version
docker compose version
```

---

## 四、上传项目代码

任选一种方式：

### 方式 A：Git 克隆（**推荐，日常用 pull 更新**）

```bash
cd /opt
git clone https://github.com/zhangdingyi123/zhibo-master.git zhibo
cd zhibo
```

**后续更新代码**（在 ECS 上）：

```bash
cd /opt/zhibo
bash scripts/ecs-update.sh
# 等价于：git pull → migrate → redeploy
```

`git pull` 若报 `GnuTLS recv error`：

```bash
git -c http.version=HTTP/1.1 pull --ff-only
# 或改用 SSH：git remote set-url origin git@github.com:zhangdingyi123/zhibo-master.git
```

### 方式 B：本地上传（仅首次无 Git 时备用）

```bash
# 在本地项目根目录执行
scp -r -i ~/.ssh/your-key.pem . root@你的ECS公网IP:/opt/zhibo
```

---

## 五、配置生产环境变量

```bash
cd /opt/zhibo
cp .env.example .env
nano .env   # 或 vi .env
```

**生产 `.env` 示例**（密码务必改成强密码）：

```env
# MySQL（与 docker-compose.prod.yml 中 mysql 服务一致）
MYSQL_ROOT_PASSWORD=你的Root强密码
MYSQL_DATABASE=zhibo
MYSQL_USER=zhibo
MYSQL_PASSWORD=你的Zhibo强密码

# 后端连接串（主机名必须是 mysql，不是 localhost）
MYSQL_DSN=zhibo:你的Zhibo强密码@tcp(mysql:3306)/zhibo?charset=utf8mb4&parseTime=True&loc=Local

# Redis（主机名必须是 redis）
REDIS_ADDR=redis:6379
REDIS_PASSWORD=
REDIS_DB=0

# 双域名 CORS
FRONTEND_URL=https://jj520.xyz
FRONTEND_URLS=https://jj520.xyz,https://mgongchang.xyz

# JWT 密钥（随机长字符串）
JWT_SECRET=请换成至少32位随机字符串
```

生成随机 JWT 密钥：

```bash
openssl rand -hex 32
```

---

## 六、一键启动

```bash
cd /opt/zhibo
docker compose -f docker-compose.prod.yml up -d --build
```

首次构建约 5–15 分钟（拉镜像 + 编译 Go + 构建前端）。

### 查看状态

```bash
docker compose -f docker-compose.prod.yml ps
docker compose -f docker-compose.prod.yml logs -f backend
```

全部 `healthy` / `running` 即可。

### 验证（HTTP）

```bash
curl http://127.0.0.1/api/v1/health
# 期望：{"status":"ok"} 或类似健康响应
```

浏览器访问（域名解析生效后）：

| 地址 | 预期 |
|------|------|
| http://jj520.xyz | 跳转到用户端 `/app` |
| http://mgongchang.xyz | 跳转到主播端 `/admin` |
| http://jj520.xyz/api/v1/health | API 健康检查 |

### 监控可视化（Grafana）

一键启动（**推荐走 80 端口 `/monitor/`，免开 3000**）：

```bash
cd /opt/zhibo
bash scripts/observability-up.sh
```

| 地址 | 说明 |
|------|------|
| `http://<ECS公网IP>/monitor/` | Grafana（admin / zhibo） |
| `http://<ECS公网IP>/monitor/d/zhibo-auction` | 竞拍监控大盘 |
| `http://<ECS公网IP>/api/v1/metrics` | JSON 指标 |

详见 [observability.md](./observability.md)。

---

## 七、MySQL 与 Redis 说明

### 7.1 自动初始化

MySQL 容器**第一次**启动时，会按字母顺序执行：

```
backend/migrations/001_schema.sql
backend/migrations/002_seed.sql
backend/migrations/003_auth.sql
```

包含表结构 + 演示种子数据 + 登录字段。

### 7.2 手动检查

```bash
# MySQL
docker compose -f docker-compose.prod.yml exec mysql \
  mysql -uzhibo -p你的Zhibo强密码 zhibo -e "SHOW TABLES;"

# Redis
docker compose -f docker-compose.prod.yml exec redis redis-cli ping
# 期望：PONG
```

### 7.3 演示账号

密码均为 `123456`：

| 角色 | 手机号 |
|------|--------|
| 主播 | 13800000001 |
| 买家 | 13800000002 |

---

## 八、配置 HTTPS（必做）

生产环境务必上 HTTPS（WebSocket 在 HTTPS 下更稳定）。

### 方式 A：Certbot 免费证书（推荐）

**先临时停 nginx 容器**（certbot standalone 需占用 80 端口）：

```bash
cd /opt/zhibo
docker compose -f docker-compose.prod.yml stop nginx
```

安装 certbot 并申请证书：

```bash
apt install -y certbot
certbot certonly --standalone \
  -d jj520.xyz -d www.jj520.xyz \
  -d mgongchang.xyz -d www.mgongchang.xyz \
  --email 你的邮箱@example.com \
  --agree-tos --no-eff-email
```

证书路径一般为：

```
/etc/letsencrypt/live/jj520.xyz/fullchain.pem
/etc/letsencrypt/live/jj520.xyz/privkey.pem
```

将 `deploy/nginx.conf` 替换为 HTTPS 版本（见下方「附录 A」），然后修改 `docker-compose.prod.yml` 中 nginx 挂载：

```yaml
  nginx:
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./deploy/nginx-https.conf:/etc/nginx/conf.d/default.conf:ro
      - /etc/letsencrypt:/etc/letsencrypt:ro
```

重启：

```bash
docker compose -f docker-compose.prod.yml up -d nginx
```

证书自动续期（crontab）：

```bash
crontab -e
# 添加（每月 1 号凌晨续期并 reload nginx）
0 3 1 * * certbot renew --quiet && docker compose -f /opt/zhibo/docker-compose.prod.yml exec nginx nginx -s reload
```

### 方式 B：阿里云免费 SSL 证书

1. 阿里云控制台 → SSL 证书 → 免费申请  
2. 下载 Nginx 格式证书（`.pem` + `.key`）  
3. 上传到 ECS `/opt/zhibo/deploy/certs/`  
4. 在 nginx 配置中引用证书路径并挂载到容器

---

## 九、常用运维命令

```bash
cd /opt/zhibo

# 查看所有容器
docker compose -f docker-compose.prod.yml ps

# 查看日志
docker compose -f docker-compose.prod.yml logs -f backend
docker compose -f docker-compose.prod.yml logs -f nginx

# 重启单个服务
docker compose -f docker-compose.prod.yml restart backend

# 更新代码后重新部署（推荐一键脚本）
bash scripts/ecs-update.sh

# 或分步执行：
git pull
bash scripts/migrate.sh          # 已有数据卷时必须手动跑增量迁移
bash scripts/redeploy.sh

# 停止全部
docker compose -f docker-compose.prod.yml down

# 停止并删除数据卷（慎用！会清空数据库）
docker compose -f docker-compose.prod.yml down -v
```

### 增量数据库迁移

> MySQL 容器**只在首次创建数据卷**时执行 `backend/migrations/*.sql`。  
> 后续新增迁移（如 `005_order_pay_expire.sql`）需**手动执行**。

```bash
cd /opt/zhibo

# 自动检测并应用未执行的迁移（可重复运行）
bash scripts/migrate.sh
```

**手动验证迁移是否成功**：

```bash
# 1. 确认 pay_expire_at 字段存在
docker compose -f docker-compose.prod.yml exec mysql sh -c \
  'mysql -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DATABASE" -e "
    SHOW COLUMNS FROM orders LIKE '\''pay_expire_at'\'';
  "'

# 期望输出含：pay_expire_at | datetime(3) | YES | ...

# 2. 查看待支付订单是否已补截止时间
docker compose -f docker-compose.prod.yml exec mysql sh -c \
  'mysql -u"$MYSQL_USER" -p"$MYSQL_PASSWORD" "$MYSQL_DATABASE" -e "
    SELECT id, order_no, status, pay_expire_at FROM orders WHERE status='\''pending_pay'\'';
  "'

# 3. 确认 backend 已加载新代码（日志含 order expiry 或重启时间）
docker compose -f docker-compose.prod.yml logs --tail 20 backend
```

**API 验证**（登录买家后，浏览器 F12 → Network）：

- `GET /api/v1/orders` 响应中每项应含 `product` 对象，不再只有 `order` 裸字段
- 订单对象含 `payExpireAt` 字段（待支付时有值）

### MySQL 备份

```bash
docker compose -f docker-compose.prod.yml exec -T mysql \
  mysqldump -uroot -p"$MYSQL_ROOT_PASSWORD" zhibo \
  > backup_$(date +%F_%H%M%S).sql
```

---

## 十、故障排查

| 现象 | 排查 |
|------|------|
| 域名打不开 | DNS 是否生效；安全组 80/443 是否放行 |
| 502 Bad Gateway | `docker compose logs backend` 看后端是否启动；MySQL 是否 healthy |
| 前端白屏 | `docker compose logs frontend`；浏览器 F12 看网络请求 |
| API CORS 报错 | 检查 `.env` 中 `FRONTEND_URLS` 是否包含当前访问域名（含 `https://`） |
| WebSocket 连不上 | 确认 nginx 配置了 `Upgrade`/`Connection`；是否已上 HTTPS |
| 登录失败 | 确认 `003_auth.sql` 已执行；用演示账号 13800000001 / 123456 |
| MySQL 连不上 | `MYSQL_DSN` 主机必须是 `mysql`，不是 `localhost` |

---

## 附录 A：HTTPS 版 Nginx 配置模板

保存为 `deploy/nginx-https.conf`：

```nginx
upstream zhibo_backend {
    server backend:8081;
}

# HTTP → HTTPS
server {
    listen 80;
    server_name jj520.xyz www.jj520.xyz mgongchang.xyz www.mgongchang.xyz;
    return 301 https://$host$request_uri;
}

# 用户端
server {
    listen 443 ssl;
    server_name jj520.xyz www.jj520.xyz;

    ssl_certificate     /etc/letsencrypt/live/jj520.xyz/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/jj520.xyz/privkey.pem;

    location = / { return 302 /app; }

    location /api/ {
        proxy_pass http://zhibo_backend;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 3600s;
    }

    location / {
        proxy_pass http://frontend:80;
        proxy_set_header Host $host;
    }
}

# 主播端
server {
    listen 443 ssl;
    server_name mgongchang.xyz www.mgongchang.xyz;

    ssl_certificate     /etc/letsencrypt/live/jj520.xyz/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/jj520.xyz/privkey.pem;

    location = / { return 302 /admin; }

    location /api/ {
        proxy_pass http://zhibo_backend;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 3600s;
    }

    location / {
        proxy_pass http://frontend:80;
        proxy_set_header Host $host;
    }
}
```

> 若 certbot 把证书放在 `jj520.xyz` 目录下，两个域名可共用同一张多域名证书。

---

## 附录 B：费用粗估（按量）

| 项目 | 参考 |
|------|------|
| ECS 2C4G | 约 50–100 元/月（活动价更低） |
| 域名 | 已有 |
| 带宽 | 按固定带宽或流量计费 |
| SSL | Let's Encrypt 免费 |

---

## 附录 C：后续升级（可选）

演示跑通后如需更稳：

1. **MySQL** → 阿里云 RDS（改 `MYSQL_DSN` 为 RDS 内网地址，compose 里去掉 mysql 服务）  
2. **Redis** → 阿里云云数据库 Redis（改 `REDIS_ADDR`，compose 里去掉 redis 服务）  
3. **镜像仓库** → 阿里云 ACR 存构建好的镜像，ECS 只拉镜像不编译

详见 [mysql-redis.md](./mysql-redis.md)。
