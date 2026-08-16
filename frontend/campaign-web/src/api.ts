const API_GATEWAY_URL = 'http://localhost:8000';
const CAMPAIGN_API_BASE = `${API_GATEWAY_URL}/api/campaign`;
const AUTH_BASE = `${API_GATEWAY_URL}/auth`;
const WA_GATEWAY_URL = `${API_GATEWAY_URL}/api/wa`;

async function ensureAuthenticated() {
  let token = localStorage.getItem('accessToken');
  if (!token) {
    window.dispatchEvent(new Event('auth-required'));
  }
  return token;
}

async function retryWithBackoff<T>(
  fn: () => Promise<T>,
  maxRetries: number = 3,
  baseDelay: number = 300
): Promise<T> {
  let lastError: Error | undefined;

  for (let attempt = 0; attempt <= maxRetries; attempt++) {
    try {
      return await fn();
    } catch (error) {
      lastError = error as Error;

      if (attempt === maxRetries) break;

      const delay = baseDelay * Math.pow(2, attempt);
      const jitter = Math.random() * 100;
      await new Promise(resolve => setTimeout(resolve, delay + jitter));
    }
  }

  throw lastError;
}

export async function apiClient(endpoint: string, options: RequestInit = {}) {
  const token = await ensureAuthenticated();

  const headers = new Headers(options.headers || {});
  headers.set('Content-Type', 'application/json');
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }

  const path = endpoint.startsWith('/') ? endpoint : `/${endpoint}`;

  return retryWithBackoff(async () => {
    const response = await fetch(`${CAMPAIGN_API_BASE}${path}`, { ...options, headers });

    if (response.status === 401 || response.status === 403) {
      localStorage.removeItem('accessToken');
      localStorage.removeItem('refreshToken');
      localStorage.removeItem('tenantId');
      localStorage.removeItem('userName');
      localStorage.removeItem('userRole');
      window.dispatchEvent(new Event('auth-required'));
    }

    if (!response.ok && response.status >= 500) {
      throw new Error(`Server error: ${response.status}`);
    }

    return response;
  });
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
  async getDashboard(regionType: string = 'nasional', regionId: string = '') {
    const url = new URL(`${API_GATEWAY_URL}/api/public/campaign/dashboard`);
    if (regionType !== 'nasional') {
      url.searchParams.append('region_type', regionType);
      if (regionId) url.searchParams.append('region_id', regionId);
    }
    return fetch(url.toString());
  },

  // F037: Affiliate leaderboard (public)
  async getAffiliateLeaderboard() {
    return apiClient('/affiliate/leaderboard');
  }
};

// F037: Affiliate referral API
export const affiliateApi = {
  async redeemReferral(referralCode: string) {
    return apiClient('/affiliate/redeem-referral', {
      method: 'POST',
      body: JSON.stringify({ referral_code: referralCode })
    });
  }
};

export const waApi = {
  async status(tenantId: string) {
    const token = localStorage.getItem('accessToken');
    const headers = new Headers();
    if (token) {
      headers.set('Authorization', `Bearer ${token}`);
    }
    return fetch(`${WA_GATEWAY_URL}/status?tenant_id=${tenantId}`, { headers });
  },

  async qr(tenantId: string) {
    const token = localStorage.getItem('accessToken');
    const headers = new Headers();
    if (token) {
      headers.set('Authorization', `Bearer ${token}`);
    }
    return fetch(`${WA_GATEWAY_URL}/qr?tenant_id=${tenantId}`, { headers });
  }
};
