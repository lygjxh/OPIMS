<template>
  <div class="rd">
    <div class="page-head">
      <div>
        <h1 class="page-title">时限雷达</h1>
        <p class="page-desc">
          合同时效倒计时 · 按<strong>日历日</strong>计算 ·
          逾期即丧失索赔权，不可挽回（细则 5.1 / 6.3、附件 G 第十节）
        </p>
      </div>
      <div class="head-ops">
        <el-button type="primary" @click="openForm()">
          <el-icon><Plus /></el-icon>登记时效条款
        </el-button>
        <el-button @click="load" :loading="loading">
          <el-icon><Refresh /></el-icon>刷新
        </el-button>
      </div>
    </div>

    <!-- 我方将丧失权利的，单独置顶。这类损失无补救途径，不能混在普通列表里 -->
    <el-alert v-if="riskAlerts.length" type="error" show-icon :closable="false"
      class="risk-banner">
      <template #title>
        <strong>{{ riskAlerts.length }} 项「我方沉默将丧失权利」已进入提醒窗口</strong>
      </template>
      <ul class="risk-list">
        <li v-for="r in riskAlerts" :key="r.id + r.title">
          <span class="rb-proj">{{ r.project }}</span>
          {{ r.title }}
          <span class="rb-due">{{ dueText(r) }}</span>
        </li>
      </ul>
      <p class="risk-tip">
        此类条款届满后损失由我方自负、无补救途径，与「错过一次索赔机会」性质不同。
      </p>
    </el-alert>

    <div class="kpi-row">
      <div class="kpi danger"><span class="k-lb">已逾期</span><span class="k-v">{{ s.overdue }}</span></div>
      <div class="kpi danger"><span class="k-lb">今日到期／提醒</span><span class="k-v">{{ s.due }}</span></div>
      <div class="kpi warn"><span class="k-lb">紧急（&lt;7天）</span><span class="k-v">{{ s.urgent }}</span></div>
      <div class="kpi"><span class="k-lb">7–14 天</span><span class="k-v">{{ s.soon }}</span></div>
      <div class="kpi"><span class="k-lb">已关闭</span><span class="k-v">{{ s.closed }}</span></div>
      <div class="kpi"><span class="k-lb">在册时效</span><span class="k-v">{{ s.total }}<em>/{{ s.projects }}个项目</em></span></div>
    </div>

    <el-alert v-if="scanNote" :title="scanNote" type="info" show-icon :closable="false" />
    <el-alert v-if="warnings.length" type="warning" show-icon :closable="false"
      :title="`${warnings.length} 条附件 E 记录无法解析`">
      <ul class="warn-list"><li v-for="(w,i) in warnings" :key="i">{{ w }}</li></ul>
    </el-alert>

    <el-card>
      <template #header>
        <div class="card-head">
          <span>时效清单 · 共 {{ items.length }} 条</span>
          <div class="filters">
            <el-radio-group v-model="filter" size="small">
              <el-radio-button value="active">未办结</el-radio-button>
              <el-radio-button value="risk">仅我方风险</el-radio-button>
              <el-radio-button value="all">全部</el-radio-button>
            </el-radio-group>
          </div>
        </div>
      </template>

      <el-empty v-if="!shown.length"
        description="暂无时效记录。点右上角「登记时效条款」录入，或等项目部提交附件 E" />

      <table v-else class="tbl">
        <thead>
          <tr>
            <th>紧急度</th><th>方向</th><th>项目</th><th>事项</th>
            <th>代码</th><th>到期日</th><th>剩余</th><th>来源</th><th>状态</th><th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="it in shown" :key="it.source + it.id + it.project + it.title + it.due_date"
            :class="rowClass(it)">
            <td><span class="lv" :class="it.level">{{ levelText(it.level) }}</span></td>
            <td>
              <span class="dir" :class="it.direction">
                {{ it.direction === 'risk' ? '我方风险' : '索赔机会' }}
              </span>
            </td>
            <td class="nowrap">{{ it.project }}</td>
            <td>
              {{ it.title }}
              <span v-if="it.signal_code" class="sig">{{ it.signal_code }}</span>
              <div v-if="it.clause" class="sub-line">条款 {{ it.clause }}</div>
            </td>
            <td class="nowrap">
              {{ it.code }}
              <div v-if="it.code_name" class="sub-line">{{ it.code_name }}</div>
            </td>
            <td class="nowrap">{{ it.due_date }}</td>
            <td class="nowrap" :class="{ neg: it.remaining < 0 }">
              {{ it.level === 'closed' ? '—' : remainText(it) }}
            </td>
            <td class="nowrap"><span class="src">{{ srcText(it.source) }}</span></td>
            <td class="nowrap">{{ statusText(it.status) }}</td>
            <td class="nowrap">
              <template v-if="it.id">
                <el-button link type="primary" size="small" @click="openForm(it)">编辑</el-button>
                <el-button v-if="it.level !== 'closed'" link type="success" size="small"
                  @click="markDone(it)">办结</el-button>
                <el-button v-if="it.level !== 'closed'" link size="small"
                  @click="openNotClaim(it)">不构成索赔</el-button>
              </template>
              <span v-else class="only-e">仅附件 E</span>
            </td>
          </tr>
        </tbody>
      </table>

      <p class="foot-note">
        经复核不构成索赔的记录<strong>不会删除</strong>，只改状态并留存理由——
        它是我方已尽注意义务的证明（附件 G 十二.4）。
      </p>
    </el-card>

    <!-- 登记 / 编辑 -->
    <el-dialog v-model="formOpen" :title="form.id ? '编辑时效条款' : '登记时效条款'" width="640px">
      <el-form label-width="110px" class="frm">
        <el-form-item label="项目" required>
          <el-select v-model="form.project" filterable placeholder="选择项目" style="width:100%">
            <el-option v-for="p in projects" :key="p" :label="p" :value="p" />
          </el-select>
        </el-form-item>

        <el-form-item label="方向" required>
          <el-radio-group v-model="form.direction">
            <el-radio value="claim">索赔机会</el-radio>
            <el-radio value="risk">我方风险</el-radio>
          </el-radio-group>
          <div class="dir-hint" :class="form.direction">
            <template v-if="form.direction === 'risk'">
              <strong>我方沉默将使自己丧失权利</strong>（细则 6.3「视为认可」、附件 G G10.5）。
              损失由我方自负、无补救途径，系统会在到期前强制提醒。
            </template>
            <template v-else>
              业主的行为给了我方索赔机会，逾期未通知即丧失索赔权。
            </template>
          </div>
        </el-form-item>

        <el-form-item label="事项" required>
          <el-input v-model="form.title" placeholder="如：业主指令未在 14 日内书面反对即视为接受" />
        </el-form-item>

        <el-form-item label="时效代码">
          <el-select v-model="form.code" clearable placeholder="T1–T9" style="width:100%">
            <el-option v-for="(n, c) in codeNames" :key="c" :label="`${c} ${n}`" :value="c" />
          </el-select>
        </el-form-item>

        <el-form-item label="附件 G 信号">
          <el-select v-model="form.signal_code" clearable placeholder="缺失型信号 G10.1–G10.5"
            style="width:100%">
            <el-option v-for="(n, c) in signalNames" :key="c" :label="`${c} ${n}`" :value="c" />
          </el-select>
        </el-form-item>

        <el-form-item label="合同条款号">
          <el-input v-model="form.clause" placeholder="如 SC 20.1" />
        </el-form-item>

        <el-form-item label="起算日">
          <el-date-picker v-model="form.trigger_date" type="date" value-format="YYYY-MM-DD"
            placeholder="实际知道或应当知道之日" style="width:100%" />
        </el-form-item>

        <el-form-item label="时限天数">
          <el-input-number v-model="form.due_days" :min="0" :max="3650" />
          <span class="fld-hint">日历日，不扣周末与节假日（细则 5.1）</span>
        </el-form-item>

        <el-form-item label="到期日">
          <el-date-picker v-model="form.due_date" type="date" value-format="YYYY-MM-DD"
            placeholder="留空则由起算日 + 时限天数自动推算" style="width:100%" />
        </el-form-item>

        <el-form-item label="提前提醒">
          <el-input-number v-model="form.remind_days" :min="0" :max="90" />
          <span class="fld-hint">
            天。「我方风险」类留 0 时自动按 3 日（细则 6.3.2）
          </span>
        </el-form-item>

        <el-form-item label="备注">
          <el-input v-model="form.remark" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>

      <template #footer>
        <el-button @click="formOpen = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>

    <!-- 判为不构成索赔：必须写理由 -->
    <el-dialog v-model="ncOpen" title="判为「不构成索赔」" width="520px">
      <p class="nc-tip">
        记录不会删除，仅改状态并留存本次判断理由。<br>
        <strong>{{ ncItem?.project }}</strong> · {{ ncItem?.title }}
      </p>
      <el-input v-model="ncReason" type="textarea" :rows="3"
        placeholder="请写明不构成索赔的理由，如：经复核该指令属合同范围内工作，不产生工期影响" />
      <template #footer>
        <el-button @click="ncOpen = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="confirmNotClaim">确认</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import { Refresh, Plus } from '@element-plus/icons-vue'

