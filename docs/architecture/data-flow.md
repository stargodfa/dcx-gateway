# 数据流转与结构体契约

英文版本：[Data Flow and Struct Contracts](data-flow.en.md)。

本文档说明当前 Go 数据网关中，数据如何在结构体、分层、HTTP DTO 以及 FastAPI/UE WebSocket 消息之间流转。

本文档基于当前代码中的以下模块整理：

- `internal/model/entity`
- `internal/model/domain`
- `internal/model/dto`
- `internal/model/protocol/gateway`
- `internal/model/protocol/ue`
- `internal/adapter/security`
- `internal/repository`
- `internal/cache`
- `internal/service/script_service.go`
- `internal/service/mapper/ue_mapper.go`
- `internal/api/handler/script_handler.go`
- `internal/api/router.go`
- `internal/websocket/fastapi_client.go`
- `internal/bootstrap/app.go`
- `migrations/001_init_schema.sql`
- `migrations/002_seed_mock_data.sql`

本文档只做说明，不改变 HTTP 契约、`gateway.DataBundle` 结构、mapper 输出、数据库迁移脚本或业务行为。

## 1. 总体结构体流转

当前文字版链路如下：

```text
external raw / mock
  -> domain.AccessSummary / domain.FireAlert
  + entity.Script / ScriptModule / DataBinding / VideoSource
  -> gateway.ModuleData / ChartData / VideoSource
  -> gateway.DataBundle
  -> dto.APIResponse[gateway.DataBundle]
  -> ue.ChartDataPush / ue.VideoStreams / ue.WSMessage
```

各层含义：

- `domain` 结构体表示外部数据进入网关后形成的内部业务对象。它不是 MySQL 表，也不是 UE 协议结构。
- `entity` 结构体表示本地 MySQL 配置骨架。它不保存实时业务值，例如今日出入人数、消防告警状态。
- `gateway.DataBundle` 是 Go 网关为 FastAPI 生成的统一展示数据包。
- `dto` 结构体只作为 HTTP 请求/响应外壳使用。
- `protocol/ue` 结构体用于兼容现有 FastAPI/UE 协议消息。
- `mapper` 只负责把 `gateway` 结构转换为 `ue` 协议消息。

## 2. Entity 配置取值来源

Entity 来自本地 MySQL 表，表结构定义在 `migrations/001_init_schema.sql` 中。当前 `security_main` 的种子示例来自 `migrations/002_seed_mock_data.sql`。

