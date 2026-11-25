CREATE TABLE IF NOT EXISTS scripts (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    script_id VARCHAR(64) NOT NULL COMMENT 'Go网关业务脚本ID',
    name VARCHAR(128) NOT NULL COMMENT '脚本名称',
    scene_id VARCHAR(64) NOT NULL COMMENT '关联UE/FastAPI场景ID引用',
    status VARCHAR(32) NOT NULL DEFAULT 'enabled' COMMENT '脚本状态：enabled/disabled',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (id),
    UNIQUE KEY uk_scripts_script_id (script_id),
    KEY idx_scripts_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Go网关业务脚本配置表';

CREATE TABLE IF NOT EXISTS script_modules (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    module_id VARCHAR(64) NOT NULL COMMENT '脚本内模块ID',
    script_id VARCHAR(64) NOT NULL COMMENT '所属业务脚本ID',
    type VARCHAR(64) NOT NULL COMMENT '模块类型：stat_card/chart/video/alert_list/model_point/hud等',
    title VARCHAR(128) NOT NULL COMMENT '模块展示标题',
    data_key VARCHAR(128) DEFAULT NULL COMMENT '业务数据key',
    template VARCHAR(64) DEFAULT NULL COMMENT 'Go侧展示模板ID',
    target_kind VARCHAR(32) DEFAULT NULL COMMENT '目标资源类型：chart/video/model/hud',
    target_id VARCHAR(128) DEFAULT NULL COMMENT '目标资源ID：chart_id/stream_id/instance_id等',
    sort_order INT NOT NULL DEFAULT 0 COMMENT '模块排序',
    enabled TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
    config_json JSON DEFAULT NULL COMMENT '模块轻量扩展配置JSON',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (id),
    UNIQUE KEY uk_script_modules_module_id (module_id),
    KEY idx_script_modules_script_id (script_id),
    KEY idx_script_modules_data_key (data_key),
    KEY idx_script_modules_target (target_kind, target_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='脚本展示模块配置表';

CREATE TABLE IF NOT EXISTS data_bindings (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    data_key VARCHAR(128) NOT NULL COMMENT '业务数据key',
    binding_id VARCHAR(128) NOT NULL COMMENT 'FastAPI chart_data_push使用的binding_id',
    chart_id VARCHAR(128) DEFAULT NULL COMMENT 'UE/FastAPI图表ID',
    module_id VARCHAR(128) DEFAULT NULL COMMENT '关联脚本模块ID',
    enabled TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (id),
    UNIQUE KEY uk_data_bindings_data_key (data_key),
    KEY idx_data_bindings_binding_id (binding_id),
    KEY idx_data_bindings_chart_id (chart_id),
    KEY idx_data_bindings_module_id (module_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='业务数据与UE/FastAPI图表绑定表';

CREATE TABLE IF NOT EXISTS video_sources (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    video_id VARCHAR(64) NOT NULL COMMENT '视频源ID，对应UE/FastAPI stream_id',
    name VARCHAR(128) NOT NULL COMMENT '视频源名称',
    play_url VARCHAR(512) NOT NULL COMMENT 'WebRTC或播放器访问地址',
    source_type VARCHAR(64) NOT NULL DEFAULT 'mock' COMMENT '视频源类型：mock/live等',
    bind_model_id VARCHAR(128) DEFAULT NULL COMMENT '关联UE模型点位或业务绑定ID',
    enabled TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (id),
    UNIQUE KEY uk_video_sources_video_id (video_id),
    KEY idx_video_sources_bind_model_id (bind_model_id),
    KEY idx_video_sources_enabled (enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='视频源长期配置表';

CREATE TABLE IF NOT EXISTS model_bindings (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    bind_model_id VARCHAR(128) NOT NULL COMMENT 'UE模型实例ID或Go业务绑定点位ID',
    object_type VARCHAR(64) NOT NULL COMMENT '业务对象类型：gate/building/room/device等',
    object_id VARCHAR(128) NOT NULL COMMENT '业务对象ID',
    data_keys_json JSON DEFAULT NULL COMMENT '关联业务数据key数组JSON',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (id),
    UNIQUE KEY uk_model_bindings_bind_model_id (bind_model_id),
    KEY idx_model_bindings_object (object_type, object_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='业务对象与UE模型点位绑定表';

CREATE TABLE IF NOT EXISTS chart_templates (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    template_id VARCHAR(64) NOT NULL COMMENT 'Go侧图表或卡片模板ID',
    name VARCHAR(128) NOT NULL COMMENT '模板名称',
    type VARCHAR(64) NOT NULL COMMENT '模板类型：chart/stat_card/table/alert_list等',
    config_json JSON DEFAULT NULL COMMENT '模板轻量配置JSON',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (id),
    UNIQUE KEY uk_chart_templates_template_id (template_id),
    KEY idx_chart_templates_type (type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Go侧图表与展示模板配置表';
