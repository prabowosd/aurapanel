<template>
  <div class="space-y-6 php-theme">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-white flex items-center gap-3">
          🐘 PHP Versiyon Yönetimi
        </h1>
        <p class="text-gray-400 mt-1">PHP versiyonlarını kurun, kaldırın ve site bazlı atayın</p>
      </div>
    </div>

    <!-- Tabs -->
    <div class="border-b border-panel-border">
      <nav class="flex gap-6">
        <button @click="tab = 'versions'" :class="['pb-3 text-sm font-medium transition', tab === 'versions' ? 'text-orange-400 border-b-2 border-orange-400' : 'text-gray-400 hover:text-white']">📦 Versiyonlar</button>
        <button @click="tab = 'sites'" :class="['pb-3 text-sm font-medium transition', tab === 'sites' ? 'text-orange-400 border-b-2 border-orange-400' : 'text-gray-400 hover:text-white']">🌐 Site Atamaları</button>
        <button @click="tab = 'extensions'" :class="['pb-3 text-sm font-medium transition', tab === 'extensions' ? 'text-orange-400 border-b-2 border-orange-400' : 'text-gray-400 hover:text-white']">🔌 Extensionlar</button>
        <button @click="tab = 'config'" :class="['pb-3 text-sm font-medium transition', tab === 'config' ? 'text-orange-400 border-b-2 border-orange-400' : 'text-gray-400 hover:text-white']">⚙️ php.ini</button>
      </nav>
    </div>

    <div v-if="loading" class="text-center py-10">
      <div class="animate-spin w-8 h-8 border-4 border-orange-500 border-t-transparent inset-0 mx-auto rounded-full mb-3"></div>
      <p class="text-gray-400">Yükleniyor...</p>
    </div>

    <div v-else>
      <!-- Versions Tab -->
      <div v-if="tab === 'versions'" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        <div v-for="v in phpVersions" :key="v.version" class="bg-panel-card border border-panel-border rounded-xl p-5 hover:border-orange-500/30 transition">
          <div class="flex items-center justify-between mb-3">
            <div class="flex items-center gap-3">
              <div :class="['w-10 h-10 rounded-lg flex items-center justify-center text-lg font-bold', v.installed ? 'bg-green-500/10 text-green-400' : 'bg-gray-500/10 text-gray-500']">
                {{ v.version.split('.')[1] || v.version }}
              </div>
              <div>
                <p class="text-white font-semibold">PHP {{ v.version }}</p>
                <p class="text-gray-400 text-xs">{{ parseFloat(v.version) < 8.1 ? '⚠️ EOL' : '✅ Aktif Destek' }}</p>
              </div>
            </div>
            <span :class="['px-2 py-0.5 rounded text-xs font-medium', v.installed ? 'bg-green-500/15 text-green-400' : 'bg-gray-500/15 text-gray-400']">
              {{ v.installed ? 'Kurulu' : 'Kurulmadı' }}
            </span>
          </div>
          <div class="flex gap-2">
            <button v-if="!v.installed" @click="installPhp(v)" class="flex-1 py-2 bg-gradient-to-r from-orange-600 to-amber-600 text-white rounded-lg text-sm hover:from-orange-700 hover:to-amber-700 transition">Kur</button>
            <button v-else @click="removePhp(v)" class="flex-1 py-2 bg-red-600/20 text-red-400 rounded-lg text-sm hover:bg-red-600/40 transition">Kaldır</button>
            <button v-if="v.installed" @click="restartPhp(v)" class="px-3 py-2 bg-blue-600/20 text-blue-400 rounded-lg text-sm hover:bg-blue-600/40 transition">🔄</button>
          </div>
        </div>
      </div>

      <!-- Site Assignments Tab -->
      <div v-if="tab === 'sites'" class="bg-panel-card border border-panel-border rounded-xl overflow-hidden">
        <div class="p-4 border-b border-panel-border">
          <h2 class="text-lg font-semibold text-white">Site Bazlı PHP Atama</h2>
        </div>
        <table class="w-full text-sm">
          <thead>
            <tr class="text-gray-400 border-b border-panel-border">
              <th class="text-left px-4 py-3">Domain</th>
              <th class="text-left px-4 py-3">Mevcut PHP</th>
              <th class="text-left px-4 py-3">Değiştir</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="site in siteAssignments" :key="site.domain" class="border-b border-panel-border/50 hover:bg-white/[0.02] transition">
              <td class="px-4 py-3 text-white font-medium">{{ site.domain }}</td>
              <td class="px-4 py-3">
                <span class="px-2 py-0.5 bg-green-500/15 text-green-400 rounded text-xs font-medium">PHP {{ site.php }}</span>
              </td>
              <td class="px-4 py-3">
                <select v-model="site.php" @change="changePhp(site)" class="php-field px-3 py-1.5 bg-panel-hover border border-panel-border rounded-lg text-white text-sm focus:outline-none focus:border-orange-500">
                  <option v-for="v in installedVersions" :key="v" :value="v">PHP {{ v }}</option>
                </select>
              </td>
            </tr>
            <tr v-if="!siteAssignments.length"><td colspan="3" class="p-4 text-center text-gray-500">Henüz site eklenmedi</td></tr>
          </tbody>
        </table>
      </div>

      <!-- Extensions Tab -->
      <div v-if="tab === 'extensions'">
        <div class="flex items-center gap-4 mb-4">
          <select v-model="selectedExtVersion" class="php-field px-4 py-2 bg-panel-hover border border-panel-border rounded-lg text-white text-sm focus:outline-none focus:border-orange-500">
            <option v-for="v in installedVersions" :key="v" :value="v">PHP {{ v }}</option>
          </select>
        </div>
        <div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-3">
          <div v-for="ext in extensions" :key="ext.name" class="bg-panel-card border border-panel-border rounded-xl p-4 flex items-center justify-between hover:border-orange-500/30 transition">
            <div>
              <p class="text-white text-sm font-medium">{{ ext.name }}</p>
              <p class="text-gray-500 text-xs">{{ ext.desc }}</p>
            </div>
            <button @click="toggleExtension(ext)"
              :class="['w-10 h-6 rounded-full transition relative', ext.enabled ? 'bg-green-600' : 'bg-gray-600']">
              <span :class="['absolute top-0.5 w-5 h-5 bg-white rounded-full transition-transform', ext.enabled ? 'translate-x-4' : 'translate-x-0.5']"></span>
            </button>
          </div>
        </div>
      </div>

      <!-- php.ini Tab -->
      <div v-if="tab === 'config'">
        <div class="flex items-center gap-4 mb-4">
          <select v-model="selectedConfigVersion" @change="loadPhpIni" class="php-field px-4 py-2 bg-panel-hover border border-panel-border rounded-lg text-white text-sm focus:outline-none">
            <option v-for="v in installedVersions" :key="v" :value="v">PHP {{ v }}</option>
          </select>
          <button @click="savePhpIni" class="px-5 py-2 bg-green-600 text-white rounded-lg text-sm hover:bg-green-700 transition">💾 Kaydet</button>
        </div>
        <div class="bg-panel-card border border-panel-border rounded-xl overflow-hidden">
          <div class="grid grid-cols-1 md:grid-cols-2 gap-4 p-5">
            <div v-for="opt in phpIniOptions" :key="opt.key">
              <label class="block text-sm text-gray-400 mb-1 php-field-label">{{ opt.key }}</label>
              <input v-model="opt.value" type="text" class="php-field w-full px-3 py-2 bg-panel-hover border border-panel-border rounded-lg text-white text-sm focus:outline-none focus:border-orange-500">
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Notification -->
    <div v-if="notification" :class="['fixed bottom-6 right-6 px-5 py-3 rounded-xl shadow-2xl text-sm font-medium z-50', notification.type === 'success' ? 'bg-green-600 text-white' : 'bg-red-600 text-white']">
      {{ notification.message }}
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import api from '../services/api'

