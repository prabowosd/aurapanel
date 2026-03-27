<template>
  <div class="space-y-5">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-2xl font-bold text-white">{{ t('website_manage.title') }}</h1>
        <p class="mt-1 text-sm text-gray-400">{{ domain }}</p>
      </div>
      <div class="flex gap-2">
        <button class="btn-secondary" @click="goBack">{{ t('website_manage.back') }}</button>
        <button class="btn-secondary" @click="refreshAll">{{ t('website_manage.refresh') }}</button>
      </div>
    </div>

    <div v-if="error" class="aura-card border-red-500/30 bg-red-500/5 text-red-400">{{ error }}</div>

    <div class="aura-card space-y-4">
      <div class="flex flex-wrap items-center gap-2">
        <span class="text-sm text-gray-400">{{ t('website_manage.status') }}:</span>
        <span :class="isSuspended ? 'text-yellow-400' : 'text-brand-400'">
          {{ isSuspended ? t('website_manage.suspended') : t('website_manage.active') }}
        </span>
        <span class="text-gray-500">|</span>
        <span class="text-sm text-gray-400">{{ t('website_manage.ssl') }}:</span>
        <span :class="site.ssl ? 'text-brand-400' : 'text-yellow-400'">
          {{ site.ssl ? t('website_manage.ssl_active') : t('website_manage.ssl_missing') }}
        </span>
      </div>
      <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
        <input v-model="form.owner" class="aura-input" :placeholder="t('website_manage.owner')" />
        <select v-model="form.php_version" class="aura-input">
          <option v-for="version in phpVersions" :key="version" :value="version">PHP {{ version }}</option>
        </select>
        <input v-model="form.package" class="aura-input" :placeholder="t('website_manage.package')" />
        <input v-model="form.email" class="aura-input" :placeholder="t('website_manage.admin_email')" />
      </div>
      <div class="flex flex-wrap gap-2">
        <button class="btn-primary" @click="saveWebsite">{{ t('website_manage.save') }}</button>
        <button class="btn-secondary" @click="toggleSuspend">
          {{ isSuspended ? t('website_manage.unsuspend') : t('website_manage.suspend') }}
        </button>
        <button class="btn-secondary" @click="issueSsl">{{ t('website_manage.issue_ssl') }}</button>
      </div>
    </div>

    <div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
      <div class="aura-card space-y-3">
        <h3 class="font-semibold text-white">{{ t('website_manage.alias_title') }}</h3>
        <div class="flex gap-2">
          <input v-model="aliasInput" class="aura-input flex-1" :placeholder="t('website_manage.alias_placeholder')" />
          <button class="btn-primary" @click="addAlias">{{ t('website_manage.add_alias') }}</button>
        </div>
        <div class="max-h-40 space-y-2 overflow-auto">
          <div v-for="alias in aliases" :key="alias.alias" class="flex items-center justify-between rounded-lg border border-panel-border px-3 py-2 text-sm">
            <span class="text-gray-200">{{ alias.alias }}</span>
            <button class="btn-danger px-2 py-1" @click="deleteAlias(alias.alias)">{{ t('common.delete') }}</button>
          </div>
        </div>
      </div>

      <div class="aura-card space-y-3">
        <h3 class="font-semibold text-white">{{ t('website_manage.open_basedir') }}</h3>
        <label class="inline-flex items-center gap-2 text-sm text-gray-300">
          <input v-model="advanced.open_basedir" type="checkbox" class="h-4 w-4" />
          {{ t('website_manage.enabled') }}
        </label>
        <button class="btn-primary" @click="saveOpenBasedir">{{ t('website_manage.save') }}</button>
      </div>

      <div class="aura-card space-y-3 lg:col-span-2">
        <h3 class="font-semibold text-white">{{ t('website_manage.rewrite') }}</h3>
        <textarea v-model="advanced.rewrite_rules" rows="7" class="aura-input w-full font-mono text-xs"></textarea>
        <button class="btn-primary" @click="saveRewrite">{{ t('website_manage.save') }}</button>
      </div>

      <div class="aura-card space-y-3 lg:col-span-2">
        <h3 class="font-semibold text-white">{{ t('website_manage.vhost_config') }}</h3>
        <textarea v-model="advanced.vhost_config" rows="10" class="aura-input w-full font-mono text-xs"></textarea>
        <button class="btn-primary" @click="saveVhost">{{ t('website_manage.save') }}</button>
      </div>

      <div class="aura-card space-y-3 lg:col-span-2">
        <h3 class="font-semibold text-white">{{ t('website_manage.custom_ssl') }}</h3>
        <textarea v-model="customSsl.cert_pem" rows="6" class="aura-input w-full font-mono text-xs" placeholder="-----BEGIN CERTIFICATE-----"></textarea>
        <textarea v-model="customSsl.key_pem" rows="6" class="aura-input w-full font-mono text-xs" placeholder="-----BEGIN PRIVATE KEY-----"></textarea>
        <button class="btn-primary" @click="saveCustomSsl">{{ t('website_manage.save') }}</button>
      </div>
    </div>

    <div class="aura-card space-y-4">
      <div class="flex flex-wrap items-center gap-2">
        <button
          class="btn-secondary"
          :class="insightTab === 'traffic' ? 'border-brand-500 text-brand-300' : ''"
          @click="insightTab = 'traffic'"
        >
          Trafik ve Istatistikler
        </button>
        <button
          class="btn-secondary"
          :class="insightTab === 'logs' ? 'border-brand-500 text-brand-300' : ''"
          @click="insightTab = 'logs'"
        >
          Loglar
        </button>
        <button class="btn-secondary ml-auto" @click="refreshInsights">{{ t('website_manage.refresh') }}</button>
      </div>

      <div v-if="insightTab === 'traffic'" class="space-y-4">
        <div class="flex flex-wrap items-center gap-3">
          <label class="text-xs text-gray-400">Aralik</label>
          <select v-model.number="trafficHours" class="aura-input w-40" @change="loadTraffic">
            <option :value="6">Son 6 saat</option>
            <option :value="24">Son 24 saat</option>
            <option :value="72">Son 3 gun</option>
            <option :value="168">Son 7 gun</option>
          </select>
          <span class="text-xs text-gray-500" v-if="traffic?.source_log">Kaynak: {{ traffic.source_log }}</span>
        </div>

        <div class="grid grid-cols-1 gap-3 md:grid-cols-3">
          <div class="rounded-lg border border-panel-border bg-panel-dark p-3">
            <p class="text-xs text-gray-400">Toplam Hit</p>
            <p class="mt-1 text-lg font-semibold text-white">{{ traffic.totals?.hits || 0 }}</p>
          </div>
          <div class="rounded-lg border border-panel-border bg-panel-dark p-3">
            <p class="text-xs text-gray-400">Tekil Ziyaretci</p>
            <p class="mt-1 text-lg font-semibold text-white">{{ traffic.totals?.visitors || 0 }}</p>
          </div>
          <div class="rounded-lg border border-panel-border bg-panel-dark p-3">
            <p class="text-xs text-gray-400">Bant Genisligi</p>
            <p class="mt-1 text-lg font-semibold text-white">{{ formatBytes(traffic.totals?.bandwidth_bytes || 0) }}</p>
          </div>
        </div>

        <div class="rounded-lg border border-panel-border p-3">
          <p class="text-sm text-white font-semibold mb-2">Saatlik Trafik</p>
          <div v-if="trafficLoading" class="text-xs text-gray-400">Yukleniyor...</div>
          <div v-else-if="!traffic.series?.length" class="text-xs text-gray-500">Secili aralikta trafik kaydi yok.</div>
          <div v-else class="space-y-2">
            <div v-for="item in traffic.series" :key="item.bucket" class="space-y-1">
              <div class="flex justify-between text-[11px] text-gray-400">
                <span>{{ item.bucket }}</span>
                <span>{{ item.hits }} hit</span>
              </div>
              <div class="h-2 rounded bg-panel-dark overflow-hidden">
                <div
                  class="h-full rounded bg-gradient-to-r from-cyan-500 to-brand-500"
                  :style="{ width: `${Math.max(4, Math.round((item.hits / Math.max(1, maxTrafficHit)) * 100))}%` }"
                ></div>
              </div>
            </div>
          </div>
        </div>

        <div class="rounded-lg border border-panel-border p-3">
          <p class="text-sm text-white font-semibold mb-2">Top URL</p>
          <div v-if="!traffic.top_paths?.length" class="text-xs text-gray-500">Veri yok.</div>
          <div v-else class="max-h-56 overflow-auto divide-y divide-panel-border">
            <div v-for="row in traffic.top_paths" :key="row.path" class="py-2 text-xs">
              <p class="font-mono text-gray-200 break-all">{{ row.path }}</p>
              <p class="text-gray-500 mt-1">{{ row.hits }} hit • {{ formatBytes(row.bandwidth_bytes) }}</p>
            </div>
          </div>
        </div>
      </div>

      <div v-else class="space-y-3">
        <div class="flex gap-2">
          <button class="btn-secondary" :class="logKind === 'access' ? 'border-brand-500 text-brand-300' : ''" @click="changeLogKind('access')">
            {{ t('website_manage.logs_access') }}
          </button>
          <button class="btn-secondary" :class="logKind === 'error' ? 'border-brand-500 text-brand-300' : ''" @click="changeLogKind('error')">
            {{ t('website_manage.logs_error') }}
          </button>
        </div>
        <pre class="max-h-[360px] overflow-auto whitespace-pre-wrap rounded-lg border border-panel-border bg-panel-dark p-3 text-xs text-gray-200">{{ logs.join('\n') || t('website_manage.no_logs') }}</pre>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import api from '../services/api'

