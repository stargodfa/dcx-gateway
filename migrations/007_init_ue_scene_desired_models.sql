-- Init structured scene_desired_config model instances.
-- Each row corresponds to one scene_models_desired[] item from scene_desired_config.json.

CREATE TABLE IF NOT EXISTS ue_scene_desired_models (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'Primary key',

  instance_id VARCHAR(128) NOT NULL COMMENT 'Unique model instance id referenced by init_config.instance_ids',
  model_id VARCHAR(128) NOT NULL COMMENT 'Logical model id shared by instances',

  uri VARCHAR(1024) NOT NULL COMMENT 'UE model resource URI',
  format VARCHAR(64) NOT NULL DEFAULT 'udatasmith' COMMENT 'Model format',
  source_path VARCHAR(1024) NOT NULL COMMENT 'Original source path',

  display_name VARCHAR(255) NOT NULL COMMENT 'Display name',
  visible TINYINT(1) NOT NULL DEFAULT 1 COMMENT 'Whether the instance should be visible',

  bounds_x DOUBLE NULL COMMENT 'Bounds size X',
  bounds_y DOUBLE NULL COMMENT 'Bounds size Y',
  bounds_z DOUBLE NULL COMMENT 'Bounds size Z',

  position_x DOUBLE NOT NULL DEFAULT 0 COMMENT 'Transform position X',
  position_y DOUBLE NOT NULL DEFAULT 0 COMMENT 'Transform position Y',
  position_z DOUBLE NOT NULL DEFAULT 0 COMMENT 'Transform position Z',

  rotation_yaw DOUBLE NOT NULL DEFAULT 0 COMMENT 'Transform rotation yaw',
  rotation_pitch DOUBLE NOT NULL DEFAULT 0 COMMENT 'Transform rotation pitch',
  rotation_roll DOUBLE NOT NULL DEFAULT 0 COMMENT 'Transform rotation roll',

  scale_x DOUBLE NOT NULL DEFAULT 1 COMMENT 'Transform scale X',
  scale_y DOUBLE NOT NULL DEFAULT 1 COMMENT 'Transform scale Y',
  scale_z DOUBLE NOT NULL DEFAULT 1 COMMENT 'Transform scale Z',

  enabled TINYINT(1) NOT NULL DEFAULT 1 COMMENT 'Soft delete flag',
  sort_order INT NOT NULL DEFAULT 0 COMMENT 'Output order',
  config_json LONGTEXT NULL COMMENT 'Extra unknown JSON fields for this model instance',

  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Created time',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Updated time',

  PRIMARY KEY (id),
  UNIQUE KEY uk_instance_id (instance_id),
  KEY idx_model_id (model_id),
  KEY idx_enabled (enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Structured UE scene_desired_config model instances';
