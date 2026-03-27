<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-white">{{ t('reseller_acl.title') }}</h1>
        <p class="mt-1 text-gray-400">{{ t('reseller_acl.subtitle') }}</p>
      </div>
      <button class="btn-secondary" @click="loadAll">{{ t('reseller_acl.refresh') }}</button>
    </div>

    <div v-if="error" class="aura-card border-red-500/30 bg-red-500/5 text-red-400">{{ error }}</div>
    <div v-if="success" class="aura-card border-green-500/30 bg-green-500/5 text-green-300">{{ success }}</div>

    <div class="border-b border-panel-border">
      <nav class="flex gap-5">
        <button @click="tab = 'quotas'" :class="tabClass('quotas')">{{ t('reseller_acl.tabs.quotas') }}</button>
        <button @click="tab = 'whitelabel'" :class="tabClass('whitelabel')">{{ t('reseller_acl.tabs.whitelabel') }}</button>
        <button @click="tab = 'policies'" :class="tabClass('policies')">{{ t('reseller_acl.tabs.policies') }}</button>
        <button @click="tab = 'assignments'" :class="tabClass('assignments')">{{ t('reseller_acl.tabs.assignments') }}</button>
      </nav>
    </div>

    <div v-if="tab === 'quotas'" class="aura-card space-y-4">
      <h2 class="font-semibold text-white">{{ t('reseller_acl.quotas.title') }}</h2>
      <div class="grid grid-cols-1 gap-3 md:grid-cols-6">
        <select v-model="quotaForm.username" class="aura-input">
          <option disabled value="">{{ t('reseller_acl.quotas.user') }}</option>
          <option v-for="user in users" :key="user.username" :value="user.username">{{ user.username }}</option>
        </select>
        <input v-model="quotaForm.plan" class="aura-input" :placeholder="t('reseller_acl.quotas.plan')" />
        <input v-model.number="quotaForm.disk_gb" type="number" class="aura-input" :placeholder="t('reseller_acl.quotas.disk')" />
        <input v-model.number="quotaForm.bandwidth_gb" type="number" class="aura-input" :placeholder="t('reseller_acl.quotas.bandwidth')" />
        <input v-model.number="quotaForm.max_sites" type="number" class="aura-input" :placeholder="t('reseller_acl.quotas.max_sites')" />
        <button class="btn-primary" @click="saveQuota">{{ t('reseller_acl.quotas.save') }}</button>
      </div>
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-panel-border text-gray-400">
              <th class="px-2 py-2 text-left">{{ t('reseller_acl.quotas.user') }}</th>
              <th class="px-2 py-2 text-left">{{ t('reseller_acl.quotas.plan') }}</th>
              <th class="px-2 py-2 text-left">{{ t('reseller_acl.quotas.disk') }}</th>
              <th class="px-2 py-2 text-left">{{ t('reseller_acl.quotas.bandwidth') }}</th>
              <th class="px-2 py-2 text-left">{{ t('reseller_acl.quotas.max_sites') }}</th>
              <th class="px-2 py-2 text-right">{{ t('reseller_acl.quotas.action') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="quota in quotas" :key="quota.username" class="border-b border-panel-border/40">
              <td class="px-2 py-2 text-white">{{ quota.username }}</td>
              <td class="px-2 py-2 text-gray-300">{{ quota.plan }}</td>
              <td class="px-2 py-2 text-gray-300">{{ quota.disk_gb }} GB</td>
              <td class="px-2 py-2 text-gray-300">{{ quota.bandwidth_gb }} GB</td>
              <td class="px-2 py-2 text-gray-300">{{ quota.max_sites }}</td>
              <td class="px-2 py-2 text-right">
                <button class="btn-secondary px-2 py-1 text-xs" @click="quotaForm = { ...quota }">{{ t('common.edit') }}</button>
              </td>
            </tr>
            <tr v-if="quotas.length === 0">
              <td colspan="6" class="py-6 text-center text-gray-500">{{ t('reseller_acl.quotas.empty') }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div v-if="tab === 'whitelabel'" class="aura-card space-y-4">
      <h2 class="font-semibold text-white">{{ t('reseller_acl.whitelabel.title') }}</h2>
      <div class="grid grid-cols-1 gap-3 md:grid-cols-4">
        <select v-model="wlForm.username" class="aura-input">
          <option disabled value="">{{ t('reseller_acl.whitelabel.user') }}</option>
          <option v-for="user in users" :key="user.username" :value="user.username">{{ user.username }}</option>
        </select>
        <input v-model="wlForm.panel_name" class="aura-input" :placeholder="t('reseller_acl.whitelabel.panel_name')" />
        <input v-model="wlForm.logo_url" class="aura-input md:col-span-2" :placeholder="t('reseller_acl.whitelabel.logo_url')" />
      </div>
      <button class="btn-primary" @click="saveWhiteLabel">{{ t('reseller_acl.whitelabel.save') }}</button>
      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-panel-border text-gray-400">
              <th class="px-2 py-2 text-left">{{ t('reseller_acl.whitelabel.user') }}</th>
              <th class="px-2 py-2 text-left">{{ t('reseller_acl.whitelabel.panel_name') }}</th>
              <th class="px-2 py-2 text-left">{{ t('reseller_acl.whitelabel.logo_url') }}</th>
              <th class="px-2 py-2 text-right">{{ t('reseller_acl.whitelabel.action') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="label in whiteLabels" :key="label.username" class="border-b border-panel-border/40">
              <td class="px-2 py-2 text-white">{{ label.username }}</td>
              <td class="px-2 py-2 text-gray-300">{{ label.panel_name }}</td>
              <td class="break-all px-2 py-2 font-mono text-xs text-gray-400">{{ label.logo_url || '-' }}</td>
              <td class="px-2 py-2 text-right">
                <button class="btn-secondary px-2 py-1 text-xs" @click="wlForm = { ...label }">{{ t('common.edit') }}</button>
              </td>
            </tr>
            <tr v-if="whiteLabels.length === 0">
              <td colspan="4" class="py-6 text-center text-gray-500">{{ t('reseller_acl.whitelabel.empty') }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div v-if="tab === 'policies'" class="aura-card space-y-4">
      <h2 class="font-semibold text-white">{{ t('reseller_acl.policies.title') }}</h2>
      <div class="grid grid-cols-1 gap-3 md:grid-cols-4">
        <input v-model="policyForm.name" class="aura-input" :placeholder="t('reseller_acl.policies.name')" />
        <input v-model="policyForm.description" class="aura-input" :placeholder="t('reseller_acl.policies.description')" />
        <input v-model="policyPermissionsRaw" class="aura-input md:col-span-2" :placeholder="t('reseller_acl.policies.permissions')" />
      </div>
      <div class="flex gap-2">
        <button class="btn-primary" @click="savePolicy">{{ t('reseller_acl.policies.save') }}</button>
        <button class="btn-secondary" @click="resetPolicyForm">{{ t('reseller_acl.policies.reset') }}</button>
      </div>
      <div class="space-y-2">
        <div v-for="policy in policies" :key="policy.id" class="aura-card border border-panel-border/60">
          <div class="flex items-center justify-between gap-3">
            <div>
              <p class="font-semibold text-white">{{ policy.name }}</p>
              <p class="text-xs text-gray-400">{{ policy.description }}</p>
              <p class="mt-1 font-mono text-xs text-gray-500">{{ (policy.permissions || []).join(', ') }}</p>
            </div>
            <div class="flex gap-2">
              <button class="btn-secondary px-2 py-1 text-xs" @click="editPolicy(policy)">{{ t('common.edit') }}</button>
              <button class="btn-danger px-2 py-1 text-xs" @click="deletePolicy(policy.id)">{{ t('common.delete') }}</button>
            </div>
          </div>
        </div>
        <p v-if="policies.length === 0" class="text-sm text-gray-500">{{ t('reseller_acl.policies.empty') }}</p>
      </div>
    </div>

    <div v-if="tab === 'assignments'" class="aura-card space-y-4">
      <h2 class="font-semibold text-white">{{ t('reseller_acl.assignments.title') }}</h2>
      <div class="grid grid-cols-1 gap-3 md:grid-cols-3">
        <select v-model="assignmentForm.username" class="aura-input">
          <option disabled value="">{{ t('reseller_acl.assignments.user') }}</option>
          <option v-for="user in users" :key="user.username" :value="user.username">{{ user.username }}</option>
        </select>
        <select v-model="assignmentForm.policy_id" class="aura-input">
          <option disabled value="">{{ t('reseller_acl.assignments.policy') }}</option>
          <option v-for="policy in policies" :key="policy.id" :value="policy.id">{{ policy.name }}</option>
        </select>
        <button class="btn-primary" @click="saveAssignment">{{ t('reseller_acl.assignments.assign') }}</button>
      </div>

      <div class="overflow-x-auto">
        <table class="w-full text-sm">
          <thead>
            <tr class="border-b border-panel-border text-gray-400">
              <th class="px-2 py-2 text-left">{{ t('reseller_acl.assignments.user') }}</th>
              <th class="px-2 py-2 text-left">{{ t('reseller_acl.assignments.policy') }}</th>
              <th class="px-2 py-2 text-left">{{ t('reseller_acl.assignments.effective_permissions') }}</th>
              <th class="px-2 py-2 text-right">{{ t('reseller_acl.assignments.action') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="assignment in assignments" :key="assignment.username" class="border-b border-panel-border/40">
              <td class="px-2 py-2 text-white">{{ assignment.username }}</td>
              <td class="px-2 py-2 text-gray-300">{{ policyName(assignment.policy_id) }}</td>
              <td class="break-all px-2 py-2 font-mono text-xs text-gray-400">{{ (effectiveMap[assignment.username] || []).join(', ') }}</td>
              <td class="px-2 py-2 text-right">
                <button class="btn-danger px-2 py-1 text-xs" @click="deleteAssignment(assignment.username)">{{ t('common.delete') }}</button>
              </td>
            </tr>
            <tr v-if="assignments.length === 0">
              <td colspan="4" class="py-6 text-center text-gray-500">{{ t('reseller_acl.assignments.empty') }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '../services/api'

const { t } = useI18n({ useScope: 'global' })

const tab = ref('quotas')
const error = ref('')
const success = ref('')

const users = ref([])
const quotas = ref([])
const whiteLabels = ref([])
const policies = ref([])
const assignments = ref([])
const effectiveMap = ref({})

const quotaForm = ref({ username: '', plan: '', disk_gb: 10, bandwidth_gb: 0, max_sites: 1, updated_at: 0 })
const wlForm = ref({ username: '', panel_name: '', logo_url: '', updated_at: 0 })
const policyForm = ref({ id: '', name: '', description: '', permissions: [], updated_at: 0 })
const policyPermissionsRaw = ref('')
const assignmentForm = ref({ username: '', policy_id: '', updated_at: 0 })

function tabClass(key) {
  return [
    'pb-3 text-sm font-medium transition',
    tab.value === key ? 'text-brand-400 border-b-2 border-brand-400' : 'text-gray-400 hover:text-white',
  ]
}

function apiErrorMessage(err, fallbackKey) {
  return err?.response?.data?.message || err?.message || t(fallbackKey)
}

function policyName(id) {
  return policies.value.find(policy => policy.id === id)?.name || id
}

function resetPolicyForm() {
  policyForm.value = { id: '', name: '', description: '', permissions: [], updated_at: 0 }
  policyPermissionsRaw.value = ''
}

function editPolicy(item) {
  policyForm.value = { ...item }
  policyPermissionsRaw.value = (item.permissions || []).join(', ')
}

async function loadUsers() {
  const res = await api.get('/users/list')
  users.value = res.data?.data || []
}

async function loadQuotas() {
  const res = await api.get('/reseller/quotas')
  quotas.value = res.data?.data || []
}

async function loadWhiteLabels() {
  const res = await api.get('/reseller/whitelabel')
  whiteLabels.value = res.data?.data || []
}

async function loadPolicies() {
  const res = await api.get('/acl/policies')
  policies.value = res.data?.data || []
}

async function loadAssignments() {
  const res = await api.get('/acl/assignments')
  assignments.value = res.data?.data || []
  const map = {}
  await Promise.all(
    (assignments.value || []).map(async item => {
      try {
        const perms = await api.get('/acl/effective', { params: { username: item.username } })
        map[item.username] = perms.data?.data || []
      } catch {
        map[item.username] = []
      }
    }),
  )
  effectiveMap.value = map
}

async function loadAll() {
  error.value = ''
  success.value = ''
  try {
    await Promise.all([loadUsers(), loadQuotas(), loadWhiteLabels(), loadPolicies(), loadAssignments()])
  } catch (err) {
    error.value = apiErrorMessage(err, 'reseller_acl.messages.load_failed')
  }
}

async function saveQuota() {
  error.value = ''
  success.value = ''
  try {
    await api.post('/reseller/quotas', quotaForm.value)
    success.value = t('reseller_acl.messages.quota_saved')
    await loadQuotas()
  } catch (err) {
    error.value = apiErrorMessage(err, 'reseller_acl.messages.quota_save_failed')
  }
}

async function saveWhiteLabel() {
  error.value = ''
  success.value = ''
  try {
    await api.post('/reseller/whitelabel', wlForm.value)
    success.value = t('reseller_acl.messages.whitelabel_saved')
    await loadWhiteLabels()
  } catch (err) {
    error.value = apiErrorMessage(err, 'reseller_acl.messages.whitelabel_failed')
  }
}

async function savePolicy() {
  error.value = ''
  success.value = ''
  try {
    const payload = {
      ...policyForm.value,
      permissions: policyPermissionsRaw.value.split(',').map(item => item.trim()).filter(Boolean),
    }
    await api.post('/acl/policies', payload)
    success.value = t('reseller_acl.messages.policy_saved')
    resetPolicyForm()
    await loadPolicies()
  } catch (err) {
    error.value = apiErrorMessage(err, 'reseller_acl.messages.policy_save_failed')
  }
}

async function deletePolicy(id) {
  if (!window.confirm(t('reseller_acl.messages.policy_delete_confirm'))) return
  error.value = ''
  success.value = ''
  try {
    await api.delete('/acl/policies', { params: { id } })
    success.value = t('reseller_acl.messages.policy_deleted')
    await loadAll()
  } catch (err) {
    error.value = apiErrorMessage(err, 'reseller_acl.messages.policy_delete_failed')
  }
}

async function saveAssignment() {
  error.value = ''
  success.value = ''
  try {
    await api.post('/acl/assignments', assignmentForm.value)
    success.value = t('reseller_acl.messages.assignment_saved')
    await loadAssignments()
  } catch (err) {
    error.value = apiErrorMessage(err, 'reseller_acl.messages.assignment_save_failed')
  }
}

async function deleteAssignment(username) {
  if (!window.confirm(t('reseller_acl.messages.assignment_delete_confirm'))) return
  error.value = ''
  success.value = ''
  try {
    await api.delete('/acl/assignments', { params: { username } })
    success.value = t('reseller_acl.messages.assignment_deleted')
    await loadAssignments()
  } catch (err) {
    error.value = apiErrorMessage(err, 'reseller_acl.messages.assignment_delete_failed')
  }
}

onMounted(loadAll)
</script>
