# Go 网关对接 UE/FastAPI 差距与数据库扩展判断

> 状态：历史归档。本文保留阶段性分析或设计提案，不代表当前功能与数据库现状。当前用法请以[文档首页](../README.md)、[接口说明](../api/ue-config.md)及源码为准。

本文是基于当前源码和 JSON 配置的二次静态验证结论。分析过程中未启动 UE、FastAPI、Web，也未修改 Go 业务代码、UE/FastAPI/Web 代码、JSON 配置、migrations、mapper、DataBundle 或数据库 schema。

## 1. 总体结论

当前 Go 网关不需要先做大范围扩展，也不建议把 UE/FastAPI 的 JSON 配置整体搬进 Go MySQL。

真实阻塞点主要有三类：

1. mapper 协议形状不匹配：Go 当前向 `/ws_data` 推 `rows`，FastAPI 当前 `chart_data_push` 校验要求 `values` 或 `cell_values`。
2. seed/config ID 未对齐：Go seed 使用 `security.access.today`、`security.fire.status` 作为 `binding_id`，UE/FastAPI `chart_binding.json` 当前使用 `sim_3`、`sim_4`。
3. UE/FastAPI 编排配置未对齐：Go seed 中的 `security_main` / `campus_security` 未出现在 `report_scripts.json`，`chart_h` / `chart_l` 出现在 `chart_binding.json`，但当前 `chart_config.json` 未包含这两个 chart。

当前 Go MySQL 表对第一阶段“业务展示配置”是基本够用的。近期优先不改 schema，通过 seed/config 对齐和 mapper 小修来收口。只有当 UE/FastAPI 明确要求 Go 承担跨协议映射、外部系统配置、或多套 UE script id 映射时，才考虑新增字段或表。

## 2. 实际读取目录与文件

Go 数据网关：

```text
<gateway-repo>
<gateway-repo>\internal
<gateway-repo>\migrations
```

UE / FastAPI / Web 工程：

```text
<ue-client-repo>
<ue-client-repo>\config
<ue-client-repo>\server
<ue-client-repo>\web
<ue-client-repo>\project_cpp\Source
```

重点读取：

```text
internal/model/entity
internal/model/protocol/gateway
internal/model/protocol/ue
internal/service/script_service.go
internal/service/mapper/ue_mapper.go
internal/repository
internal/cache
migrations/001_init_schema.sql
migrations/002_seed_mock_data.sql
docs/architecture/data-flow.md
docs/archive/ue-integration-analysis.md
config/chart_binding.json
config/chart_config.json
config/report_scripts.json
config/video_streams.json
config/model_config.json
config/model_table.json
config/path_config.json
config/render_config.json
config/init_config.json
config/drone_config.json
server/protocol/v1.py
server/ws/handlers.py
server/core/control/handlers_web.py
server/core/control/orchestration/dispatch.py
web/src/services/ueControl.ts
web/src/pages/CmdReport/ReportScriptPanel.tsx
project_cpp/Source/project_cpp/Private/Session/Handlers/DcChartMessageHandler.cpp
project_cpp/Source/project_cpp/Private/Session/Handlers/DcWarmupMessageHandler.cpp
```

## 3. 关键搜索词

```text
scripts
script_modules
data_bindings
video_sources
model_bindings
chart_templates
DataBundle
BuildDataBundle
Preload
BundleToWSMessages
ChartDataPush
VideoStreams
chart_data_push
video_streams_set
/ws_data
binding_id
chart_id
cell_values
values
rows
chart_binding
chart_config
report_scripts
video_streams
model_config
model_table
path_config
render_config
init_config
drone_config
warmup_video_catalog
warmup_data_catalog
```

## 4. 当前真实阻塞点

