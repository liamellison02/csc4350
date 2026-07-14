import hashlib

from sqlalchemy import select

from app.models import Agent, AuditLog, Configuration, ConfigVersion, Rollout


def test_create_configuration_as_operator_201_and_audit_row(
    client, auth_headers, db_session, user_ids
):
    resp = client.post(
        "/configurations",
        json={"name": "edge collector config", "label_selector": "env=edge"},
        headers=auth_headers("operator@helmsman.local"),
    )
    assert resp.status_code == 201
    body = resp.json()
    assert isinstance(body["id"], int)
    assert body["name"] == "edge collector config"
    assert body["label_selector"] == "env=edge"
    assert body["current_version_id"] is None

    audit_rows = db_session.scalars(select(AuditLog)).all()
    assert len(audit_rows) == 1
    row = audit_rows[0]
    assert row.action == "config change"
    assert row.target_type == "configuration"
    assert row.target_id == str(body["id"])
    assert row.user_id == user_ids["operator@helmsman.local"]


def test_create_configuration_as_viewer_403_and_denied_audit_row(
    client, auth_headers, db_session, user_ids
):
    resp = client.post(
        "/configurations",
        json={"name": "sneaky config", "label_selector": "env=prod"},
        headers=auth_headers("viewer@helmsman.local"),
    )
    assert resp.status_code == 403

    assert db_session.scalars(select(Configuration)).all() == []
    audit_rows = db_session.scalars(select(AuditLog)).all()
    assert len(audit_rows) == 1
    row = audit_rows[0]
    assert row.action == "denied attempt"
    assert row.user_id == user_ids["viewer@helmsman.local"]


def test_create_configuration_as_admin_201(client, auth_headers):
    resp = client.post(
        "/configurations",
        json={"name": "admin config", "label_selector": "env=dev"},
        headers=auth_headers("admin@helmsman.local"),
    )
    assert resp.status_code == 201
    assert resp.json()["name"] == "admin config"


def test_create_configuration_duplicate_name_409(
    client, auth_headers, db_session
):
    headers = auth_headers("operator@helmsman.local")
    first = client.post(
        "/configurations",
        json={"name": "dup config", "label_selector": None},
        headers=headers,
    )
    assert first.status_code == 201, first.text
    second = client.post(
        "/configurations",
        json={"name": "dup config", "label_selector": None},
        headers=headers,
    )
    assert second.status_code == 409
    assert second.json()["detail"] == "name already exists"

    rows = db_session.scalars(
        select(Configuration).where(Configuration.name == "dup config")
    ).all()
    assert len(rows) == 1


YAML_V1 = "receivers:\n  otlp:\n    protocols:\n      grpc:\n"
YAML_V2 = YAML_V1 + "exporters:\n  debug:\n"


def _make_config(client, auth_headers, name="cfg", selector=None):
    resp = client.post(
        "/configurations",
        json={"name": name, "label_selector": selector},
        headers=auth_headers("operator@helmsman.local"),
    )
    assert resp.status_code == 201, resp.text
    return resp.json()


def test_list_configurations_any_role(client, auth_headers):
    _make_config(client, auth_headers, name="cfg-a")
    resp = client.get(
        "/configurations", headers=auth_headers("viewer@helmsman.local")
    )
    assert resp.status_code == 200
    assert [c["name"] for c in resp.json()] == ["cfg-a"]


def test_get_configuration_404(client, auth_headers):
    resp = client.get(
        "/configurations/999", headers=auth_headers("viewer@helmsman.local")
    )
    assert resp.status_code == 404


def test_create_version_increments_and_sets_current(
    client, auth_headers, db_session
):
    config = _make_config(client, auth_headers)
    headers = auth_headers("operator@helmsman.local")

    r1 = client.post(
        f"/configurations/{config['id']}/versions",
        json={"yaml": YAML_V1},
        headers=headers,
    )
    assert r1.status_code == 201, r1.text
    v1 = r1.json()
    assert v1["version_no"] == 1
    assert v1["hash"] == hashlib.sha256(YAML_V1.encode()).hexdigest()

    r2 = client.post(
        f"/configurations/{config['id']}/versions",
        json={"yaml": YAML_V2},
        headers=headers,
    )
    v2 = r2.json()
    assert v2["version_no"] == 2

    current = client.get(
        f"/configurations/{config['id']}", headers=headers
    ).json()
    assert current["current_version_id"] == v2["id"]

    audit = db_session.scalars(
        select(AuditLog).where(AuditLog.action == "config change")
    ).all()
    # one row for the config create + one per version
    assert len(audit) == 3


def test_create_version_invalid_yaml_400(client, auth_headers):
    config = _make_config(client, auth_headers)
    resp = client.post(
        f"/configurations/{config['id']}/versions",
        json={"yaml": "receivers: [unclosed"},
        headers=auth_headers("operator@helmsman.local"),
    )
    assert resp.status_code == 400


