<template>
  <div class="ov">
    <div class="page-head">
      <div>
        <h1 class="page-title">进度管理</h1>
        <p class="page-desc">{{ today }} · 今天必须处理的事排在最前</p>
      </div>
      <el-button @click="loadAll" :loading="anyLoading">
        <el-icon><Refresh /></el-icon>重新扫描
      </el-button>
    </div>

    <!-- ========== 一、红线区 ========== -->
    <section class="zone">
      <div class="zone-head">
        <span class="zone-tag bad">红线</span>
        <h2 class="zone-title">今天必须动</h2>
        <span class="zone-sub">逾期即丧失权利，无补救途径</span>
      </div>

      <div v-if="radar.loading" class="placeholder">扫描中…</div>
      <div v-else-if="radar.err" class="placeholder err">{{ radar.err }}</div>
      <template v-else>
        <div class="redline-row">
          <button type="button" class="rl" :class="{ hot: r.overdue > 0 }" @click="go('/progress/radar')">
            <span class="rl-v">{{ r.overdue }}</span>
            <span class="rl-lb">已逾期</span>
          </button>
          <button type="button" class="rl" :class="{ hot: r.due > 0 }" @click="go('/progress/radar')">
            <span class="rl-v">{{ r.due }}</span>
            <span class="rl-lb">今日到期／提醒</span>
          </button>
          <button type="button" class="rl" :class="{ hot: r.risk_alert > 0 }" @click="go('/progress/radar')">
            <span class="rl-v">{{ r.risk_alert }}</span>
            <span class="rl-lb">我方将丧失权利</span>
            <span class="rl-note">视为认可类</span>
          </button>
          <button type="button" class="rl" @click="go('/progress/radar')">
            <span class="rl-v">{{ r.urgent }}</span>
            <span class="rl-lb">紧急（&lt;7天）</span>
          </button>
          <div class="rl muted" title="需先建「偏差与纠偏」模块才能统计">
            <span class="rl-v">—</span>
            <span class="rl-lb">红灯超 30 日无纠偏</span>
            <span class="rl-note">待建</span>
          </div>
        </div>

        <!-- 逾期和今日到期的逐条点名，驾驶舱上直接能看到是哪个项目 -->
        <ul v-if="hotItems.length" class="hot-list">
          <li v-for="(it, i) in hotItems" :key="i" :class="{ risk: it.direction === 'risk' }">
            <span class="hl-lv">{{ it.remaining < 0 ? `逾期${-it.remaining}天` : '今日' }}</span>
            <span class="hl-proj">{{ it.project }}</span>
            <span class="hl-title">{{ it.title }}</span>
            <span v-if="it.direction === 'risk'" class="hl-tag">我方风险</span>
          </li>
        </ul>
        <p v-else class="all-clear">✓ 当前没有逾期或今日到期的时效事项</p>
      </template>
    </section>

    <!-- ========== 二、本期例行 ========== -->
    <section class="zone">
      <div class="zone-head">
        <span class="zone-tag">例行</span>
        <h2 class="zone-title">本期例行</h2>
        <span class="zone-sub">{{ periodLabel }}</span>
      </div>

      <div class="grid">
        <button type="button" class="card" :class="check.tone" @click="go('/progress/check')">
          <div class="c-head">
            <el-icon class="c-ic"><Document /></el-icon>
            <span class="c-title">月度报送</span>
            <span v-if="check.tag" class="c-tag" :class="check.tone">{{ check.tag }}</span>
          </div>
          <div v-if="check.loading" class="c-body muted">扫描中…</div>
          <div v-else-if="check.err" class="c-body"><span class="c-err">{{ check.err }}</span></div>
          <template v-else>
            <div class="c-body">
              <span class="c-main">{{ check.main }}</span><span class="c-unit">{{ check.unit }}</span>
            </div>
            <div class="c-sub">
              <span v-for="(x, i) in check.subs" :key="i" class="c-chip" :class="x.tone">
                {{ x.label }} <strong>{{ x.value }}</strong>
              </span>
            </div>
          </template>
          <div class="c-foot">{{ check.foot }}</div>
        </button>

        <button type="button" class="card" :class="ind.tone" @click="go('/progress/indicators')">
          <div class="c-head">
            <el-icon class="c-ic"><TrendCharts /></el-icon>
            <span class="c-title">进度指标</span>
            <span v-if="ind.tag" class="c-tag" :class="ind.tone">{{ ind.tag }}</span>
          </div>
          <div v-if="ind.loading" class="c-body muted">扫描中…</div>
          <div v-else-if="ind.err" class="c-body"><span class="c-err">{{ ind.err }}</span></div>
          <template v-else>
            <div class="c-body">
              <span class="c-main">{{ ind.main }}</span><span class="c-unit">{{ ind.unit }}</span>
            </div>
            <div class="c-sub">
              <span v-for="(x, i) in ind.subs" :key="i" class="c-chip" :class="x.tone">
                {{ x.label }} <strong>{{ x.value }}</strong>
              </span>
            </div>
          </template>
          <div class="c-foot">{{ ind.foot }}</div>
        </button>

        <button type="button" class="card" :class="eot.tone" @click="go('/progress/eot')">
          <div class="c-head">
            <el-icon class="c-ic"><Money /></el-icon>
            <span class="c-title">工期索赔</span>
            <span v-if="eot.tag" class="c-tag" :class="eot.tone">{{ eot.tag }}</span>
          </div>
          <div v-if="eot.loading" class="c-body muted">扫描中…</div>
          <div v-else-if="eot.err" class="c-body"><span class="c-err">{{ eot.err }}</span></div>
          <template v-else>
            <div class="c-body">
              <span class="c-main">{{ eot.main }}</span><span class="c-unit">{{ eot.unit }}</span>
            </div>
            <div class="c-sub">
              <span v-for="(x, i) in eot.subs" :key="i" class="c-chip" :class="x.tone">
                {{ x.label }} <strong>{{ x.value }}</strong>
              </span>
            </div>
          </template>
          <div class="c-foot">{{ eot.foot }}</div>
        </button>
      </div>
    </section>

    <!-- ========== 三、考核视图 ========== -->
    <section class="zone">
      <div class="zone-head">
        <span class="zone-tag">考核</span>
        <h2 class="zone-title">考核指标</h2>
        <span class="zone-sub">细则 7.1 · 季度综合、年度总评</span>
      </div>

      <table class="kpi-tbl">
        <thead>
          <tr><th>考核指标</th><th>权重</th><th>目标值</th><th>当前</th><th>得分</th><th>数据来源</th></tr>
        </thead>
        <tbody>
          <tr v-for="k in appraisal" :key="k.name" :class="{ pending: k.value === null }">
            <td>{{ k.name }}</td>
            <td class="num">{{ k.weight }}%</td>
            <td class="num">{{ k.target }}</td>
            <td class="num">{{ k.value === null ? '—' : k.display }}</td>
            <td class="num"><strong>{{ k.value === null ? '—' : k.score.toFixed(1) }}</strong></td>
            <td class="src">{{ k.source }}</td>
          </tr>
        </tbody>
      </table>

      <div class="downgrade">
        <span class="dg-label">一票降档</span>
        <span class="dg-item" :class="{ hit: r.overdue > 0 }">
          Time Bar 逾期致丧失索赔权 · {{ r.overdue > 0 ? `已触发 ${r.overdue} 项` : '未触发' }}
        </span>
        <span class="dg-item muted">月报连续两月未交 · 待建</span>
        <span class="dg-item muted">红灯 30 日无纠偏方案 · 待建</span>
      </div>
    </section>

    <el-alert v-if="errs.length" type="info" show-icon :closable="false"
      :title="`${errs.length} 项数据暂时取不到`">
      <ul class="warn-list"><li v-for="(e, i) in errs" :key="i">{{ e }}</li></ul>
      <p class="warn-tip">多半是根目录未设置、对应附件尚未提交，或该期次无数据。</p>
    </el-alert>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import { Refresh, TrendCharts, Document, Money } from '@element-plus/icons-vue'

