from datetime import datetime

from sqlalchemy import JSON, DateTime, ForeignKey, String, Text, UniqueConstraint, func
from sqlalchemy.dialects.postgresql import JSONB
from sqlalchemy.orm import Mapped, mapped_column

from .db import Base

# jsonb on postgres, plain json elsewhere (sqlite tests)
JSONVariant = JSON().with_variant(JSONB(), "postgresql")


class User(Base):
    __tablename__ = "users"

    id: Mapped[int] = mapped_column(primary_key=True)
    email: Mapped[str] = mapped_column(String(255), unique=True)
    password_hash: Mapped[str] = mapped_column(String(255))
    role: Mapped[str] = mapped_column(String(50))
    is_active: Mapped[bool] = mapped_column(default=True)
    created_at: Mapped[datetime] = mapped_column(
        DateTime, server_default=func.current_timestamp()
    )


class Agent(Base):
    __tablename__ = "agents"

    instance_uid: Mapped[str] = mapped_column(String(255), primary_key=True)
    hostname: Mapped[str] = mapped_column(String(255))
    labels: Mapped[dict] = mapped_column(JSONVariant, default=dict)
    agent_type: Mapped[str | None] = mapped_column(String(100))
    version: Mapped[str | None] = mapped_column(String(100))
    status: Mapped[str] = mapped_column(String(50), default="disconnected")
    last_seen: Mapped[datetime | None] = mapped_column(DateTime)
    effective_config_hash: Mapped[str | None] = mapped_column(String(255))


class Configuration(Base):
    __tablename__ = "configurations"

    id: Mapped[int] = mapped_column(primary_key=True)
    name: Mapped[str] = mapped_column(String(255), unique=True)
    label_selector: Mapped[str | None] = mapped_column(String(255))
    # circular fk with config_versions, added via alter in schema.sql
    current_version_id: Mapped[int | None] = mapped_column(
        ForeignKey("config_versions.id", use_alter=True, name="fk_current_version")
    )


class ConfigVersion(Base):
    __tablename__ = "config_versions"
    __table_args__ = (UniqueConstraint("configuration_id", "version_no"),)

    id: Mapped[int] = mapped_column(primary_key=True)
    configuration_id: Mapped[int] = mapped_column(ForeignKey("configurations.id"))
    version_no: Mapped[int]
    yaml: Mapped[str] = mapped_column(Text)
    hash: Mapped[str] = mapped_column(String(255))
    author_id: Mapped[int] = mapped_column(ForeignKey("users.id"))
    created_at: Mapped[datetime] = mapped_column(
        DateTime, server_default=func.current_timestamp()
    )


class Rollout(Base):
    __tablename__ = "rollouts"

    id: Mapped[int] = mapped_column(primary_key=True)
    config_version_id: Mapped[int] = mapped_column(ForeignKey("config_versions.id"))
    agent_instance_uid: Mapped[str] = mapped_column(
        String(255), ForeignKey("agents.instance_uid")
    )
    status: Mapped[str] = mapped_column(String(50), default="pending")
    applied_at: Mapped[datetime | None] = mapped_column(DateTime)
    error: Mapped[str | None] = mapped_column(String(255))


class AuditLog(Base):
    __tablename__ = "audit_logs"

    id: Mapped[int] = mapped_column(primary_key=True)
    user_id: Mapped[int | None] = mapped_column(ForeignKey("users.id"))
    action: Mapped[str] = mapped_column(String(100))
    target_type: Mapped[str | None] = mapped_column(String(100))
    target_id: Mapped[str | None] = mapped_column(String(100))
    detail: Mapped[str | None] = mapped_column(Text)
    created_at: Mapped[datetime] = mapped_column(
        DateTime, server_default=func.current_timestamp()
    )
