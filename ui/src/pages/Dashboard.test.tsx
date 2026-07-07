import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { AuthContext } from '../auth/context'
import type { AuthContextValue } from '../auth/context'
import { Dashboard } from './Dashboard'
import * as api from '../lib/api'
import type { Agent } from '../lib/api'

vi.mock('../lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../lib/api')>()
  return { ...actual, login: vi.fn(), getMe: vi.fn(), getAgents: vi.fn() }
})

const agents: Agent[] = [
  {
    instance_uid: 'uid-prod-01',
    hostname: 'collector-prod-01',
    labels: { env: 'prod' },
    agent_type: 'otelcol',
    version: '0.100.0',
    status: 'healthy',
    last_seen: '2026-07-07T12:00:00Z',
    effective_config_hash: 'abcdef1234567890',
  },
  {
    instance_uid: 'uid-edge-01',
    hostname: 'collector-edge-01',
    labels: { env: 'edge' },
    agent_type: 'otelcol',
    version: '0.100.0',
    status: 'disconnected',
    last_seen: null,
    effective_config_hash: null,
  },
]

const authValue: AuthContextValue = {
  user: { id: 1, email: 'admin@helmsman.local', role: 'admin' },
  token: 'tok-abc',
  loading: false,
  login: vi.fn(),
  logout: vi.fn(),
}

describe('Dashboard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders the mocked agents with their status text', async () => {
    vi.mocked(api.getAgents).mockResolvedValue(agents)

    render(
      <MemoryRouter>
        <AuthContext.Provider value={authValue}>
          <Dashboard />
        </AuthContext.Provider>
      </MemoryRouter>,
    )

    expect(await screen.findByText('collector-prod-01')).toBeInTheDocument()
    expect(screen.getByText('collector-edge-01')).toBeInTheDocument()
    expect(screen.getByText('healthy')).toBeInTheDocument()
    expect(screen.getByText('disconnected')).toBeInTheDocument()
    expect(api.getAgents).toHaveBeenCalledWith('tok-abc')
  })
})
