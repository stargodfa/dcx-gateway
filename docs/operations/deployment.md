# 部署说明

本文档说明 Go 数据网关从本地验证、Linux ARM64 构建、打包、上传 Firefly 开发板、systemd 启动，到 OpenAPI 文档服务部署的完整流程。

公开版本只分发虚构演示种子和配置模板。systemd 使用通用 `dcx-gateway` 账号及 `/opt/dcx-gateway`、`/opt/dcx-gateway-docs` 目录；部署前请自行创建账号、目录并设置 SSH 上传与服务运行所需权限，或修改模板。公开文件范围与本地配置规则见 [公开版本清理说明](public-release.md)。

## 1. 环境依赖

- Go: 以 `go.mod` 中版本为准。
- MySQL: 建议 MySQL 8.0+。
- Redis: 建议与网关同网段可访问。
- Linux ARM64 / Firefly 开发板。
- 网关默认端口: `8080`。
- OpenAPI 文档服务默认端口: `8090`。

当前 migrations 使用 `utf8mb4_0900_ai_ci`，这是 MySQL 8.0 collation。若使用 MariaDB，需要统一评估并调整所有 migrations 的 collation，例如 `utf8mb4_general_ci` 或 `utf8mb4_unicode_ci`，不要只改单个 migration。

## 2. 配置文件

仓库只提交脱敏模板：

```text
configs/config.example.yaml
```

真实配置文件不要提交。当前 `cmd/dcx-gateway/main.go` 固定读取：

```text
configs/config.dev.yaml
```

因此部署包启动前需要在目标目录创建：

```bash
cp configs/config.example.yaml configs/config.dev.yaml
vi configs/config.dev.yaml
```

请按现场环境修改：

- `mysql.dsn`
- `redis.addr`
- `redis.password`
- `redis.key_prefix`
- `fastapi.base_url`
- `fastapi.ws_data_url`
- `security.static_token`
- `server.max_body_bytes`
- `app.port`

### Redis 项目隔离

`redis.key_prefix` 统一作用于 Session、DataBundle、当前脚本、模块、视频状态以及图表运行时的 binding/chart 两类索引。配置为 `dcx-gateway` 或 `dcx-gateway:` 都会使用 `dcx-gateway:` 前缀；不同实例共用同一 Redis DB 时应配置不同的非空前缀。

升级后，原先未加前缀的业务缓存会按新前缀重新生成，不自动回读或删除旧键。已有 Session 的键格式保持一致。前缀为空仍兼容无前缀键，但不提供项目隔离。

真实 Redis 的隔离回归测试：

```bash
DCX_GATEWAY_TEST_REDIS_ADDR=127.0.0.1:6379 go test ./internal/bootstrap -run TestCachesRedisNamespaceIsolation -count=1
```

只对可丢弃的测试 Redis 执行该命令。测试只清理自己生成的唯一前缀，不执行 `FLUSHDB`。

不要提交 `configs/config.yaml` 或 `configs/config.dev.yaml`。

## 3. 本地验证

部署融合后的补充配置接口前，须在当前 `mysql.dsn` 对应数据库执行 `migrations/014_init_ue_aux_configs.sql`，创建锚点、设备、集群目录和授权注册表四张表。应用沿用手动 migration 方式。存量数据迁移及 Server 地址、鉴权切换见 [补充配置融合说明](../api/aux-config.md)。

```bash
gofmt -w .
go test ./...
go vet ./...
go build ./...
```

本地运行：

```bash
go run ./cmd/dcx-gateway
```

基础验证：

```bash
curl http://127.0.0.1:8080/api/health
curl "http://127.0.0.1:8080/api/health?deep=true"
curl http://127.0.0.1:8080/api/ue/config/bootstrap
```

## 4. ARM64 构建

使用脚本：

```bash
bash scripts/build_arm64.sh
```

脚本会执行：

