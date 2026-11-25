# Data Flow and Struct Contracts

中文版：[数据流转与结构体契约](data-flow.md)。

This document describes how the current gateway moves data across structs, layers, HTTP DTOs, and FastAPI/UE WebSocket messages.

It is based on the current code in:

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

This document is descriptive only. It does not change the HTTP contract, the `gateway.DataBundle` shape, mapper output, migrations, or business behavior.

## 1. Overall Struct Flow

Current text flow:

```text
external raw / mock
  -> domain.AccessSummary / domain.FireAlert
  + entity.Script / ScriptModule / DataBinding / VideoSource
  -> gateway.ModuleData / ChartData / VideoSource
  -> gateway.DataBundle
  -> dto.APIResponse[gateway.DataBundle]
  -> ue.ChartDataPush / ue.VideoStreams / ue.WSMessage
```

Layer meanings:

- `domain` structs are internal business objects after external data enters the gateway. They are not MySQL tables and are not UE protocol structs.
- `entity` structs are the local MySQL configuration skeleton. They do not store realtime business values such as today's visitor count or fire alert state.
- `gateway.DataBundle` is the unified display data package produced by the Go gateway for FastAPI.
- `dto` structs are HTTP request/response shells only.
- `protocol/ue` structs are compatibility messages for the existing FastAPI/UE protocol.
- `mapper` only converts `gateway` structs to `ue` protocol messages.

## 2. Entity Configuration Value Sources

Entities come from local MySQL tables defined in `migrations/001_init_schema.sql`. The current `security_main` seed examples come from `migrations/002_seed_mock_data.sql`.

