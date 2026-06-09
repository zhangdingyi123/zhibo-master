# 监控可视化（完整方案）

## 架构

```
用户浏览器
    │
    ├─ http://<IP>/monitor/          → Nginx → Grafana（竞拍大盘）
    ├─ http://<IP>/api/v1/metrics    → Backend JSON 指标（调试）
    └─ http://<IP>/metrics           → Backend Prometheus 文本

Prometheus（容器内）每 15s 抓取 zhibo-backend:8081/metrics
Grafana 自动加载数据源 + 「Zhibo 竞拍监控」大盘
```

**推荐访问 `/monitor/`**：走 80 端口，与业务站相同，**无需**在安全组额外开 3000/9090。

---

## 一键部署（ECS）

```bash
cd /opt/zhibo
git pull                                    # 确保有最新 deploy/grafana/
bash scripts/observability-up.sh
```

浏览器打开：

| 地址 | 说明 |
|------|------|
| `http://47.97.176.185/monitor/` | Grafana 登录 |
| `http://47.97.176.185/monitor/d/zhibo-auction` | 竞拍监控大盘 |
| `http://47.97.176.185/api/v1/metrics` | JSON 指标 |

登录：`admin` / `zhibo`

---

## 手动部署（docker-compose 1.29 会报 ContainerConfig）

ECS 上 **不要用 `docker-compose up` 启监控**，直接用脚本（内部 `docker run`）：

```bash
cd /opt/zhibo
bash scripts/observability-up.sh
```

`.env` 中可设置 Grafana 子路径（与公网 IP 一致）：

```env
GRAFANA_ROOT_URL=http://47.97.176.185/monitor/
```

修改后重新运行 `bash scripts/observability-up.sh`。

---

## 大盘内容

| 面板 | 指标 |
|------|------|
| WebSocket 连接 / 活跃房间 | `zhibo_ws_connections` / `zhibo_ws_rooms` |
| 出价失败率 | `zhibo_bid_failure_rate` |
| 缓存写失败率 | `zhibo_cache_write_failure_rate` |
| 出价吞吐 | `rate(zhibo_bid_*_total[1m])` |
| 快照缓存命中 | hits / misses / 命中率 |
| 缓存写穿 | OnBid / Refresh 成功失败重试 |

配置文件：

- 大盘 JSON：`deploy/grafana/dashboards/zhibo.json`
- 数据源：`deploy/grafana/provisioning/datasources/prometheus.yml`
- 告警规则：`deploy/prometheus/alerts.yml`

---

## 产生数据（演示）

1. 浏览器进直播间，WebSocket 连接后 WS 连接数 > 0
2. 出几次价 → 出价吞吐曲线上升
3. 或压测：

```bash
bash backend/scripts/bid_stress.sh http://127.0.0.1 <场次ID>
```

---

## 故障排查

### Dashboard not found

```bash
# 确认大盘文件存在且已挂载
ls deploy/grafana/dashboards/zhibo.json
docker exec zhibo-grafana ls -la /var/lib/grafana/dashboards/
docker restart zhibo-grafana
```

仍无大盘 → Grafana UI：**Dashboards → Import**，粘贴 `deploy/grafana/dashboards/zhibo.json`。

### Prometheus 无数据

```bash
curl -s http://127.0.0.1:9090/api/v1/targets | python3 -m json.tool
# zhibo-api 的 health 应为 "up"

curl -s http://127.0.0.1/metrics | grep zhibo_bid
```

### /monitor/ 502

```bash
docker ps | grep grafana
docker-compose -f docker-compose.prod.yml up -d --no-deps grafana nginx
```

### docker-compose 报 ContainerConfig

**不要用 `docker-compose up` 重建容器**。改用：

```bash
bash scripts/observability-up.sh
```

脚本使用 `docker run`，绕过 compose 1.29 与新版 Docker 的不兼容。

---

## 代码索引

| 模块 | 路径 |
|------|------|
| 指标采集 | `backend/internal/infra/metrics/` |
| JSON 接口 | `GET /api/v1/metrics` |
| Prometheus 文本 | `GET /metrics` |
| Compose | `docker-compose.prod.yml` |
| Nginx 反代 | `deploy/nginx.conf` → `/monitor/` |
| 启动脚本 | `scripts/observability-up.sh` |
