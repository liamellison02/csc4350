import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes, useParams } from 'react-router-dom'
import { AuthContext } from '../auth/context'
import type { AuthContextValue } from '../auth/context'
import { ConfigEditor } from './ConfigEditor'
import * as api from '../lib/api'
import type { Configuration, ConfigVersion } from '../lib/api'

// keep ApiError real (instanceof + message), mock only the network calls
vi.mock('../lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../lib/api')>()
  return {
    ...actual,
    getConfiguration: vi.fn(),
    getVersions: vi.fn(),
    createConfiguration: vi.fn(),
    createVersion: vi.fn(),
  }
})

const TOKEN = 'tok-abc'
const CONFIG_ID = 2

const config: Configuration = {
  id: CONFIG_ID,
  name: 'prod-collectors',
  label_selector: 'env=prod',
  current_version_id: 20,
}

// versions[0] is the newest; the editor prefills from it
const versions: ConfigVersion[] = [
  {
    id: 20,
    configuration_id: CONFIG_ID,
    version_no: 2,
    yaml: 'receivers:\n  otlp:\n    protocols:\n      grpc:\n',
    hash: 'newhash',
    author_id: 1,
    created_at: '2026-07-07T12:00:00Z',
  },
  {
    id: 19,
    configuration_id: CONFIG_ID,
    version_no: 1,
    yaml: 'receivers:\n  hostmetrics:\n',
    hash: 'oldhash',
    author_id: 1,
    created_at: '2026-07-06T12:00:00Z',
  },
]

const createdConfig: Configuration = {
  id: 7,
  name: 'new-collectors',
  label_selector: null,
  current_version_id: null,
}

const savedVersion: ConfigVersion = {
  id: 30,
  configuration_id: 7,
  version_no: 1,
  yaml: 'x: 1',
  hash: 'h',
  author_id: 1,
  created_at: '2026-07-08T12:00:00Z',
}

function makeAuth(role: string): AuthContextValue {
  return {
    user: { id: 1, email: 'operator@helmsman.local', role },
    token: TOKEN,
    loading: false,
    login: vi.fn(),
    logout: vi.fn(),
  }
}

// marker rendered by the history route so navigation can be asserted
function HistoryMarker() {
  const { id } = useParams()
  return <div data-testid="history-route">history for {id}</div>
}

function renderEditor(path: string, role = 'operator') {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <AuthContext.Provider value={makeAuth(role)}>
        <Routes>
          <Route path="/configurations/:id/edit" element={<ConfigEditor />} />
          <Route
            path="/configurations/:id/history"
            element={<HistoryMarker />}
          />
        </Routes>
      </AuthContext.Provider>
    </MemoryRouter>,
  )
}

