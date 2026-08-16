import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import RealCount from '../RealCount.vue'

// Mock fetch globally
global.fetch = vi.fn()

describe('RealCount.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('Component Rendering', () => {
    it('renders the main heading', () => {
      const wrapper = mount(RealCount)
      expect(wrapper.find('h2').text()).toBe('Real Count C1 & Saksi')
    })

    it('renders all KPI stat cards', () => {
      const wrapper = mount(RealCount)
      expect(wrapper.text()).toContain('Suara Paslon Kita')
      expect(wrapper.text()).toContain('Suara Lawan')
      expect(wrapper.text()).toContain('Suara Tidak Sah / Batal')
      expect(wrapper.text()).toContain('Data TPS Masuk')
    })

    it('renders tab navigation buttons', () => {
      const wrapper = mount(RealCount)
      const tabs = wrapper.findAll('.tab-btn')

      expect(tabs.length).toBeGreaterThanOrEqual(2)
      expect(wrapper.text()).toContain('Dashboard Hasil')
      expect(wrapper.text()).toContain('Absensi Saksi')
    })
  })

  describe('Stats Display', () => {
    it('displays initial zero values', () => {
      const wrapper = mount(RealCount)

      expect(wrapper.text()).toContain('0')
    })

    it('displays TPS progress text', () => {
      const wrapper = mount(RealCount)

      expect(wrapper.text()).toContain('/ 0')
      expect(wrapper.text()).toContain('% Selesai')
    })
  })

  describe('Tab Switching', () => {
    it('renders tab buttons', () => {
      const wrapper = mount(RealCount)

      const dashboardTab = wrapper.findAll('.tab-btn').find(btn =>
        btn.text().includes('Dashboard Hasil')
      )

      expect(dashboardTab?.exists()).toBe(true)
    })

    it('switches to saksi tab when clicked', async () => {
      const wrapper = mount(RealCount)

      const saksiTab = wrapper.findAll('.tab-btn').find(btn =>
        btn.text().includes('Absensi Saksi')
      )

      await saksiTab?.trigger('click')
      await wrapper.vm.$nextTick()

      expect(saksiTab?.classes()).toContain('active')
    })
  })

  describe('Data Fetching', () => {
    it('calls API when refresh button is clicked', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          success: true,
          data: {
            stats: { candidate_votes: 100 },
            results: []
          }
        })
      })
      global.fetch = mockFetch

      const wrapper = mount(RealCount)

      const refreshButton = wrapper.findAll('button').find(btn =>
        btn.text().includes('Refresh Data')
      )

      await refreshButton?.trigger('click')

      expect(mockFetch).toHaveBeenCalled()
    })
  })

  describe('Loading State', () => {
    it('renders refresh button', () => {
      const wrapper = mount(RealCount)

      const refreshButton = wrapper.findAll('button').find(btn =>
        btn.text().includes('Refresh') || btn.text().includes('Memuat')
      )

      expect(refreshButton?.exists()).toBe(true)
    })
  })

  describe('Component Lifecycle', () => {
    it('fetches data on mount', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({
          success: true,
          data: { stats: {}, results: [] }
        })
      })
      global.fetch = mockFetch

      mount(RealCount)
      await new Promise(resolve => setTimeout(resolve, 0))

      expect(mockFetch).toHaveBeenCalled()
    })
  })

  describe('Edge Cases', () => {
    it('renders without crashing', () => {
      const wrapper = mount(RealCount)

      expect(wrapper.exists()).toBe(true)
      expect(wrapper.find('h2').exists()).toBe(true)
    })
  })
})
