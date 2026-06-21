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

  async getPlans() {
    const res = await authFetch(`${API_BASE}/api/superadmin/billing/plans`)
    return res.json()
  },

  async updatePlan(planId: string, data: { price_monthly: number; price_yearly: number; is_active?: boolean; sort_order?: number }) {
    const res = await authFetch(`${API_BASE}/api/superadmin/billing/plans/${encodeURIComponent(planId)}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    })
    return res.json()
  },

  async listVouchers(params?: { plan_id?: string; used?: string; limit?: number }) {
    const qs = new URLSearchParams()
    if (params?.plan_id) qs.set('plan_id', params.plan_id)
    if (params?.used) qs.set('used', params.used)
    if (params?.limit) qs.set('limit', String(params.limit))
    const res = await authFetch(`${API_BASE}/api/superadmin/billing/vouchers?${qs}`, {
      headers: { 'Content-Type': 'application/json' },
    })
    return res.json()
  },

  async generateVouchers(data: { plan_id: string; validity_days: number; quantity: number; program_name?: string; max_uses?: number; voucher_type?: string; discount_value?: number }) {
    const res = await authFetch(`${API_BASE}/api/superadmin/billing/vouchers/generate`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    })
    return res.json()
  },

  async deleteVoucher(id: string) {
    const res = await authFetch(`${API_BASE}/api/superadmin/billing/vouchers?id=${encodeURIComponent(id)}`, {
      method: 'DELETE',
    })
    return res.json()
  },

  async getAddonPrices() {
    const res = await authFetch(`${API_BASE}/api/superadmin/billing/addon-prices`)
    return res.json()
  },

  async updateAddonPrice(addonKey: string, data: { price_cents?: number; unit?: string; is_active?: boolean; description?: string }) {
    const res = await authFetch(`${API_BASE}/api/superadmin/billing/addon-prices/${encodeURIComponent(addonKey)}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    })
    return res.json()
  },

  // F057: Feature Matrix & Addon Gating
  async getFeatureMatrix() {
    const res = await authFetch(`${API_BASE}/api/superadmin/billing/feature-matrix`)
    return res.json()
  },

  async toggleFeature(data: { plan_id: string; feature_key: string; is_enabled: boolean }) {
    const res = await authFetch(`${API_BASE}/api/superadmin/billing/feature-matrix`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    })
    return res.json()
  },

  async getAvailableFeatures() {
    const res = await authFetch(`${API_BASE}/api/superadmin/billing/available-features`)
    return res.json()
  },

  async upsertAvailableFeature(data: Record<string, any>) {
    const res = await authFetch(`${API_BASE}/api/superadmin/billing/available-features`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    })
    return res.json()
  },

  async updateAvailableFeature(featureKey: string, data: Record<string, any>) {
    const res = await authFetch(`${API_BASE}/api/superadmin/billing/available-features/${encodeURIComponent(featureKey)}`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    })
    return res.json()
  },

  async deleteAvailableFeature(featureKey: string) {
    const res = await authFetch(`${API_BASE}/api/superadmin/billing/available-features/${encodeURIComponent(featureKey)}`, {
      method: 'DELETE',
    })
    return res.json()
  },

  async getAddonGating() {
    const res = await authFetch(`${API_BASE}/api/superadmin/billing/addon-gating`)
    return res.json()
  },

  async updateAddonGating(data: { feature_key: string; min_tier?: string; default_enabled: string[] }) {
    const res = await authFetch(`${API_BASE}/api/superadmin/billing/addon-gating`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
    })
    return res.json()
  },

}