| Struct | Field | DB table field | Current seed example | Purpose | In current V9 main path |
|---|---|---|---|---|---|
| `entity.Script` | `ScriptID` | `scripts.script_id` | `security_main` | Selects one report/display script | Yes |
| `entity.Script` | `Name` | `scripts.name` | security report script name | Human-readable script name | Indirect |
| `entity.Script` | `SceneID` | `scripts.scene_id` | `campus_security` | Fills `gateway.DataBundle.SceneID` | Yes |
| `entity.Script` | `Status` | `scripts.status` | `enabled` | Disabled scripts are rejected by service | Yes |
| `entity.ScriptModule` | `ModuleID` | `script_modules.module_id` | `today_access_card`, `fire_alert_card`, `gate_video_panel` | Display module identity inside a script | Yes |
| `entity.ScriptModule` | `ScriptID` | `script_modules.script_id` | `security_main` | Groups modules under a script | Yes |
| `entity.ScriptModule` | `Type` | `script_modules.type` | `stat_card`, `alert_status`, `video` | Fills `gateway.ModuleData.Type` | Yes |
| `entity.ScriptModule` | `Title` | `script_modules.title` | today access/fire alert/gate video titles | Fills module/chart title | Yes |
| `entity.ScriptModule` | `DataKey` | `script_modules.data_key` | `security.access.today`, `security.fire.status`, `security.video.gate_01` | Business semantic key used by service switch and binding lookup | Yes |
| `entity.ScriptModule` | `Template` | `script_modules.template` | `stat_card_basic`, `status_basic`, `video_basic` | Fills `gateway.ChartData.Template` when chart data exists | Yes for charts |
| `entity.ScriptModule` | `TargetKind` | `script_modules.target_kind` | `chart`, `video` | Tells service whether target is chart/video/model/hud | Yes |
| `entity.ScriptModule` | `TargetID` | `script_modules.target_id` | `chart_h`, `chart_l`, `vs_cam_65` | Target resource ID; for video it is used as `video_id` lookup | Yes |
| `entity.ScriptModule` | `SortOrder` | `script_modules.sort_order` | `1`, `2`, `3` | Repository orders modules by `sort_order ASC, id ASC` | Yes |
| `entity.ScriptModule` | `Enabled` | `script_modules.enabled` | `1` | Repository filters enabled modules | Yes |
| `entity.ScriptModule` | `ConfigJSON` | `script_modules.config_json` | `{"position":"left_top"}` | Reserved display config | Not used by current service |
| `entity.DataBinding` | `DataKey` | `data_bindings.data_key` | `security.access.today`, `security.fire.status` | Joins business `data_key` to runtime binding/chart IDs | Yes |
| `entity.DataBinding` | `BindingID` | `data_bindings.binding_id` | `security.access.today`, `security.fire.status` | Becomes `gateway.ChartData.BindingID`, then `ue.ChartDataPush.BindingID` | Yes |
| `entity.DataBinding` | `ChartID` | `data_bindings.chart_id` | `chart_h`, `chart_l` | Becomes `gateway.ChartData.ChartID`; FastAPI can map binding to chart container | Yes |
| `entity.DataBinding` | `ModuleID` | `data_bindings.module_id` | `today_access_card`, `fire_alert_card` | Optional back-reference to module | Indirect |
| `entity.DataBinding` | `Enabled` | `data_bindings.enabled` | `1` | Repository filters enabled bindings | Yes |
| `entity.VideoSource` | `VideoID` | `video_sources.video_id` | `vs_cam_65` | Becomes `gateway.VideoSource.VideoID`, then `ue.VideoStream.StreamID` | Yes |
| `entity.VideoSource` | `Name` | `video_sources.name` | gate video title | Becomes `gateway.VideoSource.Title`, then `ue.VideoStream.Name` | Yes |
| `entity.VideoSource` | `PlayURL` | `video_sources.play_url` | `https://media.example.com/demo/player?src=demo_camera` | Becomes `gateway.VideoSource.PlayURL`, then `webrtc_player_url` | Yes |
| `entity.VideoSource` | `SourceType` | `video_sources.source_type` | `mock` | Describes source kind | Not used by current mapper |
| `entity.VideoSource` | `BindModelID` | `video_sources.bind_model_id` | `gate_model_01` | Gateway-side context for binding video source to UE model/point | Preserved in DataBundle |
| `entity.VideoSource` | `Enabled` | `video_sources.enabled` | `1` | Repository filters enabled videos | Yes |
| `entity.ModelBinding` | `BindModelID` | `model_bindings.bind_model_id` | `gate_model_01` | Future model/point binding key | Not used in current main path |
| `entity.ModelBinding` | `ObjectType` | `model_bindings.object_type` | `gate` | Future external object type | Not used |
| `entity.ModelBinding` | `ObjectID` | `model_bindings.object_id` | `gate_01` | Future external object ID | Not used |
| `entity.ModelBinding` | `DataKeysJSON` | `model_bindings.data_keys_json` | `["security.access.today","security.video.gate_01"]` | Future model-to-data relation | Not used |
| `entity.ChartTemplate` | `TemplateID` | `chart_templates.template_id` | `stat_card_basic`, `status_basic`, `video_basic` | Future chart/template config lookup | Not used in current service |
| `entity.ChartTemplate` | `Name` | `chart_templates.name` | basic card/status/video names | Human-readable template name | Not used |
| `entity.ChartTemplate` | `Type` | `chart_templates.type` | `stat_card`, `status`, `video` | Template category | Not used |
| `entity.ChartTemplate` | `ConfigJSON` | `chart_templates.config_json` | width/height JSON | Future display config | Not used |

Key points:

- `scripts.script_id` identifies the whole script, for example `security_main`.
- `script_modules.module_id` identifies each display module in that script.
- `script_modules.data_key` is the semantic business key used to build module/chart data.
- `script_modules.target_kind` and `script_modules.target_id` map a module to a chart/video/model/hud target.
- `data_bindings.binding_id` is the runtime data binding ID for `chart_data_push`.
- `data_bindings.chart_id` is the FastAPI/UE chart container ID.
- `video_sources.video_id` is the video stream identity used by `video_streams_set.stream_id`.
- `video_sources.play_url` is the URL passed to FastAPI/UE as `webrtc_player_url`.
- `video_sources.bind_model_id` is currently preserved in gateway context and can later connect a video source to a UE model/point.

## 3. Domain Business Object Value Sources

In the default mock mode, security domain objects are generated by `internal/adapter/security/mock_adapter.go`.

`http_adapter.go` is implemented and can be selected with `adapters.security.mode: http` and the corresponding URL, path, and authentication settings. The service consumes domain objects rather than external raw fields. The adapter maps external fields into domain structs. The following table describes the default mock data.

