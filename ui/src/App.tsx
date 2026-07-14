import { Navigate, Route, Routes } from 'react-router-dom'
import { ProtectedRoute } from './components/ProtectedRoute'
import { Login } from './pages/Login'
import { Dashboard } from './pages/Dashboard'
import { ConfigEditor } from './pages/ConfigEditor'
import { VersionHistory } from './pages/VersionHistory'

function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />

      <Route
        path="/configurations/:id/edit"
        element={
          <ProtectedRoute>
            <ConfigEditor />
          </ProtectedRoute>
        }
      />
      <Route
        path="/configurations/:id/history"
        element={
          <ProtectedRoute>
            <VersionHistory />
          </ProtectedRoute>
        }
      />

      <Route
        path="/"
        element={
          <ProtectedRoute>
            <Dashboard />
          </ProtectedRoute>
        }
      />

      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

export default App