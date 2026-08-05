<template>
  <div class="ov">
    <div class="page-head">
      <div>
        <h1 class="page-title">进度管理</h1>
        <p class="page-desc">
          五个子模块的当期概览 · 数据来自各项目云盘目录，点卡片进入明细
        </p>
      </div>
      <el-button @click="loadAll" :loading="anyLoading">
        <el-icon><Refresh /></el-icon>重新扫描
      </el-button>
    </div>

    <div class="grid">
      <button v-for="c in cards" :key="c.key" type="button" class="card" :class="c.state.tone"
        @click="go(c.path)">
        <div class="c-head">
          <el-icon class="c-ic"><component :is="c.icon" /></el-icon>
          <span class="c-title">{{ c.title }}</span>
          <span v-if="c.state.tag" class="c-tag" :class="c.state.tone">{{ c.state.tag }}</span>
        </div>

        <div v-if="c.state.loading" class="c-body muted">扫描中…</div>
        <div v-else-if="c.state.err" class="c-body">
          <div class="c-err">{{ c.state.err }}</div>
        </div>
        <template v-else>
          <div class="c-body">
            <span class="c-main">{{ c.state.main }}</span>
            <span class="c-unit">{{ c.state.unit }}</span>
          </div>
          <div class="c-sub">
            <span v-for="(s, i) in c.state.subs" :key="i" class="c-chip" :class="s.tone">
              {{ s.label }} <strong>{{ s.value }}</strong>
            </span>
          </div>
        </template>

        <div class="c-foot">{{ c.state.foot || c.desc }}</div>
      </button>
    </div>

    <el-alert v-if="errs.length" type="info" show-icon :closable="false"
      :title="`${errs.length} 个子模块暂时取不到数据`">
      <ul class="warn-list">
        <li v-for="(e, i) in errs" :key="i">{{ e }}</li>
      </ul>
      <p class="warn-tip">
        多半是根目录未设置、对应附件尚未提交，或该期次无数据。可点进子页面看具体提示。
      </p>
    </el-alert>
  </div>
</template>

