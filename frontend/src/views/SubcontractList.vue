<template>
  <div class="subcontract-page">
    <div class="toolbar">
      <el-button type="primary" @click="triggerImport">{{ $t('common.import') }}
        <input ref="importInput" type="file" accept=".xlsx" hidden @change="doImport" />
      </el-button>
      <el-button @click="doExport">{{ $t('common.export') }}</el-button>
      <el-button type="success" @click="openDialog()">{{ $t('common.create') }}</el-button>
      <el-select v-model="dim" style="width:140px;margin-left:8px">
        <el-option label="按项目" value="project" />
        <el-option label="按分包商" value="sub" />
      </el-select>
      <el-select v-model="filterTier" clearable placeholder="层级" style="width:110px">
        <el-option v-for="t in tiers" :key="t" :label="t" :value="t" />
      </el-select>
      <el-select v-model="filterCategory" clearable placeholder="专业分类" style="width:110px">
        <el-option v-for="c in categories" :key="c" :label="c" :value="c" />
      </el-select>
      <el-select v-model="filterProfession" clearable placeholder="专业" style="width:120px">
        <el-option v-for="p in professions" :key="p" :label="p" :value="p" />
      </el-select>
      <el-input v-model="keyword" :placeholder="$t('common.search')" clearable style="width:200px" @input="load" />
    </div>
    <div class="stats-bar">
      <span>共 <b>{{ total }}</b> 条合同，合计 <b>{{ fmtAmount(totalAmount) }}</b> 万元</span>
    </div>
    <el-table :data="records" stripe max-height="calc(100vh - 220px)" @row-click="showDetail">
      <el-table-column label="项目名称" min-width="180" show-overflow-tooltip>
        <template #default="{ row }">{{ row.project_name || row.project_short_name || '-' }}</template>
      </el-table-column>
      <el-table-column prop="sub_name" label="分包商名称" min-width="160" show-overflow-tooltip />
      <el-table-column prop="sub_tier" label="层级" width="90" />
      <el-table-column prop="profession_category" label="专业分类" width="90" />
      <el-table-column prop="standardized_profession" label="主专业" width="100" />
      <el-table-column prop="contract_no" label="合同编号" width="160" show-overflow-tooltip />
      <el-table-column label="合同额(万元)" width="120" align="right">
        <template #default="{ row }">{{ row.contract_amount ? row.contract_amount.toLocaleString() : '-' }}</template>
      </el-table-column>
      <el-table-column prop="progress_percent" label="施工状态" width="100" align="center" />
      <el-table-column label="" width="60" fixed="right">
        <template #default="{ row }"><el-button size="small" text @click.stop="openDialog(row)">{{ $t('common.edit') }}</el-button></template>
      </el-table-column>
    </el-table>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { SUBCONTRACTOR_TIERS, PROFESSION_CATEGORIES } from '../data/professions'

const records = ref<any[]>([])
const total = ref(0)
const totalAmount = ref(0)
const dim = ref('project')
const filterTier = ref('')
const filterCategory = ref('')
const filterProfession = ref('')
const keyword = ref('')
const importInput = ref<HTMLInputElement | null>(null)
const tiers = SUBCONTRACTOR_TIERS

function triggerImport() { importInput.value?.click() }
const categories = computed(() => PROFESSION_CATEGORIES.map(c => c.name))
const professions = computed(() => {
  if (!filterCategory.value) {
    let all: string[] = []
    PROFESSION_CATEGORIES.forEach(c => all.push(...c.professions))
    return [...new Set(all)]
  }
  const cat = PROFESSION_CATEGORIES.find(c => c.name === filterCategory.value)
  return cat ? cat.professions : []
})

async function load() {
  const params: any = { dim: dim.value }
  if (filterTier.value) params.tier = filterTier.value
  if (filterCategory.value) params.category = filterCategory.value
  if (filterProfession.value) params.profession = filterProfession.value
  if (keyword.value) params.keyword = keyword.value
  try {
    const { data } = await axios.get('/api/subcontract', { params })
    records.value = data.records || []
    total.value = data.total_count || 0
    totalAmount.value = data.total_amount || 0
  } catch { /* ignore */ }
}

async function doImport(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  const fd = new FormData()
  fd.append('file', file)
  try {
    const { data } = await axios.post('/api/subcontract/import', fd)
    ElMessage.success('Imported ' + data.imported)
    load()
  } catch (err: any) { ElMessage.error(err?.response?.data || 'Import failed') }
}

async function doExport() {
  const params: any = {}
  if (filterTier.value) params.tier = filterTier.value
  if (filterCategory.value) params.category = filterCategory.value
  if (filterProfession.value) params.profession = filterProfession.value
  if (keyword.value) params.keyword = keyword.value
  const { data } = await axios.get('/api/subcontract/export', { params, responseType: 'blob' })
  const url = URL.createObjectURL(new Blob([data]))
  const a = document.createElement('a')
  a.href = url; a.download = 'subcontract_export.xlsx'; a.click()
  URL.revokeObjectURL(url)
}

async function showDetail(row: any) {
  try {
    const { data } = await axios.get('/api/subcontract/' + row.id)
    ElMessage.info(`${data.sub_name} | ${data.sub_tier || '-'} | ${data.standardized_profession || '-'}`)
  } catch { ElMessage.error('Failed') }
}

function openDialog(row?: any) {
  const payload = row ? { ...row } : {
    sub_name: '', sub_tier: '', contract_no: '', contract_name: '',
    contract_amount: 0, supplement_amount: 0, progress_percent: '',
    evaluation_completed: '', remarks: ''
  }
  ElMessageBox.prompt('Edit subcontractor (simplified)', 'Edit', {
    confirmButtonText: 'Save', inputValue: JSON.stringify(payload)
  }).then(() => ElMessage.success('Saved (stub)'))
}

function fmtAmount(v: number) {
  if (!v) return '0'
  if (v >= 10000) return (v / 10000).toFixed(2) + ' 亿'
  return v.toLocaleString('zh-CN', { maximumFractionDigits: 0 })
}

onMounted(load)
</script>

<style scoped>
.subcontract-page { display: flex; flex-direction: column; gap: 10px; }
.toolbar { display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }
.stats-bar { font-size: 14px; color: #606266; padding: 4px 0; }
</style>