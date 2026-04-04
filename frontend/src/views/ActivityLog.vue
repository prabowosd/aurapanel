<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between gap-4">
      <div>
        <h1 class="text-2xl font-bold text-white flex items-center gap-3">
          <BookOpen class="w-7 h-7 text-indigo-400" />
          {{ t('activity_log.title') }}
        </h1>
        <p class="text-gray-400 mt-1">{{ t('activity_log.subtitle') }}</p>
      </div>
      <button
        @click="loadLogs"
        class="px-4 py-2 bg-panel-hover text-gray-300 rounded-lg text-sm hover:bg-gray-600 transition flex items-center gap-2"
      >
        <RefreshCw class="w-4 h-4" :class="{ 'animate-spin': loading }" />
        {{ t('common.refresh') }}
      </button>
    </div>

    <!-- Filters -->
    <div class="aura-card">
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div>
          <label class="block text-sm text-gray-400 mb-1">{{ t('activity_log.filter_user') }}</label>
          <select v-model="filterUser" class="aura-input" @change="applyFilters">
            <option value="">{{ t('activity_log.filter_all') }}</option>
            <option v-for="user in uniqueUsers" :key="user" :value="user">{{ user }}</option>
          </select>
        </div>
        <div>
          <label class="block text-sm text-gray-400 mb-1">{{ t('activity_log.filter_date_from') }}</label>
          <input v-model="filterDateFrom" type="datetime-local" class="aura-input" @change="applyFilters" />
        </div>
        <div>
          <label class="block text-sm text-gray-400 mb-1">{{ t('activity_log.filter_date_to') }}</label>
          <input v-model="filterDateTo" type="datetime-local" class="aura-input" @change="applyFilters" />
        </div>
      </div>
    </div>

    <!-- Table -->
    <div class="bg-panel-card border border-panel-border rounded-xl overflow-hidden">
      <div class="p-4 border-b border-panel-border flex items-center justify-between">
        <h2 class="text-lg font-semibold text-white">
          {{ t('activity_log.title') }}
          <span class="ml-2 text-sm font-normal text-gray-400">{{ t('activity_log.records_count', { count: filteredLogs.length }) }}</span>
        </h2>
        <div class="flex items-center gap-2 text-xs text-gray-500">
          <span class="w-2 h-2 rounded-full bg-green-400 inline-block animate-pulse"></span>
          {{ t('activity_log.auto_refresh', { seconds: 30 }) }}
        </div>
      </div>

      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="text-gray-400 border-b border-panel-border">
              <th class="text-left px-4 py-3 font-medium">{{ t('activity_log.timestamp') }}</th>
              <th class="text-left px-4 py-3 font-medium">{{ t('activity_log.user') }}</th>
              <th class="text-left px-4 py-3 font-medium">{{ t('activity_log.action') }}</th>
              <th class="text-left px-4 py-3 font-medium">{{ t('activity_log.detail') }}</th>
              <th class="text-left px-4 py-3 font-medium">{{ t('activity_log.ip') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="entry in paginatedLogs"
              :key="entry.id || `${entry.timestamp}-${entry.user}-${entry.action}`"
              class="border-b border-panel-border/50 hover:bg-white/[0.02] transition"
            >
              <td class="px-4 py-3 text-gray-300 whitespace-nowrap font-mono text-xs">
                {{ formatDate(entry.timestamp) }}
              </td>
              <td class="px-4 py-3">
                <span class="px-2 py-0.5 rounded bg-indigo-500/15 text-indigo-300 text-xs font-medium">
                  {{ entry.user || '-' }}
                </span>
              </td>
              <td class="px-4 py-3">
                <span :class="['px-2 py-0.5 rounded text-xs font-medium', actionBadgeClass(entry.action)]">
                  {{ entry.action || '-' }}
                </span>
              </td>
              <td class="px-4 py-3 text-gray-300 max-w-xs truncate" :title="entry.detail">
                {{ entry.detail || '-' }}
              </td>
              <td class="px-4 py-3 text-gray-400 font-mono text-xs">
                {{ entry.ip || '-' }}
              </td>
            </tr>
            <tr v-if="paginatedLogs.length === 0">
              <td colspan="5" class="px-4 py-14 text-center text-gray-500">
                <BookOpen class="w-8 h-8 mx-auto mb-2 opacity-30" />
                {{ t('activity_log.no_data') }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Pagination -->
      <div v-if="totalPages > 1" class="p-4 border-t border-panel-border flex items-center justify-between">
        <p class="text-sm text-gray-400">
          {{ t('activity_log.pagination_summary', { page: currentPage, totalPages, count: filteredLogs.length }) }}
        </p>
        <div class="flex gap-2">
          <button
            @click="currentPage--"
            :disabled="currentPage === 1"
            class="px-3 py-1.5 bg-panel-hover text-gray-300 rounded-lg text-sm hover:bg-gray-600 transition disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {{ t('activity_log.previous') }}
          </button>
          <button
            @click="currentPage++"
            :disabled="currentPage === totalPages"
            class="px-3 py-1.5 bg-panel-hover text-gray-300 rounded-lg text-sm hover:bg-gray-600 transition disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {{ t('activity_log.next') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { BookOpen, RefreshCw } from 'lucide-vue-next'
import api from '../services/api'

const { t, locale } = useI18n({ useScope: 'global' })

const logs = ref([])
const loading = ref(false)
const filterUser = ref('')
const filterDateFrom = ref('')
const filterDateTo = ref('')
const currentPage = ref(1)
const perPage = 50

let autoRefreshTimer = null

const uniqueUsers = computed(() => {
  const users = logs.value.map(e => e.user).filter(Boolean)
  return [...new Set(users)].sort()
})

const filteredLogs = computed(() => {
  let result = logs.value

  if (filterUser.value) {
    result = result.filter(e => e.user === filterUser.value)
  }

  if (filterDateFrom.value) {
    const from = new Date(filterDateFrom.value).getTime()
    result = result.filter(e => new Date(e.timestamp).getTime() >= from)
  }

  if (filterDateTo.value) {
    const to = new Date(filterDateTo.value).getTime()
    result = result.filter(e => new Date(e.timestamp).getTime() <= to)
  }

  return result
})

const totalPages = computed(() => Math.max(1, Math.ceil(filteredLogs.value.length / perPage)))

const paginatedLogs = computed(() => {
  const start = (currentPage.value - 1) * perPage
  return filteredLogs.value.slice(start, start + perPage)
})

function applyFilters() {
  currentPage.value = 1
}

function formatDate(timestamp) {
  if (!timestamp) return '-'
  try {
    return new Date(timestamp).toLocaleString(locale.value || 'en-US', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    })
  } catch {
    return String(timestamp)
  }
}

function actionBadgeClass(action) {
  if (!action) return 'bg-gray-500/15 text-gray-400'
  const a = action.toLowerCase()
  if (a.includes('delete') || a.includes('remove') || a.includes('drop')) return 'bg-red-500/15 text-red-400'
  if (a.includes('create') || a.includes('add') || a.includes('install')) return 'bg-green-500/15 text-green-400'
  if (a.includes('update') || a.includes('edit') || a.includes('change')) return 'bg-yellow-500/15 text-yellow-400'
  if (a.includes('login') || a.includes('auth')) return 'bg-blue-500/15 text-blue-400'
  return 'bg-gray-500/15 text-gray-400'
}

async function loadLogs() {
  loading.value = true
  try {
    const res = await api.get('/activity/log')
    logs.value = res.data?.data || []
  } catch {
    logs.value = []
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadLogs()
  autoRefreshTimer = setInterval(loadLogs, 30000)
})

onBeforeUnmount(() => {
  if (autoRefreshTimer) clearInterval(autoRefreshTimer)
})
</script>
