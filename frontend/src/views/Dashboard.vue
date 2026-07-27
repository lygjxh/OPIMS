<template>
  <div class="dashboard">
    <div class="page-head">
      <div>
        <h1 class="page-title">项目总览</h1>
        <p class="page-desc">海外在建项目分布与状态一览</p>
      </div>
    </div>

    <!-- KPI 卡片：点击跳转项目清单并按状态筛选 -->
    <div class="kpi-grid">
      <button v-for="k in kpis" :key="k.key" class="kpi"
        :class="{ zero: k.value === 0 && k.key !== 'total' }"
        :style="{ '--kpi': k.color }" type="button"
        :title="k.status ? `查看${k.label}项目` : '查看全部项目'"
        @click="gotoList(k.status)">
        <div class="kpi-top">
          <span class="kpi-label">{{ k.label }}</span>
          <el-icon class="kpi-ic"><component :is="k.icon" /></el-icon>
        </div>
        <div class="kpi-num">{{ k.value }}</div>
        <div class="kpi-foot">
          <span class="kpi-pct">{{ k.key === 'total' ? '全部项目' : pctText(k.value) }}</span>
        </div>
        <div class="kpi-bar"><span :style="{ width: barWidth(k.value) }"></span></div>
      </button>
    </div>

    <div class="grid-2">
      <!-- 地图 -->
      <el-card class="map-card">
        <template #header>
          <div class="card-head">
            <span>项目全球分布</span>
            <div class="legend">
              <span v-for="s in STATUS_LIST" :key="s.name">
                <i :class="'mk mk-' + s.shape" :style="{ '--mk': s.color }"></i>{{ s.name }}
              </span>
            </div>
          </div>
        </template>
        <div class="map-wrap">
          <div id="worldmap" ref="mapRef" class="worldmap"></div>
          <!-- B: 空状态提示 -->
          <div v-if="mapEmpty" class="map-empty">
            <el-icon :size="26"><LocationInformation /></el-icon>
            <p class="me-title">暂无可定位的项目</p>
            <p class="me-desc">项目未填写 GPS 坐标且国别无法识别。<br />可在项目详情中填写 GPS 坐标以精确定位。</p>
          </div>
        </div>
        <div v-if="approxCount" class="map-note">
          <el-icon><InfoFilled /></el-icon>
          其中 {{ approxCount }} 个项目无 GPS 坐标，按国别近似定位（虚线圈）
        </div>
      </el-card>

      <!-- 合同额分布：饼图，可按地区/国别切换，可按状态筛选 -->
      <el-card class="amount-card">
        <template #header>
          <div class="card-head">
            <span>合同额分布</span>
            <el-radio-group v-model="groupBy" size="small">
              <el-radio-button label="region">按地区</el-radio-button>
              <el-radio-button label="country">按国别</el-radio-button>
            </el-radio-group>
          </div>
        </template>

        <!-- 状态筛选：全部（单选） + 在建/未开工/完工（可多选） -->
        <div class="status-filter">
          <button type="button" class="sf-btn" :class="{ on: !statusPicked.length }"
            @click="statusPicked = []">全部</button>
          <button v-for="s in PICKABLE_STATUS" :key="s" type="button"
            class="sf-btn" :class="{ on: statusPicked.includes(s) }"
            @click="toggleStatus(s)">{{ s }}</button>
        </div>

        <template v-if="distribution?.slices?.length">
          <!-- 环形图：SVG 手绘，无第三方图表依赖 -->
          <div class="donut-wrap">
            <svg class="donut" viewBox="0 0 200 200" role="img"
              :aria-label="`合同额分布，共 ${fmtYi(Number(distribution?.total_amount) || 0)} 亿元`">
              <g v-for="(s, i) in slices" :key="s.name">
                <path :d="s.path" :fill="s.color" class="slice"
                  :class="{ dim: hoverIdx >= 0 && hoverIdx !== i, clickable: canDrill(s.name) }"
                  @mouseenter="hoverIdx = Number(i)" @mouseleave="hoverIdx = -1"
                  @click="drillDown({ name: s.name, amount: s.amount, count: s.count })">
                  <title>{{ s.name }}：{{ fmtYi(s.amount) }} 亿元（{{ s.pct }}%）· {{ s.count }} 个项目</title>
                </path>
              </g>
              <!-- 圆心汇总 -->
              <text x="100" y="94" class="dn-num" text-anchor="middle">{{ fmtYi(Number(distribution?.total_amount) || 0) }}</text>
              <text x="100" y="112" class="dn-unit" text-anchor="middle">亿元</text>
              <text x="100" y="128" class="dn-sub" text-anchor="middle">{{ distribution?.total_count || 0 }} 个项目</text>
            </svg>
          </div>

          <!-- 图例：名称+金额+占比，不让颜色单独承担信息 -->
          <ul class="dn-legend">
            <li v-for="(s, i) in slices" :key="s.name"
              :class="{ dim: hoverIdx >= 0 && hoverIdx !== i, clickable: canDrill(s.name) }"
              @mouseenter="hoverIdx = Number(i)" @mouseleave="hoverIdx = -1"
              @click="drillDown({ name: s.name, amount: s.amount, count: s.count })">
              <i class="sw" :style="{ background: s.color }"></i>
              <span class="lg-name">{{ s.name }}</span>
              <span class="lg-val">{{ fmtYi(s.amount) }}<em>亿</em></span>
              <span class="lg-pct">{{ s.pct }}%</span>
            </li>
          </ul>
          <p v-if="groupBy === 'region'" class="bar-tip">
            <el-icon><InfoFilled /></el-icon>点击扇区或图例可展开国别明细
          </p>
        </template>
        <p v-else class="empty-hint">当前筛选条件下暂无合同额数据</p>
      </el-card>
    </div>

    <!-- 快捷入口（整行） -->
    <el-card class="shortcuts-card">
      <template #header>{{ $t('dashboard.shortcuts') }}</template>
      <div class="shortcuts">
        <button class="shortcut" v-for="item in shortcuts" :key="item.path"
          type="button" :title="item.label" @click="goShortcut(item)">
          <el-icon :size="16"><component :is="item.icon" /></el-icon>
          <span>{{ item.label }}</span>
        </button>
      </div>
    </el-card>

    <!-- 地图点击 → 该国出入境政策摘要 -->
    <el-dialog v-model="policyVisible" :title="`${policyItem?.country} · 出入境政策`" width="480px">
      <template v-if="policyItem">
        <div class="pol-risk" :class="'pr-' + riskKey(policyItem.risk_level)">
          <strong>{{ policyItem.risk_level || '未评级' }}风险</strong>
          <span>{{ policyItem.risk_note }}</span>
        </div>
        <table class="pol-table">
          <tbody>
            <tr><th>签证类型</th><td>{{ policyItem.visa_type || '-' }}</td></tr>
            <tr><th>预计办理周期</th><td>{{ policyItem.total_cycle || '-' }}</td></tr>
            <tr><th>更新日期</th><td>
              {{ policyItem.updated_at || '未填写' }}
              <span v-if="policyItem.is_stale" class="pol-stale">
                （{{ policyItem.stale_days }} 天前，建议核查）
              </span>
            </td></tr>
          </tbody>
        </table>
      </template>
      <template #footer>
        <el-button @click="policyVisible = false">关闭</el-button>
        <el-button type="primary" @click="gotoPolicyDetail">查看完整档案</el-button>
      </template>
    </el-dialog>

    <!-- 地区下钻弹窗 -->
    <el-dialog v-model="drillVisible" :title="`${drillRegion} · 国别明细`" width="520px">
      <div class="bars">
        <div v-for="c in drillRows" :key="c.name" class="bar-row"
          :title="`${c.name}：${fmtYi(c.amount)} 亿元 · ${c.count} 个项目`">
          <span class="bar-name">{{ c.name }}</span>
          <div class="bar-track">
            <span class="bar-fill" :style="{ width: drillWidth(c.amount) }"></span>
          </div>
          <span class="bar-val">{{ fmtYi(c.amount) }}<em>亿</em></span>
          <span class="bar-cnt">{{ c.count }}个</span>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import axios from 'axios'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import {
  OfficeBuilding, List, FolderOpened, WarningFilled, UserFilled, TrendCharts,
  CircleCheck, Connection, Avatar, Loading, Clock, Warning, Select,
  LocationInformation, InfoFilled, Flag,
} from '@element-plus/icons-vue'
import { COUNTRY_COORDS, spreadOffset } from '../data/countryCoords'

