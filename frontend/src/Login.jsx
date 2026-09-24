import { useState } from 'react'
import { api } from './api.js'

export default function Login({ onAuth }) {
  const [mode, setMode] = useState('login')
  const [email, setEmail] = useState('demo@pulse.local')
  const [password, setPassword] = useState('')
  const [name, setName] = useState('')
  const [err, setErr] = useState('')

  async function submit(e) {
    e.preventDefault()
    setErr('')
    try {
      const data = mode === 'login'
        ? await api.login(email, password)
        : await api.register(email, password, name)
      localStorage.setItem('pulse_token', data.token)
      onAuth(data.user)
    } catch (ex) {
      setErr(ex.message)
    }
  }

  return (
    <div className="center">
      <form className="card" onSubmit={submit}>
        <h1>Pulse</h1>
        <p className="muted">API monitoring & incidents</p>
        {err && <div className="error">{err}</div>}
        <input placeholder="email" value={email} onChange={(e) => setEmail(e.target.value)} />
        <input placeholder="password (min 8)" type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
        {mode === 'register' && <input placeholder="name" value={name} onChange={(e) => setName(e.target.value)} />}
        <button type="submit">{mode === 'login' ? 'Login' : 'Register'}</button>
        <button type="button" className="link" onClick={() => setMode(mode === 'login' ? 'register' : 'login')}>
          {mode === 'login' ? 'Need an account? Register' : 'Have an account? Login'}
        </button>
      </form>
    </div>
  )
}
