<template>
  <div class="pi">
    <div class="bar">
      <el-date-picker v-model="period" type="month" value-format="YYYY-MM"
        format="YYYY 年 M 月" :clearable="false" style="width:150px" @change="load" />
      <el-button @click="load" :loading="loading"><el-icon><Refresh /></el-icon>重新扫描</el-button>
      <span class="hint">数据源：各项目 06.Report/Monthly Report/ 下的附件 B-1（{项目简称}-MPD-YYYYMM.xlsx）</span>
    </div>

    <div class="kpi-row">
      <div class="kpi red"><span class="k-lb">红（严重）</span><span class="k-v">{{ s.red }}</span></div>
      <div class="kpi orange"><span class="k-lb">橙（预警）</span><span class="k-v">{{ s.orange }}</span></div>
      <div class="kpi yellow"><span class="k-lb">黄（关注）</span><span class="k-v">{{ s.yellow }}</span></div>
      <div class="kpi green"><span class="k-lb">绿（正常）</span><span class="k-v">{{ s.green }}</span></div>
      <div class="kpi"><span class="k-lb">已报数据</span><span class="k-v">{{ s.total }}<em>/{{ projectCount }}</em></span></div>
      <div class="kpi" :class="{ warn: s.mismatch }">
        <span class="k-lb">自评不一致</span><span class="k-v">{{ s.mismatch }}</span>
      </div>
    </div>

    <el-alert v-if="warnings.length" type="warning" show-icon :closable="false"
      :title="`${warnings.length} 条数据存在问题`">
      <ul class="warn-list"><li v-for="(w,i) in warnings" :key="i">{{ w }}</li></ul>
    </el-alert>

    <el-card>
      <template #header>
        <div class="card-head">
          <span>进度指标与风险灯 · {{ period }}</span>
          <span class="hint">按细则 4.1 自动判定，取最严重项</span>
        </div>
      </template>

      <el-empty v-if="!items.length"
        description="该周期尚无项目提交进度数据表（附件 B-1）" />

      <table v-else class="tbl">
        <thead>
          <tr><th>风险灯</th><th>项目</th><th>关键路径</th><th>SPI</th>
            <th>累计完成率</th><th>本期完成率</th><th>判定依据</th><th>应采取的措施</th><th>自评</th></tr>
        </thead>
        <tbody>
          <tr v-for="(it,i) in items" :key="i" :class="'row-' + key(it.light)">
            <td><span class="lamp" :class="'lp-' + key(it.light)">{{ it.light }}</span></td>
            <td class="nw">
              {{ it.project }}
              <div class="sub">数据日期 {{ it.data_date || '-' }}</div>
            </td>
            <td class="num" :class="{ bad: it.lag_days > 7 }">
              {{ it.lag_days > 0 ? '滞后 ' + it.lag_days + ' 天' : (it.lag_days < 0 ? '提前 ' + (-it.lag_days) + ' 天' : '持平') }}
            </td>
            <td class="num">
              <span v-if="it.spi_valid" :class="{ bad: it.spi < 0.9 }">{{ it.spi.toFixed(2) }}</span>
              <span v-else class="muted" title="累计计划产值缺失，无法计算">—</span>
            </td>
            <td class="num">{{ it.rate_cum ? it.rate_cum.toFixed(1) + '%' : '-' }}</td>
            <td class="num">{{ it.rate_period ? it.rate_period.toFixed(1) + '%' : '-' }}</td>
            <td class="c-why">{{ it.light_why }}</td>
            <td class="c-act">{{ it.action }}</td>
            <td class="nw">
              <span v-if="!it.self_light" class="muted">未填</span>
              <span v-else-if="it.mismatch" class="mismatch" :title="`项目自评「${it.self_light}」与系统判定「${it.light}」不一致，建议核实`">
                {{ it.self_light }} ⚠
              </span>
              <span v-else class="ok">{{ it.self_light }} ✓</span>
            </td>
          </tr>
        </tbody>
      </table>

      <p class="note">
        <el-icon><InfoFilled /></el-icon>
        SPI 由系统按「累计实际产值 ÷ 累计计划产值」统一计算，项目部不填——各自计算口径不同则无法横向比较。
        自评与系统判定不一致的项目已标出，通常意味着数据填报有误或存在系统未覆盖的特殊情况。
      </p>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import axios from 'axios'