const { t } = useI18n()
const router = useRouter()
const mapRef = ref<HTMLElement | null>(null)
const dashboard = ref<any>({ total_projects: 0, status_counts: {}, project_markers: [] })
const projects = ref<any[]>([])
const mapEmpty = ref(false)
const approxCount = ref(0)

/* 状态 → 颜色 + 地图标记形状（形状是颜色之外的第二重编码，红绿色盲也能区分） */
const STATUS_LIST = [
  { name: '在建', color: '#2a78d6', shape: 'dot' },
  { name: '未开工', color: '#eda100', shape: 'ring' },
  { name: '停工', color: '#d03b3b', shape: 'square' },
  { name: '完工', color: '#0ca30c', shape: 'small' },
]
const statusColor = (s: string) => STATUS_LIST.find(x => x.name === s)?.color || '#7a8699'
const statusShape = (s: string) => STATUS_LIST.find(x => x.name === s)?.shape || 'dot'

const kpis = computed(() => [
  { key: 'total', label: t('dashboard.total'), status: '', value: dashboard.value.total_projects || 0, color: 'var(--c-primary)', icon: OfficeBuilding },
  { key: 'in', label: t('project.inProgress'), status: '在建', value: dashboard.value.status_counts?.['在建'] || 0, color: 'var(--c-status-inprogress)', icon: Loading },
  { key: 'not', label: t('project.notStarted'), status: '未开工', value: dashboard.value.status_counts?.['未开工'] || 0, color: 'var(--c-status-notstarted)', icon: Clock },
  { key: 'sus', label: t('project.suspended'), status: '停工', value: dashboard.value.status_counts?.['停工'] || 0, color: 'var(--c-status-suspended)', icon: Warning },
  { key: 'done', label: t('project.completed'), status: '完工', value: dashboard.value.status_counts?.['完工'] || 0, color: 'var(--c-status-completed)', icon: Select },
])

