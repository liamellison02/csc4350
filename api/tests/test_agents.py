AGENT_COLUMNS = {
    "instance_uid",
    "hostname",
    "labels",
    "agent_type",
    "version",
    "status",
    "last_seen",
    "effective_config_hash",
}


def test_agents_requires_auth(client, seeded_agents):
    resp = client.get("/agents")
    assert resp.status_code == 401


def test_agents_lists_seeded_agents(client, seeded_agents, auth_headers):
    resp = client.get("/agents", headers=auth_headers("viewer@helmsman.local"))
    assert resp.status_code == 200
    rows = resp.json()
    assert len(rows) == len(seeded_agents)
    for row in rows:
        assert set(row) == AGENT_COLUMNS
    by_uid = {row["instance_uid"]: row for row in rows}
    assert by_uid["agent-001"]["hostname"] == "collector-prod-01"
    assert by_uid["agent-001"]["status"] == "healthy"
    assert by_uid["agent-001"]["labels"] == {"env": "prod"}
    assert by_uid["agent-003"]["status"] == "disconnected"
    assert by_uid["agent-003"]["effective_config_hash"] is None
