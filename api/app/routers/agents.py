from fastapi import APIRouter, Depends
from sqlalchemy import select
from sqlalchemy.orm import Session

from ..deps import get_current_user, get_db
from ..models import Agent, User
from ..schemas import AgentOut

router = APIRouter(tags=["agents"])


@router.get("/agents", response_model=list[AgentOut])
def list_agents(
    user: User = Depends(get_current_user),
    db: Session = Depends(get_db),
) -> list[Agent]:
    return list(db.scalars(select(Agent)).all())