| 结构体 | 字段 | 数据库表字段 | 当前 seed 示例 | 用途 | 是否参与当前 V9 主链路 |
|---|---|---|---|---|---|
| `entity.Script` | `ScriptID` | `scripts.script_id` | `security_main` | 选择一个汇报/展示脚本 | 是 |
| `entity.Script` | `Name` | `scripts.name` | 安防汇报脚本名称 | 脚本人类可读名称 | 间接 |
| `entity.Script` | `SceneID` | `scripts.scene_id` | `campus_security` | 填充 `gateway.DataBundle.SceneID` | 是 |
| `entity.Script` | `Status` | `scripts.status` | `enabled` | Service 会拒绝禁用脚本 | 是 |
| `entity.ScriptModule` | `ModuleID` | `script_modules.module_id` | `today_access_card`、`fire_alert_card`、`gate_video_panel` | 脚本内展示模块 ID | 是 |
| `entity.ScriptModule` | `ScriptID` | `script_modules.script_id` | `security_main` | 将模块归属到某个脚本 | 是 |
| `entity.ScriptModule` | `Type` | `script_modules.type` | `stat_card`、`alert_status`、`video` | 填充 `gateway.ModuleData.Type` | 是 |
| `entity.ScriptModule` | `Title` | `script_modules.title` | 今日出入人数、消防告警、校门口监控 | 填充模块/图表标题 | 是 |
| `entity.ScriptModule` | `DataKey` | `script_modules.data_key` | `security.access.today`、`security.fire.status`、`security.video.gate_01` | 业务语义键，用于 service switch 和绑定查询 | 是 |
| `entity.ScriptModule` | `Template` | `script_modules.template` | `stat_card_basic`、`status_basic`、`video_basic` | 有图表数据时填充 `gateway.ChartData.Template` | 图表参与 |
| `entity.ScriptModule` | `TargetKind` | `script_modules.target_kind` | `chart`、`video` | 告诉 service 目标是图表、视频、模型或 HUD | 是 |
| `entity.ScriptModule` | `TargetID` | `script_modules.target_id` | `chart_h`、`chart_l`、`vs_cam_65` | 目标资源 ID；视频模块时作为 `video_id` 查询 | 是 |
| `entity.ScriptModule` | `SortOrder` | `script_modules.sort_order` | `1`、`2`、`3` | Repository 按 `sort_order ASC, id ASC` 排序 | 是 |
| `entity.ScriptModule` | `Enabled` | `script_modules.enabled` | `1` | Repository 只查询启用模块 | 是 |
| `entity.ScriptModule` | `ConfigJSON` | `script_modules.config_json` | `{"position":"left_top"}` | 预留展示配置 | 当前 service 未使用 |
| `entity.DataBinding` | `DataKey` | `data_bindings.data_key` | `security.access.today`、`security.fire.status` | 将业务 `data_key` 关联到运行时绑定/图表 ID | 是 |
| `entity.DataBinding` | `BindingID` | `data_bindings.binding_id` | `security.access.today`、`security.fire.status` | 进入 `gateway.ChartData.BindingID`，再进入 `ue.ChartDataPush.BindingID` | 是 |
| `entity.DataBinding` | `ChartID` | `data_bindings.chart_id` | `chart_h`、`chart_l` | 进入 `gateway.ChartData.ChartID`，FastAPI 可据此定位图表容器 | 是 |
| `entity.DataBinding` | `ModuleID` | `data_bindings.module_id` | `today_access_card`、`fire_alert_card` | 可选模块反向引用 | 间接 |
| `entity.DataBinding` | `Enabled` | `data_bindings.enabled` | `1` | Repository 只查询启用绑定 | 是 |
| `entity.VideoSource` | `VideoID` | `video_sources.video_id` | `vs_cam_65` | 进入 `gateway.VideoSource.VideoID`，再进入 `ue.VideoStream.StreamID` | 是 |
| `entity.VideoSource` | `Name` | `video_sources.name` | 校门口监控名称 | 进入 `gateway.VideoSource.Title`，再进入 `ue.VideoStream.Name` | 是 |
| `entity.VideoSource` | `PlayURL` | `video_sources.play_url` | `https://media.example.com/demo/player?src=demo_camera` | 进入 `gateway.VideoSource.PlayURL`，再进入 `webrtc_player_url` | 是 |
| `entity.VideoSource` | `SourceType` | `video_sources.source_type` | `mock` | 描述视频源类型 | 当前 mapper 未使用 |
| `entity.VideoSource` | `BindModelID` | `video_sources.bind_model_id` | `gate_model_01` | 网关侧保留：视频源与 UE 模型/点位绑定 | 保留在 DataBundle |
| `entity.VideoSource` | `Enabled` | `video_sources.enabled` | `1` | Repository 只查询启用视频源 | 是 |
| `entity.ModelBinding` | `BindModelID` | `model_bindings.bind_model_id` | `gate_model_01` | 未来模型/点位绑定键 | 当前主链路未使用 |
| `entity.ModelBinding` | `ObjectType` | `model_bindings.object_type` | `gate` | 未来外部对象类型 | 未使用 |
| `entity.ModelBinding` | `ObjectID` | `model_bindings.object_id` | `gate_01` | 未来外部对象 ID | 未使用 |
| `entity.ModelBinding` | `DataKeysJSON` | `model_bindings.data_keys_json` | `["security.access.today","security.video.gate_01"]` | 未来模型与数据关系 | 未使用 |
| `entity.ChartTemplate` | `TemplateID` | `chart_templates.template_id` | `stat_card_basic`、`status_basic`、`video_basic` | 未来图表/模板配置查询 | 当前 service 未使用 |
| `entity.ChartTemplate` | `Name` | `chart_templates.name` | 基础卡片/状态/视频模板名称 | 模板人类可读名称 | 未使用 |
| `entity.ChartTemplate` | `Type` | `chart_templates.type` | `stat_card`、`status`、`video` | 模板类别 | 未使用 |
| `entity.ChartTemplate` | `ConfigJSON` | `chart_templates.config_json` | 宽高 JSON | 未来展示配置 | 未使用 |

关键点：

