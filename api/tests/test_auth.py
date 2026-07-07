def test_login_success_returns_token_and_user(client, user_ids):
    resp = client.post(
        "/auth/login",
        json={"email": "admin@helmsman.local", "password": "admin123!"},
    )
    assert resp.status_code == 200
    body = resp.json()
    assert body["token_type"] == "bearer"
    assert isinstance(body["access_token"], str) and body["access_token"]
    assert body["user"] == {
        "id": user_ids["admin@helmsman.local"],
        "email": "admin@helmsman.local",
        "role": "admin",
    }


def test_login_wrong_password_401(client):
    resp = client.post(
        "/auth/login",
        json={"email": "admin@helmsman.local", "password": "wrong"},
    )
    assert resp.status_code == 401
    assert resp.json() == {"detail": "invalid email or password"}


def test_login_unknown_email_401(client):
    resp = client.post(
        "/auth/login",
        json={"email": "nobody@helmsman.local", "password": "admin123!"},
    )
    assert resp.status_code == 401
    assert resp.json() == {"detail": "invalid email or password"}


def test_login_inactive_user_401(client):
    resp = client.post(
        "/auth/login",
        json={"email": "inactive@helmsman.local", "password": "inactive123!"},
    )
    assert resp.status_code == 401
    assert resp.json() == {"detail": "invalid email or password"}


def test_me_returns_current_user(client, auth_headers, user_ids):
    resp = client.get("/auth/me", headers=auth_headers("operator@helmsman.local"))
    assert resp.status_code == 200
    assert resp.json() == {
        "id": user_ids["operator@helmsman.local"],
        "email": "operator@helmsman.local",
        "role": "operator",
    }


def test_me_without_token_401(client):
    resp = client.get("/auth/me")
    assert resp.status_code == 401


def test_me_with_garbage_token_401(client):
    resp = client.get(
        "/auth/me",
        headers={"Authorization": "Bearer not.a.real.jwt"},
    )
    assert resp.status_code == 401