import { Refresh, InfoFilled } from '@element-plus/icons-vue'

const period = ref('')
const items = ref<any[]>([])
const warnings = ref<string[]>([])
const projectCount = ref(0)
const loading = ref(false)
const s = ref<any>({ total: 0, green: 0, yellow: 0, orange: 0, red: 0, mismatch: 0 })

const key = (l: string) =>
  ({ 红: 'red', 橙: 'orange', 黄: 'yellow', 绿: 'green' } as Record<string, string>)[l] || 'green'

async function load() {
  loading.value = true
  try {
    const { data } = await axios.get('/api/progress/indicators', { params: { period: period.value } })
    items.value = data.items || []
    warnings.value = data.warnings || []
    s.value = data.summary || s.value
    projectCount.value = data.project_count || 0
    period.value = data.period
  } catch { items.value = [] }
  loading.value = false
}
onMounted(load)
defineExpose({ load })
</script>

<style scoped>
.pi { display: flex; flex-direction: column; gap: 14px; }
.bar { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.hint { font-size: 11.5px; color: var(--c-text-muted); }

.kpi-row { display: grid; grid-template-columns: repeat(6, 1fr); gap: 12px; }
.kpi {
  background: var(--c-surface); border: 1px solid var(--c-border);
  border-radius: var(--radius); padding: 12px 14px;
  display: flex; flex-direction: column; gap: 3px; box-shadow: var(--shadow-card);
}
.k-lb { font-size: 11.5px; color: var(--c-text-muted); }
.k-v { font-size: 22px; font-weight: 800; color: var(--c-text-strong); }
.k-v em { font-style: normal; font-size: 13px; font-weight: 400; color: var(--c-text-muted); }
.kpi.red .k-v { color: #b32d2d; }
.kpi.orange .k-v { color: #a8600a; }
.kpi.yellow .k-v { color: #8a5d00; }
.kpi.green .k-v { color: #0a7a0a; }
.kpi.warn { border-color: #f0e3c5; background: #fdfaf3; }

.card-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap; }
.warn-list { margin: 6px 0 0; padding-left: 18px; font-size: 12px; line-height: 1.8; }

.tbl { width: 100%; border-collapse: collapse; font-size: 12.5px; }
.tbl th, .tbl td { border: 1px solid var(--c-border); padding: 6px 9px; text-align: left; vertical-align: top; }
.tbl thead th { background: #f1f5fb; color: var(--c-text-strong); font-weight: 600; white-space: nowrap; }
.row-red { background: #fdf6f6; }
.row-orange { background: #fdf9f4; }

/* 灯必须带文字，不能只靠颜色（红绿色盲无法区分） */
.lamp { display: inline-block; padding: 3px 11px; border-radius: 11px; font-size: 12px; font-weight: 700; }
.lp-red { background: #fbeaea; color: #b32d2d; }
.lp-orange { background: #fdf0e2; color: #a8600a; }
.lp-yellow { background: #fdf3de; color: #8a5d00; }
.lp-green { background: #e8f6e8; color: #0a7a0a; }

.nw { white-space: nowrap; }
.num { text-align: right; white-space: nowrap; font-variant-numeric: tabular-nums; }
.num.bad { color: #b32d2d; font-weight: 600; }
.sub { font-size: 10.5px; color: var(--c-text-muted); margin-top: 1px; font-weight: 400; }
.c-why { max-width: 200px; color: var(--c-text); }
.c-act { max-width: 220px; color: var(--c-text-muted); font-size: 11.5px; }
.muted { color: var(--c-text-muted); }
.mismatch { color: #a8600a; font-weight: 600; cursor: help; }
.ok { color: #0a7a0a; }

.note { display: flex; align-items: flex-start; gap: 6px; margin: 12px 0 0; font-size: 11.5px; color: var(--c-text-muted); line-height: 1.7; }
@media (max-width: 1200px) { .kpi-row { grid-template-columns: repeat(3,1fr); } }
</style>
