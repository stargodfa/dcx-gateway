# UE/FastAPI/Web JSON 配置对接静态分析

> 状态：历史归档。本文保留阶段性分析或设计提案，不代表当前功能与数据库现状。当前用法请以[文档首页](../README.md)、[接口说明](../api/ue-config.md)及源码为准。

本文档基于当前工作区的静态源码分析，结合 `docs/architecture/data-flow.md` 中的 Go 数据网关结构体流转说明，梳理 UE/FastAPI/Web 工程中 `config/*.json` 的职责、字段含义、数据流向，以及它们与 Go 数据网关 `DataBundle` 和 `/ws_data` 推送的对接关系。

本次只做文档分析，不修改业务代码、JSON 配置、Go 网关代码、FastAPI 代码、Web 前端代码或 UE 代码。

## 0. 静态分析范围与限制

当前工作目录是 Go 数据网关仓库 `dcx-gateway`。静态扫描结果显示：

- 存在：`configs/config.dev.yaml`、`internal/model/protocol/ue/*`、`internal/service/mapper/ue_mapper.go`、`internal/websocket/fastapi_client.go`、`docs/architecture/data-flow.md`。
- 未发现：`config/*.json` 目录。
- 未发现：`server/`、`web/`、`Source/` 目录。
- 未发现：`docs/protocol v1.0.md`、`docs/protocol v1.1.md`、`docs/protocol v1.1.1.md` 等 UE/FastAPI 项目文档。

因此本文对 JSON 文件的实际读取方、修改方、下发方、UE Runtime 消费细节，只能基于当前 Go 代码中的 `protocol/ue` 结构体、mapper、文档和命名进行静态推断。凡当前仓库源码无法确认的地方，均明确标为：

```text
源码中未确认 / 需要后续联调确认
```

## 1. 已使用的关键搜索词与结论

使用的关键搜索词：

```text
chart_config
chart_binding
video_streams
report_scripts
model_config
model_table
path_config
render_config
drone_config
init_config
chart_data_push
video_streams_set
/ws_data
/ws_ue
/ws_web
report_prepare
report_chapter_goto
state_sync
model_load
model_set
path_set
DcV1UeSession
ChartDataPush
VideoStreams
WSMessage
```

静态结论：

| 搜索对象 | 当前仓库命中情况 | 结论 |
|---|---|---|
| `chart_config.json` | 命中 `internal/model/protocol/ue/chart_config.go` 注释和结构体 | Go 侧已有兼容结构体，但未发现实际 JSON 文件和读取逻辑 |
| `chart_binding.json` | 命中 `internal/model/protocol/ue/chart_binding.go` | Go 侧明确认为它是 `binding_id -> chart_id` 映射 |
| `video_streams.json` | 命中 `internal/model/protocol/ue/video_streams.go` 和 mapper | Go mapper 会生成 `video_streams_set` 消息 |
| `report_scripts.json` | 命中 `internal/model/protocol/ue/report_scripts.go` | Go 侧有兼容结构体，但当前网关不生成/替换该配置 |
| `model_config.json` / `model_table.json` | 命中 `internal/model/protocol/ue/model_config.go` | Go 侧有模型配置结构体，但当前 DataBundle 主链路未使用 |
| `path_config.json` | 命中 `internal/model/protocol/ue/path_config.go` | Go 侧有路径配置结构体，但当前网关不控制路径 |
| `render_config.json` | 命中 `internal/model/protocol/ue/render_config.go` | Go 侧有渲染配置结构体，但当前网关不控制渲染 |
| `init_config.json` | 命中 `internal/model/protocol/ue/init_config.go` | Go 侧有启动配置结构体，但当前网关不生成启动配置 |
| `drone_config.json` | 当前仓库未命中 | 源码中未确认 |
| `/ws_data` | 命中 `fastapi_client.go`、mapper、配置 yaml | Go 网关只作为 client 推送消息到 FastAPI `/ws_data` |
| `/ws_web` / `/ws_ue` | 当前仓库未命中 | 源码中未确认 |
| `chart_data_push` | 命中 mapper 常量 | Go 会通过 `/ws_data` 推送图表运行时数据 |
| `video_streams_set` | 命中 mapper 常量 | Go 会通过 `/ws_data` 推送视频流列表消息 |

