# UE 配置接口

本文档面向 UE 工程师、Server/FastAPI 工程师和现场联调人员，描述 dcx-gateway 的 UE 配置接口行为。

## 响应风格

UE Config API 的成功响应直接返回 JSON 本体，不包 `code/data`。

示例：

```json
{
  "version": 1,
  "entries": []
}
```

错误响应使用当前网关统一错误格式，通常包含 `error` 字段。

## 补充配置接口

`model-anchors`、`sensor-devices`、`cluster-agents-catalog`、`cluster-agent-registry` 已融合到当前网关，共 16 个读写接口，沿用当前鉴权、请求追踪与 JSON 响应格式。接口、事务、版本及存量数据迁移规则见 [补充配置融合说明](aux-config.md)，完整结构见 [OpenAPI](openapi.yaml)。

## 核心边界

以下既有实例示例统一使用演示种子中的 `demo_model_1`。批量语法中的 `demo_model_2` 仅用于说明多个 ID 的请求格式，不是预置实例；操作模型实例前需要先通过 model-table 创建它。通用几何示例不代表演示模型资产参数。

- Go 数据网关不直接控制 UE。
- FastAPI/Server 仍是 UE 控制入口。
- `scene-desired` 是 generated 视图，不开放写接口。
- `init_config.instance_ids` 是启动加载白名单，不校验 `scene_desired`、`ue_scene_desired_models` 或 `ue_model_instances`。
- `model-types` 管 `model_id` 资产。
- `model-table` 管 `instance_id` 目录。
- `model-config placements` 管 `visible / position / rotation / scale`。
- `bounds_size` 输出时按 `base_bounds_size * abs(scale)` 计算，不在 placement 表长期保存。

## init_config

### GET /api/ue/config/init-config

读取 `ue_config_documents.config_key=init_config` 的完整 JSON。

### PUT /api/ue/config/init-config

只允许修改：

- `camera`
- `camera_id`
- `position`
- `rotation`
- `auto_load_on_startup`

不允许修改：

- `instance_id`
- `instance_ids`
- `instance_ids_add`
- `instance_ids_delete`

如果请求中出现 instance 相关字段，返回 400，并提示使用 instance_ids create/delete 接口。

### POST /api/ue/config/init-config/instance_ids/create

把实例加入启动白名单。

支持单个：

```json
{"instance_id":"demo_model_1"}
```

支持批量：

```json
{"instance_ids":["demo_model_1","demo_model_2"]}
```

规则：

- trim 空格。
- 空字符串返回 400。
- 请求内去重。
- 已存在 ID 不重复追加。
- 新增项按请求顺序追加。
- 成功返回完整 init_config JSON。

### DELETE /api/ue/config/init-config/instance_ids/delete

从启动白名单移除实例，不删除 instance，不删除 placement。

支持单个：

```json
{"instance_id":"demo_model_1"}
```

支持批量：

```json
{"instance_ids":["demo_model_1","demo_model_2"]}
```

不存在的 ID 会被忽略，重复删除仍成功。

## model-types

`model-types` 是 `model_id` 资产权威，数据源是 `ue_model_types`。

### GET /api/ue/config/model-types

返回 enabled model types。

### POST /api/ue/config/model-types/create

创建 model type。已存在且内容相同则幂等成功；已存在但内容不同返回 409，提示使用 PUT。

### PUT /api/ue/config/model-types

部分更新已存在 model type。允许字段：

- `uri`
- `format`
- `source_path`
- `base_bounds_size`
- `bounds_negative_extent_axis`
- `enabled`

不存在的 `model_id` 返回 404。未传字段保持 DB 原值。

### DELETE /api/ue/config/model-types/delete

软删除 model type，即设置 `ue_model_types.enabled=false`。

如果仍有 enabled instance 引用该 `model_id`，返回 409。不会自动删除 instances 或 placements。

## model-table

`model-table` 是 `instance_id` 目录权威，数据源是 `ue_model_instances`。