const router = useRouter()
const today = ref('')
const periodLabel = ref('')

type Sub = { label: string; value: string | number; tone?: string }
function blank() {
  return { loading: true, err: '', main: '—' as string | number, unit: '',
    subs: [] as Sub[], tone: '', tag: '', foot: '' }
}

const radar = reactive({ loading: true, err: '' })
const r = ref<any>({ overdue: 0, due: 0, urgent: 0, soon: 0, closed: 0, total: 0, projects: 0, risk_alert: 0 })
const hotItems = ref<any[]>([])

const check = reactive(blank())
const ind = reactive(blank())
const eot = reactive(blank())

const anyLoading = computed(() => radar.loading || check.loading || ind.loading || eot.loading)
const errs = computed(() => {
  const out: string[] = []
  if (radar.err) out.push('时限雷达：' + radar.err)
  if (check.err) out.push('月度报送：' + check.err)
  if (ind.err) out.push('进度指标：' + ind.err)
  if (eot.err) out.push('工期索赔：' + eot.err)
  return out
})

// 细则 7.1 考核指标体系。目前只有「月报按时提交率」有数据源，
// 其余三项依赖尚未建设的模块，宁可显示「待建」也不填假数字。
const appraisal = computed(() => {
  const compliance = check.err ? null : complianceRate.value
  return [
    { name: '里程碑完成率', weight: 30, target: '≥95%', value: null, display: '', score: 0,
      source: '待建：计划与考核' },
    { name: '总体进度偏差 SPI', weight: 25, target: '≥0.95', value: spi.value,
      display: spi.value === null ? '' : spi.value.toFixed(2),
      score: (spi.value || 0) * 25, source: '进度指标（附件 B-1）' },
    { name: '季度产值完成率', weight: 10, target: '≥95%', value: null, display: '', score: 0,
      source: '待建：计划与考核' },
    { name: '月报按时提交率', weight: 10, target: '100%', value: compliance,
      display: compliance === null ? '' : (compliance * 100).toFixed(0) + '%',
      score: (compliance || 0) * 10, source: '月度报送' },
    { name: 'EOT 时效合规率', weight: 25, target: '100%', value: null, display: '', score: 0,
      source: '待建：需 PCO 应交/实交口径' },
  ]
})
const complianceRate = ref<number | null>(null)
const spi = ref<number | null>(null)