## 2. 总体边界判断

当前 Go 数据网关已经完成的数据链路：

```text
external data / mock
  -> Adapter
  -> Domain
  -> Entity configuration
  -> Service
  -> gateway.DataBundle
  -> dto HTTP response
  -> mapper -> protocol/ue.WSMessage
  -> FastAPIClient -> FastAPI /ws_data
```

Go 网关职责：

- 接入业务数据。
- 适配字段。
- 读 MySQL 配置骨架。
- 读写 Redis 缓存。
- 组装 `gateway.DataBundle`。
- 将 `DataBundle` 转为 `chart_data_push` / `video_streams_set` 等 `/ws_data` 消息。

FastAPI/UE/Web 仍应负责：

- 相机。
- 模型。
- 路径。
- 汇报脚本编排。
- 渲染控制。
- UE Runtime 通信。
- Web 控制入口。

重要结论：

```text
Go 网关不是 UE 控制入口。
Go 网关不应直接替换 report_scripts、model_config、path_config、render_config 等 UE/FastAPI 控制配置。
Go 网关与 UE/FastAPI 的关键契约点是 script_id、scene_id、binding_id、chart_id、stream_id、bind_model_id。
```

## 3. 配置文件分类总表

| 文件 | 主要作用 | 静态/动态 | 读取方 | 是否 Go 网关直接生成 | Go 网关对接方式 | 备注 |
|---|---|---|---|---|---|---|
| `report_scripts.json` | 汇报脚本、章节、场景、相机、模型、路径和预热资源编排 | 静态编排配置 | 源码中未确认；推测 FastAPI/Web/UE 使用 | 否 | Go 只按 `script_id` 提供 DataBundle 和 `/ws_data` 数据 | 不应由 Go 网关替代 |
| `chart_config.json` | 图表容器、HUD 面板、表格/卡片显示配置 | 静态 UI 配置 | 源码中未确认；Go 侧有 `ue.ChartConfig` 结构 | 否 | Go 的 `ChartData.ChartID/Template/Title` 需与之对齐 | 不写运行时业务值 |
| `chart_binding.json` | `binding_id -> chart_id` 映射 | 静态绑定配置 | 源码中未确认；Go 侧有 `ue.ChartBinding` 结构 | 否 | Go 推送 `chart_data_push.binding_id` | 必须与 Go `data_bindings.binding_id` 对齐 |
| `video_streams.json` | 视频源/视频窗口默认配置 | 静态或可被运行时覆盖的配置 | 源码中未确认；Go 侧有 `ue.VideoStreams` 结构 | 否，Go 只推运行时 `video_streams_set` | `gateway.VideoSource -> ue.VideoStreams` | Go 不负责拉流/转码 |
| `model_config.json` | UE 模型实例状态、可见性、Transform 等 | 静态/运行时状态配置 | 源码中未确认；Go 侧有 `ue.ModelConfig` | 否 | Go 仅保留 `bind_model_id` / `ModelBinding` 关系 | 模型资源仍由 UE/FastAPI 维护 |
| `model_table.json` | 可操作模型实例清单 | 静态资源表 | 源码中未确认；Go 侧有 `ue.ModelTable` | 否 | Go 的 `bind_model_id` 需要能关联实例/点位 | 需后续确认 ID 命名 |
| `path_config.json` | 路径、导航、waypoints、播放参数 | 静态控制配置 | 源码中未确认；Go 侧有 `ue.PathConfig` | 否 | 当前无直接对接 | 路径控制仍归 FastAPI/UE |
| `render_config.json` | 渲染参数、后处理、光照等 | 静态/启动或运行时控制配置 | 源码中未确认；Go 侧有 `ue.RenderConfig` | 否 | 当前无直接对接 | 与业务数据网关关系弱 |
| `init_config.json` | 启动初始化配置，相机、自动加载、实例 ID | 启动配置 | 源码中未确认；Go 侧有 `ue.InitConfig` | 否 | 当前无直接对接 | 启动生命周期仍归 FastAPI/UE |
| `drone_config.json` | 无人机/移动视角/位置上报相关配置 | 源码中未确认 | 当前仓库未命中 | 否 | 当前无直接对接 | 需要 UE/FastAPI 项目源码确认 |
| `config/ndisplay/*` | nDisplay/多机渲染配置 | 静态控制配置 | 当前仓库未包含 | 否 | 当前无直接对接 | 不属于 Go 网关职责 |

