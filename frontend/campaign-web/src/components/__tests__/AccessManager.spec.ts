import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import AccessManager from '../AccessManager.vue'

// Mock fetch globally
global.fetch = vi.fn()

describe('AccessManager.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
  })

  describe('Component Rendering', () => {
    it('renders the main heading', () => {
      const wrapper = mount(AccessManager)
      expect(wrapper.find('h2').text()).toBe('Pengaturan & Integrasi')
    })

    it('renders Telegram Bot API section', () => {
      const wrapper = mount(AccessManager)
      expect(wrapper.text()).toContain('Telegram Bot API')
      expect(wrapper.find('#telegram-key-input').exists()).toBe(true)
    })

    it('renders WhatsApp Gateway section', () => {
      const wrapper = mount(AccessManager)
      expect(wrapper.text()).toContain('WhatsApp Gateway')
    })
  })

  describe('Telegram Token Management', () => {
    it('binds input to telegramKey model', async () => {
      const wrapper = mount(AccessManager)
      const input = wrapper.find('#telegram-key-input')

      await input.setValue('123456789:ABCdefGHIjkl')
      expect((input.element as HTMLInputElement).value).toBe('123456789:ABCdefGHIjkl')
    })

    it('calls saveTelegramKey when save button is clicked', async () => {
      const mockFetch = vi.fn().mockResolvedValue({
        ok: true,
        json: async () => ({ success: true })
      })
      global.fetch = mockFetch

      const wrapper = mount(AccessManager)
      const input = wrapper.find('#telegram-key-input')
      await input.setValue('test-token')

      const saveButton = wrapper.findAll('button').find(btn =>
        btn.text().includes('Simpan Token')
      )
      await saveButton?.trigger('click')
      await new Promise(resolve => setTimeout(resolve, 10))

      expect(mockFetch).toHaveBeenCalled()
    })

    it('handles save error gracefully', async () => {
      const mockFetch = vi.fn().mockRejectedValue(new Error('Network error'))
      global.fetch = mockFetch

      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

      const wrapper = mount(AccessManager)
      const input = wrapper.find('#telegram-key-input')
      await input.setValue('test-token')

      const saveButton = wrapper.findAll('button').find(btn =>
        btn.text().includes('Simpan Token')
      )
      await saveButton?.trigger('click')

      await wrapper.vm.$nextTick()
      expect(consoleSpy).toHaveBeenCalled()
      consoleSpy.mockRestore()
    })
  })

  describe('WhatsApp Status Display', () => {
    it('shows QR button when WA not connected', async () => {
      const mockFetch = vi.fn()
        .mockResolvedValueOnce({ ok: true, json: async () => ({ success: true, data: {} }) })
        .mockResolvedValueOnce({ ok: true, json: async () => ({ success: true, data: [] }) })
        .mockResolvedValueOnce({ ok: true, json: async () => ({ success: true, data: { status: 'disconnected' } }) })

      global.fetch = mockFetch

      const wrapper = mount(AccessManager)
      await new Promise(resolve => setTimeout(resolve, 10))

      const qrButton = wrapper.findAll('button').find(btn =>
        btn.text().includes('Tampilkan QR Code WA')
      )
      expect(qrButton?.exists()).toBe(true)
    })

    it('triggers QR generation when button clicked', async () => {
      const mockFetch = vi.fn()
        .mockResolvedValueOnce({ ok: true, json: async () => ({ success: true, data: {} }) })
        .mockResolvedValueOnce({ ok: true, json: async () => ({ success: true, data: [] }) })
        .mockResolvedValueOnce({ ok: true, json: async () => ({ success: true, data: { status: 'disconnected' } }) })
        .mockResolvedValueOnce({ ok: true, json: async () => ({ success: true, data: { qr: 'data:image/png;base64,test' } }) })

      global.fetch = mockFetch

      const wrapper = mount(AccessManager)
      await new Promise(resolve => setTimeout(resolve, 20))

      const qrButton = wrapper.findAll('button').find(btn =>
        btn.text().includes('Tampilkan QR Code WA')
      )
      await qrButton?.trigger('click')
      await new Promise(resolve => setTimeout(resolve, 20))

      // Verify the QR endpoint was called
      expect(mockFetch).toHaveBeenCalledTimes(4)
      expect(mockFetch).toHaveBeenLastCalledWith(
        expect.stringContaining('/api/wa/qr'),
        expect.any(Object)
      )
    })
  })

  describe('QR Generation', () => {
    it('handles QR generation error gracefully', async () => {
      const mockFetch = vi.fn()
        .mockResolvedValueOnce({ ok: true, json: async () => ({ success: true, data: {} }) })
        .mockResolvedValueOnce({ ok: true, json: async () => ({ success: true, data: [] }) })
        .mockResolvedValueOnce({ ok: true, json: async () => ({ success: true, data: { status: 'disconnected' } }) })
        .mockRejectedValueOnce(new Error('Gateway error'))

      global.fetch = mockFetch

      const alertSpy = vi.spyOn(window, 'alert').mockImplementation(() => {})
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

      const wrapper = mount(AccessManager)
      await new Promise(resolve => setTimeout(resolve, 10))

      const qrButton = wrapper.findAll('button').find(btn =>
        btn.text().includes('Tampilkan QR Code WA')
      )
      await qrButton?.trigger('click')
      await new Promise(resolve => setTimeout(resolve, 10))

      expect(alertSpy).toHaveBeenCalledWith('Gagal menghubungi WA Gateway')
      expect(consoleSpy).toHaveBeenCalled()

      alertSpy.mockRestore()
      consoleSpy.mockRestore()
    })
  })

  describe('Data Fetching on Mount', () => {
    it('fetches initial data when component mounts', async () => {
      const mockFetch = vi.fn()
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ success: true, data: { telegram_bot_token: 'existing-token' } })
        })
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ success: true, data: [] })
        })
        .mockResolvedValueOnce({
          ok: true,
          json: async () => ({ success: true, data: { status: 'disconnected' } })
        })

      global.fetch = mockFetch

      mount(AccessManager)
      await new Promise(resolve => setTimeout(resolve, 0))

      expect(mockFetch).toHaveBeenCalledTimes(3)
    })
  })
})
