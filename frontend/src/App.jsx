import { useEffect, useState } from 'react'
import Login from './Login.jsx'
import Dashboard from './Dashboard.jsx'
import { api } from './api.js'

export default function App() {
  const [user, setUser] = useState(null)
  const [checking, setChecking] = useState(!!localStorage.getItem('pulse_token'))

  useEffect(() => {
    if (!localStorage.getItem('pulse_token')) return
    api.me().then(setUser).catch(() => localStorage.removeItem('pulse_token')).finally(() => setChecking(false))
  }, [])

  if (checking) return <div className="wrap muted">loading…</div>
  if (!user) return <Login onAuth={setUser} />
  return <Dashboard user={user} onLogout={() => { localStorage.removeItem('pulse_token'); setUser(null) }} />
}