### GET /api/ue/config/model-table

默认返回 generated model table，按 `sort_order, instance_id` 排序。

### POST /api/ue/config/model-table/entries/create

创建实例目录项，同时为该 instance 创建默认 placement。

默认 placement：

```text
visible=true
position=0,0,0
rotation=0,0,0
scale=1,1,1
```

已存在且内容相同则幂等成功；已存在但内容不同返回 409。

### PUT /api/ue/config/model-table/entries

部分更新实例目录。允许字段：

- `model_id`
- `display_name`
- `sort_order`
- `enabled`

只更新请求中出现的字段。未传字段保持 DB 原值。`instance_id` 不存在返回 404。

恢复软删除实例时，使用：

```json
{"instance_id":"demo_model_1","enabled":true}
```

### DELETE /api/ue/config/model-table/entries/delete

软删除 instance，即设置 `ue_model_instances.enabled=false`。

效果：

- 该 instance 不再出现在 `model-table`。
- 该 instance 不再出现在 `model-config?mode=structured/runtime`。
- 该 instance 不再出现在 `scene-desired?mode=generated`。
- 不自动修改 `init_config.instance_ids`。
- 不自动清空 placement。

## model-config

### GET /api/ue/config/model-config

默认读取 raw `ue_config_documents.config_key=model_config`。不要用默认接口验证 placement 三表更新。

### GET /api/ue/config/model-config?mode=structured

返回 placements 结构，不包含 bounds、uri、source_path。

### GET /api/ue/config/model-config?mode=runtime

返回兼容 Server/UE runtime load 的结构，包含：

- `model_id`
- `visible`
- `transform`
- `bounds_size`

`bounds_size = base_bounds_size * abs(scale)`。

### PUT /api/ue/config/model-config/placements

批量或单个更新 placement。

推荐请求体使用 canonical `visible + transform`：

```json
{
  "instance_id": "demo_model_1",
  "visible": false,
  "transform": {
    "position": {"x": 1, "y": 2, "z": 3},
    "rotation": {"yaw": 0, "pitch": 0, "roll": 0},
    "scale": {"x": 1, "y": 1, "z": 1}
  }
}
```

也支持批量：

```json
{
  "placements": [
    {
      "instance_id": "demo_model_1",
      "visible": false
    }
  ]
}
```

兼容旧 flat 字段 `position / rotation / scale`，但不要和 `transform` 混用。混用返回 400。

禁止字段：

- `bounds_size`
- `uri`
- `format`
- `source_path`
- `model_id`
- `display_name`

这些字段出现会返回 400。

### PUT /api/ue/config/model-config/placements/{instance_id}

更新单个 placement，`instance_id` 来自 path。请求体语义同批量接口。

### DELETE /api/ue/config/model-config/placements/delete

删除 placement 行，也就是重置位姿。

请求：

```json
{"instance_id":"demo_model_1"}
```

或：

```json
{"instance_ids":["demo_model_1","demo_model_2"]}
```

注意：

- 只删除 `ue_model_instance_placements` 行。
- 不删除 instance。
- 不修改 `init_config.instance_ids`。
- 如果 instance 仍 `enabled=1`，查询 generated 视图时仍会出现该 instance，但 placement 会补默认值。

## scene-desired

### GET /api/ue/config/scene-desired

默认等价：

```http
GET /api/ue/config/scene-desired?mode=generated
```

由以下三表生成：

- `ue_model_types`
- `ue_model_instances`
- `ue_model_instance_placements`

不再默认读取 raw `scene_desired_config`，也不读取 `ue_scene_desired_models`。

### GET /api/ue/config/scene-desired?mode=legacy

迁移期兜底，从 `ue_config_documents.config_key=scene_desired_config` 读取 raw snapshot。

## raw preset PUT

以下接口是完整 JSON 文档替换，不做结构化拆分，不做字段 patch：

