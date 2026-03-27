<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between gap-4">
      <div>
        <h1 class="text-2xl font-bold text-white flex items-center gap-3">
          <HardDrive class="w-7 h-7 text-teal-400" />
          {{ t('db_backup.title') }}
        </h1>
        <p class="text-gray-400 mt-1">{{ t('db_backup.subtitle') }}</p>
      </div>
      <button
        @click="loadAll"
        class="px-4 py-2 bg-panel-hover text-gray-300 rounded-lg text-sm hover:bg-gray-600 transition"
      >
        Yenile
      </button>
    </div>

    <!-- Engine Tabs -->
    <div class="border-b border-panel-border">
      <nav class="flex gap-6">
        <button
          @click="activeEngine = 'mariadb'"
          :class="['pb-3 text-sm font-medium transition', activeEngine === 'mariadb' ? 'text-orange-400 border-b-2 border-orange-400' : 'text-gray-400 hover:text-white']"
        >
          MariaDB / MySQL
        </button>
        <button
          @click="activeEngine = 'postgres'"
          :class="['pb-3 text-sm font-medium transition', activeEngine === 'postgres' ? 'text-blue-400 border-b-2 border-blue-400' : 'text-gray-400 hover:text-white']"
        >
          PostgreSQL
        </button>
      </nav>
    </div>

    <!-- Database List -->
    <div class="bg-panel-card border border-panel-border rounded-xl overflow-hidden">
      <div class="p-4 border-b border-panel-border">
        <h2 class="text-lg font-semibold text-white">
          {{ activeEngine === 'mariadb' ? 'MariaDB' : 'PostgreSQL' }} Veritabanları
        </h2>
      </div>
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="text-gray-400 border-b border-panel-border">
              <th class="text-left px-4 py-3 font-medium">Veritabanı</th>
              <th class="text-left px-4 py-3 font-medium">Boyut</th>
              <th class="text-right px-4 py-3 font-medium">İşlem</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="db in currentDatabases"
              :key="db.name"
              class="border-b border-panel-border/50 hover:bg-white/[0.02] transition"
            >
              <td class="px-4 py-3 text-white font-medium font-mono">{{ db.name }}</td>
              <td class="px-4 py-3 text-gray-400">{{ db.size || '-' }}</td>
              <td class="px-4 py-3 text-right">
                <button
                  @click="createBackup(db.name)"
                  :disabled="backingUp === db.name"
                  class="px-3 py-1.5 bg-teal-600/20 text-teal-300 rounded-lg text-xs hover:bg-teal-600/40 transition disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1.5 ml-auto"
                >
                  <DatabaseBackupIcon class="w-3.5 h-3.5" />
                  <span v-if="backingUp === db.name">Yedekleniyor...</span>
                  <span v-else>{{ t('db_backup.create_backup') }}</span>
                </button>
              </td>
            </tr>
            <tr v-if="currentDatabases.length === 0">
              <td colspan="3" class="px-4 py-12 text-center text-gray-500">
                Veritabanı bulunamadı
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Backup History -->
    <div class="bg-panel-card border border-panel-border rounded-xl overflow-hidden">
      <div class="p-4 border-b border-panel-border flex items-center justify-between">
        <h2 class="text-lg font-semibold text-white">{{ t('db_backup.backup_history') }}</h2>
        <button @click="loadBackups" class="px-3 py-1.5 bg-panel-hover text-gray-300 rounded-lg text-sm hover:bg-gray-600 transition">
          Yenile
        </button>
      </div>
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="text-gray-400 border-b border-panel-border">
              <th class="text-left px-4 py-3 font-medium">Dosya</th>
              <th class="text-left px-4 py-3 font-medium">Engine</th>
              <th class="text-left px-4 py-3 font-medium">Boyut</th>
              <th class="text-left px-4 py-3 font-medium">Tarih</th>
              <th class="text-right px-4 py-3 font-medium">İşlem</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="backup in backups"
              :key="backup.id || backup.filename"
              class="border-b border-panel-border/50 hover:bg-white/[0.02] transition"
            >
              <td class="px-4 py-3 text-white font-mono text-xs">{{ backup.filename || backup.id }}</td>
              <td class="px-4 py-3">
                <span :class="['px-2 py-0.5 rounded text-xs font-medium', backup.engine === 'mariadb' ? 'bg-orange-500/15 text-orange-400' : 'bg-blue-500/15 text-blue-400']">
                  {{ backup.engine === 'mariadb' ? 'MariaDB' : 'PostgreSQL' }}
                </span>
              </td>
              <td class="px-4 py-3 text-gray-400">{{ backup.size || '-' }}</td>
              <td class="px-4 py-3 text-gray-400 text-xs">{{ formatDate(backup.created_at) }}</td>
              <td class="px-4 py-3 text-right space-x-2">
                <button
                  @click="downloadBackup(backup)"
                  class="px-2 py-1 bg-blue-600/20 text-blue-300 rounded text-xs hover:bg-blue-600/40 transition"
                >
                  {{ t('db_backup.download') }}
                </button>
                <button
                  @click="restoreBackup(backup)"
                  :disabled="restoring === (backup.id || backup.filename)"
                  class="px-2 py-1 bg-green-600/20 text-green-300 rounded text-xs hover:bg-green-600/40 transition disabled:opacity-50"
                >
                  {{ t('db_backup.restore') }}
                </button>
                <button
                  @click="deleteBackup(backup)"
                  class="px-2 py-1 bg-red-600/20 text-red-400 rounded text-xs hover:bg-red-600/40 transition"
                >
                  {{ t('common.delete') }}
                </button>
              </td>
            </tr>
            <tr v-if="backups.length === 0">
              <td colspan="5" class="px-4 py-12 text-center text-gray-500">
                <HardDrive class="w-8 h-8 mx-auto mb-2 opacity-30" />
                {{ t('db_backup.no_backups') }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Notification Toast -->
    <div
      v-if="notification"
      :class="['fixed bottom-6 right-6 px-5 py-3 rounded-xl shadow-2xl text-sm font-medium z-50 transition-all', notification.type === 'success' ? 'bg-green-600 text-white' : 'bg-red-600 text-white']"
    >
      {{ notification.message }}
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { HardDrive, DatabaseBackup as DatabaseBackupIcon } from 'lucide-vue-next'
import api from '../services/api'

const { t } = useI18n()

const activeEngine = ref('mariadb')
const mariadbDbs = ref([])
const postgresDbs = ref([])
const backups = ref([])
const backingUp = ref(null)
const restoring = ref(null)
const notification = ref(null)

const currentDatabases = computed(() =>
  activeEngine.value === 'mariadb' ? mariadbDbs.value : postgresDbs.value
)

function showNotif(message, type = 'success') {
  notification.value = { message, type }
  setTimeout(() => { notification.value = null }, 3500)
}

function formatDate(timestamp) {
  if (!timestamp) return '-'
  try {
    return new Date(timestamp).toLocaleString('tr-TR', {
      year: 'numeric', month: '2-digit', day: '2-digit',
      hour: '2-digit', minute: '2-digit',
    })
  } catch {
    return String(timestamp)
  }
}

async function loadDatabases() {
  const [mariaRes, pgRes] = await Promise.allSettled([
    api.get('/db/mariadb/list'),
    api.get('/db/postgres/list'),
  ])
  mariadbDbs.value = mariaRes.status === 'fulfilled' ? mariaRes.value.data?.data || [] : []
  postgresDbs.value = pgRes.status === 'fulfilled' ? pgRes.value.data?.data || [] : []
}

async function loadBackups() {
  try {
    const res = await api.get('/db/backup/list')
    backups.value = res.data?.data || []
  } catch {
    backups.value = []
  }
}

async function loadAll() {
  await Promise.all([loadDatabases(), loadBackups()])
}

async function createBackup(dbName) {
  backingUp.value = dbName
  try {
    await api.post('/db/backup/create', {
      db_name: dbName,
      engine: activeEngine.value,
    })
    showNotif(t('db_backup.backup_created'))
    await loadBackups()
  } catch (err) {
    showNotif(err?.response?.data?.message || t('common.error'), 'error')
  } finally {
    backingUp.value = null
  }
}

async function restoreBackup(backup) {
  if (!confirm(t('db_backup.restore_confirm'))) return
  const id = backup.id || backup.filename
  restoring.value = id
  try {
    await api.post('/db/backup/restore', { backup_id: id })
    showNotif(t('common.success'))
  } catch (err) {
    showNotif(err?.response?.data?.message || t('common.error'), 'error')
  } finally {
    restoring.value = null
  }
}

async function downloadBackup(backup) {
  const backupId = backup.id || backup.filename || ''
  if (!backupId) return

  try {
    const res = await api.get('/db/backup/download', {
      params: { id: backupId },
      responseType: 'blob',
    })
    const blob = new Blob([res.data], { type: 'application/gzip' })
    const url = window.URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = backup.filename || `${backupId}.sql.gz`
    document.body.appendChild(link)
    link.click()
    link.remove()
    window.URL.revokeObjectURL(url)
  } catch (err) {
    showNotif(err?.response?.data?.message || t('common.error'), 'error')
  }
}

async function deleteBackup(backup) {
  if (!confirm(t('common.confirm_delete'))) return
  const id = backup.id || backup.filename
  try {
    await api.post('/db/backup/delete', { backup_id: id })
    showNotif(t('common.success'))
    await loadBackups()
  } catch (err) {
    showNotif(err?.response?.data?.message || t('common.error'), 'error')
  }
}

watch(activeEngine, () => {
  loadDatabases()
})

onMounted(loadAll)
</script>
