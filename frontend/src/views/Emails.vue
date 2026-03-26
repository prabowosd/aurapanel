<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between gap-3">
      <div>
        <h1 class="text-2xl font-bold text-white">Email Manager</h1>
        <p class="text-gray-400 mt-1">Mailbox, forward, catch-all, routing, DKIM ve webmail SSO yonetimi.</p>
      </div>
      <button class="btn-primary" @click="showAddModal = true">Mailbox Ekle</button>
    </div>

    <div class="border-b border-panel-border">
      <nav class="flex gap-6">
        <button @click="tab='mailboxes'" :class="tabClass('mailboxes')">Mailbox</button>
        <button @click="tab='forwards'" :class="tabClass('forwards')">Forward</button>
        <button @click="tab='routing'" :class="tabClass('routing')">Routing</button>
        <button @click="tab='dkim'" :class="tabClass('dkim')">DKIM</button>
      </nav>
    </div>

    <div v-if="error" class="aura-card border-red-500/30 bg-red-500/5 text-red-400">{{ error }}</div>

    <div v-if="tab === 'mailboxes'" class="space-y-4">
      <div class="flex items-center justify-between">
        <p class="text-sm text-gray-400">{{ mailboxes.length }} mailbox bulundu</p>
        <button class="btn-secondary" @click="loadMailboxes">Yenile</button>
      </div>

      <div v-if="mailboxes.length === 0" class="aura-card text-center py-12 text-gray-500">Mailbox yok.</div>

      <div v-else class="space-y-3">
        <div v-for="mb in mailboxes" :key="mb.address" class="aura-card flex flex-col sm:flex-row gap-3 justify-between sm:items-center">
          <div>
            <p class="text-white font-semibold">{{ mb.address }}</p>
            <p class="text-xs text-gray-400">{{ mb.used_mb }} / {{ mb.quota_mb }} MB</p>
          </div>
          <div class="flex gap-2">
            <button class="btn-secondary px-2 py-1 text-xs" @click="generateWebmailSso(mb.address)">Webmail SSO</button>
            <button class="btn-danger px-2 py-1 text-xs" @click="deleteMailbox(mb.address)">Sil</button>
          </div>
        </div>
      </div>
    </div>

    <div v-if="tab === 'forwards'" class="space-y-4">
      <div class="aura-card space-y-4">
        <h3 class="text-white font-semibold">Forward + Catch-All</h3>
        <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
          <select v-model="forwardForm.domain" class="aura-input">
            <option disabled value="">Domain secin</option>
            <option v-for="d in domains" :key="d" :value="d">{{ d }}</option>
          </select>
          <input v-model="forwardForm.source" class="aura-input" placeholder="info veya info@example.com" />
          <input v-model="forwardForm.target" class="aura-input" placeholder="target@example.net" />
        </div>
        <div class="flex gap-2">
          <button class="btn-primary" @click="addForward">Forward Ekle</button>
          <button class="btn-secondary" @click="loadForwards">Listeyi Yenile</button>
        </div>
      </div>

      <div class="aura-card space-y-3">
        <h3 class="text-white font-semibold">Catch-All</h3>
        <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
          <select v-model="catchAllForm.domain" class="aura-input">
            <option disabled value="">Domain secin</option>
            <option v-for="d in domains" :key="d" :value="d">{{ d }}</option>
          </select>
          <input v-model="catchAllForm.target" class="aura-input" placeholder="catchall@target.com" />
          <label class="inline-flex items-center gap-2 text-sm text-gray-300">
            <input v-model="catchAllForm.enabled" type="checkbox" class="w-4 h-4" />
            Etkin
          </label>
        </div>
        <button class="btn-primary" @click="saveCatchAll">Catch-All Kaydet</button>
      </div>

      <div class="aura-card overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-panel-border text-gray-400">
              <th class="text-left py-2 px-2">Domain</th>
              <th class="text-left py-2 px-2">Source</th>
              <th class="text-left py-2 px-2">Target</th>
              <th class="text-right py-2 px-2">Islem</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="f in forwards" :key="`${f.domain}-${f.source}`" class="border-b border-panel-border/40">
              <td class="py-2 px-2 text-gray-200">{{ f.domain }}</td>
              <td class="py-2 px-2 text-white font-mono">{{ f.source }}</td>
              <td class="py-2 px-2 text-gray-300">{{ f.target }}</td>
              <td class="py-2 px-2 text-right">
                <button class="btn-danger px-2 py-1 text-xs" @click="deleteForward(f)">Sil</button>
              </td>
            </tr>
            <tr v-if="forwards.length === 0"><td colspan="4" class="py-8 text-center text-gray-500">Forward yok.</td></tr>
          </tbody>
        </table>
      </div>
    </div>

    <div v-if="tab === 'routing'" class="space-y-4">
      <div class="aura-card space-y-3">
        <h3 class="text-white font-semibold">Routing Rule</h3>
        <div class="grid grid-cols-1 md:grid-cols-4 gap-3">
          <select v-model="routingForm.domain" class="aura-input">
            <option disabled value="">Domain secin</option>
            <option v-for="d in domains" :key="d" :value="d">{{ d }}</option>
          </select>
          <input v-model="routingForm.pattern" class="aura-input" placeholder="^support\\+.*" />
          <input v-model="routingForm.target" class="aura-input" placeholder="team@example.com" />
          <input v-model.number="routingForm.priority" type="number" class="aura-input" placeholder="100" />
        </div>
        <button class="btn-primary" @click="addRouting">Routing Ekle</button>
      </div>

      <div class="aura-card overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-panel-border text-gray-400">
              <th class="text-left py-2 px-2">Domain</th>
              <th class="text-left py-2 px-2">Pattern</th>
              <th class="text-left py-2 px-2">Target</th>
              <th class="text-left py-2 px-2">Priority</th>
              <th class="text-right py-2 px-2">Islem</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="r in routingRules" :key="r.id" class="border-b border-panel-border/40">
              <td class="py-2 px-2 text-gray-200">{{ r.domain }}</td>
              <td class="py-2 px-2 text-white font-mono">{{ r.pattern }}</td>
              <td class="py-2 px-2 text-gray-300">{{ r.target }}</td>
              <td class="py-2 px-2 text-gray-300">{{ r.priority }}</td>
              <td class="py-2 px-2 text-right">
                <button class="btn-danger px-2 py-1 text-xs" @click="deleteRouting(r)">Sil</button>
              </td>
            </tr>
            <tr v-if="routingRules.length === 0"><td colspan="5" class="py-8 text-center text-gray-500">Routing rule yok.</td></tr>
          </tbody>
        </table>
      </div>
    </div>

    <div v-if="tab === 'dkim'" class="space-y-4">
      <div class="aura-card space-y-3">
        <h3 class="text-white font-semibold">DKIM Manager</h3>
        <div class="flex gap-2">
          <select v-model="dkimDomain" class="aura-input max-w-sm">
            <option disabled value="">Domain secin</option>
            <option v-for="d in domains" :key="d" :value="d">{{ d }}</option>
          </select>
          <button class="btn-secondary" @click="loadDkim">Getir</button>
          <button class="btn-primary" @click="rotateDkim">Rotate</button>
        </div>
        <div v-if="dkimRecord" class="rounded-xl border border-panel-border p-3 space-y-2">
          <p class="text-sm text-gray-300">Selector: <span class="text-white font-mono">{{ dkimRecord.selector }}</span></p>
          <p class="text-xs text-gray-400">TXT: {{ dkimRecord.selector }}._domainkey.{{ dkimRecord.domain }}</p>
          <textarea class="aura-input w-full font-mono text-xs" rows="4" :value="dkimRecord.public_key" readonly />
        </div>
      </div>
    </div>

    <Teleport to="body">
      <div v-if="showAddModal" class="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4">
        <div class="bg-panel-card border border-panel-border rounded-2xl p-6 w-full max-w-md">
          <h2 class="text-xl font-bold text-white mb-4">Yeni Mailbox</h2>
          <div class="space-y-3">
            <input v-model="mailboxForm.address" class="aura-input w-full" placeholder="info@example.com" />
            <input v-model="mailboxForm.password" type="password" class="aura-input w-full" placeholder="Password" />
            <input v-model.number="mailboxForm.quota_mb" type="number" class="aura-input w-full" placeholder="Quota MB" />
          </div>
          <div class="flex gap-3 mt-6">
            <button class="btn-secondary flex-1" @click="showAddModal=false">Iptal</button>
            <button class="btn-primary flex-1" @click="addMailbox">Olustur</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import api from '../services/api'

