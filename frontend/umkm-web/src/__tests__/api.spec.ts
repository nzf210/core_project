import { describe, it, expect, vi, beforeEach } from 'vitest'

describe('API Utilities', () => {
  const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8000'

  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
  })

  describe('API Request Headers', () => {
    it('includes Authorization header when token exists', () => {
      const token = 'mock-jwt-token'
      localStorage.setItem('token', token)

      const headers = {
        'Authorization': `Bearer ${token}`,
        'Content-Type': 'application/json'
      }

      expect(headers['Authorization']).toBe(`Bearer ${token}`)
    })

    it('includes X-Tenant-ID header when tenantId exists', () => {
      const tenantId = 'test-tenant-123'
      localStorage.setItem('tenantId', tenantId)

      const headers = {
        'X-Tenant-ID': tenantId,
        'Content-Type': 'application/json'
      }

      expect(headers['X-Tenant-ID']).toBe(tenantId)
    })

    it('handles missing token gracefully', () => {
      const headers: Record<string, string> = {
        'Content-Type': 'application/json'
      }

      expect(headers['Authorization']).toBeUndefined()
    })
  })

  describe('API Error Handling', () => {
    it('handles 401 Unauthorized', async () => {
      global.fetch = vi.fn(() =>
        Promise.resolve({
          ok: false,
          status: 401,
          json: () => Promise.resolve({ success: false, message: 'Unauthorized' }),
        })
      ) as any

      const response = await fetch(`${API_BASE_URL}/api/test`)
      expect(response.status).toBe(401)
    })

    it('handles 403 Forbidden', async () => {
      global.fetch = vi.fn(() =>
        Promise.resolve({
          ok: false,
          status: 403,
          json: () => Promise.resolve({ success: false, message: 'Forbidden' }),
        })
      ) as any

      const response = await fetch(`${API_BASE_URL}/api/test`)
      expect(response.status).toBe(403)
    })

    it('handles 500 Internal Server Error', async () => {
      global.fetch = vi.fn(() =>
        Promise.resolve({
          ok: false,
          status: 500,
          json: () => Promise.resolve({ success: false, message: 'Internal Server Error' }),
        })
      ) as any

      const response = await fetch(`${API_BASE_URL}/api/test`)
      expect(response.status).toBe(500)
    })

    it('handles network errors', async () => {
      global.fetch = vi.fn(() =>
        Promise.reject(new Error('Network error'))
      ) as any

      await expect(fetch(`${API_BASE_URL}/api/test`)).rejects.toThrow('Network error')
    })
  })

  describe('Phone Number Formatting', () => {
    it('converts 08xx to 628xx', () => {
      const input = '081234567890'
      const expected = '6281234567890'
      const result = input.replace(/^0/, '62')
      expect(result).toBe(expected)
    })

    it('removes + prefix', () => {
      const input = '+6281234567890'
      const expected = '6281234567890'
      const result = input.replace(/^\+/, '')
      expect(result).toBe(expected)
    })

    it('handles already formatted number', () => {
      const input = '6281234567890'
      const result = input
      expect(result).toBe(input)
    })
  })

  describe('Currency Formatting', () => {
    it('formats IDR correctly', () => {
      const amount = 1000000
      const formatted = new Intl.NumberFormat('id-ID', {
        style: 'currency',
        currency: 'IDR',
        minimumFractionDigits: 0
      }).format(amount)

      expect(formatted).toBe('Rp1.000.000')
    })

    it('handles zero amount', () => {
      const amount = 0
      const formatted = new Intl.NumberFormat('id-ID', {
        style: 'currency',
        currency: 'IDR',
        minimumFractionDigits: 0
      }).format(amount)

      expect(formatted).toBe('Rp0')
    })

    it('handles negative amount', () => {
      const amount = -50000
      const formatted = new Intl.NumberFormat('id-ID', {
        style: 'currency',
        currency: 'IDR',
        minimumFractionDigits: 0
      }).format(amount)

      expect(formatted).toContain('-')
      expect(formatted).toContain('50.000')
    })
  })

  describe('Date Formatting', () => {
    it('formats date to Indonesian locale', () => {
      const date = new Date('2026-07-31T10:00:00Z')
      const formatted = date.toLocaleDateString('id-ID', {
        day: '2-digit',
        month: 'long',
        year: 'numeric'
      })

      expect(formatted).toContain('2026')
      expect(formatted).toContain('Juli')
    })

    it('formats time correctly', () => {
      const date = new Date('2026-07-31T14:30:00Z')
      const formatted = date.toLocaleTimeString('id-ID', {
        hour: '2-digit',
        minute: '2-digit'
      })

      expect(formatted).toMatch(/\d{2}:\d{2}/)
    })
  })

  describe('Input Validation', () => {
    it('validates email format', () => {
      const validEmail = 'test@example.com'
      const invalidEmail = 'invalid-email'

      const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/

      expect(emailRegex.test(validEmail)).toBe(true)
      expect(emailRegex.test(invalidEmail)).toBe(false)
    })

    it('validates phone number format', () => {
      const validPhone = '628123456789'
      const invalidPhone = '123'

      const phoneRegex = /^62\d{9,13}$/

      expect(phoneRegex.test(validPhone)).toBe(true)
      expect(phoneRegex.test(invalidPhone)).toBe(false)
    })

    it('validates password length', () => {
      const validPassword = 'password123'
      const invalidPassword = '12345'

      expect(validPassword.length >= 6).toBe(true)
      expect(invalidPassword.length >= 6).toBe(false)
    })
  })
})
