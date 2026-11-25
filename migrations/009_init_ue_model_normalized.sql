-- Init normalized UE model config tables.
-- model_id / instance_id / placement are split so scene_desired_config can be generated.

CREATE TABLE IF NOT EXISTS ue_model_types (
  model_id VARCHAR(128) NOT NULL COMMENT 'Logical model type id and primary key',
  uri VARCHAR(1024) NOT NULL COMMENT 'UE model resource URI',
  format VARCHAR(64) NOT NULL DEFAULT 'udatasmith' COMMENT 'Model format',
  source_path VARCHAR(1024) NOT NULL COMMENT 'Original source path',
  base_bounds_x DOUBLE NOT NULL COMMENT 'Scale=1 base bounds size X',
  base_bounds_y DOUBLE NOT NULL COMMENT 'Scale=1 base bounds size Y',
  base_bounds_z DOUBLE NOT NULL COMMENT 'Scale=1 base bounds size Z',
  bounds_negative_extent_axis INT NULL COMMENT 'Optional bounds negative extent axis: 0=X, 1=Y, 2=Z',
  enabled TINYINT(1) NOT NULL DEFAULT 1 COMMENT 'Soft delete flag',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Created time',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Updated time',
  PRIMARY KEY (model_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='UE model type asset metadata';

CREATE TABLE IF NOT EXISTS ue_model_instances (
  instance_id VARCHAR(128) NOT NULL COMMENT 'Model instance id and primary key',
  model_id VARCHAR(128) NOT NULL COMMENT 'Referenced ue_model_types.model_id',
  display_name VARCHAR(255) NOT NULL DEFAULT '' COMMENT 'Display name for Web/FastAPI catalog',
  sort_order INT NOT NULL DEFAULT 0 COMMENT 'Stable output order',
  enabled TINYINT(1) NOT NULL DEFAULT 1 COMMENT 'Soft delete flag',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Created time',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Updated time',
  PRIMARY KEY (instance_id),
  KEY idx_model_id (model_id),
  KEY idx_enabled (enabled)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='UE model instance catalog entries';

CREATE TABLE IF NOT EXISTS ue_model_instance_placements (
  instance_id VARCHAR(128) NOT NULL COMMENT 'Model instance id and primary key',
  visible TINYINT(1) NOT NULL DEFAULT 1 COMMENT 'Whether the instance should be visible',
  position_x DOUBLE NOT NULL DEFAULT 0 COMMENT 'Transform position X',
  position_y DOUBLE NOT NULL DEFAULT 0 COMMENT 'Transform position Y',
  position_z DOUBLE NOT NULL DEFAULT 0 COMMENT 'Transform position Z',
  rotation_yaw DOUBLE NOT NULL DEFAULT 0 COMMENT 'Transform rotation yaw',
  rotation_pitch DOUBLE NOT NULL DEFAULT 0 COMMENT 'Transform rotation pitch',
  rotation_roll DOUBLE NOT NULL DEFAULT 0 COMMENT 'Transform rotation roll',
  scale_x DOUBLE NOT NULL DEFAULT 1 COMMENT 'Transform scale X',
  scale_y DOUBLE NOT NULL DEFAULT 1 COMMENT 'Transform scale Y',
  scale_z DOUBLE NOT NULL DEFAULT 1 COMMENT 'Transform scale Z',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Created time',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Updated time',
  PRIMARY KEY (instance_id),
  KEY idx_visible (visible)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='UE model instance placement and visibility';