function go(path: string) { router.push(path) }
function errText(e: any): string {
  return e?.response?.data?.error || e?.message || '取数失败'
}

async function loadRadar() {
  radar.loading = true; radar.err = ''
  try {
    const { data } = await axios.get('/api/radar')
    r.value = data.summary || r.value
    today.value = data.today || ''
    hotItems.value = (data.items || [])
      .filter((i: any) => i.level === 'overdue' || i.level === 'due')
      .slice(0, 8)
  } catch (e) { radar.err = errText(e) }
  radar.loading = false
}

async function loadCheck() {
  check.loading = true; check.err = ''
  try {
    const { data } = await axios.get('/api/progress/check')
    const d = data.summary || {}
    const missing = d.missing || 0
    complianceRate.value = d.compliance ?? null
    check.main = d.compliance != null ? (d.compliance * 100).toFixed(0) : '—'
    check.unit = '% 按时率'
    check.subs = [
      { label: '按时', value: d.submitted || 0, tone: 'ok' },
      { label: '迟交', value: d.late || 0, tone: (d.late || 0) > 0 ? 'warn' : '' },
      { label: '缺交', value: missing, tone: missing > 0 ? 'bad' : '' },
    ]
    check.tone = missing > 0 ? 'bad' : (d.late || 0) > 0 ? 'warn' : ''
    check.tag = missing > 0 ? `${missing} 项缺交` : ''
    check.foot = `${data.period} · 在建项目 ${d.project_count || 0} 个`
    periodLabel.value = data.period || ''
  } catch (e) { check.err = errText(e); complianceRate.value = null }
  check.loading = false
}

async function loadIndicators() {
  ind.loading = true; ind.err = ''
  try {
    const { data } = await axios.get('/api/progress/indicators')
    const d = data.summary || {}
    const attention = (d.red || 0) + (d.orange || 0)
    // spi_count 为 0 表示没有一个项目能算出 SPI，此时 avg_spi 是 0 而非真实均值
    spi.value = (d.spi_count || 0) > 0 ? d.avg_spi : null
    ind.main = d.total || 0
    ind.unit = '个项目已报'
    ind.subs = [
      { label: '红', value: d.red || 0, tone: (d.red || 0) > 0 ? 'bad' : '' },
      { label: '橙', value: d.orange || 0, tone: (d.orange || 0) > 0 ? 'warn' : '' },
      { label: '黄', value: d.yellow || 0 },
      { label: '绿', value: d.green || 0, tone: 'ok' },
    ]
    ind.tone = (d.red || 0) > 0 ? 'bad' : attention > 0 ? 'warn' : ''
    ind.tag = attention > 0 ? `${attention} 项需关注` : ''
    ind.foot = `${data.period} · 在建项目 ${data.project_count || 0} 个`
  } catch (e) { ind.err = errText(e); spi.value = null }
  ind.loading = false
}