## 4. `report_scripts.json`

### 4.1 作用判断

根据 `internal/model/protocol/ue/report_scripts.go`，该配置对应汇报脚本编排：

- `ReportScripts.Version`
- `ReportScripts.Scripts`
- `ReportScript.ScriptID`
- `ReportScript.Label`
- `ReportChapter.ChapterID`
- `ReportChapter.Label`
- `ReportScene.SceneID`
- `ReportScene.Label`
- `ReportScene.Camera`
- `ReportScene.Bindings`
- `ReportScene.Warmup`

它更偏向汇报编排配置，而不是业务数据配置。

### 4.2 字段含义

| 字段 | 含义 | Go 网关是否应生成 |
|---|---|---|
| `script_id` | 汇报脚本 ID，例如 `security_main` | 否，Go 只使用同名 ID 提供数据 |
| `chapter_id` | 汇报章节 ID | 否 |
| `scene_id` | 场景 ID，例如 `campus_security` | Go 的 `DataBundle.SceneID` 需与之对齐 |
| `camera` | 场景相机位置/旋转 | 否 |
| `bindings` | 场景模型绑定/显隐/Transform | 否 |
| `warmup.data` | 预热数据源或 binding ID | 可能与 Go `binding_id` 对齐，需确认 |
| `warmup.videos` | 预热视频资源 ID | 可能与 `stream_id` / `video_id` 对齐，需确认 |
| `warmup.charts` | 预热图表 ID | 可能与 `chart_id` 对齐，需确认 |

### 4.3 与 Go 网关关系

Go 网关当前输出：

- `gateway.DataBundle.ScriptID`
- `gateway.DataBundle.SceneID`
- `gateway.ModuleData`
- `gateway.ChartData`
- `gateway.VideoSource`

建议边界：

```text
report_scripts.json 继续由 FastAPI/UE/Web 工程维护。
Go 网关只根据 script_id 提供该脚本所需的业务数据包和 /ws_data 运行时数据。
Go 网关不直接生成、替换或执行 report_scripts.json。
```

### 4.4 未确认点

- FastAPI 如何读取并执行 `report_scripts.json`：源码中未确认。
- Web 控制台如何触发脚本：源码中未确认。
- UE Runtime 最终收到的脚本执行指令格式：源码中未确认。
- `warmup.data/videos/charts` 是否直接对应 Go 的 `binding_id/video_id/chart_id`：需要后续联调确认。

## 5. `chart_config.json`

### 5.1 作用判断

根据 `internal/model/protocol/ue/chart_config.go`，该配置对应图表/HUD/表格面板配置：

- `ChartConfig.Version`
- `ChartConfig.OverlayHUDMaxPanelWidth`
- `ChartConfig.Charts`
- `ChartPanel.ChartID`
- `ChartPanel.ChartName`
- `ChartPanel.Visible`
- `ChartPanel.Rows`
- `ChartPanel.Cols`
- `ChartPanel.ColFields`
- `ChartPanel.RowFields`
- `ChartPanel.CellValues`
- `ChartPanel.LayoutMode`

它是图表容器和显示配置，不是动态业务数据。

### 5.2 与 Go `ChartData` 的区别

```text
chart_config.json = 图表容器/展示配置
Go DataBundle.ChartData = 图表运行时业务数据
```

Go `gateway.ChartData` 字段：

- `ChartID`
- `BindingID`
- `Template`
- `Title`
- `Rows`
- `XAxis`
- `Series`

对接关系：

| Go 字段 | 目标配置字段 | 说明 |
|---|---|---|
| `ChartData.ChartID` | `chart_config.charts[].chart_id` | 需要保持一致，用于定位图表容器 |
| `ChartData.Template` | 源码中未确认 | 当前 Go 保留模板 ID；是否对应 chart_config 需确认 |
| `ChartData.Title` | `chart_name` 或显示标题 | 命名是否要求一致需确认 |
| `ChartData.Rows` | 运行时数据，不应写入静态配置 | 通过 `/ws_data` 推送 |
| `ChartData.XAxis/Series` | 运行时数据，不应写入静态配置 | 通过 `/ws_data` payload 透传 |

