<template>
  <div class="dashboard">
    <div class="page-head">
      <div>
        <h1 class="page-title">项目总览</h1>
        <p class="page-desc">海外在建项目分布与状态一览</p>
      </div>
    </div>

    <!-- KPI 卡片 -->
    <div class="kpi-grid">
      <div v-for="k in kpis" :key="k.key" class="kpi" :style="{ '--kpi': k.color }">
        <div class="kpi-top">
          <span class="kpi-label">{{ k.label }}</span>
          <el-icon class="kpi-ic"><component :is="k.icon" /></el-icon>
        </div>
        <div class="kpi-num tnum">{{ k.value }}</div>
        <div class="kpi-bar"><span :style="{ width: barWidth(k.value) }"></span></div>
      </div>
    </div>

    <div class="grid-2">
      <!-- 地图 -->
      <el-card class="map-card">
        <template #header>
          <div class="card-head">
            <span>项目全球分布</span>
            <div class="legend">
              <span><i style="background:var(--c-status-inprogress)"></i>在建</span>
              <span><i style="background:var(--c-status-notstarted)"></i>未开工</span>
              <span><i style="background:var(--c-status-suspended)"></i>停工</span>
              <span><i style="background:var(--c-status-completed)"></i>完工</span>
            </div>
          </div>
        </template>
        <div id="worldmap" ref="mapRef" class="worldmap"></div>
      </el-card>

      <!-- 快捷入口 -->
      <el-card class="shortcuts-card">
        <template #header>{{ $t('dashboard.shortcuts') }}</template>
        <div class="shortcuts">
          <div class="shortcut" v-for="item in shortcuts" :key="item.path"
            @click="$router.push(item.path)">
            <el-icon :size="22"><component :is="item.icon" /></el-icon>
            <span>{{ item.label }}</span>
          </div>
        </div>
      </el-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import axios from 'axios'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import { OfficeBuilding, List, FolderOpened, WarningFilled, UserFilled, TrendCharts, CircleCheck, Connection, Avatar, Loading, Clock, Warning, Select } from '@element-plus/icons-vue'

const { t } = useI18n()
const mapRef = ref<HTMLElement | null>(null)
const dashboard = ref<any>({ total_projects: 0, status_counts: {}, project_markers: [] })

const kpis = computed(() => [
  { key: 'total', label: t('dashboard.total'), value: dashboard.value.total_projects || 0, color: 'var(--c-primary)', icon: OfficeBuilding },
  { key: 'in', label: t('project.inProgress'), value: dashboard.value.status_counts?.['在建'] || 0, color: 'var(--c-status-inprogress)', icon: Loading },
  { key: 'not', label: t('project.notStarted'), value: dashboard.value.status_counts?.['未开工'] || 0, color: 'var(--c-status-notstarted)', icon: Clock },
  { key: 'sus', label: t('project.suspended'), value: dashboard.value.status_counts?.['停工'] || 0, color: 'var(--c-status-suspended)', icon: Warning },
  { key: 'done', label: t('project.completed'), value: dashboard.value.status_counts?.['完工'] || 0, color: 'var(--c-status-completed)', icon: Select },
])

function barWidth(v: number) {
  const total = dashboard.value.total_projects || 0
  if (!total) return '0%'
  return Math.min(100, Math.round((v / total) * 100)) + '%'
}

const shortcuts = [
  { path: '/projects', label: t('menu.projects'), icon: List },
  { path: '/files', label: t('menu.files'), icon: FolderOpened },
  { path: '/blacklist-sub', label: t('menu.blacklistSub'), icon: WarningFilled },
  { path: '/blacklist-person', label: t('menu.blacklistPerson'), icon: UserFilled },
  { path: '/progress', label: t('menu.progress'), icon: TrendCharts },
  { path: '/quality', label: t('menu.quality'), icon: CircleCheck },
  { path: '/subcontract', label: t('menu.subcontract'), icon: Connection },
  { path: '/personnel', label: t('menu.personnel'), icon: Avatar },
]

