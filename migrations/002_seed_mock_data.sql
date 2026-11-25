-- 002_seed_mock_data.sql

INSERT INTO scripts (script_id, name, scene_id, status)
VALUES
('security_main', '安防汇报脚本', 'campus_security', 'enabled')
ON DUPLICATE KEY UPDATE
name = VALUES(name),
scene_id = VALUES(scene_id),
status = VALUES(status);


INSERT INTO script_modules
(module_id, script_id, type, title, data_key, template, target_kind, target_id, sort_order, enabled, config_json)
VALUES
('today_access_card', 'security_main', 'stat_card', '今日出入人数', 'security.access.today', 'stat_card_basic', 'chart', 'chart_h', 1, 1, JSON_OBJECT('position', 'left_top')),
('fire_alert_card', 'security_main', 'alert_status', '消防告警状态', 'security.fire.status', 'status_basic', 'chart', 'chart_l', 2, 1, JSON_OBJECT('position', 'left_middle')),
('gate_video_panel', 'security_main', 'video', '校门口监控', 'security.video.gate_01', 'video_basic', 'video', 'vs_cam_65', 3, 1, JSON_OBJECT('position', 'right_top'))
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
('security.access.today', 'sim_3', 'chart_h', 'today_access_card', 1),
('security.fire.status', 'sim_4', 'chart_l', 'fire_alert_card', 1)
ON DUPLICATE KEY UPDATE
binding_id = VALUES(binding_id),
chart_id = VALUES(chart_id),
module_id = VALUES(module_id),
enabled = VALUES(enabled);


INSERT INTO video_sources
(video_id, name, play_url, source_type, bind_model_id, enabled)
VALUES
('vs_cam_65', '校门口监控', 'https://media.example.com/demo/player?src=demo_camera', 'mock', 'gate_model_01', 1)
ON DUPLICATE KEY UPDATE
name = VALUES(name),
play_url = VALUES(play_url),
source_type = VALUES(source_type),
bind_model_id = VALUES(bind_model_id),
enabled = VALUES(enabled);


INSERT INTO model_bindings
(bind_model_id, object_type, object_id, data_keys_json)
VALUES
('gate_model_01', 'gate', 'gate_01', JSON_ARRAY('security.access.today', 'security.video.gate_01'))
ON DUPLICATE KEY UPDATE
object_type = VALUES(object_type),
object_id = VALUES(object_id),
data_keys_json = VALUES(data_keys_json);


INSERT INTO chart_templates
(template_id, name, type, config_json)
VALUES
('stat_card_basic', '基础指标卡', 'stat_card', JSON_OBJECT('width', 320, 'height', 120)),
('status_basic', '基础状态卡', 'status', JSON_OBJECT('width', 320, 'height', 120)),
('video_basic', '基础视频窗', 'video', JSON_OBJECT('width', 480, 'height', 270))
ON DUPLICATE KEY UPDATE
name = VALUES(name),
type = VALUES(type),
config_json = VALUES(config_json);
