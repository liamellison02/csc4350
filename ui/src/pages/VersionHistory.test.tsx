import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { AuthContext } from '../auth/context'
import type { AuthContextValue } from '../auth/context'
import { VersionHistory } from './VersionHistory'
import * as api from '../lib/api'
import type { Configuration, ConfigVersion } from '../lib/api'

// keep ApiError real (instanceof + message), mock only the network calls
vi.mock('../lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../lib/api')>()
  return {
    ...actual,
    getConfiguration: vi.fn(),
    getVersions: vi.fn(),
    rollback: vi.fn(),
  }
})

const TOKEN = 'tok-abc'
const CONFIG_ID = 2

// current is the middle version (id 20) so the marker assertion proves
// placement on the correct section, not just the first row
const config: Configuration = {
  id: CONFIG_ID,
  name: 'prod-collectors',
  label_selector: 'env=prod',
  current_version_id: 20,
}

// newest first, matching the api order (version_no desc); versions[0]
// adds an exporters block over versions[1] so the default diff shows it
const versions: ConfigVersion[] = [
  {
    id: 30,
    configuration_id: CONFIG_ID,
    version_no: 3,
    yaml: 'receivers:\n  otlp:\n    protocols:\n      grpc:\nexporters:\n  debug:\n',
    hash: 'h3',
    author_id: 7,
    created_at: '2026-07-12T10:00:00Z',
  },
  {
    id: 20,
    configuration_id: CONFIG_ID,
    version_no: 2,
    yaml: 'receivers:\n  otlp:\n    protocols:\n      grpc:\n',
    hash: 'h2',
    author_id: 4,
    created_at: '2026-07-08T10:00:00Z',
  },
  {
    id: 10,
    configuration_id: CONFIG_ID,
    version_no: 1,
    yaml: 'receivers:\n  hostmetrics:\n',
    hash: 'h1',
    author_id: 4,
    created_at: '2026-07-03T10:00:00Z',
  },
]

function makeAuth(role: string): AuthContextValue {
  return {
    user: { id: 1, email: 'operator@helmsman.local', role },
    token: TOKEN,
    loading: false,
    login: vi.fn(),
    logout: vi.fn(),
  }
}

function renderHistory(role = 'operator', path = `/configurations/${CONFIG_ID}/history`) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <AuthContext.Provider value={makeAuth(role)}>
        <Routes>
          <Route path="/configurations/:id/history" element={<VersionHistory />} />
        </Routes>
      </AuthContext.Provider>
    </MemoryRouter>,
  )
}

describe('VersionHistory', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(api.getConfiguration).mockResolvedValue(config)
    vi.mocked(api.getVersions).mockResolvedValue(versions)
    vi.mocked(api.rollback).mockResolvedValue(config)
  })

  it('renders versions newest first with the current marker on the right section', async () => {
    renderHistory()

    await screen.findByRole('heading', { name: 'Version 3' })

    const versionHeadings = screen
      .getAllByRole('heading', { level: 2 })
      .map((h) => h.textContent?.trim())
      .filter((t) => t?.startsWith('Version'))

    expect(versionHeadings).toEqual([
      'Version 3',
      'Version 2 - Current',
      'Version 1',
    ])
    expect(api.getConfiguration).toHaveBeenCalledWith(TOKEN, CONFIG_ID)
    expect(api.getVersions).toHaveBeenCalledWith(TOKEN, CONFIG_ID)
  })

  it('renders a diff with a "+ exporters:" line for the two default versions', async () => {
    const user = userEvent.setup()
    const { container } = renderHistory()

    const compareBtn = await screen.findByRole('button', { name: /^compare$/i })
    await user.click(compareBtn)

    await waitFor(() => {
      const pre = container.querySelector('pre')
      expect(pre).not.toBeNull()
      expect(pre?.textContent).toContain('+ exporters:')
    })
  })

  it('hides rollback for viewers and rolls back a non-current version for operators', async () => {
    const user = userEvent.setup()
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(true)

    // viewer: no rollback controls anywhere
    const viewerView = renderHistory('viewer')
    await screen.findByRole('heading', { name: 'Version 3' })
    expect(
      screen.queryAllByRole('button', { name: 'Rollback to this version' }),
    ).toHaveLength(0)
    viewerView.unmount()

    // operator: a rollback button on each non-current version (2 of 3)
    renderHistory('operator')
    await screen.findByRole('heading', { name: 'Version 3' })
    expect(
      screen.getAllByRole('button', { name: 'Rollback to this version' }),
    ).toHaveLength(2)

    // click the rollback control inside the version 3 section
    const v3section = screen
      .getByRole('heading', { name: 'Version 3' })
      .closest('section') as HTMLElement
    const versionsCallsBefore = vi.mocked(api.getVersions).mock.calls.length

    await user.click(
      within(v3section).getByRole('button', { name: 'Rollback to this version' }),
    )

    expect(confirmSpy).toHaveBeenCalled()
    await waitFor(() =>
      expect(api.rollback).toHaveBeenCalledWith(TOKEN, CONFIG_ID, 30),
    )
    // re-fetch after rollback: versions are loaded again
    await waitFor(() =>
      expect(vi.mocked(api.getVersions).mock.calls.length).toBeGreaterThan(
        versionsCallsBefore,
      ),
    )
  })

  it('shows a rollback ApiError in an alert', async () => {
    const user = userEvent.setup()
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    vi.mocked(api.rollback).mockRejectedValue(
      new api.ApiError(400, 'version is already current'),
    )

    renderHistory('operator')
    await screen.findByRole('heading', { name: 'Version 3' })

    const rollbackButtons = screen.getAllByRole('button', {
      name: 'Rollback to this version',
    })
    await user.click(rollbackButtons[0])

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('version is already current')
  })
})