const total = computed(() => dashboard.value.total_projects || 0)
function barWidth(v: number) {
  if (!total.value) return '0%'
  return Math.min(100, Math.round((v / total.value) * 100)) + '%'
}
function pctText(v: number) {
  if (!total.value) return '—'
  if (v === 0) return '无'
  return `占比 ${Math.round((v / total.value) * 100)}%`
}
function gotoList(status: string) {
  router.push({ path: '/projects', query: status ? { status } : {} })
}

/* ---- 合同额分布（环形图）：可按地区/国别切换，可按状态筛选 ---- */

const groupBy = ref<'region' | 'country'>('region')

/** 状态筛选：空数组 = 全部；否则为选中的状态（可多选） */
const PICKABLE_STATUS = ['在建', '未开工', '完工']
const statusPicked = ref<string[]>([])
function toggleStatus(s: string) {
  const i = statusPicked.value.indexOf(s)
  if (i >= 0) statusPicked.value.splice(i, 1)
  else statusPicked.value.push(s)
}

/** 从后端 API 加载合同额分布数据 */
const distribution = ref<any>(null)
const distLoading = ref(false)

async function loadDistribution() {
  distLoading.value = true
  try {
    const params: any = { dim: groupBy.value }
    if (statusPicked.value.length) {
      params.status = statusPicked.value.join(',')
    }
    const { data } = await axios.get('/api/dashboard/contract-distribution', { params })
    distribution.value = data
  } catch { distribution.value = null }
  distLoading.value = false
}