| Domain field | Current mock value | Business meaning | Future real source | Gateway destination |
|---|---|---|---|---|
| `AccessSummary.InCount` | `1800` | Today's entries | Access-control system adapter | `ChartData.Rows` |
| `AccessSummary.OutCount` | `1480` | Today's exits | Access-control system adapter | `ChartData.Rows` |
| `AccessSummary.TotalCount` | `3280` | Today's total count | Access-control system adapter | `ModuleData.Value`, `ChartData.Rows` |
| `AccessSummary.Timestamp` | `time.Now().Format(time.RFC3339)` | Snapshot time | Access-control system adapter | `ModuleData.Timestamp`, `ChartDataPush.Timestamp` through bundle time |
| `FireAlert.AlertID` | `fire_mock_001` | Alert identity | Fire system adapter | `ModuleData.Payload` |
| `FireAlert.Level` | `normal` | Fire alert severity/state | Fire system adapter | `ModuleData.Value`, `ModuleData.Payload`, `ChartData.Rows` |
| `FireAlert.Location` | gate location | Alert location | Fire system adapter | `ModuleData.Payload`, `ChartData.Rows` |
| `FireAlert.Message` | fire status normal message | Alert text | Fire system adapter | `ModuleData.Payload`, `ChartData.Rows` |
| `FireAlert.Timestamp` | `time.Now().Format(time.RFC3339)` | Alert time | Fire system adapter | `ModuleData.Payload` |

Concrete mappings in `buildModuleData`:

- `AccessSummary.TotalCount -> gateway.ModuleData.Value`
- `AccessSummary.Timestamp -> gateway.ModuleData.Timestamp`
- `AccessSummary.InCount/OutCount/TotalCount -> gateway.ChartData.Rows`
- `FireAlert.Level -> gateway.ModuleData.Value`
- `[]FireAlert -> gateway.ModuleData.Payload`
- `FireAlert.Level/Location/Message -> gateway.ChartData.Rows`

Other domain structs such as `EducationSnapshot`, `EnergySnapshot`, `LabSnapshot`, and `VideoStatus` already exist for future domains. The current `security_main` minimum path uses `AccessSummary`, `FireAlert`, and `VideoStatus`.

## 4. `gateway.DataBundle` Assembly Chain

The actual code path is in `internal/service/script_service.go`.

### `BuildDataBundle(ctx, scriptID, useCache)`

Actual order:

1. If `useCache == true` and `scriptCache != nil`, call `scriptCache.GetPreloadBundle(ctx, scriptID)`.
   - On cache error, log Warn and continue.
   - On cache hit, return the cached `gateway.DataBundle`.
2. Call `scriptRepo.FindByScriptID(ctx, scriptID)`.
   - Reads `scripts`.
   - If missing or DB error, return error.
3. Check `script.Status`.
   - If not `enabled`, return `script disabled`.
4. Call `scriptModuleRepo.ListByScriptID(ctx, scriptID)`.
   - Reads enabled `script_modules`.
   - Orders by `sort_order ASC, id ASC`.
5. Call `collectDataKeys(modules)`.
   - Collects non-empty unique `ScriptModule.DataKey` values.
6. Call `dataBindingRepo.ListByDataKeys(ctx, dataKeys)`.
   - Reads enabled `data_bindings` for all collected data keys.
7. Build `bindingMap[dataKey] = entity.DataBinding`.
8. Create a new `gateway.DataBundle`.
   - `BundleID`: service-generated `bundle_<UnixNano>`.
   - `ScriptID`: from `entity.Script.ScriptID`.
   - `SceneID`: from `entity.Script.SceneID`.
   - `Version`: service-generated `"v1"`.
   - `GeneratedAt`: service-generated `time.Now().Format(time.RFC3339)`.
   - `TTLSeconds`: service-generated `30`.
   - `Modules`, `Charts`, `Videos`, `ModelBindings`: initialized slices.
9. For each `entity.ScriptModule`, call `buildModuleData(ctx, module, bindingMap)`.
   - `ScriptModule + DataBinding + domain data -> gateway.ModuleData + gateway.ChartData`.
   - On module build error, log Warn and continue to the next module.
10. Append non-nil `ModuleData` to `bundle.Modules`.
11. Append non-nil `ChartData` to `bundle.Charts`.
12. If `moduleCache != nil`, write module data with `moduleCache.SetModuleData`.
   - Cache write failure logs Warn and does not fail the bundle.
13. If `module.TargetKind == "video"` and `module.TargetID != ""`, call `videoSourceRepo.FindByVideoID(ctx, module.TargetID)`.
   - Reads `video_sources` by `video_id = module.TargetID`.
14. Determine video status.
   - Default is `"online"`.
   - If `videoCache` has a cached `domain.VideoStatus`, use its `Status`.
   - If no cached status, write a default online status to Redis.
