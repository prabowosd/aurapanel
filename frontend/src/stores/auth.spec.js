import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'

vi.mock('../services/api', () => ({
  default: {
    post: vi.fn(),
  },
}))

import api from '../services/api'
import { useAuthStore } from './auth'

describe('auth store', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    sessionStorage.clear()
    vi.clearAllMocks()
  })

  it('stores token in session storage by default', () => {
    const store = useAuthStore()
    store.setAuth('session-token', { role: 'admin' }, false)

    expect(sessionStorage.getItem('aura_token')).toBe('session-token')
    expect(localStorage.getItem('aura_token')).toBeNull()
    expect(store.persistent).toBe(false)
  })

  it('stores token in local storage when remember is enabled', () => {
    const store = useAuthStore()
    store.setAuth('persistent-token', { role: 'admin' }, true)

    expect(localStorage.getItem('aura_token')).toBe('persistent-token')
    expect(localStorage.getItem('aura_persist')).toBe('1')
    expect(sessionStorage.getItem('aura_token')).toBeNull()
    expect(store.persistent).toBe(true)
  })

  it('login uses api response and persists auth state', async () => {
    api.post.mockResolvedValueOnce({
      data: {
        token: 'jwt-token',
        user: { email: 'admin@server.com', role: 'admin' },
      },
    })

    const store = useAuthStore()
    const ok = await store.login('admin@server.com', 'secret', true)

    expect(ok).toBe(true)
    expect(api.post).toHaveBeenCalledWith('/auth/login', {
      email: 'admin@server.com',
      password: 'secret',
    })
    expect(store.isAuthenticated).toBe(true)
    expect(localStorage.getItem('aura_token')).toBe('jwt-token')
  })
})
