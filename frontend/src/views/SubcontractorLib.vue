<template>
  <div class="sub-lib-page">
    <div class="toolbar">
      <el-button type="primary" @click="openDialog()">{{ $t('common.create') }}</el-button>
      <el-button @click="triggerImport">导入</el-button>
      <el-dropdown @command="doExport">
        <el-button>导出<el-icon class="el-icon--right"><arrow-down /></el-icon></el-button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="all">导出全部</el-dropdown-item>
            <el-dropdown-item command="filtered">导出当前筛选结果</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
      <el-button @click="downloadTemplate">模板下载</el-button>
      <input ref="fileInput" type="file" accept=".xlsx" style="display:none" @change="onFileChosen" />

      <span class="spacer" />

      <el-input v-model="country" clearable placeholder="国别" style="width:100px" @input="load" />
      <el-select v-model="category" clearable placeholder="分类" style="width:100px" @change="onCategoryFilter">
        <el-option v-for="c in categories" :key="c" :label="c" :value="c" />
      </el-select>
      <el-select v-model="profession" clearable placeholder="专业" style="width:110px" @change="load">
        <el-option v-for="p in filterProfessions" :key="p" :label="p" :value="p" />
      </el-select>
      <el-input v-model="grade" clearable placeholder="分包商等级" style="width:110px" @input="load" />
      <el-input v-model="classification" clearable placeholder="分包商分级" style="width:120px" @input="load" />
      <el-input v-model="keyword" clearable :placeholder="'编号/简称/名称'" style="width:180px" @input="load" />
    </div>

    <el-alert
      v-if="unmatched.length"
      type="warning"
      show-icon
      :closable="false"
      class="unmatched-alert"
      :title="`项目分包台账中有 ${unmatched.length} 个分包商名称在库中无匹配，合同数/总金额无法汇总`"
    >
      <div class="unmatched-list">{{ unmatched.join('、') }}</div>
      <div class="unmatched-tip">请将台账中的分包商名称改为库内「分包商名称」，或在库中补录对应分包商。</div>
    </el-alert>

    <el-table :data="list" stripe max-height="calc(100vh - 200px)" @row-click="showDetail">
      <el-table-column prop="sub_no" label="分包商编号" width="130" />
      <el-table-column prop="short_name" label="简称" width="110" show-overflow-tooltip />
      <el-table-column prop="full_name" label="分包商名称" min-width="180" show-overflow-tooltip>
        <template #default="{ row }">
          {{ row.full_name }}
          <el-tooltip v-if="row.assoc_unit" :content="'关联单位：' + row.assoc_unit" placement="top">
            <el-icon class="assoc-badge"><Link /></el-icon>
          </el-tooltip>
        </template>
      </el-table-column>
      <el-table-column prop="country" label="国别" width="80" />
      <el-table-column prop="established_date" label="成立日期" width="100" />
      <el-table-column prop="reg_capital" label="注册资金" width="100" align="right" />
      <el-table-column prop="legal_rep" label="法人" width="120" show-overflow-tooltip />
      <el-table-column prop="agent" label="委托代理人" width="120" show-overflow-tooltip />
      <el-table-column prop="grade" label="分包商等级" width="100" />
      <el-table-column prop="classification" label="分包商分级" width="110" />
      <el-table-column prop="category" label="分类" width="80" />
      <el-table-column prop="profession" label="专业" width="100" />
      <el-table-column label="合同数" width="80" align="center">
        <template #default="{ row }">{{ row.contract_count || 0 }}</template>
      </el-table-column>
      <el-table-column label="总金额(万元)" width="130" align="right">
        <template #default="{ row }">{{ row.total_amount ? row.total_amount.toLocaleString() : '-' }}</template>
      </el-table-column>
      <el-table-column label="" width="60" fixed="right">
        <template #default="{ row }">
          <el-button size="small" text @click.stop="openDialog(row)">{{ $t('common.edit') }}</el-button>
        </template>
      </el-table-column>
    </el-table>

    <!-- 详情 -->
    <el-dialog v-model="detailVisible" title="分包商详情" width="760px" top="3vh">
      <template v-if="detail">
        <!-- 黑名单状态 -->
        <div v-if="blStatus" class="bl-section">
          <el-tag v-if="blStatus === '列入中'" type="danger" effect="dark" size="large">⚠ 黑名单（列入中）</el-tag>
          <el-tag v-else type="info" effect="plain" size="large">黑名单（{{ blStatus }}）</el-tag>
          <span class="bl-meta">
            {{ blLevel ? '限制等级:' + blLevel : '' }} {{ blDate ? '列入:' + blDate : '' }} {{ blReason ? '原因:' + blReason : '' }}
          </span>
        </div>
        <el-alert
          v-if="assocBlacklisted"
          type="warning"
          show-icon
          :closable="false"
          class="bl-section"
          :title="'关联单位「' + detail.assoc_unit + '」已列入黑名单'"
        />

        <el-descriptions :column="2" border size="small" title="基本信息">
          <el-descriptions-item label="编号">{{ detail.sub_no }}</el-descriptions-item>
          <el-descriptions-item label="简称">{{ detail.short_name || '-' }}</el-descriptions-item>
          <el-descriptions-item label="名称" :span="2">{{ detail.full_name }}</el-descriptions-item>
          <el-descriptions-item label="国别">{{ detail.country || '-' }}</el-descriptions-item>
          <el-descriptions-item label="企业性质">{{ detail.enterprise_type || '-' }}</el-descriptions-item>
          <el-descriptions-item label="成立日期">{{ detail.established_date || '-' }}</el-descriptions-item>
          <el-descriptions-item label="注册资金">{{ detail.reg_capital || '-' }}</el-descriptions-item>
          <el-descriptions-item label="所在地区">{{ detail.region || '-' }}</el-descriptions-item>
          <el-descriptions-item label="详细地址">{{ detail.address || '-' }}</el-descriptions-item>
          <el-descriptions-item label="经营范围" :span="2">{{ detail.business_scope || '-' }}</el-descriptions-item>
        </el-descriptions>

        <el-descriptions :column="2" border size="small" title="人员" class="group">
          <el-descriptions-item label="法人及身份证">{{ detail.legal_rep || '-' }}</el-descriptions-item>
          <el-descriptions-item label="法人联系方式">{{ detail.legal_rep_phone || '-' }}</el-descriptions-item>
          <el-descriptions-item label="委托代理人及身份证">{{ detail.agent || '-' }}</el-descriptions-item>
          <el-descriptions-item label="委托代理人联系方式">{{ detail.agent_phone || '-' }}</el-descriptions-item>
        </el-descriptions>

        <el-descriptions :column="2" border size="small" title="资质与评级" class="group">
          <el-descriptions-item label="资质类别及等级">{{ detail.qualification || '-' }}</el-descriptions-item>
          <el-descriptions-item label="资信等级">{{ detail.credit_rating || '-' }}</el-descriptions-item>
          <el-descriptions-item label="分包商等级">{{ detail.grade || '-' }}</el-descriptions-item>
          <el-descriptions-item label="分包商分级">{{ detail.classification || '-' }}</el-descriptions-item>
          <el-descriptions-item label="分类">{{ detail.category || '-' }}</el-descriptions-item>
          <el-descriptions-item label="专业">{{ detail.profession || '-' }}</el-descriptions-item>
        </el-descriptions>

        <el-descriptions :column="2" border size="small" title="上报信息" class="group">
          <el-descriptions-item label="上报项目">{{ detail.report_project || '-' }}</el-descriptions-item>
          <el-descriptions-item label="推荐人">{{ detail.recommender || '-' }}</el-descriptions-item>
          <el-descriptions-item label="上报单位">{{ detail.report_unit || '-' }}</el-descriptions-item>
          <el-descriptions-item label="单位负责人">{{ detail.unit_head || '-' }}</el-descriptions-item>
          <el-descriptions-item label="关联单位">{{ detail.assoc_unit || '-' }}</el-descriptions-item>
          <el-descriptions-item label="备注">{{ detail.notes || '-' }}</el-descriptions-item>
        </el-descriptions>

        <div class="group">
          <div class="group-title">资质文件</div>
          <div class="quals">
            <div v-for="t in qualTypes" :key="t" class="qual-item">
              <span class="qual-label">{{ t }}</span>
              <el-button v-if="quals[t]" size="small" text type="primary" @click="openQual(t)">
                <el-icon><document /></el-icon> 打开
              </el-button>
              <span v-else class="qual-missing">未上传</span>
            </div>
          </div>
          <div class="qual-tip">文件命名规则：<code>{{ detail.sub_no }}-资质类型.*</code>，存放于 <code>07.Sub-contractor\01.Qualification Documents</code></div>
        </div>

        <div class="group summary">
          合同数 <b>{{ detail.contract_count || 0 }}</b> · 总金额 <b>{{ (detail.total_amount || 0).toLocaleString() }}</b> 万元（来自项目分包汇总）
        </div>
      </template>
    </el-dialog>

    <!-- 新建/编辑 -->
    <el-dialog v-model="editVisible" :title="editing.id ? $t('common.edit') : $t('common.create')" width="760px" top="3vh">
      <el-form :model="form" label-width="118px" size="small">
        <el-divider content-position="left">基本信息</el-divider>
        <el-row :gutter="12">
          <el-col :span="8"><el-form-item label="分包商编号" required><el-input v-model="form.sub_no" placeholder="YYYYMM-序号" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="分包商简称"><el-input v-model="form.short_name" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="国别"><el-input v-model="form.country" /></el-form-item></el-col>
        </el-row>
        <el-form-item label="分包商名称" required><el-input v-model="form.full_name" /></el-form-item>
        <el-row :gutter="12">
          <el-col :span="8"><el-form-item label="企业性质"><el-input v-model="form.enterprise_type" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="成立日期"><el-input v-model="form.established_date" placeholder="YYYY-MM-DD" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="注册资金"><el-input v-model="form.reg_capital" /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="所在地区"><el-input v-model="form.region" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="详细地址"><el-input v-model="form.address" /></el-form-item></el-col>
        </el-row>
        <el-form-item label="经营范围"><el-input v-model="form.business_scope" type="textarea" :rows="2" /></el-form-item>

        <el-divider content-position="left">人员</el-divider>
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="法人及身份证"><el-input v-model="form.legal_rep" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="法人联系方式"><el-input v-model="form.legal_rep_phone" /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="委托代理人及身份证"><el-input v-model="form.agent" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="代理人联系方式"><el-input v-model="form.agent_phone" /></el-form-item></el-col>
        </el-row>

        <el-divider content-position="left">资质与评级</el-divider>
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="资质类别及等级"><el-input v-model="form.qualification" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="资信等级"><el-input v-model="form.credit_rating" /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="分包商等级"><el-input v-model="form.grade" placeholder="如 正常使用 / 优先使用" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="分包商分级"><el-input v-model="form.classification" placeholder="如 核心层分包商" /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="分类"><el-select v-model="form.category" clearable style="width:100%" @change="form.profession = ''"><el-option v-for="c in categories" :key="c" :label="c" :value="c" /></el-select></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="专业"><el-select v-model="form.profession" clearable style="width:100%"><el-option v-for="p in formProfessions" :key="p" :label="p" :value="p" /></el-select></el-form-item></el-col>
        </el-row>

        <el-divider content-position="left">上报信息</el-divider>
        <el-row :gutter="12">
          <el-col :span="8"><el-form-item label="上报项目"><el-input v-model="form.report_project" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="推荐人"><el-input v-model="form.recommender" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="上报单位"><el-input v-model="form.report_unit" /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="单位负责人"><el-input v-model="form.unit_head" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="关联单位"><el-input v-model="form.assoc_unit" /></el-form-item></el-col>
        </el-row>
        <el-form-item label="备注"><el-input v-model="form.notes" type="textarea" :rows="2" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button v-if="editing.id" type="danger" @click="doDelete">{{ $t('common.delete') }}</el-button>
        <el-button type="primary" @click="save">{{ $t('common.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import axios from 'axios'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowDown, Link, Document } from '@element-plus/icons-vue'
import { PROFESSION_CATEGORIES } from '../data/professions'

const list = ref<any[]>([])
const unmatched = ref<string[]>([])
const country = ref('')
const category = ref('')
const profession = ref('')
const grade = ref('')
const classification = ref('')
const keyword = ref('')

const detailVisible = ref(false)
const detail = ref<any>(null)
const blStatus = ref('')
const blLevel = ref('')
const blDate = ref('')
const blReason = ref('')
const assocBlacklisted = ref(false)
const quals = ref<Record<string, string>>({})
const qualTypes = ['营业执照', '组织机构代码证', '税务登记证', '安全生产许可证', '资质证书']

const editVisible = ref(false)
const editing = ref<any>({})
const form = ref<any>({})
const fileInput = ref<HTMLInputElement | null>(null)

const categories = computed(() => PROFESSION_CATEGORIES.map(c => c.name))
const allProfessions = computed(() => [...new Set(PROFESSION_CATEGORIES.flatMap(c => c.professions))])
const filterProfessions = computed(() => professionsOf(category.value))
const formProfessions = computed(() => professionsOf(form.value.category))

function professionsOf(cat: string): string[] {
  if (!cat) return allProfessions.value
  const c = PROFESSION_CATEGORIES.find(x => x.name === cat)
  return c ? c.professions : allProfessions.value
}

function currentParams() {
  const p: any = {}
  if (country.value) p.country = country.value
  if (category.value) p.category = category.value
  if (profession.value) p.profession = profession.value
  if (grade.value) p.grade = grade.value
  if (classification.value) p.classification = classification.value
  if (keyword.value) p.keyword = keyword.value
  return p
}

function onCategoryFilter() {
  profession.value = ''
  load()
}

async function load() {
  try {
    const { data } = await axios.get('/api/subcontractors', { params: currentParams() })
    list.value = data.records || []
    unmatched.value = data.unmatched || []
  } catch { ElMessage.error('加载失败') }
}

async function showDetail(row: any) {
  try {
    const { data } = await axios.get('/api/subcontractors/' + row.id)
    detail.value = data.base
    blStatus.value = data.blacklist_status || ''
    blLevel.value = data.restrict_level || ''
    blDate.value = data.list_date || ''
    blReason.value = data.list_reason || ''
    assocBlacklisted.value = !!data.assoc_blacklisted
    quals.value = data.quals || {}
    detailVisible.value = true
  } catch { ElMessage.error('加载详情失败') }
}

function openQual(type: string) {
  if (!detail.value?.sub_no) return
  axios.get('/api/subcontractors/qual/open', { params: { no: detail.value.sub_no, type } })
    .catch(() => ElMessage.error('打开失败'))
}

function openDialog(row?: any) {
  editing.value = row ? { ...row } : {}
  form.value = row ? { ...row } : {
    sub_no: '', short_name: '', full_name: '', country: '', enterprise_type: '',
    established_date: '', reg_capital: '', region: '', address: '', business_scope: '',
    legal_rep: '', legal_rep_phone: '', agent: '', agent_phone: '',
    qualification: '', credit_rating: '', grade: '', classification: '', category: '', profession: '',
    report_project: '', recommender: '', report_unit: '', unit_head: '', assoc_unit: '', notes: '',
  }
  editVisible.value = true
}

async function save() {
  if (!form.value.sub_no?.trim()) { ElMessage.warning('分包商编号不能为空'); return }
  if (!form.value.full_name?.trim()) { ElMessage.warning('分包商名称不能为空'); return }
  try {
    if (editing.value.id) {
      await axios.put('/api/subcontractors/' + editing.value.id, form.value)
    } else {
      await axios.post('/api/subcontractors', form.value)
    }
    ElMessage.success('已保存')
    editVisible.value = false
    load()
  } catch (err: any) {
    ElMessage.error(err?.response?.data || '保存失败')
  }
}

async function doDelete() {
  await ElMessageBox.confirm('确认删除该分包商？', '提示', { type: 'warning' })
  await axios.delete('/api/subcontractors/' + editing.value.id)
  ElMessage.success('已删除')
  editVisible.value = false
  load()
}

// ---- 导入 / 导出 / 模板 ----
function triggerImport() {
  fileInput.value?.click()
}

async function onFileChosen(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  const fd = new FormData()
  fd.append('file', file)
  try {
    const { data } = await axios.post('/api/subcontractors/import', fd)
    const msg = `导入完成：新增 ${data.added} 条 / 更新 ${data.updated} 条 / 跳过 ${data.skipped} 条`
    if (data.skipped) {
      const detail = (data.errors || []).map((x: any) => `${x.row}：${x.msg}`).join('\n')
      ElMessageBox.alert(detail || '无详情', msg, { type: 'warning' })
    } else {
      ElMessage.success(msg)
    }
    load()
  } catch (err: any) {
    ElMessage.error(err?.response?.data || '导入失败')
  } finally {
    input.value = ''
  }
}

function doExport(scope: string) {
  const p = new URLSearchParams(scope === 'all' ? {} : currentParams())
  p.set('scope', scope)
  window.open('/api/subcontractors/export?' + p.toString())
}

function downloadTemplate() {
  window.open('/api/subcontractors/export/template')
}

onMounted(load)
</script>

<style scoped>
.sub-lib-page { display: flex; flex-direction: column; gap: 10px; }
.toolbar { display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }
.spacer { flex: 1 1 auto; }
.unmatched-alert :deep(.el-alert__content) { width: 100%; }
.unmatched-list { font-size: 12px; line-height: 1.6; margin-top: 4px; }
.unmatched-tip { font-size: 12px; color: #b88230; margin-top: 4px; }
.assoc-badge { color: #e6a23c; vertical-align: -2px; margin-left: 2px; }
.group { margin-top: 14px; }
.group-title { font-weight: 600; font-size: 14px; margin-bottom: 8px; }
.bl-section { margin-bottom: 12px; }
.bl-meta { margin-left: 8px; color: #606266; font-size: 13px; }
.quals { display: flex; flex-wrap: wrap; gap: 14px; }
.qual-item { display: flex; align-items: center; gap: 4px; min-width: 150px; }
.qual-label { font-size: 13px; color: #303133; }
.qual-missing { font-size: 12px; color: #c0c4cc; }
.qual-tip { font-size: 12px; color: #909399; margin-top: 8px; }
.qual-tip code { background: #f5f5f5; padding: 1px 4px; border-radius: 3px; }
.summary { font-size: 13px; color: #606266; }
.summary b { color: #303133; }
</style>