- `scripts.script_id` 标识整个脚本，例如 `security_main`。
- `script_modules.module_id` 标识脚本中的一个展示模块。
- `script_modules.data_key` 是业务语义键，用于生成模块/图表数据。
- `script_modules.target_kind` 和 `script_modules.target_id` 将模块映射到图表、视频、模型或 HUD 目标。
- `data_bindings.binding_id` 是 `chart_data_push` 使用的运行时数据绑定 ID。
- `data_bindings.chart_id` 是 FastAPI/UE 图表容器 ID。
- `video_sources.video_id` 是视频流身份，会映射为 `video_streams_set.stream_id`。
- `video_sources.play_url` 会映射为 FastAPI/UE 使用的 `webrtc_player_url`。
- `video_sources.bind_model_id` 当前作为网关上下文保留，后续可用于把视频源关联到 UE 模型/点位。

## 3. Domain 业务对象取值来源

默认 Mock 模式下，security 相关 domain 对象由 `internal/adapter/security/mock_adapter.go` 生成。

`http_adapter.go` 已实现，可通过 `adapters.security.mode: http` 和对应的地址、路径及鉴权配置接入外部系统。Service 仍然消费 domain 对象，而不直接依赖外部系统原始字段。Adapter 负责把外部字段转换成 domain 结构体。下表以默认 Mock 数据为例。

| Domain 字段 | 当前 Mock 值 | 业务含义 | 后续真实来源 | 进入的 Gateway 字段 |
|---|---|---|---|---|
| `AccessSummary.InCount` | `1800` | 今日进入人数 | 门禁系统 Adapter | `ChartData.Rows` |
| `AccessSummary.OutCount` | `1480` | 今日离开人数 | 门禁系统 Adapter | `ChartData.Rows` |
| `AccessSummary.TotalCount` | `3280` | 今日总人数/总人次 | 门禁系统 Adapter | `ModuleData.Value`、`ChartData.Rows` |
| `AccessSummary.Timestamp` | `time.Now().Format(time.RFC3339)` | 快照时间 | 门禁系统 Adapter | `ModuleData.Timestamp`、通过 bundle 时间进入 `ChartDataPush.Timestamp` |
| `FireAlert.AlertID` | `fire_mock_001` | 告警 ID | 消防系统 Adapter | `ModuleData.Payload` |
| `FireAlert.Level` | `normal` | 消防告警等级/状态 | 消防系统 Adapter | `ModuleData.Value`、`ModuleData.Payload`、`ChartData.Rows` |
| `FireAlert.Location` | 校门口位置 | 告警位置 | 消防系统 Adapter | `ModuleData.Payload`、`ChartData.Rows` |
| `FireAlert.Message` | 消防状态正常提示 | 告警文案 | 消防系统 Adapter | `ModuleData.Payload`、`ChartData.Rows` |
| `FireAlert.Timestamp` | `time.Now().Format(time.RFC3339)` | 告警时间 | 消防系统 Adapter | `ModuleData.Payload` |

`buildModuleData` 中的具体映射：

- `AccessSummary.TotalCount -> gateway.ModuleData.Value`
- `AccessSummary.Timestamp -> gateway.ModuleData.Timestamp`
- `AccessSummary.InCount/OutCount/TotalCount -> gateway.ChartData.Rows`
- `FireAlert.Level -> gateway.ModuleData.Value`
- `[]FireAlert -> gateway.ModuleData.Payload`
- `FireAlert.Level/Location/Message -> gateway.ChartData.Rows`

其他 domain 结构体，例如 `EducationSnapshot`、`EnergySnapshot`、`LabSnapshot`、`VideoStatus` 已经为后续业务域预留。当前 `security_main` 最小链路主要使用 `AccessSummary`、`FireAlert` 和 `VideoStatus`。

## 4. `gateway.DataBundle` 组装链路

真实代码路径位于 `internal/service/script_service.go`。

### `BuildDataBundle(ctx, scriptID, useCache)`

当前实际顺序：

1. 如果 `useCache == true` 且 `scriptCache != nil`，调用 `scriptCache.GetPreloadBundle(ctx, scriptID)`。
   - 缓存异常时记录 Warn 并继续。
   - 缓存命中时直接返回缓存的 `gateway.DataBundle`。
2. 调用 `scriptRepo.FindByScriptID(ctx, scriptID)`。
   - 读取 `scripts`。
   - 如果脚本不存在或数据库异常，返回 error。
3. 检查 `script.Status`。
   - 如果不是 `enabled`，返回 `script disabled`。
