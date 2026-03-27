import { defineStore } from 'pinia'

const STORAGE_KEY = 'aura_notifications'
const MAX_ITEMS = 100
const DEDUPE_WINDOW_MS = 4000

function safeParse(raw) {
  if (!raw) return []
  try {
    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed
      .filter(item => item && typeof item === 'object' && item.id)
      .map(item => ({
        id: String(item.id),
        title: String(item.title || 'Bildirim'),
        message: String(item.message || ''),
        type: normalizeType(item.type),
        source: String(item.source || 'system'),
        read: !!item.read,
        createdAt: Number(item.createdAt || Date.now()),
      }))
      .sort((a, b) => b.createdAt - a.createdAt)
      .slice(0, MAX_ITEMS)
  } catch {
    return []
  }
}

function normalizeType(type) {
  const t = String(type || '').toLowerCase().trim()
  if (t === 'success' || t === 'warning' || t === 'error' || t === 'info') {
    return t
  }
  return 'info'
}

function persist(items) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(items))
}

function nowId() {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 10)}`
}

export const useNotificationStore = defineStore('notifications', {
  state: () => ({
    items: safeParse(localStorage.getItem(STORAGE_KEY)),
  }),
  getters: {
    unreadCount: (state) => state.items.filter(item => !item.read).length,
    orderedItems: (state) => [...state.items].sort((a, b) => b.createdAt - a.createdAt),
  },
  actions: {
    add(notification) {
      const normalized = {
        id: nowId(),
        title: String(notification?.title || 'Bildirim'),
        message: String(notification?.message || ''),
        type: normalizeType(notification?.type),
        source: String(notification?.source || 'system'),
        read: false,
        createdAt: Date.now(),
      }

      const hasRecentDuplicate = this.items.some(item => {
        const sameContent =
          item.title === normalized.title &&
          item.message === normalized.message &&
          item.type === normalized.type &&
          item.source === normalized.source
        const closeInTime = Math.abs(normalized.createdAt - Number(item.createdAt || 0)) <= DEDUPE_WINDOW_MS
        return sameContent && closeInTime
      })

      if (hasRecentDuplicate) {
        return null
      }

      this.items = [normalized, ...this.items].slice(0, MAX_ITEMS)
      persist(this.items)
      return normalized.id
    },
    markRead(id) {
      this.items = this.items.map(item => (item.id === id ? { ...item, read: true } : item))
      persist(this.items)
    },
    markAllRead() {
      this.items = this.items.map(item => ({ ...item, read: true }))
      persist(this.items)
    },
    remove(id) {
      this.items = this.items.filter(item => item.id !== id)
      persist(this.items)
    },
    clearAll() {
      this.items = []
      persist(this.items)
    },
  },
})
