-- P1-1 auth users table.
-- Stores long-lived gateway user accounts. Redis stores short-lived sessions.

CREATE TABLE IF NOT EXISTS ue_users (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT 'Primary key',
  username VARCHAR(64) NOT NULL COMMENT 'Login username',
  password_hash VARCHAR(255) NOT NULL COMMENT 'bcrypt password hash',
  role VARCHAR(32) NOT NULL DEFAULT 'operator' COMMENT 'User role, e.g. operator/admin',
  enabled TINYINT(1) NOT NULL DEFAULT 1 COMMENT 'Whether the user can login',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT 'Creation time',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Update time',
  PRIMARY KEY (id),
  UNIQUE KEY uk_username (username),
  KEY idx_enabled (enabled),
  KEY idx_role (role)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Gateway auth users';
