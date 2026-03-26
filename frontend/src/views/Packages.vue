<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-white">{{ t('packages.title') }}</h1>
        <p class="text-gray-400 mt-1">{{ t('packages.subtitle') }}</p>
      </div>
      <button class="btn-primary" @click="showAddModal = true">
        <Plus class="w-5 h-5" />
        {{ t('packages.add_new') }}
      </button>
    </div>

    <div v-if="loading" class="aura-card text-center py-12">
      <Loader2 class="w-8 h-8 text-brand-500 animate-spin mx-auto mb-3" />
      <p class="text-gray-400">{{ t('common.loading') }}</p>
    </div>

    <div v-else class="grid grid-cols-1 md:grid-cols-3 gap-6">
      <div v-for="pkg in packages" :key="pkg.id" class="aura-card">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-lg font-bold text-white">{{ pkg.name }}</h3>
          <span class="px-2 py-1 text-xs font-semibold bg-panel-dark rounded border border-panel-border whitespace-nowrap">
            {{ pkg.plan_type === 'reseller' ? t('packages.type_reseller') : t('packages.type_hosting') }}
          </span>
        </div>

        <ul class="space-y-3 mb-6">
          <li class="flex items-center text-sm text-gray-300">
            <HardDrive class="w-4 h-4 text-brand-500 mr-3" />
            <span class="font-medium text-white w-28">{{ t('packages.disk') }}:</span>
            {{ pkg.disk_gb > 0 ? pkg.disk_gb + ' GB' : t('packages.unlimited') }}
          </li>
          <li class="flex items-center text-sm text-gray-300">
            <Activity class="w-4 h-4 text-blue-500 mr-3" />
            <span class="font-medium text-white w-28">{{ t('packages.bandwidth') }}:</span>
            {{ pkg.bandwidth_gb > 0 ? pkg.bandwidth_gb + ' GB' : t('packages.unlimited') }}
          </li>
          <li class="flex items-center text-sm text-gray-300">
            <Globe class="w-4 h-4 text-purple-500 mr-3" />
            <span class="font-medium text-white w-28">{{ t('packages.domains') }}:</span>
            {{ pkg.domains > 0 ? pkg.domains : t('packages.unlimited') }}
          </li>
          <li class="flex items-center text-sm text-gray-300">
            <Database class="w-4 h-4 text-orange-500 mr-3" />
            <span class="font-medium text-white w-28">{{ t('packages.databases') }}:</span>
            {{ pkg.databases > 0 ? pkg.databases : t('packages.unlimited') }}
          </li>
          <li class="flex items-center text-sm text-gray-300">
            <Mail class="w-4 h-4 text-green-500 mr-3" />
            <span class="font-medium text-white w-28">{{ t('packages.emails') }}:</span>
            {{ pkg.emails > 0 ? pkg.emails : t('packages.unlimited') }}
          </li>
        </ul>

        <div class="flex items-center gap-3">
          <button class="btn-danger p-2" :title="t('common.delete')" @click="deletePackage(pkg)">
            <Trash2 class="w-4 h-4" />
          </button>
        </div>
      </div>
      <div v-if="packages.length === 0 && !loading" class="col-span-3 aura-card text-center py-12 text-gray-400">
        {{ t('common.no_data') }}
      </div>
    </div>

    <!-- Add Modal -->
    <Teleport to="body">
      <div v-if="showAddModal" class="fixed inset-0 bg-black/60 flex items-center justify-center z-50 p-4">
        <div class="bg-panel-card border border-panel-border rounded-2xl p-8 w-full max-w-md shadow-2xl">
          <h2 class="text-xl font-bold text-white mb-6">{{ t('packages.add_new') }}</h2>
          <div class="space-y-4">
            <div>
              <label class="block text-sm text-gray-400 mb-1">{{ t('packages.plan_name') }}</label>
              <input v-model="form.name" type="text" class="aura-input w-full" />
            </div>
            <div>
              <label class="block text-sm text-gray-400 mb-1">{{ t('packages.plan_type') }}</label>
              <select v-model="form.plan_type" class="aura-input w-full">
                <option value="hosting">{{ t('packages.type_hosting') }}</option>
                <option value="reseller">{{ t('packages.type_reseller') }}</option>
              </select>
            </div>
            <div class="grid grid-cols-2 gap-3">
              <div>
                <label class="block text-sm text-gray-400 mb-1">{{ t('packages.disk') }} (GB)</label>
                <input v-model.number="form.disk_gb" type="number" class="aura-input w-full" placeholder="0=Sınırsız" />
              </div>
              <div>
                <label class="block text-sm text-gray-400 mb-1">{{ t('packages.bandwidth') }} (GB)</label>
                <input v-model.number="form.bandwidth_gb" type="number" class="aura-input w-full" placeholder="0=Sınırsız" />
              </div>
              <div>
                <label class="block text-sm text-gray-400 mb-1">{{ t('packages.domains') }}</label>
                <input v-model.number="form.domains" type="number" class="aura-input w-full" placeholder="0=Sınırsız" />
              </div>
              <div>
                <label class="block text-sm text-gray-400 mb-1">{{ t('packages.databases') }}</label>
                <input v-model.number="form.databases" type="number" class="aura-input w-full" placeholder="0=Sınırsız" />
              </div>
            </div>
          </div>
          <div class="flex gap-3 mt-8">
            <button class="btn-secondary flex-1" @click="showAddModal = false">{{ t('common.cancel') }}</button>
            <button class="btn-primary flex-1" :disabled="addLoading" @click="addPackage">
              <Loader2 v-if="addLoading" class="w-4 h-4 animate-spin mr-2 inline" />
              {{ t('common.create') }}
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Plus, HardDrive, Activity, Globe, Database, Mail, Trash2, Loader2 } from 'lucide-vue-next'
import api from '../services/api'

const { t } = useI18n()
const packages = ref([])
const loading = ref(true)
const showAddModal = ref(false)
const addLoading = ref(false)
const form = ref({ name: '', plan_type: 'hosting', disk_gb: 10, bandwidth_gb: 0, domains: 1, databases: 3, emails: 10 })

async function loadPackages() {
  loading.value = true
  try {
    const res = await api.get('/packages/list')
    packages.value = res.data.data || []
  } finally {
    loading.value = false
  }
}

async function addPackage() {
  if (!form.value.name) return
  addLoading.value = true
  try {
    await api.post('/packages/create', form.value)
    showAddModal.value = false
    await loadPackages()
  } finally {
    addLoading.value = false
  }
}

async function deletePackage(pkg) {
  if (!confirm(t('common.confirm_delete'))) return
  await api.post('/packages/delete', { id: pkg.id })
  await loadPackages()
}

onMounted(loadPackages)
</script>
