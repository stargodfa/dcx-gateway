-- Synthetic public demo; no original scene coordinates or asset paths.
INSERT INTO ue_scene_desired_models
(instance_id, model_id, uri, format, source_path, display_name, visible,
 bounds_x, bounds_y, bounds_z, position_x, position_y, position_z,
 rotation_yaw, rotation_pitch, rotation_roll, scale_x, scale_y, scale_z,
 enabled, sort_order, config_json)
VALUES
('demo_model_1', 'demo_model', 'file://./models/demo/demo.udatasmith', 'udatasmith',
 './models/demo/demo.udatasmith', '演示模型', 1, 100, 100, 100,
 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 0, JSON_OBJECT('source', 'public_demo'))
ON DUPLICATE KEY UPDATE
model_id=VALUES(model_id), uri=VALUES(uri), format=VALUES(format),
source_path=VALUES(source_path), display_name=VALUES(display_name), visible=VALUES(visible),
bounds_x=VALUES(bounds_x), bounds_y=VALUES(bounds_y), bounds_z=VALUES(bounds_z),
position_x=VALUES(position_x), position_y=VALUES(position_y), position_z=VALUES(position_z),
rotation_yaw=VALUES(rotation_yaw), rotation_pitch=VALUES(rotation_pitch), rotation_roll=VALUES(rotation_roll),
scale_x=VALUES(scale_x), scale_y=VALUES(scale_y), scale_z=VALUES(scale_z),
enabled=VALUES(enabled), sort_order=VALUES(sort_order), config_json=VALUES(config_json);