15. Append `gateway.VideoSource` to `bundle.Videos`.
   - `VideoID`: `entity.VideoSource.VideoID`
   - `Title`: `entity.VideoSource.Name`
   - `PlayURL`: `entity.VideoSource.PlayURL`
   - `Status`: cache/default status
   - `BindModelID`: `entity.VideoSource.BindModelID`
16. Return `gateway.DataBundle`.

### `buildModuleData(ctx, module, bindingMap)`

Base `gateway.ModuleData` fields come from `entity.ScriptModule`:

- `ModuleID <- module.ModuleID`
- `Type <- module.Type`
- `Title <- module.Title`
- `DataKey <- module.DataKey`
- `TargetKind <- module.TargetKind`
- `TargetID <- module.TargetID`
- `Timestamp <- service-generated current time`, unless overwritten by domain timestamp

For `security.access.today`:

- Calls `securityAdapter.GetAccessSummary(ctx)`.
- `ModuleData.Value <- AccessSummary.TotalCount`
- `ModuleData.Unit <- "people/times"` in current code this is the Chinese unit string.
- `ModuleData.Timestamp <- AccessSummary.Timestamp`
- Creates `ChartData.Rows` with metric rows for in/out/total counts.

For `security.fire.status`:

- Calls `securityAdapter.GetFireAlerts(ctx)`.
- `ModuleData.Value <- "normal"` unless first alert has another level.
- `ModuleData.Payload <- []domain.FireAlert`
- Creates `ChartData.Rows` from alert level/location/message.

For `security.video.*`:

- `ModuleData.Value <- "online"`.
- Video metadata is appended later by the video branch in `BuildDataBundle`.

For chart data:

- If `chartData != nil` and a `DataBinding` exists for `module.DataKey`:
  - `ChartData.BindingID <- DataBinding.BindingID`
  - `ChartData.ChartID <- DataBinding.ChartID`

Summary:

```text
Script + ScriptModule + DataBinding + Domain Data
  -> ModuleData + ChartData

VideoSource entity + videoCache
  -> gateway.VideoSource
```

## 5. `ChartData` Fields: `x_axis`, `series`, `rows`

`gateway.ChartData` has three display data forms:

```go
XAxis  []string      `json:"x_axis,omitempty"`
Series []ChartSeries `json:"series,omitempty"`
Rows   [][]string    `json:"rows,omitempty"`
```

### 5.1 `x_axis`

`x_axis` is horizontal-axis data.

It fits coordinate-axis charts such as:

- line charts
- bar charts
- trend charts

Example:

```json
["08:00", "09:00", "10:00"]
```

### 5.2 `series`

`series` is chart data series. Its values usually align one-to-one with `x_axis`.

It fits:

- line charts
- bar charts
- multi-series statistics

Example:

```json
[
  {"name":"entry count","unit":"people/times","data":[120,180,260]},
  {"name":"exit count","unit":"people/times","data":[90,140,200]}
]
```

### 5.3 `rows`

`rows` is two-dimensional table/list/card data.

Current V9 `chart_data_push` mainly uses `rows`.

It fits:

- metric cards
- alert lists
- status panels
- table panels

Example:

```json
[
  ["metric", "value", "unit"],
  ["today entries", "1800", "people/times"],
  ["today exits", "1480", "people/times"],
  ["today total", "3280", "people/times"]
]
```

Rule of thumb:

```text
x_axis + series are for coordinate-axis charts.
rows is for tables, metric cards, status cards, and alert lists.
Current security_main mainly uses rows.
x_axis/series are reserved for future trend/statistics charts.
```

Mapper behavior:

- `Rows` is always copied to `ue.ChartDataPush.Rows`.
- If `Series` or `XAxis` is non-empty, the mapper also sets `ue.ChartDataPush.Payload = chart`, so FastAPI can receive the full `gateway.ChartData`.

## 6. `VideoSource` Fields and Conversion Chain

`gateway.VideoSource` is not the video stream itself. It is video source/window metadata for FastAPI/UE.

| Gateway field | Source | Mapper destination | Meaning |
|---|---|---|---|
| `video_id` | `video_sources.video_id` | `ue.VideoStream.StreamID` | Video stream identity, for example `vs_cam_65` |
| `title` | `video_sources.name` | `ue.VideoStream.Name` | Display name |
| `play_url` | `video_sources.play_url` | `ue.VideoStream.WebRTCPlayerURL` | Player URL |
| `status` | `videoCache` or default `"online"` | Not currently sent in `video_streams_set` | Gateway-side runtime state |
| `bind_model_id` | `video_sources.bind_model_id` | Not currently sent in `video_streams_set` | Gateway-side binding to UE model/point, for example `gate_model_01` |

