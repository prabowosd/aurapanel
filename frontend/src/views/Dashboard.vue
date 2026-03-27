<template>
  <div class="space-y-6">
    <div v-if="error" class="aura-card border-red-500/30 bg-red-500/5 text-red-300">
      {{ error }}
    </div>

    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
      <div v-for="stat in stats" :key="stat.name" class="aura-card hover:translate-y-[-2px]">
        <div class="flex items-center justify-between mb-4">
          <div class="text-gray-400 font-medium">{{ stat.name }}</div>
          <component :is="stat.icon" class="w-5 h-5" :class="stat.iconColor" />
        </div>
        <div class="text-3xl font-bold text-white">{{ stat.value }}</div>
        <div class="mt-2 text-sm" :class="stat.trend > 0 ? 'text-brand-400' : 'text-gray-500'">
          {{ t('dashboard.trend_week', { sign: stat.trend > 0 ? '+' : '', trend: stat.trend }) }}
        </div>
      </div>
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
      <div class="lg:col-span-2 aura-card min-h-[400px]">
        <div class="flex items-center justify-between mb-6">
          <h2 class="text-lg font-semibold text-white">{{ t('dashboard.system_load_map') }}</h2>
          <button class="btn-secondary text-sm px-3 py-1.5" @click="loadDashboard" :disabled="loading">
            {{ loading ? t('dashboard.refreshing') : t('dashboard.refresh') }}
          </button>
        </div>
        <div class="flex items-center justify-center h-[300px] border-2 border-dashed border-panel-border rounded-xl">
          <div class="text-center">
            <Activity class="w-12 h-12 text-brand-500 mx-auto mb-3 opacity-50" :class="loading ? 'animate-pulse' : ''" />
            <p class="text-gray-400 font-medium">{{ t('dashboard.uptime', { value: uptimeHuman }) }}</p>
            <p class="text-sm text-gray-500 mt-1">{{ t('dashboard.load_avg', { value: loadAvg }) }}</p>
          </div>
        </div>
      </div>

      <div class="aura-card">
        <h2 class="text-lg font-semibold text-white mb-6">{{ t('dashboard.sre_log') }}</h2>
        <div class="space-y-4" v-if="logs.length">
          <div v-for="log in logs" :key="log.id" class="flex gap-4">
            <div class="mt-1 relative flex items-center justify-center">
              <div class="w-2 h-2 rounded-full ring-4 ring-panel-darker" :class="log.color"></div>
              <div class="absolute top-3 bottom-[-16px] w-[1px] bg-panel-border last:hidden"></div>
            </div>
            <div>
              <p class="text-sm font-medium text-gray-200">{{ log.title }}</p>
              <p class="text-xs text-gray-500 mt-0.5">{{ log.time }}</p>
            </div>
          </div>
        </div>
        <p v-else class="text-sm text-gray-500">{{ t('dashboard.empty_logs') }}</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Server, Globe, Database, ShieldCheck, Activity } from 'lucide-vue-next'
import api from '../services/api'

const { t } = useI18n({ useScope: 'global' })

const loading = ref(false)
const error = ref('')
const uptimeHuman = ref(t('dashboard.na'))
const loadAvg = ref(t('dashboard.na'))

const stats = ref([
  { name: t('dashboard.stats.active_websites'), value: '0', icon: Globe, iconColor: 'text-blue-400', trend: 0 },
  { name: t('dashboard.stats.server_uptime'), value: t('dashboard.na'), icon: Server, iconColor: 'text-brand-400', trend: 0 },
  { name: t('dashboard.stats.databases'), value: '0', icon: Database, iconColor: 'text-orange-400', trend: 0 },
  { name: t('dashboard.stats.threats_blocked'), value: '0', icon: ShieldCheck, iconColor: 'text-red-400', trend: 0 },
])

const logs = ref([])

function summarizeTime(value) {
  if (!value) return t('dashboard.na')
  const text = String(value)
  return text.length > 28 ? `${text.slice(0, 28)}...` : text
}

async function loadDashboard() {
  loading.value = true
  error.value = ''

  try {
    const [
      vhostsRes,
      mariaRes,
      pgRes,
      ebpfRes,
      metricsRes,
      servicesRes,
    ] = await Promise.all([
      api.get('/vhost/list'),
      api.get('/db/mariadb/list'),
      api.get('/db/postgres/list'),
      api.get('/security/ebpf/events'),
      api.get('/status/metrics'),
      api.get('/status/services'),
    ])

    const websites = Array.isArray(vhostsRes.data?.data) ? vhostsRes.data.data : []
    const mariaDbs = Array.isArray(mariaRes.data?.data) ? mariaRes.data.data : []
    const pgDbs = Array.isArray(pgRes.data?.data) ? pgRes.data.data : []
    const ebpfEvents = Array.isArray(ebpfRes.data?.data) ? ebpfRes.data.data : []
    const metrics = metricsRes.data?.data || {}
    const services = Array.isArray(servicesRes.data?.data) ? servicesRes.data.data : []

    uptimeHuman.value = metrics.uptime_human || t('dashboard.na')
    loadAvg.value = metrics.load_avg || t('dashboard.na')

    const runningServices = services.filter(s => String(s.status).toLowerCase() === 'running').length

    stats.value = [
      { name: t('dashboard.stats.active_websites'), value: String(websites.length), icon: Globe, iconColor: 'text-blue-400', trend: 0 },
      { name: t('dashboard.stats.server_uptime'), value: summarizeTime(metrics.uptime_human), icon: Server, iconColor: 'text-brand-400', trend: 0 },
      { name: t('dashboard.stats.databases'), value: String(mariaDbs.length + pgDbs.length), icon: Database, iconColor: 'text-orange-400', trend: 0 },
      { name: t('dashboard.stats.threats_blocked'), value: String(ebpfEvents.length), icon: ShieldCheck, iconColor: 'text-red-400', trend: 0 },
    ]

    const serviceLog = services.slice(0, 2).map((service, index) => ({
      id: `svc-${index}`,
      title: `${service.name}: ${service.status}`,
      time: t('dashboard.service_check'),
      color: String(service.status).toLowerCase() === 'running' ? 'bg-brand-400' : 'bg-yellow-400',
    }))

    const ebpfLog = ebpfEvents.slice(0, 3).map((entry, index) => ({
      id: `evt-${index}`,
      title: String(entry),
      time: t('dashboard.security_event'),
      color: 'bg-red-400',
    }))

    if (!ebpfLog.length && runningServices > 0) {
      serviceLog.unshift({
        id: 'svc-summary',
        title: t('dashboard.running_services', { count: runningServices }),
        time: t('dashboard.runtime_snapshot'),
        color: 'bg-blue-400',
      })
    }

    logs.value = [...ebpfLog, ...serviceLog].slice(0, 5)
  } catch (err) {
    error.value = err?.response?.data?.message || err?.message || t('dashboard.load_failed')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadDashboard()
})
</script>
