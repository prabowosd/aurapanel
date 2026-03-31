<template>
  <div class="space-y-6 max-w-5xl">
    <div class="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
      <div>
        <h1 class="text-2xl font-bold text-white">{{ t('panel_update.title') }}</h1>
        <p class="text-gray-400 mt-1">{{ t('panel_update.subtitle') }}</p>
      </div>
      <div class="flex gap-2">
        <button class="btn-secondary" :disabled="loading" @click="loadUpdateStatus">
          {{ loading ? t('panel_update.checking') : t('panel_update.check_now') }}
        </button>
        <button
          class="btn-primary"
          :disabled="loading || updating"
          @click="applyUpdate"
        >
          {{ updating ? t('panel_update.updating') : t('panel_update.update_now') }}
        </button>
      </div>
    </div>

    <div class="aura-card border border-panel-border/80 bg-panel-card/95">
      <div class="flex flex-col gap-4 xl:flex-row xl:items-start xl:justify-between">
        <div>
          <p class="text-xs font-semibold uppercase tracking-[0.2em] text-gray-500">{{ t('panel_update.channel') }}</p>
          <h2 class="text-xl font-bold text-white mt-2">{{ updateStatus.current_version || '-' }}</h2>
          <p class="mt-3 text-sm text-gray-400">
            {{ updateStatus.update_available ? t('panel_update.update_available') : t('panel_update.up_to_date') }}
          </p>
        </div>

        <div
          class="inline-flex items-center rounded-full px-4 py-2 text-sm font-semibold"
          :class="updateStatus.update_available ? 'bg-amber-500/15 text-amber-300 border border-amber-500/30' : 'bg-brand-500/10 text-brand-300 border border-brand-500/20'"
        >
          {{ updateStatus.update_available ? t('panel_update.badges.update_available') : t('panel_update.badges.up_to_date') }}
        </div>
      </div>

      <div class="mt-5 grid grid-cols-1 gap-4 md:grid-cols-3">
        <div class="rounded-xl border border-panel-border/70 bg-panel-darker/60 p-4">
          <p class="text-xs uppercase tracking-[0.18em] text-gray-500">{{ t('panel_update.current_version') }}</p>
          <p class="mt-2 text-lg font-semibold text-white">{{ updateStatus.current_version || '-' }}</p>
        </div>
        <div class="rounded-xl border border-panel-border/70 bg-panel-darker/60 p-4">
          <p class="text-xs uppercase tracking-[0.18em] text-gray-500">{{ t('panel_update.latest_release') }}</p>
          <p class="mt-2 text-lg font-semibold text-white">{{ updateStatus.latest_version || t('panel_update.not_checked') }}</p>
          <p v-if="updateStatus.published_at" class="mt-1 text-xs text-gray-500">{{ formatReleaseDate(updateStatus.published_at) }}</p>
        </div>
        <div class="rounded-xl border border-panel-border/70 bg-panel-darker/60 p-4">
          <p class="text-xs uppercase tracking-[0.18em] text-gray-500">{{ t('panel_update.source') }}</p>
          <p class="mt-2 text-lg font-semibold text-white">{{ updateStatus.source || 'GitHub Releases' }}</p>
          <p v-if="updateStatus.checked_at" class="mt-1 text-xs text-gray-500">{{ t('panel_update.last_checked') }}: {{ formatReleaseDate(updateStatus.checked_at) }}</p>
        </div>
      </div>

      <div v-if="updateStatus.release_notes || updateStatus.error || updateStatus.release_url" class="mt-4 rounded-xl border border-panel-border/60 bg-black/10 p-4">
        <p v-if="updateStatus.release_notes" class="text-sm text-gray-300">{{ updateStatus.release_notes }}</p>
        <p v-if="updateStatus.error" class="text-sm text-yellow-300">{{ updateStatus.error }}</p>
        <a
          v-if="updateStatus.release_url"
          :href="updateStatus.release_url"
          target="_blank"
          rel="noreferrer"
          class="mt-3 inline-flex items-center text-sm font-medium text-brand-300 hover:text-brand-200 transition"
        >
          {{ t('panel_update.view_release') }}
        </a>
      </div>
    </div>

    <div v-if="message" class="bg-emerald-500/10 border border-emerald-500/30 rounded-xl p-4 text-sm text-emerald-200">
      {{ message }}
    </div>

    <div v-if="warnings.length" class="bg-yellow-500/10 border border-yellow-500/30 rounded-xl p-4 text-sm text-yellow-200">
      <p class="font-semibold mb-2">{{ t('panel_update.warnings') }}</p>
      <ul class="space-y-1">
        <li v-for="item in warnings" :key="item">- {{ item }}</li>
      </ul>
    </div>

    <div v-if="steps.length" class="bg-panel-card border border-panel-border rounded-xl p-4 text-sm text-gray-300">
      <p class="font-semibold text-white mb-2">{{ t('panel_update.steps') }}</p>
      <ul class="space-y-1">
        <li v-for="item in steps" :key="item">- {{ item }}</li>
      </ul>
    </div>

    <div v-if="error" class="bg-red-500/10 border border-red-500/30 rounded-xl p-4 text-sm text-red-200">
      {{ error }}
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '../services/api'

const { t } = useI18n({ useScope: 'global' })

const loading = ref(false)
const updating = ref(false)
const message = ref('')
const error = ref('')
const warnings = ref([])
const steps = ref([])

const updateStatus = ref({
  current_version: 'Aura Panel V1',
  latest_version: '',
  latest_tag: '',
  update_available: false,
  release_name: '',
  release_url: '',
  release_notes: '',
  published_at: '',
  source: 'GitHub Releases',
  checked_at: '',
  error: '',
})

function formatReleaseDate(value) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return date.toLocaleString()
}

async function loadUpdateStatus() {
  loading.value = true
  error.value = ''
  try {
    const res = await api.get('/status/update')
    if (res.data?.status !== 'success') {
      throw new Error(res.data?.message || t('panel_update.messages.load_failed'))
    }
    updateStatus.value = {
      ...updateStatus.value,
      ...(res.data?.data || {}),
    }
  } catch (err) {
    error.value = err?.response?.data?.message || err?.message || t('panel_update.messages.load_failed')
  } finally {
    loading.value = false
  }
}

async function applyUpdate() {
  error.value = ''
  message.value = ''
  warnings.value = []
  steps.value = []
  updating.value = true
  try {
    const res = await api.post('/status/update/apply', {})
    if (res.data?.status !== 'success') {
      throw new Error(res.data?.message || t('panel_update.messages.apply_failed'))
    }
    const payload = res.data?.data || {}
    message.value = res.data?.message || t('panel_update.messages.applied')
    warnings.value = Array.isArray(payload.warnings) ? payload.warnings : []
    steps.value = Array.isArray(payload.steps) ? payload.steps : []
    await loadUpdateStatus()
  } catch (err) {
    error.value = err?.response?.data?.message || err?.message || t('panel_update.messages.apply_failed')
  } finally {
    updating.value = false
  }
}

onMounted(loadUpdateStatus)
</script>