### 5.3 Go 网关不应做的事

- 不应把业务值写入 `chart_config.json`。
- 不应直接管理图表容器布局。
- 不应替代 Web/UE 对 HUD 和图表面板的配置管理。

## 6. `chart_binding.json`

### 6.1 作用判断

根据 `internal/model/protocol/ue/chart_binding.go`：

```go
type ChartBinding struct {
    Version  int
    Bindings []ChartBindingItem
}

type ChartBindingItem struct {
    BindingID string
    ChartID   string
}
```

它定义 `binding_id -> chart_id` 的映射。

### 6.2 关键链路

```text
Go data_key
  -> Go data_bindings.binding_id
  -> chart_data_push.binding_id
  -> FastAPI chart_binding
  -> chart_id
  -> UE chart container update
```

当前 Go 侧：

- `entity.DataBinding.DataKey` 来自 MySQL `data_bindings.data_key`
- `entity.DataBinding.BindingID` 来自 MySQL `data_bindings.binding_id`
- `entity.DataBinding.ChartID` 来自 MySQL `data_bindings.chart_id`
- `gateway.ChartData.BindingID` 进入 `ue.ChartDataPush.BindingID`
- `gateway.ChartData.ChartID` 保留在 HTTP DataBundle 中

### 6.3 必须对齐字段

| Go/MySQL 字段 | UE/FastAPI 配置字段 | 要求 |
|---|---|---|
| `data_bindings.binding_id` | `chart_binding.bindings[].binding_id` | 必须一致 |
| `data_bindings.chart_id` | `chart_binding.bindings[].chart_id` | 应一致 |
| `script_modules.data_key` | Go 内部业务语义键 | 与 binding 关系要稳定 |

### 6.4 未确认点

- FastAPI 收到 `chart_data_push` 后，是按 `binding_id` 查 `chart_id`，还是直接信任 payload 中的其他字段：源码中未确认。
- 是否要求同时传 `chart_id`：当前 Go mapper 只把 `binding_id` 放进 `ChartDataPush` 顶层，`chart_id` 只在 `Payload` 包含完整 chart 时可能出现。

## 7. `video_streams.json`

### 7.1 作用判断

根据 `internal/model/protocol/ue/video_streams.go`：

```go
type VideoStreams struct {
    Version int
    Streams []VideoStream
}

type VideoStream struct {
    StreamID        string
    Name            string
    WebRTCPlayerURL string
}
```

它是视频源/视频窗口配置或运行时视频流列表。

### 7.2 Go 映射关系

当前 mapper：

```text
gateway.VideoSource.VideoID -> ue.VideoStream.StreamID
gateway.VideoSource.Title   -> ue.VideoStream.Name
gateway.VideoSource.PlayURL -> ue.VideoStream.WebRTCPlayerURL
```

然后包装为：

```text
ue.WSMessage{
  Type: "video_streams_set",
  Payload: ue.VideoStreams
}
```

### 7.3 字段说明

| 字段 | 来源 | 说明 |
|---|---|---|
| `stream_id` | Go `VideoSource.VideoID` / MySQL `video_sources.video_id` | 视频流 ID |
| `name` | Go `VideoSource.Title` / MySQL `video_sources.name` | 视频显示名称 |
| `webrtc_player_url` | Go `VideoSource.PlayURL` / MySQL `video_sources.play_url` | WebRTC 播放地址 |
| `rtsp_url` | 当前 Go 协议结构未包含 | 源码中未确认 |
| `hls_url` | 当前 Go 协议结构未包含 | 源码中未确认 |
| `bind_model_id` | Go `VideoSource.BindModelID` 保留在 DataBundle | 当前 mapper 未下发到 `video_streams_set` |
| `status` | Go `VideoSource.Status` 来自 videoCache 或默认 online | 当前 mapper 未下发到 `video_streams_set` |

### 7.4 边界

Go 网关只管理视频源元数据和播放地址，不负责拉流、转码或大规模视频分发。

需要后续确认：

- UE/FastAPI 是否需要 `status`。
- UE/FastAPI 是否消费 `bind_model_id`。
- `play_url` 是否必须是 WebRTC URL，还是也支持 RTSP/HLS。