/* 分类配色：取自 dataviz 规范已验证的分类色序，固定顺序、不循环复用。
   超过 7 类时尾部合并为「其他 N 项」——该逻辑由后端 /api/dashboard/contract-distribution 完成。 */
const SLICE_COLORS = ['#2a78d6', '#eb6834', '#1baf7a', '#eda100', '#e87ba4', '#008300', '#4a3aa7']

const hoverIdx = ref(-1)

/** 环形图扇区：从 API 数据计算 SVG 路径与颜色 */
const slices = computed(() => {
  const raw = distribution.value?.slices || []
  // 注意：后端 JSON 字段为下划线命名（total_amount / total_count），
  // 写成驼峰会取到 undefined，导致 total 退化为 1、扇区角度爆表、饼图画坏。
  const total = Number(distribution.value?.total_amount) || 1
  let angle = -90 // 从 12 点方向开始
  return raw.map((g: any, i: number) => {
    const sweep = total > 0 ? (g.amount / total) * 360 : 0
    const path = arcPath(100, 100, 88, 56, angle, angle + sweep)
    angle += sweep
    return {
      name: String(g.name),
      amount: Number(g.amount) || 0,
      count: Number(g.count) || 0,
      color: SLICE_COLORS[i % SLICE_COLORS.length],
      pct: total > 0 ? (g.amount / total * 100).toFixed(1) : '0.0',
      path,
    }
  })
})

/** 生成环形扇区路径；相邻扇区留 1.2° 间隙，等效"表面留白"分隔 */
function arcPath(cx: number, cy: number, rOut: number, rIn: number, a0: number, a1: number) {
  const gap = Math.min(1.2, Math.abs(a1 - a0) / 4)
  const s = a0 + gap / 2, e = a1 - gap / 2
  const full = e - s >= 359.5
  const rad = (d: number) => (d * Math.PI) / 180
  const pt = (r: number, d: number) => [cx + r * Math.cos(rad(d)), cy + r * Math.sin(rad(d))]
  if (full) {
    // 单一分组占满：画整圆环（两段半圆拼接，避免弧长 360° 的退化）
    return `M ${cx - rOut} ${cy} A ${rOut} ${rOut} 0 1 1 ${cx + rOut} ${cy} A ${rOut} ${rOut} 0 1 1 ${cx - rOut} ${cy} Z ` +
           `M ${cx - rIn} ${cy} A ${rIn} ${rIn} 0 1 0 ${cx + rIn} ${cy} A ${rIn} ${rIn} 0 1 0 ${cx - rIn} ${cy} Z`
  }
  const large = e - s > 180 ? 1 : 0
  const [x1, y1] = pt(rOut, s), [x2, y2] = pt(rOut, e)
  const [x3, y3] = pt(rIn, e), [x4, y4] = pt(rIn, s)
  return `M ${x1} ${y1} A ${rOut} ${rOut} 0 ${large} 1 ${x2} ${y2} L ${x3} ${y3} A ${rIn} ${rIn} 0 ${large} 0 ${x4} ${y4} Z`
}

const canDrill = (name: string) =>
  groupBy.value === 'region' && name !== '未分类' && !name.startsWith('其他 ')

/* ---- 地区下钻 ---- */
const drillVisible = ref(false)
const drillRegion = ref('')
const drillRows = ref<any[]>([])
const drillLoading = ref(false)

async function drillDown(g: any) {
  if (!canDrill(g.name)) return
  drillRegion.value = g.name
  drillVisible.value = true
  drillLoading.value = true
  try {
    const params: any = {}
    if (statusPicked.value.length) params.status = statusPicked.value.join(',')
    const { data } = await axios.get(`/api/dashboard/region-detail/${encodeURIComponent(g.name)}`, { params })
    drillRows.value = data?.countries || data || []
  } catch { drillRows.value = [] }
  drillLoading.value = false
}
const drillMax = computed(() => Math.max(...drillRows.value.map(r => r.amount), 1))
function drillWidth(v: number) { return Math.max(1.5, (v / drillMax.value) * 100) + '%' }

