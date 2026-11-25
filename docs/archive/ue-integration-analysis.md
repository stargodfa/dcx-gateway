# Go 网关与 UE/FastAPI 配置联动分析

> 状态：历史归档。本文保留阶段性分析或设计提案，不代表当前功能与数据库现状。当前用法请以[文档首页](../README.md)、[接口说明](../api/ue-config.md)及源码为准。

本文基于当前两个本地目录做静态源码分析，不启动 UE、FastAPI、Web，也不修改业务代码或配置文件。

## 分析范围

Go 数据网关：

```text
<gateway-repo>
```

UE / FastAPI / Web 项目：

```text
<ue-client-repo>
```

重点检索目录：

```text
internal/
migrations/
config/
server/
web/
project_cpp/Source/
```

说明：提示中的 `Source` 目录在 UE 项目根目录下不存在，当前实际 C++ 源码目录为 `project_cpp/Source`。

## 检索关键词

本次主要检索：

```text
data-bundle
preload
BuildDataBundle
BundleToWSMessages
ChartDataPush
VideoStreams
video_streams_set
chart_data_push
ws_data
ws_web
ws_ue
binding_id
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
nDisplay
security_main
campus_security
chart_h
chart_l
vs_cam_65
```

## 当前 Go 网关链路

Go 网关当前已经形成 HTTP DataBundle 和 preload 推送链路：

```text
GET /api/scripts/:id/data-bundle
  -> handler
  -> service.BuildDataBundle
  -> repository / cache / adapter
  -> gateway.DataBundle

POST /api/scripts/:id/preload
  -> handler
  -> service.Preload
  -> BuildDataBundle
  -> ScriptCache.SetPreloadBundle
  -> mapper.BundleToWSMessages
  -> FastAPIClient.PushMessages
  -> FastAPI /ws_data
```

Go 侧仍保持分层清晰：

| 层级 | 当前职责 |
| --- | --- |
| API / DTO | 绑定 HTTP 参数，调用 service，返回统一响应 |
| Service | 查询 repository/cache/adapter，组装 `gateway.DataBundle`，preload 后触发 WS 推送 |
| Repository / Entity | 读取本地 MySQL 长期配置 |
| Adapter / Domain | 提供外部业务数据或 mock 数据 |
| Protocol / Mapper | 将 `gateway.DataBundle` 转为 FastAPI/UE 协议消息 |
| Bootstrap | 创建并持有 MySQL、Redis、FastAPI WS client、Logger |

## FastAPI `/ws_data` 当前协议事实

FastAPI 的 `/ws_data` 是外部数据进入通道，但当前实现只处理：

```text
chart_data_push
```

关键约束：

1. `type` 必须是 `chart_data_push`。
2. `payload.binding_id` 必须是非空字符串。
3. payload 必须包含 `values` 或 `cell_values`，可以在顶层，也可以在 `table` 内。
4. `/ws_data` 当前没有处理 `video_streams_set`。
5. `seq` 会由 FastAPI 控制层重新分配，Go 侧传入的 seq 不是关键字段。
6. 未发现 `/ws_data` 对 `client_id`、`role`、`token` 的强制校验。

当前 Go mapper 发出的 chart payload 主要包含：

```text
binding_id
timestamp
rows
```

这与 FastAPI 当前 `chart_data_push` 校验不完全匹配，因为 FastAPI 要求 `values` 或 `cell_values`，不会把 `rows` 当作有效图表数据。

## UE 图表接收约束

UE C++ 图表处理逻辑接收的是 FastAPI 转发后的 `chart_data_push`，并要求 payload 中存在：

```text
chart_id
```

Go 不一定需要直接发送 `chart_id`，因为 FastAPI 会根据 `chart_binding.json` 中的 `binding_id -> chart_id` 映射补齐 `chart_id`。但这要求：

```text
Go 发送的 binding_id == FastAPI chart_binding.json 中配置的 binding_id
```

否则 FastAPI 无法解析出 `chart_id`，UE 侧图表不会更新。

## 当前配置对齐情况

### report_scripts.json

当前 UE/FastAPI 配置中已有脚本包括：

```text
testscript
drone.view
```

未发现：

```text
security_main
campus_security
```

因此 Go 侧 `security_main` / `campus_security` 当前只是 Go 网关自己的脚本闭环，尚未与 UE/FastAPI 的 report script 编排配置对齐。

### chart_binding.json

当前绑定关系包括：

```text
sim_3 -> chart_h
sim_4 -> chart_l
```

Go 当前 DataBundle 中的业务数据 key / binding_id 是：

```text
security.access.today
security.fire.status
```

因此现状不匹配：

```text
security.access.today != sim_3
security.fire.status != sim_4
```

FastAPI 不能通过当前配置把 Go 的 binding_id 解析到 `chart_h` / `chart_l`。

### chart_config.json

当前 `chart_config.json` 中未发现：

```text
chart_h
chart_l
```

但 `chart_binding.json` 中存在 `sim_3 -> chart_h`、`sim_4 -> chart_l`。这说明 UE/FastAPI 图表容器配置与绑定配置自身也存在潜在不一致，需要 UE/FastAPI 配置侧确认 `chart_h`、`chart_l` 是否为真实可渲染图表容器。

### video_streams.json

当前存在：

```text
vs_cam_65
```

并且 URL 与 Go 侧 seed 数据中的视频地址一致。因此视频源 ID 本身是可对齐的。

