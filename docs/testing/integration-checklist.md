# 联调验收清单

本文档用于 UE 工程师、Server/FastAPI 工程师和后端联调时逐项确认。

## 演示环境准备

先按[项目 README](../../readme.md)在空演示库执行 14 个编号 migration，并从模板创建私有配置。公开版本不预置登录用户。

本清单默认使用静态 token：在私有配置中设置自己的随机 `security.static_token`、启用 `security.static_token_enabled` 后重启网关。在同一个 Bash 会话中设置：

```bash
export GATEWAY_URL="http://127.0.0.1:8080"
read -r -s -p '输入私有配置中的 Gateway token: ' GATEWAY_TOKEN
export GATEWAY_TOKEN
```

默认 `security.protect_read=false`，以下 GET 可以直接访问；若启用读鉴权，GET 同样需要添加鉴权请求头。Bearer 登录章节需要先自行创建 bcrypt 用户。

| 资源 | 新库中存在的 ID |
|---|---|
| 模型类型 / 实例 | `demo_model` / `demo_model_1` |
| 数据脚本 | `security_main`、`energy_main`、`education_main` |
| 汇报脚本 / 章节 / 场景 | `security_main` / `demo_chapter` / `demo_scene` |
| 预热数据 / 图表绑定 | `security_main` / `sim_3`、`sim_4` |

下面的写操作会修改演示数据，建议在独立演示库按顺序验证。

新建演示库的预热资源已在种子中对齐。若沿用此前初始化的演示库，可只更新该演示场景的预热项，不必重新导入整份 raw 配置：

```bash
curl -X PUT "${GATEWAY_URL}/api/ue/config/report-scripts/scenes/warmup" \
  -H "X-Gateway-Token: ${GATEWAY_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"script_id":"security_main","chapter_id":"demo_chapter","scene_id":"demo_scene","warmup":{"data":["security_main"],"charts":["sim_3","sim_4"],"videos":[]}}'
```

## 1. 基础连通性

```bash
curl ${GATEWAY_URL}/api/health
curl ${GATEWAY_URL}/api/ue/config/bootstrap
```

预期：

- `/api/health` 返回 `{"status":"ok"}`。
- `/api/ue/config/bootstrap` 返回 snake_case 字段：`init_config/path_config/chart_config/chart_binding/drone_config/model_table/model_config/scene_desired`。

## 2. 模型配置读取

```bash
curl ${GATEWAY_URL}/api/ue/config/model-table
curl "${GATEWAY_URL}/api/ue/config/model-config?mode=structured"
curl "${GATEWAY_URL}/api/ue/config/model-config?mode=runtime"
curl "${GATEWAY_URL}/api/ue/config/scene-desired?mode=generated"
```

预期：

- model-table 返回 enabled instance catalog。
- structured 返回 placements。
- runtime 返回 `last_load_by_instance_id`，包含计算后的 `bounds_size`。
- scene-desired 返回 generated `scene_models_desired`。

## 3. placement 修改验收

```bash
curl -X PUT "${GATEWAY_URL}/api/ue/config/model-config/placements/demo_model_1" \
  -H "X-Gateway-Token: ${GATEWAY_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"visible":false}'
```

预期：

- `GET model-config?mode=structured` 中 `demo_model_1.visible=false`。
- `GET model-config?mode=runtime` 中 `demo_model_1.visible=false`。
- `GET scene-desired?mode=generated` 中 `demo_model_1.visible=false`。

不要用默认 `GET /api/ue/config/model-config` 验证 placement。

## 4. placement 重置验收

```bash
curl -X DELETE "${GATEWAY_URL}/api/ue/config/model-config/placements/delete" \
  -H "X-Gateway-Token: ${GATEWAY_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"instance_id":"demo_model_1"}'
```

预期：

- DB 中 `ue_model_instance_placements` 不再有 `demo_model_1`。
- 如果 `ue_model_instances.enabled=1`，则 structured/runtime/generated 中仍出现 `demo_model_1`，但 placement 为默认值：
  - `visible=true`
  - `position=0,0,0`
  - `rotation=0,0,0`
  - `scale=1,1,1`

## 5. 禁用实例验收

```bash
curl -X DELETE "${GATEWAY_URL}/api/ue/config/model-table/entries/delete" \
  -H "X-Gateway-Token: ${GATEWAY_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"instance_id":"demo_model_1"}'
```

预期：

- `ue_model_instances.enabled=0`。
- model-table / model-config / scene-desired 不再输出 `demo_model_1`。
- `init_config.instance_ids` 不会自动变化。

恢复：

```bash
curl -X PUT "${GATEWAY_URL}/api/ue/config/model-table/entries" \
  -H "X-Gateway-Token: ${GATEWAY_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"instance_id":"demo_model_1","enabled":true}'
```