## 8. `model_config.json` 与 `model_table.json`

### 8.1 作用判断

根据 `internal/model/protocol/ue/model_config.go`：

- `ModelTable` 对应 `model_table.json`。
- `ModelConfig` 对应 `model_config.json`。
- `SceneDesiredConfig` 描述 server 希望 UE 当前加载哪些模型。

涉及字段：

- `instance_id`
- `model_id`
- `display_name`
- `visible`
- `transform`
- `bounds_size`
- `uri`
- `format`
- `source_path`

### 8.2 与 Go 网关关系

Go 侧当前有：

- `gateway.ModelBinding`
- `entity.ModelBinding`
- `gateway.VideoSource.BindModelID`
- `entity.VideoSource.BindModelID`

但当前 `security_main` 主链路：

- `ModelBindings` 初始化为空。
- `model_bindings` 表 seed 存在，但 service 未读取。
- `bind_model_id` 只保留在 `gateway.VideoSource`，mapper 未下发给 UE。

建议边界：

```text
模型资源和实例配置继续由 UE/FastAPI 项目维护。
Go 网关只通过 bind_model_id / object_id / data_key 维护业务数据与模型点位的绑定关系。
```

### 8.3 后续需要对齐的 ID

如果后续实现“点击模型点位显示业务数据/视频”，至少需要确认：

| Go 字段 | UE/FastAPI 字段 | 用途 |
|---|---|---|
| `VideoSource.BindModelID` | `instance_id` 或绑定点位 ID | 将视频源绑定到模型/点位 |
| `ModelBinding.BindModelID` | `instance_id` 或绑定点位 ID | 将业务对象绑定到 UE 实例 |
| `ModelBinding.ObjectID` | 外部业务对象 ID | 例如 `gate_01` |
| `ModelBinding.DataKeysJSON` | Go `data_key` 列表 | 点位关联哪些业务数据 |

当前源码无法确认 `bind_model_id` 是否就是 UE `instance_id`。

## 9. `path_config.json`

### 9.1 作用判断

根据 `internal/model/protocol/ue/path_config.go`，该配置包含：

- `PathConfig.Version`
- `NavPlayback`
- `PlansByID`
- `PathPlan.PathID`
- `PathPlan.Name`
- `PathPlan.Waypoints`

它更像相机路径、导航路线、waypoint 和路径播放参数配置。

### 9.2 边界

```text
Go 网关不直接控制 UE 路径。
路径、相机、导航仍由 FastAPI/UE 控制链路负责。
```

如果业务数据需要触发路径切换，建议由 FastAPI 的汇报编排或 Web 控制入口触发，而不是 Go 网关直接控制 UE。

未确认点：

- Web 前端如何触发路径播放：源码中未确认。
- UE 如何执行路径：源码中未确认。
- 是否存在 `/ws_ue` 指令如 `path_set`：当前仓库未确认。

## 10. `render_config.json`、`init_config.json`、`drone_config.json`

### 10.1 `render_config.json`

根据 `internal/model/protocol/ue/render_config.go`，该配置包含：

- `shadows`
- `AO`
- `bloom`
- `eye_adaptation`
- `extend_luminance_range`
- `local_exposure_highlight`
- `local_exposure_shadow`
- `exposure_bias`
- `tonemapper_gamma`
- `contrast`
- `saturation`
- `sharpen`
- `vignette`

它属于 UE Runtime / FastAPI 控制配置，与 Go 业务数据网关关系弱。

Go 网关不应读写它。

### 10.2 `init_config.json`

根据 `internal/model/protocol/ue/init_config.go`，该配置包含：

- `camera`
- `auto_load_on_startup`
- `instance_ids`

它是启动初始化配置。

Go 网关不应生成或替换它。

### 10.3 `drone_config.json`

当前仓库没有命中 `drone_config`，也没有对应 Go 结构体。

可能是无人机/无人车/移动视角/位置上报相关配置，但在当前源码中未确认。

## 11. FastAPI Server 数据流分析

当前 Go 仓库只包含 `internal/websocket/fastapi_client.go`，没有 FastAPI server 源码。

已确认：

