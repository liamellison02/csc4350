import hashlib

import yaml as yaml_parser
from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy import func, select
from sqlalchemy.exc import IntegrityError
from sqlalchemy.orm import Session

from ..deps import get_current_user, get_db, require_role
from ..models import AuditLog, Configuration, ConfigVersion, Rollout, User
from ..schemas import (
    ConfigurationCreate,
    ConfigurationOut,
    ConfigVersionCreate,
    ConfigVersionOut,
    RollbackRequest,
    RolloutOut,
)

router = APIRouter(tags=["configurations"])


def _config_or_404(db: Session, config_id: int) -> Configuration:
    config = db.get(Configuration, config_id)
    if config is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND,
            detail="configuration not found",
        )
    return config


@router.get("/configurations", response_model=list[ConfigurationOut])
def list_configurations(
    user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
) -> list[Configuration]:
    return list(
        db.scalars(select(Configuration).order_by(Configuration.id)).all()
    )


@router.get("/configurations/{config_id}", response_model=ConfigurationOut)
def get_configuration(
    config_id: int,
    user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
) -> Configuration:
    return _config_or_404(db, config_id)


@router.post(
    "/configurations",
    response_model=ConfigurationOut,
    status_code=status.HTTP_201_CREATED,
)
def create_configuration(
    body: ConfigurationCreate,
    user: User = Depends(require_role("operator")),
    db: Session = Depends(get_db),
) -> Configuration:
    config = Configuration(name=body.name, label_selector=body.label_selector)
    db.add(config)
    # unique name can collide; surface it as a 409 like users.py
    try:
        db.flush()
    except IntegrityError:
        db.rollback()
        raise HTTPException(
            status_code=status.HTTP_409_CONFLICT,
            detail="name already exists",
        ) from None
    db.add(
        AuditLog(
            user_id=user.id,
            action="config change",
            target_type="configuration",
            target_id=str(config.id),
            detail=f"created configuration {config.name}",
        )
    )
    db.commit()
    db.refresh(config)
    return config


@router.get(
    "/configurations/{config_id}/versions",
    response_model=list[ConfigVersionOut],
)
def list_versions(
    config_id: int,
    user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
) -> list[ConfigVersion]:
    _config_or_404(db, config_id)
    return list(
        db.scalars(
            select(ConfigVersion)
            .where(ConfigVersion.configuration_id == config_id)
            .order_by(ConfigVersion.version_no.desc())
        ).all()
    )


@router.post(
    "/configurations/{config_id}/versions",
    response_model=ConfigVersionOut,
    status_code=status.HTTP_201_CREATED,
)
def create_version(
    config_id: int,
    body: ConfigVersionCreate,
    user: User = Depends(require_role("operator")),
    db: Session = Depends(get_db),
) -> ConfigVersion:
    config = _config_or_404(db, config_id)
    try:
        parsed = yaml_parser.safe_load(body.yaml)
    except yaml_parser.YAMLError:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST, detail="invalid yaml"
        ) from None
    if parsed is None:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="empty configuration",
        )

    digest = hashlib.sha256(body.yaml.encode()).hexdigest()
    version = None
    # unique (configuration_id, version_no) can race; retry once
    for attempt in (1, 2):
        next_no = (
            db.scalar(
                select(func.max(ConfigVersion.version_no)).where(
                    ConfigVersion.configuration_id == config_id
                )
            )
            or 0
        ) + 1
        version = ConfigVersion(
            configuration_id=config_id,
            version_no=next_no,
            yaml=body.yaml,
            hash=digest,
            author_id=user.id,
        )
        db.add(version)
        try:
            db.flush()
            break
        except IntegrityError:
            db.rollback()
            if attempt == 2:
                raise HTTPException(
                    status_code=status.HTTP_409_CONFLICT,
                    detail="concurrent version creation, retry",
                ) from None
            config = _config_or_404(db, config_id)

    config.current_version_id = version.id
    db.add(
        AuditLog(
            user_id=user.id,
            action="config change",
            target_type="config_version",
            target_id=str(version.id),
            detail=(
                f"created version {version.version_no} "
                f"of configuration {config.name}"
            ),
        )
    )
    db.commit()
    db.refresh(version)
    return version


@router.post(
    "/configurations/{config_id}/rollback",
    response_model=ConfigurationOut,
)
def rollback_configuration(
    config_id: int,
    body: RollbackRequest,
    user: User = Depends(require_role("operator")),
    db: Session = Depends(get_db),
) -> Configuration:
    config = _config_or_404(db, config_id)
    version = db.get(ConfigVersion, body.version_id)
    if version is None or version.configuration_id != config_id:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="version does not belong to this configuration",
        )
    if config.current_version_id == version.id:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="version is already current",
        )
    config.current_version_id = version.id
    db.add(
        AuditLog(
            user_id=user.id,
            action="rollback",
            target_type="configuration",
            target_id=str(config.id),
            detail=(
                f"rolled back {config.name} "
                f"to version {version.version_no}"
            ),
        )
    )
    db.commit()
    db.refresh(config)
    return config


@router.get(
    "/configurations/{config_id}/rollouts",
    response_model=list[RolloutOut],
)
def list_rollouts(
    config_id: int,
    user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
) -> list[Rollout]:
    _config_or_404(db, config_id)
    return list(
        db.scalars(
            select(Rollout)
            .join(ConfigVersion, Rollout.config_version_id == ConfigVersion.id)
            .where(ConfigVersion.configuration_id == config_id)
            .order_by(Rollout.id.desc())
        ).all()
    )