- `PUT /api/ue/config/path-config`
- `PUT /api/ue/config/chart-config`
- `PUT /api/ue/config/chart-binding`
- `PUT /api/ue/config/drone-config`
- `PUT /api/ue/config/render-config`
- `PUT /api/ue/config/video-streams`
- `PUT /api/ue/config/report-scripts`

写入目标都是 `ue_config_documents.content_json`。

特别说明：

- `PUT chart-config` 只更新 raw `chart_config`，不更新 `ue_chart_configs`，不影响 `chart-config?mode=structured/runtime`。
- `PUT chart-binding` 只更新 raw `chart_binding`，不更新 `data_bindings`，不影响 generated。
- `PUT video-streams` 只更新 raw `video_streams`，不更新 `video_sources`。
- `PUT report-scripts` 只更新 raw `report_scripts`，不更新 scripts/script_modules。

## bootstrap

### GET /api/ue/config/bootstrap

用于 Server 冷启动 hydrate，返回聚合配置：

```json
{
  "version": 1,
  "init_config": {},
  "path_config": {},
  "chart_config": {},
  "chart_binding": {},
  "drone_config": {},
  "model_table": {},
  "model_config": {},
  "scene_desired": {}
}
```

字段来源：

- `init_config`: raw
- `path_config`: raw
- `chart_config`: raw
- `chart_binding`: raw；raw 缺失时可 fallback generated
- `drone_config`: raw
- `model_table`: generated
- `model_config`: runtime
- `scene_desired`: generated

## 删除接口语义对照

| 接口 | 实际动作 | generated 视图效果 |
|---|---|---|
| `DELETE /api/ue/config/init-config/instance_ids/delete` | 从启动白名单移除 | 不删除 instance，不影响 model-table |
| `DELETE /api/ue/config/model-config/placements/delete` | 删除 placement 行 | enabled instance 仍输出，使用默认 placement |
| `DELETE /api/ue/config/model-table/entries/delete` | 软删除 instance，`enabled=false` | instance 从 model-table/model-config/scene-desired 消失 |
| `DELETE /api/ue/config/model-types/delete` | 软删除 model type | 若仍被 enabled instance 引用则返回 409 |

## 常见误解

### 修改 placement 后为什么 GET /model-config 看不到？

默认：

```http
GET /api/ue/config/model-config
```

可能返回 raw/legacy 文档。验证 placement 请使用：

```http
GET /api/ue/config/model-config?mode=structured
GET /api/ue/config/model-config?mode=runtime
GET /api/ue/config/scene-desired?mode=generated
```

### 删除 placement 后为什么实例还在？

`DELETE /model-config/placements/delete` 只是重置位姿。只要 instance 仍 `enabled=1`，generated 查询仍会输出该 instance，并补默认 placement。

### 删除 model-table entry 后为什么 placement 行还在？

model-table delete 是软删除 instance，不自动清空 placement。恢复 instance 后可能继续使用旧 placement；如需重置，请调用 placement delete 或 placement PUT。

### test_model_* 不显示怎么办？

先检查 `ue_model_instances.enabled`。如果为 0，可用 `PUT /api/ue/config/model-table/entries` 将 `enabled` 改回 true。

## 认证鉴权

P1-1 起，网关支持两种鉴权方式：

1. 正式登录 token：
   - `POST /api/auth/login`
   - 后续请求使用 `Authorization: Bearer <token>`
   - MySQL `ue_users` 保存用户账号和 bcrypt 密码 hash。
   - Redis 保存短期 session。Redis 重启后需要重新登录。
2. 联调兜底 token：
   - 请求头 `X-Gateway-Token: <static-token>`
   - 仅用于内网联调，生产环境应关闭或使用强随机值。

公开版本不预置登录用户或密码 hash。登录账号须由部署者自行创建，并使用本地生成的 bcrypt hash。模板中的静态 token 默认关闭且为空；本地写接口演示可以在私有配置中设置随机 token 后启用。

放行规则：