4. 调用 `scriptModuleRepo.ListByScriptID(ctx, scriptID)`。
   - 读取启用的 `script_modules`。
   - 按 `sort_order ASC, id ASC` 排序。
5. 调用 `collectDataKeys(modules)`。
   - 收集非空且去重后的 `ScriptModule.DataKey`。
6. 调用 `dataBindingRepo.ListByDataKeys(ctx, dataKeys)`。
   - 批量读取所有 data_key 对应的启用 `data_bindings`。
7. 构建 `bindingMap[dataKey] = entity.DataBinding`。
8. 创建新的 `gateway.DataBundle`。
   - `BundleID`：service 生成，格式为 `bundle_<UnixNano>`。
   - `ScriptID`：来自 `entity.Script.ScriptID`。
   - `SceneID`：来自 `entity.Script.SceneID`。
   - `Version`：service 生成，当前为 `"v1"`。
   - `GeneratedAt`：service 生成，格式为 `time.Now().Format(time.RFC3339)`。
   - `TTLSeconds`：service 生成，当前为 `30`。
   - `Modules`、`Charts`、`Videos`、`ModelBindings`：初始化为空切片。
9. 遍历每个 `entity.ScriptModule`，调用 `buildModuleData(ctx, module, bindingMap)`。
   - `ScriptModule + DataBinding + domain 数据 -> gateway.ModuleData + gateway.ChartData`。
   - 如果某个模块构建失败，记录 Warn 并继续处理下一个模块。
10. 非 nil 的 `ModuleData` 追加到 `bundle.Modules`。
11. 非 nil 的 `ChartData` 追加到 `bundle.Charts`。
12. 如果 `moduleCache != nil`，写入模块数据 `moduleCache.SetModuleData`。
   - 模块缓存写入失败只记录 Warn，不导致 DataBundle 失败。
13. 如果 `module.TargetKind == "video"` 且 `module.TargetID != ""`，调用 `videoSourceRepo.FindByVideoID(ctx, module.TargetID)`。
   - 根据 `video_id = module.TargetID` 读取 `video_sources`。
14. 决定视频状态。
   - 默认状态为 `"online"`。
   - 如果 `videoCache` 中已有 `domain.VideoStatus`，使用缓存状态。
   - 如果无缓存状态，则向 Redis 写入默认 online 状态。
15. 追加 `gateway.VideoSource` 到 `bundle.Videos`。
   - `VideoID`：来自 `entity.VideoSource.VideoID`
   - `Title`：来自 `entity.VideoSource.Name`
   - `PlayURL`：来自 `entity.VideoSource.PlayURL`
   - `Status`：来自缓存或默认值
   - `BindModelID`：来自 `entity.VideoSource.BindModelID`
16. 返回 `gateway.DataBundle`。

### `buildModuleData(ctx, module, bindingMap)`

基础 `gateway.ModuleData` 字段来自 `entity.ScriptModule`：

- `ModuleID <- module.ModuleID`
- `Type <- module.Type`
- `Title <- module.Title`
- `DataKey <- module.DataKey`
- `TargetKind <- module.TargetKind`
- `TargetID <- module.TargetID`
- `Timestamp <- service 生成当前时间`，除非被 domain 时间覆盖

对于 `security.access.today`：

- 调用 `securityAdapter.GetAccessSummary(ctx)`。
- `ModuleData.Value <- AccessSummary.TotalCount`
- `ModuleData.Unit <- "人次"`
- `ModuleData.Timestamp <- AccessSummary.Timestamp`
- 创建 `ChartData.Rows`，包含进入、离开、总数等指标行。

对于 `security.fire.status`：

- 调用 `securityAdapter.GetFireAlerts(ctx)`。
- `ModuleData.Value <- "normal"`，除非第一条告警有其他等级。
- `ModuleData.Payload <- []domain.FireAlert`
- 通过告警等级、位置、信息创建 `ChartData.Rows`。

对于 `security.video.*`：

- `ModuleData.Value <- "online"`。
- 视频元数据后续由 `BuildDataBundle` 中的视频分支追加。

对于图表数据：

- 如果 `chartData != nil` 且 `module.DataKey` 存在对应 `DataBinding`：
  - `ChartData.BindingID <- DataBinding.BindingID`
  - `ChartData.ChartID <- DataBinding.ChartID`

总结：

