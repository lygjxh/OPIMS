<template>
  <div class="subcontract-page">
    <div class="toolbar">
      <el-button type="primary" @click="triggerImport">{{ $t('common.import') }}
        <input ref="importInput" type="file" accept=".xlsx" hidden @change="doImport" />
      </el-button>
      <el-button @click="doExport">{{ $t('common.export') }}</el-button>
      <el-button type="success" @click="openDialog()">{{ $t('common.create') }}</el-button>
      <el-select v-model="dim" style="width:140px;margin-left:8px" @change="load">
        <el-option label="按项目" value="project" />
        <el-option label="按分包商" value="sub" />
      </el-select>
      <el-select v-model="filterTier" clearable placeholder="层级" style="width:110px" @change="load">
        <el-option v-for="t in tiers" :key="t" :label="t" :value="t" />
      </el-select>
      <el-select v-model="filterCategory" clearable placeholder="专业分类" style="width:110px" @change="load">
        <el-option v-for="c in categories" :key="c" :label="c" :value="c" />
      </el-select>
      <el-select v-model="filterProfession" clearable placeholder="专业" style="width:120px" @change="load">
        <el-option v-for="p in professions" :key="p" :label="p" :value="p" />
      </el-select>
      <el-input v-model="keyword" :placeholder="$t('common.search')" clearable style="width:200px" @input="load" />
    </div>
    <div class="stats-bar">
      <span>共 <b>{{ total }}</b> 条合同，合计 <b>{{ fmtAmount(totalAmount) }}</b> 万元</span>
    </div>
    <el-table :data="records" stripe max-height="calc(100vh - 220px)" @row-click="showDetail"
      :row-class-name="rowClassName">
      <el-table-column label="项目简称" min-width="140" show-overflow-tooltip>
        <template #default="{ row }">{{ row.project_short_name || row.project_name || '-' }}</template>
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

    <!-- 详情弹窗：6分组 -->
    <el-dialog v-model="detailVisible" title="分包详情" width="680px" top="4vh">
      <template v-if="detail">
        <div class="detail-groups">
          <!-- 基础 -->
          <div class="dg"><div class="dg-h">基础信息</div>
            <div class="dg-b"><el-description term="分包商名称" :content="detail.sub_name" />
              <el-description term="实控人" :content="detail.sub_controller || '-'" />
              <el-description term="实控人电话" :content="detail.sub_controller_phone || '-'" /></div></div>
          <!-- 层级与专业 -->
          <div class="dg"><div class="dg-h">层级与专业</div>
            <div class="dg-b"><el-description term="分包商层级" :content="detail.sub_tier || '-'" />
              <el-description term="主专业分类" :content="detail.profession_category || '-'" />
              <el-description term="主专业" :content="detail.standardized_profession || '-'" />
              <el-description term="合同专业" :content="detail.sub_contract_profession && detail.sub_contract_profession !== detail.standardized_profession ? detail.sub_contract_profession : '同主专业'" /></div></div>
          <!-- 合同 -->
          <div class="dg"><div class="dg-h">合同信息</div>
            <div class="dg-b"><el-description term="合同编号" :content="detail.contract_no || '-'" />
              <el-description term="合同名称" :content="detail.contract_name || '-'" />
              <el-description term="合同额(万元)" :content="detail.contract_amount?.toLocaleString() || '-'" />
              <el-description term="补充协议金额" :content="detail.supplement_amount?.toLocaleString() || '-'" />
              <el-description term="签订日期" :content="detail.contract_date || '-'" /></div></div>
          <!-- 施工进度 -->
          <div class="dg"><div class="dg-h">施工进度</div>
            <div class="dg-b"><el-description term="进场时间" :content="detail.entry_date || '-'" />
              <el-description term="预计撤场" :content="detail.exit_date || '-'" />
              <el-description term="完成百分比" :content="detail.progress_percent || '-'" />
              <el-description term="人员数量" :content="detail.personnel_count != null ? String(detail.personnel_count) : '-'" /></div></div>
          <!-- 人员配置 -->
          <div class="dg"><div class="dg-h">人员配置</div>
            <div class="dg-b"><el-description term="现场负责人" :content="detail.site_leader || '-'" />
              <el-description term="审批版本" :content="detail.site_leader_approved || '-'" />
              <el-description term="在岗情况" :content="detail.site_leader_status || '-'" />
              <el-description term="技术负责人" :content="detail.tech_leader || '-'" />
              <el-description term="审批版本" :content="detail.tech_leader_approved || '-'" />
              <el-description term="在岗情况" :content="detail.tech_leader_status || '-'" />
              <el-description term="安全员" :content="detail.safety_officer || '-'" />
              <el-description term="审批版本" :content="detail.safety_officer_approved || '-'" />
              <el-description term="在岗情况" :content="detail.safety_officer_status || '-'" />
              <el-description term="与合同一致" :content="detail.contract_compliance || '-'" />
              <el-description term="不一致说明" :content="detail.noncompliance_note || '-'" /></div></div>
          <!-- 评价与备注 -->
          <div class="dg"><div class="dg-h">评价与备注</div>
            <div class="dg-b"><el-description term="完工履约评价" :content="detail.evaluation_completed || '-'" />
              <el-description term="备注" :content="detail.remarks || '-'" /></div></div>
        </div>
        <!-- 黑名单状态 -->
        <div v-if="detail.blacklisted" class="bl-warn">
          <el-tag type="danger" effect="dark">⚠ 该分包商已列入黑名单</el-tag>
        </div>
      </template>
    </el-dialog>
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
const detailVisible = ref(false)
const detail = ref<any>(null)
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

function rowClassName({ row }: any) {
  return row.blacklisted ? 'blacklisted-row' : ''
}

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
    ElMessage.success('导入 ' + (data.imported || 0) + ' 条')
    load()
  } catch (err: any) { ElMessage.error(err?.response?.data || '导入失败') }
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
    detail.value = data
    detailVisible.value = true
  } catch { ElMessage.error('加载详情失败') }
}

function openDialog(row?: any) {
  const payload = row ? { ...row } : {
    sub_name: '', sub_tier: '', contract_no: '', contract_name: '',
    contract_amount: 0, supplement_amount: 0, progress_percent: '',
    evaluation_completed: '', remarks: ''
  }
  ElMessageBox.prompt('编辑分包商（简化版）', '编辑', {
    confirmButtonText: '保存', inputValue: JSON.stringify(payload)
  }).then(() => ElMessage.success('保存成功（桩代码）'))
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
.detail-groups { display: flex; flex-direction: column; gap: 16px; max-height: 60vh; overflow-y: auto; }
.dg { border: 1px solid #ebeef5; border-radius: 6px; }
.dg-h { background: #f5f7fa; padding: 8px 12px; font-weight: 600; font-size: 14px; border-bottom: 1px solid #ebeef5; }
.dg-b { padding: 8px 12px; display: grid; grid-template-columns: 1fr 1fr; gap: 4px 16px; }
.dg-b :deep(.el-description) { font-size: 13px; }
.dg-b :deep(.el-description__term) { color: #909399; }
.dg-b :deep(.el-description__content) { color: #303133; }
.bl-warn { margin-top: 12px; text-align: center; }
:deep(.blacklisted-row) { background-color: #f5f5f5 !important; }
:deep(.blacklisted-row:hover) { background-color: #eee !important; }
</style>
