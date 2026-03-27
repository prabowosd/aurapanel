<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-white">{{ t('log_viewer.title') }}</h1>
        <p class="text-gray-400 mt-1">{{ t('log_viewer.subtitle') }}</p>
      </div>
      <button class="btn-secondary" @click="loadLogs">{{ t('log_viewer.refresh') }}</button>
    </div>

    <div class="aura-card space-y-3">
      <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
        <input v-model="domain" class="aura-input" :placeholder="t('log_viewer.domain_placeholder')" />
        <input v-model.number="lines" type="number" class="aura-input" :placeholder="t('log_viewer.lines_placeholder')" />
        <button class="btn-primary" @click="loadLogs">{{ t('log_viewer.fetch') }}</button>
      </div>
    </div>

    <div class="aura-card">
      <h2 class="text-lg font-bold text-white mb-3">{{ t('log_viewer.output') }}</h2>
      <pre class="bg-panel-dark border border-panel-border rounded-lg p-4 text-xs text-gray-300 overflow-auto max-h-[520px]">{{ logs.join('\n') }}</pre>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '../services/api'

const { t } = useI18n({ useScope: 'global' })

const domain = ref('example.com')
const lines = ref(50)
const logs = ref([])

async function loadLogs() {
  const res = await api.get('/monitor/logs/site', { params: { domain: domain.value, lines: lines.value } })
  logs.value = res.data.data || []
}
</script>