```text
Script + ScriptModule + DataBinding + Domain Data
  -> ModuleData + ChartData

VideoSource entity + videoCache
  -> gateway.VideoSource
```

## 5. `ChartData` 字段：`x_axis`、`series`、`rows`

`gateway.ChartData` 有三种展示数据表达方式：

```go
XAxis  []string      `json:"x_axis,omitempty"`
Series []ChartSeries `json:"series,omitempty"`
Rows   [][]string    `json:"rows,omitempty"`
```

### 5.1 `x_axis`

`x_axis` 表示横轴数据。

适合坐标轴类图表，例如：

- 折线图
- 柱状图
- 趋势图

示例：

```json
["08:00", "09:00", "10:00"]
```

### 5.2 `series`

`series` 表示图表数据序列。通常与 `x_axis` 一一对应。

适合：

- 折线图
- 柱状图
- 多系列统计图

示例：

```json
[
  {"name":"进入人数","unit":"人次","data":[120,180,260]},
  {"name":"离开人数","unit":"人次","data":[90,140,200]}
]
```

### 5.3 `rows`

`rows` 表示二维表格、列表、指标卡数据。

当前 V9 的 `chart_data_push` 主要使用 `rows`。

适合：

- 指标卡
- 告警列表
- 状态面板
- 表格面板

示例：

```json
[
  ["指标", "数值", "单位"],
  ["今日进入", "1800", "人次"],
  ["今日离开", "1480", "人次"],
  ["今日总数", "3280", "人次"]
]
```

经验规则：

```text
x_axis + series 用于坐标轴类图表。
rows 用于表格、指标卡、状态卡和告警列表。
当前 security_main 主要使用 rows。
x_axis/series 为后续趋势图和统计图预留。
```

Mapper 行为：

- `Rows` 会固定复制到 `ue.ChartDataPush.Rows`。
- 如果 `Series` 或 `XAxis` 非空，mapper 还会设置 `ue.ChartDataPush.Payload = chart`，使 FastAPI 可以收到完整的 `gateway.ChartData`。

## 6. `VideoSource` 字段与转换链路

`gateway.VideoSource` 不是视频流本身，而是面向 FastAPI/UE 的视频源/视频窗口元数据。

| Gateway 字段 | 来源 | Mapper 目标 | 含义 |
|---|---|---|---|
| `video_id` | `video_sources.video_id` | `ue.VideoStream.StreamID` | 视频流身份，例如 `vs_cam_65` |
| `title` | `video_sources.name` | `ue.VideoStream.Name` | 展示名称 |
| `play_url` | `video_sources.play_url` | `ue.VideoStream.WebRTCPlayerURL` | 播放器 URL |
| `status` | `videoCache` 或默认 `"online"` | 当前 `video_streams_set` 未下发 | 网关侧运行时状态 |
| `bind_model_id` | `video_sources.bind_model_id` | 当前 `video_streams_set` 未下发 | 网关侧 UE 模型/点位绑定，例如 `gate_model_01` |

当前转换：

```text
gateway.VideoSource{
  VideoID: "vs_cam_65",
  Title: "校门口监控",
  PlayURL: "https://media.example.com/demo/player?src=demo_camera",
  Status: "online",
  BindModelID: "gate_model_01",
}

->

ue.VideoStreams{
  Streams: [
    {
      StreamID: "vs_cam_65",
      Name: "校门口监控",
      WebRTCPlayerURL: "https://media.example.com/demo/player?src=demo_camera"
    }
  ]
}

->

ue.WSMessage{
  Type: "video_streams_set",
  Payload: ue.VideoStreams
}
```

当前 `video_streams_set` 主要发送 `video_id/title/play_url`。

`status` 和 `bind_model_id` 作为 gateway 层上下文保留。如果后续 FastAPI/UE 协议支持，可以扩展到下游 payload。

Go 网关不负责拉流、转码或大规模视频分发，只管理视频源元数据和播放地址。

## 7. DTO 返回链路

HTTP 路由：

```http
GET /api/scripts/:id/data-bundle
```

结构体流转：

```text
handler.GetDataBundle
  -> dto.GetDataBundleURI
  -> dto.GetDataBundleQuery
  -> service.BuildDataBundle
  -> gateway.DataBundle
  -> dto.APIResponse[gateway.DataBundle]
  -> HTTP JSON
```

细节：

