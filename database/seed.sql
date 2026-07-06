INSERT INTO users (email, password_hash, role, is_active)
VALUES
('admin@helmsman.local', 'fake_hash_1', 'admin', true),
('operator@helmsman.local', 'fake_hash_2', 'operator', true),
('viewer@helmsman.local', 'fake_hash_3', 'viewer', true);

INSERT INTO agents (instance_uid, hostname, labels, agent_type, version, status, last_seen, effective_config_hash)
VALUES
('agent-001', 'collector-prod-01', '{"env":"prod"}', 'otel-collector', '1.0.0', 'active', CURRENT_TIMESTAMP, 'hash-prod-v1'),
('agent-002', 'collector-dev-01', '{"env":"dev"}', 'otel-collector', '1.0.0', 'active', CURRENT_TIMESTAMP, 'hash-dev-v1');

INSERT INTO configurations (name, label_selector)
VALUES
('Production Collector Config', 'env=prod'),
('Development Collector Config', 'env=dev');

INSERT INTO config_versions (configuration_id, version_no, yaml, hash, author_id)
VALUES
(1, 1, 'receivers:\n  otlp:\nprocessors:\n  batch:', 'hash-prod-v1', 1),
(2, 1, 'receivers:\n  otlp:\nexporters:\n  debug:', 'hash-dev-v1', 2);

UPDATE configurations SET current_version_id = 1 WHERE id = 1;
UPDATE configurations SET current_version_id = 2 WHERE id = 2;

INSERT INTO rollouts (config_version_id, agent_instance_uid, status, applied_at, error)
VALUES
(1, 'agent-001', 'applied', CURRENT_TIMESTAMP, NULL),
(2, 'agent-002', 'pending', NULL, NULL);

INSERT INTO audit_logs (user_id, action, target_type, target_id, detail)
VALUES
(1, 'created configuration', 'configuration', '1', 'Created production collector config'),
(2, 'created rollout', 'rollout', '2', 'Started rollout for development config');