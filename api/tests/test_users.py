from sqlalchemy import select

from app.models import AuditLog


def test_list_users_admin_only(client, auth_headers):
    resp = client.get("/users", headers=auth_headers("admin@helmsman.local"))
    assert resp.status_code == 200
    emails = [u["email"] for u in resp.json()]
    assert "operator@helmsman.local" in emails
    assert all("password" not in u for u in resp.json())

    denied = client.get(
        "/users", headers=auth_headers("operator@helmsman.local")
    )
    assert denied.status_code == 403


def test_create_user_and_login(client, auth_headers, db_session):
    resp = client.post(
        "/users",
        json={
            "email": "new@helmsman.local",
            "password": "newpass123!",
            "role": "viewer",
        },
        headers=auth_headers("admin@helmsman.local"),
    )
    assert resp.status_code == 201, resp.text
    body = resp.json()
    assert body["role"] == "viewer"
    assert body["is_active"] is True

    login = client.post(
        "/auth/login",
        json={"email": "new@helmsman.local", "password": "newpass123!"},
    )
    assert login.status_code == 200

    row = db_session.scalars(
        select(AuditLog).where(AuditLog.action == "user admin")
    ).one()
    assert row.target_id == str(body["id"])


def test_create_user_duplicate_email_409(client, auth_headers):
    resp = client.post(
        "/users",
        json={
            "email": "viewer@helmsman.local",
            "password": "whatever123!",
            "role": "viewer",
        },
        headers=auth_headers("admin@helmsman.local"),
    )
    assert resp.status_code == 409


def test_create_user_bad_role_422(client, auth_headers):
    resp = client.post(
        "/users",
        json={
            "email": "x@helmsman.local",
            "password": "whatever123!",
            "role": "superadmin",
        },
        headers=auth_headers("admin@helmsman.local"),
    )
    assert resp.status_code == 422


def test_patch_role_and_deactivate(client, auth_headers, user_ids, db_session):
    target = user_ids["viewer@helmsman.local"]
    resp = client.patch(
        f"/users/{target}",
        json={"role": "operator", "is_active": False},
        headers=auth_headers("admin@helmsman.local"),
    )
    assert resp.status_code == 200
    assert resp.json()["role"] == "operator"
    assert resp.json()["is_active"] is False

    audit = db_session.scalars(
        select(AuditLog).where(AuditLog.action == "user admin")
    ).all()
    assert len(audit) == 1


def test_patch_self_deactivation_400(client, auth_headers, user_ids):
    admin_id = user_ids["admin@helmsman.local"]
    resp = client.patch(
        f"/users/{admin_id}",
        json={"is_active": False},
        headers=auth_headers("admin@helmsman.local"),
    )
    assert resp.status_code == 400


def test_patch_unknown_user_404(client, auth_headers):
    resp = client.patch(
        "/users/999",
        json={"role": "viewer"},
        headers=auth_headers("admin@helmsman.local"),
    )
    assert resp.status_code == 404