- `OPTIONS` 放行。
- `GET /api/health` 放行。
- `POST /api/auth/login` 放行。
- `GET /api/auth/me` 永远需要鉴权。
- `POST /api/auth/logout` 永远需要鉴权。
- `security.protect_read=false` 时普通 GET 放行。
- `security.protect_read=true` 时普通 GET 也需要鉴权。
- UE Config 写接口和 `POST /api/scripts/:id/preload` 需要鉴权。

登录中的未知依赖错误统一返回 `500 {"error":"auth_failed"}`；数据库及 Redis 错误详情只写服务端日志。401/403 的既有含义不变。

Redis session 查询失败时，middleware 返回 `503 {"error":"auth_unavailable"}`；token 缺失或错误时返回 `401 {"error":"unauthorized"}`。

## P1 联调加固

### health?deep=true

```http
GET /api/health
GET /api/health?deep=true
```

普通 health 保持轻量，返回：

```json
{"status":"ok"}
```

`deep=true` 会检查 MySQL ping、Redis ping、`fastapi.ws_data_url` 是否配置。MySQL 或 Redis 异常返回 HTTP 503，`status=degraded`。对外响应只返回通用 `ok/error/degraded` 状态，不暴露数据库或 Redis 原始错误；原始错误只写服务端日志。FastAPI WS 本轮只检查配置，不主动连接。

### raw preset PUT body limit

以下接口默认请求体最大 8MB，由 `server.max_body_bytes` 配置：

- `PUT /api/ue/config/path-config`
- `PUT /api/ue/config/chart-config`
- `PUT /api/ue/config/chart-binding`
- `PUT /api/ue/config/drone-config`
- `PUT /api/ue/config/render-config`
- `PUT /api/ue/config/video-streams`
- `PUT /api/ue/config/report-scripts`

超限返回 HTTP 413：

```json
{"error":"request body too large"}
```

## V20 /ws_data 推送与后台任务说明

- `GET /api/runtime/ws-data/status` 返回 Gateway 到 FastAPI `/ws_data` 的 best-effort 推送状态，包括连接状态、最近连接/断开/pong/推送时间、成功/失败计数、重连计数和按消息类型累计的推送数量。该接口不返回鉴权 header、token 或完整消息 payload。
- `POST /api/scripts/:id/preload` 保持原有语义：已鉴权用户触发后会构建 DataBundle、写 Redis preload cache，并在配置了 `fastapi.ws_data_url` 时尽力推送 FastAPI `/ws_data`。WS 推送失败只写 warn 日志，不让 HTTP preload 返回 500。
- `scheduler.preload.push_ws` 默认是 `false`。默认 scheduler 只做 cache-only preload，不会在未登录情况下自动推送 UE。只有运维显式设置 `scheduler.preload.push_ws=true` 时，scheduler 才会以 `source=scheduler auth_type=system` 的后台任务身份周期性推送 `/ws_data`。
- `mock_push.enabled` 默认是 `false`。它只用于开发联调；显式开启后会以 `source=mock_push auth_type=system` 周期性调用当前 Adapter 数据并推送 `/ws_data`。
- `fastapi.ws_auth` 是 Gateway 连接 FastAPI `/ws_data` 的服务对服务凭证，支持 `none`、`bearer`、`api_key`、`header`。它不是用户登录 token，Gateway 不会把用户 `Authorization: Bearer <token>` 转发给 FastAPI。
- `adapters.security.auth`、`adapters.energy.auth`、`adapters.education.auth` 是 Gateway 调外部学校系统 HTTP Adapter 时使用的服务凭证，支持 `none`、`bearer`、`api_key`、`header`。它们同样不是用户登录 token。
- 当前 `/ws_data` 推送仍是 best-effort + 稳定长连接 + stats，不是 ACK/retry/queue/dead-letter 可靠消息系统。完整可靠投递需要 FastAPI 配合 ACK、去重和消息序号协议，后续单独设计。

raw preset PUT 仍是 whole-document replace，只写 `ue_config_documents`；不影响结构化表和 runtime cache。