const { t } = useI18n({ useScope: 'global' })
const route = useRoute()
const router = useRouter()

const domain = computed(() => String(route.params.domain || '').toLowerCase())
const phpVersions = ['8.4', '8.3', '8.2', '8.1', '8.0', '7.4']

const error = ref('')
const site = ref({})
const form = ref({ owner: 'aura', php_version: '8.3', package: 'default', email: '' })
const aliases = ref([])
const aliasInput = ref('')
const advanced = ref({ open_basedir: false, rewrite_rules: '', vhost_config: '' })
const customSsl = ref({ cert_pem: '', key_pem: '' })
const logKind = ref('access')
const logs = ref([])
const insightTab = ref('traffic')
const trafficHours = ref(24)
const trafficLoading = ref(false)
const traffic = ref({
  totals: { hits: 0, visitors: 0, bandwidth_bytes: 0 },
  series: [],
  top_paths: [],
  source_log: '',
})
const maxTrafficHit = computed(() => Math.max(1, ...(traffic.value.series || []).map(item => Number(item.hits || 0))))

const isSuspended = computed(() => String(site.value?.status || 'active').toLowerCase() === 'suspended')

function msg(err, fallbackKey) {
  return err?.response?.data?.message || err?.message || t(fallbackKey)
}

async function loadSite() {
  const res = await api.get('/vhost/list', { params: { search: domain.value, page: 1, per_page: 100 } })
  const data = res.data?.data || []
  site.value = data.find(item => String(item.domain || '').toLowerCase() === domain.value) || {}
  form.value = {
    owner: site.value.owner || site.value.user || 'aura',
    php_version: site.value.php_version || site.value.php || '8.3',
    package: site.value.package || 'default',
    email: site.value.email || `webmaster@${domain.value}`,
  }
}

