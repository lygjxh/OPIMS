<template>
  <div class="cp">
    <div class="page-head">
      <div>
        <h1 class="page-title">国别档案</h1>
        <p class="page-desc">各国出入境政策一览 · 数据来自政策库，在 Obsidian 中维护</p>
      </div>
      <el-button v-if="!errMsg" text type="primary" @click="showInternal">
        <el-icon><Document /></el-icon>公司内部制度摘要
      </el-button>
    </div>

    <!-- 目录未配置：给出明确指引，而不是白屏 -->
    <el-card v-if="errMsg" class="err-card">
      <div class="err">
        <el-icon :size="30"><FolderDelete /></el-icon>
        <p class="err-title">{{ errMsg }}</p>
        <p class="err-dir" v-if="expectedDir">期望路径：{{ expectedDir }}</p>
        <el-button type="primary" plain @click="$router.push('/files')">前往设置根目录</el-button>
      </div>
    </el-card>

    <template v-else>
      <div class="stat-bar" v-if="list.length">
        <span>共 <b>{{ list.length }}</b> 个国别档案</span>
        <span v-if="staleCount" class="warn">
          <el-icon><Warning /></el-icon>{{ staleCount }} 个超过 90 天未更新，建议核查
        </span>
      </div>

      <div class="grid" v-loading="loading">
        <article v-for="c in list" :key="c.country" class="card"
          :style="{ '--rk': riskColor(c.risk_level) }" @click="openDetail(c)">
          <header class="card-h">
            <div class="c-name">
              {{ c.country }}
              <span v-if="c.country_raw !== c.country" class="c-alias">{{ c.country_raw }}</span>
            </div>
            <span class="risk" :class="'risk-' + riskKey(c.risk_level)">
              <i class="risk-dot"></i>{{ c.risk_level || '未评级' }}风险
            </span>
          </header>

          <dl class="kv">
            <div><dt>签证类型</dt><dd>{{ c.visa_type || '-' }}</dd></div>
            <div><dt>预计周期</dt><dd>{{ c.total_cycle || '-' }}</dd></div>
          </dl>

          <footer class="card-f">
            <span class="upd" :class="{ stale: c.is_stale }">
              <el-icon><Clock /></el-icon>
              更新于 {{ c.updated_at || '未填写' }}
              <template v-if="c.is_stale">（{{ c.stale_days }} 天前，建议核查）</template>
            </span>
            <span class="proj" v-if="projectCount[c.country] !== undefined">
              <el-icon><OfficeBuilding /></el-icon>{{ projectCount[c.country] }} 个项目
            </span>
          </footer>
        </article>
      </div>

      <el-empty v-if="!loading && !list.length" description="政策库中暂无国别档案" />
    </template>

    <!-- 详情 -->
    <el-drawer v-model="detailVisible" :title="detail?.country + ' · 出入境政策'" size="60%">
      <div v-if="detail" class="detail">
        <div class="d-alert" :class="'risk-' + riskKey(detail.risk_level)">
          <strong>{{ detail.risk_level || '未评级' }}风险</strong>
          <span>{{ detail.risk_note }}</span>
        </div>
        <p v-if="detail.is_stale" class="d-stale">
          <el-icon><Warning /></el-icon>
          该档案已 {{ detail.stale_days }} 天未更新，出入境政策变化较快，建议核查后再据此排期。
        </p>

        <table class="meta-table">
          <tbody>
            <tr v-for="f in metaFields" :key="f.label">
              <th>{{ f.label }}</th>
              <td>{{ f.value || '-' }}</td>
            </tr>
          </tbody>
        </table>

        <div class="md" v-html="detail.body_html"></div>
      </div>
    </el-drawer>

    <!-- 内部制度 -->
    <el-drawer v-model="internalVisible" title="公司内部管理制度摘要" size="55%">
      <p class="d-stale">
        <el-icon><InfoFilled /></el-icon>
        目标国政策与公司内部流程需叠加计算，才是真实的人员到岗周期。
      </p>
      <div class="md" v-html="internalHTML"></div>
    </el-drawer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import axios from 'axios'
import { ElMessage } from 'element-plus'
import { Clock, Warning, OfficeBuilding, Document, FolderDelete, InfoFilled } from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const list = ref<any[]>([])
const loading = ref(false)
const errMsg = ref('')
const expectedDir = ref('')
const projectCount = ref<Record<string, number>>({})

