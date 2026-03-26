<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-white">Security Center</h1>
        <p class="text-gray-400 mt-1">
          Zero-Trust guvenlik ozelliklerini plan bazli yonetin.
        </p>
      </div>
      <button class="btn-secondary" @click="loadAll">Yenile</button>
    </div>

    <div class="aura-card">
      <div class="flex flex-wrap gap-2">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          class="px-3 py-2 rounded-lg text-sm transition"
          :class="activeTab === tab.id ? 'bg-brand-500/20 text-brand-300 border border-brand-500/30' : 'bg-panel-dark text-gray-300 border border-panel-border hover:bg-panel-hover'"
          @click="setTab(tab.id)"
        >
          {{ tab.label }}
        </button>
      </div>
    </div>

    <div v-if="activeTab === 'overview'" class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      <div v-for="item in statusCards" :key="item.key" class="aura-card">
        <div class="flex items-center justify-between">
          <h3 class="text-sm font-semibold text-gray-300">{{ item.label }}</h3>
          <span :class="item.value ? 'text-green-400' : 'text-yellow-400'">
            {{ item.value ? 'Aktif' : 'Kismi' }}
          </span>
        </div>
      </div>
    </div>

    <div v-if="activeTab === 'firewall'" class="aura-card space-y-4">
      <h2 class="text-lg font-bold text-white">Firewall (nftables)</h2>
      <div class="grid grid-cols-1 md:grid-cols-4 gap-3">
        <input v-model="firewallForm.ip_address" class="aura-input" placeholder="IP address" />
        <input v-model="firewallForm.reason" class="aura-input" placeholder="Reason" />
        <select v-model="firewallForm.block" class="aura-input">
          <option :value="true">Block</option>
          <option :value="false">Allow</option>
        </select>
        <button class="btn-primary" @click="addFirewallRule">Kural Ekle</button>
      </div>
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="text-gray-400 border-b border-panel-border">
              <th class="text-left py-2">IP</th>
              <th class="text-left py-2">Action</th>
              <th class="text-left py-2">Reason</th>
              <th class="text-right py-2">Islem</th>
            </tr>
          </thead>
          <tbody>
            <tr v-if="firewallRules.length === 0">
              <td colspan="4" class="py-4 text-center text-gray-400">Kural yok</td>
            </tr>
            <tr v-for="rule in firewallRules" :key="rule.ip_address" class="border-b border-panel-border/60">
              <td class="py-2 font-mono">{{ rule.ip_address }}</td>
              <td class="py-2">{{ rule.block ? 'Block' : 'Allow' }}</td>
              <td class="py-2 text-gray-300">{{ rule.reason }}</td>
              <td class="py-2 text-right">
                <button class="btn-danger px-3 py-1 text-xs" @click="deleteFirewallRule(rule.ip_address)">Sil</button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div v-if="activeTab === 'waf'" class="aura-card space-y-4">
      <h2 class="text-lg font-bold text-white">ML-WAF Test</h2>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
        <input v-model="wafInput.path" class="aura-input" placeholder="/path" />
        <input v-model="wafInput.query" class="aura-input" placeholder="query string" />
        <input v-model="wafInput.user_agent" class="aura-input" placeholder="user-agent" />
        <input v-model="wafInput.ip" class="aura-input" placeholder="IP" />
      </div>
      <textarea v-model="wafInput.body" class="aura-input w-full min-h-24" placeholder="request body"></textarea>
      <button class="btn-primary" @click="runWafTest">WAF Analiz Et</button>
      <div v-if="wafResult" class="bg-panel-dark border border-panel-border rounded-lg p-4 text-sm">
        <p><strong>Allowed:</strong> {{ wafResult.allowed }}</p>
        <p><strong>Score:</strong> {{ wafResult.score }}</p>
        <p><strong>Reason:</strong> {{ wafResult.reason }}</p>
      </div>
    </div>

    <div v-if="activeTab === '2fa'" class="aura-card space-y-4">
      <h2 class="text-lg font-bold text-white">2FA (TOTP)</h2>
      <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
        <input v-model="totp.account_name" class="aura-input md:col-span-2" placeholder="Account name (admin@aurapanel)" />
        <button class="btn-primary" @click="setup2fa">Setup 2FA</button>
      </div>
      <div v-if="totp.secret" class="bg-panel-dark border border-panel-border rounded-lg p-4 space-y-3">
        <p class="text-sm"><strong>Secret:</strong> <span class="font-mono">{{ totp.secret }}</span></p>
        <img v-if="totp.qr_base64" :src="`data:image/png;base64,${totp.qr_base64}`" alt="2FA QR" class="w-40 h-40 border border-panel-border rounded" />
        <div class="flex gap-3">
          <input v-model="totp.token" class="aura-input" placeholder="Enter OTP token" />
          <button class="btn-secondary" @click="verify2fa">Verify</button>
        </div>
        <p v-if="totp.verifyResult !== null" :class="totp.verifyResult ? 'text-green-400' : 'text-red-400'">
          {{ totp.verifyResult ? 'Token dogrulandi' : 'Token gecersiz' }}
        </p>
      </div>
    </div>

    <div v-if="activeTab === 'ssh'" class="aura-card space-y-4">
      <h2 class="text-lg font-bold text-white">SSH Key Manager</h2>
      <div class="grid grid-cols-1 md:grid-cols-4 gap-3">
        <input v-model="ssh.user" class="aura-input" placeholder="user" />
        <input v-model="ssh.title" class="aura-input" placeholder="title" />
        <input v-model="ssh.public_key" class="aura-input md:col-span-2" placeholder="ssh-ed25519 AAAA..." />
      </div>
      <div class="flex gap-3">
        <button class="btn-primary" @click="addSshKey">Key Ekle</button>
        <button class="btn-secondary" @click="loadSshKeys">Listele</button>
      </div>
      <div class="space-y-2">
        <div v-for="key in sshKeys" :key="key.id" class="bg-panel-dark border border-panel-border rounded-lg p-3 flex items-center justify-between gap-3">
          <div>
            <p class="text-sm text-white">{{ key.user }} · {{ key.title }}</p>
            <p class="text-xs text-gray-400 font-mono break-all">{{ key.public_key }}</p>
          </div>
          <button class="btn-danger px-3 py-1 text-xs" @click="deleteSshKey(key)">Sil</button>
        </div>
      </div>
    </div>

    <div v-if="activeTab === 'hardening'" class="aura-card space-y-4">
      <h2 class="text-lg font-bold text-white">One-Click Hardening</h2>
      <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
        <select v-model="hardening.stack" class="aura-input">
          <option value="wordpress">WordPress</option>
          <option value="laravel">Laravel</option>
          <option value="generic">Generic</option>
        </select>
        <input v-model="hardening.domain" class="aura-input md:col-span-2" placeholder="example.com" />
      </div>
      <button class="btn-primary" @click="applyHardening">Hardening Uygula</button>
      <div v-if="hardeningResult" class="bg-panel-dark border border-panel-border rounded-lg p-4">
        <p class="text-sm text-white mb-2"><strong>{{ hardeningResult.domain }}</strong> icin uygulanan kurallar:</p>
        <ul class="list-disc pl-5 text-sm text-gray-300">
          <li v-for="rule in hardeningResult.applied_rules" :key="rule">{{ rule }}</li>
        </ul>
      </div>
    </div>

    <div v-if="activeTab === 'kernel'" class="aura-card space-y-4">
      <h2 class="text-lg font-bold text-white">Kernel Security</h2>
      <div class="flex gap-3">
        <button class="btn-secondary" @click="loadImmutableStatus">Immutable Status</button>
        <button class="btn-secondary" @click="loadEbpfEvents">eBPF Events</button>
      </div>
      <div class="flex gap-3">
        <input v-model="livePatchTarget" class="aura-input" placeholder="kernel" />
        <button class="btn-primary" @click="runLivePatch">Live Patch</button>
      </div>
      <pre v-if="immutableStatus" class="bg-panel-dark border border-panel-border rounded-lg p-3 text-xs text-gray-300 overflow-auto">{{ JSON.stringify(immutableStatus, null, 2) }}</pre>
      <div class="space-y-2">
        <div v-for="(ev, idx) in ebpfEvents" :key="idx" class="bg-panel-dark border border-panel-border rounded-lg p-3 text-sm text-gray-300">
          {{ ev }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import api from '../services/api'

const route = useRoute()
const router = useRouter()

const tabs = [
  { id: 'overview', label: 'Overview' },
  { id: 'firewall', label: 'Firewall' },
  { id: 'waf', label: 'ML-WAF' },
  { id: '2fa', label: '2FA' },
  { id: 'ssh', label: 'SSH Keys' },
  { id: 'hardening', label: 'Hardening' },
  { id: 'kernel', label: 'Kernel Security' }
]

const activeTab = ref(route.query.tab || 'overview')
const status = ref({})
const firewallRules = ref([])
const sshKeys = ref([])
const wafResult = ref(null)
const hardeningResult = ref(null)
const immutableStatus = ref(null)
const ebpfEvents = ref([])
const livePatchTarget = ref('kernel')

const firewallForm = ref({
  ip_address: '',
  block: true,
  reason: ''
})

const wafInput = ref({
  method: 'GET',
  path: '/',
  query: '',
  body: '',
  user_agent: 'AuraPanel-Security-Test',
  ip: '127.0.0.1'
})

const totp = ref({
  account_name: 'admin@aurapanel',
  secret: '',
  qr_base64: '',
  token: '',
  verifyResult: null
})

const ssh = ref({
  user: 'root',
  title: '',
  public_key: ''
})

const hardening = ref({
  stack: 'wordpress',
  domain: ''
})

const statusCards = computed(() => [
  { key: 'ebpf', label: 'eBPF Monitoring', value: status.value.ebpf_monitoring },
  { key: 'waf', label: 'ML-WAF', value: status.value.ml_waf },
  { key: 'totp', label: '2FA (TOTP)', value: status.value.totp_2fa },
  { key: 'wg', label: 'WireGuard Federation', value: status.value.wireguard_federation },
  { key: 'immutable', label: 'Immutable OS', value: status.value.immutable_os_support },
  { key: 'livepatch', label: 'Live Patching', value: status.value.live_patching },
  { key: 'hardening', label: 'One-Click Hardening', value: status.value.one_click_hardening },
  { key: 'fw', label: 'nft Firewall', value: status.value.nft_firewall },
  { key: 'ssh', label: 'SSH Key Manager', value: status.value.ssh_key_manager }
])

function setTab(tab) {
  activeTab.value = tab
  router.replace({ query: { ...route.query, tab } })
}

watch(
  () => route.query.tab,
  (tab) => {
    activeTab.value = tab || 'overview'
  }
)

async function loadStatus() {
  const res = await api.get('/security/status')
  status.value = res.data.data || {}
}

async function loadFirewallRules() {
  const res = await api.get('/security/firewall/rules')
  firewallRules.value = res.data.data || []
}

async function addFirewallRule() {
  await api.post('/security/firewall', firewallForm.value)
  firewallForm.value.ip_address = ''
  firewallForm.value.reason = ''
  await loadFirewallRules()
}

async function deleteFirewallRule(ip) {
  await api.delete('/security/firewall/rules', { params: { ip_address: ip } })
  await loadFirewallRules()
}

async function runWafTest() {
  const res = await api.post('/security/waf', wafInput.value)
  wafResult.value = res.data
}

async function setup2fa() {
  const res = await api.post('/security/2fa/setup', { account_name: totp.value.account_name })
  totp.value.secret = res.data.data.secret
  totp.value.qr_base64 = res.data.data.qr_base64
  totp.value.verifyResult = null
}

async function verify2fa() {
  const res = await api.post('/security/2fa/verify', { secret: totp.value.secret, token: totp.value.token })
  totp.value.verifyResult = !!res.data.valid
}

async function addSshKey() {
  await api.post('/security/ssh-keys', ssh.value)
  ssh.value.title = ''
  ssh.value.public_key = ''
  await loadSshKeys()
}

async function loadSshKeys() {
  const params = ssh.value.user ? { user: ssh.value.user } : {}
  const res = await api.get('/security/ssh-keys', { params })
  sshKeys.value = res.data.data || []
}

async function deleteSshKey(key) {
  await api.delete('/security/ssh-keys', {
    params: {
      user: key.user,
      key_id: key.id
    }
  })
  await loadSshKeys()
}

async function applyHardening() {
  const res = await api.post('/security/hardening/apply', hardening.value)
  hardeningResult.value = res.data.data
}

async function loadImmutableStatus() {
  const res = await api.get('/security/immutable/status')
  immutableStatus.value = res.data.data
}

async function loadEbpfEvents() {
  const res = await api.get('/security/ebpf/events')
  ebpfEvents.value = res.data.data || []
}

async function runLivePatch() {
  await api.post('/security/live-patch', { target: livePatchTarget.value })
}

async function loadAll() {
  await Promise.all([loadStatus(), loadFirewallRules(), loadSshKeys()])
}

onMounted(loadAll)
</script>