| 阻塞点 | 当前事实 | 判断 |
| --- | --- | --- |
| `chart_data_push` payload | Go mapper 输出 `rows`；FastAPI 要求 `values` 或 `cell_values` | 需要改 mapper，不需要改 DataBundle |
| `binding_id` | Go seed 为 `security.access.today` / `security.fire.status`；UE 配置为 `sim_3` / `sim_4` | seed/config 对齐问题，不是 schema 问题 |
| `chart_id` | FastAPI 会根据 `chart_binding` 用 `binding_id` 补 `chart_id`；UE C++ 最终要求 `chart_id` | Go 可不直接存完整 UE chart_config，但必须保证 binding 可解析 |
| `chart_h` / `chart_l` | 出现在 `chart_binding.json`，未出现在当前 `chart_config.json` | UE/FastAPI 配置一致性问题 |
| `security_main` / `campus_security` | Go seed 中存在，`report_scripts.json` 中不存在 | 编排配置未对齐，不建议 Go 复制完整 report_scripts |
| `video_streams_set` | Go mapper 会推；FastAPI `/ws_data` 只处理 `chart_data_push`，非该类型直接跳过 | 当前推送不会产生视频效果，需要确认是否保留 |
| 视频流 `vs_cam_65` | Go `video_sources.video_id` 与 UE `video_streams.stream_id` 已对齐 | seed 已基本可用 |
| 模型绑定 | Go 有 `model_bindings.bind_model_id`，但当前 DataBundle 构建未填充 `ModelBindings` | 业务能力未接入，不是本次必须 schema 扩展 |

## 5. Go MySQL 表逐表判断

| 表名 | 当前职责 | 字段是否够用 | seed 是否与 UE/FastAPI 对齐 | 是否新增字段 | 是否新增表 | 判断 |
| --- | --- | --- | --- | --- | --- | --- |
| `scripts` | 保存 Go 网关可构建的业务脚本：`script_id`、`scene_id`、状态 | 近期够用 | 未对齐。Go 为 `security_main` / `campus_security`，UE 为 `testscript`、`drone.view` 等 | 暂不新增 | 否 | 先通过 seed 或 UE `report_scripts.json` 对齐。只有出现 Go script id 与 UE report script id 必须长期不同的需求，才考虑 `ue_script_id` / `ue_scene_id` |
| `script_modules` | 保存脚本内业务展示模块及目标资源引用 | 近期够用 | 部分未对齐。`target_id=chart_h/chart_l` 当前不在 `chart_config.json`；`vs_cam_65` 对齐 | 暂不新增 | 否 | `config_json` 已能承载轻量扩展。payload shape 不建议放表里，先由 mapper 固定规则处理 |
| `data_bindings` | 保存业务 `data_key` 到 FastAPI/UE 图表绑定关系 | 够用 | 未对齐。`binding_id` 当前应与 `chart_binding.json` 一致 | 不新增 | 否 | 已有 `binding_id` 和 `chart_id`，不需要 `ue_binding_id` / `ue_chart_id`。直接让字段值与 UE 配置一致即可 |
| `video_sources` | 保存长期视频源配置 | 够用 | `vs_cam_65` 和 URL 与 `video_streams.json` 对齐 | 不新增 | 否 | `status` 不应进 MySQL，当前 Redis `VideoCache` 已符合运行态状态设计 |
| `model_bindings` | 保存业务对象与 UE 点位/实例绑定关系 | 基本够用 | 当前仅有 `gate_model_01`，是否等于真实 UE instance_id 需确认 | 条件性新增 | 否 | 若 `bind_model_id` 被定义为业务别名，且 UE 必须用真实 `instance_id`，才考虑新增 `ue_instance_id`；若直接把 `bind_model_id` 约定为 UE instance_id，则不改 schema |
| `chart_templates` | 保存 Go 展示模板引用或轻量模板配置 | 够用但当前未被主流程读取 | 与 UE `chart_config.json` 无直接对齐 | 不新增 | 否 | 不应镜像完整 chart_config。需要使用时只保存 Go 展示模板或转换策略 |

## 6. UE JSON 是否应进入 Go MySQL

### 6.1 report_scripts.json

判断：Go 不应保存完整 `report_scripts.json`。

源码依据：

1. FastAPI `report_scripts` 由 `server/core/config_io/report_scripts.py` 读写。
2. Web 报告脚本编辑器直接维护 scenes、camera、bindings、warmup videos/charts。
3. FastAPI orchestration 使用 `find_scene` 读取 `report_scripts.json` 并驱动 UE 编排。

Go 应保存：

```text
script_id
scene_id
业务模块配置
```

