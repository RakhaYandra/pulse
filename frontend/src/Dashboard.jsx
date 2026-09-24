import { useEffect, useState } from 'react'
import { api } from './api.js'
import MonitorDetail, { Dot } from './MonitorDetail.jsx'

export default function Dashboard({ user, onLogout }) {
  const [summary, setSummary] = useState(null)
  const [monitors, setMonitors] = useState([])
  const [incidents, setIncidents] = useState([])
  const [tab, setTab] = useState('monitors')
  const [sel, setSel] = useState(null)
  const [form, setForm] = useState({ name: '', url: '', interval_seconds: 300, timeout_seconds: 5, failure_threshold: 3, recovery_threshold: 2 })
  const [err, setErr] = useState('')

  async function load() {
    const [s, ms, ins] = await Promise.all([api.summary(), api.monitors(), api.incidents()])
    setSummary(s); setMonitors(ms); setIncidents(ins)
  }

  useEffect(() => { load().catch(() => {}); const t = setInterval(load, 15000); return () => clearInterval(t) }, [])

  async function create(e) {
    e.preventDefault()
    setErr('')
    try {
      await api.createMonitor({ ...form, interval_seconds: +form.interval_seconds, timeout_seconds: +form.timeout_seconds, failure_threshold: +form.failure_threshold, recovery_threshold: +form.recovery_threshold })
      setForm({ name: '', url: '', interval_seconds: 300, timeout_seconds: 5, failure_threshold: 3, recovery_threshold: 2 })
      load()
    } catch (ex) { setErr(ex.message) }
  }

  if (sel) return <MonitorDetail id={sel} onBack={() => { setSel(null); load() }} />

  return (
    <div className="wrap">
      <header>
        <h1>Pulse</h1>
        <div>{user.email} <button className="link" onClick={onLogout}>logout</button></div>
      </header>
      {summary && (
        <div className="stats">
          <div className="stat"><b>{summary.total_monitors}</b><span>monitors</span></div>
          <div className="stat"><b>{summary.up}</b><span>up</span></div>
          <div className="stat"><b>{summary.down}</b><span>down</span></div>
          <div className="stat"><b>{summary.active_incidents}</b><span>active incidents</span></div>
          <div className="stat"><b>{Number(summary.uptime_24h).toFixed(2)}%</b><span>uptime 24h</span></div>
        </div>
      )}
      <nav>
        <button className={tab === 'monitors' ? 'active' : ''} onClick={() => setTab('monitors')}>Monitors</button>
        <button className={tab === 'incidents' ? 'active' : ''} onClick={() => setTab('incidents')}>Incidents ({incidents.filter((i) => i.status === 'OPEN').length})</button>
        <button className={tab === 'new' ? 'active' : ''} onClick={() => setTab('new')}>+ New</button>
      </nav>
      {tab === 'monitors' && monitors.map((m) => (
        <div key={m.id} className="row" onClick={() => setSel(m.id)}>
          <div><b>{m.name}</b> <Dot status={m.status} /><br /><span className="muted">{m.url} · every {m.interval_seconds}s{m.is_active ? '' : ' · PAUSED'}</span></div>
          <div onClick={(e) => e.stopPropagation()}>
            {m.is_active
              ? <button onClick={async () => { await api.pause(m.id); load() }}>pause</button>
              : <button onClick={async () => { await api.resume(m.id); load() }}>resume</button>}
            <button onClick={async () => { if (confirm('Delete?')) { await api.remove(m.id); load() } }}>del</button>
          </div>
        </div>
      ))}
      {tab === 'monitors' && monitors.length === 0 && <div className="muted">No monitors yet — create one.</div>}
      {tab === 'incidents' && incidents.map((in_) => (
        <div key={in_.id} className="row">
          <div><b>{in_.status}</b> — {in_.monitor_name}<br /><span className="muted">{in_.reason} · {new Date(in_.started_at).toLocaleString()}{in_.resolved_at ? ' → resolved' : ''}</span></div>
        </div>
      ))}
      {tab === 'new' && (
        <form className="card" onSubmit={create}>
          {err && <div className="error">{err}</div>}
          <input placeholder="Name: Payment API" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
          <input placeholder="URL: https://api.example.com/health" value={form.url} onChange={(e) => setForm({ ...form, url: e.target.value })} />
          <label>Interval (s, ≥60)<input type="number" value={form.interval_seconds} onChange={(e) => setForm({ ...form, interval_seconds: e.target.value })} /></label>
          <label>Timeout (s)<input type="number" value={form.timeout_seconds} onChange={(e) => setForm({ ...form, timeout_seconds: e.target.value })} /></label>
          <label>Failure threshold<input type="number" value={form.failure_threshold} onChange={(e) => setForm({ ...form, failure_threshold: e.target.value })} /></label>
          <label>Recovery threshold<input type="number" value={form.recovery_threshold} onChange={(e) => setForm({ ...form, recovery_threshold: e.target.value })} /></label>
          <button type="submit">Create monitor</button>
        </form>
      )}
    </div>
  )
}