async function loadEOT() {
  eot.loading = true; eot.err = ''
  try {
    const { data } = await axios.get('/api/eot')
    const d = data.summary || {}
    const overdue = (d.overdue_cn || 0) + (d.overdue_pco || 0)
    eot.main = d.total || 0
    eot.unit = '项索赔在跟踪'
    eot.subs = [
      { label: 'CN 逾期', value: d.overdue_cn || 0, tone: (d.overdue_cn || 0) > 0 ? 'bad' : '' },
      { label: 'PCO 逾期', value: d.overdue_pco || 0, tone: (d.overdue_pco || 0) > 0 ? 'bad' : '' },
      { label: '累计请求', value: `${d.total_days || 0} 天` },
      { label: '已关闭', value: d.closed_count || 0 },
    ]
    eot.tone = overdue > 0 ? 'bad' : ''
    eot.tag = overdue > 0 ? `${overdue} 项已逾期` : ''
    eot.foot = data.warning || `覆盖 ${d.projects || 0} 个在建项目`
  } catch (e) { eot.err = errText(e) }
  eot.loading = false
}

// 四个接口都要扫云盘，并发发出、各自独立降级，
// 不要合成一个 Promise.all —— 一个目录读不到会让整页空白。
function loadAll() {
  loadRadar(); loadCheck(); loadIndicators(); loadEOT()
}
onMounted(loadAll)
</script>

<style scoped>
.ov { display: flex; flex-direction: column; gap: 18px; }
.page-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; }
.page-title { font-size: 22px; font-weight: 700; color: var(--c-text-strong); margin: 0; }
.page-desc { font-size: 13px; color: var(--c-text-muted); margin: 4px 0 0; }

