import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { AuthContext } from '../auth/context'
import type { AuthContextValue } from '../auth/context'
import { Dashboard } from './Dashboard'
import * as api from '../lib/api'
import type { Agent, Configuration } from '../lib/api'

vi.mock('../lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../lib/api')>()
  return {
    ...actual,
    login: vi.fn(),
    getMe: vi.fn(),
    getAgents: vi.fn(),
    getConfigurations: vi.fn(),
  }
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

const configs: Configuration[] = [
  {
    id: 1,
    name: 'prod-collectors',
    label_selector: 'env=prod',
    current_version_id: 10,
  },
  {
    id: 2,
    name: 'edge-collectors',
    label_selector: null,
    current_version_id: null,
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
    vi.mocked(api.getAgents).mockResolvedValue([])
    vi.mocked(api.getConfigurations).mockResolvedValue([])
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

  it('lists configurations with edit and history links', async () => {
    vi.mocked(api.getConfigurations).mockResolvedValue(configs)

    render(
      <MemoryRouter>
        <AuthContext.Provider value={authValue}>
          <Dashboard />
        </AuthContext.Provider>
      </MemoryRouter>,
    )

    expect(await screen.findByText('prod-collectors')).toBeInTheDocument()
    expect(screen.getByText('edge-collectors')).toBeInTheDocument()

    const editLinks = screen.getAllByRole('link', { name: 'edit' })
    expect(editLinks[0]).toHaveAttribute('href', '/configurations/1/edit')
    expect(editLinks[1]).toHaveAttribute('href', '/configurations/2/edit')

    const historyLinks = screen.getAllByRole('link', { name: 'history' })
    expect(historyLinks[0]).toHaveAttribute('href', '/configurations/1/history')
    expect(historyLinks[1]).toHaveAttribute('href', '/configurations/2/history')

    expect(
      screen.getByRole('link', { name: 'New configuration' }),
    ).toHaveAttribute('href', '/configurations/new/edit')
    expect(api.getConfigurations).toHaveBeenCalledWith('tok-abc')
  })
})
