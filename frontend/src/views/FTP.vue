<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between gap-3">
      <div>
        <h1 class="text-2xl font-bold text-white">FTP Manager</h1>
        <p class="text-gray-400 mt-1">PureFTPd sanal kullanici yonetimi (create/list/delete/password).</p>
      </div>
      <button class="btn-primary" @click="showCreate = true">FTP Kullanici Ekle</button>
    </div>

    <div v-if="error" class="aura-card border-red-500/30 bg-red-500/5 text-red-400">{{ error }}</div>
    <div v-if="success" class="aura-card border-green-500/30 bg-green-500/5 text-green-300">{{ success }}</div>

    <div class="aura-card space-y-4">
      <div class="flex flex-col md:flex-row md:items-end gap-3">
        <div class="w-full md:max-w-sm">
          <label class="block text-sm text-gray-400 mb-1">Domain Filtresi</label>
          <select v-model="selectedDomain" class="aura-input w-full" @change="onDomainFilterChange">
            <option value="">Tum domainler</option>
            <option v-for="d in domains" :key="d" :value="d">{{ d }}</option>
          </select>
        </div>
        <button class="btn-secondary" @click="loadFtpUsers">Yenile</button>
      </div>

      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-panel-border text-gray-400">
              <th class="text-left py-2 px-2">Username</th>
              <th class="text-left py-2 px-2">Domain</th>
              <th class="text-left py-2 px-2">Home</th>
              <th class="text-left py-2 px-2">Created</th>
              <th class="text-right py-2 px-2">Islem</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="item in ftpUsers"
              :key="item.username"
              class="border-b border-panel-border/50"
            >
              <td class="py-2 px-2 text-white font-mono">{{ item.username }}</td>
              <td class="py-2 px-2 text-gray-300">{{ item.domain || '-' }}</td>
              <td class="py-2 px-2 text-gray-400 font-mono text-xs break-all">{{ item.home_dir }}</td>
              <td class="py-2 px-2 text-gray-400">{{ formatTime(item.created_at) }}</td>
              <td class="py-2 px-2 text-right">
                <div class="flex justify-end gap-2">
                  <button class="btn-secondary px-2 py-1 text-xs" @click="openReset(item.username)">Sifre</button>
                  <button class="btn-danger px-2 py-1 text-xs" @click="removeUser(item.username)">Sil</button>
                </div>
              </td>
            </tr>
            <tr v-if="ftpUsers.length === 0">
              <td colspan="5" class="text-center py-8 text-gray-500">FTP kullanicisi bulunamadi.</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <Teleport to="body">
      <div v-if="showCreate" class="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4">
        <div class="bg-panel-card border border-panel-border rounded-2xl p-6 w-full max-w-lg">
          <h2 class="text-xl font-bold text-white mb-4">FTP Kullanici Olustur</h2>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
            <div>
              <label class="block text-sm text-gray-400 mb-1">Domain (opsiyonel)</label>
              <select v-model="createForm.domain" class="aura-input w-full" @change="onCreateDomainChange">
                <option value="">Secilmedi</option>
                <option v-for="d in domains" :key="d" :value="d">{{ d }}</option>
              </select>
            </div>
            <div>
              <label class="block text-sm text-gray-400 mb-1">Username</label>
              <input v-model="createForm.username" class="aura-input w-full" placeholder="siteftp" />
            </div>
            <div class="md:col-span-2">
              <label class="block text-sm text-gray-400 mb-1">Password</label>
              <input v-model="createForm.password" type="password" class="aura-input w-full" />
            </div>
            <div class="md:col-span-2">
              <label class="block text-sm text-gray-400 mb-1">Home Directory</label>
              <input v-model="createForm.home_dir" class="aura-input w-full" placeholder="/home/domain/public_html" />
            </div>
          </div>
          <div class="flex gap-3 mt-6">
            <button class="btn-secondary flex-1" @click="showCreate = false">Iptal</button>
            <button class="btn-primary flex-1" @click="createUser">Olustur</button>
          </div>
        </div>
      </div>
    </Teleport>

    <Teleport to="body">
      <div v-if="showReset" class="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4">
        <div class="bg-panel-card border border-panel-border rounded-2xl p-6 w-full max-w-md">
          <h2 class="text-xl font-bold text-white mb-4">FTP Sifre Guncelle</h2>
          <p class="text-sm text-gray-400 mb-3">Kullanici: <span class="text-white font-mono">{{ resetForm.username }}</span></p>
          <input v-model="resetForm.new_password" type="password" class="aura-input w-full" placeholder="Yeni sifre" />
          <div class="flex gap-3 mt-6">
            <button class="btn-secondary flex-1" @click="showReset = false">Iptal</button>
            <button class="btn-primary flex-1" @click="updatePassword">Guncelle</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import api from '../services/api'