def test_create_version_empty_yaml_400(client, auth_headers):
    config = _make_config(client, auth_headers)
    resp = client.post(
        f"/configurations/{config['id']}/versions",
        json={"yaml": "   \n"},
        headers=auth_headers("operator@helmsman.local"),
    )
    assert resp.status_code == 400


def test_create_version_as_viewer_403(client, auth_headers):
    config = _make_config(client, auth_headers)
    resp = client.post(
        f"/configurations/{config['id']}/versions",
        json={"yaml": YAML_V1},
        headers=auth_headers("viewer@helmsman.local"),
    )
    assert resp.status_code == 403


def test_versions_list_newest_first(client, auth_headers):
    config = _make_config(client, auth_headers)
    headers = auth_headers("operator@helmsman.local")
    for body in (YAML_V1, YAML_V2):
        client.post(
            f"/configurations/{config['id']}/versions",
            json={"yaml": body},
            headers=headers,
        )
    resp = client.get(
        f"/configurations/{config['id']}/versions",
        headers=auth_headers("viewer@helmsman.local"),
    )
    assert [v["version_no"] for v in resp.json()] == [2, 1]


def test_rollback_flips_pointer_and_audits(client, auth_headers, db_session):
    config = _make_config(client, auth_headers)
    headers = auth_headers("operator@helmsman.local")
    v1 = client.post(
        f"/configurations/{config['id']}/versions",
        json={"yaml": YAML_V1},
        headers=headers,
    ).json()
    client.post(
        f"/configurations/{config['id']}/versions",
        json={"yaml": YAML_V2},
        headers=headers,
    )

    resp = client.post(
        f"/configurations/{config['id']}/rollback",
        json={"version_id": v1["id"]},
        headers=headers,
    )
    assert resp.status_code == 200
    assert resp.json()["current_version_id"] == v1["id"]

    row = db_session.scalars(
        select(AuditLog).where(AuditLog.action == "rollback")
    ).one()
    assert row.target_id == str(config["id"])

    # no new version rows were created by the rollback
    versions = db_session.scalars(select(ConfigVersion)).all()
    assert len(versions) == 2


def test_rollback_current_version_400(client, auth_headers):
    config = _make_config(client, auth_headers)
    headers = auth_headers("operator@helmsman.local")
    v1 = client.post(
        f"/configurations/{config['id']}/versions",
        json={"yaml": YAML_V1},
        headers=headers,
    ).json()
    resp = client.post(
        f"/configurations/{config['id']}/rollback",
        json={"version_id": v1["id"]},
        headers=headers,
    )
    assert resp.status_code == 400


def test_rollback_foreign_version_400(client, auth_headers):
    config_a = _make_config(client, auth_headers, name="cfg-a")
    config_b = _make_config(client, auth_headers, name="cfg-b")
    headers = auth_headers("operator@helmsman.local")
    v_a = client.post(
        f"/configurations/{config_a['id']}/versions",
        json={"yaml": YAML_V1},
        headers=headers,
    ).json()
    resp = client.post(
        f"/configurations/{config_b['id']}/rollback",
        json={"version_id": v_a["id"]},
        headers=headers,
    )
    assert resp.status_code == 400


def test_rollback_as_viewer_403(client, auth_headers):
    config = _make_config(client, auth_headers)
    headers = auth_headers("operator@helmsman.local")
    v1 = client.post(
        f"/configurations/{config['id']}/versions",
        json={"yaml": YAML_V1},
        headers=headers,
    ).json()
    client.post(
        f"/configurations/{config['id']}/versions",
        json={"yaml": YAML_V2},
        headers=headers,
    )
    before = client.get(
        f"/configurations/{config['id']}", headers=headers
    ).json()["current_version_id"]

    resp = client.post(
        f"/configurations/{config['id']}/rollback",
        json={"version_id": v1["id"]},
        headers=auth_headers("viewer@helmsman.local"),
    )
    assert resp.status_code == 403

    # the rollback was denied, so the current pointer did not move
    after = client.get(
        f"/configurations/{config['id']}", headers=headers
    ).json()["current_version_id"]
    assert after == before


def test_rollouts_listing_newest_first(client, auth_headers, db_session):
    config = _make_config(client, auth_headers)
    headers = auth_headers("operator@helmsman.local")
    v1 = client.post(
        f"/configurations/{config['id']}/versions",
        json={"yaml": YAML_V1},
        headers=headers,
    ).json()
    db_session.add(
        Agent(instance_uid="agent-e2e", hostname="h", labels={})
    )
    db_session.add_all(
        [
            Rollout(config_version_id=v1["id"], agent_instance_uid="agent-e2e", status="applied"),
            Rollout(config_version_id=v1["id"], agent_instance_uid="agent-e2e", status="pending"),
        ]
    )
    db_session.commit()

    resp = client.get(
        f"/configurations/{config['id']}/rollouts",
        headers=auth_headers("viewer@helmsman.local"),
    )
    assert resp.status_code == 200
    statuses = [r["status"] for r in resp.json()]
    assert statuses == ["pending", "applied"]
