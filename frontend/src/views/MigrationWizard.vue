<template>
  <div class="space-y-6">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div>
        <h1 class="text-2xl font-bold text-white">Migration Wizard</h1>
        <p class="text-sm text-gray-400 mt-1">
          cPanel/CyberPanel hesap (tek hesap/site) yedeklerini analiz edin, dönüşüm planını ve import durumunu izleyin.
        </p>
      </div>
      <button class="btn-secondary" @click="resetAll">Temizle</button>
    </div>

    <div v-if="error" class="aura-card border-red-500/30 bg-red-500/5 text-red-400">
      {{ error }}
    </div>
    <div v-if="success" class="aura-card border-green-500/30 bg-green-500/5 text-green-400">
      {{ success }}
    </div>

    <div class="aura-card space-y-4">
      <h2 class="text-lg font-semibold text-white">1. Backup Yükle ve Kaynak Seç</h2>
      <p class="text-xs text-gray-400">
        Not: Bu ekran şu an tam sunucu imajı değil, hesap/site tabanlı backup (.tar.gz/.tgz/.zip) importu içindir.
      </p>
      <div class="grid grid-cols-1 gap-3 lg:grid-cols-3">
        <div class="space-y-2 lg:col-span-2">
          <label class="block text-sm text-gray-400">Backup dosyası (.tar.gz / .tgz / .zip)</label>
          <input
            type="file"
            accept=".tar,.tar.gz,.tgz,.zip,application/gzip,application/x-gzip,application/zip"
            class="aura-input w-full"
            @change="onFileSelect"
          />
        </div>
        <div class="space-y-2">
          <label class="block text-sm text-gray-400">Kaynak panel</label>
          <select v-model="sourceType" class="aura-input w-full">
            <option value="auto">Otomatik</option>
            <option value="cpanel">cPanel</option>
            <option value="cyberpanel">CyberPanel</option>
            <option value="plesk">Plesk</option>
            <option value="generic">Generic</option>
          </select>
        </div>
      </div>

      <div class="flex flex-wrap gap-2">
        <button class="btn-primary" :disabled="uploading || !selectedFile" @click="uploadBackup">
          {{ uploading ? 'Yükleniyor...' : 'Yedeği Yükle' }}
        </button>
      </div>

      <div v-if="uploading || uploadProgress > 0" class="space-y-1">
        <div class="h-2 rounded bg-panel-dark overflow-hidden">
          <div class="h-full bg-gradient-to-r from-blue-500 to-brand-500 transition-all" :style="{ width: `${uploadProgress}%` }"></div>
        </div>
        <p class="text-xs text-gray-400">Upload: %{{ uploadProgress }}</p>
      </div>

      <div class="space-y-2">
        <label class="block text-sm text-gray-400">Arşiv yolu</label>
        <input
          v-model="archivePath"
          class="aura-input w-full font-mono text-xs"
          placeholder="/var/lib/aurapanel/migrations/uploads/cpmove-account.tar.gz"
        />
      </div>
    </div>

    <div class="aura-card space-y-4">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <h2 class="text-lg font-semibold text-white">2. Backup Analizi</h2>
        <button class="btn-primary" :disabled="analyzing || !archivePath" @click="analyzeBackup">
          {{ analyzing ? 'Analiz ediliyor...' : 'Analiz Et' }}
        </button>
      </div>

      <div v-if="analysis" class="space-y-4">
        <div class="grid grid-cols-1 gap-3 md:grid-cols-6">
          <div class="rounded-lg border border-panel-border bg-panel-dark p-3">
            <p class="text-xs text-gray-400">Panel</p>
            <p class="text-sm text-white mt-1">{{ analysis.source_type }}</p>
          </div>
          <div class="rounded-lg border border-panel-border bg-panel-dark p-3">
            <p class="text-xs text-gray-400">Dosya</p>
            <p class="text-sm text-white mt-1">{{ analysis.stats.file_count }}</p>
          </div>
          <div class="rounded-lg border border-panel-border bg-panel-dark p-3">
            <p class="text-xs text-gray-400">DB</p>
            <p class="text-sm text-white mt-1">{{ analysis.stats.database_count }}</p>
          </div>
          <div class="rounded-lg border border-panel-border bg-panel-dark p-3">
            <p class="text-xs text-gray-400">E-Posta</p>
            <p class="text-sm text-white mt-1">{{ analysis.stats.email_count }}</p>
          </div>
          <div class="rounded-lg border border-panel-border bg-panel-dark p-3">
            <p class="text-xs text-gray-400">Arşiv Boyutu</p>
            <p class="text-sm text-white mt-1">{{ analysis.archive_size_human || '-' }}</p>
          </div>
          <div class="rounded-lg border border-panel-border bg-panel-dark p-3">
            <p class="text-xs text-gray-400">Pre-check</p>
            <p class="text-sm mt-1" :class="isPrecheckReady ? 'text-green-400' : 'text-red-400'">
              {{ isPrecheckReady ? 'READY' : 'BLOCKED' }}
            </p>
          </div>
        </div>

        <div class="rounded-lg border border-panel-border p-3 space-y-3">
          <div class="flex flex-wrap items-center justify-between gap-2">
            <h3 class="text-sm font-semibold text-white">Pre-check Raporu</h3>
            <p class="text-xs text-gray-300">
              Tahmini süre: <span class="text-white">{{ etaText(analysis.precheck?.eta_seconds) }}</span>
            </p>
          </div>

          <div class="space-y-2 text-xs">
            <div
              v-for="(item, idx) in analysis.precheck?.checks || []"
              :key="`${item.name}-${idx}`"
              class="rounded border border-panel-border bg-panel-dark px-3 py-2"
            >
              <p class="font-semibold" :class="checkClass(item.status)">{{ item.name }} - {{ item.status }}</p>
              <p class="text-gray-300 mt-1">{{ item.detail }}</p>
            </div>
          </div>

          <div v-if="(analysis.precheck?.conflicts || []).length" class="rounded-lg border border-red-500/30 bg-red-500/5 p-3">
            <p class="text-xs text-red-300 mb-2">Conflicts:</p>
            <p
              v-for="(conflict, idx) in analysis.precheck?.conflicts || []"
              :key="`${conflict.type}-${conflict.target}-${idx}`"
              class="text-xs mb-1"
              :class="conflictClass(conflict.severity)"
            >
              - [{{ conflict.type }}] {{ conflict.target }}: {{ conflict.message }}
            </p>
          </div>

          <div v-if="(analysis.precheck?.recommendations || []).length" class="rounded-lg border border-cyan-500/30 bg-cyan-500/5 p-3">
            <p class="text-xs text-cyan-300 mb-2">Öneriler:</p>
            <p v-for="(rec, idx) in analysis.precheck?.recommendations || []" :key="idx" class="text-xs text-cyan-100 mb-1">
              - {{ rec }}
            </p>
          </div>
        </div>

        <div class="grid grid-cols-1 gap-4 lg:grid-cols-3">
          <div class="rounded-lg border border-panel-border p-3">
            <h3 class="text-sm font-semibold text-white mb-2">MySQL Dump Dosyaları</h3>
            <div class="max-h-48 overflow-auto text-xs font-mono text-gray-300 space-y-1">
              <p v-for="item in analysis.mysql_dumps" :key="item">{{ item }}</p>
              <p v-if="analysis.mysql_dumps.length === 0" class="text-gray-500">Kayıt yok.</p>
            </div>
          </div>
          <div class="rounded-lg border border-panel-border p-3">
            <h3 class="text-sm font-semibold text-white mb-2">E-Posta Hesapları</h3>
            <div class="max-h-48 overflow-auto text-xs font-mono text-gray-300 space-y-1">
              <p v-for="item in analysis.email_accounts" :key="item">{{ item }}</p>
              <p v-if="analysis.email_accounts.length === 0" class="text-gray-500">Kayıt yok.</p>
            </div>
          </div>
          <div class="rounded-lg border border-panel-border p-3">
            <h3 class="text-sm font-semibold text-white mb-2">VHost Adayları</h3>
            <div class="max-h-48 overflow-auto text-xs font-mono text-gray-300 space-y-1">
              <p v-for="item in analysis.vhost_candidates" :key="item">{{ item }}</p>
              <p v-if="analysis.vhost_candidates.length === 0" class="text-gray-500">Kayıt yok.</p>
            </div>
          </div>
        </div>

        <div v-if="analysis.warnings?.length" class="rounded-lg border border-yellow-500/30 bg-yellow-500/5 p-3">
          <p class="text-xs text-yellow-300 mb-1">Uyarılar:</p>
          <p v-for="(w, i) in analysis.warnings" :key="i" class="text-xs text-yellow-200">- {{ w }}</p>
        </div>
      </div>
    </div>

    <div class="aura-card space-y-4">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <h2 class="text-lg font-semibold text-white">3. Import Başlat ve İzle</h2>
        <button class="btn-primary" :disabled="importStarting || !archivePath || !isPrecheckReady" @click="startImport">
          {{ importStarting ? 'Başlatılıyor...' : 'Import Başlat' }}
        </button>
      </div>
      <p class="text-xs text-gray-400">Import sadece pre-check sonucu READY ise başlatılır.</p>

      <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
        <div>
          <label class="block text-sm text-gray-400 mb-1">Hedef owner</label>
          <input v-model="targetOwner" class="aura-input w-full" placeholder="aura" />
        </div>
        <div>
          <label class="block text-sm text-gray-400 mb-1">Job ID</label>
          <input :value="job?.id || '-'" class="aura-input w-full font-mono text-xs" disabled />
        </div>
      </div>

      <div v-if="job" class="space-y-3">
        <div class="flex flex-wrap items-center gap-2 text-sm">
          <span class="text-gray-400">Durum:</span>
          <span :class="statusClass(job.status)">{{ job.status }}</span>
          <span class="text-gray-500">|</span>
          <span class="text-gray-300">%{{ job.progress }}</span>
          <button class="btn-secondary ml-auto" @click="fetchJobStatus">Yenile</button>
        </div>

        <div class="h-2 rounded bg-panel-dark overflow-hidden">
          <div class="h-full bg-gradient-to-r from-emerald-500 to-cyan-500 transition-all" :style="{ width: `${job.progress || 0}%` }"></div>
        </div>

        <div class="rounded-lg border border-panel-border bg-panel-dark p-3">
          <p class="text-xs text-gray-400 mb-2">Canlı Log</p>
          <pre class="max-h-64 overflow-auto text-xs text-gray-200 whitespace-pre-wrap">{{ (job.logs || []).join('\n') }}</pre>
        </div>

        <div v-if="job.summary" class="rounded-lg border border-panel-border p-3 space-y-2">
          <p class="text-sm text-white font-semibold">Import Özeti</p>
          <p class="text-xs text-gray-300">DB çıktı dosyası: {{ job.summary.converted_db_files?.length || 0 }}</p>
          <p class="text-xs text-gray-300 font-mono">Mail plani: {{ job.summary.email_plan_file }}</p>
          <p class="text-xs text-gray-300 font-mono">VHost plani: {{ job.summary.vhost_plan_file }}</p>
          <p class="text-xs text-gray-400">
            Sistem import modu:
            <span class="text-gray-200">{{ job.summary.system_apply_enabled ? 'AKTİF' : 'DRY-RUN' }}</span>
          </p>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, ref } from 'vue'
