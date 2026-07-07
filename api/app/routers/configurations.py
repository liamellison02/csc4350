from fastapi import APIRouter, Depends, status
from sqlalchemy.orm import Session

from ..deps import get_db, require_role
from ..models import AuditLog, Configuration, User
from ..schemas import ConfigurationCreate, ConfigurationOut

router = APIRouter(tags=["configurations"])


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
    db.flush()
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