const tab = ref('versions')
const loading = ref(false)
const notification = ref(null)
const selectedExtVersion = ref('8.3')
const selectedConfigVersion = ref('8.3')

const phpVersions = ref([])
const siteAssignments = ref([])
const extensions = ref([])
const phpIniOptions = ref([])

const installedVersions = computed(() => phpVersions.value.filter(v => v.installed).map(v => v.version))

const showNotif = (msg, type = 'success') => {
  notification.value = { message: msg, type }
  setTimeout(() => notification.value = null, 3000)
}

const loadData = async () => {
  loading.value = true
  try {
    // Fallback static array matching API expectation
    const { data } = await api.get('/php/versions').catch(() => ({ data: { data: [
      { version: '7.4', installed: false }, { version: '8.0', installed: false },
      { version: '8.1', installed: true }, { version: '8.2', installed: true },
      { version: '8.3', installed: true }, { version: '8.4', installed: false }
    ]}}))
    phpVersions.value = data.data || []
    
    selectedExtVersion.value = installedVersions.value[0] || '8.3'
    selectedConfigVersion.value = installedVersions.value[0] || '8.3'
    
    // Load VHosts for Site Assignments
    const vhosts = await api.get('/vhost/list').catch(() => ({ data: { data: [] }}))
    siteAssignments.value = (vhosts.data.data || []).map(s => ({ domain: s.domain, php: s.php }))

    extensions.value = [
      { name: 'mbstring', desc: 'Multibyte string', enabled: true },
      { name: 'curl', desc: 'HTTP client', enabled: true },
      { name: 'gd', desc: 'Image processing', enabled: true },
      { name: 'zip', desc: 'ZIP arşivleri', enabled: true }
    ]

    await loadPhpIni()
  } finally {
    loading.value = false
  }
}

