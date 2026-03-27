<template>
  <div class="space-y-4">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-white flex items-center gap-3">
          <FolderOpen class="w-7 h-7 text-orange-400" />
          {{ $t('file_manager.title') || 'File Manager' }}
        </h1>
        <p class="text-gray-400 mt-1">{{ $t('file_manager.subtitle') || 'Sunucu dosyalarÄ±nÄ± yÃ¶netin' }}</p>
      </div>
      <div class="flex gap-2">
        <button v-if="selectedFiles.length > 0" @click="showCompressModal = true" class="px-4 py-2 bg-blue-600/20 text-blue-400 border border-blue-600/30 rounded-lg text-sm hover:bg-blue-600/40 transition">ğŸ—œ {{ $t('file_manager.compress') || 'ArÅŸive Ekle' }}</button>
        <button v-if="selectedFiles.length > 0" @click="trashSelectedItems" class="px-4 py-2 bg-red-600/20 text-red-400 border border-red-600/30 rounded-lg text-sm hover:bg-red-600/40 transition">ğŸ—‘ {{ $t('file_manager.trash') || 'Ã‡Ã¶p Kutusuna GÃ¶nder' }}</button>
        <button v-if="selectedFiles.length > 0" @click="deleteSelectedItems" class="px-4 py-2 bg-red-700/30 text-red-300 border border-red-500/40 rounded-lg text-sm hover:bg-red-700/50 transition">Permanent Delete</button>
        <button @click="showUploadModal = true" class="px-4 py-2 bg-gradient-to-r from-orange-600 to-amber-600 text-white rounded-lg text-sm hover:from-orange-700 hover:to-amber-700 transition">ğŸ“¤ {{ $t('file_manager.upload') || 'YÃ¼kle' }}</button>
        <button @click="showNewModal = true" class="px-4 py-2 bg-panel-hover text-gray-300 rounded-lg text-sm hover:bg-gray-600 transition">ğŸ“„ {{ $t('file_manager.new_file') || 'Yeni Dosya' }}</button>
        <button @click="showNewFolderModal = true" class="px-4 py-2 bg-panel-hover text-gray-300 rounded-lg text-sm hover:bg-gray-600 transition">ğŸ“ {{ $t('file_manager.new_folder') || 'Yeni KlasÃ¶r' }}</button>
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
      <div v-if="loading" class="p-8 text-center text-gray-400">{{ $t('common.loading') || 'YÃ¼kleniyor...' }}</div>
      <table v-else class="w-full text-sm">
        <thead>
          <tr class="text-gray-400 border-b border-panel-border">
            <th class="text-left px-4 py-3 font-medium w-8">
              <input type="checkbox" :checked="allSelected" @change="toggleAll" class="accent-orange-500">
            </th>
            <th class="text-left px-4 py-3 font-medium">{{ $t('file_manager.name') || 'Ä°sim' }}</th>
            <th class="text-left px-4 py-3 font-medium">{{ $t('file_manager.size') || 'Boyut' }}</th>
            <th class="text-left px-4 py-3 font-medium">{{ $t('file_manager.permissions') || 'Ä°zin' }}</th>
            <th class="text-left px-4 py-3 font-medium">{{ $t('file_manager.modified') || 'DeÄŸiÅŸtirilme' }}</th>
            <th class="text-right px-4 py-3 font-medium">{{ $t('file_manager.actions') || 'Ä°ÅŸlem' }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-if="currentPath !== '/home'" class="border-b border-panel-border/50 hover:bg-white/[0.02] transition cursor-pointer" @click="goUp">
            <td class="px-4 py-2.5"></td>
            <td class="px-4 py-2.5 text-gray-400 font-mono">..</td>
            <td colspan="4"></td>
          </tr>
          <tr v-for="item in fileList" :key="item.name" class="border-b border-panel-border/50 hover:bg-white/[0.02] transition">
            <td class="px-4 py-2.5">
              <input type="checkbox" :value="item.name" v-model="selectedFiles" class="accent-orange-500">
            </td>
            <td class="px-4 py-2.5">
              <button @click="item.is_dir ? enterDir(item.name) : editFile(item)" class="flex items-center gap-2 text-white hover:text-orange-400 transition font-mono text-xs">
                <span>{{ item.is_dir ? 'ğŸ“' : fileIcon(item.name) }}</span>
                {{ item.name }}
              </button>
            </td>
            <td class="px-4 py-2.5 text-gray-400 text-xs">{{ item.is_dir ? 'â€”' : formatBytes(item.size) }}</td>
            <td class="px-4 py-2.5 text-gray-400 font-mono text-xs">{{ item.permissions }}</td>
            <td class="px-4 py-2.5 text-gray-400 text-xs">{{ new Date(item.modified).toLocaleString() }}</td>
            <td class="px-4 py-2.5 text-right">
              <div class="flex justify-end gap-1">
                <button v-if="isArchive(item.name)" @click="extractItem(item)" class="px-2 py-1 bg-purple-600/20 text-purple-400 rounded text-xs hover:bg-purple-600/40 transition" :title="$t('file_manager.extract') || 'Buraya Ã‡Ä±kart'">ğŸ“¦</button>
                <button v-if="!item.is_dir" @click="editFile(item)" class="px-2 py-1 bg-blue-600/20 text-blue-400 rounded text-xs hover:bg-blue-600/40 transition">ğŸ“</button>
                <button @click="renameItem(item)" class="px-2 py-1 bg-yellow-600/20 text-yellow-400 rounded text-xs hover:bg-yellow-600/40 transition">âœï¸</button>
                <button @click="trashSingleItem(item)" class="px-2 py-1 bg-red-600/20 text-red-400 rounded text-xs hover:bg-red-600/40 transition">ğŸ—‘</button>
                <button @click="deleteSingleItem(item)" class="px-2 py-1 bg-red-700/30 text-red-300 rounded text-xs hover:bg-red-700/50 transition">✕</button>
              </div>
            </td>
          </tr>
          <tr v-if="fileList.length === 0" class="border-b border-panel-border/50"><td colspan="6" class="p-8 text-center text-gray-500">{{ $t('file_manager.empty_dir') || 'KlasÃ¶r boÅŸ' }}</td></tr>
        </tbody>
      </table>
    </div>

    <!-- Editor Modal -->
    <div v-if="showEditor" class="fixed inset-0 bg-black/70 backdrop-blur-sm flex items-center justify-center z-50">
      <div class="bg-panel-card border border-panel-border rounded-2xl w-full max-w-4xl h-[80vh] flex flex-col shadow-2xl">
        <div class="flex items-center justify-between p-4 border-b border-panel-border">
          <h3 class="text-white font-semibold font-mono text-sm">{{ editingFile?.name }}</h3>
          <div class="flex gap-2">
            <button @click="saveFile" class="px-4 py-1.5 bg-green-600 text-white rounded-lg text-sm hover:bg-green-700 transition">ğŸ’¾ {{ $t('common.save') || 'Kaydet' }}</button>
            <button @click="showEditor = false" class="px-4 py-1.5 bg-panel-hover text-gray-300 rounded-lg text-sm hover:bg-gray-600 transition">âœ– {{ $t('common.close') || 'Kapat' }}</button>
          </div>
        </div>
        <textarea v-model="editorContent" class="flex-1 p-4 bg-[#0d1117] text-green-400 font-mono text-sm resize-none focus:outline-none" spellcheck="false"></textarea>
      </div>
    </div>

    <!-- Upload Modal -->
    <div v-if="showUploadModal" class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50" @click.self="showUploadModal = false">
      <div class="bg-panel-card border border-panel-border rounded-2xl w-full max-w-md p-6 shadow-2xl">
        <h3 class="text-xl font-bold text-white mb-4">ğŸ“¤ {{ $t('file_manager.upload') || 'Dosya YÃ¼kle' }}</h3>
        <div class="border-2 border-dashed border-panel-border rounded-xl p-8 text-center hover:border-orange-500 transition">
          <p class="text-gray-400">{{ $t('file_manager.drag_drop') || 'DosyalarÄ±nÄ±zÄ± buraya sÃ¼rÃ¼kleyin' }}</p>
          <input type="file" multiple class="mt-3 text-white">
        </div>
        <button @click="showUploadModal = false" class="mt-4 w-full py-2.5 bg-gradient-to-r from-orange-600 to-amber-600 text-white rounded-lg font-medium">{{ $t('common.close') || 'Kapat' }}</button>
      </div>
    </div>

    <!-- Compress Modal -->
    <div v-if="showCompressModal" class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50" @click.self="showCompressModal = false">
      <div class="bg-panel-card border border-panel-border rounded-2xl w-full max-w-sm p-6 shadow-2xl">
        <h3 class="text-lg font-bold text-white mb-4">ğŸ—œ {{ $t('file_manager.compress_title') || 'ArÅŸiv OluÅŸtur' }}</h3>
        <input v-model="compressName" type="text" placeholder="arsiv_adi" class="w-full px-4 py-2.5 bg-panel-hover border border-panel-border rounded-lg text-white mb-4">
        <select v-model="compressFormat" class="w-full px-4 py-2.5 bg-panel-hover border border-panel-border rounded-lg text-white mb-4">
          <option value="zip">.zip</option>
          <option value="tar.gz">.tar.gz</option>
        </select>
        <div class="flex gap-3">
          <button @click="compressSelected" class="flex-1 py-2 bg-blue-600 text-white rounded-lg text-sm">{{ $t('common.create') || 'OluÅŸtur' }}</button>
          <button @click="showCompressModal = false" class="px-4 py-2 bg-panel-hover text-gray-300 rounded-lg text-sm">{{ $t('common.cancel') || 'Ä°ptal' }}</button>
        </div>
      </div>
    </div>

    <!-- New File Modal -->
    <div v-if="showNewModal" class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50" @click.self="showNewModal = false">
      <div class="bg-panel-card border border-panel-border rounded-2xl w-full max-w-sm p-6 shadow-2xl">
        <h3 class="text-lg font-bold text-white mb-4">ğŸ“„ {{ $t('file_manager.new_file') || 'Yeni Dosya' }}</h3>
        <input v-model="newFileName" type="text" placeholder="dosya_adi.txt" class="w-full px-4 py-2.5 bg-panel-hover border border-panel-border rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-orange-500">
        <div class="flex gap-3 mt-4">
          <button @click="createFile" class="flex-1 py-2 bg-gradient-to-r from-orange-600 to-amber-600 text-white rounded-lg text-sm">{{ $t('common.create') || 'OluÅŸtur' }}</button>
          <button @click="showNewModal = false" class="px-4 py-2 bg-panel-hover text-gray-300 rounded-lg text-sm">{{ $t('common.cancel') || 'Ä°ptal' }}</button>
        </div>
      </div>
    </div>

    <!-- New Folder Modal -->
    <div v-if="showNewFolderModal" class="fixed inset-0 bg-black/60 backdrop-blur-sm flex items-center justify-center z-50" @click.self="showNewFolderModal = false">
      <div class="bg-panel-card border border-panel-border rounded-2xl w-full max-w-sm p-6 shadow-2xl">
        <h3 class="text-lg font-bold text-white mb-4">ğŸ“ {{ $t('file_manager.new_folder') || 'Yeni KlasÃ¶r' }}</h3>
        <input v-model="newFolderName" type="text" placeholder="klasor_adi" class="w-full px-4 py-2.5 bg-panel-hover border border-panel-border rounded-lg text-white placeholder-gray-500 focus:outline-none focus:border-orange-500">
        <div class="flex gap-3 mt-4">
          <button @click="createFolder" class="flex-1 py-2 bg-gradient-to-r from-orange-600 to-amber-600 text-white rounded-lg text-sm">{{ $t('common.create') || 'OluÅŸtur' }}</button>
          <button @click="showNewFolderModal = false" class="px-4 py-2 bg-panel-hover text-gray-300 rounded-lg text-sm">{{ $t('common.cancel') || 'Ä°ptal' }}</button>
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
import { useI18n } from 'vue-i18n'
import { FolderOpen } from 'lucide-vue-next'
import api from '../services/api'

const { t } = useI18n()
const currentPath = ref('/home')
const fileList = ref([])
const selectedFiles = ref([])
const loading = ref(false)
const notification = ref(null)

const showEditor = ref(false)
const showUploadModal = ref(false)
const showNewModal = ref(false)
const showNewFolderModal = ref(false)
const showCompressModal = ref(false)

const editingFile = ref(null)
const editorContent = ref('')
const newFileName = ref('')
const newFolderName = ref('')
const compressName = ref('archive')
const compressFormat = ref('zip')

const breadcrumb = computed(() => currentPath.value.split('/').filter(Boolean))
const allSelected = computed(() => fileList.value.length > 0 && selectedFiles.value.length === fileList.value.length)

const toggleAll = (e) => {
  if (e.target.checked) {
    selectedFiles.value = fileList.value.map(f => f.name)
  } else {
    selectedFiles.value = []
  }
}

const showNotif = (msg, type = 'success') => {
  notification.value = { message: msg, type }
  setTimeout(() => notification.value = null, 3000)
}

const formatBytes = (bytes) => {
  if (bytes === 0) return '0 B'
  const k = 1024, sizes = ['B', 'KB', 'MB', 'GB', 'TB'], i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const isArchive = (name) => {
  const ext = name.split('.').pop().toLowerCase()
  return ['zip', 'gz', 'tar', 'tgz'].includes(ext)
}

const fileIcon = (name) => {
  const ext = name.split('.').pop().toLowerCase()
  const map = { php: 'ğŸ˜', js: 'ğŸŸ¨', html: 'ğŸŒ', css: 'ğŸ¨', py: 'ğŸ', txt: 'ğŸ“', json: 'ğŸ“‹', xml: 'ğŸ“„', log: 'ğŸ“œ', sh: 'âš™ï¸', conf: 'âš™ï¸', sql: 'ğŸ—ƒï¸', zip: 'ğŸ“¦', gz: 'ğŸ“¦', tar: 'ğŸ“¦', tgz: 'ğŸ“¦', jpg: 'ğŸ–¼ï¸', png: 'ğŸ–¼ï¸', svg: 'ğŸ–¼ï¸', md: 'ğŸ“˜' }
  return map[ext] || 'ğŸ“„'
}

const loadFiles = async () => {
  loading.value = true
  selectedFiles.value = []
  try {
    const res = await api.post('/files/list', { path: currentPath.value })
    fileList.value = res.data.data || []
  } catch (e) {
    showNotif(t('common.error') || 'Hata', 'error')
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
    editorContent.value = res.data.data || ''
    showEditor.value = true
  } catch (e) {
    showNotif(t('common.error') || 'Okuma hatasÄ±', 'error')
  }
}

const saveFile = async () => {
  try {
    const path = currentPath.value.replace(/\/$/, '') + '/' + editingFile.value.name
    await api.post('/files/write', { path, content: editorContent.value })
    showNotif(t('common.success') || 'Kaydedildi')
    showEditor.value = false
    loadFiles()
  } catch (e) {
    showNotif(t('common.error') || 'Hata', 'error')
  }
}

const renameItem = async (item) => {
  const newName = prompt(t('common.new_name') || 'Yeni isim:', item.name)
  if (newName && newName !== item.name) {
    try {
      const old_path = currentPath.value.replace(/\/$/, '') + '/' + item.name
      const new_path = currentPath.value.replace(/\/$/, '') + '/' + newName
      await api.post('/files/rename', { old_path, new_path })
      showNotif(t('common.success') || 'Yeniden adlandÄ±rÄ±ldÄ±')
      loadFiles()
    } catch {
      showNotif(t('common.error') || 'AdlandÄ±rma baÅŸarÄ±sÄ±z', 'error')
    }
  }
}

const trashSingleItem = async (item) => {
  if (!confirm(`${t('common.confirm_delete') || 'Emin misiniz?'} (${item.name})`)) return
  try {
    const path = currentPath.value.replace(/\/$/, '') + '/' + item.name
    await api.post('/files/trash', { path })
    showNotif(t('common.success') || 'Ã‡Ã¶p kutusuna taÅŸÄ±ndÄ±')
    loadFiles()
  } catch {
    showNotif(t('common.error') || 'Silme baÅŸarÄ±sÄ±z', 'error')
  }
}

const trashSelectedItems = async () => {
  if (!selectedFiles.value.length || !confirm(t('common.confirm_delete') || 'SeÃ§ili Ã¶ÄŸeleri Ã§Ã¶p kutusuna taÅŸÄ±mak istediÄŸinize emin misiniz?')) return
  try {
    for (const name of selectedFiles.value) {
      const path = currentPath.value.replace(/\/$/, '') + '/' + name
      await api.post('/files/trash', { path })
    }
    showNotif(t('common.success') || 'Ã–ÄŸeler Ã§Ã¶p kutusuna taÅŸÄ±ndÄ±')
    loadFiles()
  } catch {
    showNotif(t('common.error') || 'BazÄ± Ã¶ÄŸeler taÅŸÄ±namadÄ±', 'error')
    loadFiles()
  }
}


const deleteSingleItem = async (item) => {
  if (!confirm(`Permanent delete? (${item.name})`)) return
  try {
    const path = currentPath.value.replace(/\/$/, '') + '/' + item.name
    await api.post('/files/delete', { path })
    showNotif('Item permanently deleted')
    loadFiles()
  } catch {
    showNotif('Permanent delete failed', 'error')
  }
}

const deleteSelectedItems = async () => {
  if (!selectedFiles.value.length || !confirm('Permanently delete selected items? This cannot be undone.')) return
  try {
    for (const name of selectedFiles.value) {
      const path = currentPath.value.replace(/\/$/, '') + '/' + name
      await api.post('/files/delete', { path })
    }
    showNotif('Selected items permanently deleted')
    loadFiles()
  } catch {
    showNotif('Permanent delete failed for one or more items', 'error')
    loadFiles()
  }
}
const compressSelected = async () => {
  if (!compressName.value) return
  try {
    const dest_path = currentPath.value.replace(/\/$/, '') + '/' + compressName.value + '.' + compressFormat.value
    const sources = selectedFiles.value.map(name => currentPath.value.replace(/\/$/, '') + '/' + name)
    await api.post('/files/compress', { format: compressFormat.value, dest_path, sources })
    showNotif(t('common.success') || 'ArÅŸiv oluÅŸturuldu')
    showCompressModal.value = false
    loadFiles()
  } catch (e) {
    showNotif(t('common.error') || 'ArÅŸivleme baÅŸarÄ±sÄ±z', 'error')
  }
}

const extractItem = async (item) => {
  try {
    const archive_path = currentPath.value.replace(/\/$/, '') + '/' + item.name
    const dest_dir = currentPath.value
    await api.post('/files/extract', { archive_path, dest_dir })
    showNotif(t('common.success') || 'ArÅŸiv Ã§Ä±karÄ±ldÄ±')
    loadFiles()
  } catch (e) {
    showNotif(t('common.error') || 'Ã‡Ä±karma baÅŸarÄ±sÄ±z', 'error')
  }
}

const createFile = async () => {
  if (newFileName.value) {
    try {
      const path = currentPath.value.replace(/\/$/, '') + '/' + newFileName.value
      await api.post('/files/write', { path, content: '' })
      showNotif(t('common.success') || 'Dosya oluÅŸturuldu')
      newFileName.value = ''
      showNewModal.value = false
      loadFiles()
    } catch {
      showNotif(t('common.error') || 'OluÅŸturulamadÄ±', 'error')
    }
  }
}

const createFolder = async () => {
  if (newFolderName.value) {
    try {
      const path = currentPath.value.replace(/\/$/, '') + '/' + newFolderName.value
      await api.post('/files/create_dir', { path })
      showNotif(t('common.success') || 'KlasÃ¶r oluÅŸturuldu')
      newFolderName.value = ''
      showNewFolderModal.value = false
      loadFiles()
    } catch {
      showNotif(t('common.error') || 'OluÅŸturulamadÄ±', 'error')
    }
  }
}

onMounted(loadFiles)
</script>