```bash
GOOS=linux GOARCH=arm64 go build -o dist/dcx-gateway-arm64/dcx-gateway ./cmd/dcx-gateway
```

构建产物：

```text
dist/dcx-gateway-arm64/dcx-gateway
```

## 5. 打包

```bash
bash scripts/package_arm64.sh
```

打包产物：

```text
dist/dcx-gateway-arm64.tar.gz
```

包内包含：

```text
dcx-gateway-arm64/
  dcx-gateway
  readme.md
  configs/config.example.yaml
  migrations/
  docs/README.md
  docs/architecture/
  docs/api/openapi.yaml
  docs/api/swagger-ui/
  docs/operations/
  docs/testing/
  docs/archive/
  systemd/dcx-gateway.service
```

包内保留完整 `docs/` 分类与根 README，跨目录文档链接可继续使用。文档中的源码路径和开发检查命令以源码工程为执行环境。

包内不包含：

```text
configs/config.yaml
configs/config.dev.yaml
.git/
dist/
*.zip
```

## 6. 上传 Firefly 开发板

不要在脚本里写死 IP。使用参数传入远端目标：

```bash
bash scripts/deploy_firefly.sh deploy@<host>
```

脚本会上传：

```text
dist/dcx-gateway-arm64.tar.gz
```

并解压到：

```text
/opt/dcx-gateway
```

最终目录：

```text
/opt/dcx-gateway/dcx-gateway-arm64
```

如果远端没有 `configs/config.dev.yaml`，脚本只提示，不自动覆盖配置。首次部署时请登录开发板：

```bash
cd /opt/dcx-gateway/dcx-gateway-arm64
cp configs/config.example.yaml configs/config.dev.yaml
vi configs/config.dev.yaml
```

## 7. systemd 安装网关服务

在开发板上执行：

```bash
cd /opt/dcx-gateway/dcx-gateway-arm64
sudo cp systemd/dcx-gateway.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable dcx-gateway
sudo systemctl start dcx-gateway
```

查看状态和日志：

```bash
sudo systemctl status dcx-gateway
journalctl -u dcx-gateway -f
```

服务文件：

```text
systemd/dcx-gateway.service
```

注意：当前二进制没有 `-config` 参数，systemd 直接启动二进制，并依赖 `WorkingDirectory` 下的 `configs/config.dev.yaml`。

## 8. 部署 OpenAPI 文档服务

本地临时启动：

```bash
bash scripts/serve_openapi_docs.sh
```

默认端口 `8090`，可覆盖：

```bash
DOCS_PORT=8091 bash scripts/serve_openapi_docs.sh
```

本地访问：

```text
http://127.0.0.1:8090/openapi.yaml
http://127.0.0.1:8090/swagger-ui/
```

Swagger UI 已离线化，`docs/api/swagger-ui/` 内包含本地 `swagger-ui.css`、`swagger-ui-bundle.js`、`swagger-ui-standalone-preset.js`。访问 `http://<ip>:8090/swagger-ui/` 不需要公网；`http://<ip>:8090/openapi.yaml` 仍可直接访问。

部署到 Firefly：

```bash
bash scripts/deploy_openapi_docs.sh deploy@<host>
```

脚本会上传：

```text
docs/api/openapi.yaml
docs/api/swagger-ui/
scripts/serve_openapi_docs.sh
systemd/dcx-gateway-interface.service
```

远端目录：

```text
/opt/dcx-gateway-docs
```

在开发板上安装 docs service：

```bash
sudo cp /opt/dcx-gateway-docs/systemd/dcx-gateway-interface.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable dcx-gateway-interface
sudo systemctl start dcx-gateway-interface
sudo systemctl status dcx-gateway-interface
journalctl -u dcx-gateway-interface -f
```

访问：

```text
http://<ip>:8090/openapi.yaml
http://<ip>:8090/swagger-ui/
```

## 9. 部署后验证

在开发板本机：

