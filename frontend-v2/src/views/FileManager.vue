<template>
  <div class="file-manager">
    <div class="toolbar">
      <span class="root-label">{{ $t('file.rootPath') }}:</span>
      <el-input :model-value="rootPath" readonly style="flex:1;min-width:200px" />
      <el-button @click="setRoot">{{ $t('file.setRoot') }}</el-button>
      <el-button type="primary" @click="loadFiles">{{ $t('file.scan') }}</el-button>
    </div>

    <el-input v-model="search" :placeholder="$t('common.search')" clearable style="width:240px;margin-bottom:12px" />

    <div v-if="loading" style="text-align:center;padding:40px"><el-icon class="is-loading"><Loading /></el-icon></div>

    <el-table v-else :data="filteredFiles" row-key="path" style="width:100%" max-height="calc(100vh - 240px)">
      <el-table-column :label="$t('project.projectName')" min-width="300">
        <template #default="{ row }">
          <span :style="{ paddingLeft: (row._depth || 0) * 20 + 'px' }">
            <el-icon v-if="row.is_dir" style="margin-right:6px"><Folder /></el-icon>
            <el-icon v-else style="margin-right:6px"><Document /></el-icon>
            {{ row.name }}
          </span>
        </template>
      </el-table-column>
      <el-table-column prop="size" label="Size" width="100" />
      <el-table-column prop="mod_time" label="Modified" width="180" />
      <el-table-column label="" width="100">
        <template #default="{ row }">
          <el-button v-if="!row.is_dir" size="small" text @click="openFile(row)">{{ $t('file.openFile') }}</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import axios from 'axios'

const rootPath = ref('')
const files = ref<any[]>([])
const search = ref('')
const loading = ref(false)

const filteredFiles = computed(() => {
  if (!search.value) return files.value
  const q = search.value.toLowerCase()
  return files.value.filter((f: any) => f.name.toLowerCase().includes(q))
})

function flattenTree(nodes: any[], depth = 0): any[] {
  const result: any[] = []
  for (const n of nodes) {
    result.push({ ...n, _depth: depth })
    if (n.children && n.children.length) {
      result.push(...flattenTree(n.children, depth + 1))
    }
  }
  return result
}

async function loadFiles() {
  loading.value = true
  try {
    const { data } = await axios.get('/api/files/scan')
    rootPath.value = data.root || ''
    files.value = flattenTree(data.files || [])
  } catch { files.value = [] }
  finally { loading.value = false }
}

async function setRoot() {
  // For simplicity, use a prompt
  const p = prompt('Enter file root path:', rootPath.value)
  if (p) {
    await axios.post('/api/files/root', { path: p })
    rootPath.value = p
    await loadFiles()
  }
}

function openFile(row: any) {
  window.open('/api/files/' + encodeURIComponent(row.path), '_blank')
}

onMounted(async () => {
  try {
    const { data } = await axios.get('/api/files/root')
    rootPath.value = data.path || ''
  } catch {}
})
</script>

<style scoped>
.file-manager { display: flex; flex-direction: column; gap: 12px; }
.toolbar { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.root-label { white-space: nowrap; font-weight: 500; }
</style>
