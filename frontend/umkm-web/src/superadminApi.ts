const API_BASE = import.meta.env.VITE_API_URL || (import.meta.env.PROD ? window.location.origin : 'http://localhost:8000')

function getToken(): string {
  return localStorage.getItem('access_token') || ''
}

function getRefreshToken(): string {
  return localStorage.getItem('refresh_token') || ''
}

function setTokens(data: { accessToken: string; refreshToken: string }) {
  localStorage.setItem('access_token', data.accessToken)
  localStorage.setItem('refresh_token', data.refreshToken)
}

let isRefreshing = false
let refreshPromise: Promise<boolean> | null = null

async function tryRefreshToken(): Promise<boolean> {
  if (isRefreshing && refreshPromise) return refreshPromise

  isRefreshing = true
  refreshPromise = (async () => {
    try {
      const res = await fetch(`${API_BASE}/auth/refresh`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refreshToken: getRefreshToken() }),
      })
      const data = await res.json()
      if (data.success && data.data) {
        setTokens(data.data)
        return true
      }
      return false
    } catch {
      return false
    } finally {
      isRefreshing = false
      refreshPromise = null
    }
  })()

  return refreshPromise
}

async function authFetch(url: string, options: RequestInit = {}): Promise<Response> {
  const res = await fetch(url, {
    ...options,
    headers: {
      ...(options.headers || {}),
      'Authorization': `Bearer ${getToken()}`,
    },
  })

  if (res.status === 401) {
    const refreshed = await tryRefreshToken()
    if (refreshed) {
      return fetch(url, {
        ...options,
        headers: {
          ...(options.headers || {}),
          'Authorization': `Bearer ${getToken()}`,
        },
      })
    } else {
      localStorage.clear()
      window.location.href = '/superadmin/login'
      throw new Error('Session expired')
    }
  }

  return res
}

export const superadminApi = {
  async login(username: string, password: string) {
    const res = await fetch(`${API_BASE}/auth/superadmin/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password }),
    })
    return res.json()
  },

  async getVerifierStatus() {
    const res = await authFetch(`${API_BASE}/api/superadmin/verifier/status`)
    return res.json()
  },

  async getVerifierQR() {
    const res = await authFetch(`${API_BASE}/api/superadmin/verifier/qr`)
    return res.json()
  },

  async disconnectVerifier() {
    const res = await authFetch(`${API_BASE}/api/superadmin/verifier/disconnect`, { method: 'POST' })
    return res.json()
  },

  async getTenants() {
    const res = await authFetch(`${API_BASE}/api/superadmin/tenants`, {
      headers: { 'Content-Type': 'application/json' },
    })
    return res.json()
  },

  async deleteTenant(tenantId: string) {
    const res = await authFetch(`${API_BASE}/api/superadmin/tenants?id=${encodeURIComponent(tenantId)}`, {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' },
    })
    return res.json()
  },

  async getTenantProfile(tenantId: string) {
    const res = await authFetch(`${API_BASE}/api/superadmin/tenants/profile?id=${encodeURIComponent(tenantId)}`)
    return res.json()
  },

  async updateTenantProfile(data: Record<string, any>) {
    const res = await authFetch(`${API_BASE}/api/superadmin/tenants/profile`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    })
    return res.json()
  },

  async uploadTenantLogo(tenantId: string, file: File) {
    const formData = new FormData()
    formData.append('logo', file)
    const res = await authFetch(`${API_BASE}/api/superadmin/tenants/profile/logo?id=${encodeURIComponent(tenantId)}`, {
      method: 'POST',
      body: formData,
    })
    return res.json()
  },
}