### X-Request-Id / X-Trace-Id

请求 ID 规则：

- 优先使用 `X-Request-Id`。
- 没有 `X-Request-Id` 时使用 `X-Trace-Id`。
- 两者都没有时由网关生成。
- `X-Request-Id` / `X-Trace-Id` 最多保留 128 字符，超出会截断。
- 响应头返回 `X-Request-Id`。
- 请求日志包含 `request_id/method/path/status/latency/client_ip`。

日志不应打印 password、access_token、Authorization、X-Gateway-Token 或 raw preset body。

### Auth 当前边界

当前没有注册接口、用户管理 CRUD、RBAC、refresh token、审计日志或登录失败限流。用户由 migration seed 或后续管理能力提供。

## Phase 1 样板脚本

脚本 DataBundle 接口不属于 UE Config API，但现场联调常一起验证。当前第一阶段包含三条 Mock 样板脚本：

| script_id | 名称 | 说明 |
| --- | --- | --- |
| `security_main` | 安防汇报脚本 | 安防出入、消防状态和视频样板 |
| `energy_main` | 后勤能耗汇报脚本 | 今日总能耗、楼栋能耗排行 |
| `education_main` | 教学汇报脚本 | 教室使用概况、今日课堂状态 |

验证命令：

```bash
curl http://127.0.0.1:8080/api/scripts/security_main/data-bundle
curl http://127.0.0.1:8080/api/scripts/energy_main/data-bundle
curl http://127.0.0.1:8080/api/scripts/education_main/data-bundle
```

三条脚本当前均走 Go 网关 Mock Adapter，数据先进入统一 DataBundle，再由 preload 链路写入缓存并尽力推送 FastAPI `/ws_data`。本阶段不连接真实门禁、能耗或教务系统。

注意：`migrations/013_seed_phase1_scripts.sql` 负责写入 `energy_main` / `education_main` 及对应 phase1 chart 配置。重复执行该 seed 会将 `chart_energy_total`、`chart_energy_building`、`chart_education_usage`、`chart_education_schedule` 恢复为样板默认配置；如现场已手工调整这些 chart，重跑前需要备份。

## Scheduler 预热

Scheduler 预热不是新的 HTTP API，而是启动后的后台任务。它按配置周期调用现有 `ScriptService.Preload(ctx, scriptID, force=true)`，将脚本 DataBundle 写入现有 Redis preload cache。

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

预热失败只写日志，不影响 HTTP 服务，也不影响其他脚本预热。真实外部 Adapter 接入后，scheduler 会周期性刷新对应 Adapter 数据。

注意：Scheduler 复用 `ScriptService.PreloadWithOptions(ctx, scriptID, Force=true, PushWS=push_ws)`。`push_ws=false` 时只刷新 Redis preload cache；只有显式配置 `push_ws=true` 时才会沿用 Preload 推送 FastAPI `/ws_data`。

## Adapter 接入准备

当前 Security / Energy / Education Adapter 默认均为 `mock`，因此没有真实外部系统也可以启动和联调。HTTP 模式只是接入骨架：

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

字段映射：
- `in_count/out_count/total_count/timestamp` -> `domain.AccessSummary`
- `alerts[].alert_id/level/location/message/timestamp` -> `[]domain.FireAlert`
- `buildings[].building_id/building_name/value/unit/timestamp` -> `domain.EnergySnapshot`
- `classrooms[].room_id/room_name/status/course_name/teacher_name/timestamp` -> `domain.EducationSnapshot`

外层数组字段必须明确存在：缺少 `alerts` / `buildings` / `classrooms` 会被视为上游协议错误并返回 adapter error；字段存在但值为显式空数组 `[]` 时是合法无数据，网关会返回空列表/空快照，不会让 DataBundle 生成 panic。

HTTP Adapter 非 2xx、JSON decode 失败或关键字段缺失会返回 error；service 仍按模块级降级处理，当前模块 warn 后跳过，不让整个 DataBundle panic。`mode=http` 但 `base_url` 为空时 warn fallback mock。
## report_scripts 层级编辑接口

