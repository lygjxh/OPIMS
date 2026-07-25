<template>
  <div class="blacklist">
    <div class="toolbar">
      <el-button type="primary" @click="openDialog()">{{ $t('common.create') }}</el-button>
      <el-select v-model="filterCountry" :placeholder="$t('project.country')" clearable style="width:150px">
        <el-option v-for="c in countries" :key="c" :label="c" :value="c" />
      </el-select>
      <el-select v-model="filterStatus" :placeholder="$t('project.projectStatus')" clearable style="width:130px">
        <el-option :label="$t('blacklist.listed')" value="列入中" />
        <el-option :label="$t('blacklist.delisted')" value="已拉出" />
      </el-select>
    </div>

    <el-table :data="filtered" style="width:100%" max-height="calc(100vh - 200px)" :row-class-name="rowClass">
      <el-table-column prop="sub_short_name" :label="$t('blacklist.subShortName')" width="140" />
      <el-table-column prop="sub_full_name" :label="$t('blacklist.subFullName')" min-width="180" />
      <el-table-column prop="country" :label="$t('project.country')" width="100" />
      <el-table-column prop="related_project" label="提报项目" width="140" />
      <el-table-column prop="restrict_level" label="限制等级" width="100">
        <template #default="{ row }">
          <el-tag v-if="row.restrict_level" :type="levelType(row.restrict_level)" size="small">{{ row.restrict_level }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="list_reason" label="列入原因" min-width="160" />
      <el-table-column prop="list_date" :label="$t('blacklist.listDate')" width="110" />
      <el-table-column prop="status" label="状态" width="90">
        <template #default="{ row }">
          <el-tag :type="row.status === '列入中' ? 'danger' : 'info'">{{ row.status }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="" width="260" fixed="right">
        <template #default="{ row }">
          <el-button size="small" text @click="openDialog(row)" :disabled="row.status === '已拉出'">{{ $t('common.edit') }}</el-button>
          <el-button v-if="row.status === '列入中'" size="small" text type="warning" @click="delist(row)">{{ $t('blacklist.delist') }}</el-button>
          <el-button size="small" text type="danger" @click="remove(row)" :disabled="row.status === '已拉出'">{{ $t('common.delete') }}</el-button>
          <el-button size="small" text type="info" @click="showLog(row)">日志</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="editing.id ? $t('common.edit') : $t('common.create')" width="600px">
      <el-form :model="form" label-width="110px">
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item :label="$t('blacklist.subShortName')" required>
              <el-input v-model="form.sub_short_name" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label="$t('blacklist.subFullName')" required>
              <el-input v-model="form.sub_full_name" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item :label="$t('project.country')"><el-input v-model="form.country" /></el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="提报项目"><el-input v-model="form.related_project" /></el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="列入原因"><el-input v-model="form.list_reason" type="textarea" /></el-form-item>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item :label="$t('blacklist.listDate')" required>
              <el-date-picker v-model="form.list_date" type="date" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="限制日期"><el-date-picker v-model="form.restrict_until" type="date" style="width:100%" /></el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="限制等级">
              <el-select v-model="form.restrict_level" style="width:100%" clearable>
                <el-option label="黑名单" value="黑名单" />
                <el-option label="限制使用" value="限制使用" />
                <el-option label="已退库" value="已退库" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="列入提报人"><el-input v-model="form.list_reporter" /></el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="save">{{ $t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <!-- 操作日志弹窗 -->
    <el-dialog v-model="logVisible" title="操作日志" width="500px">
      <template v-if="logs.length">
        <div v-for="log in logs" :key="log.id" class="log-item">
          <el-tag size="small" :type="log.action === '列入' ? 'danger' : log.action === '拉出' ? 'warning' : log.action === '删除' ? 'danger' : 'info'">
            {{ log.action }}
          </el-tag>
          <span class="log-detail">{{ log.detail || '—' }}</span>
          <span class="log-time">{{ log.created_at }}</span>
        </div>
      </template>
      <p v-else style="text-align:center;color:#999">暂无操作记录</p>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'

const list = ref<any[]>([])
const filterCountry = ref('')
const filterStatus = ref('')
const dialogVisible = ref(false)
const logVisible = ref(false)
const logs = ref<any[]>([])
const editing = ref<any>({})
const form = ref<any>({})

const countries = computed(() => [...new Set(list.value.map((x: any) => x.country).filter(Boolean))] as string[])

const filtered = computed(() => list.value.filter((x: any) => {
  if (filterStatus.value && x.status !== filterStatus.value) return false
  if (filterCountry.value && x.country !== filterCountry.value) return false
  return true
}))

async function load() {
  const { data } = await axios.get('/api/blacklist/subcontractor')
  list.value = data || []
}

function openDialog(row?: any) {
  editing.value = row ? { ...row } : {}
  form.value = row ? { ...row, list_date: row.list_date || '', restrict_until: row.restrict_until || '' }
    : { sub_short_name: '', sub_full_name: '', country: '', related_project: '',
        list_reason: '', list_date: '', restrict_level: '', restrict_until: '', list_reporter: '' }
  dialogVisible.value = true
}

async function save() {
  const payload = { ...form.value, list_date: fmtDate(form.value.list_date), restrict_until: fmtDate(form.value.restrict_until) }
  if (editing.value.id) {
    await axios.put('/api/blacklist/subcontractor/' + editing.value.id, payload)
  } else {
    await axios.post('/api/blacklist/subcontractor', payload)
  }
  ElMessage.success('Saved')
  dialogVisible.value = false
  load()
}

async function delist(row: any) {
  const reason = prompt('拉出原因:')
  if (!reason) return
  await axios.put('/api/blacklist/subcontractor/' + row.id, {
    ...row, delist_reason: reason, delist_date: new Date().toISOString().slice(0,10),
    delist_reporter: '', status: '已拉出'
  })
  ElMessage.success('Delisted')
  load()
}

async function remove(row: any) {
  await ElMessageBox.confirm('确认删除？')
  await axios.delete('/api/blacklist/subcontractor/' + row.id)
  ElMessage.success('Deleted')
  load()
}

async function showLog(row: any) {
  try {
    const { data } = await axios.get('/api/blacklist/subcontractor/' + row.id + '/audit')
    logs.value = data || []
  } catch { logs.value = [] }
  logVisible.value = true
}

function fmtDate(d: any) {
  if (!d) return ''
  if (typeof d === 'string') return d.slice(0, 10)
  return new Date(d).toISOString().slice(0, 10)
}

function rowClass({ row }: any) {
  if (row.status === '已拉出') return 'row-delisted'
  if (row.restrict_level === '黑名单') return 'row-blacklist'
  if (row.restrict_level === '限制使用') return 'row-restricted'
  return ''
}

function levelType(level: string) {
  if (level === '黑名单') return 'danger'
  if (level === '限制使用') return 'warning'
  return 'info'
}

onMounted(load)
</script>

<style scoped>
.blacklist { display: flex; flex-direction: column; gap: 12px; }
.toolbar { display: flex; gap: 8px; flex-wrap: wrap; }
:deep(.row-blacklist) { background-color: #ffebee; }
:deep(.row-restricted) { background-color: #fff3e0; }
:deep(.row-delisted) { background-color: #f5f5f5; }
.log-item { display: flex; align-items: center; gap: 10px; padding: 8px 0; border-bottom: 1px solid #f0f0f0; font-size: 13px; }
.log-detail { flex: 1; color: #606266; }
.log-time { color: #909399; font-size: 12px; white-space: nowrap; }
</style>