const items = ref<any[]>([])
const warnings = ref<string[]>([])
const scanNote = ref('')
const projects = ref<string[]>([])
const loading = ref(false)
const saving = ref(false)
const filter = ref<'active' | 'risk' | 'all'>('active')
const s = ref<any>({ total: 0, overdue: 0, due: 0, urgent: 0, soon: 0, closed: 0, projects: 0, risk_alert: 0 })

// 与附件 E「代码说明」一致
const codeNames: Record<string, string> = {
  T1: '变更/延误通知（Change Notice）',
  T2: '初步变更令（PCO）提交',
  T3: '不可抗力通知',
  T4: '索赔详细报告',
  T5: '滚动 PCO 更新',
  T6: '业主答复期届满',
  T9: '其他合同专有时效',
}
// 附件 G 第十节 缺失型信号
const signalNames: Record<string, string> = {
  'G10.1': '业主未按期答复我方报审',
  'G10.2': '业主未按期答复 CN/PCO',
  'G10.3': '业主未按期完成其义务',
  'G10.4': 'TQ/RFI 超期未获答复',
  'G10.5': '我方「视为认可」期限届满 ⚠️',
}

const shown = computed(() => {
  if (filter.value === 'all') return items.value
  if (filter.value === 'risk') return items.value.filter(i => i.direction === 'risk')
  return items.value.filter(i => i.level !== 'closed')
})