const tab = ref('mailboxes')
const error = ref('')
const showAddModal = ref(false)

const mailboxes = ref([])
const forwards = ref([])
const routingRules = ref([])
const dkimRecord = ref(null)
const dkimDomain = ref('')

const sites = ref([])

const mailboxForm = ref({
  address: '',
  password: '',
  quota_mb: 2048,
})

const forwardForm = ref({
  domain: '',
  source: '',
  target: '',
})

const catchAllForm = ref({
  domain: '',
  enabled: false,
  target: '',
})

const routingForm = ref({
  domain: '',
  pattern: '',
  target: '',
  priority: 100,
})

const domains = computed(() => {
  const fromSites = (sites.value || []).map(s => s.domain).filter(Boolean)
  const fromMailbox = (mailboxes.value || []).map(m => m.domain).filter(Boolean)
  return [...new Set([...fromSites, ...fromMailbox])]
})

function tabClass(key) {
  return [
    'pb-3 text-sm font-medium transition',
    tab.value === key ? 'text-brand-400 border-b-2 border-brand-400' : 'text-gray-400 hover:text-white',
  ]
}

function apiErrorMessage(e, fallback) {
  return e?.response?.data?.message || e?.message || fallback
}

async function loadSites() {
  try {
    const res = await api.get('/vhost/list')
    sites.value = res.data?.data || []
  } catch {
    sites.value = []
  }
}