- Go 从 `configs/config.dev.yaml` 读取 `fastapi.ws_data_url`。
- `websocket.FastAPIClient` 使用 gorilla websocket 连接该 URL。
- `PushMessages` 逐条发送 `ue.WSMessage` JSON。
- 消息类型由 mapper 生成：
  - `chart_data_push`
  - `video_streams_set`

源码中未确认：

- `/ws_data` 收到消息后是直接广播给 UE，还是先更新 server 状态。
- `/ws_web`、`/ws_ue` 的存在和协议细节。
- `chart_data_push` 是否要求 `role`、`client_id`、`token`。
- `video_streams_set` 是否要求 ACK 或 request ID。
- `seq` 是否必须严格连续。
- `chart_data_push.payload` 的完整字段要求。
- `video_streams_set.payload` 是否支持 `status` / `bind_model_id`。

## 12. Web 前端数据流分析

当前仓库没有 `web/` 前端源码。

因此以下内容源码中未确认：

- Web 控制台是否读取 `chart_config.json`。
- Web 是否展示/编辑 `chart_binding.json`。
- Web 是否展示/编辑 `video_streams.json`。
- Web 是否触发 `report_scripts.json` 中的脚本。
- Web 是否直接消费 Go 网关推送的数据。

基于边界建议：

```text
Web 前端应继续作为控制入口和配置展示入口。
Go 网关只通过 HTTP DataBundle 和 FastAPI /ws_data 提供业务数据。
Web 是否直接访问 Go HTTP 接口需要后续项目集成确认。
```

## 13. UE Runtime 数据流分析

当前仓库没有 UE `Source/` 代码。

从 Go 侧 `protocol/ue` 可确认的结构：

- `ue.WSMessage`
- `ue.ChartDataPush`
- `ue.VideoStreams`
- `ue.VideoStream`
- `ue.ChartBinding`
- `ue.ChartConfig`
- `ue.ModelConfig`
- `ue.PathConfig`
- `ue.RenderConfig`
- `ue.ReportScripts`
- `ue.InitConfig`

源码中未确认：

- UE 最终是按 `binding_id` 还是 `chart_id` 更新图表。
- UE 是否直接消费 `chart_data_push`，还是由 FastAPI 转换后再发 UE。
- UE 对视频流字段是否只要求 `stream_id/name/webrtc_player_url`。
- UE 是否消费 `bind_model_id`。
- UE 是否消费 `status`。
- `DcV1UeSession` 是否存在以及具体协议字段。

当前 Go mapper 已确认：

```text
ChartData.BindingID -> ChartDataPush.BindingID
ChartData.Rows      -> ChartDataPush.Rows
VideoSource.VideoID -> VideoStream.StreamID
VideoSource.Title   -> VideoStream.Name
VideoSource.PlayURL -> VideoStream.WebRTCPlayerURL
```

## 14. Go 网关字段对接表