Go 不应保存：

```text
camera
path
model actions
scene bindings transform
完整 chapters/scenes 编排
```

近期建议：不改 schema。若需要让 `security_main` 参与 UE 编排，优先在 UE/FastAPI 配置侧新增或对齐 `report_scripts.json`；或者把 Go seed 的 `scripts.script_id/scene_id` 改成现有 UE report script/scene。

只有当 Go 内部必须保留业务脚本 ID，且 UE/FastAPI 使用另一套 report script ID 时，才考虑在 `scripts` 增加：

```text
ue_script_id
ue_scene_id
```

### 6.2 chart_config.json

判断：Go 不应保存完整 `chart_config.json`。

源码依据：

1. FastAPI 启动时 hydrate `chart_config` 到 state。
2. Web `ChartRender` / `CmdReport` 以 `chart_config.json` 作为 chart layout、rows、cols、visible、panel_key 的来源。
3. UE 连接时 FastAPI 会下发完整 `chart_config_set`。

Go 应保存：

```text
chart_id 引用
template 引用
业务标题
业务数据 key
```

Go 不应保存：

```text
position
size
visible
panel_key
rows/cols 的 UE 布局真相
完整 cell_values 初始模板
```

当前 `chart_templates` 不应镜像 `chart_config`。它更适合保存 Go 侧展示模板或未来 mapper 转换策略。

### 6.3 chart_binding.json

判断：Go 的 `data_bindings` 必须与 `chart_binding.json` 对齐，但不需要新增字段。

当前已有字段：

```text
data_key
binding_id
chart_id
module_id
enabled
```

这些字段足够表达：

```text
业务数据 key -> FastAPI binding_id -> UE chart_id
```

两种对齐方案：

| 方案 | 做法 | Go 影响 | UE/FastAPI 影响 | 可读性 | 扩展性 | 推荐 |
| --- | --- | --- | --- | --- | --- | --- |
| A | Go seed 改用 UE 现有 `sim_3` / `sim_4` | 改 seed，业务语义变弱 | 不改 UE 配置 | 较差 | 一般 | 临时联调可用 |
| B | UE `chart_binding.json` 改用 `security.access.today` / `security.fire.status` | Go seed 保持业务语义 | 改 UE 配置 | 较好 | 较好 | 推荐 |

推荐方案 B：让 binding_id 使用业务语义 ID，`chart_binding.json` 负责映射到 UE chart_id。这样 Go、FastAPI state、Web chart_realtime 都更容易读懂。

### 6.4 video_streams.json

判断：Go `video_sources` 不需要镜像完整 `video_streams.json`，但应保持 `video_id == stream_id`。

当前事实：

```text
Go video_sources.video_id = vs_cam_65
UE video_streams.stream_id = vs_cam_65
```

这部分已经对齐。

`status` 应继续放 Redis，因为它是在线、播放、延迟等运行态状态，不是长期配置。

`video_streams_set` 当前不被 FastAPI `/ws_data` 消费。后续有两个选择：

1. 暂时保留但视为无效副作用，只用于未来协议扩展。
2. 从 preload WS 消息中去掉视频推送，视频继续走 `report_scripts.warmup.videos -> video_streams.json -> warmup_video_catalog`。

当前更稳的是方案 2，但这是后续代码建议，本次不修改。

### 6.5 model_config.json / model_table.json

判断：Go 不应保存 UE 模型资源配置，只应保存业务对象与 UE 点位/实例的绑定关系。

Go 应保存：

```text
业务对象类型
业务对象 ID
bind_model_id 或 ue_instance_id
关联 data_keys
```

Go 不应保存：

```text
model asset path
transform
visible
bounds
UE 加载状态
完整 model table
```

是否新增字段取决于团队约定：

```text
如果 bind_model_id 直接等于 UE instance_id，则不改 schema。
如果 bind_model_id 是 Go 业务别名，而 UE 另有 instance_id，则可考虑新增 ue_instance_id。
```

### 6.6 path/render/init/drone/nDisplay

这些配置属于 UE/FastAPI 的渲染、路径、初始化、无人机和部署控制，不应进入 Go 网关数据库。

