-- dcx-gateway database inspection checklist.
-- This file only contains read-only SQL. Do not add DELETE or UPDATE statements.

-- 1. 模型类型数量
SELECT COUNT(*) AS model_type_count
FROM ue_model_types;

-- 2. 启用实例数量
SELECT COUNT(*) AS enabled_instance_count
FROM ue_model_instances
WHERE enabled = 1;

-- 3. placement 数量
SELECT COUNT(*) AS placement_count
FROM ue_model_instance_placements;

-- 4. 查看某个实例目录状态
SELECT instance_id, model_id, display_name, enabled
FROM ue_model_instances
WHERE instance_id = 'demo_model_1';

-- 5. 查看某个实例 placement
SELECT *
FROM ue_model_instance_placements
WHERE instance_id = 'demo_model_1';

-- 6. 查看 demo_model 系列状态
SELECT instance_id, model_id, display_name, enabled
FROM ue_model_instances
WHERE instance_id LIKE 'demo_model_%'
ORDER BY instance_id;

-- 7. 查看 raw config documents
SELECT config_key, file_name, enabled, updated_at
FROM ue_config_documents
ORDER BY config_key;

-- 8. 检查 chart runtime 相关结构化表
SELECT COUNT(*) AS chart_config_count
FROM ue_chart_configs;

-- 9. 检查脚本配置
SELECT script_id, name, scene_id, status
FROM scripts
ORDER BY script_id;

-- 10. 检查模块绑定
SELECT script_id, module_id, data_key, target_kind, target_id
FROM script_modules
ORDER BY script_id, module_id;

-- 11. 检查认证用户状态，不查询 password_hash
SELECT id, username, role, enabled, updated_at
FROM ue_users
ORDER BY username;
