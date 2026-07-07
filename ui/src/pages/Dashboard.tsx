import { useEffect, useState } from 'react'
import { useAuth } from '../auth/useAuth'
import { getAgents } from '../lib/api'
import type { Agent } from '../lib/api'

function shortHash(hash: string | null): string {
  if (!hash) return '-'
  return hash.length > 12 ? `${hash.slice(0, 12)}...` : hash
}

export function Dashboard() {
  const { user, token, logout } = useAuth()
  const [agents, setAgents] = useState<Agent[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!token) return
    let active = true
    getAgents(token)
      .then((rows) => {
        if (!active) return
        setAgents(rows)
        setError(null)
      })
      .catch((err) => {
        if (!active) return
        setError(err instanceof Error ? err.message : 'failed to load agents')
      })
      .finally(() => {
        if (active) setLoading(false)
      })
    return () => {
      active = false
    }
  }, [token])

  return (
    <div className="app-shell">
      <header className="topbar">
        <span className="brand">Helmsman</span>
        <div className="topbar-right">
          {user && (
            <span className="user">
              <span className="user-email">{user.email}</span>
              <span className={`badge role role-${user.role}`}>{user.role}</span>
            </span>
          )}
          <button type="button" className="ghost" onClick={logout}>
            log out
          </button>
        </div>
      </header>

      <main className="content">
        <h2>fleet</h2>
        {loading && <p className="muted">loading agents...</p>}
        {error && (
          <div className="banner error" role="alert">
            {error}
          </div>
        )}
        {!loading && !error && agents.length === 0 && (
          <p className="muted">no agents registered yet.</p>
        )}
        {!loading && !error && agents.length > 0 && (
          <div className="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>hostname</th>
                  <th>instance uid</th>
                  <th>type</th>
                  <th>version</th>
                  <th>status</th>
                  <th>last seen</th>
                  <th>config hash</th>
                </tr>
              </thead>
              <tbody>
                {agents.map((agent) => (
                  <tr key={agent.instance_uid}>
                    <td>{agent.hostname}</td>
                    <td className="mono">{agent.instance_uid}</td>
                    <td>{agent.agent_type ?? '-'}</td>
                    <td>{agent.version ?? '-'}</td>
                    <td>
                      <span className={`badge status status-${agent.status}`}>
                        {agent.status}
                      </span>
                    </td>
                    <td>{agent.last_seen ?? '-'}</td>
                    <td className="mono">{shortHash(agent.effective_config_hash)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </main>
    </div>
  )
}
