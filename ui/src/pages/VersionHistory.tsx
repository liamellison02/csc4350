import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { diffLines } from 'diff'
import { useAuth } from '../auth/useAuth'
import {
  ApiError,
  getConfiguration,
  getVersions,
  rollback,
  type Configuration,
  type ConfigVersion,
} from '../lib/api'

function renderDiff(fromYaml: string, toYaml: string): string {
  return diffLines(fromYaml, toYaml)
    .flatMap((part) => {
      const prefix = part.added ? '+ ' : part.removed ? '- ' : '  '
      return part.value
        .split('\n')
        .filter((line, i, arr) => !(line === '' && i === arr.length - 1))
        .map((line) => prefix + line)
    })
    .join('\n')
}

export function VersionHistory() {
  const { id } = useParams()
  const configId = Number(id)
  const { token, user } = useAuth()
  const canRollback = user?.role === 'operator' || user?.role === 'admin'

  const [config, setConfig] = useState<Configuration | null>(null)
  const [versions, setVersions] = useState<ConfigVersion[]>([])
  const [fromId, setFromId] = useState<number | null>(null)
  const [toId, setToId] = useState<number | null>(null)
  const [diffText, setDiffText] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const refresh = useCallback(() => {
    if (!token || Number.isNaN(configId)) return
    Promise.all([getConfiguration(token, configId), getVersions(token, configId)])
      .then(([cfg, vers]) => {
        setConfig(cfg)
        setVersions(vers)
        if (vers.length >= 2) {
          setFromId(vers[1].id)
          setToId(vers[0].id)
        }
      })
      .catch((err) => setError(err instanceof ApiError ? err.message : 'failed to load'))
  }, [token, configId])

  useEffect(() => {
    refresh()
  }, [refresh])

  function handleCompare() {
    const from = versions.find((v) => v.id === fromId)
    const to = versions.find((v) => v.id === toId)
    if (!from || !to) return
    setDiffText(renderDiff(from.yaml, to.yaml))
  }

  async function handleRollback(version: ConfigVersion) {
    if (!token) return
    if (!window.confirm(`Roll back to version ${version.version_no}?`)) return
    setError(null)
    try {
      await rollback(token, configId, version.id)
      setDiffText(null)
      refresh()
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'rollback failed')
    }
  }

  return (
    <main>
      <h1>Version History</h1>
      <p>
        View previous versions of {config ? config.name : 'this collector configuration'}.
      </p>

      <Link to={`/configurations/${id}/edit`}>Back to Configuration Editor</Link>

      {error && <p role="alert">{error}</p>}

      {versions.length >= 2 && (
        <section>
          <h2>Compare</h2>
          <label htmlFor="compare-from">compare from</label>{' '}
          <select
            id="compare-from"
            value={fromId ?? ''}
            onChange={(e) => setFromId(Number(e.target.value))}
          >
            {versions.map((v) => (
              <option key={v.id} value={v.id}>
                Version {v.version_no}
              </option>
            ))}
          </select>{' '}
          <label htmlFor="compare-to">compare to</label>{' '}
          <select
            id="compare-to"
            value={toId ?? ''}
            onChange={(e) => setToId(Number(e.target.value))}
          >
            {versions.map((v) => (
              <option key={v.id} value={v.id}>
                Version {v.version_no}
              </option>
            ))}
          </select>{' '}
          <button type="button" onClick={handleCompare}>
            Compare
          </button>
          {diffText !== null && <pre>{diffText}</pre>}
        </section>
      )}

      {versions.map((version) => (
        <section key={version.id}>
          <hr />
          <h2>
            Version {version.version_no}
            {config?.current_version_id === version.id ? ' - Current' : ''}
          </h2>
          <p>Created by: user #{version.author_id}</p>
          <p>Created: {new Date(version.created_at).toLocaleString()}</p>
          {canRollback && config?.current_version_id !== version.id && (
            <button type="button" onClick={() => handleRollback(version)}>
              Rollback to this version
            </button>
          )}
        </section>
      ))}
    </main>
  )
}