/** 万元 → 亿元 */
function fmtYi(wan: number | string) {
  const yi = Number(wan) / 10000 || 0
  return yi >= 100 ? yi.toFixed(0) : yi >= 10 ? yi.toFixed(1) : yi.toFixed(2)
}

const shortcuts = [
  { path: '/projects', label: t('menu.projects'), icon: List, ready: true },
  { path: '/files', label: t('menu.files'), icon: FolderOpened, ready: true },
  { path: '/blacklist-sub', label: t('menu.blacklistSub'), icon: WarningFilled, ready: true },
  { path: '/country-profile', label: t('menu.countryProfile'), icon: Flag, ready: true },
  { path: '/blacklist-person', label: t('menu.blacklistPerson'), icon: UserFilled, ready: false },
  { path: '/progress', label: t('menu.progress'), icon: TrendCharts, ready: true },
  { path: '/quality', label: t('menu.quality'), icon: CircleCheck, ready: false },
  { path: '/subcontract', label: t('menu.subcontract'), icon: Connection, ready: true },
  { path: '/subcontractors', label: t('menu.subcontractors'), icon: OfficeBuilding, ready: true },
  { path: '/personnel', label: t('menu.personnel'), icon: Avatar, ready: false },
]
function goShortcut(item: any) {
  if (item.ready) router.push(item.path)
  else ElMessage.info(`${item.label} 模块建设中`)
}

let map: L.Map | null = null

onMounted(async () => {
  try {
    const { data } = await axios.get('/api/dashboard')
    dashboard.value = data
  } catch { /* 无数据 */ }
  try {
    // status=all：取全部状态，用于地图国别定位
    const { data } = await axios.get('/api/projects', { params: { status: 'all' } })
    projects.value = data.projects || []
  } catch { projects.value = [] }
  await loadDistribution()
  loadPolicies()
  await nextTick()
  drawMap()
})

// deep 必需：statusPicked 是数组 ref，toggleStatus 用 push/splice 原地修改，
// 引用不变，默认（浅）监听不会触发，状态筛选会失效。
watch([groupBy, statusPicked], loadDistribution, { deep: true })

function drawMap() {
  if (!mapRef.value) return
  map = L.map('worldmap', { worldCopyJump: true }).setView([22, 40], 2)
  L.tileLayer('https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png', {
    attribution: '&copy; OpenStreetMap',
  }).addTo(map)

  const gpsMarkers = (dashboard.value.project_markers || []).filter((m: any) => m.lat || m.lng)
  const placed = new Set<string>(gpsMarkers.map((m: any) => m.short_name))
  const bounds: [number, number][] = []

  // 1) 有真实 GPS 的项目：实心标记
  gpsMarkers.forEach((m: any) => {
    addMarker(m.lat, m.lng, m.short_name || m.country, m.status, m.country, false)
    bounds.push([m.lat, m.lng])
  })

  // 2) A：无 GPS 但国别可识别 → 落到国家代表坐标（虚线圈表示近似）
  const perCountry = new Map<string, number>()
  let approx = 0
  for (const p of projects.value) {
    if (placed.has(p.short_name)) continue
    if (p.gps_lat || p.gps_lng) continue
    const base = COUNTRY_COORDS[p.country]
    if (!base) continue
    const idx = perCountry.get(p.country) || 0
    perCountry.set(p.country, idx + 1)
    const [dy, dx] = spreadOffset(idx)
    const lat = base[0] + dy, lng = base[1] + dx
    addMarker(lat, lng, p.short_name, p.project_status, p.country, true)
    bounds.push([lat, lng])
    approx++
  }
  approxCount.value = approx
  mapEmpty.value = bounds.length === 0

  if (bounds.length > 1) map!.fitBounds(bounds as any, { padding: [40, 40], maxZoom: 4 })
  else if (bounds.length === 1) map!.setView(bounds[0], 4)
}