`report_scripts` 是 UE/FastAPI 汇报编排配置，不是 Go DataBundle 脚本配置。

数据源边界：

- full/raw：`GET /api/ue/config/report-scripts` 读取 `ue_config_documents.config_key=report_scripts` 的 `content_json`。
- light/generated：`GET /api/ue/config/report-scripts?mode=light` 从 `scripts / script_modules / data_bindings / video_sources` 生成轻量目录。
- light/generated 与 full/raw 使用同一套 warmup ID 语义：`warmup.data` 是 `data_script_id`，`warmup.charts` 是 `binding_id`，`warmup.videos` 是 `video_id`。generated light 只列出有 `data_bindings.binding_id` 的可推送图表绑定，不会把 `chart_id` 塞进 `warmup.charts`。
- 层级 CRUD 只修改 full/raw `report_scripts`，不会自动修改 light/generated，也不会修改 DataBundle、scheduler 或 `/api/scripts/{id}/data-bundle`。
- `PUT /api/ue/config/report-scripts` 仍然是整份 JSON 替换。
- Server 后续应直接读取 Gateway 的 `GET /api/ue/config/report-scripts` 获取 full report_scripts。
- 层级 CRUD 会按 Gateway 当前 `protocol/ue.ReportScripts` 结构重新写回整份 report_scripts；protocol 未定义的未知字段不会保留。UE/Server 如果新增字段，需要先同步 Gateway protocol。

新增写接口都需要 `Authorization: Bearer <token>` 或 `X-Gateway-Token`：

- `POST /api/ue/config/report-scripts/scripts/create`
- `PUT /api/ue/config/report-scripts/scripts`
- `DELETE /api/ue/config/report-scripts/scripts/delete`
- `POST /api/ue/config/report-scripts/chapters/create`
- `PUT /api/ue/config/report-scripts/chapters`
- `DELETE /api/ue/config/report-scripts/chapters/delete`
- `POST /api/ue/config/report-scripts/scenes/create`
- `PUT /api/ue/config/report-scripts/scenes`
- `DELETE /api/ue/config/report-scripts/scenes/delete`
- `PUT /api/ue/config/report-scripts/scenes/camera`
- `POST /api/ue/config/report-scripts/scenes/bindings/create`
- `PUT /api/ue/config/report-scripts/scenes/bindings`
- `DELETE /api/ue/config/report-scripts/scenes/bindings/delete`
- `PUT /api/ue/config/report-scripts/scenes/warmup`

成功响应直接返回完整 report_scripts JSON 本体，不包 `code/data`。

层级语义：

- 新建 script 时 `chapters=[]`。
- 新建 chapter 时 `scenes=[]`。
- 新建 scene 可以不带 camera，输出为 `camera:null`；后续用 `PUT /scenes/camera` 设置。
- `PUT /scenes` 只改 `scene_id/label`，不改 camera、bindings、warmup。
- 如果 `PUT /scenes` 请求中携带 `camera`、`bindings` 或 `warmup`，Gateway 返回 400；请分别使用 `/scenes/camera`、`/scenes/bindings/*`、`/scenes/warmup`。
- bindings 以 `instance_id` 为 key；`PUT /scenes/bindings` 只替换请求中给出的 binding，不是全量覆盖。
- `visible=false` 和 `transform:null` 会保留。
- warmup 替换时会 trim、去空、去重；保存时不强校验 `data/charts/videos` 中的 ID 是否已经存在。`warmup.data` 是 `data_script_id`，`warmup.charts` 是 `binding_id`，`warmup.videos` 是 `video_id`。运行 `POST /api/ue/runtime/report-scenes/preload` 时，Gateway 会按 `warmup.data` 逐项执行 preload；不存在或失败的 `data_script_id` 会体现在 `results[].success=false` 和 `results[].error` 中。
- DELETE 接口幂等，目标不存在也返回当前完整 report_scripts。

