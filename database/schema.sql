DROP TABLE IF EXISTS audit_logs, rollouts, config_versions, configurations, agents, users CASCADE;

CREATE TABLE users (
  id SERIAL PRIMARY KEY,
  email VARCHAR(255) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  role VARCHAR(50) NOT NULL,
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE agents (
  instance_uid VARCHAR(255) PRIMARY KEY,
  hostname VARCHAR(255) NOT NULL,
  labels JSONB,
  agent_type VARCHAR(100),
  version VARCHAR(100),
  status VARCHAR(50),
  last_seen TIMESTAMP,
  effective_config_hash VARCHAR(255)
);

CREATE TABLE configurations (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  label_selector VARCHAR(255),
  current_version_id INT
);

CREATE TABLE config_versions (
  id SERIAL PRIMARY KEY,
  configuration_id INT NOT NULL REFERENCES configurations(id),
  version_no INT NOT NULL,
  yaml TEXT NOT NULL,
  hash VARCHAR(255),
  author_id INT REFERENCES users(id),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE configurations
ADD CONSTRAINT fk_current_version
FOREIGN KEY (current_version_id)
REFERENCES config_versions(id);

CREATE TABLE rollouts (
  id SERIAL PRIMARY KEY,
  config_version_id INT NOT NULL REFERENCES config_versions(id),
  agent_instance_uid VARCHAR(255) NOT NULL REFERENCES agents(instance_uid),
  status VARCHAR(50),
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
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);