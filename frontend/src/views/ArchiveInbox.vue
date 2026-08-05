<template>
  <div class="ai">
    <!-- 标题由外层「月度报送」容器提供，这里只留本标签页的操作 -->
    <div class="page-head">
      <div class="sec-desc">把收到的文件按规范改名并归入对应项目目录，省去手工重命名与搬运</div>
      <div class="head-ops">
        <el-input v-model="dir" placeholder="待归档目录" style="width:340px" clearable />
        <el-button @click="load" :loading="loading"><el-icon><Refresh /></el-icon>扫描</el-button>
        <el-button @click="openLog"><el-icon><Tickets /></el-icon>操作记录</el-button>
      </div>
    </div>

    <el-alert v-if="errMsg" type="warning" show-icon :closable="false" :title="errMsg">
      <template #default>
        <span v-if="expectedDir">默认目录：{{ expectedDir }}——可在上方输入框改为其它目录后重新扫描</span>
      </template>
    </el-alert>

    <el-card v-else>
      <template #header>
        <div class="card-head">
          <span>待归档文件 · {{ files.length }} 个</span>
          <span class="hint">{{ dirShown }}</span>
        </div>
      </template>

      <el-empty v-if="!files.length" description="该目录下没有待归档的文件" />

      <table v-else class="tbl">
        <thead>
          <tr><th>文件名</th><th>大小</th><th>修改时间</th><th>项目</th><th>类型</th><th>周期</th><th>操作</th></tr>
        </thead>
        <tbody>
          <tr v-for="f in files" :key="f.path">
            <td class="c-name" :title="f.name">{{ f.name }}</td>
            <td class="c-num">{{ fmtSize(f.size) }}</td>
            <td class="c-num">{{ f.mod_time }}</td>
            <td>
              <el-select v-model="f.project" placeholder="选择项目" size="small" style="width:150px" filterable>
                <el-option v-for="p in projects" :key="p.ShortName" :label="p.ShortName" :value="p.ShortName" />
              </el-select>
            </td>
            <td>
              <el-select v-model="f.code" placeholder="类型" size="small" style="width:170px">
                <el-option v-for="c in checklist" :key="c.code"
                  :label="c.code + ' ' + c.name" :value="c.code" />
              </el-select>
            </td>
            <td>
              <el-input v-model="f.period" placeholder="202607" size="small" style="width:96px" />
            </td>
            <td>
              <el-button size="small" type="primary" plain
                :disabled="!f.project || !f.code || !f.period" @click="preview(f)">
                归档
              </el-button>
            </td>
          </tr>
        </tbody>
      </table>

      <p class="note">
        <el-icon><InfoFilled /></el-icon>
        系统会先复制到目标位置并校验，确认无误后才删除原件；每次操作留有记录，可回滚。
        目标已存在同名文件时不会覆盖，而是提示改用新版本号。
      </p>
    </el-card>

    <!-- 归档确认 -->
    <el-dialog v-model="planVisible" title="确认归档" width="560px">
      <template v-if="plan">
        <el-alert v-if="plan.error" type="error" :title="plan.error" :closable="false" show-icon />
        <template v-else>
          <table class="plan">
            <tbody>
              <tr><th>源文件</th><td>{{ plan.source_name }}</td></tr>
              <tr><th>归档为</th><td><b class="tgt">{{ plan.target_name }}</b></td></tr>
              <tr><th>目标目录</th><td class="path">{{ plan.target_dir }}</td></tr>
              <tr><th>文件类型</th><td>{{ plan.code }} · {{ plan.code_name }}</td></tr>
            </tbody>
          </table>

          <el-alert v-if="plan.conflict" type="warning" show-icon :closable="false"
            title="目标位置已存在同名文件" style="margin-top:12px">
            <p style="margin:4px 0">为避免覆盖已有资料，建议改用新版本号：<b>{{ plan.suggested }}</b></p>
            <el-radio-group v-model="conflictAction" size="small">
              <el-radio-button label="version">用新版本号（推荐）</el-radio-button>
              <el-radio-button label="overwrite">覆盖已有文件</el-radio-button>
            </el-radio-group>
          </el-alert>

          <el-form label-width="70px" style="margin-top:12px">
            <el-form-item label="操作人">
              <el-input v-model="operator" placeholder="姓名" style="width:160px" size="small" />
            </el-form-item>
          </el-form>
        </template>
      </template>
      <template #footer>
        <el-button @click="planVisible = false">取消</el-button>
        <el-button type="primary" :disabled="!!plan?.error" @click="apply">确认归档</el-button>
      </template>
    </el-dialog>

    <!-- 操作记录 -->
    <el-dialog v-model="logVisible" title="归档操作记录" width="760px">
      <el-empty v-if="!logs.length" description="暂无归档记录" />
      <table v-else class="tbl">
        <thead><tr><th>时间</th><th>操作人</th><th>项目</th><th>归档为</th><th>状态</th><th>操作</th></tr></thead>
        <tbody>
          <tr v-for="l in logs" :key="l.id">
            <td class="c-num">{{ l.created_at }}</td>
            <td>{{ l.operator || '-' }}</td>
            <td>{{ l.project }}</td>
            <td class="c-name" :title="l.target_path">{{ baseName(l.target_path) }}</td>
            <td>
              <span v-if="l.undone" class="tag-undone">已回滚</span>
              <span v-else class="tag-ok">已归档</span>
            </td>
            <td>
              <el-button v-if="l.can_undo" size="small" text type="danger" @click="undo(l)">回滚</el-button>
              <span v-else-if="!l.undone" class="muted" title="目标文件已移动或源位置被占用">不可回滚</span>
            </td>
          </tr>
        </tbody>
      </table>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh, Tickets, InfoFilled } from '@element-plus/icons-vue'

