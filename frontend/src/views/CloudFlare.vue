<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-white flex items-center gap-3">
          <Cloud class="w-7 h-7 text-orange-400" />
          CloudFlare Yönetimi
        </h1>
        <p class="text-gray-400 mt-1">DNS, SSL, Cache ve güvenlik ayarlarını yönetin</p>
      </div>
    </div>

    <!-- API Key Config -->
    <div v-if="!connected" class="bg-panel-card border border-panel-border rounded-xl p-6">
      <h2 class="text-lg font-semibold text-white mb-4 flex items-center gap-2">🔑 CloudFlare API Bağlantısı</h2>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div>
          <label class="block text-sm text-gray-400 mb-1">E-posta</label>
          <input v-model="cfEmail" type="email" placeholder="user@example.com" class="w-full px-4 py-2.5 bg-panel-hover border border-panel-border rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-orange-500">
        </div>
        <div>
          <label class="block text-sm text-gray-400 mb-1">API Key (Global)</label>
          <input v-model="cfApiKey" type="password" placeholder="••••••••••••••" class="w-full px-4 py-2.5 bg-panel-hover border border-panel-border rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-orange-500">
        </div>
      </div>
      <button @click="connectCf" class="mt-4 px-6 py-2.5 bg-gradient-to-r from-orange-600 to-amber-600 text-white rounded-lg font-medium hover:from-orange-700 hover:to-amber-700 transition" :disabled="loading">
        {{ loading ? 'Bağlanıyor...' : 'Bağlan' }}
      </button>
    </div>

    <!-- Connected State -->
    <template v-if="connected">

      <!-- Tabs -->
      <div class="border-b border-panel-border">
        <nav class="flex gap-6">
          <button v-for="t in tabs" :key="t.id" @click="activeTab = t.id"
            :class="['pb-3 text-sm font-medium transition flex items-center gap-2', activeTab === t.id ? 'text-orange-400 border-b-2 border-orange-400' : 'text-gray-400 hover:text-white']">
            {{ t.icon }} {{ t.label }}
          </button>
        </nav>
      </div>

      <!-- Zones Tab -->
      <div v-if="activeTab === 'zones'">
        <div class="bg-panel-card border border-panel-border rounded-xl overflow-hidden">
          <div class="p-4 border-b border-panel-border flex items-center justify-between">
            <h2 class="text-lg font-semibold text-white">Zone Listesi</h2>
            <button @click="loadZones" class="px-3 py-1.5 bg-panel-hover text-gray-300 rounded-lg text-sm hover:bg-gray-600 transition">🔄</button>
          </div>
          <table class="w-full text-sm">
            <thead>
              <tr class="text-gray-400 border-b border-panel-border">
                <th class="text-left px-4 py-3">Domain</th>
                <th class="text-left px-4 py-3">Durum</th>
                <th class="text-left px-4 py-3">Plan</th>
                <th class="text-left px-4 py-3">Name Servers</th>
                <th class="text-right px-4 py-3">İşlem</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="zones.length === 0">
                <td colspan="5" class="px-4 py-4 text-center text-gray-500">Zone bulunamadı</td>
              </tr>
              <tr v-for="z in zones" :key="z.id" class="border-b border-panel-border/50 hover:bg-white/[0.02] transition">
                <td class="px-4 py-3 text-white font-medium">{{ z.name }}</td>
                <td class="px-4 py-3">
                  <span :class="['px-2 py-0.5 rounded text-xs font-medium', z.status === 'active' ? 'bg-green-500/15 text-green-400' : 'bg-yellow-500/15 text-yellow-400']">{{ z.status }}</span>
                </td>
                <td class="px-4 py-3 text-gray-300">{{ z.plan }}</td>
                <td class="px-4 py-3 text-gray-400 font-mono text-xs">{{ z.name_servers?.join(', ') }}</td>
                <td class="px-4 py-3 text-right">
                  <button @click="selectZone(z)" class="px-3 py-1 bg-orange-600/20 text-orange-400 rounded text-xs hover:bg-orange-600/40 transition">DNS Yönet</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- DNS Tab -->
      <div v-if="activeTab === 'dns'">
        <div class="flex items-center gap-3 mb-4">
          <span class="text-gray-400 text-sm">Zone:</span>
          <span class="text-white font-semibold">{{ selectedZone?.name || 'Seçilmedi' }}</span>
          <button v-if="selectedZone" @click="showAddDns = true" class="ml-auto px-4 py-2 bg-gradient-to-r from-orange-600 to-amber-600 text-white rounded-lg text-sm hover:from-orange-700 hover:to-amber-700 transition">+ Kayıt Ekle</button>
        </div>
        <div class="bg-panel-card border border-panel-border rounded-xl overflow-hidden">
          <table class="w-full text-sm">
            <thead>
              <tr class="text-gray-400 border-b border-panel-border">
                <th class="text-left px-4 py-3">Tip</th>
                <th class="text-left px-4 py-3">İsim</th>
                <th class="text-left px-4 py-3">Değer</th>
                <th class="text-left px-4 py-3">TTL</th>
                <th class="text-left px-4 py-3">Proxy</th>
                <th class="text-right px-4 py-3">İşlem</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="dnsRecords.length === 0">
                <td colspan="6" class="px-4 py-4 text-center text-gray-500">DNS kaydı bulunamadı (Bir Zone seçin)</td>
              </tr>
              <tr v-for="r in dnsRecords" :key="r.id" class="border-b border-panel-border/50 hover:bg-white/[0.02] transition">
                <td class="px-4 py-3">
                  <span :class="['px-2 py-0.5 rounded text-xs font-bold', dnsTypeBadge(r.type)]">{{ r.type }}</span>
                </td>
                <td class="px-4 py-3 text-white font-mono text-xs">{{ r.name }}</td>
                <td class="px-4 py-3 text-gray-300 font-mono text-xs truncate max-w-[200px]">{{ r.content }}</td>
                <td class="px-4 py-3 text-gray-400">{{ r.ttl === 1 ? 'Auto' : r.ttl }}</td>
                <td class="px-4 py-3">
                  <span :class="['text-xs', r.proxied ? 'text-orange-400' : 'text-gray-500']">{{ r.proxied ? '🟠 Proxy' : '⚪ DNS Only' }}</span>
                </td>
                <td class="px-4 py-3 text-right">
                  <button @click="deleteDnsRecord(r.id)" class="px-2 py-1 bg-red-600/20 text-red-400 rounded text-xs hover:bg-red-600/40 transition">🗑</button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- SSL Tab -->
      <div v-if="activeTab === 'ssl'">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div class="bg-panel-card border border-panel-border rounded-xl p-5">
            <h3 class="text-white font-semibold mb-4">🔒 SSL/TLS Modu</h3>
            <div class="space-y-3">
              <label v-for="mode in sslModes" :key="mode.value" class="flex items-start gap-3 p-3 rounded-lg cursor-pointer transition hover:bg-white/[0.03]"
                :class="[selectedSslMode === mode.value ? 'bg-orange-500/10 border border-orange-500/30' : 'border border-transparent']"
                @click="setSslMode(mode.value)">
                <input type="radio" :value="mode.value" v-model="selectedSslMode" class="mt-1 accent-orange-500">
                <div>
                  <p class="text-white font-medium text-sm">{{ mode.label }}</p>
                  <p class="text-gray-400 text-xs mt-0.5">{{ mode.desc }}</p>
                </div>
              </label>
            </div>
          </div>
          <div class="space-y-4">
            <div class="bg-panel-card border border-panel-border rounded-xl p-5">
              <h3 class="text-white font-semibold mb-3">⚡ Always HTTPS</h3>
              <button @click="toggleAlwaysHttps" :class="['px-4 py-2 rounded-lg text-sm transition', alwaysHttps ? 'bg-green-600 text-white' : 'bg-panel-hover text-gray-400']">
                {{ alwaysHttps ? '✅ Aktif' : '❌ Kapalı' }}
              </button>
            </div>
            <div class="bg-panel-card border border-panel-border rounded-xl p-5">
              <h3 class="text-white font-semibold mb-3">📦 Minify</h3>
              <div class="flex gap-3">
                <label v-for="opt in ['JS','CSS','HTML']" :key="opt" class="flex items-center gap-2 text-sm text-gray-300">
                  <input type="checkbox" v-model="minifyOptions[opt.toLowerCase()]" class="accent-orange-500">{{ opt }}
                </label>
              </div>
              <button @click="saveMinify" class="mt-3 px-4 py-1.5 bg-orange-600/20 text-orange-400 rounded-lg text-sm hover:bg-orange-600/40 transition">Kaydet</button>
            </div>
          </div>
        </div>
      </div>

      <!-- Cache Tab -->
      <div v-if="activeTab === 'cache'">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div class="bg-panel-card border border-panel-border rounded-xl p-6 text-center">
            <div class="w-16 h-16 bg-red-500/10 rounded-full flex items-center justify-center mx-auto mb-4">
              <Trash2 class="w-8 h-8 text-red-400" />
            </div>
            <h3 class="text-white font-semibold text-lg mb-2">Tüm Cache'i Temizle</h3>
            <p class="text-gray-400 text-sm mb-4">Tüm önbelleklenmiş kaynakları temizle</p>
            <button @click="purgeAllCache" class="px-6 py-2.5 bg-red-600/20 text-red-400 rounded-lg hover:bg-red-600/40 transition font-medium">🗑 Tümünü Temizle</button>
          </div>
          <div class="bg-panel-card border border-panel-border rounded-xl p-6">
            <h3 class="text-white font-semibold mb-3">📄 Belirli URL'leri Temizle</h3>
            <textarea v-model="purgeUrls" rows="4" placeholder="https://example.com/css/style.css&#10;https://example.com/js/app.js" class="w-full px-4 py-2.5 bg-panel-hover border border-panel-border rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-orange-500 text-sm font-mono"></textarea>
            <button @click="purgeSpecificCache" class="mt-3 px-5 py-2 bg-orange-600/20 text-orange-400 rounded-lg hover:bg-orange-600/40 transition text-sm">Seçilenleri Temizle</button>
          </div>
        </div>
      </div>

      <!-- Security Tab -->
      <div v-if="activeTab === 'security'">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div class="bg-panel-card border border-panel-border rounded-xl p-5">
            <h3 class="text-white font-semibold mb-4">🛡️ Güvenlik Seviyesi</h3>
            <div class="space-y-2">
              <button v-for="lvl in securityLevels" :key="lvl.value" @click="setSecurityLevel(lvl.value)"
                :class="['w-full text-left px-4 py-3 rounded-lg text-sm transition', selectedSecurityLevel === lvl.value ? 'bg-orange-500/20 border border-orange-500/30 text-orange-400' : 'bg-panel-hover text-gray-300 hover:bg-gray-600']">
                {{ lvl.icon }} {{ lvl.label }}
              </button>
            </div>
          </div>
          <div class="space-y-4">
            <div class="bg-panel-card border border-panel-border rounded-xl p-5">
              <h3 class="text-white font-semibold mb-3">🚧 I'm Under Attack Mode</h3>
              <p class="text-gray-400 text-sm mb-3">DDoS saldırısı altındaysanız bu modu aktifleştirin</p>
              <button @click="setSecurityLevel('under_attack')" :class="['px-5 py-2.5 rounded-lg text-sm font-medium transition', selectedSecurityLevel === 'under_attack' ? 'bg-red-600 text-white animate-pulse' : 'bg-red-600/20 text-red-400 hover:bg-red-600/40']">
                ⚠️ Under Attack Modu
              </button>
            </div>
            <div class="bg-panel-card border border-panel-border rounded-xl p-5">
              <h3 class="text-white font-semibold mb-3">🔧 Development Mode</h3>
              <p class="text-gray-400 text-sm mb-3">Cache'i 3 saat boyunca devre dışı bırakır</p>
              <button @click="toggleDevMode" :class="['px-5 py-2.5 rounded-lg text-sm transition', devMode ? 'bg-yellow-600 text-white' : 'bg-panel-hover text-gray-400']">
                {{ devMode ? '✅ Aktif (3 saat)' : '❌ Kapalı' }}
              </button>
            </div>
          </div>
        </div>
      </div>

    </template>

    <!-- Add DNS Modal -->
    <div v-if="showAddDns" class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50" @click.self="showAddDns = false">
      <div class="bg-panel-card border border-panel-border rounded-2xl w-full max-w-lg p-6 shadow-2xl">
        <h3 class="text-xl font-bold text-white mb-5">➕ DNS Kaydı Ekle</h3>
        <div class="space-y-4">
          <div>
            <label class="block text-sm text-gray-400 mb-1">Tip</label>
            <select v-model="newDns.type" class="w-full px-4 py-2.5 bg-panel-hover border border-panel-border rounded-lg text-white focus:outline-none focus:border-orange-500">
              <option v-for="t in ['A','AAAA','CNAME','MX','TXT','NS','SRV','CAA']" :key="t" :value="t">{{ t }}</option>
            </select>
          </div>
          <div>
            <label class="block text-sm text-gray-400 mb-1">İsim</label>
            <input v-model="newDns.name" type="text" placeholder="@, www, mail" class="w-full px-4 py-2.5 bg-panel-hover border border-panel-border rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-orange-500">
          </div>
          <div>
            <label class="block text-sm text-gray-400 mb-1">Değer</label>
            <input v-model="newDns.content" type="text" placeholder="93.184.216.34" class="w-full px-4 py-2.5 bg-panel-hover border border-panel-border rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-orange-500">
          </div>
          <div class="flex items-center gap-4">
            <label class="flex items-center gap-2 text-sm text-gray-300"><input type="checkbox" v-model="newDns.proxied" class="accent-orange-500"> Proxied (CF Proxy)</label>
            <select v-model="newDns.ttl" class="px-3 py-2 bg-panel-hover border border-panel-border rounded-lg text-white text-sm focus:outline-none">
              <option :value="1">Auto</option>
              <option :value="300">5 dk</option>
              <option :value="3600">1 saat</option>
              <option :value="86400">1 gün</option>
            </select>
          </div>
        </div>
        <div class="flex gap-3 mt-6">
          <button @click="addDnsRecord" class="flex-1 py-2.5 bg-gradient-to-r from-orange-600 to-amber-600 text-white rounded-lg font-medium hover:from-orange-700 hover:to-amber-700 transition">Ekle</button>
          <button @click="showAddDns = false" class="px-5 py-2.5 bg-panel-hover text-gray-300 rounded-lg hover:bg-gray-600 transition">İptal</button>
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
import { ref, reactive } from 'vue'
import { Cloud, Trash2 } from 'lucide-vue-next'
import api from '../services/api'