let map: L.Map | null = null

onMounted(async () => {
  try {
    const { data } = await axios.get('/api/dashboard')
    dashboard.value = data
  } catch (e) { /* no data yet */ }

  await nextTick()
  if (mapRef.value) {
    map = L.map('worldmap').setView([20, 0], 2)
    L.tileLayer('https://{s}.basemaps.cartocdn.com/light_all/{z}/{x}/{y}{r}.png', {
      attribution: '&copy; OpenStreetMap',
    }).addTo(map)

    const markers = dashboard.value.project_markers || []
    const colors: Record<string, string> = {
      '在建': '#2563eb', '未开工': '#d97706', '停工': '#dc2626', '完工': '#16a34a'
    }
    markers.forEach((m: any) => {
      if (m.lat && m.lng) {
        const color = colors[m.status] || '#666'
        const circle = L.circleMarker([m.lat, m.lng], {
          radius: 7, fillColor: color, color: '#fff', weight: 1.5, fillOpacity: 0.9
        }).addTo(map!)
        circle.bindTooltip(m.short_name || m.country)
      }
    })
  }
})
</script>

<style scoped>
.dashboard { display: flex; flex-direction: column; gap: 20px; }
.page-title { font-size: 22px; font-weight: 700; color: var(--c-text-strong); margin: 0; }
.page-desc { font-size: 13px; color: var(--c-text-muted); margin: 4px 0 0; }

/* KPI */
.kpi-grid { display: grid; grid-template-columns: repeat(5, 1fr); gap: 16px; }
.kpi {
  background: var(--c-surface); border: 1px solid var(--c-border);
  border-radius: var(--radius); padding: 18px 18px 14px;
  box-shadow: var(--shadow-card); position: relative; overflow: hidden;
  transition: transform .2s, box-shadow .2s;
}
.kpi::before {
  content: ''; position: absolute; left: 0; top: 0; bottom: 0; width: 4px; background: var(--kpi);
}
.kpi:hover { transform: translateY(-3px); box-shadow: var(--shadow-pop); }
.kpi-top { display: flex; align-items: center; justify-content: space-between; }
.kpi-label { font-size: 13px; color: var(--c-text-muted); font-weight: 500; }
.kpi-ic { font-size: 18px; color: var(--kpi); opacity: .85; }
.kpi-num { font-size: 34px; font-weight: 800; color: var(--c-text-strong); line-height: 1.1; margin: 10px 0 8px; }
.kpi-bar { height: 4px; border-radius: 3px; background: #eef2f8; overflow: hidden; }
.kpi-bar span { display: block; height: 100%; border-radius: 3px; background: var(--kpi); transition: width .5s ease; }

/* 布局 */
.grid-2 { display: grid; grid-template-columns: 2fr 1fr; gap: 16px; align-items: start; }
.card-head { display: flex; align-items: center; justify-content: space-between; }
.legend { display: flex; gap: 14px; font-size: 12px; font-weight: 400; color: var(--c-text-muted); }
.legend span { display: inline-flex; align-items: center; gap: 5px; }
.legend i { width: 9px; height: 9px; border-radius: 50%; display: inline-block; }
.worldmap { height: 440px; border-radius: 8px; }

/* 快捷入口 */
.shortcuts { display: grid; grid-template-columns: repeat(2, 1fr); gap: 12px; }
.shortcut {
  display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 8px;
  padding: 20px 10px; border-radius: 10px; cursor: pointer;
  background: #f6f9fe; border: 1px solid var(--c-border); color: var(--c-primary-700);
  font-size: 13px; font-weight: 500; transition: all .18s;
}
.shortcut:hover { background: var(--c-primary); color: #fff; border-color: var(--c-primary); transform: translateY(-2px); }

@media (max-width: 1200px) {
  .kpi-grid { grid-template-columns: repeat(2, 1fr); }
  .grid-2 { grid-template-columns: 1fr; }
}
</style>