Go 只应通过 FastAPI 的既有控制入口表达业务数据或业务意图，不应直接管理这些渲染和编排配置。

## 7. mapper 是否需要改

需要改 mapper，但不需要改 DataBundle。

当前 Go：

```go
type gateway.ChartData struct {
    Rows [][]string `json:"rows,omitempty"`
}
```

当前 mapper：

```go
payload := ue.ChartDataPush{
    BindingID: chart.BindingID,
    Timestamp: bundle.GeneratedAt,
    Rows:      chart.Rows,
}
```

当前 FastAPI：

```text
payload.binding_id 必须非空
payload.cell_values 或 payload.values 必须存在
payload.rows 不会被当作有效字段
```

当前 UE C++：

```text
最终需要 payload.chart_id
优先解析 table.cell_values / cell_values
否则解析 values
```

FastAPI 会基于 `chart_binding` 把 `binding_id` 解析成 `chart_id` 并转发给 UE。因此 Go 不必须直接发 `chart_id`，但绑定配置必须对齐。

最小修改建议：

1. 保持 `gateway.ChartData.Rows` 不变，作为 Go DataBundle 的展示数据。
2. mapper 将 `Rows` 转成 `cell_values`。
3. `protocol/ue.ChartDataPush` 建议新增 `CellValues [][]string json:"cell_values,omitempty"`。
4. 可选保留 `Rows` 作为兼容字段，但不能只发 `rows`。
5. 不建议为了这件事修改 DataBundle。
6. 不建议新增数据库字段保存 payload shape。

建议目标 payload：

```json
{
  "type": "chart_data_push",
  "payload": {
    "binding_id": "security.access.today",
    "cell_values": [
      ["指标", "数值", "单位"],
      ["今日进入", "1800", "人次"],
      ["今日离开", "1480", "人次"],
      ["今日总数", "3280", "人次"]
    ]
  }
}
```

如果团队决定先采用现有 UE binding：

```json
{
  "type": "chart_data_push",
  "payload": {
    "binding_id": "sim_3",
    "cell_values": [
      ["指标", "数值", "单位"],
      ["今日进入", "1800", "人次"],
      ["今日离开", "1480", "人次"],
      ["今日总数", "3280", "人次"]
    ]
  }
}
```

也可以使用 FastAPI 支持的嵌套形状：

```json
{
  "type": "chart_data_push",
  "payload": {
    "binding_id": "security.access.today",
    "table": {
      "cell_values": [
        ["指标", "数值", "单位"],
        ["今日进入", "1800", "人次"],
        ["今日离开", "1480", "人次"],
        ["今日总数", "3280", "人次"]
      ]
    }
  }
}
```

顶层 `cell_values` 更直接，建议优先使用。

## 8. DataBundle 是否需要改

不需要。

`gateway.DataBundle` 目前表达的是 Go 网关给 FastAPI 的统一展示数据包，不是 UE 原生协议。它包含：

```text
modules
charts
videos
model_bindings
```

`charts.rows` 可以继续作为 Go 侧业务表格数据。协议适配应放在 mapper 层完成，而不是把 DataBundle 改成 UE/FastAPI 的 `cell_values` 命名。

## 9. protocol/ue 是否需要改

建议小改，但不是数据库扩展。

当前 `protocol/ue.ChartDataPush` 缺少：

```text
cell_values
values
chart_id
table
```

最小建议：

```text
新增 CellValues [][]string `json:"cell_values,omitempty"`
可选新增 Values map[string]string 或 map[string]any
可选新增 ChartID string `json:"chart_id,omitempty"`，用于调试或特殊绕过，但正常仍由 FastAPI 解析
```

若想避免结构体扩展，也可以让 mapper 直接构造 `map[string]any` payload。但从可维护性看，小幅扩展 `protocol/ue.ChartDataPush` 更清晰。

## 10. 数据库扩展判断表