const riskAlerts = computed(() =>
  items.value.filter(i => i.direction === 'risk' && (i.level === 'overdue' || i.level === 'due')))

function levelText(l: string) {
  return ({ overdue: '已逾期', due: '今日到期', urgent: '紧急', soon: '临近', normal: '正常', closed: '已关闭' } as any)[l] || l
}
function statusText(st: string) {
  return ({ open: '跟踪中', done: '已办结', not_claim: '不构成索赔' } as any)[st] || st || '跟踪中'
}
function srcText(src: string) {
  return ({ registry: '中心登记', attach_e: '附件 E', both: '登记+附件E' } as any)[src] || src
}
function remainText(it: any) {
  return it.remaining < 0 ? `逾期 ${-it.remaining} 天` : `${it.remaining} 天`
}
function dueText(it: any) {
  return it.remaining < 0 ? `已逾期 ${-it.remaining} 天` : `剩 ${it.remaining} 天（${it.due_date}）`
}
function rowClass(it: any) {
  if (it.level === 'closed') return 'r-closed'
  if (it.direction === 'risk' && (it.level === 'overdue' || it.level === 'due')) return 'r-risk'
  if (it.level === 'overdue' || it.level === 'due') return 'r-bad'
  return ''
}

async function load() {
  loading.value = true
  try {
    const { data } = await axios.get('/api/radar')
    items.value = data.items || []
    warnings.value = data.warnings || []
    scanNote.value = data.scan_note || ''
    s.value = data.summary || s.value
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || '加载失败')
  }
  loading.value = false
}

async function loadProjects() {
  try {
    const { data } = await axios.get('/api/projects', { params: { status: 'all' } })
    const list = Array.isArray(data) ? data : (data.items || data.projects || [])
    projects.value = list
      .filter((p: any) => p.project_status === '在建' && p.short_name)
      .map((p: any) => p.short_name)
  } catch { projects.value = [] }
}

// ---- 登记 / 编辑 ----
const formOpen = ref(false)
const form = ref<any>({})
function blankForm() {
  return {
    id: 0, project: '', code: '', signal_code: '', title: '', clause: '',
    direction: 'claim', trigger_date: '', due_days: 0, due_date: '',
    remind_days: 0, status: 'open', remark: '',
  }
}
function openForm(it?: any) {
  form.value = it ? { ...blankForm(), ...it } : blankForm()
  formOpen.value = true
}

async function save() {
  saving.value = true
  try {
    if (form.value.id) {
      await axios.put('/api/radar/registry/' + form.value.id, form.value)
    } else {
      await axios.post('/api/radar/registry', form.value)
    }
    ElMessage.success('已保存')
    formOpen.value = false
    await load()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || '保存失败')
  }
  saving.value = false
}

async function markDone(it: any) {
  saving.value = true
  try {
    await axios.put('/api/radar/registry/' + it.id, { ...it, status: 'done' })
    ElMessage.success('已办结')
    await load()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || '操作失败')
  }
  saving.value = false
}