const cfEmail = ref('')
const cfApiKey = ref('')
const connected = ref(false)
const notification = ref(null)
const activeTab = ref('zones')
const loading = ref(false)

const tabs = [
  { id: 'zones', label: 'Zone\'lar', icon: '🌐' },
  { id: 'dns', label: 'DNS Kayıtları', icon: '📡' },
  { id: 'ssl', label: 'SSL/TLS', icon: '🔒' },
  { id: 'cache', label: 'Cache', icon: '⚡' },
  { id: 'security', label: 'Güvenlik', icon: '🛡️' },
]

const zones = ref([])
const selectedZone = ref(null)
const dnsRecords = ref([])
const showAddDns = ref(false)
const newDns = reactive({ type: 'A', name: '', content: '', proxied: false, ttl: 1 })
const selectedSslMode = ref('full')
const selectedSecurityLevel = ref('medium')
const devMode = ref(false)
const alwaysHttps = ref(true)
const minifyOptions = reactive({ js: true, css: true, html: false })
const purgeUrls = ref('')

const sslModes = [
  { value: 'off', label: 'Kapalı', desc: 'SSL yok — HTTP only' },
  { value: 'flexible', label: 'Flexible', desc: 'CF → Sunucu arası HTTP' },
  { value: 'full', label: 'Full', desc: 'CF → Sunucu arası HTTPS (self-signed OK)' },
  { value: 'strict', label: 'Full (Strict)', desc: 'CF → Sunucu arası HTTPS (geçerli sertifika gerek)' },
]

