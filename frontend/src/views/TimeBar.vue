<template>
  <div class="tb">
    <div class="page-head">
      <div>
        <h1 class="page-title">合同时效预警</h1>
        <p class="page-desc">
          各在建项目的 Time Bar 倒计时 ·
          <strong>逾期即丧失索赔权，不可挽回</strong>
        </p>
      </div>
      <el-button @click="load" :loading="loading">
        <el-icon><Refresh /></el-icon>重新扫描
      </el-button>
    </div>

    <el-card v-if="errMsg" class="err-card">
      <div class="err">
        <el-icon :size="30"><FolderDelete /></el-icon>
        <p class="err-title">{{ errMsg }}</p>
        <p class="err-dir" v-if="expectedDir">期望路径：{{ expectedDir }}</p>
        <el-button type="primary" plain @click="$router.push('/files')">前往设置根目录</el-button>
      </div>
    </el-card>

    <el-alert v-else-if="warning" :title="warning" type="info" show-icon :closable="false" />

    <template v-else>
      <div class="kpi-row">
        <div class="kpi danger"><span class="k-lb">已逾期</span><span class="k-v">{{ s.overdue }}</span></div>
        <div class="kpi red"><span class="k-lb">紧急（&lt;7天）</span><span class="k-v">{{ s.red }}</span></div>
        <div class="kpi amber"><span class="k-lb">关注（7–14天）</span><span class="k-v">{{ s.amber }}</span></div>
        <div class="kpi green"><span class="k-lb">正常（&gt;14天）</span><span class="k-v">{{ s.green }}</span></div>
        <div class="kpi"><span class="k-lb">已完成</span><span class="k-v">{{ s.done }}</span></div>
        <div class="kpi"><span class="k-lb">覆盖项目</span><span class="k-v">{{ s.projects }}</span></div>
      </div>

      <!-- 解析告警：不能静默忽略，否则看板漏项而无人察觉 -->
      <el-alert v-if="warnings.length" type="warning" show-icon :closable="false"
        :title="`${warnings.length} 条时效日历数据无法解析，可能有遗漏`">
        <ul class="warn-list">
          <li v-for="(w, i) in warnings" :key="i">{{ w }}</li>
        </ul>
      </el-alert>

      <el-card>
        <template #header>
          <div class="card-head">
            <span>时效清单 · 共 {{ s.total }} 条</span>
            <div class="filters">
              <el-radio-group v-model="filter" size="small">
                <el-radio-button label="active">仅未完成</el-radio-button>
                <el-radio-button label="all">全部</el-radio-button>
              </el-radio-group>
              <span class="legend">
                <i class="lg l-red"></i>&lt;7天　<i class="lg l-amber"></i>7–14天　<i class="lg l-green"></i>&gt;14天
              </span>
            </div>
          </div>
        </template>

        <el-empty v-if="!shown.length" description="暂无时效记录。请确认各项目已在 05.Schedule/Time Bar/ 下提交附件 E" />

        <table v-else class="tbl">
          <thead>
            <tr>
              <th>剩余</th><th>项目</th><th>时效事项</th><th>合同条款</th>
              <th>触发事件</th><th>触发日期</th><th>时限</th><th>到期日</th>
              <th>责任人</th><th>状态</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(it, i) in shown" :key="i" :class="'row-' + lightKey(it.light)">
              <td class="c-rem">
                <span class="badge" :class="'b-' + lightKey(it.light)">
                  {{ remText(it) }}
                </span>
              </td>
              <td class="c-proj">{{ it.project }}</td>
              <td class="c-code">
                <b>{{ it.code }}</b>
                <div class="sub">{{ it.code_name }}</div>
              </td>
              <td>{{ it.clause || '-' }}</td>
              <td class="c-event" :title="it.event">{{ it.event || '-' }}</td>
              <td class="c-date">{{ it.trigger_at }}</td>
              <td class="c-date">{{ it.days }} 日历日</td>
              <td class="c-date"><b>{{ it.due_at }}</b></td>
              <td>{{ it.owner || '-' }}</td>
              <td>{{ it.status || '-' }}</td>
            </tr>
          </tbody>
        </table>

        <p class="note">
          <el-icon><InfoFilled /></el-icon>
          时限一律按<b>日历日</b>计算，不扣除周末与节假日（细则 V1.1 第 5.1 节）。
          剩余天数每次打开实时重算，不做缓存。数据源为各项目 05.Schedule/Time Bar/ 下的附件 E。
        </p>
      </el-card>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import axios from 'axios'
import { Refresh, FolderDelete, InfoFilled } from '@element-plus/icons-vue'

const items = ref<any[]>([])
const warnings = ref<string[]>([])
const s = ref<any>({ total: 0, overdue: 0, red: 0, amber: 0, green: 0, done: 0, projects: 0 })
const loading = ref(false)
const errMsg = ref('')
const expectedDir = ref('')
const warning = ref('')
const filter = ref<'active' | 'all'>('active')