- `dto.GetDataBundleURI` 将 `:id` 绑定到 `ScriptID`。
- `dto.GetDataBundleQuery` 绑定 `use_cache`。
- DTO 不承载业务逻辑。
- DTO 只表示 HTTP 输入/输出。
- `gateway.DataBundle` 是 `data` 字段里的主体值。
- `code/message/data` 是 API 响应外壳。

简化响应：

```json
{
  "code": "OK",
  "data": {
    "script_id": "security_main",
    "scene_id": "campus_security",
    "modules": [],
    "charts": [],
    "videos": []
  }
}
```

失败响应使用 `dto.Fail(code, message)`，返回外壳：

```json
{
  "code": "BUILD_DATA_BUNDLE_FAILED",
  "message": "..."
}
```

## 8. WebSocket 推送链路

HTTP 路由：

```http
POST /api/scripts/:id/preload
```

结构体流转：

```text
handler.Preload
  -> dto.GetDataBundleURI
  -> dto.PreloadRequest
  -> service.Preload
  -> gateway.DataBundle
  -> scriptCache.SetPreloadBundle
  -> mapper.BundleToWSMessages
  -> []ue.WSMessage
  -> FastAPIClient.PushMessages
  -> FastAPI /ws_data
```

规则：

- `GET /api/scripts/:id/data-bundle` 不触发 WebSocket 推送。
- 只有 `POST /api/scripts/:id/preload` 触发 WebSocket 推送。
- WebSocket 推送失败只记录 Warn，不导致 HTTP preload 失败。
- 如果 `WSDataURL` 为空，`fastAPIClient` 为 nil，跳过推送。
- Go 网关只推送到 FastAPI `/ws_data`，不直接控制 UE。

`handler.Preload` 返回 `dto.PreloadResponse`，不是完整 bundle：

```json
{
  "code": "OK",
  "data": {
    "task_id": "bundle_...",
    "script_id": "security_main",
    "status": "done"
  }
}
```

## 9. Mapper 转换规则

真实 mapper 位于 `internal/service/mapper/ue_mapper.go`。

### 9.1 `ChartData -> chart_data_push`

转换关系：

```text
gateway.ChartData.BindingID -> ue.ChartDataPush.BindingID
gateway.ChartData.Rows      -> ue.ChartDataPush.Rows
bundle.GeneratedAt          -> ue.ChartDataPush.Timestamp
```

当前 mapper 不映射 `gateway.ChartData.Value`，因为 `ChartData` 没有 `Value` 字段。

当前 mapper 对 `x_axis` 和 `series` 的行为：

- 如果 `chart.Series` 或 `chart.XAxis` 非空，则 `ue.ChartDataPush.Payload = chart`。
- 这意味着 FastAPI 可以在 `payload` 中收到完整 `gateway.ChartData`。
- `Rows` 仍然会复制到 `ue.ChartDataPush.Rows`。

没有 `BindingID` 的 ChartData 会被跳过，不会生成 `chart_data_push` 消息。

### 9.2 `VideoSource -> video_streams_set`

转换关系：

```text
gateway.VideoSource.VideoID -> ue.VideoStream.StreamID
gateway.VideoSource.Title   -> ue.VideoStream.Name
gateway.VideoSource.PlayURL -> ue.VideoStream.WebRTCPlayerURL
[]VideoStream               -> ue.VideoStreams
ue.VideoStreams             -> ue.WSMessage{Type:"video_streams_set"}
```

`gateway.VideoSource.Status` 和 `gateway.VideoSource.BindModelID` 当前不会进入 `ue.VideoStream`。

### 9.3 `seq`

`seq` 是 WebSocket 消息序号。

- 当前调用使用 `BundleToWSMessages(bundle, 0)`。
- 消息序号从 `0` 开始。
- 消息按生成顺序递增。
- 图表消息先生成。
- 如果存在视频，则在图表消息之后追加一条 `video_streams_set` 消息。

典型 `security_main` 输出：

```text
seq=0 chart_data_push security.access.today
seq=1 chart_data_push security.fire.status
seq=2 video_streams_set vs_cam_65
```

## 10. Binding 链路

核心链路：

```text
script_id
  -> module_id
  -> data_key
  -> binding_id
  -> chart_id / stream_id / model_id
```

含义：

