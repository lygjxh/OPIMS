<template>
  <div class="pc">
    <div class="page-head">
      <div>
        <h1 class="page-title">报送核查</h1>
        <p class="page-desc">按周期核查各在建项目的进度文件报送情况</p>
      </div>
      <div class="head-ops">
        <el-date-picker v-model="period" type="month" placeholder="选择周期"
          value-format="YYYY-MM" format="YYYY 年 M 月" :clearable="false"
          style="width:150px" @change="load" />
        <el-button @click="load" :loading="loading">
          <el-icon><Refresh /></el-icon>重新扫描
        </el-button>
      </div>
    </div>

    <!-- 目录缺失 -->
    <el-card v-if="errMsg" class="err-card">
      <div class="err">
        <el-icon :size="30"><FolderDelete /></el-icon>
        <p class="err-title">{{ errMsg }}</p>
        <p class="err-dir" v-if="expectedDir">期望路径：{{ expectedDir }}</p>
        <el-button type="primary" plain @click="$router.push('/files')">前往设置根目录</el-button>
      </div>
    </el-card>

    <el-alert v-else-if="warning" :title="warning" type="info" show-icon :closable="false" />

    <template v-else-if="res">
      <!-- 汇总 -->
      <div class="kpi-row">
        <div class="kpi"><span class="k-lb">在建项目</span><span class="k-v">{{ res.summary.project_count }}</span></div>
        <div class="kpi ok"><span class="k-lb">已交</span><span class="k-v">{{ res.summary.submitted }}</span></div>
        <div class="kpi late"><span class="k-lb">迟交</span><span class="k-v">{{ res.summary.late }}</span></div>
        <div class="kpi miss"><span class="k-lb">缺交</span><span class="k-v">{{ res.summary.missing }}</span></div>
        <div class="kpi"><span class="k-lb">按时率</span><span class="k-v">{{ res.summary.compliance.toFixed(0) }}%</span></div>
      </div>

      <!-- 看板：行=项目，列=文件类型 -->
      <el-card>
        <template #header>
          <div class="card-head">
            <span>报送状态看板 · {{ res.period }}</span>
            <div class="legend">
              <span v-for="s in LEGEND" :key="s.k">
                <i class="lg" :class="'st-' + s.c"></i>{{ s.k }}
              </span>
            </div>
          </div>
        </template>

        <div class="board-wrap">
          <table class="board">
            <thead>
              <tr>
                <th class="c-proj">项目</th>
                <th v-for="it in periodicItems" :key="it.code" :title="it.name">
                  {{ it.code }}
                  <span class="th-sub">{{ it.name.slice(0, 6) }}</span>
                </th>
                <th class="c-plan">进度计划</th>
                <th class="c-rate">按时率</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="r in res.rows" :key="r.short_name">
                <td class="c-proj">
                  <div class="p-name">{{ r.short_name }}</div>
                  <div class="p-country">{{ r.country || '-' }}</div>
                </td>
                <td v-for="c in r.cells" :key="c.code" class="c-cell">
                  <span class="st" :class="'st-' + statusClass(c.status)"
                    :title="cellTip(c)">
                    {{ c.status }}
                  </span>
                  <div class="c-date" v-if="c.submit_at">{{ c.submit_at }}</div>
                  <div class="c-date warn" v-else-if="!c.dir_exists">未建目录</div>
                </td>
                <td class="c-plan">
                  <template v-if="r.plans && r.plans.length">
                    <span v-for="p in r.plans" :key="p.level" class="plan-tag"
                      :title="p.file_name">{{ p.level }} v{{ p.version }}</span>
                  </template>
                  <span v-else class="muted">无</span>
                </td>
                <td class="c-rate">
                  <span :class="rateClass(r.compliance)">{{ r.compliance.toFixed(0) }}%</span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <p class="board-note">
          <el-icon><InfoFilled /></el-icon>
          周报按周计，不进月度看板；本月周报份数见下方。进度计划为事件驱动，只显示各层级当前最新版本。
        </p>
      </el-card>

      <!-- 待人工识别 -->
      <el-card v-if="res.unmatched && res.unmatched.length">
        <template #header>
          <div class="card-head">
            <span>待人工识别 · {{ res.unmatched.length }} 个文件</span>
            <span class="hint">命名不符合规范，系统无法自动归类</span>
          </div>
        </template>
        <table class="um">
          <thead><tr><th>项目</th><th>所在目录</th><th>文件名</th></tr></thead>
          <tbody>
            <tr v-for="(u, i) in res.unmatched" :key="i">
              <td>{{ u.project }}</td>
              <td class="mono">{{ u.folder }}</td>
              <td>{{ u.file_name }}</td>
            </tr>
          </tbody>
        </table>
        <p class="board-note">
          <el-icon><InfoFilled /></el-icon>
          规范格式：<code>{项目简称}-{类型代码}-{周期}[-v版本].{扩展名}</code>，
          如 <code>尼日利亚PLF项目-MPR-202607.pdf</code>
        </p>
      </el-card>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import axios from 'axios'
import { Refresh, FolderDelete, InfoFilled } from '@element-plus/icons-vue'

const period = ref('')
const res = ref<any>(null)
const loading = ref(false)
const errMsg = ref('')
const expectedDir = ref('')
const warning = ref('')

const LEGEND = [
  { k: '已交', c: 'ok' }, { k: '迟交', c: 'late' }, { k: '缺交', c: 'miss' },
]

// 看板只显示月度与季度项，周报单列（见后端说明）
const periodicItems = computed(() =>
  (res.value?.checklist || []).filter((i: any) => i.cycle !== '周'))

