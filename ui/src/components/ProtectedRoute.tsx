import type { ReactNode } from 'react'
import { Navigate } from 'react-router-dom'
import { useAuth } from '../auth/useAuth'

export function ProtectedRoute({ children }: { children: ReactNode }) {
  const { token, loading } = useAuth()

  if (loading) {
    return <div className="center-note">loading...</div>
  }
  if (!token) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}
