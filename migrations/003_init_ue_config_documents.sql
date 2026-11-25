CREATE TABLE IF NOT EXISTS ue_config_documents (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    config_key VARCHAR(128) NOT NULL COMMENT '配置唯一key，如chart_config或ndisplay/cluster_1node_2viewport',
    file_name VARCHAR(255) NOT NULL COMMENT '原始配置文件名',
    content_json LONGTEXT NOT NULL COMMENT 'UE/FastAPI原始JSON配置内容',
    source VARCHAR(64) NOT NULL DEFAULT 'ue_config' COMMENT '配置来源标识',
    enabled TINYINT(1) NOT NULL DEFAULT 1 COMMENT '是否启用',
    version VARCHAR(64) NOT NULL DEFAULT 'v1' COMMENT '配置版本标识',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
    PRIMARY KEY (id),
    UNIQUE KEY uk_config_key (config_key),
    KEY idx_enabled (enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='UE/FastAPI原始配置JSON文档表';