## 6. raw preset PUT 验收

```bash
curl -X PUT "${GATEWAY_URL}/api/ue/config/path-config" \
  -H "X-Gateway-Token: ${GATEWAY_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"version":1,"nav_playback":{"use_duration":true,"planned_duration_s":10,"planned_speed_cm_s":100},"plans_by_id":{}}'
```

说明：

- `PUT chart-config` 只更新 raw `chart_config`，不影响 structured/runtime。
- `PUT chart-binding` 只更新 raw `chart_binding`，不影响 generated。
- `PUT video-streams` 只更新 raw `video_streams`，不影响 `video_sources`。
- `PUT report-scripts` 只更新 raw `report_scripts`，不影响 scripts/script_modules。

## 7. preload 验收

```bash
curl "${GATEWAY_URL}/api/scripts/security_main/data-bundle"
curl -X POST "${GATEWAY_URL}/api/scripts/security_main/preload" \
  -H "X-Gateway-Token: ${GATEWAY_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"force":true}'
```

预期：

- DataBundle 返回 `modules/charts/videos`。
- preload 生成 DataBundle、写 Redis，并尽力推送 FastAPI `/ws_data`。
- FastAPI `/ws_data` 不可达时，preload 不应因 WS 失败而返回 500。

请求体可省略，`force` 默认 false；提供请求体时必须是单个 JSON 对象，`force` 若存在则必须为布尔值。以下请求预期 HTTP 400，且不会进入预热业务：

```bash
curl -X POST "${GATEWAY_URL}/api/scripts/security_main/preload" \
  -H "X-Gateway-Token: ${GATEWAY_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"force":"invalid-type"}'
```

`null`、数组、截断 JSON、尾随第二个 JSON 值及 `{"force":null}` 同样返回 400。修复示例为 `{"force":true}` 或 `{}`。
## 8. 认证鉴权验收

登录：

```bash
curl -X POST "${GATEWAY_URL}/api/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"operator","password":"<your-password>"}'
```

预期：

- 返回 `access_token`、`token_type=Bearer`、`expires_in` 和 `user`。
- 响应中不包含 `password_hash`、`token_hash` 或静态 token。
- 用户名/密码错误为 401，禁用用户为 403；未知数据库或 Redis 错误为 `500 {"error":"auth_failed"}`，详细错误只保留在服务端日志。

Bearer token 调写接口：

```bash
curl -X PUT "${GATEWAY_URL}/api/ue/config/model-config/placements/demo_model_1" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"visible":false}'
```

静态 token 兜底：

```bash
curl -X PUT "${GATEWAY_URL}/api/ue/config/model-config/placements/demo_model_1" \
  -H "Content-Type: application/json" \
  -H "X-Gateway-Token: <your-private-token>" \
  -d '{"visible":false}'
```

当前放行规则：

- `GET /api/health` 放行。
- `POST /api/auth/login` 放行。
- `GET /api/auth/me` 和 `POST /api/auth/logout` 必须鉴权。
- `security.protect_read=false` 时普通 GET 放行。
- UE Config 写接口和 `POST /api/scripts/:id/preload` 必须鉴权。

## 9. health?deep=true 验收

```bash
curl "${GATEWAY_URL}/api/health?deep=true"
```

预期：

- MySQL 正常时 `checks.mysql.status=ok`。
- Redis 正常时 `checks.redis.status=ok`。
- `checks.fastapi_ws.configured` 反映 `fastapi.ws_data_url` 是否配置。
- MySQL 或 Redis 异常时 HTTP 503，`status=degraded`。响应只暴露通用 `error` 状态，不返回原始连接错误、内网地址或端口；原始错误只写服务端日志。

## 10. raw preset body limit 验收

默认 raw preset PUT 最大 body 为 8MB，可通过 `server.max_body_bytes` 配置。

正常小 body：

```bash
curl -X PUT "${GATEWAY_URL}/api/ue/config/path-config" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"version":1}'
```

超限时预期 HTTP 413：

```json
{"error":"request body too large"}
```

该限制只作用于 raw preset PUT，不影响 placement PUT、model-table、model-types、init-config、bootstrap、preload。

## 11. request_id / trace_id 验收

指定 request id：

```bash
curl -i "${GATEWAY_URL}/api/health" \
  -H "X-Request-Id: test-rid-001"
```

预期响应头包含：

```text
X-Request-Id: test-rid-001
```

如果只传 `X-Trace-Id`，响应 `X-Request-Id` 使用该值；如果都不传，网关生成非空 `X-Request-Id`。`X-Request-Id` / `X-Trace-Id` 最多保留 128 字符，超出会截断。请求日志包含 `request_id/method/path/status/latency/client_ip`。

