# 域名备案（ICP）— 双域名上线必读

> 在**大陆阿里云 ECS** 上，通过 `jj520.xyz`、`mgongchang.xyz` 访问网站，**必须**完成工信部 ICP 备案。  
> 未备案时访问域名会显示阿里云「域名暂时无法访问」拦截页，**与项目代码、nginx 配置无关**，无法通过改代码去掉。

备案通过后，本项目已配置：

| 域名 | 入口 |
|------|------|
| `jj520.xyz` | 用户端 `/app` |
| `mgongchang.xyz` | 主播端 `/admin` |

---

## 一、备案前准备

| 材料 | 说明 |
|------|------|
| 域名 | `jj520.xyz`、`mgongchang.xyz` 已在阿里云或其他注册商 |
| ECS | 大陆节点，公网 IP `47.97.176.185` |
| 主体 | 个人或企业（个人备案较常见） |
| 身份证 | 个人备案需本人证件 + 人脸识别 |
| 手机号 | 备案验证用 |

**注意**：两个域名可以备在同一主体下；若 `jj520.xyz` 与 `mgongchang.xyz` 在不同注册商，都需解析到同一 ECS IP，并在备案时如实填写网站信息。

---

## 二、GoDaddy 域名转入阿里云（方案 A）

> 报错「注册商 GoDaddy.com, LLC 未获得工信部批复」→ 须先把域名转入阿里云，再提交备案。  
> `mgongchang.xyz` 若在 GoDaddy，同样操作一遍。

### 时间线

| 阶段 | 耗时 |
|------|------|
| GoDaddy 解锁 + 获取转移码 | 10 分钟 |
| 阿里云提交转入 | 10 分钟 |
| 转移生效 | 3–7 天 |
| ICP 备案审核 | 7–20 天 |
| **合计** | **约 2–4 周** |

### 第一步：GoDaddy 准备转出

