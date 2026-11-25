-- Seed phase-1 mock report scripts for logistics energy and teaching.
-- These rows extend the existing DataBundle sample chain without connecting real external systems.

INSERT INTO scripts
(script_id, name, scene_id, status)
VALUES
('energy_main', '后勤能耗汇报脚本', 'campus_energy', 'enabled'),
('education_main', '教学汇报脚本', 'campus_teaching', 'enabled')
ON DUPLICATE KEY UPDATE
name = VALUES(name),
scene_id = VALUES(scene_id),
status = VALUES(status);

INSERT INTO chart_templates
(template_id, name, type, config_json)
VALUES
('table_basic', '基础表格', 'table', JSON_OBJECT('layout', 'rows'))
ON DUPLICATE KEY UPDATE
name = VALUES(name),
type = VALUES(type),
config_json = VALUES(config_json);

INSERT INTO script_modules
(module_id, script_id, type, title, data_key, template, target_kind, target_id, sort_order, enabled, config_json)
VALUES
('energy_total_card', 'energy_main', 'stat_card', '今日总能耗', 'energy.total.today', 'stat_card_basic', 'chart', 'chart_energy_total', 10, 1, JSON_OBJECT('source', 'mock_energy')),
('energy_building_table', 'energy_main', 'table', '楼栋能耗排行', 'energy.building.snapshot', 'table_basic', 'chart', 'chart_energy_building', 20, 1, JSON_OBJECT('source', 'mock_energy')),
('classroom_usage_card', 'education_main', 'stat_card', '教室使用概况', 'education.classroom.usage', 'stat_card_basic', 'chart', 'chart_education_usage', 10, 1, JSON_OBJECT('source', 'mock_education')),
('classroom_schedule_table', 'education_main', 'table', '今日课堂状态', 'education.classroom.schedule', 'table_basic', 'chart', 'chart_education_schedule', 20, 1, JSON_OBJECT('source', 'mock_education'))
ON DUPLICATE KEY UPDATE
script_id = VALUES(script_id),
type = VALUES(type),
title = VALUES(title),
data_key = VALUES(data_key),
template = VALUES(template),
target_kind = VALUES(target_kind),
target_id = VALUES(target_id),
sort_order = VALUES(sort_order),
enabled = VALUES(enabled),
config_json = VALUES(config_json);

INSERT INTO data_bindings
(data_key, binding_id, chart_id, module_id, enabled)
VALUES
('energy.total.today', 'energy.total.today', 'chart_energy_total', 'energy_total_card', 1),
('energy.building.snapshot', 'energy.building.snapshot', 'chart_energy_building', 'energy_building_table', 1),
('education.classroom.usage', 'education.classroom.usage', 'chart_education_usage', 'classroom_usage_card', 1),
('education.classroom.schedule', 'education.classroom.schedule', 'chart_education_schedule', 'classroom_schedule_table', 1)
ON DUPLICATE KEY UPDATE
binding_id = VALUES(binding_id),
chart_id = VALUES(chart_id),
module_id = VALUES(module_id),
enabled = VALUES(enabled);

INSERT INTO ue_chart_configs
(chart_id, chart_name, panel_key, visible, applied, layout_mode, rows_count, cols_count,
 col_fields_json, row_fields_json, pair_label_cols_json, pair_label_rows_json,
 default_cell_values_json, template, enabled, sort_order, config_json)
VALUES
('chart_energy_total', '今日总能耗', NULL, 1, 1, 'column_pairs', 3, 3,
 JSON_ARRAY('指标', '数值', '单位'),
 JSON_ARRAY('今日总能耗', '覆盖楼栋', ''),
 JSON_ARRAY(0),
 NULL,
 JSON_ARRAY(
   JSON_ARRAY('指标', '数值', '单位'),
   JSON_ARRAY('今日总能耗', '', 'kWh'),
   JSON_ARRAY('覆盖楼栋', '', '栋')
 ),
 'stat_card_basic', 1, 130, JSON_OBJECT('source', 'phase1_mock_seed')),
('chart_energy_building', '楼栋能耗排行', NULL, 1, 1, 'column_pairs', 5, 3,
 JSON_ARRAY('楼栋', '能耗', '单位'),
 JSON_ARRAY('图书馆', '教学楼A', '食堂', '宿舍区', ''),
 JSON_ARRAY(0),
 NULL,
 JSON_ARRAY(
   JSON_ARRAY('楼栋', '能耗', '单位'),
   JSON_ARRAY('图书馆', '', 'kWh'),
   JSON_ARRAY('教学楼A', '', 'kWh'),
   JSON_ARRAY('食堂', '', 'kWh'),
   JSON_ARRAY('宿舍区', '', 'kWh')
 ),
 'table_basic', 1, 140, JSON_OBJECT('source', 'phase1_mock_seed')),
('chart_education_usage', '教室使用概况', NULL, 1, 1, 'column_pairs', 4, 3,
 JSON_ARRAY('状态', '数量', '单位'),
 JSON_ARRAY('上课中', '空闲', '未知', ''),
 JSON_ARRAY(0),
 NULL,
 JSON_ARRAY(
   JSON_ARRAY('状态', '数量', '单位'),
   JSON_ARRAY('上课中', '', '间'),
   JSON_ARRAY('空闲', '', '间'),
   JSON_ARRAY('未知', '', '间')
 ),
 'stat_card_basic', 1, 150, JSON_OBJECT('source', 'phase1_mock_seed')),
('chart_education_schedule', '今日课堂状态', NULL, 1, 1, 'column_pairs', 5, 4,
 JSON_ARRAY('教室', '状态', '课程', '教师'),
 JSON_ARRAY('A101', 'A102', 'B201', 'B305', ''),
 JSON_ARRAY(0),
 NULL,
 JSON_ARRAY(
   JSON_ARRAY('教室', '状态', '课程', '教师'),
   JSON_ARRAY('A101', '', '', ''),
   JSON_ARRAY('A102', '', '', ''),
   JSON_ARRAY('B201', '', '', ''),
   JSON_ARRAY('B305', '', '', '')
 ),
 'table_basic', 1, 160, JSON_OBJECT('source', 'phase1_mock_seed'))
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
