<template>
  <div class="space-y-6 max-w-4xl">
    <div>
      <h1 class="text-2xl font-bold text-white flex items-center gap-2">
        <Settings2 class="w-6 h-6 text-indigo-400" />
        {{ t('panel_port.title') }}
      </h1>
      <p class="text-gray-400 mt-1">
        {{ t('panel_port.subtitle') }}
      </p>
    </div>

    <div class="bg-amber-500/10 border border-amber-500/30 rounded-xl p-4">
      <p class="text-amber-300 font-semibold text-sm">{{ t('panel_port.notice_title') }}</p>
      <p class="text-amber-100/90 text-sm mt-1">
        {{ t('panel_port.notice_body') }}
      </p>
    </div>

    <div class="bg-blue-500/10 border border-blue-500/30 rounded-xl p-4 text-sm">
      <p class="text-blue-300 font-semibold">{{ t('panel_port.current_config') }}</p>
      <p class="text-blue-100 mt-1">
        {{ t('panel_port.current_port') }}: <span class="font-bold">{{ currentPort }}</span>
        <span class="text-blue-200/80">({{ gatewayAddr }})</span>
      </p>
    </div>

    <div class="bg-panel-card border border-panel-border rounded-xl p-6 space-y-5">
      <div>
        <label class="block text-xs text-gray-400 uppercase tracking-wide mb-2">{{ t('panel_port.new_port') }}</label>
        <input
          v-model.number="port"
          type="number"
          min="1"
          max="65535"
          class="w-full bg-panel-darker border border-panel-border rounded-lg px-4 py-3 text-white focus:outline-none focus:border-indigo-400"
          :placeholder="t('panel_port.port_placeholder')"
        />
        <p class="text-xs text-gray-500 mt-2">{{ t('panel_port.valid_range') }}</p>
      </div>

      <label class="flex items-center gap-3 text-sm text-gray-300">
        <input v-model="openFirewall" type="checkbox" class="accent-indigo-500" />
        {{ t('panel_port.open_firewall') }}
      </label>

      <div class="pt-2">
        <button
          @click="changePort"
          :disabled="loading || saving"
          class="px-5 py-2.5 rounded-lg bg-indigo-600 hover:bg-indigo-500 disabled:opacity-50 text-white font-medium transition inline-flex items-center gap-2"
        >
          <Loader2 v-if="saving" class="w-4 h-4 animate-spin" />
          <Save v-else class="w-4 h-4" />
          <span>{{ saving ? t('panel_port.changing') : t('panel_port.change') }}</span>
        </button>
      </div>
    </div>

    <div v-if="message" class="bg-emerald-500/10 border border-emerald-500/30 rounded-xl p-4 text-sm text-emerald-200">
      <p class="font-semibold">{{ message }}</p>
      <p class="mt-1">{{ t('panel_port.reconnect_url') }}: <span class="font-mono">{{ newAccessUrl }}</span></p>
    </div>

    <div v-if="warnings.length" class="bg-yellow-500/10 border border-yellow-500/30 rounded-xl p-4 text-sm text-yellow-200">
      <p class="font-semibold mb-2">{{ t('panel_port.warnings') }}</p>
      <ul class="space-y-1">
        <li v-for="item in warnings" :key="item">- {{ item }}</li>
      </ul>
    </div>

    <div v-if="firewallActions.length" class="bg-panel-card border border-panel-border rounded-xl p-4 text-sm text-gray-300">
      <p class="font-semibold text-white mb-2">{{ t('panel_port.firewall_actions') }}</p>
      <ul class="space-y-1">
        <li v-for="item in firewallActions" :key="item">- {{ item }}</li>
      </ul>
    </div>

    <div v-if="error" class="bg-red-500/10 border border-red-500/30 rounded-xl p-4 text-sm text-red-200">
      {{ error }}
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { Loader2, Save, Settings2 } from 'lucide-vue-next'
import api from '../services/api'

const { t } = useI18n({ useScope: 'global' })

const loading = ref(false)
const saving = ref(false)
const port = ref(8090)
const currentPort = ref(8090)
const gatewayAddr = ref(':8090')
const openFirewall = ref(true)
const message = ref('')
const error = ref('')
const warnings = ref([])
const firewallActions = ref([])

const newAccessUrl = computed(() => {
  if (typeof window === 'undefined') {
    return `http://YOUR_SERVER_IP:${port.value}`
  }
  return `${window.location.protocol}//${window.location.hostname}:${port.value}`
})

const loadPanelPort = async () => {
  loading.value = true
  error.value = ''
  try {
    const response = await api.get('/status/panel-port')
    if (response.data?.status !== 'success') {
      throw new Error(response.data?.message || t('panel_port.messages.load_failed'))
    }

    const payload = response.data?.data || {}
    const fetchedPort = Number(payload.current_port || 8090)
    currentPort.value = fetchedPort
    port.value = fetchedPort
    gatewayAddr.value = payload.gateway_addr || `:${fetchedPort}`
  } catch (err) {
    error.value = err?.message || t('panel_port.messages.load_failed')
  } finally {
    loading.value = false
  }
}

const changePort = async () => {
  error.value = ''
  message.value = ''
  warnings.value = []
  firewallActions.value = []

  const targetPort = Number(port.value)
  if (!Number.isInteger(targetPort) || targetPort < 1 || targetPort > 65535) {
    error.value = t('panel_port.messages.invalid_port')
    return
  }

  saving.value = true
  try {
    const response = await api.post('/status/panel-port', {
      port: targetPort,
      open_firewall: openFirewall.value
    })

    if (response.data?.status !== 'success') {
      throw new Error(response.data?.message || t('panel_port.messages.update_failed'))
    }

    const payload = response.data?.data || {}
    currentPort.value = targetPort
    gatewayAddr.value = payload.gateway_addr || `:${targetPort}`
    firewallActions.value = Array.isArray(payload.firewall_actions) ? payload.firewall_actions : []
    warnings.value = Array.isArray(payload.warnings) ? payload.warnings : []

    const restartNote = payload.restart_scheduled
      ? ` ${t('panel_port.messages.restart_scheduled')}`
      : ''
    message.value = t('panel_port.messages.updated', { port: targetPort, restartNote })
  } catch (err) {
    const apiMessage = err?.response?.data?.message
    if (!apiMessage && err?.message && err.message.toLowerCase().includes('network')) {
      message.value = t('panel_port.messages.network_reconnect', { url: newAccessUrl.value })
      return
    }
    error.value = apiMessage || err?.message || t('panel_port.messages.update_failed')
  } finally {
    saving.value = false
  }
}

onMounted(loadPanelPort)
</script>