<script setup lang="ts">
import { reactive, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import axios from 'axios'
import {
  Refresh, TrendCharts, Document, Money, AlarmClock, FolderChecked,
} from '@element-plus/icons-vue'

const router = useRouter()

type Sub = { label: string; value: string | number; tone?: string }
type CardState = {
  loading: boolean
  err: string
  main: string | number
  unit: string
  subs: Sub[]
  tone: string
  tag: string
  foot: string
}

function blank(): CardState {
  return { loading: true, err: '', main: '—', unit: '', subs: [], tone: '', tag: '', foot: '' }
}

const st = reactive<Record<string, CardState>>({
  indicators: blank(), check: blank(), eot: blank(), timebar: blank(), archive: blank(),
})

const cards = computed(() => [
  {
    key: 'indicators', title: '指标', icon: TrendCharts, path: '/progress/indicators',
    desc: '附件 B-1 月度进度数据 · 风险灯按细则 4.1 判定', state: st.indicators,
  },
  {
    key: 'check', title: '报送核查', icon: Document, path: '/progress/check',
    desc: '扫描各项目目录，核对当期应交文件', state: st.check,
  },
  {
    key: 'eot', title: '工期索赔', icon: Money, path: '/progress/eot',
    desc: '附件 D 索赔台账 · CN 10 日、PCO 21 日历日', state: st.eot,
  },
  {
    key: 'timebar', title: '合同时效预警', icon: AlarmClock, path: '/progress/timebar',
    desc: '附件 E 时效日历 · 逾期即丧失索赔权', state: st.timebar,
  },
  {
    key: 'archive', title: '收文归档', icon: FolderChecked, path: '/progress/archive',
    desc: '06.Received File 待归档收文', state: st.archive,
  },
])

const anyLoading = computed(() => Object.values(st).some(s => s.loading))
const errs = computed(() =>
  cards.value.filter(c => c.state.err).map(c => `${c.title}：${c.state.err}`))

function go(path: string) {
  router.push(path)
}

// 后端返回的错误体是 {error, detail}；网络层错误取 message。
function errText(e: any): string {
  return e?.response?.data?.error || e?.message || '取数失败'
}

async function one(key: string, fn: () => Promise<void>) {
  const s = st[key]
  s.loading = true
  s.err = ''
  try {
    await fn()
  } catch (e) {
    s.err = errText(e)
    s.tone = ''
    s.tag = ''
  }
  s.loading = false
}

async function loadIndicators() {
  const { data } = await axios.get('/api/progress/indicators')
  const d = data.summary || {}
  const s = st.indicators
  const attention = (d.red || 0) + (d.orange || 0)
  s.main = d.total || 0
  s.unit = '个项目已报'
  s.subs = [
    { label: '红', value: d.red || 0, tone: (d.red || 0) > 0 ? 'bad' : '' },
    { label: '橙', value: d.orange || 0, tone: (d.orange || 0) > 0 ? 'warn' : '' },
    { label: '黄', value: d.yellow || 0 },
    { label: '绿', value: d.green || 0, tone: 'ok' },
  ]
  s.tone = (d.red || 0) > 0 ? 'bad' : attention > 0 ? 'warn' : ''
  s.tag = attention > 0 ? `${attention} 项需关注` : ''
  s.foot = `${data.period} · 在建项目 ${data.project_count || 0} 个`
}

async function loadCheck() {
  const { data } = await axios.get('/api/progress/check')
  const d = data.summary || {}
  const s = st.check
  const missing = d.missing || 0
  s.main = d.compliance != null ? (d.compliance * 100).toFixed(0) : '—'
  s.unit = '% 按时率'
  s.subs = [
    { label: '按时', value: d.submitted || 0, tone: 'ok' },
    { label: '迟交', value: d.late || 0, tone: (d.late || 0) > 0 ? 'warn' : '' },
    { label: '缺交', value: missing, tone: missing > 0 ? 'bad' : '' },
  ]
  s.tone = missing > 0 ? 'bad' : (d.late || 0) > 0 ? 'warn' : ''
  s.tag = missing > 0 ? `${missing} 项缺交` : ''
  s.foot = `${data.period} · 在建项目 ${d.project_count || 0} 个`
}

async function loadEOT() {
  const { data } = await axios.get('/api/eot')
  const d = data.summary || {}
  const s = st.eot
  const overdue = (d.overdue_cn || 0) + (d.overdue_pco || 0)
  s.main = d.total || 0
  s.unit = '项索赔在跟踪'
  s.subs = [
    { label: 'CN 逾期', value: d.overdue_cn || 0, tone: (d.overdue_cn || 0) > 0 ? 'bad' : '' },
    { label: 'PCO 逾期', value: d.overdue_pco || 0, tone: (d.overdue_pco || 0) > 0 ? 'bad' : '' },
    { label: '累计请求', value: `${d.total_days || 0} 天` },
    { label: '已关闭', value: d.closed_count || 0 },
  ]
  s.tone = overdue > 0 ? 'bad' : ''
  s.tag = overdue > 0 ? `${overdue} 项已逾期` : ''
  s.foot = data.warning || `覆盖 ${d.projects || 0} 个在建项目`
}

async function loadTimeBar() {
  const { data } = await axios.get('/api/timebar')
  const d = data.summary || {}
  const s = st.timebar
  const urgent = (d.overdue || 0) + (d.red || 0)
  s.main = d.overdue || 0
  s.unit = '项已逾期'
  s.subs = [
    { label: '紧急 <7天', value: d.red || 0, tone: (d.red || 0) > 0 ? 'bad' : '' },
    { label: '7–14 天', value: d.amber || 0, tone: (d.amber || 0) > 0 ? 'warn' : '' },
    { label: '充裕', value: d.green || 0, tone: 'ok' },
    { label: '已办结', value: d.done || 0 },
  ]
  s.tone = urgent > 0 ? 'bad' : (d.amber || 0) > 0 ? 'warn' : ''
  s.tag = (d.overdue || 0) > 0 ? '有逾期' : (d.red || 0) > 0 ? '紧急' : ''
  s.foot = data.warning || `共 ${d.total || 0} 条时效 · ${d.projects || 0} 个在建项目`
}

async function loadArchive() {
  const { data } = await axios.get('/api/archive/inbox')
  const files = data.files || []
  const s = st.archive
  const named = files.filter((f: any) => f.project && f.code && f.period).length
  s.main = files.length
  s.unit = '件待归档'
  s.subs = [
    { label: '可自动识别', value: named, tone: named > 0 ? 'ok' : '' },
    { label: '需人工指定', value: files.length - named,
      tone: files.length - named > 0 ? 'warn' : '' },
  ]
  s.tone = ''
  s.tag = files.length > 0 ? '待处理' : ''
  s.foot = data.dir || ''
}

function loadAll() {
  // 五个接口都要扫云盘，并发发出，各卡片独立降级，不让某一个失败拖垮整页
  one('indicators', loadIndicators)
  one('check', loadCheck)
  one('eot', loadEOT)
  one('timebar', loadTimeBar)
  one('archive', loadArchive)
}
onMounted(loadAll)
</script>

<style scoped>
.ov { display: flex; flex-direction: column; gap: 14px; }
.page-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; }
.page-title { font-size: 22px; font-weight: 700; color: var(--c-text-strong); margin: 0; }
.page-desc { font-size: 13px; color: var(--c-text-muted); margin: 4px 0 0; }

.grid {
  display: grid; gap: 14px;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
}
.card {
  display: flex; flex-direction: column; gap: 8px;
  padding: 16px 18px; text-align: left; font: inherit; cursor: pointer;
  background: var(--c-surface);
  border: 1px solid var(--c-border);
  border-left: 3px solid var(--c-border-strong);
  border-radius: var(--radius);
  transition: box-shadow .18s, transform .18s, border-color .18s;
}
.card:hover { box-shadow: 0 6px 18px rgba(15, 23, 42, .09); transform: translateY(-1px); }
.card:focus-visible { outline: 2px solid var(--c-secondary); outline-offset: 2px; }
/* 严重度只靠左侧色条会让红绿色盲无法区分，右上角一律配文字标签 */
.card.warn { border-left-color: var(--c-warning, #d97706); }
.card.bad { border-left-color: var(--c-danger, #dc2626); }

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
.c-err { font-size: 12.5px; color: var(--c-danger, #dc2626); line-height: 1.5; }

.c-sub { display: flex; flex-wrap: wrap; gap: 6px; }
.c-chip {
  font-size: 11.5px; padding: 2px 7px; border-radius: 5px;
  background: var(--c-bg); color: var(--c-text-muted);
}
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

.warn-list { margin: 4px 0 0; padding-left: 18px; font-size: 12.5px; line-height: 1.7; }
.warn-tip { margin: 6px 0 0; font-size: 12px; color: var(--c-text-muted); }
</style>