| 数据项 | 当前是否已存 | 当前表/字段 | 应进 Go MySQL | 应放 Redis | 应留 UE/FastAPI JSON | 是否需要新增字段/表 | 理由 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `script_id` | 是 | `scripts.script_id` | 是 | 否 | 可在 `report_scripts` 有对应项 | 否 | Go 业务脚本入口 |
| `scene_id` | 是 | `scripts.scene_id` | 是 | 否 | 是，UE 编排也有 scene | 否，先 seed 对齐 | Go 可保存引用，不保存完整 scene |
| UE report script id | 未单独存 | 无 | 条件性 | 否 | 是 | 条件性新增 `ue_script_id` | 仅当 Go/UE 两套 ID 必须分离才需要 |
| `module_id` | 是 | `script_modules.module_id` | 是 | 否 | 否 | 否 | Go 业务展示模块 ID |
| `data_key` | 是 | `script_modules.data_key` / `data_bindings.data_key` | 是 | 可缓存数据结果 | 否 | 否 | Go 查 adapter 和绑定图表的核心 key |
| `binding_id` | 是 | `data_bindings.binding_id` | 是 | 否 | 是，`chart_binding` 也维护 | 否 | 需要与 FastAPI chart_binding 对齐 |
| `chart_id` | 是 | `data_bindings.chart_id` / `script_modules.target_id` | 作为引用可存 | 否 | 是 | 否 | Go 只存引用，不存图表布局 |
| chart layout/position/size | 部分轻量位置在 `config_json` | `script_modules.config_json` | 不建议 | 否 | 是 | 否 | UE/FastAPI 控制渲染布局 |
| chart template | 是 | `chart_templates` / `script_modules.template` | 是，轻量模板 | 否 | chart_config 保持 UE 侧 | 否 | Go 展示模板，不等于 UE chart_config |
| payload shape | 否 | 无 | 不建议近期入库 | 否 | 协议代码维护 | 否 | mapper 固定转换即可 |
| `video_id` / `stream_id` | 是 | `video_sources.video_id` | 是 | 状态可 Redis | 是，`video_streams.stream_id` | 否 | 保持同一个 ID |
| video play_url | 是 | `video_sources.play_url` | 是 | 否 | 是，`video_streams` | 否 | Go 可保存业务视频源，UE 也有视频库 |
| video status | 否，运行时缓存 | `VideoCache` | 否 | 是 | 可有 UE 状态 | 否 | 运行态，不进 MySQL |
| `bind_model_id` | 是 | `video_sources.bind_model_id` / `model_bindings.bind_model_id` | 是 | 否 | 可映射 UE instance | 条件性 | 若其就是 UE instance_id，不新增 |
| UE `model_id` / `instance_id` | 部分 | `model_bindings.bind_model_id` | 只存绑定引用 | 否 | 是 | 条件性新增 `ue_instance_id` | 仅当现字段语义不够 |
| model asset path / transform / visible | 否 | 无 | 否 | 否 | 是 | 否 | UE 渲染和场景状态 |
| path/camera config | 否 | 无 | 否 | 否 | 是 | 否 | UE/FastAPI 编排控制 |
| render/nDisplay config | 否 | 无 | 否 | 否 | 是 | 否 | 部署和渲染配置 |
| external adapter endpoint | 否 | 无 | 条件性 | 可缓存健康状态 | 否 | 未来可新增 `adapter_configs` | 接真实外部系统时再做 |
| external credentials | 否 | 无 | 条件性，不建议明文 | 否 | 否 | 未来按安全方案新增 | 当前阶段不接真实系统 |
| data refresh interval | 否 | 可放 `script_modules.config_json` | 条件性 | 否 | 否 | 暂不新增 | 接 scheduler 后再定 |
| schema/protocol version | 部分 | `DataBundle.Version`，UE envelope version | 不建议新增 | 否 | 协议代码维护 | 否 | 当前不是数据库职责 |

## 11. 近期不需要改 schema 的内容

以下都可通过 seed/config 或 mapper 解决：

```text
binding_id/chart_id 对齐
script_id/scene_id 对齐
video stream_id/play_url 对齐
chart_data_push 的 rows -> cell_values 转换
video status 放 Redis
chart layout/visible/position 留在 UE/FastAPI JSON
report_scripts camera/path/model/actions 留在 UE/FastAPI JSON
```

## 12. 可能需要新增字段的条件

