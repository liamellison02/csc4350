from sqlalchemy import select

from app.models import AuditLog, Configuration


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