| Go 网关字段 | 来源 | 目标 UE/FastAPI 字段 | 对应 JSON 配置 | 说明 |
|---|---|---|---|---|
| `DataBundle.script_id` | `scripts.script_id` | `report_scripts.script_id` | `report_scripts.json` | 应保持同一脚本 ID |
| `DataBundle.scene_id` | `scripts.scene_id` | `report_scripts.scenes[].scene_id` | `report_scripts.json` | 应保持场景 ID 对齐 |
| `ChartData.binding_id` | `data_bindings.binding_id` | `chart_data_push.binding_id` | `chart_binding.json` | 最关键运行时图表绑定 ID |
| `ChartData.chart_id` | `data_bindings.chart_id` | `chart_binding.chart_id` / chart container ID | `chart_binding.json`, `chart_config.json` | 当前 HTTP DataBundle 保留；mapper 顶层未单独下发 |
| `ChartData.template` | `script_modules.template` | 模板/展示配置 ID | `chart_config.json` 或 chart template 配置 | 当前对应关系需确认 |
| `ChartData.rows` | Service 由 domain 数据生成 | `chart_data_push.rows` | 不应写入静态 JSON | 当前 security_main 主要使用 |
| `ChartData.x_axis` | Service 未来生成 | `chart_data_push.payload.x_axis` | 不应写入静态 JSON | 当前保留给趋势图 |
| `ChartData.series` | Service 未来生成 | `chart_data_push.payload.series` | 不应写入静态 JSON | 当前保留给多系列图 |
| `VideoSource.video_id` | `video_sources.video_id` | `video_streams_set.streams[].stream_id` | `video_streams.json` | 应与 stream_id 对齐 |
| `VideoSource.title` | `video_sources.name` | `video_streams_set.streams[].name` | `video_streams.json` | 视频显示名称 |
| `VideoSource.play_url` | `video_sources.play_url` | `video_streams_set.streams[].webrtc_player_url` | `video_streams.json` | 当前 mapper 下发为 WebRTC player URL |
| `VideoSource.status` | `videoCache` 或默认 `online` | 当前 mapper 未下发 | 可能扩展 `video_streams.json` | 是否需要下发需确认 |
| `VideoSource.bind_model_id` | `video_sources.bind_model_id` | 当前 mapper 未下发；可能对应 model instance/bind point | `model_table.json`, `model_config.json` | 模型点位关联需确认 |
| `ModuleData.module_id` | `script_modules.module_id` | 源码中未确认 | 可能对应 Web/报表模块 | Go HTTP DataBundle 中保留 |
| `ModuleData.data_key` | `script_modules.data_key` | 间接关联 `binding_id` | `chart_binding.json` 间接关联 | Go 内部业务语义键 |
| `ModuleData.target_kind` | `script_modules.target_kind` | chart/video/model/hud target kind | 多类配置 | 决定 target_id 语义 |
| `ModuleData.target_id` | `script_modules.target_id` | chart_id / stream_id / model_id | `chart_config.json`, `video_streams.json`, model config | 视频模块当前用作 `video_id` |

## 15. 当前对接结论

### 15.1 Go 网关现在应该对接的点

- `script_id`
- `scene_id`
- `data_key`
- `binding_id`
- `chart_id`
- `video_id` / `stream_id`
- `play_url` / `webrtc_player_url`
- `bind_model_id`，当前先保留在 DataBundle
- `/ws_data` 消息类型：
  - `chart_data_push`
  - `video_streams_set`

### 15.2 Go 网关暂时不应触碰的点

- `report_scripts.json` 编排。
- 相机控制。
- 模型加载、显隐、Transform。
- 路径播放。
- 渲染参数。
- init 启动配置。
- nDisplay 配置。
- UE Runtime WebSocket 控制协议。
- Web 控制台配置管理逻辑。

### 15.3 下一阶段联调最关键字段

需要和 UE/FastAPI 工程师确认：

| 字段 | 需要确认的问题 |
|---|---|
| `binding_id` | FastAPI/UE 是否按它定位图表 |
| `chart_id` | 是否需要在 `chart_data_push` 顶层一起传 |
| `stream_id` | 是否必须等于 Go `video_id` |
| `webrtc_player_url` | 是否接受当前 `play_url` 格式 |
| `bind_model_id` | 是否对应 UE `instance_id` 或其他点位 ID |
| `status` | 视频状态是否需要下发 |
| `seq` | 是否必须连续、是否用于幂等/ACK |
| `role/client_id/token` | `/ws_data` 是否需要鉴权或身份字段 |

### 15.4 需要和 Go MySQL 配置保持一致的 JSON

| Go MySQL 配置 | 需要对齐的 JSON |
|---|---|
| `scripts.script_id` | `report_scripts.json` |
| `scripts.scene_id` | `report_scripts.json` |
| `data_bindings.binding_id` | `chart_binding.json` |
| `data_bindings.chart_id` | `chart_binding.json`, `chart_config.json` |
| `video_sources.video_id` | `video_streams.json` |
| `video_sources.play_url` | `video_streams.json` |
| `video_sources.bind_model_id` | `model_table.json`, `model_config.json` |

### 15.5 可能需要 mapper 微调的字段

当前不修改 mapper，只记录风险：

- `chart_data_push` 是否需要顶层 `chart_id`。
- `video_streams_set` 是否需要 `status`。
- `video_streams_set` 是否需要 `bind_model_id`。
- `ChartData.XAxis/Series` 是否应该进入专门字段，而不是仅放在 `payload`。
- `/ws_data` 是否需要 `role`、`client_id`、`request_id` 或 ACK 字段。

## 16. 对接风险表