1. 登录 [GoDaddy 域名控制台](https://account.godaddy.com/products)
2. 点进 **`jj520.xyz`**
3. **解锁域名**（Domain Lock → 关闭 / 解锁）
4. 获取 **转移授权码（Authorization Code / EPP Code）**
   - 一般在：域名设置 → **转出域名** / **Transfer to Another Registrar**
   - 授权码会发到注册邮箱
5. **关闭隐私保护**（WHOIS Privacy），否则可能转入失败
6. 确认域名注册 **未满 60 天**、**未在 15 天内转移过**（ICANN 规则）

你截图里的「转发」**不用设置**；转入期间 DNS 可保持原样，转入成功后再改解析。

### 第二步：阿里云发起转入

1. 登录 [阿里云域名控制台](https://dc.console.aliyun.com/)
2. 左侧 **域名转入** → 输入 `jj520.xyz`
3. 填写 GoDaddy 发来的 **转移密码（授权码）**
4. 支付转入费用（`.xyz` 通常含 1 年续费，以页面为准）
5. 提交后等待

### 第三步：GoDaddy 确认转移

1. 查收 GoDaddy 注册邮箱
2. 邮件标题类似 **Approve Transfer** / **确认转移**
3. **点链接批准转出**（不点可能 5 天后自动拒绝）

### 第四步：转入成功后配置 DNS

阿里云 → 域名 → `jj520.xyz` → **解析设置**：

| 记录类型 | 主机记录 | 记录值 |
|----------|----------|--------|
| A | `@` | `47.97.176.185` |
| A | `www` | `47.97.176.185` |

对 `mgongchang.xyz` 重复以上全部步骤。

### 第五步：再提交 ICP 备案

转入完成（阿里云域名列表显示 **正常**）后：

1. 打开 [阿里云备案](https://beian.aliyun.com/)
2. 重新提交，`jj520.xyz` 注册商应显示为 **阿里云**
3. 不再出现 GoDaddy 未批复报错

### 转入期间演示

域名仍可能被拦截，用 IP 演示：

- `http://47.97.176.185/app`
- `http://47.97.176.185/admin`

---

## 三、阿里云备案流程（约 3–20 个工作日）

### 1. 进入备案系统

1. 登录 [阿里云 ICP 备案](https://beian.aliyun.com/)
2. 点击 **开始备案** → 按向导填写

### 2. 填写网站信息（示例）

| 字段 | 建议填写 |
|------|----------|
| 网站名称 | 直播竞拍演示系统（勿含「博客」「论坛」等敏感词） |
| 域名 | `jj520.xyz`（首个站）；第二个域名可新增网站或一并添加 |
| 服务器 IP | `47.97.176.185` |
| 网站内容 | 其他 / 企业展示 / 综合（按实际选，与「电商演示」接近即可） |
| 前置审批 | 一般无需 |

### 3. 接入阿里云（关键）

域名在阿里云注册：备案审核通过后自动接入。  
域名在其他注册商：备案时需做 **接入备案**，将备案信息指向阿里云服务器。

### 4. 管局审核

- 提交后阿里云初审（1–2 天）
- 各省管局审核（约 7–20 天）
- 短信核验（工信部短信，24 小时内完成）

### 5. 备案成功标志

- 备案号下发（如 `京ICP备xxxxxxxx号`）
- 访问 `jj520.xyz` **不再**出现阿里云拦截页

---

## 四、备案通过后 — 服务器配置

### 1. `.env` CORS

```env
FRONTEND_URL=http://jj520.xyz
FRONTEND_URLS=http://jj520.xyz,http://mgongchang.xyz,https://jj520.xyz,https://mgongchang.xyz
```

### 2. 确认 DNS 解析

| 主机记录 | 类型 | 记录值 |
|----------|------|--------|
| `@` | A | `47.97.176.185` |
| `www` | A | `47.97.176.185` |

两个域名各配一套。

### 3. 重建 nginx（双域名配置已在 `deploy/nginx.conf`）

```bash
cd /opt/zhibo
git pull   # 或确保 nginx.conf 含 jj520.xyz / mgongchang.xyz

NET=$(docker inspect zhibo-backend --format '{{range $k,$v := .NetworkSettings.Networks}}{{$k}}{{end}}')

docker rm -f zhibo-nginx
docker run -d --name zhibo-nginx \
  --network "$NET" \
  -p 80:80 \
  -v /opt/zhibo/deploy/nginx.conf:/etc/nginx/conf.d/default.conf:ro \
  --restart unless-stopped \
  nginx:1.27-alpine

docker restart zhibo-backend
```

### 4. 验证

```bash
curl -s -H "Host: jj520.xyz" http://127.0.0.1/api/v1/health
curl -s -H "Host: mgongchang.xyz" http://127.0.0.1/api/v1/health
```

浏览器：

- http://jj520.xyz/app  
- http://mgongchang.xyz/admin  

### 5. HTTPS（可选，备案后）

见 [deploy-aliyun.md](./deploy-aliyun.md) 第八节 Certbot。

---

## 五、备案审核期间怎么演示？

| 方式 | 地址 | 是否需备案 |
|------|------|:----------:|
| 公网 IP | `http://47.97.176.185/app` | 否 |
| 域名 | `http://jj520.xyz/app` | **是** |

审核期间答辩可用 IP；正式上线答辩用域名。

---

## 六、常见问题

| 问题 | 处理 |
|------|------|
| 仍显示备案拦截页 | 备案未通过或未接入阿里云；在备案平台查状态 |
| 备案过了仍拦截 | 域名解析是否指向 `47.97.176.185`；是否用了香港 ECS |
| API CORS 报错 | `.env` 的 `FRONTEND_URLS` 须含当前访问的 `http(s)://域名` |
| nginx Restarting | 确认 `zhibo-backend` 容器名与 nginx 配置一致 |
| GoDaddy 未批复无法备案 | 先按第二节转入阿里云 |
| 转移一直 pending | 检查 GoDaddy 是否批准邮件；域名是否已解锁 |

---

## 七、不想备案的替代方案

| 方案 | 说明 |
|------|------|
| **香港/海外 ECS** | 域名无需大陆备案，但国内访问可能较慢 |
| **仅 IP 演示** | 当前 ECS 立即可用，不适合长期使用域名 |
| **已备案域名** | 换用自己或公司已有备案域名，改 DNS + nginx `server_name` |