Current conversion:

```text
gateway.VideoSource{
  VideoID: "vs_cam_65",
  Title: "gate camera",
  PlayURL: "https://media.example.com/demo/player?src=demo_camera",
  Status: "online",
  BindModelID: "gate_model_01",
}

->

ue.VideoStreams{
  Streams: [
    {
      StreamID: "vs_cam_65",
      Name: "gate camera",
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

Current `video_streams_set` mainly sends `video_id/title/play_url`.

`status` and `bind_model_id` are preserved as gateway-layer context. If FastAPI/UE later supports them in the protocol, they can be extended into the downstream payload.

The Go gateway does not pull streams, transcode streams, or distribute large-scale video. It only manages video source metadata and playback URLs.

## 7. DTO Return Chain

HTTP route:

```http
GET /api/scripts/:id/data-bundle
```

Struct flow:

```text
handler.GetDataBundle
  -> dto.GetDataBundleURI
  -> dto.GetDataBundleQuery
  -> service.BuildDataBundle
  -> gateway.DataBundle
  -> dto.APIResponse[gateway.DataBundle]
  -> HTTP JSON
```

Details:

- `dto.GetDataBundleURI` binds `:id` into `ScriptID`.
- `dto.GetDataBundleQuery` binds `use_cache`.
- DTO does not carry business logic.
- DTO only represents HTTP input/output.
- `gateway.DataBundle` is the main value inside the `data` field.
- `code/message/data` is the API response shell.

Simplified response:

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

Failure response uses `dto.Fail(code, message)` and returns the shell:

```json
{
  "code": "BUILD_DATA_BUNDLE_FAILED",
  "message": "..."
}
```

## 8. WebSocket Push Chain

HTTP route:

```http
POST /api/scripts/:id/preload
```

Struct flow:

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

Rules:

- `GET /api/scripts/:id/data-bundle` does not trigger WebSocket push.
- Only `POST /api/scripts/:id/preload` triggers WebSocket push.
- WebSocket push failure only logs Warn and does not fail HTTP preload.
- If `WSDataURL` is empty, `fastAPIClient` is nil and push is skipped.
- The Go gateway only pushes to FastAPI `/ws_data`; it does not directly control UE.

`handler.Preload` returns `dto.PreloadResponse`, not the whole bundle:

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

## 9. Mapper Conversion Rules

The actual mapper is `internal/service/mapper/ue_mapper.go`.

### 9.1 `ChartData -> chart_data_push`

Conversion:

```text
gateway.ChartData.BindingID -> ue.ChartDataPush.BindingID
gateway.ChartData.Rows      -> ue.ChartDataPush.Rows
bundle.GeneratedAt          -> ue.ChartDataPush.Timestamp
```

Current mapper does not map `gateway.ChartData.Value` because `ChartData` has no `Value` field.

Current mapper behavior for `x_axis` and `series`:

- If `chart.Series` or `chart.XAxis` is non-empty, `ue.ChartDataPush.Payload = chart`.
- This means FastAPI receives the full `gateway.ChartData` in `payload`.
- `Rows` is still copied to `ue.ChartDataPush.Rows`.

Charts without `BindingID` are skipped and do not become `chart_data_push` messages.

### 9.2 `VideoSource -> video_streams_set`

Conversion:

```text
gateway.VideoSource.VideoID -> ue.VideoStream.StreamID
gateway.VideoSource.Title   -> ue.VideoStream.Name
gateway.VideoSource.PlayURL -> ue.VideoStream.WebRTCPlayerURL
[]VideoStream               -> ue.VideoStreams
ue.VideoStreams             -> ue.WSMessage{Type:"video_streams_set"}
```

`gateway.VideoSource.Status` and `gateway.VideoSource.BindModelID` are not currently included in `ue.VideoStream`.

### 9.3 `seq`

`seq` is the WebSocket message sequence number.

- Current calls use `BundleToWSMessages(bundle, 0)`.
- Message sequence starts at `0`.
- Messages increment in generation order.
- Chart messages are generated first.
- If videos exist, one `video_streams_set` message is appended after chart messages.

Typical `security_main` output:

```text
seq=0 chart_data_push security.access.today
seq=1 chart_data_push security.fire.status
seq=2 video_streams_set vs_cam_65
```

## 10. Binding Chain

Key chain:

```text
script_id
  -> module_id
  -> data_key
  -> binding_id
  -> chart_id / stream_id / model_id
