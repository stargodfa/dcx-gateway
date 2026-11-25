-- Seed structured chart_config from demo configuration/chart_config.json.
-- Runtime business cell_values must be written to Redis chart runtime cache, not this MySQL table.

INSERT INTO ue_chart_config_settings
(id, version, overlay_hud_max_panel_width, enabled)
VALUES
(1, 1, 920, 1)
ON DUPLICATE KEY UPDATE
version = VALUES(version),
overlay_hud_max_panel_width = VALUES(overlay_hud_max_panel_width),
enabled = VALUES(enabled);

INSERT INTO ue_chart_configs
(chart_id, chart_name, panel_key, visible, applied, layout_mode, rows_count, cols_count,
 col_fields_json, row_fields_json, pair_label_cols_json, pair_label_rows_json,
 default_cell_values_json, template, enabled, sort_order, config_json)
VALUES
('chart_006', 'PERF 性能表', 'perf_panel', 1, 1, NULL, NULL, NULL,
 NULL, NULL, NULL, NULL, NULL, NULL, 1, 10, NULL),
('chart_008', '模型列表', 'model_list_panel', 0, 1, NULL, NULL, NULL,
 NULL, NULL, NULL, NULL, NULL, NULL, 1, 20, NULL),
('chart_007', '汇报脚本表', 'report_scripts_panel', 0, 1, NULL, NULL, NULL,
 NULL, NULL, NULL, NULL, NULL, NULL, 1, 30, NULL),
('chart_003', '模型实例表', 'runtime_models_panel', 0, 1, NULL, NULL, NULL,
 NULL, NULL, NULL, NULL, NULL, NULL, 1, 40, NULL),
('chart_004', '预热BA表', 'warmup_media_panel', 0, 1, NULL, NULL, NULL,
 NULL, NULL, NULL, NULL, NULL, NULL, 1, 50, NULL),
('chart_005', '导航业务表', 'navigation_panel', 0, 1, NULL, NULL, NULL,
 NULL, NULL, NULL, NULL, NULL, NULL, 1, 60, NULL),
('chart_001', '链路指标表', 'link_panel', 0, 1, NULL, NULL, NULL,
 NULL, NULL, NULL, NULL, NULL, NULL, 1, 70, NULL),
('chart_002', '压力参数表', 'pressure_panel', 0, 1, NULL, NULL, NULL,
 NULL, NULL, NULL, NULL, NULL, NULL, 1, 80, NULL),
('chart_12', '123', NULL, 0, 1, 'column_pairs', 4, 3,
 JSON_ARRAY('', '', ''),
 JSON_ARRAY('', '', '', ''),
 JSON_ARRAY(0),
 NULL,
 JSON_ARRAY(JSON_ARRAY('', '', ''), JSON_ARRAY('', '', ''), JSON_ARRAY('', '', ''), JSON_ARRAY('', '', '')),
 NULL, 1, 90, NULL),
('chart_333', '3', NULL, 0, 1, 'column_pairs', 10, 5,
 JSON_ARRAY('', '', '', '', ''),
 JSON_ARRAY('', '', '', '', '', '', '', '', '', ''),
 JSON_ARRAY(0, 2),
 NULL,
 JSON_ARRAY(
   JSON_ARRAY('', '', '', '', ''),
   JSON_ARRAY('', '', '', '', ''),
   JSON_ARRAY('', '', '', '', ''),
   JSON_ARRAY('', '', '', '', ''),
   JSON_ARRAY('', '', '', '', ''),
   JSON_ARRAY('', '', '', '', ''),
   JSON_ARRAY('', '', '', '', ''),
   JSON_ARRAY('', '', '', '', ''),
   JSON_ARRAY('', '', '', '', ''),
   JSON_ARRAY('', '', '', '', '')
 ),
 NULL, 1, 100, NULL),
('chart_h', '今日出入人数', NULL, 1, 1, 'column_pairs', 4, 3,
 JSON_ARRAY('指标', '数值', '单位'),
 JSON_ARRAY('今日进入', '今日离开', '今日总数', ''),
 JSON_ARRAY(0),
 NULL,
 JSON_ARRAY(
   JSON_ARRAY('指标', '数值', '单位'),
   JSON_ARRAY('今日进入', '', '人次'),
   JSON_ARRAY('今日离开', '', '人次'),
   JSON_ARRAY('今日总数', '', '人次')
 ),
 'stat_card_basic', 1, 110, JSON_OBJECT('source', 'dcx-gateway')),
('chart_l', '消防告警状态', NULL, 1, 1, 'column_pairs', 4, 3,
 JSON_ARRAY('等级', '位置', '信息'),
 JSON_ARRAY('告警等级', '告警位置', '告警信息', ''),
 JSON_ARRAY(0),
 NULL,
 JSON_ARRAY(
   JSON_ARRAY('等级', '位置', '信息'),
   JSON_ARRAY('normal', '', '无告警'),
   JSON_ARRAY('', '', ''),
   JSON_ARRAY('', '', '')
 ),
 'status_basic', 1, 120, JSON_OBJECT('source', 'dcx-gateway'))
ON DUPLICATE KEY UPDATE
chart_name = VALUES(chart_name),
panel_key = VALUES(panel_key),
visible = VALUES(visible),
applied = VALUES(applied),
layout_mode = VALUES(layout_mode),
rows_count = VALUES(rows_count),
cols_count = VALUES(cols_count),
col_fields_json = VALUES(col_fields_json),
row_fields_json = VALUES(row_fields_json),
pair_label_cols_json = VALUES(pair_label_cols_json),
pair_label_rows_json = VALUES(pair_label_rows_json),
default_cell_values_json = VALUES(default_cell_values_json),
template = VALUES(template),
enabled = VALUES(enabled),
sort_order = VALUES(sort_order),
config_json = VALUES(config_json);