const statusClass = (s: string) =>
  ({ 已交: 'ok', 迟交: 'late', 缺交: 'miss' } as Record<string, string>)[s] || 'miss'

const rateClass = (v: number) => v >= 100 ? 'r-ok' : v >= 60 ? 'r-mid' : 'r-bad'

function cellTip(c: any) {
  const parts = [`${c.name}（${c.code}）`, `周期：${c.period}`, `截止：${c.deadline}`]
  if (c.file_name) parts.push(`文件：${c.file_name}`)
  if (!c.dir_exists) parts.push('存放目录尚未建立')
  return parts.join('\n')
}

async function load() {
  loading.value = true
  errMsg.value = ''; warning.value = ''
  try {
    const { data } = await axios.get('/api/progress/check', { params: { period: period.value } })
    if (data.warning) { warning.value = data.warning; res.value = null }
    else { res.value = data; period.value = data.period }
  } catch (e: any) {
    const d = e.response?.data
    errMsg.value = d?.error || '扫描失败'
    expectedDir.value = d?.expected_dir || ''
    res.value = null
  }
  loading.value = false
}

onMounted(load)
</script>

<style scoped>
.pc { display: flex; flex-direction: column; gap: 16px; }
.page-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; }
.page-title { font-size: 22px; font-weight: 700; color: var(--c-text-strong); margin: 0; }
.page-desc { font-size: 13px; color: var(--c-text-muted); margin: 4px 0 0; }
.head-ops { display: flex; gap: 8px; align-items: center; }

/* 汇总 */
.kpi-row { display: grid; grid-template-columns: repeat(5, 1fr); gap: 12px; }
.kpi {
  background: var(--c-surface); border: 1px solid var(--c-border);
  border-radius: var(--radius); padding: 12px 16px;
  display: flex; flex-direction: column; gap: 4px; box-shadow: var(--shadow-card);
}
.k-lb { font-size: 12px; color: var(--c-text-muted); }
.k-v { font-size: 24px; font-weight: 800; color: var(--c-text-strong); }
.kpi.ok .k-v { color: #0ca30c; }
.kpi.late .k-v { color: #eda100; }
.kpi.miss .k-v { color: #d03b3b; }

.card-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap; }
.hint { font-size: 12px; color: var(--c-text-muted); font-weight: 400; }
.legend { display: flex; gap: 12px; font-size: 12px; font-weight: 400; color: var(--c-text-muted); }
.legend span { display: inline-flex; align-items: center; gap: 5px; }
.lg { width: 10px; height: 10px; border-radius: 3px; display: inline-block; }

/* 看板 */
.board-wrap { overflow-x: auto; }
.board { border-collapse: collapse; font-size: 12.5px; min-width: 100%; }
.board th, .board td {
  border: 1px solid var(--c-border); padding: 7px 9px; text-align: center; white-space: nowrap;
}
.board thead th { background: #f1f5fb; color: var(--c-text-strong); font-weight: 600; }
.th-sub { display: block; font-size: 10.5px; font-weight: 400; color: var(--c-text-muted); }
.board tbody tr:hover { background: #f6f9fe; }
.c-proj { text-align: left !important; position: sticky; left: 0; background: var(--c-surface); z-index: 1; }
.board tbody tr:hover .c-proj { background: #f6f9fe; }
.p-name { font-weight: 600; color: var(--c-text-strong); }
.p-country { font-size: 11px; color: var(--c-text-muted); }
.c-cell { min-width: 76px; }
.c-date { font-size: 10.5px; color: var(--c-text-muted); margin-top: 2px; }
.c-date.warn { color: #a8730a; }

/* 状态标签：颜色 + 文字双重表达 */
.st {
  display: inline-block; padding: 2px 8px; border-radius: 10px;
  font-size: 11.5px; font-weight: 600;
}
.st-ok { background: #e8f6e8; color: #0a7a0a; }
.st-late { background: #fdf3de; color: #8a5d00; }
.st-miss { background: #fbeaea; color: #b32d2d; }
i.lg.st-ok { background: #0ca30c; }
i.lg.st-late { background: #eda100; }
i.lg.st-miss { background: #d03b3b; }

.plan-tag {
  display: inline-block; margin: 1px 2px; padding: 1px 6px;
  background: #eef4ff; color: var(--c-primary-700);
  border-radius: 4px; font-size: 11px;
}
.muted { color: var(--c-text-muted); }
.r-ok { color: #0a7a0a; font-weight: 600; }
.r-mid { color: #8a5d00; font-weight: 600; }
.r-bad { color: #b32d2d; font-weight: 600; }

.board-note {
  display: flex; align-items: center; gap: 6px; margin: 12px 0 0;
  font-size: 11.5px; color: var(--c-text-muted);
}
.board-note code { background: #f1f5f9; padding: 1px 5px; border-radius: 4px; }

/* 待人工识别 */
.um { width: 100%; border-collapse: collapse; font-size: 12.5px; }
.um th, .um td { border: 1px solid var(--c-border); padding: 6px 10px; text-align: left; }
.um th { background: #f6f9fe; color: var(--c-text-muted); font-weight: 500; }
.mono { color: var(--c-text-muted); }

.err-card { max-width: 620px; margin: 40px auto; }
.err { text-align: center; color: var(--c-text-muted); padding: 20px; }
.err-title { font-size: 15px; color: var(--c-text-strong); margin: 12px 0 6px; font-weight: 600; }
.err-dir { font-size: 12px; margin: 0 0 18px; word-break: break-all; }

@media (max-width: 1100px) { .kpi-row { grid-template-columns: repeat(3, 1fr); } }
</style>
