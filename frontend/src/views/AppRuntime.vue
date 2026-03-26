<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-white">App Runtime</h1>
        <p class="text-gray-400 mt-1">Node.js ve Python uygulama yonetimi</p>
      </div>
      <button class="btn-secondary" @click="loadApps">Yenile</button>
    </div>

    <div class="aura-card space-y-3">
      <h2 class="text-lg font-bold text-white">Node.js</h2>
      <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
        <input v-model="node.dir" class="aura-input" placeholder="Proje dizini" />
        <input v-model="node.app_name" class="aura-input" placeholder="App name" />
        <input v-model="node.start_script" class="aura-input" placeholder="npm start" />
      </div>
      <div class="flex gap-2">
        <button class="btn-secondary" @click="nodeInstallDeps">Install Deps</button>
        <button class="btn-primary" @click="nodeStart">Start</button>
        <button class="btn-danger" @click="nodeStop">Stop</button>
      </div>
    </div>

    <div class="aura-card space-y-3">
      <h2 class="text-lg font-bold text-white">Python</h2>
      <div class="grid grid-cols-1 md:grid-cols-4 gap-3">
        <input v-model="python.dir" class="aura-input" placeholder="Proje dizini" />
        <input v-model="python.app_name" class="aura-input" placeholder="App name" />
        <input v-model="python.wsgi_module" class="aura-input" placeholder="app:app" />
        <input v-model.number="python.port" type="number" class="aura-input" placeholder="8001" />
      </div>
      <div class="flex gap-2">
        <button class="btn-secondary" @click="pythonCreateVenv">Create venv</button>
        <button class="btn-secondary" @click="pythonInstallReq">Install req</button>
        <button class="btn-primary" @click="pythonStart">Start</button>
      </div>
    </div>

    <div class="aura-card">
      <h2 class="text-lg font-bold text-white mb-3">Calisan Uygulamalar</h2>
      <div class="space-y-2">
        <div v-for="app in apps" :key="app.app_name" class="bg-panel-dark border border-panel-border rounded-lg p-3 flex justify-between">
          <div>
            <p class="text-white">{{ app.app_name }}</p>
            <p class="text-xs text-gray-400">{{ app.runtime }} · {{ app.dir }}</p>
          </div>
          <span :class="app.status === 'running' ? 'text-green-400' : 'text-yellow-400'">{{ app.status }}</span>
        </div>
        <div v-if="apps.length === 0" class="text-gray-400 text-sm">Kayit yok</div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import api from '../services/api'

const apps = ref([])
const node = ref({ dir: '', app_name: '', start_script: 'npm start' })
const python = ref({ dir: '', app_name: '', wsgi_module: 'app:app', port: 8001 })

async function loadApps() {
  const res = await api.get('/apps/runtime/list')
  apps.value = res.data.data || []
}

async function nodeInstallDeps() {
  await api.post('/apps/runtime/node/install-deps', { dir: node.value.dir })
}
async function nodeStart() {
  await api.post('/apps/runtime/node/start', node.value)
  await loadApps()
}
async function nodeStop() {
  await api.post('/apps/runtime/node/stop', { app_name: node.value.app_name })
  await loadApps()
}

async function pythonCreateVenv() {
  await api.post('/apps/runtime/python/venv', { dir: python.value.dir })
}
async function pythonInstallReq() {
  await api.post('/apps/runtime/python/install', { dir: python.value.dir })
}
async function pythonStart() {
  await api.post('/apps/runtime/python/start', python.value)
  await loadApps()
}

onMounted(loadApps)
</script>