| 可能字段 | 所在表 | 只有当满足这个条件才新增 |
| --- | --- | --- |
| `ue_script_id` | `scripts` | Go 业务 script_id 必须与 UE report script_id 长期分离 |
| `ue_scene_id` | `scripts` | Go scene_id 必须与 UE report scene_id 长期分离 |
| `ue_instance_id` | `model_bindings` | `bind_model_id` 不能直接作为 UE instance_id 使用 |
| `payload_strategy` | `chart_templates` 或 `script_modules.config_json` | 同一类 DataBundle rows 需要多种不同协议转换策略 |
| `refresh_interval_ms` | `script_modules.config_json` 或未来 scheduler 表 | 接入 scheduler，且不同模块刷新周期不同 |

当前阶段不建议直接新增这些字段。

## 13. 可能需要新增表的条件

| 可能表 | 当前是否需要 | 条件 |
| --- | --- | --- |
| `external_sources` | 否 | 接真实门禁、消防、能耗、教务等外部系统后 |
| `adapter_configs` | 否 | adapter endpoint、认证、租户、超时、重试需要持久化管理后 |
| `protocol_mappings` | 否 | 多套 FastAPI/UE 协议版本并存，且映射不能靠代码固定后 |
| `scheduler_jobs` | 否 | preload 之外需要后台周期推送后 |

## 14. seed/config 对齐建议

优先确认一个方向：

### 推荐方向

让 UE/FastAPI 的 `chart_binding.json` 使用 Go 的业务语义 binding：

```text
security.access.today -> 有效 chart_id
security.fire.status -> 有效 chart_id
```

同时确认这些 chart_id 存在于 `chart_config.json`：

```text
chart_h
chart_l
```

如果 `chart_h` / `chart_l` 不是当前有效 chart，则改为当前真实存在的 chart，例如 `chart_12` 或其他有效配置。

### 临时联调方向

让 Go seed 的 `data_bindings.binding_id` 改为：

```text
security.access.today -> sim_3
security.fire.status -> sim_4
```

这种方式改动小，但会降低 Go 侧业务语义可读性。

## 15. 后续开发任务

| 优先级 | 任务名 | 目标 | 涉及目录/文件 | 改 Go 代码 | 改 Go DB seed | 改 Go DB schema | 需要 UE/FastAPI 配合 | 验收方式 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| P0 | 确定 `chart_data_push` payload shape | 固定 Go 推送必须包含 `cell_values` 或 `values` | `server/protocol/v1.py`、`handlers_web.py`、Go mapper | 否，本阶段只确认 | 否 | 否 | 是 | 双方确认协议样例 |
| P0 | 对齐 `binding_id/chart_id` | 让 FastAPI 能从 Go binding 解析 chart_id | `data_bindings` seed、`chart_binding.json` | 否 | 条件性 | 否 | 是 | FastAPI state 中能查到绑定 |
| P0 | 确认 `chart_h/chart_l` 或改到有效 chart_id | 避免绑定到不存在的 chart | `chart_config.json`、`chart_binding.json`、Go seed | 否 | 条件性 | 否 | 是 | Web/UE 能看到目标 chart |
| P0 | 决定 `security_main` 是否对齐 `report_scripts.json` | 明确 Go script 是否参与 UE 报告编排 | `scripts` seed、`report_scripts.json` | 否 | 条件性 | 否 | 是 | report script 可引用同一场景 |
| P1 | 修改 Go mapper 输出 `cell_values` / `values` | 让 preload 推送通过 FastAPI 校验 | `internal/service/mapper/ue_mapper.go`、`protocol/ue` | 是 | 否 | 否 | 否 | POST preload 后 FastAPI 不报 INVALID_PAYLOAD |
| P1 | 修改 Go seed 或 UE chart_binding 配置 | 消除 binding 不一致 | `002_seed_mock_data.sql` 或 `chart_binding.json` | 否 | 是或改 UE config | 否 | 是 | Go binding_id 能映射 chart_id |
| P1 | 补 mapper 单元测试 | 固定 rows -> cell_values 行为 | Go test files | 是 | 否 | 否 | 否 | `go test ./...` |
| P1 | 写 e2e preload 样例 | 验证 preload 推送 FastAPI | Go/文档/脚本 | 可选 | 否 | 否 | 是 | mock `/ws_data` 收到 2 条 chart_data_push |
| P2 | 明确 `video_streams_set` 去留 | 避免无效 WS 消息 | Go mapper、FastAPI 协议 | 条件性 | 否 | 否 | 是 | 若保留则 FastAPI 支持；若不保留则 Go 不推 |
| P2 | 判断 `model_bindings` 是否进入 DataBundle | 支持模型点位业务绑定 | `script_service.go`、`model_bindings` repo | 是 | 条件性 | 否 | 是 | DataBundle 中出现可用 model_bindings |
| P2 | 整理 `chart_templates` 使用方式 | 明确模板是否只服务 Go 展示 | `chart_templates`、mapper | 条件性 | 条件性 | 否 | 否 | 模板使用规则成文 |
| P3 | 真实 Adapter | 接门禁/消防/能耗等真实外部系统 | `internal/adapter` | 是 | 条件性 | 条件性 | 否 | 外部系统联调通过 |
| P3 | scheduler | 周期刷新和推送 | `internal/scheduler` | 是 | 条件性 | 条件性 | 否 | 定时任务稳定运行 |
| P3 | Auth | 外部系统凭据和权限 | config/DB/API | 是 | 否 | 条件性 | 可选 | 安全策略通过 |