const dir = ref('')
const dirShown = ref('')
const files = ref<any[]>([])
const projects = ref<any[]>([])
const checklist = ref<any[]>([])
const loading = ref(false)
const errMsg = ref('')
const expectedDir = ref('')

const planVisible = ref(false)
const plan = ref<any>(null)
const current = ref<any>(null)
const conflictAction = ref<'version' | 'overwrite'>('version')
const operator = ref('')

const logVisible = ref(false)
const logs = ref<any[]>([])

const baseName = (p: string) => (p || '').split(/[\\/]/).pop()
function fmtSize(n: number) {
  if (n < 1024) return n + ' B'
  if (n < 1024 * 1024) return (n / 1024).toFixed(1) + ' KB'
  return (n / 1024 / 1024).toFixed(1) + ' MB'
}

async function load() {
  loading.value = true
  errMsg.value = ''
  try {
    const { data } = await axios.get('/api/archive/inbox', { params: dir.value ? { dir: dir.value } : {} })
    files.value = data.files || []
    projects.value = data.projects || []
    checklist.value = data.checklist || []
    dirShown.value = data.dir
    if (!dir.value) dir.value = data.dir
  } catch (e: any) {
    const d = e.response?.data
    errMsg.value = d?.error || '扫描失败'
    expectedDir.value = d?.expected_dir || ''
    files.value = []
  }
  loading.value = false
}

async function preview(f: any) {
  current.value = f
  conflictAction.value = 'version'
  try {
    const { data } = await axios.post('/api/archive/plan', {
      source_path: f.path, project: f.project, code: f.code, period: f.period, version: 1,
    })
    plan.value = data
    planVisible.value = true
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '预演失败')
  }
}

async function apply() {
  const f = current.value
  // 冲突时按用户选择：改版本号（默认）或显式覆盖
  let version = 1, overwrite = false
  if (plan.value?.conflict) {
    if (conflictAction.value === 'overwrite') overwrite = true
    else {
      const m = /-v(\d+)\./.exec(plan.value.suggested || '')
      version = m ? Number(m[1]) : 2
    }
  }
  try {
    const { data } = await axios.post('/api/archive/apply', {
      source_path: f.path, project: f.project, code: f.code, period: f.period,
      version, overwrite, operator: operator.value,
    })
    planVisible.value = false
    ElMessage.success(data.error ? '已归档，但' + data.error : '已归档为 ' + baseName(data.target_path))
    load()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '归档失败')
  }
}

async function openLog() {
  try {
    const { data } = await axios.get('/api/archive/log')
    logs.value = data.logs || []
    logVisible.value = true
  } catch { ElMessage.error('读取操作记录失败') }
}

async function undo(l: any) {
  try {
    await ElMessageBox.confirm(
      `将把「${baseName(l.target_path)}」移回原位置：\n${l.source_path}`, '确认回滚',
      { type: 'warning' })
  } catch { return }
  try {
    await axios.post('/api/archive/undo', null, { params: { id: l.id } })
    ElMessage.success('已回滚')
    openLog(); load()
  } catch (e: any) {
    ElMessage.error(e.response?.data?.error || '回滚失败')
  }
}

onMounted(load)
</script>

<style scoped>
.ai { display: flex; flex-direction: column; gap: 16px; }
.page-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap; }
.sec-desc { font-size: 13px; color: var(--c-text-muted); }
.head-ops { display: flex; gap: 8px; align-items: center; }

.card-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap; }
.hint { font-size: 11.5px; color: var(--c-text-muted); font-weight: 400; word-break: break-all; }

.tbl { width: 100%; border-collapse: collapse; font-size: 12.5px; }
.tbl th, .tbl td { border: 1px solid var(--c-border); padding: 6px 9px; text-align: left; vertical-align: middle; }
.tbl thead th { background: #f1f5fb; color: var(--c-text-strong); font-weight: 600; white-space: nowrap; }
.c-name { max-width: 260px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.c-num { white-space: nowrap; color: var(--c-text-muted); }
.muted { color: var(--c-text-muted); font-size: 11.5px; }

.plan { width: 100%; border-collapse: collapse; font-size: 13px; }
.plan th, .plan td { border: 1px solid var(--c-border); padding: 7px 10px; text-align: left; }
.plan th { background: #f6f9fe; color: var(--c-text-muted); font-weight: 500; width: 88px; white-space: nowrap; }
.plan .tgt { color: var(--c-primary); }
.plan .path { font-size: 11.5px; color: var(--c-text-muted); word-break: break-all; }

.tag-ok { color: #0a7a0a; font-weight: 500; }
.tag-undone { color: var(--c-text-muted); }

.note {
  display: flex; align-items: center; gap: 6px; margin: 12px 0 0;
  font-size: 11.5px; color: var(--c-text-muted);
}
</style>
