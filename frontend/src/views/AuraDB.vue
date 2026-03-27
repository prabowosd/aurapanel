<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-white">{{ t('auradb.title') }}</h1>
        <p class="text-gray-400 mt-1">{{ t('auradb.subtitle') }}</p>
      </div>
      <button class="btn-secondary" @click="listTables">{{ t('auradb.refresh_tables') }}</button>
    </div>

    <div v-if="bridgeMode" class="aura-card border-brand-500/30 bg-brand-500/10">
      <h2 class="text-lg font-semibold text-white">{{ t('auradb.bridge_active') }}</h2>
      <p class="text-sm text-gray-300 mt-2">
        {{ t('auradb.bridge_domain') }}: <span class="font-mono text-white">{{ bridgeProfile?.domain }}</span>
        | {{ t('auradb.bridge_engine') }}: <span class="font-mono text-white">{{ bridgeProfile?.engine }}</span>
        | {{ t('auradb.bridge_db') }}: <span class="font-mono text-white">{{ bridgeProfile?.db_name }}</span>
        | {{ t('auradb.bridge_user') }}: <span class="font-mono text-white">{{ bridgeProfile?.db_user }}</span>
      </p>
      <p v-if="bridgeError" class="text-sm text-red-400 mt-2">{{ bridgeError }}</p>
    </div>

    <div class="aura-card space-y-3">
      <h2 class="text-lg font-bold text-white">{{ t('auradb.connection') }}</h2>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
        <select v-model="dbType" class="aura-input" :disabled="bridgeMode">
          <option value="mariadb">MariaDB</option>
          <option value="postgresql">PostgreSQL</option>
        </select>
        <input
          v-model="connectionString"
          class="aura-input"
          :disabled="bridgeMode"
          :placeholder="bridgeMode ? t('auradb.bridge_placeholder') : t('auradb.connection_placeholder')"
        />
      </div>
    </div>

    <div class="aura-card space-y-3">
      <h2 class="text-lg font-bold text-white">{{ t('auradb.query') }}</h2>
      <textarea v-model="query" class="aura-input min-h-28" :placeholder="t('auradb.query_placeholder')"></textarea>
      <button class="btn-primary" @click="runQuery">{{ t('auradb.run_query') }}</button>
      <pre class="bg-panel-dark border border-panel-border rounded-lg p-3 text-xs text-gray-300 overflow-auto max-h-72">{{ queryResult }}</pre>
    </div>

    <div class="aura-card">
      <h2 class="text-lg font-bold text-white mb-3">{{ t('auradb.tables') }}</h2>
      <div class="flex flex-wrap gap-2">
        <span v-for="tName in tables" :key="tName" class="px-3 py-1 rounded-full bg-brand-500/10 border border-brand-500/20 text-brand-300 text-sm">
          {{ tName }}
        </span>
        <span v-if="tables.length === 0" class="text-gray-400 text-sm">{{ t('auradb.empty_tables') }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import api from '../services/api'

const { t } = useI18n({ useScope: 'global' })

const route = useRoute()
const dbType = ref('mariadb')
const connectionString = ref('localhost')
const query = ref('SELECT 1;')
const queryResult = ref('')
const tables = ref([])
const bridgeToken = ref('')
const bridgeMode = ref(false)
const bridgeProfile = ref(null)
const bridgeError = ref('')

async function runQuery() {
  const payload = bridgeMode.value
    ? {
      bridge_token: bridgeToken.value,
      query: query.value
    }
    : {
      db_type: dbType.value,
      connection_string: connectionString.value,
      query: query.value
    }

  const res = await api.post('/db/explorer/query', payload)
  queryResult.value = typeof res.data.data === 'string' ? res.data.data : JSON.stringify(res.data.data, null, 2)
}

async function listTables() {
  const payload = bridgeMode.value
    ? {
      bridge_token: bridgeToken.value
    }
    : {
      db_type: dbType.value,
      connection_string: connectionString.value
    }

  const res = await api.post('/db/explorer/tables', payload)
  tables.value = res.data.data || []
}

async function loadBridge() {
  const token = String(route.query.bridge || '').trim()
  bridgeToken.value = token
  bridgeError.value = ''

  if (!token) {
    bridgeMode.value = false
    bridgeProfile.value = null
    return
  }

  try {
    const res = await api.get('/db/explorer/bridge/resolve', { params: { token } })
    const profile = res.data?.data || null
    if (!profile) {
      bridgeMode.value = false
      bridgeProfile.value = null
      bridgeError.value = t('auradb.bridge_profile_failed')
      return
    }

    bridgeMode.value = true
    bridgeProfile.value = profile
    dbType.value = profile.engine
    connectionString.value = `bridge://${profile.domain}/${profile.db_name}@${profile.db_user}`
    await listTables()
  } catch (error) {
    bridgeMode.value = false
    bridgeProfile.value = null
    bridgeError.value = error?.response?.data?.message || error?.message || t('auradb.bridge_validation_failed')
  }
}

onMounted(async () => {
  await loadBridge()
})

watch(() => route.query.bridge, async () => {
  await loadBridge()
})
</script>
