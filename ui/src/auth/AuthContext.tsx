import { useCallback, useEffect, useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import type { User } from '../lib/api'
import { ApiError, getMe, login as apiLogin } from '../lib/api'
import { AuthContext, TOKEN_KEY } from './context'

export function AuthProvider({ children }: { children: ReactNode }) {
  const [token, setToken] = useState<string | null>(() =>
    localStorage.getItem(TOKEN_KEY),
  )
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState<boolean>(
    () => localStorage.getItem(TOKEN_KEY) !== null,
  )

  // on mount, if a token was persisted, resolve the current user.
  // a 401 means the token is stale, so drop it.
  useEffect(() => {
    const stored = localStorage.getItem(TOKEN_KEY)
    // no persisted token: initial loading state is already false
    if (!stored) {
      return
    }
    let active = true
    getMe(stored)
      .then((me) => {
        if (active) setUser(me)
      })
      .catch((err) => {
        if (!active) return
        if (err instanceof ApiError && err.status === 401) {
          localStorage.removeItem(TOKEN_KEY)
          setToken(null)
        }
        setUser(null)
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => {
      active = false
    }
  }, [])

  const login = useCallback(async (email: string, password: string) => {
    const res = await apiLogin(email, password)
    // confirm the token by resolving the user before we persist it
    try {
      const me = await getMe(res.access_token)
      localStorage.setItem(TOKEN_KEY, res.access_token)
      setToken(res.access_token)
      setUser(me)
    } catch (err) {
      localStorage.removeItem(TOKEN_KEY)
      setToken(null)
      setUser(null)
      throw err
    }
  }, [])

  const logout = useCallback(() => {
    localStorage.removeItem(TOKEN_KEY)
    setToken(null)
    setUser(null)
  }, [])

  const value = useMemo(
    () => ({ user, token, loading, login, logout }),
    [user, token, loading, login, logout],
  )

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
