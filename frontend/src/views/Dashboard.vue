<template>
  <div class="dashboard">
    <div class="stat-cards">
      <el-card class="stat-card total">
        <div class="stat-num">{{ dashboard.total_projects || 0 }}</div>
        <div class="stat-label">{{ $t('dashboard.total') }}</div>
      </el-card>
      <el-card class="stat-card in-progress">
        <div class="stat-num">{{ dashboard.status_counts?.['在建'] || 0 }}</div>
        <div class="stat-label">{{ $t('project.inProgress') }}</div>
      </el-card>
      <el-card class="stat-card not-started">
        <div class="stat-num">{{ dashboard.status_counts?.['未开工'] || 0 }}</div>
        <div class="stat-label">{{ $t('project.notStarted') }}</div>
      </el-card>
      <el-card class="stat-card suspended">
        <div class="stat-num">{{ dashboard.status_counts?.['停工'] || 0 }}</div>
        <div class="stat-label">{{ $t('project.suspended') }}</div>
      </el-card>
      <el-card class="stat-card completed">
        <div class="stat-num">{{ dashboard.status_counts?.['完工'] || 0 }}</div>
        <div class="stat-label">{{ $t('project.completed') }}</div>
      </el-card>
    </div>

    <el-card class="map-card">
      <template #header>World Map</template>
      <div id="worldmap" ref="mapRef" style="height:420px"></div>
    </el-card>

    <el-card class="shortcuts-card">
      <template #header>{{ $t('dashboard.shortcuts') }}</template>
      <div class="shortcuts">
        <div class="shortcut-item" v-for="item in shortcuts" :key="item.path" @click="$router.push(item.path)">
          <el-icon :size="28"><component :is="item.icon" /></el-icon>
          <span>{{ item.label }}</span>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import axios from 'axios'
import L from 'leaflet'
import 'leaflet/dist/leaflet.css'
import { List, FolderOpened, WarningFilled, UserFilled, TrendCharts, CircleCheck, Connection, Avatar } from '@element-plus/icons-vue'

const { t } = useI18n()
const mapRef = ref<HTMLElement | null>(null)

const dashboard = ref<any>({ total_projects: 0, status_counts: {}, project_markers: [] })

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
      '在建': '#1976d2', '未开工': '#ff9800', '停工': '#f44336', '完工': '#4caf50'
    }
    markers.forEach((m: any) => {
      if (m.lat && m.lng) {
        const color = colors[m.status] || '#666'
        const circle = L.circleMarker([m.lat, m.lng], {
          radius: 7, fillColor: color, color: '#fff', weight: 1.5, fillOpacity: 0.85
        }).addTo(map!)
        circle.bindTooltip(m.short_name || m.country)
      }
    })
  }
})
</script>

<style scoped>
.dashboard { display: flex; flex-direction: column; gap: 16px; }
.stat-cards { display: flex; gap: 16px; flex-wrap: wrap; }
.stat-card { flex: 1; min-width: 140px; text-align: center; cursor: default; }
.stat-num { font-size: 32px; font-weight: 700; }
.stat-label { color: #888; margin-top: 4px; font-size: 13px; }
.total .stat-num { color: #1976d2; }
.in-progress .stat-num { color: #1976d2; }
.not-started .stat-num { color: #ff9800; }
.suspended .stat-num { color: #f44336; }
.completed .stat-num { color: #4caf50; }
.shortcuts { display: flex; gap: 16px; flex-wrap: wrap; }
.shortcut-item { display: flex; flex-direction: column; align-items: center; gap: 6px;
  padding: 16px 20px; border-radius: 8px; background: #f5f7fa; cursor: pointer;
  min-width: 90px; transition: background .2s; }
.shortcut-item:hover { background: #e3f2fd; }
</style>