function addMarker(lat: number, lng: number, name: string, status: string, country: string, approx: boolean) {
  const color = statusColor(status)
  const shape = statusShape(status)
  // 形状差异 = 颜色之外的第二重编码
  const opt: L.CircleMarkerOptions = {
    radius: shape === 'small' ? 5 : shape === 'square' ? 8 : 7,
    fillColor: shape === 'ring' ? '#fff' : color,
    color: color,
    weight: shape === 'ring' ? 3 : 2,
    fillOpacity: shape === 'ring' ? 1 : 0.9,
    dashArray: approx ? '3,3' : undefined,
  }
  const mk = shape === 'square'
    ? L.rectangle(L.latLngBounds([lat - 0.9, lng - 0.9], [lat + 0.9, lng + 0.9]),
        { color, fillColor: color, fillOpacity: .9, weight: 2, dashArray: approx ? '3,3' : undefined })
    : L.circleMarker([lat, lng], opt)
  mk.addTo(map!)
  // tooltip 始终带状态文字 —— 不让颜色单独承担信息
  mk.bindTooltip(
    `${name}<br><span style="color:#64748b">${status || '未知状态'} · ${country || '-'}${approx ? ' · 按国别近似' : ''}</span>`,
    { direction: 'top' }
  )
  // 点击标记：若该国有出入境政策档案，弹出摘要并可跳转查看
  mk.on('click', () => showCountryPolicy(country))
}

/* ---- 地图点击 → 该国出入境政策摘要 ---- */
// 风险等级 → 样式后缀（与国别档案页保持一致；颜色之外必有文字标签）
const riskKey = (lv: string) =>
  ({ 高: 'high', 中: 'mid', 低: 'low' } as Record<string, string>)[lv] || 'na'
const policyMap = ref<Record<string, any>>({})
const policyVisible = ref(false)
const policyItem = ref<any>(null)

async function loadPolicies() {
  try {
    const { data } = await axios.get('/api/policy/countries')
    const m: Record<string, any> = {}
    for (const c of data.countries || []) m[c.country] = c
    policyMap.value = m
  } catch { /* 政策库未配置时静默跳过，不影响看板 */ }
}

function showCountryPolicy(country: string) {
  const p = policyMap.value[country]
  if (!p) {
    ElMessage.info(`${country || '该国'} 暂无出入境政策档案`)
    return
  }
  policyItem.value = p
  policyVisible.value = true
}

function gotoPolicyDetail() {
  policyVisible.value = false
  router.push({ path: '/country-profile', query: { country: policyItem.value.country } })
}
</script>

<style scoped>
.dashboard { display: flex; flex-direction: column; gap: 18px; }
.page-title { font-size: 22px; font-weight: 700; color: var(--c-text-strong); margin: 0; }
.page-desc { font-size: 13px; color: var(--c-text-muted); margin: 4px 0 0; }

/* ---- KPI ---- */
.kpi-grid { display: grid; grid-template-columns: repeat(5, 1fr); gap: 14px; }
.kpi {
  position: relative; overflow: hidden; text-align: left; cursor: pointer;
  background: var(--c-surface); border: 1px solid var(--c-border);
  border-radius: var(--radius); padding: 16px 16px 12px;
  box-shadow: var(--shadow-card); font: inherit;
  transition: transform .18s, box-shadow .18s, border-color .18s;
}
.kpi::before { content: ''; position: absolute; left: 0; top: 0; bottom: 0; width: 4px; background: var(--kpi); }
.kpi:hover { transform: translateY(-3px); box-shadow: var(--shadow-pop); border-color: var(--c-border-strong); }
.kpi:focus-visible { outline: 2px solid var(--c-primary); outline-offset: 2px; }
.kpi.zero { opacity: .62; }
.kpi.zero .kpi-bar { visibility: hidden; }
.kpi-top { display: flex; align-items: center; justify-content: space-between; }
.kpi-label { font-size: 13px; color: var(--c-text-muted); font-weight: 500; }
.kpi-ic { font-size: 17px; color: var(--kpi); opacity: .9; }
.kpi-num { font-size: 32px; font-weight: 800; color: var(--c-text-strong); line-height: 1.15; margin: 8px 0 2px; }
.kpi-foot { min-height: 16px; }
.kpi-pct { font-size: 11px; color: var(--c-text-muted); }
.kpi-bar { height: 4px; border-radius: 3px; background: #eef2f8; overflow: hidden; margin-top: 8px; }
.kpi-bar span { display: block; height: 100%; border-radius: 3px; background: var(--kpi); transition: width .5s ease; }

/* ---- 布局 ---- */
.grid-2 { display: grid; grid-template-columns: 2fr 1fr; gap: 16px; align-items: start; }
.card-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap; }

