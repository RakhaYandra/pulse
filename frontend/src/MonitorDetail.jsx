import { useEffect, useState } from 'react'
import { api } from './api.js'

function Spark({ checks }) {
  const pts = [...checks].reverse().filter((c) => c.response_time_ms != null)
  if (!pts.length) return <div className="muted">no data yet</div>
  const max = Math.max(...pts.map((c) => c.response_time_ms), 1)
  const W = 300, H = 60
  const d = pts.map((c, i) => {
    const x = (i / Math.max(pts.length - 1, 1)) * W
    const y = H - (c.response_time_ms / max) * (H - 6) - 3
    return `${i === 0 ? 'M' : 'L'}${x.toFixed(1)},${y.toFixed(1)}`
  }).join(' ')
  return (
    <svg width={W} height={H} className="spark">
      <path d={d} fill="none" stroke="#3b82f6" strokeWidth="2" />
      {pts.map((c, i) => (
        <circle key={i}
          cx={(i / Math.max(pts.length - 1, 1)) * W}
          cy={H - (c.response_time_ms / max) * (H - 6) - 3}
          r="3" fill={c.status === 'UP' ? '#22c55e' : '#ef4444'} />
      ))}
    </svg>
  )
}

export default function MonitorDetail({ id, onBack }) {
  const [monitors, setMonitors] = useState([])
  const [checks, setChecks] = useState([])
  const [incidents, setIncidents] = useState([])

  const m = monitors.find((x) => x.id === id)

  async function load() {
    const [ms, cs, ins] = await Promise.all([api.monitors(), api.checks(id), api.monitorIncidents(id)])
    setMonitors(ms); setChecks(cs); setIncidents(ins)
  }

  useEffect(() => { load().catch(() => {}); const t = setInterval(load, 15000); return () => clearInterval(t) }, [id])

  if (!m) return <div className="wrap"><button onClick={onBack}>← back</button></div>

  const ups = checks.filter((c) => c.status === 'UP').length
  const uptime = checks.length ? ((100 * ups) / checks.length).toFixed(2) : '—'
  const avg = checks.length
    ? Math.round(checks.filter((c) => c.response_time_ms != null).reduce((a, c) => a + c.response_time_ms, 0) / Math.max(checks.filter((c) => c.response_time_ms != null).length, 1))
    : '—'

  return (
    <div className="wrap">
      <button onClick={onBack}>← back</button>
      <h2>{m.name} <Dot status={m.status} /></h2>
      <div className="muted">{m.url} · every {m.interval_seconds}s · timeout {m.timeout_seconds}s</div>
      <div className="stats">
        <div className="stat"><b>{uptime}%</b><span>uptime</span></div>
        <div className="stat"><b>{avg} ms</b><span>avg response</span></div>
        <div className="stat"><b>{m.status}</b><span>status</span></div>
      </div>
      <h3>Response time</h3>
      <Spark checks={checks} />
      <h3>Recent checks</h3>
      <table>
        <thead><tr><th>Time</th><th>Status</th><th>Code</th><th>Response</th></tr></thead>
        <tbody>
          {checks.map((c, i) => (
            <tr key={i}>
              <td>{new Date(c.checked_at).toLocaleTimeString()}</td>
              <td>{c.status === 'UP' ? '✓' : '✗'} {c.status}</td>
              <td>{c.status_code ?? '—'}</td>
              <td>{c.response_time_ms != null ? c.response_time_ms + ' ms' : (c.error || '—')}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <h3>Incidents</h3>
      {incidents.length === 0 && <div className="muted">none</div>}
      {incidents.map((in_) => (
        <div key={in_.id} className="card">
          <b>{in_.status}</b> — {in_.reason}<br />
          <span className="muted">{new Date(in_.started_at).toLocaleString()}
            {in_.resolved_at ? ' → ' + new Date(in_.resolved_at).toLocaleString() : ' (ongoing)'}</span>
        </div>
      ))}
    </div>
  )
}

export function Dot({ status }) {
  const color = status === 'UP' ? '#22c55e' : status === 'DOWN' ? '#ef4444' : '#9ca3af'
  return <span style={{ color }}>●</span>
}
