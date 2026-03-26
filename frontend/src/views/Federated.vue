<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-white">Federated Nodes</h1>
        <p class="text-gray-400 mt-1">WireGuard uzerinden coklu sunucu baglantisi</p>
        <p class="text-xs text-gray-500 mt-1" v-if="mode">
          Mod: <strong>{{ mode.mode }}</strong> / Rol: <strong>{{ mode.primary ? 'primary' : 'passive' }}</strong>
        </p>
      </div>
      <button class="btn-secondary" @click="loadNodes">Yenile</button>
    </div>

    <div class="aura-card space-y-3">
      <h2 class="text-lg font-bold text-white">Node Ekle</h2>
      <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
        <input v-model="form.node_name" class="aura-input" placeholder="node-eu-1" />
        <input v-model="form.ip_address" class="aura-input" placeholder="10.0.0.2/32" />
        <input v-model="form.pub_key" class="aura-input" placeholder="wireguard public key" />
      </div>
      <button class="btn-primary" :disabled="mode && !mode.primary" @click="addNode">Node Ekle</button>
      <p v-if="mode && !mode.primary" class="text-xs text-amber-400">
        Bu node passive modda; active-passive politikasi geregi yeni peer ekleme kapali.
      </p>
    </div>

    <div class="aura-card">
      <h2 class="text-lg font-bold text-white mb-3">Bagli Node'lar</h2>
      <div class="space-y-2">
        <div v-for="node in nodes" :key="node.node_name" class="bg-panel-dark border border-panel-border rounded-lg p-3">
          <p class="text-white">{{ node.node_name }}</p>
          <p class="text-xs text-gray-400">{{ node.ip_address }}</p>
        </div>
        <div v-if="nodes.length === 0" class="text-gray-400 text-sm">Node yok</div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '../services/api'

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
