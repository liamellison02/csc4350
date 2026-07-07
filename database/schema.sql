-- helmsman postgres schema
-- source of truth: docs/submissions/data-model.md (assignment 4 erd)
-- dev bootstrap: dropped and recreated on every container init

DROP TABLE IF EXISTS audit_logs, rollouts, config_versions, configurations, agents, users CASCADE;

CREATE TABLE users (
  id SERIAL PRIMARY KEY,
  email VARCHAR(255) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  role VARCHAR(50) NOT NULL CHECK (role IN ('admin', 'operator', 'viewer')),
  is_active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE agents (
  instance_uid VARCHAR(255) PRIMARY KEY,
  hostname VARCHAR(255) NOT NULL,
  labels JSONB NOT NULL DEFAULT '{}',
  agent_type VARCHAR(100),
  version VARCHAR(100),
  status VARCHAR(50) NOT NULL DEFAULT 'disconnected'
    CHECK (status IN ('healthy', 'degraded', 'disconnected')),
  last_seen TIMESTAMP,
  effective_config_hash VARCHAR(255)
);

CREATE TABLE configurations (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL UNIQUE,
  label_selector VARCHAR(255),
  current_version_id INT
);

CREATE TABLE config_versions (
  id SERIAL PRIMARY KEY,
  configuration_id INT NOT NULL REFERENCES configurations(id),
  version_no INT NOT NULL,
  yaml TEXT NOT NULL,
  hash VARCHAR(255) NOT NULL,
  author_id INT NOT NULL REFERENCES users(id),
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (configuration_id, version_no)
);

ALTER TABLE configurations
  ADD CONSTRAINT fk_current_version
  FOREIGN KEY (current_version_id) REFERENCES config_versions(id);

CREATE TABLE rollouts (
  id SERIAL PRIMARY KEY,
  config_version_id INT NOT NULL REFERENCES config_versions(id),
  agent_instance_uid VARCHAR(255) NOT NULL REFERENCES agents(instance_uid),
  status VARCHAR(50) NOT NULL DEFAULT 'pending'
    CHECK (status IN ('pending', 'applied', 'failed')),
  applied_at TIMESTAMP,
  error VARCHAR(255)
);

CREATE TABLE audit_logs (
  id SERIAL PRIMARY KEY,
  user_id INT REFERENCES users(id),
  action VARCHAR(100) NOT NULL,
  target_type VARCHAR(100),
  target_id VARCHAR(100),
  detail TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_config_versions_configuration ON config_versions (configuration_id);
CREATE INDEX idx_rollouts_version ON rollouts (config_version_id);
CREATE INDEX idx_rollouts_agent ON rollouts (agent_instance_uid);
CREATE INDEX idx_audit_logs_user ON audit_logs (user_id);
CREATE INDEX idx_audit_logs_created ON audit_logs (created_at);
