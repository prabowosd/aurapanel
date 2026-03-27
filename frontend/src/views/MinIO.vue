<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-white">{{ t('minio.title') }}</h1>
        <p class="text-gray-400 mt-1">{{ t('minio.subtitle') }}</p>
      </div>
      <button class="btn-secondary" @click="loadBuckets">{{ t('minio.refresh') }}</button>
    </div>

    <div class="aura-card space-y-3">
      <h2 class="text-lg font-bold text-white">{{ t('minio.create_bucket') }}</h2>
      <div class="flex gap-3">
        <input v-model="bucketName" class="aura-input" :placeholder="t('minio.bucket_placeholder')" />
        <button class="btn-primary" @click="createBucket">{{ t('minio.create') }}</button>
      </div>
    </div>

    <div class="aura-card space-y-3">
      <h2 class="text-lg font-bold text-white">{{ t('minio.credentials') }}</h2>
      <div class="flex gap-3">
        <input v-model="credUser" class="aura-input" :placeholder="t('minio.user_placeholder')" />
        <button class="btn-primary" @click="createCredentials">{{ t('minio.generate') }}</button>
      </div>
      <div v-if="creds" class="bg-panel-dark border border-panel-border rounded-lg p-3 text-sm">
        <p><strong>{{ t('minio.access_key') }}:</strong> {{ creds.access_key }}</p>
        <p><strong>{{ t('minio.secret_key') }}:</strong> {{ creds.secret_key }}</p>
      </div>
    </div>

    <div class="aura-card">
      <h2 class="text-lg font-bold text-white mb-3">{{ t('minio.bucket_list') }}</h2>
      <div class="space-y-2">
        <div v-for="bucket in buckets" :key="bucket" class="bg-panel-dark border border-panel-border rounded-lg p-3 text-white">
          {{ bucket }}
        </div>
        <div v-if="buckets.length === 0" class="text-gray-400 text-sm">{{ t('minio.empty') }}</div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '../services/api'

const { t } = useI18n({ useScope: 'global' })

const bucketName = ref('')
const credUser = ref('admin')
const buckets = ref([])
const creds = ref(null)

async function loadBuckets() {
  const res = await api.get('/storage/minio/buckets')
  buckets.value = res.data.data || []
}

async function createBucket() {
  if (!bucketName.value) return
  await api.post('/storage/minio/buckets', { bucket_name: bucketName.value })
  bucketName.value = ''
  await loadBuckets()
}

async function createCredentials() {
  const res = await api.post('/storage/minio/credentials', { user: credUser.value })
  creds.value = res.data.data
}

onMounted(loadBuckets)
</script>
