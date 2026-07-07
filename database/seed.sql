-- helmsman dev seed data
-- demo credentials (bcrypt hashed below):
--   admin@helmsman.local / admin123!
--   operator@helmsman.local / operator123!
--   viewer@helmsman.local / viewer123!

INSERT INTO users (email, password_hash, role, is_active)
VALUES
('admin@helmsman.local', '$2b$12$rdaMdTIXmwLfJBjxU4Knn.1Cndvo9NOyS5enhbVmwZAz7GYa8OLUa', 'admin', true),
('operator@helmsman.local', '$2b$12$MTiIqOjbEOPGRmcWUzSl1eV6dQ6XmuDLUK1JrRELk3KN2neyS.a5C', 'operator', true),
('viewer@helmsman.local', '$2b$12$D3zw.xURVScrf.p7csYwquvW3hcBOqlKm21rA/MjKkSPlcB71ZK7W', 'viewer', true);

INSERT INTO agents (instance_uid, hostname, labels, agent_type, version, status, last_seen, effective_config_hash)
VALUES
('agent-001', 'collector-prod-01', '{"env": "prod"}', 'otel-collector', '1.0.0', 'healthy', CURRENT_TIMESTAMP, 'hash-prod-v1'),
('agent-002', 'collector-dev-01', '{"env": "dev"}', 'otel-collector', '1.0.0', 'healthy', CURRENT_TIMESTAMP, 'hash-dev-v1'),
('agent-003', 'collector-edge-01', '{"env": "edge"}', 'otel-collector', '0.9.0', 'disconnected', CURRENT_TIMESTAMP - INTERVAL '2 days', NULL);

INSERT INTO configurations (name, label_selector)
VALUES
('Production Collector Config', 'env=prod'),
('Development Collector Config', 'env=dev');

INSERT INTO config_versions (configuration_id, version_no, yaml, hash, author_id)
VALUES
(1, 1, $$receivers:
  otlp:
    protocols:
      grpc:
      http:
processors:
  batch:
exporters:
  otlp:
    endpoint: backend.prod.local:4317
service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlp]
$$, 'hash-prod-v1', 1),
(2, 1, $$receivers:
  otlp:
    protocols:
      grpc:
processors:
  batch:
exporters:
  debug:
service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [batch]
      exporters: [debug]
$$, 'hash-dev-v1', 2);

UPDATE configurations SET current_version_id = 1 WHERE id = 1;
UPDATE configurations SET current_version_id = 2 WHERE id = 2;

INSERT INTO rollouts (config_version_id, agent_instance_uid, status, applied_at, error)
VALUES
(1, 'agent-001', 'applied', CURRENT_TIMESTAMP, NULL),
(2, 'agent-002', 'pending', NULL, NULL);

INSERT INTO audit_logs (user_id, action, target_type, target_id, detail)
VALUES
(1, 'config change', 'configuration', '1', 'created production collector config'),
(2, 'rollout', 'rollout', '2', 'started rollout for development config');