| 风险点 | 可能原因 | 影响 | 建议 |
|---|---|---|---|
| `binding_id` 不一致 | Go MySQL `data_bindings` 与 `chart_binding.json` 不一致 | 图表收不到数据或更新错误图表 | 统一维护 binding ID 清单 |
| `chart_id` 不一致 | `data_bindings.chart_id`、`chart_binding.json`、`chart_config.json` 不一致 | FastAPI/UE 无法定位图表容器 | 联调前做 ID 对账 |
| `video stream_id` 不一致 | Go `video_id` 与 `video_streams.json.stream_id` 不一致 | 视频窗口无法更新 | 明确 `video_id == stream_id` 是否为契约 |
| `play_url` 类型不匹配 | Go 使用 WebRTC player URL，UE/FastAPI 可能要求 RTSP/HLS/WebRTC 不同字段 | 视频不可播放 | 确认 URL 类型和字段名 |
| `/ws_data` message type 不一致 | FastAPI 期望类型名不同 | 消息被忽略 | 固化 `chart_data_push` / `video_streams_set` 常量 |
| `chart_data_push` payload 不一致 | FastAPI 期望 `chart_id`、`value` 或其他字段 | 图表更新失败 | 对齐 payload schema |
| `video_streams_set` payload 不一致 | FastAPI 期望更多字段，如 status、bind_model_id | 视频状态或模型绑定不可用 | 确认是否扩展 mapper |
| `DataBundle.script_id` 与 `report_scripts` 不一致 | MySQL seed 与 UE/FastAPI 脚本配置不同步 | Web 触发脚本后拿不到对应数据 | 统一脚本 ID |
| `scene_id` 不一致 | Go `scripts.scene_id` 与报表场景 ID 不一致 | 场景与数据包错配 | 联调前统一 scene ID |
| `bind_model_id` 未被 UE 消费 | 当前 mapper 未下发，UE 协议也未确认 | 点击模型点位无法关联视频/数据 | 确认 UE 点位 ID 体系，再决定是否下发 |
| `/ws_data` 需要鉴权但 Go 未传 | FastAPI server 要求 token/role/client_id | WS 连接或消息被拒绝 | 确认认证协议，必要时扩展 FastAPIClient |
| `seq` 语义不一致 | Go 从 0 递增，FastAPI 可能需要全局递增或 request 级递增 | ACK/幂等异常 | 确认 seq 作用 |

## 17. 最终建议

当前 Go 网关的核心职责不是管理 UE 全部配置，而是把业务数据转换为：

```text
gateway.DataBundle
/ws_data WSMessage
```

UE/FastAPI 项目应继续维护：

- 相机。
- 模型。
- 路径。
- 渲染。
- 汇报脚本编排。
- nDisplay。
- UE Runtime 控制协议。

Go 网关与 UE/FastAPI 的契约关键点是：

```text
script_id
scene_id
binding_id
chart_id
stream_id
bind_model_id
message type
payload schema
```

当前最稳妥的下一步不是让 Go 生成 `config/*.json`，而是做一次 ID 和消息 schema 对账：

1. 对齐 `report_scripts.json.script_id` 与 Go `scripts.script_id`。
2. 对齐 `chart_binding.json.binding_id/chart_id` 与 Go `data_bindings`。
3. 对齐 `video_streams.json.stream_id` 与 Go `video_sources.video_id`。
4. 确认 `/ws_data` 对 `chart_data_push` 和 `video_streams_set` 的完整 payload 要求。
5. 确认 `bind_model_id` 是否等于 UE 模型实例 ID 或点位 ID。

## 18. 本仓库未确认项清单

由于当前目录未包含 UE/FastAPI/Web 工程源码，以下内容需要后续在对应工程中确认：

- `config/*.json` 的实际文件内容。
- FastAPI server 对 `/ws_data`、`/ws_web`、`/ws_ue` 的 handler。
- Web 前端是否读取或编辑这些 JSON。
- UE Runtime 是否直接识别 `chart_data_push` / `video_streams_set`。
- UE 是否按 `binding_id` 或 `chart_id` 更新图表。
- UE 是否消费 `bind_model_id` 和 `status`。
- `drone_config.json` 的真实字段和用途。
- `config/ndisplay/*` 的具体格式。
