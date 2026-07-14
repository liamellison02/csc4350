from datetime import datetime

from pydantic import BaseModel, ConfigDict


class LoginRequest(BaseModel):
    email: str
    password: str


class UserOut(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: int
    email: str
    role: str


class LoginResponse(BaseModel):
    access_token: str
    token_type: str = "bearer"
    user: UserOut


class AgentOut(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    instance_uid: str
    hostname: str
    labels: dict
    agent_type: str | None
    version: str | None
    status: str
    last_seen: datetime | None
    effective_config_hash: str | None


class ConfigurationCreate(BaseModel):
    name: str
    label_selector: str | None = None


class ConfigurationOut(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: int
    name: str
    label_selector: str | None
    current_version_id: int | None


class ConfigVersionCreate(BaseModel):
    yaml: str


class ConfigVersionOut(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: int
    configuration_id: int
    version_no: int
    yaml: str
    hash: str
    author_id: int
    created_at: datetime


class RollbackRequest(BaseModel):
    version_id: int


class RolloutOut(BaseModel):
    model_config = ConfigDict(from_attributes=True)

    id: int
    config_version_id: int
    agent_instance_uid: str
    status: str
    applied_at: datetime | None
    error: str | None
