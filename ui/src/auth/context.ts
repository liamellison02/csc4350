import { createContext } from 'react'
import type { User } from '../lib/api'

// localStorage key holding the bearer token across reloads
export const TOKEN_KEY = 'helmsman.token'

export interface AuthContextValue {
  user: User | null
  token: string | null
  loading: boolean
  login: (email: string, password: string) => Promise<void>
  logout: () => void
}

export const AuthContext = createContext<AuthContextValue | null>(null)