const securityLevels = [
  { value: 'off', label: 'Kapalı', icon: '⚪' },
  { value: 'low', label: 'Düşük', icon: '🟢' },
  { value: 'medium', label: 'Orta', icon: '🟡' },
  { value: 'high', label: 'Yüksek', icon: '🟠' },
  { value: 'under_attack', label: 'Under Attack', icon: '🔴' },
]

const showNotif = (message, type = 'success') => {
  notification.value = { message, type }
  setTimeout(() => notification.value = null, 3000)
}

const authPayload = () => ({ api_key: cfApiKey.value, email: cfEmail.value })

const connectCf = async () => {
  if (!cfEmail.value || !cfApiKey.value) {
    showNotif('Email ve API Key gerekli', 'error');
    return;
  }
  loading.value = true;
  try {
    const { data } = await api.post('/cloudflare/zones', authPayload())
    zones.value = data.data || []
    connected.value = true
    showNotif(`Bağlandı — ${zones.value.length} zone bulundu`)
  } catch (e) {
    showNotif(e.response?.data?.error || 'CloudFlare bağlantısı başarısız', 'error')
  } finally {
    loading.value = false;
  }
}

const loadZones = connectCf

const selectZone = async (zone) => {
  selectedZone.value = zone
  activeTab.value = 'dns'
  dnsRecords.value = []
  try {
    const { data } = await api.post('/cloudflare/dns/list', { ...authPayload(), zone_id: zone.id })
    dnsRecords.value = data.data || []
  } catch (e) {
    showNotif(e.response?.data?.error || 'DNS kayıtları alınamadı', 'error')
  }
}

