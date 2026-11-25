CREATE TABLE IF NOT EXISTS ue_chart_config_settings (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    version INT NOT NULL DEFAULT 1 COMMENT 'chart_config配置版本',
    overlay_hud_max_panel_width INT NOT NULL DEFAULT 480 COMMENT 'UE Overlay HUD最大面板宽度',
    enabled TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (id),
    KEY idx_ue_chart_config_settings_enabled (enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='UE chart_config顶层静态配置表';

CREATE TABLE IF NOT EXISTS ue_chart_configs (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    chart_id VARCHAR(128) NOT NULL COMMENT 'UE/FastAPI图表ID',
    chart_name VARCHAR(255) NOT NULL COMMENT '图表展示名称',
    panel_key VARCHAR(128) DEFAULT NULL COMMENT 'UE内置面板识别key',
    visible TINYINT(1) NOT NULL DEFAULT 1 COMMENT '默认是否可见',
    applied TINYINT(1) NOT NULL DEFAULT 1 COMMENT '自定义图表是否已应用',
    layout_mode VARCHAR(64) DEFAULT NULL COMMENT '表格布局模式：cross/column_pairs/row_pairs等',
    rows_count INT DEFAULT NULL COMMENT '默认行数',
    cols_count INT DEFAULT NULL COMMENT '默认列数',
    col_fields_json JSON DEFAULT NULL COMMENT '列字段标签数组JSON',
    row_fields_json JSON DEFAULT NULL COMMENT '行字段标签数组JSON',
    pair_label_cols_json JSON DEFAULT NULL COMMENT '成对列布局标签列索引数组JSON',
    pair_label_rows_json JSON DEFAULT NULL COMMENT '成对行布局标签行索引数组JSON',
    default_cell_values_json JSON DEFAULT NULL COMMENT '默认占位cell_values二维数组JSON',
    template VARCHAR(128) DEFAULT NULL COMMENT 'Go侧模板ID，可选',
    enabled TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
    sort_order INT NOT NULL DEFAULT 0 COMMENT '排序值',
    config_json JSON DEFAULT NULL COMMENT '保留扩展配置JSON',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (id),
    UNIQUE KEY uk_ue_chart_configs_chart_id (chart_id),
    KEY idx_ue_chart_configs_enabled (enabled),
    KEY idx_ue_chart_configs_panel_key (panel_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='UE chart_config图表静态布局配置表';