/* 图例：色块形状与地图标记一致 */
.legend { display: flex; gap: 13px; font-size: 12px; font-weight: 400; color: var(--c-text-muted); }
.legend span { display: inline-flex; align-items: center; gap: 5px; }
.mk { width: 10px; height: 10px; display: inline-block; background: var(--mk); }
.mk-dot { border-radius: 50%; }
.mk-ring { border-radius: 50%; background: #fff; border: 3px solid var(--mk); }
.mk-square { border-radius: 2px; }
.mk-small { border-radius: 50%; width: 7px; height: 7px; }

.map-wrap { position: relative; }
.worldmap { height: 430px; border-radius: 8px; }
.map-empty {
  position: absolute; inset: 0; z-index: 500; display: flex;
  flex-direction: column; align-items: center; justify-content: center; gap: 6px;
  background: rgba(248, 250, 252, .93); border-radius: 8px; color: var(--c-text-muted);
}
.me-title { font-size: 15px; font-weight: 600; color: var(--c-text-strong); margin: 4px 0 0; }
.me-desc { font-size: 12.5px; line-height: 1.7; text-align: center; margin: 0; }
.map-note {
  display: flex; align-items: center; gap: 6px; margin-top: 10px;
  font-size: 12px; color: var(--c-text-muted);
  background: #f6f9fe; border: 1px solid var(--c-border);
  border-radius: 7px; padding: 7px 10px;
}

/* ---- 快捷入口：自适应铺满，模块增多时自动换行 ---- */
.shortcuts {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(152px, 1fr));
  gap: 8px;
}
.shortcut {
  display: flex; align-items: center; gap: 8px;
  padding: 9px 12px; border-radius: 8px; cursor: pointer; font: inherit;
  background: #f6f9fe; border: 1px solid var(--c-border); color: var(--c-primary-700);
  font-size: 13px; font-weight: 500; transition: all .16s;
  text-align: left; min-width: 0;
}
.shortcut span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.shortcut :deep(.el-icon) { flex-shrink: 0; }
.shortcut:hover { background: var(--c-primary); color: #fff; border-color: var(--c-primary); }
.shortcut:focus-visible { outline: 2px solid var(--c-primary); outline-offset: 2px; }

/* ---- 合同额条形图（单一色调；长度编码量级） ---- */
/* ---- 状态筛选 ---- */
.status-filter { display: flex; gap: 6px; flex-wrap: wrap; margin-bottom: 14px; }
.sf-btn {
  font: inherit; font-size: 12px; padding: 4px 11px; cursor: pointer;
  border: 1px solid var(--c-border-strong); background: var(--c-surface);
  color: var(--c-text); border-radius: 14px; transition: all .16s;
}
.sf-btn:hover { border-color: var(--c-primary); color: var(--c-primary); }
.sf-btn.on {
  background: var(--c-primary); border-color: var(--c-primary);
  color: #fff; font-weight: 500;
}
.sf-btn:focus-visible { outline: 2px solid var(--c-primary); outline-offset: 1px; }

/* ---- 环形图 ---- */
.donut-wrap { display: flex; justify-content: center; }
.donut { width: 100%; max-width: 232px; height: auto; }
.slice { transition: opacity .18s; }
.slice.dim { opacity: .32; }
.slice.clickable { cursor: pointer; }
.dn-num { font-size: 30px; font-weight: 800; fill: var(--c-text-strong); }
.dn-unit { font-size: 11px; fill: var(--c-text-muted); }
.dn-sub { font-size: 10.5px; fill: var(--c-text-muted); }

.dn-legend { list-style: none; margin: 14px 0 0; padding: 0; }
.dn-legend li {
  display: grid; grid-template-columns: 11px 1fr auto 44px;
  align-items: center; gap: 8px; padding: 5px 6px;
  border-radius: 6px; font-size: 12.5px; transition: background .15s, opacity .18s;
}
.dn-legend li:hover { background: #f6f9fe; }
.dn-legend li.dim { opacity: .42; }
.dn-legend li.clickable { cursor: pointer; }
.sw { width: 11px; height: 11px; border-radius: 3px; display: inline-block; }
.lg-name { color: var(--c-text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.lg-val { color: var(--c-text-strong); font-weight: 600; font-variant-numeric: tabular-nums; }
.lg-val em { font-style: normal; font-size: 10.5px; font-weight: 400; color: var(--c-text-muted); margin-left: 1px; }
.lg-pct { color: var(--c-text-muted); text-align: right; font-variant-numeric: tabular-nums; }
.bar-tip {
  display: flex; align-items: center; gap: 5px; margin: 12px 0 0;
  font-size: 11.5px; color: var(--c-text-muted);
}

/* ---- 地图点击弹出的政策摘要 ---- */
.pol-risk {
  display: flex; flex-direction: column; gap: 4px;
  padding: 10px 13px; border-radius: 8px; margin-bottom: 14px;
  font-size: 12.5px; line-height: 1.7;
  background: #fdf5f5; border: 1px solid #f0d5d5;
}
.pol-risk.pr-mid { background: #fdf8ee; border-color: #f0e3c5; }
.pol-risk.pr-low { background: #f2faf2; border-color: #cfe8cf; }
.pol-risk.pr-na { background: #f6f9fe; border-color: var(--c-border); }
.pol-risk strong { font-size: 13px; }
.pol-table { width: 100%; border-collapse: collapse; font-size: 12.5px; }
.pol-table th, .pol-table td {
  border: 1px solid var(--c-border); padding: 7px 10px;
  text-align: left; vertical-align: top; line-height: 1.7;
}
.pol-table th { background: #f6f9fe; color: var(--c-text-muted); font-weight: 500; width: 96px; white-space: nowrap; }
.pol-stale { color: #a8730a; }
.bars { display: flex; flex-direction: column; gap: 2px; }
.bar-row {
  display: grid; grid-template-columns: 92px 1fr 82px 48px;
  align-items: center; gap: 12px; padding: 5px 0; border-radius: 6px;
  transition: background .15s;
}
.bar-row:hover { background: #f6f9fe; }
.bar-name { font-size: 13px; color: var(--c-text); text-align: right; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.bar-track { height: 18px; background: #f1f5fb; border-radius: 3px; overflow: hidden; }
.bar-fill {
  display: block; height: 100%;
  background: var(--c-primary);
  border-radius: 0 4px 4px 0;   /* 数据端 4px 圆角，基线端方角 */
  transition: width .6s ease;
}
.bar-val { font-size: 13px; font-weight: 600; color: var(--c-text-strong); text-align: right; font-variant-numeric: tabular-nums; }
.bar-val em { font-style: normal; font-size: 11px; font-weight: 400; color: var(--c-text-muted); margin-left: 2px; }
.bar-cnt { font-size: 11.5px; color: var(--c-text-muted); text-align: right; font-variant-numeric: tabular-nums; }
.empty-hint { font-size: 13px; color: var(--c-text-muted); text-align: center; padding: 18px 0; margin: 0; }

@media (max-width: 1400px) {
  .kpi-grid { grid-template-columns: repeat(3, 1fr); }
}
@media (max-width: 1100px) {
  .kpi-grid { grid-template-columns: repeat(2, 1fr); }
  .grid-2 { grid-template-columns: 1fr; }
  .bar-row { grid-template-columns: 72px 1fr 72px 42px; gap: 8px; }
}
</style>
