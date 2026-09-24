const API = import.meta.env.VITE_API_URL || 'http://localhost:8080'

function authHeader() {
  const t = localStorage.getItem('pulse_token')
  return t ? { Authorization: 'Bearer ' + t } : {}
}

async function req(path, opts = {}) {
  const res = await fetch(API + path, {
    ...opts,
    headers: { 'Content-Type': 'application/json', ...authHeader(), ...(opts.headers || {}) },
  })
  const body = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(body.error || 'request failed')
  return body.data
}

export const api = {
  login: (email, password) => req('/api/v1/auth/login', { method: 'POST', body: JSON.stringify({ email, password }) }),
  register: (email, password, name) => req('/api/v1/auth/register', { method: 'POST', body: JSON.stringify({ email, password, name }) }),
  summary: () => req('/api/v1/dashboard/summary'),
  monitors: () => req('/api/v1/monitors'),
  createMonitor: (m) => req('/api/v1/monitors', { method: 'POST', body: JSON.stringify(m) }),
  pause: (id) => req(`/api/v1/monitors/${id}/pause`, { method: 'POST' }),
  resume: (id) => req(`/api/v1/monitors/${id}/resume`, { method: 'POST' }),
  remove: (id) => req(`/api/v1/monitors/${id}`, { method: 'DELETE' }),
  checks: (id) => req(`/api/v1/monitors/${id}/checks?limit=20`),
  monitorIncidents: (id) => req(`/api/v1/monitors/${id}/incidents`),
  incidents: () => req('/api/v1/incidents'),
  me: () => req('/api/v1/auth/me'),
}