所有层级写操作都通过 `UpdateContentJSONAtomic(ctx, "report_scripts", mutate)` 在事务和行锁内修改 `content_json`，并发请求会基于锁内最新 JSON 进行变更。

层级 CRUD 复用 `server.max_body_bytes` 请求体限制，超限返回：

```json
{"error":"request body too large"}
```
## UE Runtime Warmup 接口

### DataBundle 脚本与 report_scripts 汇报脚本

Gateway DataBundle 脚本来自 `scripts` 表，接口使用：

- `GET /api/scripts/{data_script_id}/data-bundle`
- `POST /api/scripts/{data_script_id}/preload`

DataBundle 响应保留历史字段 `script_id` / `scene_id`，并新增等价别名：

- `data_script_id` = Gateway DataBundle 脚本 ID，例如 `security_main`
- `data_scene_id` = Gateway DataBundle 场景 ID，例如 `campus_security`

UE `report_scripts` 是另一套汇报编排配置，字段应理解为：

- `report_script_id`
- `report_chapter_id`
- `report_scene_id`

两套 ID 不要混用。

### GET /api/ue/runtime/warmup-options

该接口只读，给 UE 编辑页提供可选资源目录，不修改任何数据。GET 是否需要鉴权继续遵守 `security.protect_read`。

返回字段来源：

- `data_scripts` 来自 `scripts` / `script_modules`
- `charts` 来自 `data_bindings` / `script_modules`
- `videos` 来自 `video_sources` / `script_modules`
- `instances` 来自 `ue_model_instances`

约定：

- `warmup.data` 填 `data_script_id`，来源 `scripts.script_id`
- `warmup.charts` 填 `binding_id`，来源 `data_bindings.binding_id`，不是 `chart_id`
- `warmup.videos` 填 `video_id`，来源 `video_sources.video_id`
- `instances` 是给 `report_scripts.scenes[].bindings[].instance_id` 使用的模型实例选项

### POST /api/ue/runtime/report-scenes/preload

该接口用于 Server/FastAPI 准备进入某个 UE report scene 时，让 Gateway 读取该 scene 的 `warmup.data` 并调用已有 `ScriptService.PreloadWithOptions` 预热对应 DataBundle。

请求体中的 `script_id` / `chapter_id` / `scene_id` 属于 `report_scripts` 汇报脚本体系，不是 DataBundle 的 `data_script_id`。

默认：

- `force=true`
- `push_ws=true`

第一版行为：

- 按 `warmup.data` 完整预热对应 data script
- `warmup.charts` / `warmup.videos` 会在响应中作为 requested 资源返回，供 Server/UE 参考
- Gateway 当前不按 `warmup.charts` / `warmup.videos` 过滤 `chart_data_push` 或 `video_streams_set`

该 POST 是运行时写/推送动作，必须带 Bearer token 或 `X-Gateway-Token`。Gateway 不会把用户 Bearer token 转发给 FastAPI 或外部学校系统。

真实学校实时数据仍通过 Adapter 进入 Gateway，再由 data script 生成 DataBundle。学校数据库里的 `building_id` / `room_id` / `gate_id` 不等于 UE `instance_id`；未来如需对象到模型实例映射，可评估复用或扩展 `model_bindings`，本轮不实现。

## nDisplay 模板读取

```http
GET /api/ue/config/ndisplay/{name}
```

直接返回 `ue_config_documents.config_key=ndisplay/{name}` 的原始 JSON；`name` 末尾可附加 `.json`，读取时会去掉该后缀。未找到文档返回 404，读取鉴权由 `security.protect_read` 决定。

新库可读取：

```bash
curl http://127.0.0.1:8080/api/ue/config/ndisplay/cluster_1node_2viewport.ndisplay.template
```

演示种子只包含待配置的 nDisplay 占位文档。实际 UE 集群运行前，需要提供自己的节点与视口配置。