```

Meanings:

- `script_id`: one report/display script, for example `security_main`.
- `module_id`: one display module inside the script, for example `today_access_card`.
- `data_key`: business semantic key, for example `security.access.today`.
- `binding_id`: FastAPI/UE runtime data binding ID, for example `security.access.today`.
- `chart_id`: FastAPI/UE chart container ID, for example `chart_h`.
- `target_kind`: target type, such as `chart`, `video`, `model`, or `hud`.
- `target_id`: target resource ID. Depending on `target_kind`, it can represent `chart_id`, `video_id`, or `model_id`.

Chart example:

```text
security_main
  -> today_access_card
  -> security.access.today
  -> security.access.today
  -> chart_h
```

Fire status chart example:

```text
security_main
  -> fire_alert_card
  -> security.fire.status
  -> security.fire.status
  -> chart_l
```

Video example:

```text
security_main
  -> gate_video_panel
  -> security.video.gate_01
  -> target_kind=video
  -> target_id=vs_cam_65
  -> video_streams_set.stream_id=vs_cam_65
```

Important distinction:

- Chart modules use `data_bindings` to get `binding_id` and `chart_id`.
- Video modules use `script_modules.target_id` as `video_sources.video_id`, then mapper converts that to `video_streams_set.stream_id`.
- `bind_model_id` is loaded from `video_sources.bind_model_id` and preserved in `gateway.VideoSource`.

## 11. Current Scope of Structs and Components

The following table distinguishes extension structs that are not yet part of script data assembly from independent features that are already implemented:

| Item | Current status | Purpose and boundary |
|---|---|---|
| `gateway.DataBundle.ModelBindings` | Initialized as an empty slice | Future UE model/point binding |
| `entity.ModelBinding` / `model_bindings` | Seed exists, service does not read it yet | Future model/point to data binding |
| `entity.ChartTemplate` / `chart_templates` | Seed exists, service does not read it yet | Future configurable chart/card/table templates |
| `gateway.ChartData.XAxis` / `Series` | Defined and mapper can carry them through `Payload`, but current `security_main` mainly uses `Rows` | Future trend/statistics charts |
| `cache.AuthCache` | Used for login session storage, validation, and logout | Shares the configurable Redis project prefix with business caches |
| `scheduler.PreloadJob` | Implemented; configuration controls startup in `App.Run` | Scheduled forced preload with optional push; does not probe video availability |
| `security.http_adapter.go` | Implemented; `NewApp` selects mock or HTTP mode from configuration | External security data normalization and authentication |
| `protocol/ue` model/path/render/report structs | Used by the UE configuration APIs | Configuration persistence and protocol compatibility, independent of `DataBundle.ModelBindings` |

Extension structs such as `ModelBindings` and `ChartTemplate` are not yet connected to script data assembly. Model placement and visibility are already provided by separate UE configuration APIs and should not be confused with the reserved bundle binding field.

Extension and configuration examples:

- Future bundles can use `ModelBindings` to carry model-to-business-data relationships.
- Trend charts can use `x_axis` and `series`.
- User login uses the existing session authentication; public seeds provide no default account.
- `scheduler.preload.enabled` enables scheduled preload.
- `adapters.security.mode: http` selects the implemented HTTP adapter.

## 12. Final Summary

The current struct flow is stable:

```text
entity provides the local MySQL configuration skeleton.
domain provides internal business data after adapter normalization.
service combines entity configuration, domain data, cache state, and generated metadata.
gateway.DataBundle is the unified display data package.
dto is the HTTP request/response shell.
mapper converts gateway structs to UE/FastAPI protocol structs.
FastAPIClient sends WSMessage values to FastAPI /ws_data.
```

The current boundaries are:

- Repository reads local MySQL configuration only.
- Cache reads/writes Redis runtime cache only.
- Adapter produces domain objects from mock data or configured external HTTP systems.
- Service assembles `gateway.DataBundle` and coordinates preload.
- Handler only binds HTTP input, calls service, and returns DTO response.
- Mapper only converts `gateway` to `protocol/ue`.
- Go gateway never directly controls UE.
