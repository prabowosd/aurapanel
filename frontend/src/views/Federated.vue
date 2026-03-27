<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-white">{{ t('federated.title') }}</h1>
        <p class="text-gray-400 mt-1">{{ t('federated.subtitle') }}</p>
        <p class="text-xs text-gray-500 mt-1" v-if="mode">
          {{ t('federated.mode_status', { mode: mode.mode, role: mode.primary ? t('federated.roles.primary') : t('federated.roles.passive') }) }}
        </p>
      </div>
      <button class="btn-secondary" @click="loadNodes">{{ t('federated.refresh') }}</button>
    </div>

    <div class="aura-card space-y-3">
      <h2 class="text-lg font-bold text-white">{{ t('federated.add_node') }}</h2>
      <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
        <input v-model="form.node_name" class="aura-input" :placeholder="t('federated.node_name_placeholder')" />
        <input v-model="form.ip_address" class="aura-input" :placeholder="t('federated.ip_placeholder')" />
        <input v-model="form.pub_key" class="aura-input" :placeholder="t('federated.pubkey_placeholder')" />
      </div>
      <button class="btn-primary" :disabled="mode && !mode.primary" @click="addNode">{{ t('federated.add_node') }}</button>
      <p v-if="mode && !mode.primary" class="text-xs text-amber-400">
        {{ t('federated.passive_notice') }}
      </p>
    </div>

    <div class="aura-card">
      <h2 class="text-lg font-bold text-white mb-3">{{ t('federated.connected_nodes') }}</h2>
      <div class="space-y-2">
        <div v-for="node in nodes" :key="node.node_name" class="bg-panel-dark border border-panel-border rounded-lg p-3">
          <p class="text-white">{{ node.node_name }}</p>
          <p class="text-xs text-gray-400">{{ node.ip_address }}</p>
        </div>
        <div v-if="nodes.length === 0" class="text-gray-400 text-sm">{{ t('federated.empty') }}</div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '../services/api'

const { t } = useI18n({ useScope: 'global' })

const nodes = ref([])
const mode = ref(null)
const form = ref({
  node_name: '',
  ip_address: '',
  pub_key: ''
})

async function loadNodes() {
  const [nodesRes, modeRes] = await Promise.all([
    api.get('/federated/nodes'),
    api.get('/federated/mode')
  ])
  nodes.value = nodesRes.data.data || []
  mode.value = modeRes.data.data || null
}

async function addNode() {
  await api.post('/federated/join', form.value)
  form.value.node_name = ''
  form.value.ip_address = ''
  form.value.pub_key = ''
  await loadNodes()
}

onMounted(loadNodes)
</script>
