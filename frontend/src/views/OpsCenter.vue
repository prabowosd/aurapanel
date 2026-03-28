<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-white">Ops Center</h1>
        <p class="mt-1 text-gray-400">SRE, GitOps, Redis isolation and health checks.</p>
      </div>
    </div>

    <div v-if="notif.message" :class="['rounded-xl border px-4 py-3 text-sm', notif.type === 'error' ? 'border-red-500/40 bg-red-500/10 text-red-300' : 'border-green-500/30 bg-green-500/10 text-green-300']">
      {{ notif.message }}
    </div>

    <section class="grid grid-cols-1 gap-4 lg:grid-cols-2">
      <div class="rounded-xl border border-panel-border bg-panel-card p-5">
        <div class="mb-3 flex items-center justify-between">
          <h2 class="text-lg font-semibold text-white">Health</h2>
          <button class="rounded bg-panel-hover px-3 py-1.5 text-xs text-gray-300 transition hover:bg-gray-600" @click="loadHealth">
            Refresh
          </button>
        </div>
        <div class="space-y-2 text-sm text-gray-300">
          <p>Status: <span class="font-semibold text-white">{{ health.status || '-' }}</span></p>
          <p>Version: <span class="font-semibold text-white">{{ health.version || '-' }}</span></p>
          <p>Uptime: <span class="font-semibold text-white">{{ health.uptime ?? '-' }}</span></p>
        </div>
      </div>

      <div class="rounded-xl border border-panel-border bg-panel-card p-5">
        <div class="mb-3 flex items-center justify-between">
          <h2 class="text-lg font-semibold text-white">SRE Prediction</h2>
          <button class="rounded bg-panel-hover px-3 py-1.5 text-xs text-gray-300 transition hover:bg-gray-600" @click="loadPrediction">
            Analyze
          </button>
        </div>
        <p class="text-sm text-gray-300">{{ prediction || 'No prediction loaded yet.' }}</p>
      </div>
    </section>

    <section class="rounded-xl border border-panel-border bg-panel-card p-5">
      <h2 class="mb-3 text-lg font-semibold text-white">SRE Log Query</h2>
      <div class="flex flex-col gap-3 md:flex-row">
        <input
          v-model="sreQuery"
          type="text"
          class="aura-input flex-1"
          placeholder="e.g. 404 errors in nginx logs"
        />
        <button class="rounded bg-gradient-to-r from-orange-600 to-amber-600 px-4 py-2 text-sm font-medium text-white transition hover:from-orange-700 hover:to-amber-700" @click="runSreQuery">
          Run Query
        </button>
      </div>
      <div class="mt-4 space-y-2 text-sm text-gray-300">
        <p><span class="text-gray-400">Answer:</span> {{ sreAnswer.answer || '-' }}</p>
        <p><span class="text-gray-400">Confidence:</span> {{ sreAnswer.confidence || '-' }}</p>
        <p><span class="text-gray-400">Sources:</span> {{ (sreAnswer.matched_sources || []).join(', ') || '-' }}</p>
      </div>
    </section>

    <section class="rounded-xl border border-panel-border bg-panel-card p-5">
      <div class="mb-3 flex items-center justify-between">
        <h2 class="text-lg font-semibold text-white">SRE Optimization Suggestions</h2>
        <button class="rounded bg-panel-hover px-3 py-1.5 text-xs text-gray-300 transition hover:bg-gray-600" @click="loadOptimizations">
          Generate
        </button>
      </div>
      <ul class="space-y-2 text-sm text-gray-300">
        <li v-for="(item, idx) in optimizations" :key="idx" class="rounded-lg bg-panel-darker px-3 py-2">
          {{ item }}
        </li>
        <li v-if="optimizations.length === 0" class="text-gray-500">No suggestions loaded yet.</li>
      </ul>
    </section>

    <section class="grid grid-cols-1 gap-4 xl:grid-cols-2">
      <div class="rounded-xl border border-panel-border bg-panel-card p-5">
        <h2 class="mb-3 text-lg font-semibold text-white">GitOps Deploy</h2>
        <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
          <input v-model="gitops.domain" type="text" class="aura-input" placeholder="Domain (example.com)" />
          <input v-model="gitops.branch" type="text" class="aura-input" placeholder="Branch (main)" />
          <input v-model="gitops.repo_url" type="text" class="aura-input md:col-span-2" placeholder="Repository URL" />
          <input v-model="gitops.deploy_path" type="text" class="aura-input md:col-span-2" placeholder="/home/user/public_html/example.com" />
          <input v-model="gitops.webhook_secret" type="text" class="aura-input md:col-span-2" placeholder="Webhook secret" />
        </div>
        <button class="mt-4 rounded bg-gradient-to-r from-orange-600 to-amber-600 px-4 py-2 text-sm font-medium text-white transition hover:from-orange-700 hover:to-amber-700" @click="deployGitops">
          Deploy
        </button>
      </div>

      <div class="rounded-xl border border-panel-border bg-panel-card p-5">
        <h2 class="mb-3 text-lg font-semibold text-white">Redis Isolation</h2>
        <p class="mb-3 text-sm text-gray-400">
          Host Redis isolation provisions a dedicated systemd service per domain. Docker Redis remains a separate container runtime choice.
        </p>
        <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
          <input v-model="redis.domain" type="text" class="aura-input md:col-span-2" placeholder="Domain (example.com)" />
          <input v-model.number="redis.max_memory_mb" type="number" min="64" max="65536" class="aura-input" placeholder="Max memory (MB)" />
        </div>
        <button class="mt-4 rounded bg-gradient-to-r from-blue-600 to-indigo-600 px-4 py-2 text-sm font-medium text-white transition hover:from-blue-700 hover:to-indigo-700" @click="createRedis">
          Create Isolated Redis
        </button>
        <div class="mt-5 space-y-3">
          <div class="flex items-center justify-between">
            <h3 class="text-sm font-semibold text-white">Host Redis Instances</h3>
            <button class="rounded bg-panel-hover px-3 py-1.5 text-xs text-gray-300 transition hover:bg-gray-600" @click="loadRedisIsolations">
              Refresh
            </button>
          </div>
          <div v-if="redisIsolations.length === 0" class="rounded-lg bg-panel-darker px-3 py-3 text-sm text-gray-500">
            No host Redis isolations created yet.
          </div>
          <div v-for="item in redisIsolations" :key="item.domain" class="rounded-lg border border-panel-border/60 bg-panel-darker px-4 py-3">
            <div class="flex items-start justify-between gap-3">
              <div>
                <p class="font-medium text-white">{{ item.domain }}</p>
                <p class="text-xs text-gray-400">{{ item.runtime }}/{{ item.provisioning }} · {{ item.bind_address }}:{{ item.port }}</p>
              </div>
              <div class="text-right text-xs text-gray-400">
                <p>{{ item.unit }}</p>
                <p>{{ item.max_memory_mb }} MB</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import api from '../services/api'

