import { defineStore } from 'pinia'
import api from '../services/api'
import { useNotificationStore } from './notifications'

const TOKEN_KEY = 'aura_token'
const USER_KEY = 'aura_user'
const PERSIST_KEY = 'aura_persist'

function decodeJwtPayload(token) {
  try {
    const parts = String(token || '').split('.')
    if (parts.length < 2) return null
    const base64 = parts[1].replace(/-/g, '+').replace(/_/g, '/')
    const padded = base64 + '='.repeat((4 - (base64.length % 4)) % 4)
    return JSON.parse(atob(padded))
  } catch {
    return null
  }
}

function isTokenExpired(token) {
  const payload = decodeJwtPayload(token)
  if (!payload?.exp) return false
  const exp = Number(payload.exp)
  if (!Number.isFinite(exp)) return false
  return exp * 1000 <= Date.now()
}

function clearStoredAuth() {
  localStorage.removeItem(TOKEN_KEY)
  localStorage.removeItem(USER_KEY)
  localStorage.removeItem(PERSIST_KEY)
  sessionStorage.removeItem(TOKEN_KEY)
  sessionStorage.removeItem(USER_KEY)
}

function getInitialAuth() {
  const localToken = localStorage.getItem(TOKEN_KEY)
  const sessionToken = sessionStorage.getItem(TOKEN_KEY)
  const token = localToken || sessionToken || null
  const userRaw = localStorage.getItem(USER_KEY) || sessionStorage.getItem(USER_KEY)
  const user = userRaw ? JSON.parse(userRaw) : null
  const persistent = localStorage.getItem(PERSIST_KEY) === '1'

  if (token && isTokenExpired(token)) {
    clearStoredAuth()
    return { token: null, user: null, persistent: false }
  }

  return { token, user, persistent }
}

const initialAuth = getInitialAuth()

export const useAuthStore = defineStore('auth', {
  state: () => ({
    token: initialAuth.token,
    user: initialAuth.user,
    persistent: initialAuth.persistent,
  }),
  getters: {
    isAuthenticated: (state) => !!state.token,
    isAdmin: (state) => state.user?.role === 'admin'
  },
  actions: {
    isTokenExpired(token = this.token) {
      if (!token) return true
      return isTokenExpired(token)
    },
    ensureValidSession() {
      if (this.token && this.isTokenExpired(this.token)) {
        this.logout()
        return false
      }
      return !!this.token
    },
    async login(email, password, remember = false) {
      try {
        const response = await api.post('/auth/login', { email, password })
        this.setAuth(response.data.token, response.data.user, remember)
        const notificationStore = useNotificationStore()
        notificationStore.add({
          title: 'Welcome',
          message: `Signed in as ${response.data.user?.email || email}`,
          type: 'success',
          source: 'auth',
        })
        return true
      } catch (error) {
        console.error('Login Error', error)
        throw new Error(error.response?.data?.message || 'Giris basarisiz. Bilgilerinizi kontrol edin.')
      }
    },
    setAuth(token, user, persistent = false) {
      this.token = token
      this.user = user
      this.persistent = !!persistent

      clearStoredAuth()
      const target = this.persistent ? localStorage : sessionStorage
      target.setItem(TOKEN_KEY, token)
      target.setItem(USER_KEY, JSON.stringify(user))
      if (this.persistent) {
        localStorage.setItem(PERSIST_KEY, '1')
      }
    },
    logout() {
      const hadSession = !!this.token
      this.token = null
      this.user = null
      this.persistent = false
      clearStoredAuth()
      if (hadSession) {
        const notificationStore = useNotificationStore()
        notificationStore.add({
          title: 'Signed Out',
          message: 'Your session was closed securely.',
          type: 'info',
          source: 'auth',
        })
      }
    }
  }
})