import api from '../services/api'

const selectedFile = ref(null)
const archivePath = ref('')
const sourceType = ref('auto')
const targetOwner = ref('aura')

const uploading = ref(false)
const uploadProgress = ref(0)
const analyzing = ref(false)
const importStarting = ref(false)

const analysis = ref(null)
const job = ref(null)
const error = ref('')
const success = ref('')
const isPrecheckReady = computed(() => Boolean(analysis.value?.precheck?.ready))

let pollTimer = null

function setError(message) {
  error.value = message
  success.value = ''
}

function setSuccess(message) {
  success.value = message
  error.value = ''
}

function onFileSelect(event) {
  const file = event?.target?.files?.[0]
  selectedFile.value = file || null
}

async function uploadBackup() {
  if (!selectedFile.value) return
  error.value = ''
  success.value = ''
  uploading.value = true
  uploadProgress.value = 0
  try {
    const form = new FormData()
    form.append('file', selectedFile.value)
    const res = await api.post('/migration/upload', form, {
      timeout: 0,
      onUploadProgress: (evt) => {
        if (!evt?.total) return
        uploadProgress.value = Math.round((evt.loaded * 100) / evt.total)
      },
    })
    archivePath.value = res.data?.data?.archive_path || ''
    setSuccess('Backup dosyası yüklendi.')
  } catch (err) {
    setError(err?.response?.data?.message || 'Backup yüklenemedi.')
  } finally {
    uploading.value = false
  }
}