async function loadAliases() {
  const res = await api.get('/websites/aliases', { params: { domain: domain.value } })
  aliases.value = res.data?.data || []
}

async function loadAdvanced() {
  const [cfgRes, sslRes] = await Promise.all([
    api.get('/websites/advanced-config', { params: { domain: domain.value } }),
    api.get('/websites/custom-ssl', { params: { domain: domain.value } }),
  ])
  const cfg = cfgRes.data?.data || {}
  advanced.value = {
    open_basedir: !!cfg.open_basedir,
    rewrite_rules: cfg.rewrite_rules || '',
    vhost_config: cfg.vhost_config || '',
  }
  const ssl = sslRes.data?.data || {}
  customSsl.value = {
    cert_pem: ssl.cert_pem || '',
    key_pem: ssl.key_pem || '',
  }
}

async function loadLogs() {
  const res = await api.get('/monitor/logs/site', { params: { domain: domain.value, kind: logKind.value, lines: 200 } })
  logs.value = res.data?.data || []
}

function formatBytes(bytes) {
  const value = Number(bytes || 0)
  if (value <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const index = Math.min(units.length - 1, Math.floor(Math.log(value) / Math.log(1024)))
  const size = value / Math.pow(1024, index)
  return `${size.toFixed(size >= 10 ? 1 : 2)} ${units[index]}`
}

async function loadTraffic() {
  trafficLoading.value = true
  try {
    const res = await api.get('/analytics/website-traffic', {
      params: { domain: domain.value, hours: trafficHours.value },
    })
    traffic.value = res.data?.data || {
      totals: { hits: 0, visitors: 0, bandwidth_bytes: 0 },
      series: [],
      top_paths: [],
      source_log: '',
    }
  } catch (err) {
    traffic.value = {
      totals: { hits: 0, visitors: 0, bandwidth_bytes: 0 },
      series: [],
      top_paths: [],
      source_log: '',
    }
    error.value = msg(err, 'website_manage.messages.load_failed')
  } finally {
    trafficLoading.value = false
  }
}

async function refreshInsights() {
  if (insightTab.value === 'logs') {
    await loadLogs()
  } else {
    await loadTraffic()
  }
}

async function refreshAll() {
  error.value = ''
  try {
    await Promise.all([loadSite(), loadAliases(), loadAdvanced(), loadLogs(), loadTraffic()])
  } catch (err) {
    error.value = msg(err, 'website_manage.messages.load_failed')
  }
}

async function saveWebsite() {
  error.value = ''
  try {
    await api.post('/vhost/update', {
      domain: domain.value,
      owner: form.value.owner,
      php_version: form.value.php_version,
      package: form.value.package,
      email: form.value.email,
    })
    await loadSite()
  } catch (err) {
    error.value = msg(err, 'website_manage.messages.save_failed')
  }
}

async function toggleSuspend() {
  error.value = ''
  try {
    if (isSuspended.value) {
      await api.post('/vhost/unsuspend', { domain: domain.value })
    } else {
      await api.post('/vhost/suspend', { domain: domain.value })
    }
    await loadSite()
  } catch (err) {
    error.value = msg(err, 'website_manage.messages.status_failed')
  }
}

async function issueSsl() {
  error.value = ''
  try {
    await api.post('/ssl/issue', {
      domain: domain.value,
      email: form.value.email || `admin@${domain.value}`,
      provider: 'letsencrypt',
    })
    await loadSite()
  } catch (err) {
    error.value = msg(err, 'website_manage.messages.ssl_failed')
  }
}

async function addAlias() {
  if (!aliasInput.value) return
  error.value = ''
  try {
    await api.post('/websites/aliases', { domain: domain.value, alias: aliasInput.value })
    aliasInput.value = ''
    await loadAliases()
  } catch (err) {
    error.value = msg(err, 'website_manage.messages.alias_add_failed')
  }
}

async function deleteAlias(alias) {
  error.value = ''
  try {
    await api.delete('/websites/aliases', { params: { domain: domain.value, alias } })
    await loadAliases()
  } catch (err) {
    error.value = msg(err, 'website_manage.messages.alias_delete_failed')
  }
}

async function saveOpenBasedir() {
  error.value = ''
  try {
    await api.post('/websites/open-basedir', { domain: domain.value, enabled: !!advanced.value.open_basedir })
    await loadAdvanced()
  } catch (err) {
    error.value = msg(err, 'website_manage.messages.open_basedir_failed')
  }
}

async function saveRewrite() {
  error.value = ''
  try {
    await api.post('/websites/rewrite', { domain: domain.value, rules: advanced.value.rewrite_rules || '' })
    await loadAdvanced()
  } catch (err) {
    error.value = msg(err, 'website_manage.messages.rewrite_failed')
  }
}

async function saveVhost() {
  error.value = ''
  try {
    await api.post('/websites/vhost-config', { domain: domain.value, content: advanced.value.vhost_config || '' })
    await loadAdvanced()
  } catch (err) {
    error.value = msg(err, 'website_manage.messages.vhost_failed')
  }
}

async function saveCustomSsl() {
  error.value = ''
  try {
    await api.post('/websites/custom-ssl', {
      domain: domain.value,
      cert_pem: customSsl.value.cert_pem || '',
      key_pem: customSsl.value.key_pem || '',
    })
    await loadAdvanced()
  } catch (err) {
    error.value = msg(err, 'website_manage.messages.custom_ssl_failed')
  }
}

async function changeLogKind(kind) {
  logKind.value = kind
  await loadLogs()
}

function goBack() {
  router.push('/websites')
}

onMounted(refreshAll)
</script>
