<template>
  <div class="project-list">
    <div class="toolbar">
      <el-input v-model="search" placeholder="搜索" clearable style="width:200px" @keyup.enter="load" />
      <el-select v-model="filters.type" placeholder="项目类型" clearable style="width:100px" @change="load">
        <el-option v-for="t in TYPES" :key="t" :label="t" :value="t" />
      </el-select>
      <el-select v-model="filters.status" placeholder="项目状态" clearable style="width:130px" @change="load">
        <el-option v-for="s in STATUSES" :key="s" :label="s" :value="s" />
      </el-select>
      <el-select v-model="filters.domestic" placeholder="境内/境外" clearable style="width:110px" @change="load">
        <el-option v-for="d in DOMESTIC_OPTIONS" :key="d" :label="d" :value="d" />
      </el-select>
      <el-select v-model="filters.country" placeholder="国别" clearable style="width:120px" @change="load">
        <el-option v-for="c in countries" :key="c" :label="c" :value="c" />
      </el-select>
      <div style="flex:1" />
      <el-button type="primary" @click="showDialog(null)">新建</el-button>
      <el-button @click="importExcel">导入</el-button>
      <el-button @click="exportExcel">导出</el-button>
      <input ref="fileInput" type="file" accept=".xlsx" style="display:none" @change="onFileChange" />
    </div>

    <el-table :data="projects" stripe v-loading="loading" style="flex:1" @row-click="showDetail">
      <el-table-column prop="country" label="国别" width="120" align="center" fixed />
      <el-table-column prop="short_name" label="项目简称" width="180" />
      <el-table-column prop="implement_unit" label="实施单位" width="130" align="center" show-overflow-tooltip />
      <el-table-column prop="project_name" label="项目名称" min-width="280" show-overflow-tooltip />
      <el-table-column prop="contract_amount" label="合同额(万元)" width="150" align="right" class-name="col-amount">
        <template #default="{ row }">{{ fmtMoney(row.contract_amount) }}</template>
      </el-table-column>
      <el-table-column label="合同开工" width="100">
        <template #default="{ row }">{{ fmtDate(row.contract_start_year, row.contract_start_month) }}</template>
      </el-table-column>
      <el-table-column label="合同竣工" width="100">
        <template #default="{ row }">{{ fmtDate(row.contract_end_year, row.contract_end_month) }}</template>
      </el-table-column>
      <el-table-column prop="project_status" label="状态" width="80">
        <template #default="{ row }">
          <el-tag :type="statusTag(row.project_status)" size="small">{{ row.project_status }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="140" fixed="right">
        <template #default="{ row }">
          <el-button text type="primary" size="small" @click.stop="showDialog(row)">编辑</el-button>
          <el-divider direction="vertical" />
          <el-button text type="danger" size="small" @click.stop="del(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- Edit Dialog -->
    <el-dialog v-model="dialogVisible" :title="editingId ? '编辑项目' : '新建项目'" width="700px">
      <el-form :model="form" label-width="110px">
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="项目简称" required><el-input v-model="form.short_name" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="合同编号"><el-input v-model="form.contract_no" /></el-form-item></el-col>
        </el-row>
        <el-form-item label="项目名称" required><el-input v-model="form.project_name" /></el-form-item>
        <el-row :gutter="12">
          <el-col :span="8"><el-form-item label="项目类型"><el-select v-model="form.project_type"><el-option v-for="t in TYPES" :key="t" :label="t" :value="t" /></el-select></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="项目状态"><el-select v-model="form.project_status"><el-option v-for="s in STATUSES" :key="s" :label="s" :value="s" /></el-select></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="合同额(万元)"><el-input-number v-model="form.contract_amount" :min="0" style="width:100%" /></el-form-item></el-col>
        </el-row>
        <el-form-item label="实施单位"><el-input v-model="form.implement_unit" /></el-form-item>
        <el-row :gutter="12">
          <el-col :span="8"><el-form-item label="境内/境外"><el-select v-model="form.domestic_overseas"><el-option v-for="d in DOMESTIC_OPTIONS" :key="d" :label="d" :value="d" /></el-select></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="国别"><el-input v-model="form.country" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="GPS坐标"><el-input v-model="form.gps_input" placeholder="纬度,经度 或 Plus Code" /></el-form-item></el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible=false">取消</el-button>
        <el-button type="primary" @click="save">保存</el-button>
      </template>
    </el-dialog>

    <!-- Detail Dialog: plain table layout -->
    <el-dialog v-model="detailVisible" title="项目详情" width="950px" top="10px">
      <template v-if="detail">
        <div class="detail-scroll">

          <div class="card-group">
            <div class="card-title">基础信息</div>
            <table class="info-table"><tbody>
              <tr><td class="k">项目简称</td><td class="v">{{ detail.short_name }}</td><td class="k">合同编号</td><td class="v">{{ detail.contract_no || '-' }} <el-popover placement="bottom" :width="200" trigger="hover" @show="loadContracts"><template #reference><el-button size="small" text type="primary">📄 合同文件</el-button></template><div v-if="contracts.length"><div v-for="c in contracts" :key="c.path" style="padding:6px 0;cursor:pointer" @click="openFile(c.name)">{{ c.label }}</div></div><div v-else style="color:#999">未找到合同文件</div></el-popover></td><td class="k">项目类型</td><td class="v">{{ detail.project_type || '-' }}</td></tr>
              <tr><td class="k">项目名称</td><td class="v" colspan="5">{{ detail.project_name }}</td></tr>
              <tr><td class="k">实施单位</td><td class="v">{{ detail.implement_unit }}</td><td class="k">项目状态</td><td class="v"><el-tag :type="statusTag(detail.project_status)" size="small">{{ detail.project_status }}</el-tag></td><td class="k">境内/境外</td><td class="v">{{ detail.domestic_overseas || '-' }}</td></tr>
              <tr><td class="k">合同额(万元)</td><td class="v">{{ detail.contract_amount }}</td><td class="k">施工图预算(万元)</td><td class="v">{{ detail.budget_amount || '-' }}</td><td class="k">合同工期(月)</td><td class="v">{{ detail.contract_duration || '-' }}</td></tr>
              <tr><td class="k">合同施工范围</td><td class="v" colspan="5">{{ detail.contract_scope || '-' }}</td></tr>
              <tr><td class="k">项目重点及难点</td><td class="v" colspan="5">{{ detail.key_points || '-' }}</td></tr>
            </tbody></table>
          </div>

          <div class="card-group">
            <div class="card-title">地点 & 日期</div>
            <table class="info-table"><tbody>
              <tr><td class="k">国别</td><td class="v">{{ detail.country || '-' }}</td><td class="k">省</td><td class="v">{{ detail.province || '-' }}</td><td class="k">市</td><td class="v">{{ detail.city || '-' }}</td></tr>
              <tr><td class="k">详细地址</td><td class="v" colspan="5">{{ detail.address || '-' }}</td></tr>
              <tr><td class="k">合同开工</td><td class="v">{{ fmtDate(detail.contract_start_year, detail.contract_start_month) }}</td><td class="k">合同竣工</td><td class="v">{{ fmtDate(detail.contract_end_year, detail.contract_end_month) }}</td><td class="k">合同工期(月)</td><td class="v">{{ detail.contract_duration || '-' }}</td></tr>
              <tr><td class="k">实际开工</td><td class="v">{{ fmtDate(detail.actual_start_year, detail.actual_start_month) }}</td><td class="k">计划完工</td><td class="v">{{ fmtDate(detail.plan_end_year, detail.plan_end_month) }}</td><td class="k">实际工期(月)</td><td class="v">{{ detail.actual_duration || '-' }}</td></tr>
              <tr><td class="k">完工日期</td><td class="v">{{ fmtDate(detail.completion_year, detail.completion_month) }}</td><td class="k">GPS坐标</td><td class="v" colspan="3">{{ (detail.gps_lat || detail.gps_lng) ? fmtDMS(detail.gps_lat, detail.gps_lng) : '-' }}</td></tr>
            </tbody></table>
          </div>

          <div class="card-group">
            <div class="card-title">状态 & 进度</div>
            <table class="info-table"><tbody>
              <tr><td class="k">本月运行状态</td><td class="v">{{ detail.running_status || '-' }}</td><td class="k">施工进度</td><td class="v">{{ detail.progress_status || '-' }}</td><td class="k">完工百分比</td><td class="v">{{ detail.complete_percent || '-' }}</td></tr>
              <tr><td class="k">累计产值(万元)</td><td class="v">{{ detail.completed_output || '-' }}</td><td class="k">累计应收(万元)</td><td class="v">{{ detail.cum_receivable || '-' }}</td><td class="k">累计实收(万元)</td><td class="v">{{ detail.cum_received || '-' }}</td></tr>
              <tr><td class="k">欠付金额(万元)</td><td class="v">{{ detail.owed_amount || '-' }}</td><td class="k">管理人员</td><td class="v">{{ detail.personnel_mgmt || '-' }}</td><td class="k">劳务人员</td><td class="v">{{ detail.personnel_labor || '-' }}</td></tr>
              <tr><td class="k">异常/暂停原因</td><td class="v" colspan="5">{{ detail.abnormal_reason || '-' }}</td></tr>
              <tr><td class="k">进度完成概况</td><td class="v" colspan="5">{{ detail.progress_summary || '-' }}</td></tr>
              <tr><td class="k">存在的问题</td><td class="v" colspan="5">{{ detail.issues || '-' }}</td></tr>
            </tbody></table>
          </div>

          <div class="card-group">
            <div class="card-title">业主 / 设计 / 监理</div>
            <table class="info-table"><tbody>
              <tr><td class="k">业主单位</td><td class="v">{{ detail.owner_unit || '-' }}</td><td class="k">业主负责人</td><td class="v">{{ detail.owner_contact || '-' }}</td><td class="k">业主电话</td><td class="v">{{ detail.owner_phone || '-' }}</td></tr>
              <tr><td class="k">设计单位</td><td class="v">{{ detail.design_unit || '-' }}</td><td class="k">设计负责人</td><td class="v">{{ detail.design_contact || '-' }}</td><td class="k">设计电话</td><td class="v">{{ detail.design_phone || '-' }}</td></tr>
              <tr><td class="k">监理单位</td><td class="v">{{ detail.supervision_unit || '-' }}</td><td class="k">监理负责人</td><td class="v">{{ detail.supervision_contact || '-' }}</td><td class="k">监理电话</td><td class="v">{{ detail.supervision_phone || '-' }}</td></tr>
              <tr><td class="k">填表人</td><td class="v" colspan="5">{{ detail.reporter || '-' }}</td></tr>
            </tbody></table>
          </div>

          <div class="card-group">
            <div class="card-title">项目经理</div>
            <table class="info-table"><tbody>
              <tr><td class="k">合同约定</td><td class="v">{{ detail.pm_contract || '-' }}</td><td class="k">文件任命</td><td class="v">{{ detail.pm_appointed || '-' }}</td><td class="k">现场负责</td><td class="v">{{ detail.pm_onsite || '-' }}</td></tr>
              <tr><td class="k">电话</td><td class="v">{{ detail.pm_phone || '-' }}</td><td class="k">建造师</td><td class="v">{{ detail.pm_builder || '-' }}</td><td class="k">安全B证</td><td class="v">{{ detail.pm_safety_cert || '-' }}</td></tr>
            </tbody></table>
          </div>

          <div class="card-group">
            <div class="card-title">技术负责人</div>
            <table class="info-table"><tbody>
              <tr><td class="k">文件任命</td><td class="v">{{ detail.tech_lead_appointed || '-' }}</td><td class="k">现场负责</td><td class="v">{{ detail.tech_lead_onsite || '-' }}</td><td class="k">电话</td><td class="v">{{ detail.tech_lead_phone || '-' }}</td></tr>
              <tr><td class="k">职称</td><td class="v" colspan="5">{{ detail.tech_lead_title || '-' }}</td></tr>
            </tbody></table>
          </div>

          <div class="card-group">
            <div class="card-title">质量 / 安全 / 费控经理</div>
            <table class="info-table"><tbody>
              <tr><th></th><th>质量经理</th><th>安全经理/总监</th><th>费控经理</th></tr>
              <tr><td class="k">文件任命</td><td class="v">{{ detail.quality_mgr_appointed || '-' }}</td><td class="v">{{ detail.hse_mgr_appointed || '-' }}</td><td class="v">{{ detail.cost_mgr_appointed || '-' }}</td></tr>
              <tr><td class="k">现场负责</td><td class="v">{{ detail.quality_mgr_onsite || '-' }}</td><td class="v">{{ detail.hse_mgr_onsite || '-' }}</td><td class="v">{{ detail.cost_mgr_onsite || '-' }}</td></tr>
              <tr><td class="k">电话</td><td class="v">{{ detail.quality_mgr_phone || '-' }}</td><td class="v">{{ detail.hse_mgr_phone || '-' }}</td><td class="v">{{ detail.cost_mgr_phone || '-' }}</td></tr>
              <tr><td class="k">持证</td><td class="v">{{ detail.quality_mgr_cert || '-' }}</td><td class="v">{{ detail.hse_mgr_cert || '-' }}</td><td class="v">{{ detail.cost_mgr_cert || '-' }}</td></tr>
            </tbody></table>
          </div>

        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const projects = ref<any[]>([])
const loading = ref(false)
const conflictMode = ref('skip')
const search = ref('')
const filters = ref({ type: '', status: '', domestic: '', country: '' })
const TYPES  = ['EPC','PC','C']
const STATUSES = ['未开工','在建','停工','完工']
const DOMESTIC_OPTIONS = ['境内','境外']

const countries = ref<string[]>([])
const dialogVisible = ref(false)
const detailVisible = ref(false)
const detail = ref<any>(null)
const editingId = ref(0)
const fileInput = ref<HTMLInputElement | null>(null)

const emptyForm = () => ({
  short_name: '', contract_no: '', project_name: '', project_type: '',
  project_status: '在建', implement_unit: '', contract_amount: 0,
  domestic_overseas: '', country: '', gps_input: '',
})
const form = ref(emptyForm())

function statusTag(s: string) {
  const m: Record<string,string> = {'未开工':'warning','在建':'','停工':'danger','完工':'success'}
  return m[s]||''
}

function fmtDate(y:number,m:number) {
  if(!y) return '-'
  return `${y}-${String(m||1).padStart(2,'0')}`
}

function fmtMoney(v: number | string) {
  const n = Number(v)
  if (!n) return '0'
  return n.toLocaleString('en-US')
}

function fmtDMS(lat:number, lng:number) {
  const toDMS = (d: number, ns: boolean) => {
    const neg = d < 0; d = Math.abs(d)
    const deg = Math.floor(d)
    const min = Math.floor((d - deg) * 60)
    const sec = ((d - deg - min/60) * 3600).toFixed(1)
    const dir = ns ? (neg ? 'S' : 'N') : (neg ? 'W' : 'E')
    return `${deg}°${min}′${sec}″${dir}`
  }
  return toDMS(lat, true) + ' ' + toDMS(lng, false)
}

async function load() {
  loading.value=true
  try {
    const params: any = {}
    if (search.value) params.keyword = search.value
    if (filters.value.type) params.type = filters.value.type
    if (filters.value.status) params.status = filters.value.status
    if (filters.value.domestic) params.domestic_overseas = filters.value.domestic
    if (filters.value.country) params.country = filters.value.country
    const { data } = await axios.get('/api/projects', { params })
    projects.value = data.projects || []
    countries.value = data.countries || []
  } catch(e){ projects.value=[]; countries.value=[] }
  loading.value=false
}

function showDetail(row: any) { detail.value = row; detailVisible.value = true }

function showDialog(row: any) {
  if (row) {
    editingId.value = row.id
    form.value = { ...row, gps_input: row.gps_lat && row.gps_lng ? `${row.gps_lat}, ${row.gps_lng}` : '' }
  }
  else { editingId.value = 0; form.value = emptyForm() }
  dialogVisible.value = true
}

async function save() {
  try {
    const payload = { ...form.value }
    if (editingId.value) await axios.put('/api/projects/'+editingId.value, payload)
    else await axios.post('/api/projects', payload)
    dialogVisible.value=false; ElMessage.success('保存成功'); load()
  } catch(e:any){ ElMessage.error('保存失败: ' + (e.response?.data || e.message)) }
}

async function del(row: any) {
  try {
    await ElMessageBox.confirm('确认删除？','警告',{type:'warning'})
    await axios.delete('/api/projects/'+row.id)
    ElMessage.success('已删除'); load()
  } catch{}
}

function importExcel() {
  ElMessageBox.confirm('导入模式：覆盖将替换已有项目，跳过仅新增', '导入', { confirmButtonText: '覆盖', cancelButtonText: '跳过', type: 'info' }).then(() => { conflictMode.value = 'overwrite'; fileInput.value?.click() }).catch(() => { conflictMode.value = 'skip'; fileInput.value?.click() })
}
async function onFileChange(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  const fd = new FormData(); fd.append('file', file); fd.append('conflict', conflictMode.value)
  try {
    const { data } = await axios.post('/api/projects/import', fd)
    ElMessage.success(`导入 ${data.imported} 条，跳过 ${data.skipped} 条`)
    load()
  } catch(e:any){ ElMessage.error('导入失败') }
}

const contracts = ref<any[]>([])

async function loadContracts() {
  if (!detail.value?.short_name) return
  try {
    const { data } = await axios.get('/api/files/contracts?project=' + encodeURIComponent(detail.value.short_name))
    contracts.value = data || []
  } catch { contracts.value = [] }
}

async function openFile(fileName: string) {
  await axios.get('/api/files/view?project=' + encodeURIComponent(detail.value.short_name) + '&folder=01.Contract&file=' + encodeURIComponent(fileName))
}

async function exportExcel() {
  try {
    const { data } = await axios.get('/api/projects/export',{params:filters.value,responseType:'blob'})
    const url = URL.createObjectURL(data)
    const a = document.createElement('a'); a.href=url; a.download='projects.xlsx'; a.click()
  } catch{}
}

onMounted(load)
</script>

<style scoped>
.project-list { display: flex; flex-direction: column; gap: 12px; height: 100%; }
.toolbar { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
:deep(.el-table__row) { cursor: pointer; }
:deep(.col-amount .cell) { padding-right: 24px !important; }
.detail-scroll { max-height: 70vh; overflow-y: auto; padding-right: 4px; }
.card-group { margin-bottom: 12px; border: 1px solid #ebeef5; border-radius: 6px; overflow: hidden; }
.card-title { font-weight: 600; font-size: 14px; color: #303133; padding: 10px 16px; background: #fafafa; border-bottom: 1px solid #ebeef5; }
.info-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.info-table td, .info-table th { padding: 6px 10px; border: 1px solid #ebeef5; }
.info-table .k { background: #fafafa; color: #606266; width: 110px; white-space: nowrap; font-weight: 500; }
.info-table .v { color: #303133; word-break: break-all; }
.info-table th { background: #fafafa; font-weight: 600; text-align: center; }
</style>