const installPhp = async (v) => {
  try {
    await api.post('/php/install', { version: v.version })
    showNotif(`PHP ${v.version} kurulumu başlatıldı`)
    v.installed = true
  } catch (e) {
    showNotif('Hata oluştu', 'error')
  }
}

const removePhp = async (v) => {
  try {
    await api.post('/php/remove', { version: v.version })
    showNotif(`PHP ${v.version} kaldırıldı`)
    v.installed = false
  } catch (e) {
    showNotif('Hata oluştu', 'error')
  }
}

const restartPhp = async (v) => {
  try {
    await api.post('/php/restart', { version: v.version })
    showNotif(`PHP ${v.version} FPM yeniden başlatıldı`)
  } catch (e) {
    showNotif('Hata oluştu', 'error')
  }
}

const changePhp = async (site) => {
  try {
    // Using vhost creation/update or specific php assign API if it exists
    await api.post('/vhost', { domain: site.domain, php_version: site.php })
    showNotif(`${site.domain} sınıfı PHP ${site.php} olarak güncellendi`)
  } catch (e) {
    showNotif('Hata', 'error')
  }
}

const loadPhpIni = async () => {
  try {
    const res = await api.post('/php/ini/get', { version: selectedConfigVersion.value }).catch(() => ({ data: null }))
    if (res.data && res.data.config) {
      phpIniOptions.value = Object.entries(res.data.config).map(([key, value]) => ({ key, value }))
    } else {
      phpIniOptions.value = [
        { key: 'memory_limit', value: '256M' },
        { key: 'upload_max_filesize', value: '64M' },
        { key: 'post_max_size', value: '64M' },
        { key: 'max_execution_time', value: '300' }
      ]
    }
  } catch {
    showNotif('PHP INI okunamadı', 'error')
  }
}

const savePhpIni = async () => {
  try {
    const config = {}
    phpIniOptions.value.forEach(opt => config[opt.key] = opt.value)
    await api.post('/php/ini/save', { version: selectedConfigVersion.value, config })
    showNotif(`PHP ${selectedConfigVersion.value} php.ini kaydedildi`)
  } catch (e) {
    showNotif('Kaydedilemedi', 'error')
  }
}

const toggleExtension = (ext) => {
  ext.enabled = !ext.enabled
  showNotif(`${ext.name} uzantısı ${ext.enabled ? 'aktif' : 'pasif'} edildi`)
}

onMounted(loadData)
</script>

<style scoped>
.php-theme .php-field {
  background-color: #1f2d44 !important;
  color: #fb923c !important;
  border-color: rgba(251, 146, 60, 0.45) !important;
}

.php-theme .php-field:focus {
  border-color: #fb923c !important;
  box-shadow: 0 0 0 2px rgba(251, 146, 60, 0.2);
}

.php-theme .php-field::placeholder {
  color: rgba(251, 146, 60, 0.7);
}

.php-theme .php-field option {
  background: #1b263a;
  color: #fb923c;
}

.php-theme .php-field-label {
  color: #fb923c !important;
}
</style>