// ---- 不构成索赔 ----
const ncOpen = ref(false)
const ncItem = ref<any>(null)
const ncReason = ref('')
function openNotClaim(it: any) {
  ncItem.value = it
  ncReason.value = ''
  ncOpen.value = true
}
async function confirmNotClaim() {
  if (!ncReason.value.trim()) {
    ElMessage.warning('请填写理由')
    return
  }
  saving.value = true
  try {
    await axios.put('/api/radar/registry/' + ncItem.value.id, {
      ...ncItem.value, status: 'not_claim', closed_reason: ncReason.value,
    })
    ElMessage.success('已记录')
    ncOpen.value = false
    await load()
  } catch (e: any) {
    ElMessage.error(e?.response?.data?.error || '操作失败')
  }
  saving.value = false
}

onMounted(() => { load(); loadProjects() })
</script>

<style scoped>
.rd { display: flex; flex-direction: column; gap: 14px; }
.page-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; }
.page-title { font-size: 22px; font-weight: 700; color: var(--c-text-strong); margin: 0; }
.page-desc { font-size: 13px; color: var(--c-text-muted); margin: 4px 0 0; }
.head-ops { display: flex; gap: 8px; flex-shrink: 0; }

.risk-banner :deep(.el-alert__content) { width: 100%; }
.risk-list { margin: 6px 0 0; padding-left: 18px; font-size: 13px; line-height: 1.8; }
.rb-proj { font-weight: 600; margin-right: 6px; }
.rb-due { color: #991b1b; font-weight: 600; margin-left: 6px; }
.risk-tip { margin: 8px 0 0; font-size: 12px; opacity: .85; }

.kpi-row { display: grid; grid-template-columns: repeat(6, 1fr); gap: 12px; }
.kpi {
  background: var(--c-surface); border: 1px solid var(--c-border);
  border-radius: var(--radius); padding: 12px 14px;
  display: flex; flex-direction: column; gap: 4px;
}
.kpi.danger { border-left: 3px solid #dc2626; }
.kpi.warn { border-left: 3px solid #d97706; }
.k-lb { font-size: 12px; color: var(--c-text-muted); }
.k-v { font-size: 22px; font-weight: 700; color: var(--c-text-strong); }
.k-v em { font-size: 11px; font-weight: 400; color: var(--c-text-muted); font-style: normal; margin-left: 3px; }

.card-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap; }

.tbl { width: 100%; border-collapse: collapse; font-size: 13px; }
.tbl th, .tbl td { padding: 9px 10px; text-align: left; border-bottom: 1px solid var(--c-border); vertical-align: top; }
.tbl th { font-size: 12px; color: var(--c-text-muted); font-weight: 600; white-space: nowrap; }
.nowrap { white-space: nowrap; }
.neg { color: #dc2626; font-weight: 600; }
.sub-line { font-size: 11px; color: var(--c-text-muted); margin-top: 2px; }
.r-risk { background: #fef2f2; }
.r-bad { background: #fffbeb; }
.r-closed { opacity: .55; }

/* 紧急度与方向都不能只靠颜色区分，一律带文字 */
.lv { font-size: 11.5px; padding: 2px 7px; border-radius: 5px; white-space: nowrap; }
.lv.overdue { background: #fee2e2; color: #991b1b; font-weight: 600; }
.lv.due { background: #fee2e2; color: #991b1b; font-weight: 600; }
.lv.urgent { background: #fef3c7; color: #92400e; }
.lv.soon { background: #e0e7ff; color: #3730a3; }
.lv.normal { background: var(--c-bg); color: var(--c-text-muted); }
.lv.closed { background: var(--c-bg); color: var(--c-text-muted); }

.dir { font-size: 11.5px; padding: 2px 7px; border-radius: 5px; white-space: nowrap; }
.dir.risk { background: #7f1d1d; color: #fff; font-weight: 600; }
.dir.claim { background: var(--c-bg); color: var(--c-text-muted); }

.sig { font-size: 11px; color: #7f1d1d; background: #fee2e2; padding: 1px 5px; border-radius: 4px; margin-left: 5px; }
.src { font-size: 11.5px; color: var(--c-text-muted); }
.only-e { font-size: 11.5px; color: var(--c-text-muted); }

.warn-list { margin: 4px 0 0; padding-left: 18px; font-size: 12.5px; line-height: 1.7; }
.foot-note { margin: 12px 0 0; font-size: 12px; color: var(--c-text-muted); }

.frm { max-height: 58vh; overflow-y: auto; padding-right: 6px; }
.fld-hint { font-size: 11.5px; color: var(--c-text-muted); margin-left: 8px; }
.dir-hint {
  font-size: 12px; line-height: 1.6; margin-top: 6px;
  padding: 7px 10px; border-radius: 6px; background: var(--c-bg); color: var(--c-text-muted);
}
.dir-hint.risk { background: #fee2e2; color: #7f1d1d; }
.nc-tip { font-size: 13px; line-height: 1.7; margin: 0 0 10px; }
</style>
