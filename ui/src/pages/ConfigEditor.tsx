import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { load } from 'js-yaml'
import { useAuth } from '../auth/useAuth'
import {
  ApiError,
  createConfiguration,
  createVersion,
  getConfiguration,
  getVersions,
} from '../lib/api'

export function ConfigEditor() {
  const { id } = useParams()
  const isNew = id === 'new'
  const navigate = useNavigate()
  const { token, user } = useAuth()
  const readOnly = user?.role === 'viewer'

  const [name, setName] = useState('')
  const [selector, setSelector] = useState('')
  const [yamlText, setYamlText] = useState('')
  const [validation, setValidation] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    if (isNew || !token || !id) return
    const configId = Number(id)
    Promise.all([getConfiguration(token, configId), getVersions(token, configId)])
      .then(([config, versions]) => {
        setName(config.name)
        setSelector(config.label_selector ?? '')
        if (versions.length > 0) setYamlText(versions[0].yaml)
      })
      .catch((err) => setError(err instanceof ApiError ? err.message : 'failed to load'))
  }, [id, isNew, token])

  function handleValidate() {
    try {
      load(yamlText)
      setValidation('valid yaml')
    } catch (err) {
      setValidation(err instanceof Error ? err.message : 'invalid yaml')
    }
  }

  async function handleSave() {
    if (!token) return
    setSaving(true)
    setError(null)
    try {
      let configId = Number(id)
      if (isNew) {
        const created = await createConfiguration(token, {
          name,
          label_selector: selector.trim() === '' ? null : selector.trim(),
        })
        configId = created.id
      }
      await createVersion(token, configId, yamlText)
      navigate(`/configurations/${configId}/history`)
    } catch (err) {
      setError(err instanceof ApiError ? err.message : 'save failed')
    } finally {
      setSaving(false)
    }
  }

  return (
    <main>
      <h1>Configuration Editor</h1>
      <p>Edit and save collector configurations.</p>
      <Link to="/">Back to Dashboard</Link>

      {error && <p role="alert">{error}</p>}

      <div>
        <label htmlFor="config-name">Configuration Name</label>
        <br />
        <input
          id="config-name"
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          disabled={readOnly || !isNew}
        />
      </div>

      {isNew && (
        <div>
          <label htmlFor="config-selector">Label Selector (k=v, empty matches all agents)</label>
          <br />
          <input
            id="config-selector"
            type="text"
            value={selector}
            onChange={(e) => setSelector(e.target.value)}
            disabled={readOnly}
          />
        </div>
      )}

      <br />

      <div>
        <label htmlFor="config-content">YAML Configuration</label>
        <br />
        <textarea
          id="config-content"
          rows={18}
          cols={70}
          spellCheck={false}
          value={yamlText}
          onChange={(e) => setYamlText(e.target.value)}
          disabled={readOnly}
        />
      </div>

      {validation && <p role="status">{validation}</p>}

      <br />

      <button type="button" onClick={handleValidate}>
        Validate
      </button>
      {!readOnly && (
        <button
          type="button"
          onClick={handleSave}
          disabled={saving || yamlText.trim() === '' || (isNew && name.trim() === '')}
        >
          Save as New Version
        </button>
      )}
    </main>
  )
}