## 16. 不建议做的事

当前阶段不建议：

1. 把完整 `report_scripts.json` 存进 Go MySQL。
2. 把完整 `chart_config.json` 存进 Go MySQL。
3. 把 UE 模型资源、路径、渲染、nDisplay 配置存进 Go MySQL。
4. 为了 payload shape 修改数据库 schema。
5. 为了视频推送强行让 Go 直接控制 UE。
6. 重构 `buildModuleData`。
7. 接 Kafka、Consul、真实外部系统或 scheduler。
8. 在未确认协议前扩展大量表字段。

## 17. 需要和 UE/FastAPI 工程师确认的问题

1. `chart_h` / `chart_l` 是否仍是有效 UE 图表容器？如果是，为什么不在当前 `chart_config.json` 中？
2. `binding_id` 是否可以改为业务语义 ID，如 `security.access.today`？
3. `security_main` 是否需要出现在 `report_scripts.json`？如果需要，对应 chapter/scene/camera/warmup 应由谁维护？
4. `/ws_data` 是否未来需要支持 `video_streams_set`？如果不支持，Go 是否应停止推该消息？
5. UE/FastAPI 期望 chart runtime data 使用顶层 `cell_values`，还是 `table.cell_values`？
6. `bind_model_id` 是否应直接等于 UE `instance_id`？

## 18. 最小可行顺序

```text
1. 双方确认 chart_data_push payload 使用 cell_values 或 values
2. 确认 binding_id 命名方案
3. 对齐 chart_binding.json 与 Go data_bindings seed
4. 确认 chart_id 是否存在于 chart_config.json
5. 修改 Go mapper：Rows -> cell_values
6. 用 mock /ws_data 验证 POST preload
7. 再决定 video_streams_set 是否保留
8. 最后再讨论 schema 扩展
```

## 19. V11 更新：UE Config DB + Full/Raw API

V11 已把 UE/FastAPI 原始 JSON 配置作为文档保存到 Go 数据库：

```text
ue_config_documents.content_json
```

新的边界：

| 类型 | 来源 | 是否走 mapper |
| --- | --- | --- |
| full/raw config | `ue_config_documents.content_json` | 否 |
| generated/light config | Go 业务表 | 是，走 `ue_config_mapper.go` |
| runtime push | `gateway.DataBundle` | 是，走 `ue_mapper.go` |

本阶段不把所有 UE JSON 拆成大量业务表。`model_config`、`path_config`、`render_config`、`init_config`、`drone_config`、`ndisplay` 等仍以原始 JSON 文档形式保存和返回。

当前 binding 策略采用临时联调优先：

```text
data_key 保持 security.access.today / security.fire.status
binding_id 对齐 full chart_binding.json 的 sim_3 / sim_4
chart_id 保持 chart_h / chart_l
```

`rows -> cell_values` 已在运行时 mapper 修复，light report 输出使用专用 light struct，不返回零值 `camera`。
 14.78  21.1
