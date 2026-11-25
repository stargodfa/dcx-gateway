-- Synthetic public demo model. The asset is a placeholder and is not distributed.
INSERT INTO ue_model_types
(model_id, uri, format, source_path, base_bounds_x, base_bounds_y, base_bounds_z, bounds_negative_extent_axis, enabled)
VALUES
('demo_model', 'file://./models/demo/demo.udatasmith', 'udatasmith', './models/demo/demo.udatasmith', 100, 100, 100, NULL, 1)
ON DUPLICATE KEY UPDATE
uri=VALUES(uri), format=VALUES(format), source_path=VALUES(source_path),
base_bounds_x=VALUES(base_bounds_x), base_bounds_y=VALUES(base_bounds_y), base_bounds_z=VALUES(base_bounds_z),
bounds_negative_extent_axis=VALUES(bounds_negative_extent_axis), enabled=VALUES(enabled);

INSERT INTO ue_model_instances
(instance_id, model_id, display_name, sort_order, enabled)
VALUES ('demo_model_1', 'demo_model', '演示模型', 0, 1)
ON DUPLICATE KEY UPDATE
model_id=VALUES(model_id), display_name=VALUES(display_name), sort_order=VALUES(sort_order), enabled=VALUES(enabled);

INSERT INTO ue_model_instance_placements
(instance_id, visible, position_x, position_y, position_z, rotation_yaw, rotation_pitch, rotation_roll, scale_x, scale_y, scale_z)
VALUES ('demo_model_1', 1, 0, 0, 0, 0, 0, 0, 1, 1, 1)
ON DUPLICATE KEY UPDATE
visible=VALUES(visible), position_x=VALUES(position_x), position_y=VALUES(position_y), position_z=VALUES(position_z),
rotation_yaw=VALUES(rotation_yaw), rotation_pitch=VALUES(rotation_pitch), rotation_roll=VALUES(rotation_roll),
scale_x=VALUES(scale_x), scale_y=VALUES(scale_y), scale_z=VALUES(scale_z);
