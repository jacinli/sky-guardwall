import { Navigate, Route, Routes } from 'react-router-dom'
import { useAuthStore } from './store/auth'
import Login from './pages/Login'
import AppLayout from './components/AppLayout'
import IptablesRules from './pages/IptablesRules'
import ManagedRules from './pages/ManagedRules'

function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const token = useAuthStore((s) => s.token)
  if (!token) return <Navigate to="/login" replace />
  return <>{children}</>
}

export default function App() {
  return (
    <Routes>
      <Route path="/login" element={<Login />} />
      <Route
        path="/"
        element={
          <ProtectedRoute>
            <AppLayout />
          </ProtectedRoute>
        }
      >
        <Route index element={<IptablesRules />} />
        <Route path="managed" element={<ManagedRules />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