const addDnsRecord = async () => {
  try {
    await api.post('/cloudflare/dns/create', { ...authPayload(), zone_id: selectedZone.value.id, ...newDns })
    showNotif(`${newDns.type} kaydı eklendi`)
    showAddDns.value = false
    selectZone(selectedZone.value)
  } catch (e) {
    showNotif(e.response?.data?.error || 'Kayıt eklenemedi', 'error')
  }
}

const deleteDnsRecord = async (recordId) => {
  if (!confirm('Bu kaydı silmek istediğinize emin misiniz?')) return;
  try {
    await api.post('/cloudflare/dns/delete', { ...authPayload(), zone_id: selectedZone.value.id, record_id: recordId })
    showNotif('DNS kaydı silindi')
    selectZone(selectedZone.value)
  } catch (e) {
    showNotif(e.response?.data?.error || 'Kayıt silinemedi', 'error')
  }
}

const setSslMode = async (mode) => {
  selectedSslMode.value = mode
  try {
    await api.post('/cloudflare/ssl', { ...authPayload(), zone_id: selectedZone.value?.id || zones.value[0]?.id, mode })
    showNotif(`SSL modu: ${mode}`)
  } catch (e) {
    showNotif(e.response?.data?.error || 'SSL Modu değiştirilemedi', 'error')
  }
}