- `script_id`：一个汇报/展示脚本，例如 `security_main`。
- `module_id`：脚本里的一个展示模块，例如 `today_access_card`。
- `data_key`：业务语义键，例如 `security.access.today`。
- `binding_id`：FastAPI/UE 运行时数据绑定 ID，例如 `security.access.today`。
- `chart_id`：FastAPI/UE 图表容器 ID，例如 `chart_h`。
- `target_kind`：目标类型，例如 `chart`、`video`、`model`、`hud`。
- `target_id`：目标资源 ID。根据 `target_kind` 不同，可以表示 `chart_id`、`video_id` 或 `model_id`。

图表示例：

```text
security_main
  -> today_access_card
  -> security.access.today
  -> security.access.today
  -> chart_h
```

消防状态图表示例：

```text
security_main
  -> fire_alert_card
  -> security.fire.status
  -> security.fire.status
  -> chart_l
```

视频示例：

```text
security_main
  -> gate_video_panel
  -> security.video.gate_01
  -> target_kind=video
  -> target_id=vs_cam_65
  -> video_streams_set.stream_id=vs_cam_65
```

重要区别：

- 图表模块通过 `data_bindings` 获取 `binding_id` 和 `chart_id`。
- 视频模块使用 `script_modules.target_id` 作为 `video_sources.video_id`，再由 mapper 转为 `video_streams_set.stream_id`。
- `bind_model_id` 从 `video_sources.bind_model_id` 读取，并保留在 `gateway.VideoSource` 中。

## 11. 当前结构与组件的使用范围

以下区分尚未接入脚本数据包的扩展结构与已经实现的独立能力：

| 项目 | 当前状态 | 用途与边界 |
|---|---|---|
| `gateway.DataBundle.ModelBindings` | 初始化为空切片 | 后续 UE 模型/点位绑定 |
| `entity.ModelBinding` / `model_bindings` | seed 存在，但 service 尚未读取 | 后续模型/点位与数据绑定 |
| `entity.ChartTemplate` / `chart_templates` | seed 存在，但 service 尚未读取 | 后续可配置图表/卡片/表格模板 |
| `gateway.ChartData.XAxis` / `Series` | 已定义，mapper 可以通过 `Payload` 透传，但当前 `security_main` 主要使用 `Rows` | 后续趋势图/统计图 |
| `cache.AuthCache` | 已用于登录 Session 的存储、校验与注销 | 与业务缓存共用可配置的 Redis 项目前缀 |
| `scheduler.PreloadJob` | 已实现；由配置决定是否在 `App.Run` 中启动 | 定时强制预热，可选推送；不负责视频在线探测 |
| `security.http_adapter.go` | 已实现；`NewApp` 按配置选择 Mock 或 HTTP Adapter | 外部安防系统的字段归一化与鉴权 |
| `protocol/ue` 的模型/路径/渲染/汇报结构 | 已被 UE 配置接口使用 | 配置持久化与协议兼容；不依赖 `DataBundle.ModelBindings` |

`ModelBindings`、`ChartTemplate` 等扩展结构尚未接入当前脚本数据组装链路。模型摆放与显隐已由独立 UE 配置接口提供，不应与数据包中的预留绑定字段混淆。

扩展与配置示例：

- 数据包后续可通过 `ModelBindings` 携带模型与业务数据的绑定关系。
- 趋势图可以使用 `x_axis` 和 `series`。
- 用户登录使用已有的 Session 鉴权，公开种子不提供默认账号。
- `scheduler.preload.enabled` 可启用定时预热。
- `adapters.security.mode: http` 可选择已有的 HTTP Adapter。

## 12. 最终总结

当前结构体流转已经稳定：

```text
entity 提供本地 MySQL 配置骨架。
domain 提供 Adapter 归一化后的内部业务数据。
service 组合 entity 配置、domain 数据、cache 状态和生成型元数据。
gateway.DataBundle 是统一展示数据包。
dto 是 HTTP 请求/响应外壳。
mapper 将 gateway 结构转换为 UE/FastAPI 协议结构。
FastAPIClient 将 WSMessage 发送到 FastAPI /ws_data。
```

当前边界：

- Repository 只读取本地 MySQL 配置。
- Cache 只读写 Redis 运行时缓存。
- Adapter 从 Mock 或已配置的外部 HTTP 系统生成 domain 对象。
- Service 组装 `gateway.DataBundle` 并协调 preload。
- Handler 只绑定 HTTP 入参、调用 service、返回 DTO 响应。
- Mapper 只做 `gateway -> protocol/ue` 转换。
- Go 网关永远不直接控制 UE。
