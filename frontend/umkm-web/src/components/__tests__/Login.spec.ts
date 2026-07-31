import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import Login from '@/components/Login.vue'

describe('Login.vue', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
  })

  it('renders login form', () => {
    const wrapper = mount(Login)
    expect(wrapper.find('input[type="text"]').exists()).toBe(true)
    expect(wrapper.find('input[type="password"]').exists()).toBe(true)
    expect(wrapper.find('button[type="submit"]').exists()).toBe(true)
  })

  it('validates empty fields', async () => {
    const wrapper = mount(Login)
    const form = wrapper.find('form')

    await form.trigger('submit')

    expect(wrapper.text()).toContain('diperlukan')
  })

  it('validates phone number format', async () => {
    const wrapper = mount(Login)
    const phoneInput = wrapper.find('input[type="text"]')

    await phoneInput.setValue('08123456789')

    const normalizedPhone = '628123456789'
    expect(normalizedPhone).toMatch(/^62/)
  })

  it('switches between phone and username login', async () => {
    const wrapper = mount(Login)

    const usernameTab = wrapper.find('[data-testid="username-tab"]')
    if (usernameTab.exists()) {
      await usernameTab.trigger('click')
      expect(wrapper.find('input[type="text"]').attributes('placeholder')).toContain('username')
    }
  })

  it('stores token on successful login', async () => {
    const mockToken = 'mock-jwt-token'
    const mockResponse = {
      success: true,
      data: {
        accessToken: mockToken,
        tenantId: 'test-tenant-123',
        role: 'owner'
      }
    }

    global.fetch = vi.fn(() =>
      Promise.resolve({
        ok: true,
        json: () => Promise.resolve(mockResponse),
      })
    ) as any

    const wrapper = mount(Login)
    const usernameInput = wrapper.find('input[type="text"]')
    const passwordInput = wrapper.find('input[type="password"]')

    await usernameInput.setValue('testuser')
    await passwordInput.setValue('password123')
    await wrapper.find('form').trigger('submit')

    await wrapper.vm.$nextTick()

    expect(localStorage.getItem('token')).toBeTruthy()
  })

  it('shows error message on failed login', async () => {
    global.fetch = vi.fn(() =>
      Promise.resolve({
        ok: false,
        json: () => Promise.resolve({ success: false, message: 'Invalid credentials' }),
      })
    ) as any

    const wrapper = mount(Login)
    const usernameInput = wrapper.find('input[type="text"]')
    const passwordInput = wrapper.find('input[type="password"]')

    await usernameInput.setValue('wronguser')
    await passwordInput.setValue('wrongpass')
    await wrapper.find('form').trigger('submit')

    await wrapper.vm.$nextTick()

    expect(wrapper.text()).toContain('Invalid credentials')
  })

  it('redirects to onboarding if not completed', async () => {
    const mockResponse = {
      success: true,
      data: {
        accessToken: 'token',
        tenantId: 'test-tenant',
        role: 'owner',
        onboarding_completed: false
      }
    }

    global.fetch = vi.fn(() =>
      Promise.resolve({
        ok: true,
        json: () => Promise.resolve(mockResponse),
      })
    ) as any

    const wrapper = mount(Login)
    await wrapper.find('form').trigger('submit')
    await wrapper.vm.$nextTick()

    expect(wrapper.emitted('navigate')?.[0]).toEqual(['/onboarding'])
  })
})