async function analyzeBackup() {
  if (!archivePath.value) return
  analyzing.value = true
  error.value = ''
  success.value = ''
  try {
    const payload = {
      archive_path: archivePath.value,
      source_type: sourceType.value === 'auto' ? null : sourceType.value,
      target_owner: targetOwner.value || 'aura',
    }
    const res = await api.post('/migration/analyze', payload)
    analysis.value = res.data?.data || null
    setSuccess('Backup analizi tamamlandı.')
  } catch (err) {
    setError(err?.response?.data?.message || 'Backup analizi başarısız.')
  } finally {
    analyzing.value = false
  }
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

function startPolling() {
  stopPolling()
  pollTimer = setInterval(async () => {
    await fetchJobStatus()
  }, 2000)
}

async function startImport() {
  if (!archivePath.value) return
  if (!isPrecheckReady.value) {
    setError('Pre-check sonucunda blocker var. Import başlatılmadı.')
    return
  }
  importStarting.value = true
  error.value = ''
  success.value = ''
  try {
    const payload = {
      archive_path: archivePath.value,
      source_type: sourceType.value === 'auto' ? null : sourceType.value,
      target_owner: targetOwner.value || 'aura',
      allow_conflicts: false,
    }
    const res = await api.post('/migration/import/start', payload)
    job.value = res.data?.data || null
    if (job.value?.id) {
      startPolling()
    }
    setSuccess('Import kuyruğa alındı.')
  } catch (err) {
    setError(err?.response?.data?.message || 'Import başlatılamadı.')
  } finally {
    importStarting.value = false
  }
}

async function fetchJobStatus() {
  if (!job.value?.id) return
  try {
    const res = await api.get('/migration/import/status', { params: { id: job.value.id } })
    job.value = res.data?.data || job.value
    const state = String(job.value?.status || '').toLowerCase()
    if (state === 'completed' || state === 'failed') {
      stopPolling()
    }
  } catch (err) {
    setError(err?.response?.data?.message || 'Job durumu alınamadı.')
    stopPolling()
  }
}

function statusClass(status) {
  const value = String(status || '').toLowerCase()
  if (value === 'completed') return 'text-green-400'
  if (value === 'failed') return 'text-red-400'
  if (value === 'running') return 'text-blue-400'
  return 'text-yellow-400'
}

function checkClass(status) {
  const value = String(status || '').toLowerCase()
  if (value === 'pass') return 'text-green-400'
  if (value === 'warn') return 'text-yellow-300'
  if (value === 'fail') return 'text-red-400'
  return 'text-gray-300'
}

function conflictClass(severity) {
  const value = String(severity || '').toLowerCase()
  if (value === 'high') return 'text-red-300'
  if (value === 'medium') return 'text-orange-300'
  return 'text-yellow-200'
}

function etaText(seconds) {
  const raw = Number(seconds || 0)
  if (!raw) return '-'
  const mins = Math.round(raw / 60)
  if (mins < 1) return `${raw}s`
  if (mins < 60) return `${mins} dk`
  const h = Math.floor(mins / 60)
  const m = mins % 60
  return m === 0 ? `${h} sa` : `${h} sa ${m} dk`
}

function resetAll() {
  stopPolling()
  selectedFile.value = null
  archivePath.value = ''
  sourceType.value = 'auto'
  targetOwner.value = 'aura'
  uploadProgress.value = 0
  analysis.value = null
  job.value = null
  error.value = ''
  success.value = ''
}

onBeforeUnmount(() => {
  stopPolling()
})
</script>