## 12. Phase 1 三条样板脚本验收

当前第一阶段样板脚本均使用 Go 网关内置 Mock Adapter，不连接真实门禁、能耗或教务系统。

```bash
curl ${GATEWAY_URL}/api/scripts/security_main/data-bundle
curl ${GATEWAY_URL}/api/scripts/energy_main/data-bundle
curl ${GATEWAY_URL}/api/scripts/education_main/data-bundle
```

预期：
- `security_main` 返回安防模块、图表和视频源。
- `energy_main` 返回后勤能耗模块和图表，包括 `energy.total.today` 与 `energy.building.snapshot`。
- `education_main` 返回教学模块和图表，包括 `education.classroom.usage` 与 `education.classroom.schedule`。

预热验证：

```bash
curl -X POST "${GATEWAY_URL}/api/scripts/energy_main/preload" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"force":true}'

curl -X POST "${GATEWAY_URL}/api/scripts/education_main/preload" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"force":true}'
```

如果 `security.protect_read=true`，读取 DataBundle 也需要携带 `Authorization: Bearer <token>` 或 `X-Gateway-Token`。

注意：`migrations/013_seed_phase1_scripts.sql` 负责写入 `energy_main` / `education_main` 及对应 phase1 chart 配置。重复执行该 seed 会将 `chart_energy_total`、`chart_energy_building`、`chart_education_usage`、`chart_education_schedule` 恢复为样板默认配置；如现场已手工调整这些 chart，重跑前需要备份。

## 13. Scheduler 预热验收

配置示例：

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

启动服务后观察日志：

```bash
journalctl -u dcx-gateway -f | grep "preload job"
```

预期日志中周期性出现 `security_main`、`energy_main`、`education_main` 的 `preload job success`。Scheduler 只调用现有 `ScriptService.Preload(ctx, scriptID, force=true)`，不注册新脚本；脚本仍由 migrations seed。

注意：Scheduler 复用 `ScriptService.PreloadWithOptions(ctx, scriptID, Force=true, PushWS=push_ws)`。`push_ws=false` 时只刷新 Redis preload cache；只有显式配置 `push_ws=true` 时才会沿用 Preload 推送 FastAPI `/ws_data`。

## 14. Adapter HTTP 模式联调准备

默认配置使用 mock：

```yaml
adapters:
  security:
    mode: mock
  energy:
    mode: mock
  education:
    mode: mock
```

切换 HTTP 模式时配置 `base_url`、`timeout_seconds` 和对应 path。`mode=http` 但 `base_url` 为空时，当前策略是 warn 后 fallback mock，不影响网关启动。

外部响应字段约定：
- Security access summary: `in_count/out_count/total_count/timestamp` -> `domain.AccessSummary`
- Security fire alerts: `alerts[].alert_id/level/location/message/timestamp` -> `domain.FireAlert`
- Energy snapshot: `buildings[].building_id/building_name/value/unit/timestamp` -> `domain.BuildingEnergyMetric`
- Education classroom usage: `classrooms[].room_id/room_name/status/course_name/teacher_name/timestamp` -> `domain.ClassroomUsage`

外层数组字段必须明确存在：缺少 `alerts` / `buildings` / `classrooms` 会被视为上游协议错误并返回 adapter error；字段存在但值为显式空数组 `[]` 时是合法无数据，网关会返回空列表/空快照，不会让 DataBundle 生成 panic。

真实系统接入前，请先用 mock 保持三条样板脚本链路稳定，再逐类切换到 HTTP adapter 验证字段、单位和超时。
## report_scripts 层级 CRUD 验收

full/raw 与 light/generated 的数据源不同：

- `GET /api/ue/config/report-scripts` 返回 `ue_config_documents.report_scripts.content_json`。
- `GET /api/ue/config/report-scripts?mode=light` 从 `scripts / script_modules / data_bindings / video_sources` 生成轻量目录。
- 层级 CRUD 只修改 full/raw，不会自动修改 light/generated，也不会影响 DataBundle 或 scheduler。
- 层级 CRUD 会按 Gateway 当前 protocol 结构重写整份 report_scripts；未知字段不会保留。UE/Server 新增字段前需要先同步 Gateway protocol。
- 这些写接口复用 `server.max_body_bytes`，请求体超限返回 HTTP 413。

## V20 /ws_data 推送联调

查看 Gateway 到 FastAPI `/ws_data` 的推送状态：

```bash
curl ${GATEWAY_URL}/api/runtime/ws-data/status
```

预期响应包含：

- `enabled`
- `connected`
- `last_connect_time`
- `last_pong_time`
- `success_count`
- `failed_count`
- `message_count_by_type`