const detailVisible = ref(false)
const detail = ref<any>(null)
const internalVisible = ref(false)
const internalHTML = ref('')

const staleCount = computed(() => list.value.filter(c => c.is_stale).length)

/* 风险等级配色：与项目既有约定一致，颜色必须配文字标签，
   不能只靠颜色（红绿在红绿色盲下无法区分）。 */
const RISK = { 高: '#d03b3b', 中: '#eda100', 低: '#0ca30c' } as Record<string, string>
const riskColor = (lv: string) => RISK[lv] || '#94a3b8'
const riskKey = (lv: string) => ({ 高: 'high', 中: 'mid', 低: 'low' } as Record<string, string>)[lv] || 'na'

const metaFields = computed(() => {
  const m = detail.value?.meta || {}
  return [
    { label: '签证类型', value: m.visa_type },
    { label: '适用人员', value: (m.person_types || []).join('、') },
    { label: '办理机构', value: m.authority },
    { label: '办理方式', value: m.method },
    { label: '办理时限', value: m.duration },
    { label: '预计办理周期', value: m.total_cycle },
    { label: '停留/有效期限', value: m.stay_validity },
    { label: '是否需健康证明', value: m.need_health },
    { label: '是否需无犯罪记录', value: m.need_no_crime },
    { label: '工作许可单独办理', value: m.separate_permit },
    { label: '家属随行政策', value: m.family_policy },
    { label: '签证大致费用', value: m.cost },
    { label: '信息来源', value: m.source },
  ]
})

async function load() {
  loading.value = true
  try {
    const { data } = await axios.get('/api/policy/countries')
    list.value = data.countries || []
    errMsg.value = ''
    loadProjectCounts()
    openFromQuery()
  } catch (e: any) {
    const d = e.response?.data
    errMsg.value = d?.error || '读取政策库失败'
    expectedDir.value = d?.expected_dir || ''
    list.value = []
  }
  loading.value = false
}

/** 关联 OPIMS 项目数据，让政策与业务联动 */
async function loadProjectCounts() {
  try {
    const { data } = await axios.get('/api/projects', { params: { status: 'all' } })
    const m: Record<string, number> = {}
    for (const p of data.projects || []) {
      const k = p.country || '其他'
      m[k] = (m[k] || 0) + 1
    }
    // 无项目的国别显示 0，而不是不显示
    for (const c of list.value) if (m[c.country] === undefined) m[c.country] = 0
    projectCount.value = m
  } catch { /* 关联失败不影响政策展示 */ }
}

async function openDetail(c: any) {
  try {
    const { data } = await axios.get('/api/policy/country/' + encodeURIComponent(c.country))
    detail.value = data
    detailVisible.value = true
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '读取该国档案失败')
  }
}

/** 支持从别处（项目详情、首页地图）带 ?country=xxx 直接打开某国档案 */
async function openFromQuery() {
  const c = route.query.country
  if (typeof c !== 'string' || !c) return
  await openDetail({ country: c })
}
// 抽屉关闭时清掉 query，避免刷新页面又自动弹出
watch(detailVisible, v => {
  if (!v && route.query.country) router.replace({ path: '/country-profile' })
})

async function showInternal() {
  try {
    const { data } = await axios.get('/api/policy/internal')
    internalHTML.value = data.body_html || ''
    internalVisible.value = true
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '读取内部制度失败')
  }
}

onMounted(load)
</script>

<style scoped>
.cp { display: flex; flex-direction: column; gap: 16px; }
.page-head { display: flex; align-items: flex-start; justify-content: space-between; }
.page-title { font-size: 22px; font-weight: 700; color: var(--c-text-strong); margin: 0; }
.page-desc { font-size: 13px; color: var(--c-text-muted); margin: 4px 0 0; }

.stat-bar { display: flex; gap: 18px; align-items: center; font-size: 13px; color: var(--c-text-muted); }
.stat-bar b { color: var(--c-text-strong); font-size: 15px; }
.stat-bar .warn { display: inline-flex; align-items: center; gap: 4px; color: #a8730a; }

.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(330px, 1fr)); gap: 14px; }
.card {
  position: relative; overflow: hidden; cursor: pointer;
  background: var(--c-surface); border: 1px solid var(--c-border);
  border-radius: var(--radius); padding: 16px 18px;
  box-shadow: var(--shadow-card); transition: transform .18s, box-shadow .18s;
}
.card::before { content: ''; position: absolute; left: 0; top: 0; bottom: 0; width: 4px; background: var(--rk); }
.card:hover { transform: translateY(-3px); box-shadow: var(--shadow-pop); }

