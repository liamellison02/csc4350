// thin typed client for the helmsman management api. see the frozen
// api contract in docs/superpowers/plans for the exact response shapes.

const API_URL = import.meta.env.VITE_API_URL ?? 'http://localhost:8000'

export interface User {
  id: number
  email: string
  role: string
}

export interface Agent {
  instance_uid: string
  hostname: string
  labels: Record<string, unknown>
  agent_type: string | null
  version: string | null
  status: string
  last_seen: string | null
  effective_config_hash: string | null
}

export interface LoginResponse {
  access_token: string
  token_type: string
  user: User
}

// carries the http status so callers can branch on 401 vs other failures
export class ApiError extends Error {
  status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

interface FetchOptions {
  token?: string
  method?: string
  body?: unknown
}

async function apiFetch<T>(path: string, options: FetchOptions = {}): Promise<T> {
  const { token, method = 'GET', body } = options
  const headers: Record<string, string> = {}
  if (token) headers.Authorization = `Bearer ${token}`
  if (body !== undefined) headers['Content-Type'] = 'application/json'

  const res = await fetch(`${API_URL}${path}`, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })

  // read the body once, parse as json when present
  let data: unknown = null
  const text = await res.text()
  if (text) {
    try {
      data = JSON.parse(text)
    } catch {
      data = text
    }
  }

  if (!res.ok) {
    const detail =
      data && typeof data === 'object' && 'detail' in data
        ? String((data as { detail: unknown }).detail)
        : `request failed with status ${res.status}`
    throw new ApiError(res.status, detail)
  }

  return data as T
}

export function login(email: string, password: string): Promise<LoginResponse> {
  return apiFetch<LoginResponse>('/auth/login', {
    method: 'POST',
    body: { email, password },
  })
}

export function getMe(token: string): Promise<User> {
  return apiFetch<User>('/auth/me', { token })
}

export function getAgents(token: string): Promise<Agent[]> {
  return apiFetch<Agent[]>('/agents', { token })
}