该接口不会返回 token、鉴权 header 或完整消息 payload。若 `security.protect_read=true`，需要带 `Authorization: Bearer <token>` 或 `X-Gateway-Token`。

默认配置下：

- `scheduler.preload.push_ws=false`，scheduler 只写 Redis preload cache，不会在未登录情况下自动推 UE。
- `mock_push.enabled=false`，开发联调推送任务关闭。

只有以下场景会推送 FastAPI `/ws_data`：

1. 已鉴权用户调用 `POST /api/scripts/{id}/preload`。
2. 运维显式设置 `scheduler.preload.push_ws=true`。
3. 运维显式设置 `mock_push.enabled=true`。

FastAPI `/ws_data` 服务凭证配置在 `fastapi.ws_auth`，外部学校系统 Adapter 凭证配置在 `adapters.*.auth`。二者都是服务对服务凭证，不是 Gateway 用户登录 Bearer token。

开发联调时如需周期性推当前 mock/http Adapter 数据：

```yaml
mock_push:
  enabled: true
  interval_seconds: 5
  scripts:
    - security_main
    - energy_main
    - education_main
```

日志中后台推送应能看到 `source=scheduler` 或 `source=mock_push`，并标记 `auth_type=system`。当前推送仍是 best-effort，不包含 ACK/retry/queue/dead-letter。

新建空 scene，camera 后续补：

```bash
curl -X POST "${GATEWAY_URL}/api/ue/config/report-scripts/scenes/create" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"script_id":"security_main","chapter_id":"demo_chapter","scene_id":"scene_new","label":"新场景"}'
```

预期：`GET /api/ue/config/report-scripts` 中 `scene_new.camera=null`，`bindings=[]`，`warmup.data/videos/charts=[]`。

设置 camera：

```bash
curl -X PUT "${GATEWAY_URL}/api/ue/config/report-scripts/scenes/camera" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"script_id":"security_main","chapter_id":"demo_chapter","scene_id":"scene_new","camera":{"camera_id":"camera_main","position":{"x":0,"y":0,"z":0},"rotation":{"yaw":0,"pitch":0,"roll":0}}}'
```

新增或更新 binding：

```bash
curl -X POST "${GATEWAY_URL}/api/ue/config/report-scripts/scenes/bindings/create" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"script_id":"security_main","chapter_id":"demo_chapter","scene_id":"scene_new","bindings":[{"instance_id":"demo_model_1","model_id":"demo_model","visible":false,"transform":null}]}'
```

`PUT /api/ue/config/report-scripts/scenes/bindings` 只替换请求中的 `instance_id` 对应 binding，不会覆盖其他 bindings。

`PUT /api/ue/config/report-scripts/scenes` 只允许修改 scene_id/label；如果带 `camera`、`bindings` 或 `warmup` 会返回 400，请使用专用接口。

`PUT /api/ue/config/report-scripts/scenes/warmup` 保存时只做 trim、去空、去重，不强校验 `data/charts/videos` 中的 ID 是否已经存在。`warmup.data` 是 `data_script_id`，`warmup.charts` 是 `binding_id`，`warmup.videos` 是 `video_id`。运行 `POST /api/ue/runtime/report-scenes/preload` 时，不存在或失败的 `data_script_id` 会体现在 `results[].success=false` 和 `results[].error` 中。

DELETE 接口幂等；删除不存在的 script/chapter/scene/binding 仍返回当前完整 report_scripts。
## UE Report Scene Warmup 验收

读取可选资源目录：

```bash
curl "${GATEWAY_URL}/api/ue/runtime/warmup-options"
```

检查点：

- `data_scripts[].data_script_id` 可用于 `report_scripts.scenes[].warmup.data`
- `charts[].binding_id` 可用于 `report_scripts.scenes[].warmup.charts`
- `videos[].video_id` 可用于 `report_scripts.scenes[].warmup.videos`
- `instances[].instance_id` 可用于 `report_scripts.scenes[].bindings[].instance_id`

进入某个 report scene 前触发预热：

```bash
curl -X POST "${GATEWAY_URL}/api/ue/runtime/report-scenes/preload" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{"script_id":"security_main","chapter_id":"demo_chapter","scene_id":"demo_scene","force":true,"push_ws":true}'
```

说明：

- 请求里的 `script_id/chapter_id/scene_id` 是 UE `report_scripts` 汇报编排 ID。
- `warmup.data` 中的值是 Gateway DataBundle `data_script_id`，例如 `security_main`。
- 第一版会完整预热 `warmup.data` 指向的 data script；`warmup.charts/videos` 只作为关注资源返回，不做消息过滤。
- `push_ws=true` 时沿用 Gateway 现有 Preload 行为，best-effort 推送 FastAPI `/ws_data`。
