import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { AuthProvider } from '../auth/AuthContext'
import { Login } from './Login'
import * as api from '../lib/api'

// keep ApiError real (instanceof checks), mock only the network calls
vi.mock('../lib/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../lib/api')>()
  return { ...actual, login: vi.fn(), getMe: vi.fn(), getAgents: vi.fn() }
})

function renderLogin() {
  return render(
    <MemoryRouter initialEntries={['/login']}>
      <AuthProvider>
        <Login />
      </AuthProvider>
    </MemoryRouter>,
  )
}

describe('Login', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.clearAllMocks()
  })

  it('submits credentials and stores the token via login then getMe', async () => {
    vi.mocked(api.login).mockResolvedValue({
      access_token: 'tok-123',
      token_type: 'bearer',
      user: { id: 2, email: 'operator@helmsman.local', role: 'operator' },
    })
    vi.mocked(api.getMe).mockResolvedValue({
      id: 2,
      email: 'operator@helmsman.local',
      role: 'operator',
    })

    const user = userEvent.setup()
    renderLogin()

    await user.type(screen.getByLabelText('email'), 'operator@helmsman.local')
    await user.type(screen.getByLabelText('password'), 'operator123!')
    await user.click(screen.getByRole('button', { name: /sign in/i }))

    await waitFor(() =>
      expect(api.login).toHaveBeenCalledWith(
        'operator@helmsman.local',
        'operator123!',
      ),
    )
    await waitFor(() => expect(api.getMe).toHaveBeenCalledWith('tok-123'))
    await waitFor(() =>
      expect(localStorage.getItem('helmsman.token')).toBe('tok-123'),
    )
  })

  it('shows an error banner when login raises an ApiError', async () => {
    vi.mocked(api.login).mockRejectedValue(
      new api.ApiError(401, 'invalid email or password'),
    )

    const user = userEvent.setup()
    renderLogin()

    await user.type(screen.getByLabelText('email'), 'bad@helmsman.local')
    await user.type(screen.getByLabelText('password'), 'nope')
    await user.click(screen.getByRole('button', { name: /sign in/i }))

    const banner = await screen.findByRole('alert')
    expect(banner).toHaveTextContent('invalid email or password')
    expect(localStorage.getItem('helmsman.token')).toBeNull()
  })
})