但 `/ws_data` 当前不处理 `video_streams_set`，所以 Go preload 推送的 `video_streams_set` 不会直接驱动 UE 视频更新。当前 FastAPI/UE 视频链路更像是通过 report warmup / video catalog 使用 `stream_id` 查询 `video_streams.json`。

### model_config.json / model_table.json

这些文件主要维护 UE 模型实例、状态、变换、bounds 和模型表。Go 当前不应直接读写这些配置。若后续需要模型联动，应只通过已有协议或 FastAPI 控制入口表达意图，不能让 Go 直接控制 UE。

### path_config.json / render_config.json / init_config.json / drone_config.json

这些属于 UE/FastAPI 运行控制、初始化、渲染和无人机配置。当前 Go DataBundle 闭环不应直接介入。

### config/ndisplay

nDisplay 模板属于部署和多机渲染配置。本次 Go 网关与 DataBundle / `/ws_data` 对接不需要读取或修改。

## 主要差异与风险

| 编号 | 风险 | 影响 |
| --- | --- | --- |
| P0-1 | Go `chart_data_push` payload 使用 `rows`，FastAPI 要求 `values` 或 `cell_values` | `/ws_data` 校验可能失败，图表消息无法进入 FastAPI 状态 |
| P0-2 | Go binding_id 与 `chart_binding.json` 不一致 | FastAPI 无法解析 `chart_id`，UE 图表无法更新 |
| P0-3 | `video_streams_set` 当前不被 `/ws_data` 处理 | Go 推送视频消息不会直接产生 UE 视频效果 |
| P1-1 | `security_main` / `campus_security` 不存在于 `report_scripts.json` | Go 脚本与 UE/FastAPI 报告编排尚未对齐 |
| P1-2 | `chart_h` / `chart_l` 不存在于 `chart_config.json` | 即使 binding 对齐，UE/Web 图表容器也可能无法正常展示 |
| P2-1 | Go 与 FastAPI 对 chart payload shape 缺少契约测试 | 后续字段变化容易再次断链 |
| P2-2 | 视频链路的协议入口不明确 | 容易误以为 `/ws_data` 能动态设置视频流 |

## 可行性判断

当前 Go 网关的 HTTP DataBundle 和 preload 推送架构是可继续演进的，但还不能认为已经与 UE/FastAPI 配置完全打通。

最短可行路径不是重构 Go 架构，而是先做协议和配置对齐：

1. Go mapper 将图表数据转成 FastAPI 接受的 `cell_values` 或 `values`。
2. Go 发送的 `binding_id` 与 `chart_binding.json` 保持一致。
3. UE/FastAPI 配置侧确认对应 `chart_id` 存在于 `chart_config.json` 或真实 UE 图表容器中。
4. 视频保持使用现有 `video_streams.json` / report warmup 链路；不要假设 `video_streams_set` 已被 `/ws_data` 消费。

## 建议开发任务

| 优先级 | 任务 | 归属 | 说明 |
| --- | --- | --- | --- |
| P0 | 调整 Go chart mapper 输出 `cell_values` 或 `values` | Go | 不改 DataBundle 结构，只改 `gateway.DataBundle -> ue.WSMessage` 映射 |
| P0 | 对齐 chart binding_id | Go 或 UE/FastAPI 配置 | 二选一：Go 使用 `sim_3` / `sim_4`，或配置侧改为 `security.access.today` / `security.fire.status` |
| P0 | 确认 `chart_h` / `chart_l` 是否为有效图表容器 | UE/FastAPI 配置 | 若不是有效容器，需要在配置侧补齐或改到已有 chart_id |
| P1 | 对齐 `security_main` 与 UE report script | UE/FastAPI 配置 | 若要让 Go 的脚本概念驱动 UE 报告编排，需要在 `report_scripts.json` 中建立对应脚本/场景 |
| P1 | 明确视频更新链路 | UE/FastAPI | 当前建议继续使用 `video_streams.json` + report warmup，不把 `/ws_data` 当视频配置入口 |
| P2 | 增加 Go mapper 单元测试 | Go | 固定 `chart_data_push` payload 必须含 `binding_id` 和 `cell_values` / `values` |
| P2 | 增加 FastAPI 协议样例测试 | FastAPI | 固定 Go 推送样例能通过 `validate_chart_data_push_payload` |
| P3 | 增加跨项目契约文档 | 双方 | 记录 binding_id、chart_id、stream_id、script_id 的所有权和变更流程 |

## 不建议本阶段做的事

本阶段不建议：

1. 让 Go 直接连接 UE。
2. 让 Go 读写 UE JSON 配置。
3. 把 Go DataBundle 改成 UE 协议结构。
4. 重构 `buildModuleData`。
5. 引入 Kafka、Consul、nDisplay、多机渲染或真实外部系统。
6. 让 `/ws_data` 同时承担视频配置入口，除非 FastAPI/UE 明确新增该协议。

## 结论

Go 网关当前的分层和资源生命周期方向是合理的，HTTP DataBundle 闭环也可以作为稳定基础继续保留。真正阻塞 Go -> FastAPI -> UE 可视化联动的，不是 Go service 架构，而是当前协议 payload 和配置 ID 对齐问题。

最小收口顺序建议：

```text
先修 chart_data_push payload shape
再对齐 binding_id -> chart_id
再确认 chart_config/report_scripts
最后再讨论视频动态推送是否需要新协议
```