describe('ConfigEditor', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(api.getConfiguration).mockResolvedValue(config)
    vi.mocked(api.getVersions).mockResolvedValue(versions)
    vi.mocked(api.createConfiguration).mockResolvedValue(createdConfig)
    vi.mocked(api.createVersion).mockResolvedValue(savedVersion)
  })

  it('loads config and versions in edit mode and prefills the newest version yaml', async () => {
    renderEditor(`/configurations/${CONFIG_ID}/edit`)

    await waitFor(() =>
      expect(screen.getByLabelText('YAML Configuration')).toHaveValue(
        versions[0].yaml,
      ),
    )
    expect(api.getConfiguration).toHaveBeenCalledWith(TOKEN, CONFIG_ID)
    expect(api.getVersions).toHaveBeenCalledWith(TOKEN, CONFIG_ID)

    const nameInput = screen.getByLabelText('Configuration Name')
    expect(nameInput).toHaveValue('prod-collectors')
    expect(nameInput).toBeDisabled()
  })

  it('prefills the current version yaml, not the newest, when current is older', async () => {
    // current_version_id points at the older version (id 19), e.g. after a rollback
    vi.mocked(api.getConfiguration).mockResolvedValue({
      ...config,
      current_version_id: 19,
    })
    renderEditor(`/configurations/${CONFIG_ID}/edit`)

    await waitFor(() =>
      expect(screen.getByLabelText('YAML Configuration')).toHaveValue(
        versions[1].yaml,
      ),
    )
    expect(api.getConfiguration).toHaveBeenCalledWith(TOKEN, CONFIG_ID)
    expect(api.getVersions).toHaveBeenCalledWith(TOKEN, CONFIG_ID)
  })

  it('validates good yaml and reports a parse error for bad yaml', async () => {
    const user = userEvent.setup()
    renderEditor('/configurations/new/edit')

    const textarea = screen.getByLabelText('YAML Configuration')

    fireEvent.change(textarea, { target: { value: 'receivers:\n  otlp: enabled\n' } })
    await user.click(screen.getByRole('button', { name: /validate/i }))
    expect(screen.getByRole('status')).toHaveTextContent('valid yaml')

    fireEvent.change(textarea, { target: { value: 'receivers: [unclosed' } })
    await user.click(screen.getByRole('button', { name: /validate/i }))
    const status = screen.getByRole('status')
    expect(status).not.toHaveTextContent('valid yaml')
    expect(status).toHaveTextContent(/flow collection/i)
  })

  it('saves a new version in edit mode and navigates to the history route', async () => {
    const user = userEvent.setup()
    renderEditor(`/configurations/${CONFIG_ID}/edit`)

    // wait for the prefill so save sends the loaded yaml
    await waitFor(() =>
      expect(screen.getByLabelText('YAML Configuration')).toHaveValue(
        versions[0].yaml,
      ),
    )

    await user.click(screen.getByRole('button', { name: /save as new version/i }))

    await waitFor(() =>
      expect(api.createVersion).toHaveBeenCalledWith(
        TOKEN,
        CONFIG_ID,
        versions[0].yaml,
      ),
    )
    expect(api.createConfiguration).not.toHaveBeenCalled()
    expect(await screen.findByTestId('history-route')).toHaveTextContent(
      `history for ${CONFIG_ID}`,
    )
  })

  it('creates the configuration then a version in create mode using the returned id', async () => {
    const user = userEvent.setup()
    renderEditor('/configurations/new/edit')

    fireEvent.change(screen.getByLabelText('Configuration Name'), {
      target: { value: 'new-collectors' },
    })
    fireEvent.change(screen.getByLabelText('YAML Configuration'), {
      target: { value: 'receivers:\n  otlp: enabled\n' },
    })

    await user.click(screen.getByRole('button', { name: /save as new version/i }))

    await waitFor(() =>
      expect(api.createConfiguration).toHaveBeenCalledWith(TOKEN, {
        name: 'new-collectors',
        label_selector: null,
      }),
    )
    expect(api.createVersion).toHaveBeenCalledWith(
      TOKEN,
      createdConfig.id,
      'receivers:\n  otlp: enabled\n',
    )
    expect(await screen.findByTestId('history-route')).toHaveTextContent(
      `history for ${createdConfig.id}`,
    )
  })

  it('disables the textarea and hides the save button for viewers', () => {
    renderEditor('/configurations/new/edit', 'viewer')

    expect(screen.getByLabelText('YAML Configuration')).toBeDisabled()
    expect(
      screen.queryByRole('button', { name: /save as new version/i }),
    ).toBeNull()
    expect(
      screen.getByRole('button', { name: /validate/i }),
    ).toBeInTheDocument()
  })

  it('renders the ApiError detail in an alert when save fails', async () => {
    vi.mocked(api.createVersion).mockRejectedValue(
      new api.ApiError(400, 'invalid yaml'),
    )
    const user = userEvent.setup()
    renderEditor(`/configurations/${CONFIG_ID}/edit`)

    await waitFor(() =>
      expect(screen.getByLabelText('YAML Configuration')).toHaveValue(
        versions[0].yaml,
      ),
    )

    await user.click(screen.getByRole('button', { name: /save as new version/i }))

    const alert = await screen.findByRole('alert')
    expect(alert).toHaveTextContent('invalid yaml')
  })
})
