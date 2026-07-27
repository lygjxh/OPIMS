<template>
  <div class="project-list">
    <div class="toolbar">
      <el-input v-model="search" placeholder="搜索" clearable style="width:200px" @keyup.enter="load" />
      <el-select v-model="filters.type" placeholder="项目类型" clearable style="width:100px" @change="load">
        <el-option v-for="t in TYPES" :key="t" :label="t" :value="t" />
      </el-select>
      <el-select v-model="filters.status" placeholder="项目状态" clearable style="width:130px" @change="load">
        <el-option label="全部状态" value="all" />
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

    <div class="list-meta">
      <span>共 <b>{{ projects.length }}</b> 个项目<template v-if="projects.length">，合同额合计 <b>{{ fmtYi(sumAmount) }}</b> 亿元</template></span>
      <span v-if="activeFilterText" class="filter-chip">
        {{ activeFilterText }}
        <el-icon class="chip-x" @click="clearFilters"><Close /></el-icon>
      </span>
    </div>

    <el-table :data="pagedProjects" stripe v-loading="loading" style="flex:1"
      :default-sort="{ prop: 'contract_amount', order: 'descending' }"
      @row-click="showDetail">
      <el-table-column prop="country" label="国别" width="106" align="center" fixed sortable />
      <el-table-column prop="short_name" label="项目简称" min-width="170" show-overflow-tooltip />
      <el-table-column prop="implement_unit" label="实施单位" width="112" align="center" show-overflow-tooltip />
      <el-table-column prop="project_name" label="项目名称" min-width="230" show-overflow-tooltip />
      <el-table-column prop="contract_amount" label="合同额" width="130" align="right"
        sortable :sort-method="(a:any,b:any)=>Number(a.contract_amount)-Number(b.contract_amount)"
        class-name="col-amount">
        <template #default="{ row }">
          <span class="amt">{{ fmtAmount(row.contract_amount) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="合同开工" width="96" align="center">
        <template #default="{ row }">{{ fmtDate(row.contract_start_year, row.contract_start_month) }}</template>
      </el-table-column>
      <el-table-column label="合同竣工" width="96" align="center">
        <template #default="{ row }">{{ fmtDate(row.contract_end_year, row.contract_end_month) }}</template>
      </el-table-column>
      <el-table-column prop="project_status" label="状态" width="96" align="center" sortable>
        <template #default="{ row }">
          <span class="st" :style="{ '--st': statusColor(row.project_status) }">
            <i :class="'st-mk st-' + statusShape(row.project_status)"></i>{{ row.project_status || '-' }}
          </span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="112" fixed="right" align="center" class-name="col-ops">
        <template #default="{ row }">
          <div class="ops">
            <button type="button" class="op-btn" @click.stop="showDialog(row)">编辑</button>
            <span class="op-sep"></span>
            <button type="button" class="op-btn danger" @click.stop="del(row)">删除</button>
          </div>
        </template>
      </el-table-column>
    </el-table>

    <el-pagination v-if="projects.length > pageSize" class="pager"
      layout="total, sizes, prev, pager, next, jumper"
      :total="projects.length" :page-size="pageSize" :current-page="page"
      :page-sizes="[20, 50, 100, 200]"
      @current-change="(p:number)=>page=p"
      @size-change="(s:number)=>{ pageSize=s; page=1 }" />

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
              <tr><td class="k">国别</td><td class="v">
                <template v-if="detail.country">
                  <a v-if="hasPolicy(detail.country)" class="country-link"
                     :title="`查看 ${detail.country} 出入境政策`"
                     @click="gotoPolicy(detail.country)">
                    {{ detail.country }}<el-icon><Right /></el-icon>
                  </a>
                  <template v-else>{{ detail.country }}</template>
                </template>
                <template v-else>-</template>
              </td><td class="k">省</td><td class="v">{{ detail.province || '-' }}</td><td class="k">市</td><td class="v">{{ detail.city || '-' }}</td></tr>
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

    <!-- 导入冲突处理 -->
    <el-dialog v-model="conflictVisible" title="导入冲突" width="700px">
      <p style="color:#606266;margin-bottom:12px">以下项目与已有数据冲突（共 {{ conflictItems.length }} 条），请选择处理方式：</p>
      <div v-for="(c, i) in conflictItems" :key="i" class="conflict-item">
        <div class="ci-h">冲突 {{ i+1 }}：{{ c.short_name }}</div>
        <table class="ci-table" v-if="c.diffs && c.diffs.length">
          <tr><th>字段</th><th>已有值</th><th>新值</th></tr>
          <tr v-for="d in c.diffs" :key="d.field">
            <td>{{ d.field }}</td><td class="old">{{ d.old }}</td><td class="new">{{ d.new }}</td>
          </tr>
        </table>
        <div v-else style="color:#999;font-size:13px">（字段无差异）</div>
      </div>
      <template #footer>
        <el-button @click="doImportAfterConflict('skip')">跳过</el-button>
        <el-button type="primary" @click="doImportAfterConflict('overwrite')">覆盖</el-button>
        <el-button type="success" @click="doImportAfterConflict('keep_both')">保留两者</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Close, Right } from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()

/* 有出入境政策档案的国别（用于把项目详情里的国别做成跳转链接）。
   只对确实有档案的国别加链接，避免点进去是「未找到」。 */
const policyCountries = ref<Set<string>>(new Set())
const hasPolicy = (c: string) => policyCountries.value.has(c)
function gotoPolicy(country: string) {
  router.push({ path: '/country-profile', query: { country } })
}
async function loadPolicyCountries() {
  try {
    const { data } = await axios.get('/api/policy/countries')
    policyCountries.value = new Set((data.countries || []).map((c: any) => c.country))
  } catch { /* 政策库未配置时静默跳过，不影响项目清单本身 */ }
}
const projects = ref<any[]>([])
const page = ref(1)
const pageSize = ref(50)
const loading = ref(false)
const search = ref('')
const filters = ref({ type: '', status: '', domestic: '', country: '' })
const TYPES  = ['EPC','PC','C']
const STATUSES = ['未开工','在建','停工','完工']
const DOMESTIC_OPTIONS = ['境内','境外']

const countries = ref<string[]>([])
const dialogVisible = ref(false)
const detailVisible = ref(false)
const conflictVisible = ref(false)
const conflictItems = ref<any[]>([])
const pendingFile = ref<File | null>(null)
const detail = ref<any>(null)
const editingId = ref(0)
const fileInput = ref<HTMLInputElement | null>(null)

const emptyForm = () => ({
  short_name: '', contract_no: '', project_name: '', project_type: '',
  project_status: '在建', implement_unit: '', contract_amount: 0,
  domestic_overseas: '', country: '', gps_input: '',
})
const form = ref(emptyForm())

/* 状态 → 颜色 + 形状（形状是颜色之外的第二重编码，红绿色盲同样可辨） */
const STATUS_STYLE: Record<string, { color: string; shape: string }> = {
  '在建':   { color: '#2a78d6', shape: 'dot' },
  '未开工': { color: '#eda100', shape: 'ring' },
  '停工':   { color: '#d03b3b', shape: 'square' },
  '完工':   { color: '#0ca30c', shape: 'small' },
}
const statusColor = (s: string) => STATUS_STYLE[s]?.color || '#94a3b8'
const statusShape = (s: string) => STATUS_STYLE[s]?.shape || 'dot'
function statusTag(s: string) {
  const m: Record<string,string> = {'未开工':'warning','在建':'','停工':'danger','完工':'success'}
  return m[s]||''
}

/* 分页 + 统计 */
const pagedProjects = computed(() => {
  const start = (page.value - 1) * pageSize.value
  return projects.value.slice(start, start + pageSize.value)
})
const sumAmount = computed(() =>
  projects.value.reduce((s, p) => s + (Number(p.contract_amount) || 0), 0))

const activeFilterText = computed(() => {
  const f = filters.value
  const parts: string[] = []
  if (f.status) parts.push('状态：' + (f.status === 'all' ? '全部' : f.status))
  if (f.type) parts.push('类型：' + f.type)
  if (f.country) parts.push('国别：' + f.country)
  if (f.domestic) parts.push(f.domestic)
  if (search.value) parts.push('关键词：' + search.value)
  return parts.join(' · ')
})
function clearFilters() {
  filters.value = { type: '', status: '', domestic: '', country: '' }
  search.value = ''
  load()
}

function fmtDate(y:number,m:number) {
  if(!y) return '-'
  return `${y}-${String(m||1).padStart(2,'0')}`
}

/** 万元 → 亿元（大额更易读）；不足 1 亿仍显示万元 */
function fmtAmount(v: number | string) {
  const n = Number(v) || 0
  if (n === 0) return '-'
  if (n >= 10000) {
    const yi = n / 10000
    return (yi >= 100 ? yi.toFixed(0) : yi >= 10 ? yi.toFixed(1) : yi.toFixed(2)) + ' 亿'
  }
  return n.toLocaleString('en-US') + ' 万'
}
function fmtYi(wan: number) {
  const yi = wan / 10000
  return yi >= 100 ? yi.toFixed(0) : yi >= 10 ? yi.toFixed(1) : yi.toFixed(2)
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
  page.value = 1
  loading.value=false
}

// 支持从首页 KPI 卡片跳转带来的 ?status=xxx
function applyRouteQuery() {
  const s = route.query.status
  filters.value.status = typeof s === 'string' ? s : ''
}
watch(() => route.query.status, () => { applyRouteQuery(); load() })

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
  fileInput.value?.click()
}
async function onFileChange(e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  pendingFile.value = file
  // First, try with skip mode to detect conflicts
  const fd = new FormData(); fd.append('file', file); fd.append('conflict', 'skip')
  try {
    const { data } = await axios.post('/api/projects/import', fd)
    if (data.conflicts && data.conflicts.length > 0) {
      conflictItems.value = data.conflicts
      conflictVisible.value = true
    } else {
      ElMessage.success(`导入 ${data.imported} 条${data.skipped ? '，跳过 ' + data.skipped + ' 条' : ''}`)
      load()
    }
  } catch(e:any){ ElMessage.error('导入失败: ' + (e.response?.data || '')) }
}
async function doImportAfterConflict(mode: string) {
  if (!pendingFile.value) return
  const fd = new FormData(); fd.append('file', pendingFile.value); fd.append('conflict', mode)
  try {
    const { data } = await axios.post('/api/projects/import', fd)
    ElMessage.success(`导入 ${data.imported} 条${data.skipped ? '，跳过 ' + data.skipped + ' 条' : ''}`)
    conflictVisible.value = false
    load()
  } catch(e:any){ ElMessage.error('导入失败: ' + (e.response?.data || '')) }
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

onMounted(() => { applyRouteQuery(); load(); loadPolicyCountries() })
</script>

<style scoped>
.project-list { display: flex; flex-direction: column; gap: 12px; height: 100%; }
.toolbar { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
:deep(.el-table__row) { cursor: pointer; }
:deep(.col-amount .cell) { padding-right: 14px !important; }

/* 统计条 */
.list-meta {
  display: flex; align-items: center; gap: 12px;
  font-size: 12.5px; color: var(--c-text-muted); flex-wrap: wrap;
}
.list-meta b { color: var(--c-text-strong); font-size: 14px; }
.filter-chip {
  display: inline-flex; align-items: center; gap: 6px;
  background: #eef4ff; border: 1px solid var(--c-border);
  color: var(--c-primary-700); border-radius: 20px; padding: 3px 10px; font-size: 12px;
}
.chip-x { cursor: pointer; font-size: 12px; }
.chip-x:hover { color: var(--c-status-suspended); }

/* 合同额：等宽数字便于纵向比对 */
.amt { font-variant-numeric: tabular-nums; font-weight: 600; color: var(--c-text-strong); }

/* 状态：色块 + 文字（不单靠颜色） */
.st { display: inline-flex; align-items: center; gap: 6px; font-size: 12.5px; color: var(--c-text); }
.st-mk { width: 9px; height: 9px; flex-shrink: 0; background: var(--st); }
.st-dot { border-radius: 50%; }
.st-ring { border-radius: 50%; background: #fff; border: 2.5px solid var(--st); }
.st-square { border-radius: 2px; }
.st-small { border-radius: 50%; width: 6px; height: 6px; }

/* 操作列：单行不换行，紧凑排布 */
:deep(.col-ops .cell) { padding-left: 6px !important; padding-right: 6px !important; }
.ops { display: flex; align-items: center; justify-content: center; gap: 6px; white-space: nowrap; }
.op-btn {
  border: 0; background: none; padding: 2px 1px; cursor: pointer;
  font: inherit; font-size: 12.5px; line-height: 1.2;
  color: var(--c-primary); border-radius: 4px; transition: color .15s;
}
.op-btn:hover { color: var(--c-primary-600); text-decoration: underline; }
.op-btn.danger { color: var(--c-status-suspended); }
.op-btn.danger:hover { color: #a82c2c; }
.op-btn:focus-visible { outline: 2px solid var(--c-primary); outline-offset: 1px; }
.op-sep { width: 1px; height: 11px; background: var(--c-border-strong); flex-shrink: 0; }

/* 项目详情里的国别跳转链接：仅在该国有政策档案时出现 */
.country-link {
  display: inline-flex; align-items: center; gap: 2px;
  color: var(--c-primary); cursor: pointer; font-weight: 500;
}
.country-link:hover { text-decoration: underline; }
.country-link .el-icon { font-size: 12px; }

.pager { justify-content: flex-end; padding-top: 4px; }
.detail-scroll { max-height: 70vh; overflow-y: auto; padding-right: 4px; }
.card-group { margin-bottom: 12px; border: 1px solid #ebeef5; border-radius: 6px; overflow: hidden; }
.card-title { font-weight: 600; font-size: 14px; color: #303133; padding: 10px 16px; background: #fafafa; border-bottom: 1px solid #ebeef5; }
.info-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.info-table td, .info-table th { padding: 6px 10px; border: 1px solid #ebeef5; }
.info-table .k { background: #fafafa; color: #606266; width: 110px; white-space: nowrap; font-weight: 500; }
.info-table .v { color: #303133; word-break: break-all; }
.info-table th { background: #fafafa; font-weight: 600; text-align: center; }
.conflict-item { margin-bottom: 12px; border: 1px solid #ebeef5; border-radius: 6px; padding: 10px; }
.ci-h { font-weight: 600; margin-bottom: 6px; }
.ci-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.ci-table th, .ci-table td { padding: 4px 8px; border: 1px solid #ebeef5; text-align: left; }
.ci-table .old { color: #d03b3b; }
.ci-table .new { color: #2a78d6; }
</style>
