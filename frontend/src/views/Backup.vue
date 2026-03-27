<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between gap-3">
      <div>
        <h1 class="text-2xl font-bold text-white">{{ t('backup_center.title') }}</h1>
        <p class="mt-1 text-gray-400">{{ t('backup_center.subtitle') }}</p>
      </div>
      <button class="btn-secondary" @click="loadAll">{{ t('backup_center.refresh') }}</button>
    </div>

    <div v-if="error" class="aura-card border-red-500/30 bg-red-500/5 text-red-400">{{ error }}</div>
    <div v-if="success" class="aura-card border-green-500/30 bg-green-500/5 text-green-300">{{ success }}</div>

    <div class="grid grid-cols-1 gap-4 lg:grid-cols-2">
      <div class="aura-card space-y-3">
        <h2 class="font-semibold text-white">{{ t('backup_center.run_title') }}</h2>
        <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
          <select v-model="runForm.domain" class="aura-input" @change="onDomainChange(runForm)">
            <option disabled value="">{{ t('backup_center.select_domain') }}</option>
            <option v-for="domainName in domains" :key="domainName" :value="domainName">{{ domainName }}</option>
          </select>
          <select v-model="runForm.destination_id" class="aura-input">
            <option disabled value="">{{ t('backup_center.select_destination') }}</option>
            <option v-for="destination in destinations" :key="destination.id" :value="destination.id">{{ destination.name }}</option>
          </select>
          <input
            v-model="runForm.backup_path"
            class="aura-input md:col-span-2"
            :placeholder="t('backup_center.backup_path_placeholder')"
          />
          <label class="inline-flex items-center gap-2 text-sm text-gray-300 md:col-span-2">
            <input v-model="runForm.incremental" type="checkbox" class="h-4 w-4" />
            {{ t('backup_center.incremental_backup') }}
          </label>
        </div>
        <div class="flex gap-2">
          <button class="btn-primary" @click="runBackup">{{ t('backup_center.run_backup') }}</button>
          <button class="btn-secondary" @click="loadSnapshots">{{ t('backup_center.load_snapshots') }}</button>
        </div>
      </div>

      <div class="aura-card space-y-3">
        <h2 class="font-semibold text-white">{{ t('backup_center.restore_title') }}</h2>
        <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
          <select v-model="restoreForm.domain" class="aura-input" @change="onDomainChange(restoreForm)">
            <option disabled value="">{{ t('backup_center.select_domain') }}</option>
            <option v-for="domainName in domains" :key="domainName" :value="domainName">{{ domainName }}</option>
          </select>
          <select v-model="restoreForm.destination_id" class="aura-input">
            <option disabled value="">{{ t('backup_center.select_destination') }}</option>
            <option v-for="destination in destinations" :key="destination.id" :value="destination.id">{{ destination.name }}</option>
          </select>
          <input
            v-model="restoreForm.backup_path"
            class="aura-input md:col-span-2"
            :placeholder="t('backup_center.backup_path_placeholder')"
          />
          <input
            v-model="restoreForm.snapshot_id"
            class="aura-input md:col-span-2"
            :placeholder="t('backup_center.snapshot_placeholder')"
          />
        </div>
        <button class="btn-primary" @click="restoreBackup">{{ t('backup_center.restore_backup') }}</button>
      </div>
    </div>

    <div class="aura-card space-y-4">
      <h2 class="font-semibold text-white">{{ t('backup_center.destinations_title') }}</h2>
      <div class="grid grid-cols-1 gap-3 md:grid-cols-5">
        <input
          v-model="destinationForm.name"
          class="aura-input"
          :placeholder="t('backup_center.form.destination_name')"
        />
        <input
          v-model="destinationForm.remote_repo"
          class="aura-input md:col-span-2"
          :placeholder="t('backup_center.form.destination_repo')"
        />
        <input
          v-model="destinationForm.password"
          type="password"
          class="aura-input"
          :placeholder="t('backup_center.form.destination_password')"
        />
        <label class="inline-flex items-center gap-2 text-sm text-gray-300">
          <input v-model="destinationForm.enabled" type="checkbox" class="h-4 w-4" />
          {{ t('backup_center.enabled') }}
        </label>
      </div>
      <div class="flex gap-2">
        <button class="btn-primary" @click="saveDestination">{{ t('backup_center.save_destination') }}</button>
        <button class="btn-secondary" @click="resetDestinationForm">{{ t('backup_center.reset_destination') }}</button>
      </div>
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-panel-border text-gray-400">
              <th class="px-2 py-2 text-left">{{ t('backup_center.table.name') }}</th>
              <th class="px-2 py-2 text-left">{{ t('backup_center.table.repository') }}</th>
              <th class="px-2 py-2 text-left">{{ t('backup_center.table.state') }}</th>
              <th class="px-2 py-2 text-right">{{ t('backup_center.table.action') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="destination in destinations" :key="destination.id" class="border-b border-panel-border/40">
              <td class="px-2 py-2 text-white">{{ destination.name }}</td>
              <td class="break-all px-2 py-2 font-mono text-xs text-gray-300">{{ destination.remote_repo }}</td>
              <td class="px-2 py-2" :class="destination.enabled ? 'text-green-400' : 'text-yellow-400'">
                {{ destination.enabled ? t('backup_center.status.enabled') : t('backup_center.status.disabled') }}
              </td>
              <td class="px-2 py-2 text-right">
                <div class="flex justify-end gap-2">
                  <button class="btn-secondary px-2 py-1 text-xs" @click="editDestination(destination)">{{ t('common.edit') }}</button>
                  <button class="btn-danger px-2 py-1 text-xs" @click="deleteDestination(destination.id)">{{ t('common.delete') }}</button>
                </div>
              </td>
            </tr>
            <tr v-if="destinations.length === 0">
              <td colspan="4" class="py-6 text-center text-gray-500">{{ t('backup_center.empty.destinations') }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div class="aura-card space-y-4">
      <h2 class="font-semibold text-white">{{ t('backup_center.schedules_title') }}</h2>
      <div class="grid grid-cols-1 gap-3 md:grid-cols-6">
        <select v-model="scheduleForm.domain" class="aura-input" @change="onDomainChange(scheduleForm)">
          <option disabled value="">{{ t('backup_center.select_domain') }}</option>
          <option v-for="domainName in domains" :key="domainName" :value="domainName">{{ domainName }}</option>
        </select>
        <select v-model="scheduleForm.destination_id" class="aura-input">
          <option disabled value="">{{ t('backup_center.destination_short') }}</option>
          <option v-for="destination in destinations" :key="destination.id" :value="destination.id">{{ destination.name }}</option>
        </select>
        <input
          v-model="scheduleForm.backup_path"
          class="aura-input md:col-span-2"
          :placeholder="t('backup_center.backup_path_placeholder')"
        />
        <input
          v-model="scheduleForm.cron"
          class="aura-input"
          :placeholder="t('backup_center.form.cron_placeholder')"
        />
        <label class="inline-flex items-center gap-2 text-sm text-gray-300">
          <input v-model="scheduleForm.enabled" type="checkbox" class="h-4 w-4" />
          {{ t('backup_center.enabled') }}
        </label>
        <label class="inline-flex items-center gap-2 text-sm text-gray-300 md:col-span-2">
          <input v-model="scheduleForm.incremental" type="checkbox" class="h-4 w-4" />
          {{ t('backup_center.incremental') }}
        </label>
      </div>
      <div class="flex gap-2">
        <button class="btn-primary" @click="saveSchedule">{{ t('backup_center.save_schedule') }}</button>
        <button class="btn-secondary" @click="resetScheduleForm">{{ t('backup_center.reset_schedule') }}</button>
      </div>
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-panel-border text-gray-400">
              <th class="px-2 py-2 text-left">{{ t('backup_center.table.domain') }}</th>
              <th class="px-2 py-2 text-left">{{ t('backup_center.table.cron') }}</th>
              <th class="px-2 py-2 text-left">{{ t('backup_center.table.path') }}</th>
              <th class="px-2 py-2 text-left">{{ t('backup_center.table.destination') }}</th>
              <th class="px-2 py-2 text-right">{{ t('backup_center.table.action') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="schedule in schedules" :key="schedule.id" class="border-b border-panel-border/40">
              <td class="px-2 py-2 text-white">{{ schedule.domain }}</td>
              <td class="px-2 py-2 font-mono text-gray-300">{{ schedule.cron }}</td>
              <td class="break-all px-2 py-2 font-mono text-xs text-gray-400">{{ schedule.backup_path }}</td>
              <td class="px-2 py-2 text-gray-300">{{ destinationName(schedule.destination_id) }}</td>
              <td class="px-2 py-2 text-right">
                <div class="flex justify-end gap-2">
                  <button class="btn-secondary px-2 py-1 text-xs" @click="editSchedule(schedule)">{{ t('common.edit') }}</button>
                  <button class="btn-danger px-2 py-1 text-xs" @click="deleteSchedule(schedule.id)">{{ t('common.delete') }}</button>
                </div>
              </td>
            </tr>
            <tr v-if="schedules.length === 0">
              <td colspan="5" class="py-6 text-center text-gray-500">{{ t('backup_center.empty.schedules') }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div class="aura-card space-y-3">
      <h2 class="font-semibold text-white">{{ t('backup_center.snapshots_title') }}</h2>
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-panel-border text-gray-400">
              <th class="px-2 py-2 text-left">{{ t('backup_center.table.id') }}</th>
              <th class="px-2 py-2 text-left">{{ t('backup_center.table.time') }}</th>
              <th class="px-2 py-2 text-left">{{ t('backup_center.table.hostname') }}</th>
              <th class="px-2 py-2 text-left">{{ t('backup_center.table.tags') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="snapshot in snapshots" :key="snapshot.id" class="border-b border-panel-border/40">
              <td class="px-2 py-2 font-mono text-white">{{ snapshot.short_id || snapshot.id }}</td>
              <td class="px-2 py-2 text-gray-300">{{ snapshot.time }}</td>
              <td class="px-2 py-2 text-gray-300">{{ snapshot.hostname || '-' }}</td>
              <td class="px-2 py-2 text-gray-400">{{ (snapshot.tags || []).join(', ') }}</td>
            </tr>
            <tr v-if="snapshots.length === 0">
              <td colspan="4" class="py-6 text-center text-gray-500">{{ t('backup_center.empty.snapshots') }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '../services/api'

const { t } = useI18n({ useScope: 'global' })

const error = ref('')
const success = ref('')
const sites = ref([])
const destinations = ref([])
const schedules = ref([])
const snapshots = ref([])

const destinationForm = ref({
  id: '',
  name: '',
  remote_repo: '',
  password: '',
  enabled: true,
})

const scheduleForm = ref({
  id: '',
  domain: '',
  destination_id: '',
  backup_path: '',
  cron: '0 3 * * *',
  incremental: false,
  enabled: true,
})

const runForm = ref({
  domain: '',
  destination_id: '',
  backup_path: '',
  incremental: false,
})

const restoreForm = ref({
  domain: '',
  destination_id: '',
  backup_path: '',
  snapshot_id: '',
})

const domains = computed(() => (sites.value || []).map(site => site.domain).filter(Boolean))

function apiErrorMessage(err, fallbackKey) {
  return err?.response?.data?.message || err?.message || t(fallbackKey)
}

function onDomainChange(target) {
  if (!target.domain || target.backup_path) return
  target.backup_path = `/home/${target.domain}/public_html`
}

function destinationName(id) {
  return destinations.value.find(item => item.id === id)?.name || id
}

function destinationById(id) {
  return destinations.value.find(item => item.id === id)
}

function backupPayloadFrom(form) {
  const destination = destinationById(form.destination_id)
  if (!destination) {
    throw new Error(t('backup_center.messages.destination_not_selected'))
  }
  return {
    domain: form.domain,
    backup_path: form.backup_path,
    remote_repo: destination.remote_repo,
    password: destination.password,
    incremental: !!form.incremental,
  }
}

async function loadSites() {
  const res = await api.get('/vhost/list')
  sites.value = res.data?.data || []
}

async function loadDestinations() {
  const res = await api.get('/backup/destinations')
  destinations.value = res.data?.data || []
}

async function loadSchedules() {
  const res = await api.get('/backup/schedules')
  schedules.value = res.data?.data || []
}

async function loadAll() {
  error.value = ''
  success.value = ''
  try {
    await Promise.all([loadSites(), loadDestinations(), loadSchedules()])
    if (!runForm.value.domain && domains.value.length > 0) {
      runForm.value.domain = domains.value[0]
      runForm.value.backup_path = `/home/${runForm.value.domain}/public_html`
      restoreForm.value.domain = domains.value[0]
      restoreForm.value.backup_path = `/home/${restoreForm.value.domain}/public_html`
      scheduleForm.value.domain = domains.value[0]
      scheduleForm.value.backup_path = `/home/${scheduleForm.value.domain}/public_html`
    }
    if (!runForm.value.destination_id && destinations.value.length > 0) {
      runForm.value.destination_id = destinations.value[0].id
      restoreForm.value.destination_id = destinations.value[0].id
      scheduleForm.value.destination_id = destinations.value[0].id
    }
  } catch (err) {
    error.value = apiErrorMessage(err, 'backup_center.messages.load_failed')
  }
}

async function saveDestination() {
  error.value = ''
  success.value = ''
  try {
    const res = await api.post('/backup/destinations', destinationForm.value)
    success.value = t('backup_center.messages.destination_saved', {
      name: res.data?.data?.name || destinationForm.value.name,
    })
    resetDestinationForm()
    await loadDestinations()
  } catch (err) {
    error.value = apiErrorMessage(err, 'backup_center.messages.destination_save_failed')
  }
}

function editDestination(item) {
  destinationForm.value = { ...item }
}

function resetDestinationForm() {
  destinationForm.value = {
    id: '',
    name: '',
    remote_repo: '',
    password: '',
    enabled: true,
  }
}

async function deleteDestination(id) {
  if (!window.confirm(t('backup_center.messages.destination_delete_confirm'))) return
  error.value = ''
  success.value = ''
  try {
    await api.delete('/backup/destinations', { params: { id } })
    success.value = t('backup_center.messages.destination_deleted')
    await loadAll()
  } catch (err) {
    error.value = apiErrorMessage(err, 'backup_center.messages.destination_delete_failed')
  }
}

async function saveSchedule() {
  error.value = ''
  success.value = ''
  try {
    const res = await api.post('/backup/schedules', scheduleForm.value)
    success.value = t('backup_center.messages.schedule_saved', {
      cron: res.data?.data?.cron || scheduleForm.value.cron,
    })
    resetScheduleForm()
    await loadSchedules()
  } catch (err) {
    error.value = apiErrorMessage(err, 'backup_center.messages.schedule_save_failed')
  }
}

function editSchedule(item) {
  scheduleForm.value = { ...item }
}

function resetScheduleForm() {
  scheduleForm.value = {
    id: '',
    domain: domains.value[0] || '',
    destination_id: destinations.value[0]?.id || '',
    backup_path: domains.value[0] ? `/home/${domains.value[0]}/public_html` : '',
    cron: '0 3 * * *',
    incremental: false,
    enabled: true,
  }
}

async function deleteSchedule(id) {
  if (!window.confirm(t('backup_center.messages.schedule_delete_confirm'))) return
  error.value = ''
  success.value = ''
  try {
    await api.delete('/backup/schedules', { params: { id } })
    success.value = t('backup_center.messages.schedule_deleted')
    await loadSchedules()
  } catch (err) {
    error.value = apiErrorMessage(err, 'backup_center.messages.schedule_delete_failed')
  }
}

async function runBackup() {
  error.value = ''
  success.value = ''
  try {
    const payload = backupPayloadFrom(runForm.value)
    const res = await api.post('/backup/create', payload)
    success.value = res.data?.message || t('backup_center.messages.backup_started')
    if (res.data?.snapshot_id) {
      restoreForm.value.snapshot_id = res.data.snapshot_id
    }
    await loadSnapshots()
  } catch (err) {
    error.value = apiErrorMessage(err, 'backup_center.messages.backup_failed')
  }
}

async function loadSnapshots() {
  error.value = ''
  try {
    const payload = backupPayloadFrom(runForm.value)
    const res = await api.post('/backup/snapshots', payload)
    snapshots.value = Array.isArray(res.data?.data) ? res.data.data : []
  } catch (err) {
    error.value = apiErrorMessage(err, 'backup_center.messages.snapshots_failed')
    snapshots.value = []
  }
}

async function restoreBackup() {
  error.value = ''
  success.value = ''
  try {
    const destination = destinationById(restoreForm.value.destination_id)
    if (!destination) {
      throw new Error(t('backup_center.messages.restore_destination_required'))
    }
    const payload = {
      domain: restoreForm.value.domain,
      backup_path: restoreForm.value.backup_path,
      remote_repo: destination.remote_repo,
      password: destination.password,
      snapshot_id: restoreForm.value.snapshot_id,
    }
    const res = await api.post('/backup/restore', payload)
    success.value = res.data?.message || t('backup_center.messages.restore_started')
  } catch (err) {
    error.value = apiErrorMessage(err, 'backup_center.messages.restore_failed')
  }
}

onMounted(loadAll)
</script>
