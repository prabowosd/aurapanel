<template>
  <div class="space-y-4">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-white flex items-center gap-3">
          <FolderOpen class="w-7 h-7 text-orange-400" />
          File Manager
        </h1>
        <p class="text-gray-400 mt-1">Sunucu dosyalarını yönetin</p>
      </div>
      <div class="flex gap-2">
        <button @click="showUploadModal = true" class="px-4 py-2 bg-gradient-to-r from-orange-600 to-amber-600 text-white rounded-lg text-sm hover:from-orange-700 hover:to-amber-700 transition">📤 Yükle</button>
        <button @click="showNewModal = true" class="px-4 py-2 bg-panel-hover text-gray-300 rounded-lg text-sm hover:bg-gray-600 transition">📄 Yeni Dosya</button>
        <button @click="showNewFolderModal = true" class="px-4 py-2 bg-panel-hover text-gray-300 rounded-lg text-sm hover:bg-gray-600 transition">📁 Yeni Klasör</button>
      </div>
    </div>

    <!-- Breadcrumb -->
    <div class="flex items-center gap-1 text-sm bg-panel-card border border-panel-border rounded-lg px-4 py-2.5">
      <button v-for="(part, i) in breadcrumb" :key="i" @click="navigateTo(i)"
        :class="['hover:text-orange-400 transition', i === breadcrumb.length - 1 ? 'text-orange-400 font-semibold' : 'text-gray-400']">
        {{ part || '/' }}
      </button>
      <span v-if="breadcrumb.length > 1" class="text-gray-600 mx-1">/</span>
    </div>

    <!-- File List -->
    <div class="bg-panel-card border border-panel-border rounded-xl overflow-hidden">
      <div v-if="loading" class="p-8 text-center text-gray-400">Yükleniyor...</div>
      <table v-else class="w-full text-sm">
        <thead>
          <tr class="text-gray-400 border-b border-panel-border">
            <th class="text-left px-4 py-3 font-medium w-8"><input type="checkbox" class="accent-orange-500"></th>
            <th class="text-left px-4 py-3 font-medium">İsim</th>
            <th class="text-left px-4 py-3 font-medium">Boyut</th>
            <th class="text-left px-4 py-3 font-medium">İzin</th>
            <th class="text-left px-4 py-3 font-medium">Değiştirilme</th>
            <th class="text-right px-4 py-3 font-medium">İşlem</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="currentPath !== '/'" class="border-b border-panel-border/50 hover:bg-white/[0.02] transition cursor-pointer" @click="goUp">
            <td class="px-4 py-2.5"></td>
            <td class="px-4 py-2.5 text-gray-400 font-mono">..</td>
            <td colspan="4"></td>
          </tr>
          <tr v-for="item in fileList" :key="item.name" class="border-b border-panel-border/50 hover:bg-white/[0.02] transition">
            <td class="px-4 py-2.5"><input type="checkbox" class="accent-orange-500"></td>
            <td class="px-4 py-2.5">
              <button @click="item.is_dir ? enterDir(item.name) : editFile(item)" class="flex items-center gap-2 text-white hover:text-orange-400 transition font-mono text-xs">
                <span>{{ item.is_dir ? '📁' : fileIcon(item.name) }}</span>
                {{ item.name }}
              </button>
            </td>
            <td class="px-4 py-2.5 text-gray-400 text-xs">{{ item.is_dir ? '—' : formatBytes(item.size) }}</td>
            <td class="px-4 py-2.5 text-gray-400 font-mono text-xs">{{ item.permissions }}</td>
            <td class="px-4 py-2.5 text-gray-400 text-xs">{{ new Date(item.modified).toLocaleString() }}</td>
            <td class="px-4 py-2.5 text-right">
              <div class="flex justify-end gap-1">
                <button v-if="!item.is_dir" @click="editFile(item)" class="px-2 py-1 bg-blue-600/20 text-blue-400 rounded text-xs hover:bg-blue-600/40 transition">📝</button>
                <button @click="renameItem(item)" class="px-2 py-1 bg-yellow-600/20 text-yellow-400 rounded text-xs hover:bg-yellow-600/40 transition">✏️</button>
                <button @click="deleteItem(item)" class="px-2 py-1 bg-red-600/20 text-red-400 rounded text-xs hover:bg-red-600/40 transition">🗑</button>
              </div>
            </td>
          </tr>
          <tr v-if="fileList.length === 0" class="border-b border-panel-border/50"><td colspan="6" class="p-8 text-center text-gray-500">Klasör boş</td></tr>
        </tbody>
      </table>
    </div>

    <!-- Editor Modal -->
    <div v-if="showEditor" class="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center z-50">
      <div class="bg-panel-card border border-panel-border rounded-2xl w-full max-w-4xl h-[80vh] flex flex-col shadow-2xl">
        <div class="flex items-center justify-between p-4 border-b border-panel-border">
          <h3 class="text-white font-semibold font-mono text-sm">{{ editingFile?.name }}</h3>
          <div class="flex gap-2">
            <button @click="saveFile" class="px-4 py-1.5 bg-green-600 text-white rounded-lg text-sm hover:bg-green-700 transition">💾 Kaydet</button>
            <button @click="showEditor = false" class="px-4 py-1.5 bg-panel-hover text-gray-300 rounded-lg text-sm hover:bg-gray-600 transition">✖ Kapat</button>
          </div>
        </div>
        <textarea v-model="editorContent" class="flex-1 p-4 bg-[#0d1117] text-green-400 font-mono text-sm resize-none focus:outline-none" spellcheck="false"></textarea>
      </div>
    </div>

    <!-- Upload Modal -->
    <div v-if="showUploadModal" class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50" @click.self="showUploadModal = false">
      <div class="bg-panel-card border border-panel-border rounded-2xl w-full max-w-md p-6 shadow-2xl">
        <h3 class="text-xl font-bold text-white mb-4">📤 Dosya Yükle</h3>
        <div class="border-2 border-dashed border-panel-border rounded-xl p-8 text-center hover:border-orange-500 transition">
          <p class="text-gray-400">Dosyalarınızı buraya sürükleyin</p>
          <p class="text-gray-500 text-sm mt-1">veya tıklayarak seçin</p>
          <input type="file" multiple class="mt-3">
        </div>
        <button @click="showUploadModal = false" class="mt-4 w-full py-2.5 bg-gradient-to-r from-orange-600 to-amber-600 text-white rounded-lg font-medium">Yükle</button>
      </div>
    </div>

    <!-- New File Modal -->
    <div v-if="showNewModal" class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50" @click.self="showNewModal = false">
      <div class="bg-panel-card border border-panel-border rounded-2xl w-full max-w-sm p-6 shadow-2xl">
        <h3 class="text-lg font-bold text-white mb-4">📄 Yeni Dosya</h3>
        <input v-model="newFileName" type="text" placeholder="dosya_adi.txt" class="w-full px-4 py-2.5 bg-panel-hover border border-panel-border rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-orange-500">
        <div class="flex gap-3 mt-4">
          <button @click="createFile" class="flex-1 py-2 bg-gradient-to-r from-orange-600 to-amber-600 text-white rounded-lg text-sm">Oluştur</button>
          <button @click="showNewModal = false" class="px-4 py-2 bg-panel-hover text-gray-300 rounded-lg text-sm">İptal</button>
        </div>
      </div>
    </div>

    <!-- New Folder Modal -->
    <div v-if="showNewFolderModal" class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50" @click.self="showNewFolderModal = false">
      <div class="bg-panel-card border border-panel-border rounded-2xl w-full max-w-sm p-6 shadow-2xl">
        <h3 class="text-lg font-bold text-white mb-4">📁 Yeni Klasör</h3>
        <input v-model="newFolderName" type="text" placeholder="klasor_adi" class="w-full px-4 py-2.5 bg-panel-hover border border-panel-border rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-orange-500">
        <div class="flex gap-3 mt-4">
          <button @click="createFolder" class="flex-1 py-2 bg-gradient-to-r from-orange-600 to-amber-600 text-white rounded-lg text-sm">Oluştur</button>
          <button @click="showNewFolderModal = false" class="px-4 py-2 bg-panel-hover text-gray-300 rounded-lg text-sm">İptal</button>
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
import { ref, computed, onMounted } from 'vue'
import { FolderOpen } from 'lucide-vue-next'
import api from '../services/api'

