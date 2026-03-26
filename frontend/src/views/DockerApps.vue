<template>
  <div class="space-y-6">
    <!-- Header -->
    <div>
      <h1 class="text-2xl font-bold text-white flex items-center gap-3">
        <svg class="w-7 h-7 text-purple-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"/></svg>
        Docker Apps
      </h1>
      <p class="text-gray-400 mt-1">Hazır şablonlardan tek tıkla Docker uygulaması kurun</p>
    </div>

    <!-- Tab Nav -->
    <div class="border-b border-panel-border flex justify-between items-center">
      <nav class="flex gap-6">
        <button @click="tab = 'templates'" :class="['pb-3 text-sm font-medium transition', tab === 'templates' ? 'text-purple-400 border-b-2 border-purple-400' : 'text-gray-400 hover:text-white']">📦 Uygulama Şablonları</button>
        <button @click="tab = 'installed'" :class="['pb-3 text-sm font-medium transition', tab === 'installed' ? 'text-purple-400 border-b-2 border-purple-400' : 'text-gray-400 hover:text-white']">🐳 Kurulu Uygulamalar</button>
        <button @click="tab = 'packages'" :class="['pb-3 text-sm font-medium transition', tab === 'packages' ? 'text-purple-400 border-b-2 border-purple-400' : 'text-gray-400 hover:text-white']">📋 Docker Paketleri</button>
      </nav>
      <button @click="loadData" class="px-3 py-1.5 bg-panel-hover text-gray-300 rounded-lg text-sm hover:bg-gray-600 transition mb-3">
        🔄 Yenile
      </button>
    </div>

    <!-- Templates Grid -->
    <div v-if="tab === 'templates'" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
      <div v-for="t in templates" :key="t.id" class="bg-panel-card border border-panel-border rounded-xl p-5 hover:border-purple-500/50 transition-all duration-200 group flex flex-col justify-between">
        <div>
          <div class="flex items-start justify-between mb-3">
            <span class="text-3xl">{{ t.icon || '📦' }}</span>
            <span class="px-2 py-0.5 bg-purple-500/15 text-purple-400 rounded text-xs font-medium">{{ t.category || 'App' }}</span>
          </div>
          <h3 class="text-white font-semibold text-lg mb-1">{{ t.name }}</h3>
          <p class="text-gray-400 text-sm mb-4 leading-relaxed line-clamp-2" :title="t.description">{{ t.description }}</p>
          <div class="text-xs text-gray-500 font-mono mb-4">{{ t.image }}</div>
        </div>
        <button @click="openInstallModal(t)" class="w-full py-2 bg-purple-600/20 text-purple-400 rounded-lg text-sm font-medium hover:bg-purple-600 hover:text-white transition-all duration-200 mt-2">
          🚀 Kur
        </button>
      </div>
      <div v-if="templates.length === 0" class="col-span-full text-center py-8 text-gray-500">
        Şablon bulunamadı.
      </div>
    </div>

    <!-- Installed Apps List -->
    <div v-if="tab === 'installed'" class="bg-panel-card border border-panel-border rounded-xl overflow-hidden">
      <div class="p-4 border-b border-panel-border flex justify-between items-center">
        <h2 class="text-lg font-semibold text-white">Kurulu Docker Uygulamaları</h2>
      </div>
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="text-gray-400 border-b border-panel-border">
              <th class="text-left px-4 py-3 font-medium">Uygulama</th>
              <th class="text-left px-4 py-3 font-medium">İmaj</th>
              <th class="text-left px-4 py-3 font-medium">Durum</th>
              <th class="text-left px-4 py-3 font-medium">Portlar</th>
              <th class="text-left px-4 py-3 font-medium">Bağlı Paket</th>
              <th class="text-right px-4 py-3 font-medium">İşlem</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="installedApps.length === 0">
              <td colspan="6" class="px-4 py-4 text-center text-gray-500">
                Kurulu uygulama bulunamadı
              </td>
            </tr>
            <tr v-for="app in installedApps" :key="app.name" class="border-b border-panel-border/50 hover:bg-panel-hover/30 transition">
              <td class="px-4 py-3 text-white font-medium">{{ app.name }}</td>
              <td class="px-4 py-3 text-gray-400 font-mono text-xs">{{ app.image }}</td>
              <td class="px-4 py-3">
                <span :class="['px-2 py-1 rounded-full text-xs font-medium', app.status.includes('Up') ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400']">
                  {{ app.status.includes('Up') ? '● Çalışıyor' : '○ Durdu' }}
                </span>
              </td>
              <td class="px-4 py-3 text-gray-400 text-xs font-mono">{{ app.ports || '—' }}</td>
              <td class="px-4 py-3"><span class="px-2 py-0.5 bg-blue-500/15 text-blue-400 rounded text-xs">{{ app.package || 'Limit Yok' }}</span></td>
              <td class="px-4 py-3 text-right space-x-1">
                <button @click="removeApp(app.name)" class="px-2 py-1 bg-red-600/20 text-red-400 rounded text-xs hover:bg-red-600/40 transition">🗑 Kaldır</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Docker Packages -->
    <div v-if="tab === 'packages'" class="grid grid-cols-1 md:grid-cols-3 gap-5">
      <div v-if="packages.length === 0" class="col-span-full text-center py-8 text-gray-500">
        Kayıtlı Docker paketi bulunmamaktadır. Limitleri yönetmek için önce paket ekleyin.
      </div>
      <div v-for="pkg in packages" :key="pkg.id" class="bg-panel-card border border-panel-border rounded-xl p-6 text-center hover:border-purple-500/40 transition">
        <div class="text-4xl mb-3">{{ pkg.name.toLowerCase().includes('start') ? '🌱' : pkg.name.toLowerCase().includes('pro') ? '⚡' : '🏢' }}</div>
        <h3 class="text-xl font-bold text-white mb-2">{{ pkg.name }}</h3>
        <div class="space-y-2 text-sm text-gray-400 mb-5">
          <div>RAM Limiti: <span class="text-white font-medium">{{ pkg.memory_limit }}</span></div>
          <div>CPU Limiti: <span class="text-white font-medium">{{ pkg.cpu_limit }}</span></div>
          <div>Max Konteyner: <span class="text-white font-medium">{{ pkg.max_containers || 'Limitsiz' }}</span></div>
        </div>
      </div>
    </div>

    <!-- Install Modal -->
    <div v-if="showModal" class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50" @click.self="showModal = false">
      <div class="bg-panel-card border border-panel-border rounded-2xl w-full max-w-lg p-6 shadow-2xl">
        <h3 class="text-xl font-bold text-white mb-1">{{ selectedTemplate?.icon }} {{ selectedTemplate?.name }} Kur</h3>
        <p class="text-sm text-gray-400 mb-5">{{ selectedTemplate?.description }}</p>
        <div class="space-y-4">
          <div>
            <label class="block text-sm text-gray-400 mb-1">Uygulama Adı</label>
            <input v-model="installForm.app_name" type="text" :placeholder="`my-${selectedTemplate?.id}`" class="w-full px-4 py-2.5 bg-panel-hover border border-panel-border rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-purple-500">
          </div>
          <div>
            <label class="block text-sm text-gray-400 mb-1">Kaynak Paketi (Limitler)</label>
            <select v-model="installForm.package_id" class="w-full px-4 py-2.5 bg-panel-hover border border-panel-border rounded-lg text-white focus:outline-none focus:border-purple-500">
              <option value="">Limitsiz</option>
              <option v-for="p in packages" :key="p.id" :value="p.id">{{ p.name }} ({{ p.memory_limit }} RAM, {{ p.cpu_limit }} CPU)</option>
            </select>
          </div>
          <div>
            <label class="block text-sm text-gray-400 mb-1">Ek Ortam Değişkenleri</label>
            <input v-model="installForm.custom_env_str" type="text" placeholder="KEY=VALUE, KEY2=VALUE2" class="w-full px-4 py-2.5 bg-panel-hover border border-panel-border rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-purple-500">
            <span class="text-xs text-gray-500 inline-block mt-1">Varsayılan şablon değişkenlerinin üzerine yazar veya ekler. Virgülle ayırın.</span>
          </div>
        </div>
        <div class="flex gap-3 mt-6">
          <button @click="installApp" class="flex-1 py-2.5 bg-gradient-to-r from-purple-600 to-indigo-600 text-white rounded-lg font-medium hover:from-purple-700 hover:to-indigo-700 transition flex items-center justify-center">
            <span v-if="installing" class="animate-spin w-5 h-5 border-2 border-white/30 border-t-white rounded-full mr-2"></span>
            {{ installing ? 'Kuruluyor...' : '🚀 Kur & Başlat' }}
          </button>
          <button @click="showModal = false" class="px-5 py-2.5 bg-panel-hover text-gray-300 rounded-lg hover:bg-gray-600 transition" :disabled="installing">İptal</button>
        </div>
      </div>
    </div>

    <!-- Notification -->
    <div v-if="notification" :class="['fixed bottom-6 right-6 px-5 py-3 rounded-xl shadow-2xl text-sm font-medium transition-all duration-300 z-50', notification.type === 'success' ? 'bg-green-600 text-white' : 'bg-red-600 text-white']">
      {{ notification.message }}
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import api from '../services/api'

