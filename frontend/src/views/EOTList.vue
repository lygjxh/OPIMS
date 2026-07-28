<template>
  <div class="eot">
    <div class="kpi-row">
      <div class="kpi"><span class="k-lb">索赔事项</span><span class="k-v">{{ s.total }}</span></div>
      <div class="kpi danger"><span class="k-lb">CN 已逾期</span><span class="k-v">{{ s.overdue_cn }}</span></div>
      <div class="kpi danger"><span class="k-lb">PCO 已逾期</span><span class="k-v">{{ s.overdue_pco }}</span></div>
      <div class="kpi"><span class="k-lb">累计请求工期</span><span class="k-v">{{ s.total_days }}<em>天</em></span></div>
      <div class="kpi"><span class="k-lb">累计请求金额</span><span class="k-v">{{ fmtAmt(s.total_amount) }}</span></div>
      <div class="kpi"><span class="k-lb">已关闭</span><span class="k-v">{{ s.closed_count }}</span></div>
    </div>

    <el-alert v-if="warnings.length" type="warning" show-icon :closable="false"
      :title="`${warnings.length} 条索赔台账无法解析`">
      <ul class="warn-list"><li v-for="(w,i) in warnings" :key="i">{{ w }}</li></ul>
    </el-alert>

    <el-card>
      <template #header>
        <div class="card-head">
          <span>索赔跟踪 · 共 {{ items.length }} 条</span>
          <span class="hint">逾期的排最前，其次按索赔金额降序</span>
        </div>
      </template>

      <el-empty v-if="!items.length"
        description="暂无索赔记录。请确认各项目已在 05.Schedule/EOT/ 下提交附件 D" />

      <table v-else class="tbl">
        <thead>
          <tr><th>项目</th><th>索赔事项</th><th>类别</th><th>发现日</th>
            <th>CN 发出 / 应发</th><th>PCO 实交 / 应交</th><th>影响天数</th>
            <th>索赔金额</th><th>状态</th><th>逾期</th></tr>
        </thead>
        <tbody>
          <tr v-for="(it,i) in items" :key="i" :class="{ 'row-bad': it.overdue }">
            <td class="nw">{{ it.project }}</td>
            <td class="c-sub" :title="it.subject">{{ it.subject }}</td>
            <td class="nw"><b>{{ it.category }}</b><div class="sub">{{ it.cat_name }}</div></td>
            <td class="nw">{{ it.found_at || '-' }}</td>
            <td class="nw">
              {{ it.cn_at || '未发出' }}
              <div class="sub">应发 {{ it.cn_due_at || '-' }}</div>
            </td>
            <td class="nw">
              {{ it.pco_at || '未提交' }}
              <div class="sub">应交 {{ it.pco_due_at || '-' }}</div>
            </td>
            <td class="num">{{ it.days || '-' }}</td>
            <td class="num">{{ it.amount ? fmtAmt(it.amount) : '-' }}</td>
            <td class="nw">{{ it.status || '-' }}</td>
            <td class="nw"><span v-if="it.overdue" class="bad">{{ it.overdue }}</span><span v-else class="ok">—</span></td>
          </tr>
        </tbody>
      </table>

      <p class="note">
        <el-icon><InfoFilled /></el-icon>
        CN 应发日 = 发现日 + 10 日历日；PCO 应交日 = CN 发出日 + 21 日历日（细则 5.1）。
        逾期即丧失索赔权，请优先处理标红项。
      </p>
    </el-card>

    <div class="grid-2" v-if="byStatus.length || byCat.length">
      <el-card v-if="byStatus.length">
        <template #header>按状态分布</template>
        <table class="mini"><tbody>
          <tr v-for="c in byStatus" :key="c.name">
            <td>{{ c.name }}</td><td class="num">{{ c.count }} 条</td>
            <td class="num">{{ c.days || 0 }} 天</td><td class="num">{{ fmtAmt(c.amount) }}</td>
          </tr>
        </tbody></table>
      </el-card>
      <el-card v-if="byCat.length">
        <template #header>按索赔类别分布</template>
        <table class="mini"><tbody>
          <tr v-for="c in byCat" :key="c.name">
            <td>{{ c.name }}</td><td class="num">{{ c.count }} 条</td>
            <td class="num">{{ c.days || 0 }} 天</td><td class="num">{{ fmtAmt(c.amount) }}</td>
          </tr>
        </tbody></table>
      </el-card>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import axios from 'axios'
import { InfoFilled } from '@element-plus/icons-vue'

const items = ref<any[]>([])
const warnings = ref<string[]>([])
const byStatus = ref<any[]>([])
const byCat = ref<any[]>([])
const s = ref<any>({ total: 0, overdue_cn: 0, overdue_pco: 0, total_days: 0, total_amount: 0, closed_count: 0 })

function fmtAmt(v: number) {
  if (!v) return '-'
  if (v >= 10000) return (v / 10000).toFixed(2) + ' 万'
  return v.toLocaleString('en-US')
}

async function load() {
  try {
    const { data } = await axios.get('/api/eot')
    items.value = data.items || []
    warnings.value = data.warnings || []
    byStatus.value = data.by_status || []
    byCat.value = data.by_cat || []
    s.value = data.summary || s.value
  } catch { items.value = [] }
}
onMounted(load)
defineExpose({ load })
</script>

<style scoped>
.eot { display: flex; flex-direction: column; gap: 14px; }
.kpi-row { display: grid; grid-template-columns: repeat(6, 1fr); gap: 12px; }
.kpi {
  background: var(--c-surface); border: 1px solid var(--c-border);
  border-radius: var(--radius); padding: 12px 14px;
  display: flex; flex-direction: column; gap: 3px; box-shadow: var(--shadow-card);
}
.k-lb { font-size: 11.5px; color: var(--c-text-muted); }
.k-v { font-size: 22px; font-weight: 800; color: var(--c-text-strong); }
.k-v em { font-style: normal; font-size: 12px; font-weight: 400; color: var(--c-text-muted); margin-left: 2px; }
.kpi.danger { border-color: #e6b8b8; background: #fdf5f5; }
.kpi.danger .k-v { color: #b32d2d; }

.card-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap; }
.hint { font-size: 11.5px; color: var(--c-text-muted); font-weight: 400; }
.warn-list { margin: 6px 0 0; padding-left: 18px; font-size: 12px; line-height: 1.8; }

.tbl { width: 100%; border-collapse: collapse; font-size: 12.5px; }
.tbl th, .tbl td { border: 1px solid var(--c-border); padding: 6px 9px; text-align: left; vertical-align: top; }
.tbl thead th { background: #f1f5fb; color: var(--c-text-strong); font-weight: 600; white-space: nowrap; }
.row-bad { background: #fdf6f6; }
.nw { white-space: nowrap; }
.num { text-align: right; white-space: nowrap; font-variant-numeric: tabular-nums; }
.sub { font-size: 10.5px; color: var(--c-text-muted); margin-top: 1px; }
.c-sub { max-width: 230px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.bad { color: #b32d2d; font-weight: 600; }
.ok { color: var(--c-text-muted); }

.grid-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
.mini { width: 100%; border-collapse: collapse; font-size: 12.5px; }
.mini td { border-bottom: 1px solid var(--c-border); padding: 6px 8px; }

.note { display: flex; align-items: center; gap: 6px; margin: 12px 0 0; font-size: 11.5px; color: var(--c-text-muted); }
@media (max-width: 1200px) { .kpi-row { grid-template-columns: repeat(3,1fr); } .grid-2 { grid-template-columns: 1fr; } }
</style>