const currentPath = ref('/var/www/html')
const fileList = ref([])
const loading = ref(false)
const notification = ref(null)

const showEditor = ref(false)
const showUploadModal = ref(false)
const showNewModal = ref(false)
const showNewFolderModal = ref(false)
const editingFile = ref(null)
const editorContent = ref('')
const newFileName = ref('')
const newFolderName = ref('')

const breadcrumb = computed(() => currentPath.value.split('/').filter(Boolean))

const showNotif = (msg, type = 'success') => {
  notification.value = { message: msg, type }
  setTimeout(() => notification.value = null, 3000)
}

const formatBytes = (bytes) => {
  if (bytes === 0) return '0 B'
  const k = 1024, sizes = ['B', 'KB', 'MB', 'GB', 'TB'], i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const fileIcon = (name) => {
  const ext = name.split('.').pop().toLowerCase()
  const map = { php: '🐘', js: '🟨', html: '🌐', css: '🎨', py: '🐍', txt: '📝', json: '📋', xml: '📄', log: '📜', sh: '⚙️', conf: '⚙️', sql: '🗃️', zip: '📦', gz: '📦', tar: '📦', jpg: '🖼️', png: '🖼️', svg: '🖼️', md: '📘' }
  return map[ext] || '📄'
}

const loadFiles = async () => {
  loading.value = true
  try {
    const res = await api.post('/files/list', { path: currentPath.value })
    fileList.value = res.data.data || []
  } catch (e) {
    showNotif('Dosyalar alınamadı', 'error')
    fileList.value = []
  } finally {
    loading.value = false
  }
}

const enterDir = (name) => {
  currentPath.value = currentPath.value.replace(/\/$/, '') + '/' + name
  loadFiles()
}

const goUp = () => {
  const parts = currentPath.value.split('/').filter(Boolean)
  parts.pop()
  currentPath.value = '/' + parts.join('/') || '/'
  loadFiles()
}

const navigateTo = (i) => {
  const parts = breadcrumb.value.slice(0, i + 1)
  currentPath.value = '/' + parts.join('/')
  loadFiles()
}

const editFile = async (item) => {
  editingFile.value = item
  try {
    const path = currentPath.value.replace(/\/$/, '') + '/' + item.name
    const res = await api.post('/files/read', { path })
    editorContent.value = res.data.content || ''
    showEditor.value = true
  } catch (e) {
    showNotif('Dosya okunamadı', 'error')
  }
}

const saveFile = async () => {
  try {
    const path = currentPath.value.replace(/\/$/, '') + '/' + editingFile.value.name
    await api.post('/files/write', { path, content: editorContent.value })
    showNotif(`💾 ${editingFile.value.name} kaydedildi`)
    showEditor.value = false
    loadFiles()
  } catch (e) {
    showNotif('Dosya kaydedilemedi', 'error')
  }
}

const renameItem = async (item) => {
  const newName = prompt('Yeni isim:', item.name)
  if (newName && newName !== item.name) {
    try {
      const old_path = currentPath.value.replace(/\/$/, '') + '/' + item.name
      const new_path = currentPath.value.replace(/\/$/, '') + '/' + newName
      await api.post('/files/rename', { old_path, new_path })
      showNotif(`✏️ Yeniden adlandırıldı: ${newName}`)
      loadFiles()
    } catch {
      showNotif('Adlandırma başarısız', 'error')
    }
  }
}

const deleteItem = async (item) => {
  if (!confirm(`Silmek istediğinize emin misiniz: ${item.name}?`)) return
  try {
    const path = currentPath.value.replace(/\/$/, '') + '/' + item.name
    await api.post('/files/delete', { path })
    showNotif(`🗑 ${item.name} silindi`)
    loadFiles()
  } catch {
    showNotif('Silme başarısız', 'error')
  }
}

const createFile = async () => {
  if (newFileName.value) {
    try {
      const path = currentPath.value.replace(/\/$/, '') + '/' + newFileName.value
      await api.post('/files/write', { path, content: '' })
      showNotif(`📄 ${newFileName.value} oluşturuldu`)
      newFileName.value = ''
      showNewModal.value = false
      loadFiles()
    } catch {
      showNotif('Oluşturulamadı', 'error')
    }
  }
}

const createFolder = async () => {
  if (newFolderName.value) {
    try {
      const path = currentPath.value.replace(/\/$/, '') + '/' + newFolderName.value
      await api.post('/files/create_dir', { path })
      showNotif(`📁 ${newFolderName.value} oluşturuldu`)
      newFolderName.value = ''
      showNewFolderModal.value = false
      loadFiles()
    } catch {
      showNotif('Oluşturulamadı', 'error')
    }
  }
}

onMounted(loadFiles)
</script>