```bash
curl http://127.0.0.1:8080/api/health
curl "http://127.0.0.1:8080/api/health?deep=true"
curl http://127.0.0.1:8080/api/ue/config/bootstrap
```

在同网段机器：

```bash
curl http://<ip>:8080/api/health
curl "http://<ip>:8080/api/health?deep=true"
curl http://<ip>:8090/openapi.yaml
```

Swagger UI：

```text
http://<ip>:8090/swagger-ui/
```

## 10. Auth 与联调 token

公开版本不提供默认登录账号、密码或密码 hash；`012_seed_auth_users.sql` 保留编号但不创建用户。使用 Bearer 登录前，需要自行生成 bcrypt 密码 hash 并创建 `ue_users` 用户。仅做本地写接口演示时，也可以在私有配置中设置随机静态 token，并启用 `security.static_token_enabled`。

登录：

```bash
curl -X POST "http://<gw>:8080/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"<your-username>","password":"<your-password>"}'
```

Bearer token 调写接口：

```bash
curl -X PUT "http://<gw>:8080/api/ue/config/model-config/placements/demo_model_1" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"visible":false}'
```

静态 token 兜底：

```bash
curl -X PUT "http://<gw>:8080/api/ue/config/model-config/placements/demo_model_1" \
  -H "Content-Type: application/json" \
  -H "X-Gateway-Token: <security.static_token>" \
  -d '{"visible":false}'
```

Redis session 查询失败时，鉴权 middleware 返回 `503 {"error":"auth_unavailable"}`；token 缺失或错误时返回 `401 {"error":"unauthorized"}`。

## 11. P1 稳定性配置

### health?deep=true

轻量检查：

```bash
curl http://<gw>:8080/api/health
```

深度检查：

```bash
curl "http://<gw>:8080/api/health?deep=true"
```

检查项：

- MySQL `PingContext`
- Redis `PING`
- `fastapi.ws_data_url` 是否已配置

MySQL 或 Redis 异常时返回 HTTP 503，响应 `status=degraded`。对外响应只返回 `ok/error/degraded` 等通用状态，不暴露数据库或 Redis 原始错误；原始错误只写服务端日志。FastAPI WS 只检查配置，不主动连接。

### raw preset PUT body limit

只限制 raw preset PUT：

- `PUT /api/ue/config/path-config`
- `PUT /api/ue/config/chart-config`
- `PUT /api/ue/config/chart-binding`
- `PUT /api/ue/config/drone-config`
- `PUT /api/ue/config/render-config`
- `PUT /api/ue/config/video-streams`
- `PUT /api/ue/config/report-scripts`

默认限制 8MB：

```yaml
server:
  max_body_bytes: 8388608
```

超限返回 HTTP 413：

```json
{"error":"request body too large"}
```

该限制不影响 placement PUT、model-table、model-types、init-config、bootstrap 或 preload。

### X-Request-Id / X-Trace-Id

- 如果请求带 `X-Request-Id`，响应头返回该值。
- 如果没有 `X-Request-Id` 但带 `X-Trace-Id`，使用 `X-Trace-Id` 作为 request_id。
- 如果两者都没有，网关生成 request_id。
- `X-Request-Id` / `X-Trace-Id` 最多保留 128 字符，超出会截断。
- 请求日志包含 `request_id/method/path/status/latency/client_ip`。

不要在日志中打印 password、access_token、Authorization、X-Gateway-Token 或 raw preset body。

## 12. 常见问题

### CRLF 导致 shell 脚本报错

如果 Linux 上出现 `$'\r': command not found`：

```bash
sed -i 's/\r$//' scripts/*.sh
chmod +x scripts/*.sh
```

### 8080 / 8090 端口占用

```bash
sudo lsof -i :8080
sudo lsof -i :8090
```

修改 `configs/config.dev.yaml` 中 `app.port` 或调整 docs service 的 `DOCS_PORT`。