const route = useRoute()
const router = useRouter()

const ftpUsers = ref([])
const sites = ref([])
const error = ref('')
const success = ref('')
const showCreate = ref(false)
const showReset = ref(false)
const selectedDomain = ref(typeof route.query.domain === 'string' ? route.query.domain : '')

const createForm = ref({
  domain: selectedDomain.value,
  username: '',
  password: '',
  home_dir: '',
})

const resetForm = ref({
  username: '',
  new_password: '',
})

const domains = computed(() => (sites.value || []).map((x) => x.domain).filter(Boolean))

watch(
  () => route.query.domain,
  (value) => {
    selectedDomain.value = typeof value === 'string' ? value : ''
    loadFtpUsers()
  }
)

function formatTime(ts) {
  const value = Number(ts || 0)
  if (!value) return '-'
  return new Date(value * 1000).toLocaleString()
}

function apiErrorMessage(e, fallback) {
  return e?.response?.data?.message || e?.message || fallback
}

function defaultHome(domain) {
  if (domain) {
    return `/home/${domain}/public_html`
  }
  return '/home'
}

function onCreateDomainChange() {
  if (!createForm.value.home_dir || createForm.value.home_dir === '/home') {
    createForm.value.home_dir = defaultHome(createForm.value.domain)
  }
}

function onDomainFilterChange() {
  router.replace({
    path: '/ftp',
    query: selectedDomain.value ? { domain: selectedDomain.value } : {},
  })
}

async function loadSites() {
  try {
    const res = await api.get('/vhost/list')
    sites.value = res.data?.data || []
  } catch {
    sites.value = []
  }
}

async function loadFtpUsers() {
  error.value = ''
  try {
    const params = selectedDomain.value ? { domain: selectedDomain.value } : {}
    const res = await api.get('/ftp/list', { params })
    ftpUsers.value = res.data?.data || []
  } catch (e) {
    error.value = apiErrorMessage(e, 'FTP kullanici listesi alinamadi')
  }
}

async function createUser() {
  error.value = ''
  success.value = ''

  const payload = {
    username: createForm.value.username,
    password: createForm.value.password,
    home_dir: createForm.value.home_dir || defaultHome(createForm.value.domain),
    domain: createForm.value.domain || undefined,
  }
  if (!payload.username || !payload.password || !payload.home_dir) {
    error.value = 'username, password ve home_dir zorunludur.'
    return
  }

  try {
    const res = await api.post('/ftp/create', payload)
    success.value = res.data?.message || 'FTP kullanici olusturuldu.'
    showCreate.value = false
    createForm.value = {
      domain: selectedDomain.value,
      username: '',
      password: '',
      home_dir: defaultHome(selectedDomain.value),
    }
    await loadFtpUsers()
  } catch (e) {
    error.value = apiErrorMessage(e, 'FTP kullanici olusturulamadi')
  }
}

function openReset(username) {
  resetForm.value.username = username
  resetForm.value.new_password = ''
  showReset.value = true
}

async function updatePassword() {
  error.value = ''
  success.value = ''

  if (!resetForm.value.username || !resetForm.value.new_password) {
    error.value = 'username ve yeni sifre zorunludur.'
    return
  }

  try {
    const res = await api.post('/ftp/password', {
      username: resetForm.value.username,
      new_password: resetForm.value.new_password,
    })
    success.value = res.data?.message || 'FTP sifresi guncellendi.'
    showReset.value = false
  } catch (e) {
    error.value = apiErrorMessage(e, 'FTP sifresi guncellenemedi')
  }
}

async function removeUser(username) {
  if (!confirm(`${username} kullanicisi silinsin mi?`)) return
  error.value = ''
  success.value = ''
  try {
    const res = await api.post('/ftp/delete', { username })
    success.value = res.data?.message || 'FTP kullanici silindi.'
    await loadFtpUsers()
  } catch (e) {
    error.value = apiErrorMessage(e, 'FTP kullanici silinemedi')
  }
}

onMounted(async () => {
  createForm.value.home_dir = defaultHome(createForm.value.domain)
  await Promise.all([loadSites(), loadFtpUsers()])
})
</script>
