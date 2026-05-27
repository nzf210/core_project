const API_GATEWAY_URL = 'http://localhost:8000';
const CAMPAIGN_API_BASE = `${API_GATEWAY_URL}/api/campaign`;
const AUTH_BASE = `${API_GATEWAY_URL}/auth`;
const WA_GATEWAY_URL = 'http://localhost:8202/api/wa';

async function ensureAuthenticated() {
  let token = localStorage.getItem('accessToken');
  if (!token) {
    window.dispatchEvent(new Event('auth-required'));
  }
  return token;
}

export async function apiClient(endpoint: string, options: RequestInit = {}) {
  const token = await ensureAuthenticated();
  
  const headers = new Headers(options.headers || {});
  headers.set('Content-Type', 'application/json');
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }
  
  const path = endpoint.startsWith('/') ? endpoint : `/${endpoint}`;
  const response = await fetch(`${CAMPAIGN_API_BASE}${path}`, { ...options, headers });

  if (response.status === 401 || response.status === 403) {
    localStorage.removeItem('accessToken');
    localStorage.removeItem('refreshToken');
    localStorage.removeItem('tenantId');
    localStorage.removeItem('userName');
    localStorage.removeItem('userRole');
    window.dispatchEvent(new Event('auth-required'));
  }
  return response;
}

export const authApi = {
  async login(username: string, password: string) {
    return fetch(`${AUTH_BASE}/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password })
    });
  },

  async register(body: Record<string, any>) {
    return fetch(`${AUTH_BASE}/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    });
  },

  async verifyOTP(phoneNumber: string, otp: string) {
    return fetch(`${AUTH_BASE}/verify-otp`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ phoneNumber, otp })
    });
  },

  async verifyData(token: string, body: Record<string, any>) {
    return fetch(`${AUTH_BASE}/verify-data`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify(body)
    });
  },

  async manualRegister(token: string, body: Record<string, any>) {
    return fetch(`${AUTH_BASE}/manual-register`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${token}`
      },
      body: JSON.stringify(body)
    });
  },

  async forgotPassword(email: string) {
    return fetch(`${AUTH_BASE}/forgot-password`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email })
    });
  }
};

export const publicApi = {
  async getDashboard() {
    return fetch(`${API_GATEWAY_URL}/api/public/campaign/dashboard`);
  }
};

export const waApi = {
  async status(tenantId: string) {
    return fetch(`${WA_GATEWAY_URL}/status?tenant_id=${tenantId}`);
  },

  async qr(tenantId: string) {
    return fetch(`${WA_GATEWAY_URL}/qr?tenant_id=${tenantId}`);
  }
};