### MySQL collation 不兼容

`utf8mb4_0900_ai_ci` 需要 MySQL 8.0+。MariaDB 不支持时，需要统一调整所有 migrations 的 collation。

### config 路径错误

当前程序固定读取 `configs/config.dev.yaml`。确认：

```bash
ls /opt/dcx-gateway/dcx-gateway-arm64/configs/config.dev.yaml
```

### 权限不足

确认二进制和脚本可执行：

```bash
chmod +x /opt/dcx-gateway/dcx-gateway-arm64/dcx-gateway
chmod +x /opt/dcx-gateway-docs/scripts/serve_openapi_docs.sh
```

### systemd WorkingDirectory 错误

检查 service 文件中的路径是否存在：

```bash
systemctl cat dcx-gateway
systemctl cat dcx-gateway-interface
```

### Redis 连接失败

```bash
redis-cli -h <redis-host> -p 6379 ping
redis-cli -h <redis-host> -p 6379 -a '<password>' ping
```

### OpenAPI YAML 解析失败

优先使用 PyYAML：

```bash
python3 - <<'PY'
import yaml
with open('docs/api/openapi.yaml', 'r', encoding='utf-8') as f:
    yaml.safe_load(f)
print('openapi yaml ok')
PY
```

如果没有 PyYAML，可使用 Redocly：

```bash
npx --yes @redocly/cli lint docs/api/openapi.yaml
```

## Phase 1 样板脚本验证

部署完成后可验证三条第一阶段 Mock 样板脚本：

```bash
curl http://127.0.0.1:8080/api/scripts/security_main/data-bundle
curl http://127.0.0.1:8080/api/scripts/energy_main/data-bundle
curl http://127.0.0.1:8080/api/scripts/education_main/data-bundle
```

`security_main` 是安防汇报脚本，`energy_main` 是后勤能耗汇报脚本，`education_main` 是教学汇报脚本。当前三条脚本均使用 Mock 数据，后续可替换为真实 Adapter。

注意：`migrations/013_seed_phase1_scripts.sql` 负责写入 `energy_main` / `education_main` 及对应 phase1 chart 配置。重复执行该 seed 会将 `chart_energy_total`、`chart_energy_building`、`chart_education_usage`、`chart_education_schedule` 恢复为样板默认配置；如现场已手工调整这些 chart，重跑前需要备份。

## Scheduler 预热配置

示例配置：

```yaml
scheduler:
  preload:
    enabled: true
    interval_seconds: 60
    scripts:
      - security_main
      - energy_main
      - education_main
```

`interval_seconds <= 0` 时默认使用 60 秒；`scripts` 为空时预热任务会跳过并写 warn 日志。Scheduler 不注册新脚本，不直接访问 Redis/repository，也不重新组装 DataBundle。

注意：Scheduler 复用 `ScriptService.PreloadWithOptions(ctx, scriptID, Force=true, PushWS=push_ws)`。`push_ws=false` 时只刷新 Redis preload cache；只有显式配置 `push_ws=true` 时才会沿用 Preload 推送 FastAPI `/ws_data`。

验证日志：

```bash
journalctl -u dcx-gateway -f | grep "preload job"
```

## Adapter mock/http 切换

默认三类业务 Adapter 都使用 `mock`，不依赖真实外部系统。准备接入真实 HTTP 服务时，可按环境修改：

```yaml
adapters:
  security:
    mode: http
    base_url: "https://security.example.com"
    timeout_seconds: 3
    access_summary_path: "/api/security/access-summary"
    fire_alerts_path: "/api/security/fire-alerts"
  energy:
    mode: http
    base_url: "https://energy.example.com"
    timeout_seconds: 3
    snapshot_path: "/api/energy/snapshot"
  education:
    mode: http
    base_url: "https://education.example.com"
    timeout_seconds: 3
    classroom_usage_path: "/api/education/classroom-usage"
```

