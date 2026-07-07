import bcrypt
import pytest
from fastapi.testclient import TestClient
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker
from sqlalchemy.pool import StaticPool

from app.db import Base
from app.deps import get_db
from app.main import create_app
from app.models import Agent, User

PASSWORDS = {
    "admin@helmsman.local": "admin123!",
    "operator@helmsman.local": "operator123!",
    "viewer@helmsman.local": "viewer123!",
    "inactive@helmsman.local": "inactive123!",
}

# hash once per test session, low rounds for speed
HASHES = {
    email: bcrypt.hashpw(pw.encode(), bcrypt.gensalt(rounds=4)).decode()
    for email, pw in PASSWORDS.items()
}

USERS = [
    ("admin@helmsman.local", "admin", True),
    ("operator@helmsman.local", "operator", True),
    ("viewer@helmsman.local", "viewer", True),
    ("inactive@helmsman.local", "viewer", False),
]

AGENTS = [
    {
        "instance_uid": "agent-001",
        "hostname": "collector-prod-01",
        "labels": {"env": "prod"},
        "agent_type": "otel-collector",
        "version": "1.0.0",
        "status": "healthy",
        "effective_config_hash": "hash-prod-v1",
    },
    {
        "instance_uid": "agent-003",
        "hostname": "collector-edge-01",
        "labels": {"env": "edge"},
        "agent_type": "otel-collector",
        "version": "0.9.0",
        "status": "disconnected",
        "effective_config_hash": None,
    },
]


@pytest.fixture()
def engine():
    engine = create_engine(
        "sqlite://",
        connect_args={"check_same_thread": False},
        poolclass=StaticPool,
    )
    Base.metadata.create_all(engine)
    yield engine
    engine.dispose()


@pytest.fixture()
def session_factory(engine):
    return sessionmaker(bind=engine, autoflush=False, expire_on_commit=False)


@pytest.fixture()
def db_session(session_factory):
    session = session_factory()
    yield session
    session.close()


@pytest.fixture()
def user_ids(session_factory):
    with session_factory() as session:
        users = [
            User(email=email, password_hash=HASHES[email], role=role, is_active=active)
            for email, role, active in USERS
        ]
        session.add_all(users)
        session.commit()
        return {user.email: user.id for user in users}


@pytest.fixture()
def seeded_agents(session_factory):
    with session_factory() as session:
        session.add_all([Agent(**row) for row in AGENTS])
        session.commit()
    return AGENTS


@pytest.fixture()
def client(session_factory, user_ids):
    app = create_app()

    def override_get_db():
        db = session_factory()
        try:
            yield db
        finally:
            db.close()

    app.dependency_overrides[get_db] = override_get_db
    with TestClient(app) as test_client:
        yield test_client


@pytest.fixture()
def auth_headers(client):
    def _headers(email):
        resp = client.post(
            "/auth/login",
            json={"email": email, "password": PASSWORDS[email]},
        )
        assert resp.status_code == 200, resp.text
        return {"Authorization": f"Bearer {resp.json()['access_token']}"}

    return _headers
