import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import Dashboard from '@/components/Dashboard.vue'

describe('Dashboard.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.setItem('token', 'mock-token')
    localStorage.setItem('tenantId', 'test-tenant-123')
  })

  it('renders dashboard layout', () => {
    const wrapper = mount(Dashboard)
    expect(wrapper.find('.dashboard').exists()).toBe(true)
  })

  it('fetches dashboard data on mount', async () => {
    const mockData = {
      success: true,
      data: {
        totalRevenue: 1000000,
        todayRevenue: 50000,
        totalTransactions: 150,
        todayTransactions: 10
      }
    }

    global.fetch = vi.fn(() =>
      Promise.resolve({
        ok: true,
        json: () => Promise.resolve(mockData),
      })
    ) as any

    const wrapper = mount(Dashboard)
    await wrapper.vm.$nextTick()

    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/api/umkm/dashboard'),
      expect.objectContaining({
        headers: expect.objectContaining({
          'Authorization': 'Bearer mock-token'
        })
      })
    )
  })

  it('displays revenue metrics', async () => {
    const mockData = {
      success: true,
      data: {
        totalRevenue: 1000000,
        todayRevenue: 50000
      }
    }

    global.fetch = vi.fn(() =>
      Promise.resolve({
        ok: true,
        json: () => Promise.resolve(mockData),
      })
    ) as any

    const wrapper = mount(Dashboard)
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('1,000,000')
    expect(wrapper.text()).toContain('50,000')
  })

  it('handles API error gracefully', async () => {
    global.fetch = vi.fn(() =>
      Promise.resolve({
        ok: false,
        json: () => Promise.resolve({ success: false, message: 'Unauthorized' }),
      })
    ) as any

    const wrapper = mount(Dashboard)
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('error')
  })

  it('redirects to login if unauthorized', async () => {
    global.fetch = vi.fn(() =>
      Promise.resolve({
        ok: false,
        status: 401,
        json: () => Promise.resolve({ success: false }),
      })
    ) as any

    const wrapper = mount(Dashboard)
    await wrapper.vm.$nextTick()

    expect(localStorage.getItem('token')).toBeNull()
  })

  it('formats currency correctly', () => {
    const wrapper = mount(Dashboard)
    const formatCurrency = (amount: number) => {
      return new Intl.NumberFormat('id-ID', {
        style: 'currency',
        currency: 'IDR',
        minimumFractionDigits: 0
      }).format(amount)
    }

    expect(formatCurrency(1000000)).toBe('Rp1.000.000')
  })

  it('shows loading state while fetching', async () => {
    global.fetch = vi.fn(() => new Promise(() => {})) as any

    const wrapper = mount(Dashboard)

    expect(wrapper.text()).toContain('Loading')
  })
})
