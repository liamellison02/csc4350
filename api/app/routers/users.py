from fastapi import APIRouter, Depends, HTTPException, status
from sqlalchemy import select
from sqlalchemy.orm import Session

from ..deps import get_db, require_role
from ..models import AuditLog, User
from ..schemas import UserAdminOut, UserCreate, UserPatch
from ..security import hash_password

router = APIRouter(prefix="/users", tags=["users"])


@router.get("", response_model=list[UserAdminOut])
def list_users(
    user: User = Depends(require_role("admin")),
    db: Session = Depends(get_db),
) -> list[User]:
    return list(db.scalars(select(User).order_by(User.id)).all())


@router.post("", response_model=UserAdminOut, status_code=status.HTTP_201_CREATED)
def create_user(
    body: UserCreate,
    user: User = Depends(require_role("admin")),
    db: Session = Depends(get_db),
) -> User:
    if db.scalar(select(User).where(User.email == body.email)) is not None:
        raise HTTPException(
            status_code=status.HTTP_409_CONFLICT,
            detail="email already exists",
        )
    created = User(
        email=body.email,
        password_hash=hash_password(body.password),
        role=body.role,
        is_active=True,
    )
    db.add(created)
    db.flush()
    db.add(
        AuditLog(
            user_id=user.id,
            action="user admin",
            target_type="user",
            target_id=str(created.id),
            detail=f"created user {created.email} with role {created.role}",
        )
    )
    db.commit()
    db.refresh(created)
    return created


@router.patch("/{user_id}", response_model=UserAdminOut)
def update_user(
    user_id: int,
    body: UserPatch,
    user: User = Depends(require_role("admin")),
    db: Session = Depends(get_db),
) -> User:
    target = db.get(User, user_id)
    if target is None:
        raise HTTPException(
            status_code=status.HTTP_404_NOT_FOUND, detail="user not found"
        )
    if body.is_active is False and target.id == user.id:
        raise HTTPException(
            status_code=status.HTTP_400_BAD_REQUEST,
            detail="cannot deactivate yourself",
        )
    changes = []
    if body.role is not None and body.role != target.role:
        changes.append(f"role {target.role} -> {body.role}")
        target.role = body.role
    if body.is_active is not None and body.is_active != target.is_active:
        changes.append(f"is_active {target.is_active} -> {body.is_active}")
        target.is_active = body.is_active
    if changes:
        db.add(
            AuditLog(
                user_id=user.id,
                action="user admin",
                target_type="user",
                target_id=str(target.id),
                detail="; ".join(changes),
            )
        )
    db.commit()
    db.refresh(target)
    return target