.card-h { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
.c-name { font-size: 16px; font-weight: 700; color: var(--c-text-strong); }
.c-alias { font-size: 11.5px; font-weight: 400; color: var(--c-text-muted); margin-left: 6px; }
.risk { display: inline-flex; align-items: center; gap: 5px; font-size: 12px; font-weight: 600; white-space: nowrap; }
.risk-dot { width: 9px; height: 9px; border-radius: 50%; background: var(--rk); }
.risk-high { color: #b32d2d; }
.risk-mid { color: #8a5d00; }
.risk-low { color: #0a7a0a; }
.risk-na { color: var(--c-text-muted); }

.kv { margin: 12px 0 0; }
.kv > div { display: flex; gap: 8px; margin-bottom: 6px; font-size: 12.5px; line-height: 1.6; }
.kv dt { flex-shrink: 0; width: 64px; color: var(--c-text-muted); }
.kv dd { margin: 0; color: var(--c-text); display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }

.card-f {
  display: flex; align-items: center; justify-content: space-between; gap: 10px;
  margin-top: 12px; padding-top: 10px; border-top: 1px solid var(--c-border);
  font-size: 11.5px; color: var(--c-text-muted);
}
.card-f span { display: inline-flex; align-items: center; gap: 4px; }
.upd.stale { color: #a8730a; font-weight: 500; }

.err-card { max-width: 620px; margin: 40px auto; }
.err { text-align: center; color: var(--c-text-muted); padding: 20px; }
.err-title { font-size: 15px; color: var(--c-text-strong); margin: 12px 0 6px; font-weight: 600; }
.err-dir { font-size: 12px; margin: 0 0 18px; word-break: break-all; }

/* 详情 */
.detail { padding: 0 4px 30px; }
.d-alert {
  display: flex; flex-direction: column; gap: 4px; padding: 12px 14px;
  border-radius: 8px; font-size: 13px; line-height: 1.7; margin-bottom: 14px;
  background: #fdf5f5; border: 1px solid #f0d5d5;
}
.d-alert.risk-mid { background: #fdf8ee; border-color: #f0e3c5; }
.d-alert.risk-low { background: #f2faf2; border-color: #cfe8cf; }
.d-alert strong { font-size: 13.5px; }
.d-stale {
  display: flex; align-items: center; gap: 6px; font-size: 12.5px;
  color: #8a5d00; background: #fdf8ee; border: 1px solid #f0e3c5;
  border-radius: 7px; padding: 8px 12px; margin: 0 0 14px;
}

.meta-table { width: 100%; border-collapse: collapse; font-size: 13px; margin-bottom: 20px; }
.meta-table th, .meta-table td { border: 1px solid var(--c-border); padding: 7px 11px; text-align: left; vertical-align: top; }
.meta-table th { background: #f6f9fe; color: var(--c-text-muted); font-weight: 500; width: 130px; white-space: nowrap; }
.meta-table td { color: var(--c-text); line-height: 1.7; }

/* Obsidian 正文渲染 */
.md { font-size: 13.5px; line-height: 1.8; color: var(--c-text); }
.md :deep(h1) { font-size: 19px; color: var(--c-text-strong); margin: 22px 0 10px; }
.md :deep(h2) {
  font-size: 15.5px; color: var(--c-text-strong); margin: 22px 0 10px;
  padding-left: 9px; border-left: 3px solid var(--c-primary);
}
.md :deep(table) { width: 100%; border-collapse: collapse; margin: 10px 0; font-size: 12.5px; }
.md :deep(th), .md :deep(td) { border: 1px solid var(--c-border); padding: 6px 10px; text-align: left; }
.md :deep(th) { background: #f6f9fe; font-weight: 600; }
.md :deep(ul) { padding-left: 20px; }
.md :deep(li) { margin: 3px 0; }
.md :deep(input[type=checkbox]) { margin-right: 6px; }
.md :deep(blockquote) {
  margin: 10px 0; padding: 8px 14px; color: var(--c-text-muted);
  background: #f8fafc; border-left: 3px solid var(--c-border-strong); border-radius: 0 6px 6px 0;
}
.md :deep(a) { color: var(--c-primary); word-break: break-all; }
.md :deep(code) { background: #f1f5f9; padding: 1px 5px; border-radius: 4px; font-size: 12px; }
</style>
