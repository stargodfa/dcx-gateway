# V12.1 Chart Config Runtime Design

> 状态：历史归档。本文保留阶段性分析或设计提案，不代表当前功能与数据库现状。当前用法请以[文档首页](../README.md)、[接口说明](../api/ue-config.md)及源码为准。

## 目标

V12.1 只落地 chart_config 的最小结构化改造，不结构化 model/path/render/drone/report_scripts/nDisplay，也不改变 `gateway.DataBundle`。

核心边界：

```text
MySQL ue_chart_configs            保存静态布局、默认占位 cell_values
Redis chart runtime cache         保存 DataBundle 生成的最新运行时 cell_values
gateway.DataBundle                保持业务数据快照结构
chart_data_push                   仍是动态数据主通道
chart-config?mode=runtime         返回静态配置 + 最新运行时数据组合视图
```

Go 网关不直接控制 UE，也不直接连接 UE。Go 只按 FastAPI/UE 既有协议把运行时图表数据推给 FastAPI `/ws_data`。

## 数据分层

### MySQL

新增表：

```text
ue_chart_config_settings
ue_chart_configs
```

`ue_chart_config_settings` 保存 chart_config 顶层配置，例如：

- `version`
- `overlay_hud_max_panel_width`

`ue_chart_configs` 保存 `chart_config.charts[]` 中每个图表的静态布局：

- `chart_id`
- `chart_name`
- `panel_key`
- `visible`
- `applied`
- `layout_mode`
- `rows_count`
- `cols_count`
- `col_fields_json`
- `row_fields_json`
- `pair_label_cols_json`
- `pair_label_rows_json`
- `default_cell_values_json`

注意：`default_cell_values_json` 只是默认占位值，不保存外部实时业务数据。

### Redis

新增 runtime cache：

```text
chart:runtime:binding:<binding_id>
chart:runtime:chart:<chart_id>
```

缓存结构：

```json
{
  "data_key": "security.access.today",
  "binding_id": "sim_3",
  "chart_id": "chart_h",
  "cell_values": [["指标", "数值", "单位"]],
  "updated_at": "2026-06-02T00:00:00Z"
}
```

`BuildDataBundle` 生成 `gateway.ChartData.Rows` 后，满足以下条件时写入 Redis：

```text
binding_id != ""
chart_id != ""
len(rows) > 0
```

Redis 写失败只记录 Warn，不影响 `GET /data-bundle` 或 `POST /preload` 返回。

## API mode

### raw/full

```http
GET /api/ue/config/chart-config?mode=raw
GET /api/ue/config/chart-config?mode=full
```

继续从 `ue_config_documents.content_json` 原样返回，不走 mapper。它是迁移期兜底，不是最终结构化主线。

### structured/light

```http
GET /api/ue/config/chart-config?mode=structured
GET /api/ue/config/chart-config?mode=light
```

`light` 是旧兼容别名，等价于 `structured`。

数据来源：

```text
ue_chart_config_settings
+ ue_chart_configs.default_cell_values_json
-> protocol/ue.ChartConfig
```

该模式只返回静态布局和默认占位值，不合并运行时业务数据。

### runtime

```http
GET /api/ue/config/chart-config?mode=runtime
```

数据来源：

```text
ue_chart_config_settings
+ ue_chart_configs
+ Redis chart runtime cache
-> protocol/ue.ChartConfig
```

合并规则：

```text
每个 chart:
  先使用 default_cell_values_json
  如果按 chart_id 命中 Redis runtime cache:
    cell_values = runtime.cell_values
  如果默认值和 runtime 都不存在:
    按 rows_count / cols_count 生成空二维数组
```

## 动态数据主通道

运行时数据仍走：

```text
External / Mock
  -> Adapter
  -> domain
  -> BuildDataBundle
  -> gateway.ChartData.Rows
  -> Redis chart runtime cache
  -> chart_data_push.cell_values
  -> FastAPI /ws_data
```

`internal/service/mapper/ue_mapper.go` 只负责运行时映射：

```text
DataBundle -> WSMessage
ChartData.Rows -> ChartDataPush.CellValues
```

chart_config 的 structured/runtime JSON 生成逻辑放在 `internal/service/mapper/ue_config_mapper.go`。

## Video 边界

静态分析确认 FastAPI `/ws_data` 当前只接受 `chart_data_push`，不能假设能处理 `video_streams_set`。

因此 V12.1 默认关闭 preload 中的 `video_streams_set` 推送：

```yaml
fastapi:
  push_video_streams: false
```

保留 `video_streams_set` 类型和 mapper 代码，只在显式开启配置时输出。视频仍走：

```text
/api/ue/config/video-streams
report_scripts.warmup.videos[]
video_streams.json / video_sources
warmup_video_catalog
```

## 保持不变

- 不改变 `gateway.DataBundle`。
- 不改变 `/api/scripts/:id/data-bundle`。
- 不改变 `/api/scripts/:id/preload` 主流程。
- 不把 runtime cell_values 长期写入 MySQL。
- 不结构化 model/path/render/drone/nDisplay。
- 不让 Go 直接控制 UE。