const health = reactive({ status: '', version: '', uptime: 0 })
const prediction = ref('')
const sreQuery = ref('')
const sreAnswer = reactive({ answer: '', confidence: '', matched_sources: [] })
const optimizations = ref([])
const notif = reactive({ message: '', type: 'success' })
const redisIsolations = ref([])

const gitops = reactive({
  domain: '',
  repo_url: '',
  branch: 'main',
  deploy_path: '',
  webhook_secret: '',
})

const redis = reactive({
  domain: '',
  max_memory_mb: 512,
})

const notify = (message, type = 'success') => {
  notif.message = message
  notif.type = type
}

const loadHealth = async () => {
  try {
    const { data } = await api.get('/health')
    const payload = data.data || data
    health.status = payload.status || data.status || ''
    health.version = payload.version || payload.architecture || ''
    health.uptime = payload.uptime ?? 0
  } catch (err) {
    notify(err.response?.data?.message || err.message || 'Health check failed.', 'error')
  }
}

const loadPrediction = async () => {
  try {
    const { data } = await api.get('/monitor/sre')
    const payload = data.data || data
    prediction.value = payload.prediction || data.message || 'No prediction returned.'
  } catch (err) {
    notify(err.response?.data?.message || err.message || 'SRE prediction failed.', 'error')
  }
}

const runSreQuery = async () => {
  if (!sreQuery.value.trim()) {
    notify('Please enter a query.', 'error')
    return
  }
  try {
    const { data } = await api.post('/monitor/sre/log-query', { query: sreQuery.value.trim() })
    const payload = data.data || data
    sreAnswer.answer = payload.answer || ''
    sreAnswer.confidence = payload.confidence ?? ''
    sreAnswer.matched_sources = payload.matched_sources || []
  } catch (err) {
    notify(err.response?.data?.message || err.message || 'SRE log query failed.', 'error')
  }
}

const loadOptimizations = async () => {
  try {
    const { data } = await api.get('/monitor/sre/optimize')
    const payload = data.data || data
    optimizations.value = payload.actions || []
  } catch (err) {
    notify(err.response?.data?.message || err.message || 'SRE optimization failed.', 'error')
  }
}

const deployGitops = async () => {
  if (!gitops.domain || !gitops.repo_url || !gitops.branch || !gitops.deploy_path || !gitops.webhook_secret) {
    notify('All GitOps fields are required.', 'error')
    return
  }
  try {
    const { data } = await api.post('/gitops/deploy', { ...gitops })
    notify(data.message || 'GitOps deployment started.')
  } catch (err) {
    notify(err.response?.data?.message || err.message || 'GitOps deploy failed.', 'error')
  }
}

const createRedis = async () => {
  if (!redis.domain) {
    notify('Domain is required for Redis setup.', 'error')
    return
  }
  try {
    const { data } = await api.post('/perf/redis', {
      domain: redis.domain,
      max_memory_mb: Number(redis.max_memory_mb || 512),
    })
    notify(data.message || 'Host Redis instance created.')
    await loadRedisIsolations()
  } catch (err) {
    notify(err.response?.data?.message || err.message || 'Redis setup failed.', 'error')
  }
}

const loadRedisIsolations = async () => {
  try {
    const { data } = await api.get('/perf/redis')
    redisIsolations.value = data.data || []
  } catch (err) {
    notify(err.response?.data?.message || err.message || 'Failed to load Redis isolations.', 'error')
  }
}

onMounted(async () => {
  await loadHealth()
  await loadPrediction()
  await loadRedisIsolations()
})
</script>
