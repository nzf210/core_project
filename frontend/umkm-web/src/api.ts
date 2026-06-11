export const API_BASE = import.meta.env.VITE_API_URL || (import.meta.env.PROD ? window.location.origin : 'http://localhost:8000')

export async function initDomain() {
  const hostname = window.location.hostname
  try {
    const res = await fetch(`${API_BASE}/auth/public/tenant/resolve?domain=${hostname}`)
    if (res.ok) {
      const data = await res.json()
      if (data.success && data.data) {
        localStorage.setItem('active_domain_tenant_id', data.data.tenant_id)
        localStorage.setItem('active_domain_business_name', data.data.business_name)
        localStorage.setItem('active_domain_logo_url', data.data.logo_url)
        return
      }
    }
    // If not successful or not found, remove
    localStorage.removeItem('active_domain_tenant_id')
    localStorage.removeItem('active_domain_business_name')
    localStorage.removeItem('active_domain_logo_url')
  } catch(e) {
    console.error('Failed to resolve domain', e)
  }
}

function getTenantID(): string {
  return localStorage.getItem('tenant_id') || ''
}

function getToken(): string {
  return localStorage.getItem('access_token') || ''
}

function headers(withAuth = true): Record<string, string> {
  const h: Record<string, string> = {
    'Content-Type': 'application/json',
    'X-Tenant-ID': getTenantID(),
  }
  if (withAuth) {
    h['Authorization'] = `Bearer ${getToken()}`
  }
  return h
}

export const api = {
  // Auth (unauthenticated)
  async login(username: string, password: string) {
    const expectedTenantId = localStorage.getItem('active_domain_tenant_id') || ''
    const res = await fetch(`${API_BASE}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password, expectedTenantId }),
    })
    return res.json()
  },

  async phoneLogin(phoneNumber: string) {
    const res = await fetch(`${API_BASE}/auth/phone-login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ phoneNumber }),
    })
    return res.json()
  },

  async verifyPhoneLogin(phoneNumber: string, otp: string) {
    const expectedTenantId = localStorage.getItem('active_domain_tenant_id') || ''
    const res = await fetch(`${API_BASE}/auth/verify-phone-login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ phoneNumber, otp, expectedTenantId }),
    })
    return res.json()
  },

  async registerWA(body: Record<string, any>) {
    const res = await fetch(`${API_BASE}/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        phoneNumber: body.phoneNumber,
        password: body.password,
        username: body.username,
        email: body.email || '',
        businessName: body.businessName || '',
      }),
    })
    return res.json()
  },

  async verifyOTP(phoneNumber: string, otp: string) {
    const res = await fetch(`${API_BASE}/auth/verify-otp`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ phoneNumber, otp }),
    })
    return res.json()
  },

  async register(body: Record<string, any>) {
    const res = await fetch(`${API_BASE}/api/umkm/admin/tenants`, {
      method: 'POST',
      headers: headers(),
      body: JSON.stringify(body),
    })
    return res.json()
  },

  async forgotPassword(email: string) {
    const res = await fetch(`${API_BASE}/auth/forgot-password`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email }),
    })
    return res.json()
  },

  async resetPassword(token: string, newPassword: string) {
    const res = await fetch(`${API_BASE}/auth/reset-password`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token, newPassword }),
    })
    return res.json()
  },

  // UMKM Data (authenticated + tenant)
  async get(url: string) {
    const res = await fetch(`${API_BASE}${url}`, { headers: headers() })
    return res.json()
  },

  async post(url: string, body?: any, isMultipart = false) {
    const h = headers()
    if (isMultipart) delete h['Content-Type']
    const res = await fetch(`${API_BASE}${url}`, {
      method: 'POST',
      headers: h,
      body: isMultipart ? body : (body ? JSON.stringify(body) : undefined),
    })
    return res.json()
  },

  async put(url: string, body?: any) {
    const res = await fetch(`${API_BASE}${url}`, {
      method: 'PUT',
      headers: headers(),
      body: body ? JSON.stringify(body) : undefined,
    })
    return res.json()
  },

  async del(url: string) {
    const res = await fetch(`${API_BASE}${url}`, { method: 'DELETE', headers: headers() })
    return res.json()
  },

  // WA Gateway
  async wa(method: string, body?: Record<string, any>) {
    const waGateway = import.meta.env.VITE_WA_GATEWAY_URL || 'http://localhost:8202'
    const res = await fetch(`${waGateway}/api/wa/${method}`, {
      method: body ? 'POST' : 'GET',
      headers: body ? { 'Content-Type': 'application/json' } : {},
      body: body ? JSON.stringify(body) : undefined,
    })
    return res.json()
  },
}