const route = useRoute()
const tab = ref(route.meta.dockerAppsTab || 'templates')
const showModal = ref(false)
const selectedTemplate = ref(null)
const notification = ref(null)
const installing = ref(false)

const templates = ref([])
const installedApps = ref([])
const packages = ref([])

const installForm = ref({ app_name: '', package_id: '', custom_env_str: '' })

const showNotif = (message, type = 'success') => {
  notification.value = { message, type }
  setTimeout(() => notification.value = null, 3000)
}

const openInstallModal = (template) => {
  selectedTemplate.value = template
  installForm.value = { app_name: `my-${template.id}`, package_id: '', custom_env_str: '' }
  showModal.value = true
}

const loadData = async () => {
  try {
    const res = await api.get('/docker/apps/templates');
    templates.value = res.data?.data || [];
  } catch (e) {
    showNotif(e.response?.data?.error || 'Şablonlar yüklenemedi', 'error');
  }

  try {
    const res = await api.get('/docker/apps/installed');
    installedApps.value = res.data?.data || [];
  } catch (e) {
    // console.error(e)
  }

  try {
    const res = await api.get('/docker/packages');
    packages.value = res.data?.data || [];
  } catch (e) {
    // console.error(e)
  }
}

const installApp = async () => {
  if (installing.value) return;
  installing.value = true;
  
  try {
    await api.post('/docker/apps/install', {
      template_id: selectedTemplate.value.id,
      app_name: installForm.value.app_name || `my-${selectedTemplate.value.id}`,
      package_id: installForm.value.package_id || null,
      custom_env: installForm.value.custom_env_str ? installForm.value.custom_env_str.split(',').map(s => s.trim()) : [],
    })
    showNotif(`${selectedTemplate.value.name} başarıyla kuruldu!`)
    showModal.value = false
    tab.value = 'installed'
    loadData()
  } catch (e) {
    showNotif(e.response?.data?.error || 'Kurulum başarısız oldu', 'error')
  } finally {
    installing.value = false;
  }
}

const removeApp = async (appName) => {
  if (!confirm(`"${appName}" uygulamasını tamamen silmek istediğinize emin misiniz? Bu işlem geri alınamaz.`)) return;
  
  try {
    await api.post('/docker/apps/remove', { app_name: appName })
    showNotif(`Uygulama başarıyla kaldırıldı: ${appName}`)
    loadData()
  } catch (e) {
    showNotif(e.response?.data?.error || 'Uygulama kaldırılamadı', 'error')
  }
}

onMounted(loadData)
</script>