.zone { display: flex; flex-direction: column; gap: 10px; }
.zone-head { display: flex; align-items: baseline; gap: 9px; flex-wrap: wrap; }
.zone-title { font-size: 15px; font-weight: 700; color: var(--c-text-strong); margin: 0; }
.zone-sub { font-size: 12px; color: var(--c-text-muted); }
.zone-tag {
  font-size: 11px; padding: 2px 8px; border-radius: 5px;
  background: var(--c-bg); color: var(--c-text-muted); font-weight: 600;
}
.zone-tag.bad { background: #7f1d1d; color: #fff; }

/* ---- 红线区 ---- */
.redline-row { display: grid; grid-template-columns: repeat(5, 1fr); gap: 12px; }
.rl {
  display: flex; flex-direction: column; gap: 3px; align-items: flex-start;
  padding: 14px 16px; text-align: left; font: inherit; cursor: pointer;
  background: var(--c-surface); border: 1px solid var(--c-border);
  border-left: 3px solid var(--c-border-strong);
  border-radius: var(--radius); transition: box-shadow .18s, transform .18s;
}
.rl:hover { box-shadow: 0 6px 18px rgba(15, 23, 42, .09); transform: translateY(-1px); }
.rl:focus-visible { outline: 2px solid var(--c-secondary); outline-offset: 2px; }
.rl.hot { border-left-color: #dc2626; background: #fef2f2; }
.rl.muted { cursor: default; opacity: .6; }
.rl.muted:hover { box-shadow: none; transform: none; }
.rl-v { font-size: 26px; font-weight: 700; color: var(--c-text-strong); line-height: 1.1; }
.rl.hot .rl-v { color: #991b1b; }
.rl-lb { font-size: 12px; color: var(--c-text-muted); }
.rl-note { font-size: 10.5px; color: var(--c-text-muted); opacity: .8; }

.hot-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 1px; }
.hot-list li {
  display: flex; align-items: center; gap: 10px; flex-wrap: wrap;
  padding: 8px 12px; font-size: 13px;
  background: var(--c-surface); border: 1px solid var(--c-border);
}
.hot-list li:first-child { border-radius: var(--radius) var(--radius) 0 0; }
.hot-list li:last-child { border-radius: 0 0 var(--radius) var(--radius); }
.hot-list li.risk { background: #fef2f2; border-color: #fecaca; }
.hl-lv {
  font-size: 11.5px; padding: 2px 7px; border-radius: 5px;
  background: #fee2e2; color: #991b1b; font-weight: 600; white-space: nowrap;
}
.hl-proj { font-weight: 600; white-space: nowrap; }
.hl-title { color: var(--c-text); flex: 1; min-width: 0; }
.hl-tag {
  font-size: 11px; padding: 2px 7px; border-radius: 5px;
  background: #7f1d1d; color: #fff; white-space: nowrap;
}
.all-clear {
  margin: 0; padding: 12px 14px; font-size: 13px; color: #166534;
  background: #f0fdf4; border: 1px solid #bbf7d0; border-radius: var(--radius);
}
.placeholder {
  padding: 14px; font-size: 13px; color: var(--c-text-muted);
  background: var(--c-surface); border: 1px solid var(--c-border); border-radius: var(--radius);
}
.placeholder.err { color: #dc2626; }

/* ---- 例行卡片 ---- */
.grid { display: grid; gap: 14px; grid-template-columns: repeat(auto-fill, minmax(260px, 1fr)); }
.card {
  display: flex; flex-direction: column; gap: 8px;
  padding: 16px 18px; text-align: left; font: inherit; cursor: pointer;
  background: var(--c-surface); border: 1px solid var(--c-border);
  border-left: 3px solid var(--c-border-strong);
  border-radius: var(--radius); transition: box-shadow .18s, transform .18s;
}
.card:hover { box-shadow: 0 6px 18px rgba(15, 23, 42, .09); transform: translateY(-1px); }
.card:focus-visible { outline: 2px solid var(--c-secondary); outline-offset: 2px; }
.card.warn { border-left-color: #d97706; }
.card.bad { border-left-color: #dc2626; }
.c-head { display: flex; align-items: center; gap: 8px; }
.c-ic { font-size: 16px; color: var(--c-text-muted); }
.c-title { font-size: 14px; font-weight: 600; color: var(--c-text-strong); }
.c-tag {
  margin-left: auto; font-size: 11px; padding: 2px 7px; border-radius: 5px;
  background: var(--c-bg); color: var(--c-text-muted); white-space: nowrap;
}
.c-tag.warn { background: #fef3c7; color: #92400e; }
.c-tag.bad { background: #fee2e2; color: #991b1b; }
.c-body { display: flex; align-items: baseline; gap: 6px; min-height: 34px; }
.c-body.muted { font-size: 13px; color: var(--c-text-muted); align-items: center; }
.c-main { font-size: 26px; font-weight: 700; color: var(--c-text-strong); line-height: 1.1; }
.c-unit { font-size: 12px; color: var(--c-text-muted); }
.c-err { font-size: 12.5px; color: #dc2626; line-height: 1.5; }
.c-sub { display: flex; flex-wrap: wrap; gap: 6px; }
.c-chip { font-size: 11.5px; padding: 2px 7px; border-radius: 5px; background: var(--c-bg); color: var(--c-text-muted); }
.c-chip strong { color: var(--c-text); font-weight: 600; }
.c-chip.ok { background: #dcfce7; color: #166534; }
.c-chip.ok strong { color: #166534; }
.c-chip.warn { background: #fef3c7; color: #92400e; }
.c-chip.warn strong { color: #92400e; }
.c-chip.bad { background: #fee2e2; color: #991b1b; }
.c-chip.bad strong { color: #991b1b; }
.c-foot {
  font-size: 11.5px; color: var(--c-text-muted);
  border-top: 1px solid var(--c-border); padding-top: 8px; margin-top: auto;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}

/* ---- 考核表 ---- */
.kpi-tbl {
  width: 100%; border-collapse: collapse; font-size: 13px;
  background: var(--c-surface); border: 1px solid var(--c-border); border-radius: var(--radius);
}
.kpi-tbl th, .kpi-tbl td { padding: 9px 12px; text-align: left; border-bottom: 1px solid var(--c-border); }
.kpi-tbl th { font-size: 12px; color: var(--c-text-muted); font-weight: 600; }
.kpi-tbl tr:last-child td { border-bottom: 0; }
.kpi-tbl .num { text-align: right; white-space: nowrap; }
.kpi-tbl .src { font-size: 11.5px; color: var(--c-text-muted); }
.kpi-tbl tr.pending { opacity: .55; }

.downgrade { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; font-size: 12px; }
.dg-label { font-weight: 700; color: #7f1d1d; }
.dg-item { padding: 3px 9px; border-radius: 5px; background: var(--c-bg); color: var(--c-text-muted); }
.dg-item.hit { background: #fee2e2; color: #991b1b; font-weight: 600; }
.dg-item.muted { opacity: .65; }

.warn-list { margin: 4px 0 0; padding-left: 18px; font-size: 12.5px; line-height: 1.7; }
.warn-tip { margin: 6px 0 0; font-size: 12px; color: var(--c-text-muted); }
</style>
