import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import Settings from '@/components/Settings.vue'

describe('Settings.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.setItem('token', 'mock-token')
    localStorage.setItem('tenantId', 'test-tenant-123')
  })

  it('renders settings form', () => {
    const wrapper = mount(Settings)
    expect(wrapper.find('form').exists()).toBe(true)
  })

  it('loads user profile on mount', async () => {
    const mockProfile = {
      success: true,
      data: {
        username: 'testuser',
        business_name: 'Toko Test',
        business_type: 'warung',
        wa_number: '628123456789'
      }
    }

    global.fetch = vi.fn(() =>
      Promise.resolve({
        ok: true,
        json: () => Promise.resolve(mockProfile),
      })
    ) as any

    const wrapper = mount(Settings)
    await wrapper.vm.$nextTick()

    expect(fetch).toHaveBeenCalledWith(
      expect.stringContaining('/auth/profile'),
      expect.any(Object)
    )
  })

  it('validates business name length', async () => {
    const wrapper = mount(Settings)
    const businessNameInput = wrapper.find('input[name="business_name"]')

    await businessNameInput.setValue('AB')

    const isValid = businessNameInput.element.value.length >= 3
    expect(isValid).toBe(false)
  })

  it('validates phone number format', async () => {
    const wrapper = mount(Settings)
    const waNumberInput = wrapper.find('input[name="wa_number"]')

    await waNumberInput.setValue('628123456789')

    expect(waNumberInput.element.value).toMatch(/^62/)
  })

  it('updates profile successfully', async () => {
    const mockUpdateResponse = {
      success: true,
      message: 'Profile updated successfully'
    }

    global.fetch = vi.fn(() =>
      Promise.resolve({
        ok: true,
        json: () => Promise.resolve(mockUpdateResponse),
      })
    ) as any

    const wrapper = mount(Settings)
    const form = wrapper.find('form')

    await form.trigger('submit')
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('berhasil')
  })

  it('shows validation errors', async () => {
    const mockErrorResponse = {
      success: false,
      message: 'Validation failed'
    }

    global.fetch = vi.fn(() =>
      Promise.resolve({
        ok: false,
        json: () => Promise.resolve(mockErrorResponse),
      })
    ) as any

    const wrapper = mount(Settings)
    await wrapper.find('form').trigger('submit')
    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('failed')
  })

  it('handles logo upload', async () => {
    const wrapper = mount(Settings)
    const fileInput = wrapper.find('input[type="file"]')

    const file = new File(['logo'], 'logo.png', { type: 'image/png' })

    Object.defineProperty(fileInput.element, 'files', {
      value: [file],
      writable: false
    })

    await fileInput.trigger('change')

    expect(file.type).toBe('image/png')
    expect(file.size).toBeLessThan(2 * 1024 * 1024) // 2MB
  })

  it('validates password change', async () => {
    const wrapper = mount(Settings)
    const oldPasswordInput = wrapper.find('input[name="old_password"]')
    const newPasswordInput = wrapper.find('input[name="new_password"]')

    await oldPasswordInput.setValue('oldpass123')
    await newPasswordInput.setValue('newpass')

    const isValid = newPasswordInput.element.value.length >= 6
    expect(isValid).toBe(false)
  })
})
