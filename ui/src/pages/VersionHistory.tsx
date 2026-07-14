import { Link } from 'react-router-dom'

export function VersionHistory() {
  return (
    <main>
      <h1>Version History</h1>
      <p>View previous versions of this collector configuration.</p>

      <Link to="/configurations/1/edit">
  Back to Configuration Editor
</Link>

<br />
<br />

      <section>
        <h2>Version 3 — Current</h2>
        <p>Created by: Admin User</p>
        <p>Created: July 12, 2026</p>
        <p>Description: Updated logging settings</p>

        <button
            type="button"
            onClick={() => alert('Opening Version 3 Details')}
            >
            View
            </button>

        <button 
              type="button"
              onClick={() => alert('Comparing Version 3 with Another Version')} 
            >
                Compare
            </button>
      </section>

      <hr />

      <section>
        <h2>Version 2</h2>
        <p>Created by: Operator User</p>
        <p>Created: July 8, 2026</p>
        <p>Description: Added OTLP receiver</p>

        <button
            type="button"
            onClick={() => alert('Opening Version 2 Details')}
            >
            View
            </button>

        <button
            type="button"
            onClick={() => alert('Comparing Version 2 with Another Version')}
            >
            Compare
            </button>

        <button
            type="button"
            onClick={() => alert('Version 2 will be restored as a New Version')}
            >
            Restore
            </button>

      </section>

      <hr />

      <section>
        <h2>Version 1</h2>
        <p>Created by: System Admin</p>
        <p>Created: July 3, 2026</p>
        <p>Description: Initial configuration</p>

        <button
            type="button"
            onClick={() => alert('Opening Version 1 Details')}
            >
            View
            </button>

        <button
            type="button"
            onClick={() => alert('Comparing Version 1 with Another Version')}
            >
            Compare
            </button>

        <button
            type="button"
            onClick={() => alert('Version 1 will be restored as a New Version')}
            >
            Restore
            </button>

      </section>
    </main>
  )
}