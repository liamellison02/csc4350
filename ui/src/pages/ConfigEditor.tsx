import { Link } from 'react-router-dom'

export function ConfigEditor() {
  function handleValidate() {
    alert('Validation will be connected later.')
  }

  function handleSave() {
    alert('Saving will be connected later.')
  }

  return (
    <main>
      <h1>Configuration Editor</h1>
      <p>Edit and save collector configurations.</p>

      <div>
        <label htmlFor="config-name">Configuration Name</label>
        <br />

        <input
          id="config-name"
          type="text"
          defaultValue="Production Collector"
        />
      </div>

      <br />

      <div>
        <label htmlFor="config-content">YAML Configuration</label>
        <br />

        <textarea
          id="config-content"
          rows={18}
          cols={70}
          spellCheck={false}
          defaultValue={`receivers:
  otlp:
    protocols:
      grpc:
      http:

exporters:
  debug:

service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [debug]`}
        />
      </div>

      <br />

      <div>
        <label htmlFor="change-description">Change Description</label>
        <br />

        <input
          id="change-description"
          type="text"
          placeholder="Example: Updated logging settings"
        />
      </div>

      <br />

      <div>
        <button type="button" onClick={handleValidate}>
          Validate
        </button>

        <button type="button" onClick={handleSave}>
          Save New Version
        </button>
      </div>

      <br />

<Link to="/configurations/1/history">
  View Version History
</Link>

    </main>
  )
}