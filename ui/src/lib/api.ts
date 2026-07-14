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

export interface Configuration {
  id: number
  name: string
  label_selector: string | null
  current_version_id: number | null
}

export interface ConfigVersion {
  id: number
  configuration_id: number
  version_no: number
  yaml: string
  hash: string
  author_id: number
  created_at: string
}

export function getConfigurations(token: string): Promise<Configuration[]> {
  return apiFetch<Configuration[]>('/configurations', { token })
}

export function getConfiguration(token: string, id: number): Promise<Configuration> {
  return apiFetch<Configuration>(`/configurations/${id}`, { token })
}

export function createConfiguration(
  token: string,
  body: { name: string; label_selector: string | null },
): Promise<Configuration> {
  return apiFetch<Configuration>('/configurations', { token, method: 'POST', body })
}

export function getVersions(token: string, configId: number): Promise<ConfigVersion[]> {
  return apiFetch<ConfigVersion[]>(`/configurations/${configId}/versions`, { token })
}

export function createVersion(token: string, configId: number, yaml: string): Promise<ConfigVersion> {
  return apiFetch<ConfigVersion>(`/configurations/${configId}/versions`, {
    token,
    method: 'POST',
    body: { yaml },
  })
}

export function rollback(token: string, configId: number, versionId: number): Promise<Configuration> {
  return apiFetch<Configuration>(`/configurations/${configId}/rollback`, {
    token,
    method: 'POST',
    body: { version_id: versionId },
  })
}