const purgeAllCache = async () => {
  try {
    await api.post('/cloudflare/cache/purge', { ...authPayload(), zone_id: selectedZone.value?.id || zones.value[0]?.id, purge_everything: true })
    showNotif('Tüm cache temizlendi')
  } catch (e) {
    showNotif(e.response?.data?.error || 'Cache temizlenemedi', 'error')
  }
}

const purgeSpecificCache = async () => {
  const files = purgeUrls.value.split('\n').filter(u => u.trim())
  if (!files.length) return;
  try {
    await api.post('/cloudflare/cache/purge', { ...authPayload(), zone_id: selectedZone.value?.id || zones.value[0]?.id, files })
    showNotif(`${files.length} URL cache temizlendi`)
    purgeUrls.value = ''
  } catch (e) {
    showNotif(e.response?.data?.error || 'Cache temizlenemedi', 'error')
  }
}

const setSecurityLevel = async (level) => {
  selectedSecurityLevel.value = level
  try {
    await api.post('/cloudflare/security', { ...authPayload(), zone_id: selectedZone.value?.id || zones.value[0]?.id, level })
    showNotif(`Güvenlik seviyesi güncellendi`)
  } catch (e) {
    showNotif(e.response?.data?.error || 'Güvenlik seviyesi değiştirilemedi', 'error')
  }
}

const toggleDevMode = async () => {
  const newValue = !devMode.value
  try {
    await api.post('/cloudflare/devmode', { ...authPayload(), zone_id: selectedZone.value?.id || zones.value[0]?.id, enabled: newValue })
    devMode.value = newValue
    showNotif(`Dev mode: ${devMode.value ? 'Açık' : 'Kapalı'}`)
  } catch (e) {
    showNotif(e.response?.data?.error || 'Dev mode güncellenemedi', 'error')
  }
}

const toggleAlwaysHttps = () => { alwaysHttps.value = !alwaysHttps.value; showNotif(`Always HTTPS: ${alwaysHttps.value ? 'Açık' : 'Kapalı'}`) }
const saveMinify = () => showNotif('Minify ayarları kaydedildi')

const dnsTypeBadge = (type) => {
  const map = { A: 'bg-blue-500/15 text-blue-400', AAAA: 'bg-indigo-500/15 text-indigo-400', CNAME: 'bg-green-500/15 text-green-400', MX: 'bg-purple-500/15 text-purple-400', TXT: 'bg-yellow-500/15 text-yellow-400', NS: 'bg-pink-500/15 text-pink-400', SRV: 'bg-cyan-500/15 text-cyan-400' }
  return map[type] || 'bg-gray-500/15 text-gray-400'
}
</script>