async function loadMailboxes() {
  try {
    const res = await api.get('/mail/list')
    mailboxes.value = res.data?.data || []
  } catch (e) {
    error.value = apiErrorMessage(e, 'Mailbox listesi alinamadi')
  }
}

async function addMailbox() {
  const address = String(mailboxForm.value.address || '').trim().toLowerCase()
  if (!address || !address.includes('@')) return
  const [username, domain] = address.split('@')
  try {
    await api.post('/mail/create', {
      domain,
      username,
      password: mailboxForm.value.password || '',
      quota_mb: Number(mailboxForm.value.quota_mb || 2048),
    })
    showAddModal.value = false
    mailboxForm.value = { address: '', password: '', quota_mb: 2048 }
    await loadMailboxes()
  } catch (e) {
    error.value = apiErrorMessage(e, 'Mailbox olusturulamadi')
  }
}

async function deleteMailbox(address) {
  if (!confirm(`${address} silinsin mi?`)) return
  try {
    await api.post('/mail/delete', { address })
    await loadMailboxes()
  } catch (e) {
    error.value = apiErrorMessage(e, 'Mailbox silinemedi')
  }
}

async function loadForwards() {
  try {
    const res = await api.get('/mail/forwards')
    forwards.value = res.data?.data || []
  } catch (e) {
    error.value = apiErrorMessage(e, 'Forward listesi alinamadi')
  }
}

async function addForward() {
  if (!forwardForm.value.domain || !forwardForm.value.source || !forwardForm.value.target) return
  try {
    await api.post('/mail/forwards', {
      domain: forwardForm.value.domain,
      source: forwardForm.value.source,
      target: forwardForm.value.target,
    })
    forwardForm.value.source = ''
    forwardForm.value.target = ''
    await loadForwards()
  } catch (e) {
    error.value = apiErrorMessage(e, 'Forward kaydi olusturulamadi')
  }
}

async function deleteForward(rule) {
  try {
    await api.delete('/mail/forwards', { data: { domain: rule.domain, source: rule.source } })
    await loadForwards()
  } catch (e) {
    error.value = apiErrorMessage(e, 'Forward silinemedi')
  }
}

async function saveCatchAll() {
  if (!catchAllForm.value.domain) return
  try {
    await api.post('/mail/catch-all', {
      domain: catchAllForm.value.domain,
      enabled: !!catchAllForm.value.enabled,
      target: catchAllForm.value.target || '',
    })
  } catch (e) {
    error.value = apiErrorMessage(e, 'Catch-all kaydedilemedi')
  }
}

async function loadRouting() {
  try {
    const res = await api.get('/mail/routing')
    routingRules.value = res.data?.data || []
  } catch (e) {
    error.value = apiErrorMessage(e, 'Routing listesi alinamadi')
  }
}

async function addRouting() {
  if (!routingForm.value.domain || !routingForm.value.pattern || !routingForm.value.target) return
  try {
    await api.post('/mail/routing', {
      domain: routingForm.value.domain,
      pattern: routingForm.value.pattern,
      target: routingForm.value.target,
      priority: Number(routingForm.value.priority || 100),
    })
    routingForm.value.pattern = ''
    routingForm.value.target = ''
    await loadRouting()
  } catch (e) {
    error.value = apiErrorMessage(e, 'Routing rule olusturulamadi')
  }
}

async function deleteRouting(rule) {
  try {
    await api.delete('/mail/routing', { data: { domain: rule.domain, id: rule.id } })
    await loadRouting()
  } catch (e) {
    error.value = apiErrorMessage(e, 'Routing rule silinemedi')
  }
}

async function loadDkim() {
  if (!dkimDomain.value) return
  try {
    const res = await api.get('/mail/dkim', { params: { domain: dkimDomain.value } })
    dkimRecord.value = res.data?.data || null
  } catch (e) {
    error.value = apiErrorMessage(e, 'DKIM bilgisi alinamadi')
  }
}

async function rotateDkim() {
  if (!dkimDomain.value) return
  try {
    const res = await api.post('/mail/dkim/rotate', { domain: dkimDomain.value })
    dkimRecord.value = res.data?.data || null
  } catch (e) {
    error.value = apiErrorMessage(e, 'DKIM rotate basarisiz')
  }
}

async function generateWebmailSso(address) {
  try {
    const res = await api.post('/mail/webmail/sso', { address, ttl_seconds: 300 })
    const url = res.data?.data?.url
    if (url) {
      window.open(url, '_blank', 'noopener,noreferrer')
    }
  } catch (e) {
    error.value = apiErrorMessage(e, 'Webmail SSO link olusturulamadi')
  }
}

onMounted(async () => {
  await Promise.all([loadSites(), loadMailboxes(), loadForwards(), loadRouting()])
  if (domains.value.length > 0) {
    forwardForm.value.domain = domains.value[0]
    catchAllForm.value.domain = domains.value[0]
    routingForm.value.domain = domains.value[0]
    dkimDomain.value = domains.value[0]
  }
})
</script>
