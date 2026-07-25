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

      <!-- 快捷入口 -->
      <el-card class="shortcuts-card">
        <template #header>{{ $t('dashboard.shortcuts') }}</template>
        <div class="shortcuts">
          <button class="shortcut" v-for="item in shortcuts" :key="item.path"
            type="button" @click="goShortcut(item)">
            <el-icon :size="22"><component :is="item.icon" /></el-icon>
            <span>{{ item.label }}</span>
          </button>
        </div>
      </el-card>
    </div>

    <!-- 合同额概览 -->
    <el-card class="amount-card">
      <template #header>
        <div class="card-head">
          <span>合同额分布 · 按国别</span>
          <div class="amount-total">
            合计 <b>{{ fmtYi(totalAmount) }}</b> 亿元 · {{ byCountry.length }} 个国别
          </div>
        </div>
      </template>

      <div v-if="byCountry.length" class="bars">
        <div v-for="c in byCountry" :key="c.country" class="bar-row"
          :title="`${c.country}：${fmtYi(c.amount)} 亿元 · ${c.count} 个项目`">
          <span class="bar-name">{{ c.country }}</span>
          <div class="bar-track">
            <span class="bar-fill" :style="{ width: amtWidth(c.amount) }"></span>
          </div>
          <span class="bar-val">{{ fmtYi(c.amount) }}<em>亿</em></span>
          <span class="bar-cnt">{{ c.count }}个</span>
        </div>
      </div>
      <p v-else class="empty-hint">暂无合同额数据</p>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import axios from 'axios'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import {
  OfficeBuilding, List, FolderOpened, WarningFilled, UserFilled, TrendCharts,
  CircleCheck, Connection, Avatar, Loading, Clock, Warning, Select,
  LocationInformation, InfoFilled,
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

/* ---- 合同额按国别聚合（单一色调，长度即量级） ---- */
const byCountry = computed(() => {
  const m = new Map<string, { country: string; amount: number; count: number }>()
  for (const p of projects.value) {
    const key = p.country || '其他'
    const cur = m.get(key) || { country: key, amount: 0, count: 0 }
    cur.amount += Number(p.contract_amount) || 0
    cur.count += 1
    m.set(key, cur)
  }
  return [...m.values()].sort((a, b) => b.amount - a.amount)
})
const totalAmount = computed(() => byCountry.value.reduce((s, c) => s + c.amount, 0))
const maxAmount = computed(() => Math.max(...byCountry.value.map(c => c.amount), 1))
function amtWidth(v: number) { return Math.max(1.5, (v / maxAmount.value) * 100) + '%' }
/** 万元 → 亿元 */
function fmtYi(wan: number) {
  const yi = wan / 10000
  return yi >= 100 ? yi.toFixed(0) : yi >= 10 ? yi.toFixed(1) : yi.toFixed(2)
}

const shortcuts = [
  { path: '/projects', label: t('menu.projects'), icon: List, ready: true },
  { path: '/files', label: t('menu.files'), icon: FolderOpened, ready: true },
  { path: '/blacklist-sub', label: t('menu.blacklistSub'), icon: WarningFilled, ready: true },
  { path: '/blacklist-person', label: t('menu.blacklistPerson'), icon: UserFilled, ready: false },
  { path: '/progress', label: t('menu.progress'), icon: TrendCharts, ready: false },
  { path: '/quality', label: t('menu.quality'), icon: CircleCheck, ready: false },
  { path: '/subcontract', label: t('menu.subcontract'), icon: Connection, ready: false },
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
    // status=all：取全部状态，用于地图国别定位与合同额统计
    const { data } = await axios.get('/api/projects', { params: { status: 'all' } })
    projects.value = data.projects || []
  } catch { projects.value = [] }

  await nextTick()
  drawMap()
})

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

/* ---- 快捷入口 ---- */
.shortcuts { display: grid; grid-template-columns: repeat(2, 1fr); gap: 11px; }
.shortcut {
  display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 8px;
  padding: 18px 8px; border-radius: 10px; cursor: pointer; font: inherit;
  background: #f6f9fe; border: 1px solid var(--c-border); color: var(--c-primary-700);
  font-size: 13px; font-weight: 500; transition: all .18s;
}
.shortcut:hover { background: var(--c-primary); color: #fff; border-color: var(--c-primary); transform: translateY(-2px); }
.shortcut:focus-visible { outline: 2px solid var(--c-primary); outline-offset: 2px; }

/* ---- 合同额条形图（单一色调；长度编码量级） ---- */
.amount-total { font-size: 12.5px; color: var(--c-text-muted); font-weight: 400; }
.amount-total b { color: var(--c-text-strong); font-size: 15px; }
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