`mode` 支持 `mock` / `http`，缺失时默认 `mock`；`timeout_seconds <= 0` 时默认 3 秒。当前阶段如果 `mode=http` 但 `base_url` 为空，网关会 warn 并 fallback 到 mock，避免现场尚无真实系统时启动失败。

外部接口只作为 DataBundle 输入源，HTTP Adapter 会把外部 JSON 转换为 `domain` 对象，不会把原始外部 JSON 直接返回给 service，也不会写 DB/Redis。真实系统接入前需确认接口路径、字段名、单位和超时策略。

外部响应中的外层数组字段需要明确存在。若 `alerts` / `buildings` / `classrooms` 字段缺失，网关视为上游协议错误并返回 adapter error；若字段存在但为显式空数组 `[]`，则视为合法的“当前无数据”，返回空列表/空快照，DataBundle 生成时不会因此 panic。

## V20 /ws_data 稳定推送配置

`fastapi.ws_data_url` 配置后，Gateway 会复用一条到 FastAPI `/ws_data` 的 WebSocket 长连接，带 read loop、heartbeat ping、失败断连和下次推送自动重连。运行状态可查看：

```bash
curl http://127.0.0.1:8080/api/runtime/ws-data/status
```

如果 `security.protect_read=true`，该 GET 也需要 `Authorization: Bearer <token>` 或 `X-Gateway-Token`。

FastAPI 服务对服务鉴权使用 `fastapi.ws_auth`，不是用户登录 token：

```yaml
fastapi:
  ws_auth:
    type: none   # none / bearer / api_key / header
    header_name: ""
    token: ""
```

外部 Adapter 服务凭证同理，配置在 `adapters.*.auth` 下；不要把真实 token 写入 example 或提交仓库。

Scheduler 默认只预热缓存，不自动推 UE：

```yaml
scheduler:
  preload:
    enabled: true
    interval_seconds: 60
    push_ws: false
    scripts:
      - security_main
      - energy_main
      - education_main
```

只有显式设置 `scheduler.preload.push_ws=true`，或开启开发联调任务 `mock_push.enabled=true`，后台任务才会以 `auth_type=system` 推送 FastAPI `/ws_data`。`mock_push` 默认关闭：

```yaml
mock_push:
  enabled: false
  interval_seconds: 5
  scripts:
    - security_main
    - energy_main
    - education_main
```

当前推送是 best-effort + stats，不是 ACK/retry/queue/dead-letter 可靠消息系统；完整可靠投递需要 FastAPI 侧配合协议升级。
## UE Report Scene Warmup

V21 增加运行时 warmup 连接能力，不需要新增数据库表或 migration。

- `GET /api/ue/runtime/warmup-options` 返回可编辑资源目录。
- `POST /api/ue/runtime/report-scenes/preload` 根据 `report_scripts` 中指定 scene 的 `warmup.data` 调用现有 DataBundle preload。

命名边界：

- DataBundle 使用 `data_script_id` / `data_scene_id`，历史字段 `script_id` / `scene_id` 继续保留。
- UE 汇报编排使用 report script / chapter / scene ID。
- `warmup.data` 是 `scripts.script_id`。
- `warmup.charts` 是 `data_bindings.binding_id`，不是 `chart_id`。
- `warmup.videos` 是 `video_sources.video_id`。

`PUT /api/ue/config/report-scripts/scenes/warmup` 保存时只做数组归一化，不强校验资源 ID 是否已经存在。运行 `POST /api/ue/runtime/report-scenes/preload` 时，Gateway 会按 `warmup.data` 逐项执行 preload；不存在或失败的 `data_script_id` 会体现在 `results[].success=false` 和 `results[].error` 中。

第一版 `report-scenes/preload` 不按 charts/videos 过滤 WebSocket 消息；它会完整预热 `warmup.data` 指向的 data script，并把 requested charts/videos 写入响应供联调确认。