const shown = computed(() =>
  filter.value === 'all' ? items.value : items.value.filter(i => i.light !== '完成'))

const lightKey = (l: string) =>
  ({ 红: 'red', 黄: 'amber', 绿: 'green', 完成: 'done' } as Record<string, string>)[l] || 'green'

function remText(it: any) {
  if (it.light === '完成') return '已完成'
  if (it.remaining < 0) return `逾期 ${-it.remaining} 天`
  if (it.remaining === 0) return '今日到期'
  return `剩 ${it.remaining} 天`
}

async function load() {
  loading.value = true
  errMsg.value = ''; warning.value = ''
  try {
    const { data } = await axios.get('/api/timebar')
    if (data.warning) { warning.value = data.warning; items.value = [] }
    else {
      items.value = data.items || []
      warnings.value = data.warnings || []
      s.value = data.summary || s.value
    }
  } catch (e: any) {
    const d = e.response?.data
    errMsg.value = d?.error || '扫描失败'
    expectedDir.value = d?.expected_dir || ''
  }
  loading.value = false
}

onMounted(load)
</script>

<style scoped>
.tb { display: flex; flex-direction: column; gap: 16px; }
.page-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; }
.page-title { font-size: 22px; font-weight: 700; color: var(--c-text-strong); margin: 0; }
.page-desc { font-size: 13px; color: var(--c-text-muted); margin: 4px 0 0; }
.page-desc strong { color: #b32d2d; }

.kpi-row { display: grid; grid-template-columns: repeat(6, 1fr); gap: 12px; }
.kpi {
  background: var(--c-surface); border: 1px solid var(--c-border);
  border-radius: var(--radius); padding: 12px 14px;
  display: flex; flex-direction: column; gap: 3px; box-shadow: var(--shadow-card);
}
.k-lb { font-size: 11.5px; color: var(--c-text-muted); }
.k-v { font-size: 24px; font-weight: 800; color: var(--c-text-strong); }
.kpi.danger { border-color: #e6b8b8; background: #fdf5f5; }
.kpi.danger .k-v, .kpi.red .k-v { color: #b32d2d; }
.kpi.amber .k-v { color: #8a5d00; }
.kpi.green .k-v { color: #0a7a0a; }

.card-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap; }
.filters { display: flex; align-items: center; gap: 14px; }
.legend { font-size: 11.5px; color: var(--c-text-muted); font-weight: 400; }
.lg { width: 9px; height: 9px; border-radius: 2px; display: inline-block; margin-right: 3px; }
.l-red { background: #d03b3b; } .l-amber { background: #eda100; } .l-green { background: #0ca30c; }

.warn-list { margin: 6px 0 0; padding-left: 18px; font-size: 12px; line-height: 1.8; }

.tbl { width: 100%; border-collapse: collapse; font-size: 12.5px; }
.tbl th, .tbl td { border: 1px solid var(--c-border); padding: 7px 9px; text-align: left; vertical-align: top; }
.tbl thead th { background: #f1f5fb; color: var(--c-text-strong); font-weight: 600; white-space: nowrap; }
.tbl tbody tr:hover { background: #f6f9fe; }
.row-red { background: #fdf6f6; }
.row-amber { background: #fdfaf3; }
.row-done { opacity: .55; }

.badge {
  display: inline-block; padding: 3px 9px; border-radius: 11px;
  font-size: 11.5px; font-weight: 700; white-space: nowrap;
}
.b-red { background: #fbeaea; color: #b32d2d; }
.b-amber { background: #fdf3de; color: #8a5d00; }
.b-green { background: #e8f6e8; color: #0a7a0a; }
.b-done { background: #eef1f5; color: #8894a8; }

.c-rem { white-space: nowrap; }
.c-proj { white-space: nowrap; font-weight: 500; }
.c-code b { color: var(--c-primary-700); }
.c-code .sub { font-size: 10.5px; color: var(--c-text-muted); margin-top: 1px; }
.c-event { max-width: 260px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.c-date { white-space: nowrap; }

.note {
  display: flex; align-items: center; gap: 6px; margin: 12px 0 0;
  font-size: 11.5px; color: var(--c-text-muted);
}
.err-card { max-width: 620px; margin: 40px auto; }
.err { text-align: center; color: var(--c-text-muted); padding: 20px; }
.err-title { font-size: 15px; color: var(--c-text-strong); margin: 12px 0 6px; font-weight: 600; }
.err-dir { font-size: 12px; margin: 0 0 18px; word-break: break-all; }

@media (max-width: 1200px) { .kpi-row { grid-template-columns: repeat(3, 1fr); } }
</style>
