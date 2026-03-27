import axios from 'axios'
import { useAuthStore } from '../stores/auth'
import { useNotificationStore } from '../stores/notifications'

const defaultBaseUrl = typeof window !== 'undefined'
  ? '/api/v1'
  : 'http://127.0.0.1:8090/api/v1'

const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || defaultBaseUrl,
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
    Accept: 'application/json',
  },
})

const silentErrorHeader = 'X-Aura-Silent-Error'

function extractErrorMessage(error) {
  const responseData = error?.response?.data
  if (typeof responseData?.message === 'string' && responseData.message.trim()) {
    return responseData.message.trim()
  }
  if (typeof responseData?.error === 'string' && responseData.error.trim()) {
    return responseData.error.trim()
  }
  if (typeof error?.message === 'string' && error.message.trim()) {
    return error.message.trim()
  }
  return 'Unknown API error'
}

api.interceptors.request.use(config => {
  const authStore = useAuthStore()
  if (authStore.token && authStore.isTokenExpired(authStore.token)) {
    authStore.logout()
    window.location.href = '/login'
    return Promise.reject(new Error('Session expired'))
  }
  if (authStore.token) {
    config.headers.Authorization = `Bearer ${authStore.token}`
  }
  return config
}, error => Promise.reject(error))

api.interceptors.response.use(response => response, error => {
  const authStore = useAuthStore()
  const notificationStore = useNotificationStore()
  const status = Number(error?.response?.status || 0)
  const reqUrl = String(error?.config?.url || '')
  const silentError = String(error?.config?.headers?.[silentErrorHeader] || '').toLowerCase() === '1'
  const isLoginRequest = reqUrl.includes('/auth/login')

  if (status === 401 && !isLoginRequest) {
    notificationStore.add({
      title: 'Session Ended',
      message: 'Your session expired. Please sign in again.',
      type: 'warning',
      source: 'auth',
    })
    authStore.logout()
    window.location.href = '/login'
    return Promise.reject(error)
  }

  if (!silentError) {
    const statusText = status > 0 ? `HTTP ${status}` : 'Network Error'
    notificationStore.add({
      title: `API Error - ${statusText}`,
      message: reqUrl ? `${reqUrl}: ${extractErrorMessage(error)}` : extractErrorMessage(error),
      type: 'error',
      source: 'api',
    })
  }

  return Promise.reject(error)
})

export default api
