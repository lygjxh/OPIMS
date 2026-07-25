<template>
  <div class="sub-lib-page">
    <div class="toolbar">
      <el-button type="primary" @click="openDialog()">{{ $t('common.create') }}</el-button>
      <el-select v-model="filterType" clearable placeholder="注册类型" style="width:120px">
        <el-option v-for="t in regTypes" :key="t" :label="t" :value="t" />
      </el-select>
      <el-select v-model="filterCategory" clearable placeholder="专业分类" style="width:110px">
        <el-option v-for="c in categories" :key="c" :label="c" :value="c" />
      </el-select>
      <el-input v-model="keyword" :placeholder="$t('common.search')" clearable style="width:200px" @input="load" />
    </div>

    <el-table :data="list" stripe max-height="calc(100vh - 160px)" @row-click="showDetail">
      <el-table-column prop="short_name" label="简称" width="120" />
      <el-table-column prop="full_name" label="分包商名称" min-width="180" show-overflow-tooltip />
      <el-table-column prop="registration_type" label="注册类型" width="100" />
      <el-table-column prop="country" label="国别" width="80" />
      <el-table-column prop="profession_category" label="专业分类" width="100" />
      <el-table-column prop="profession" label="主专业" width="100" />
      <el-table-column label="项目数" width="80" align="center">
        <template #default="{ row }">{{ row.project_count || 0 }}</template>
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

    <el-dialog v-model="detailVisible" title="分包商详情" width="700px" top="3vh">
      <template v-if="detail">
        <el-descriptions :column="2" border size="small" title="基本信息">
          <el-descriptions-item label="简称">{{ detail.short_name }}</el-descriptions-item>
          <el-descriptions-item label="名称" :span="2">{{ detail.full_name }}</el-descriptions-item>
          <el-descriptions-item label="注册类型">{{ detail.registration_type }}</el-descriptions-item>
          <el-descriptions-item label="国别">{{ detail.country || '-' }}</el-descriptions-item>
          <el-descriptions-item label="专业分类">{{ detail.profession_category }}</el-descriptions-item>
          <el-descriptions-item label="主专业">{{ detail.profession }}</el-descriptions-item>
          <el-descriptions-item v-if="detail.other_professions" label="其他专业" :span="2">{{ detail.other_professions }}</el-descriptions-item>
          <el-descriptions-item v-if="detail.parent_short_name" label="母公司">{{ detail.parent_short_name }}</el-descriptions-item>
        </el-descriptions>
        <el-descriptions :column="2" border size="small" title="法定代表人" style="margin-top:12px">
          <el-descriptions-item label="姓名">{{ detail.legal_rep_name }}</el-descriptions-item>
          <el-descriptions-item label="电话">{{ detail.legal_rep_phone }}</el-descriptions-item>
          <el-descriptions-item v-if="detail.legal_rep_id" label="身份证号">{{ detail.legal_rep_id }}</el-descriptions-item>
        </el-descriptions>
        <el-descriptions :column="2" border size="small" title="联系人" style="margin-top:12px" v-if="detail.contact_name">
          <el-descriptions-item label="姓名">{{ detail.contact_name || '-' }}</el-descriptions-item>
          <el-descriptions-item label="职务">{{ detail.contact_title || '-' }}</el-descriptions-item>
          <el-descriptions-item label="电话">{{ detail.contact_phone || '-' }}</el-descriptions-item>
          <el-descriptions-item label="邮箱">{{ detail.contact_email || '-' }}</el-descriptions-item>
        </el-descriptions>
        <el-descriptions :column="2" border size="small" title="注册信息" style="margin-top:12px" v-if="detail.biz_license || detail.tax_id">
          <el-descriptions-item v-if="detail.biz_license" label="营业执照">{{ detail.biz_license }}</el-descriptions-item>
          <el-descriptions-item v-if="detail.tax_id" label="税号">{{ detail.tax_id }}</el-descriptions-item>
          <el-descriptions-item v-if="detail.reg_address" label="注册地址" :span="2">{{ detail.reg_address }}</el-descriptions-item>
          <el-descriptions-item v-if="detail.reg_capital" label="注册资本">{{ detail.reg_capital }}</el-descriptions-item>
        </el-descriptions>

        <!-- 黑名单状态 -->
        <div v-if="blStatus" class="bl-section" style="margin-top:12px">
          <el-tag v-if="blStatus === '列入中'" type="danger" effect="dark" size="large">
            ⚠ 黑名单（列入中）
          </el-tag>
          <el-tag v-else type="info" effect="plain" size="large">
            黑名单（{{ blStatus }}）
          </el-tag>
          <span style="margin-left:8px;color:#606266;font-size:13px">
            {{ blLevel ? '限制等级:'+blLevel : '' }} {{ blDate ? '列入:'+blDate : '' }} {{ blReason ? '原因:'+blReason : '' }}
          </span>
        </div>

        <!-- 关联当地公司黑名单提示 -->
        <div v-if="localBL.length" style="margin-top:12px">
          <el-alert :title="'关联当地公司已列入黑名单：' + localBL.join(', ')" type="warning" show-icon :closable="false" />
        </div>

        <!-- 合作历史 -->
        <div style="margin-top:16px">
          <div style="font-weight:600;font-size:14px;margin-bottom:8px">
            合作历史（{{ detailProjects.length }} 个项目）
          </div>
          <el-table :data="detailProjects" size="small" max-height="300" stripe v-if="detailProjects.length">
            <el-table-column prop="project_short_name" label="项目简称" width="120" />
            <el-table-column prop="project_name" label="项目名称" min-width="160" show-overflow-tooltip />
            <el-table-column prop="start_date" label="开工日期" width="90" />
            <el-table-column prop="end_date" label="完工日期" width="90" />
            <el-table-column prop="contract_no" label="合同号" width="140" show-overflow-tooltip />
            <el-table-column label="合同额" width="100" align="right">
              <template #default="{ row }">{{ row.contract_amount ? row.contract_amount.toLocaleString() : '-' }}</template>
            </el-table-column>
            <el-table-column prop="project_status" label="状态" width="80" />
          </el-table>
          <p v-else style="color:#999;font-size:13px">暂无合作历史记录</p>
        </div>
      </template>
    </el-dialog>

    <el-dialog v-model="editVisible" :title="editing.id ? $t('common.edit') : $t('common.create')" width="650px" top="3vh">
      <el-form :model="form" label-width="100px" size="small">
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="简称" required><el-input v-model="form.short_name" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="名称" required><el-input v-model="form.full_name" /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="8"><el-form-item label="注册类型"><el-select v-model="form.registration_type" style="width:100%"><el-option v-for="t in regTypes" :key="t" :label="t" :value="t" /></el-select></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="国别"><el-input v-model="form.country" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="母公司"><el-input v-model="form.parent_short_name" placeholder="当地注册时填写" /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="专业分类" required><el-select v-model="form.profession_category" style="width:100%" @change="form.profession=''"><el-option v-for="c in categories" :key="c" :label="c" :value="c" /></el-select></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="主专业" required><el-select v-model="form.profession" style="width:100%"><el-option v-for="p in categoryProfessions" :key="p" :label="p" :value="p" /></el-select></el-form-item></el-col>
        </el-row>
        <el-form-item label="其他专业"><el-input v-model="form.other_professions" placeholder="逗号分隔" /></el-form-item>
        <el-divider content-position="left">法定代表人</el-divider>
        <el-row :gutter="12">
          <el-col :span="8"><el-form-item label="姓名" required><el-input v-model="form.legal_rep_name" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="电话" required><el-input v-model="form.legal_rep_phone" /></el-form-item></el-col>
          <el-col :span="8"><el-form-item label="身份证号"><el-input v-model="form.legal_rep_id" /></el-form-item></el-col>
        </el-row>
        <el-divider content-position="left">联系人</el-divider>
        <el-row :gutter="12">
          <el-col :span="6"><el-form-item label="姓名"><el-input v-model="form.contact_name" /></el-form-item></el-col>
          <el-col :span="6"><el-form-item label="职务"><el-input v-model="form.contact_title" /></el-form-item></el-col>
          <el-col :span="6"><el-form-item label="电话"><el-input v-model="form.contact_phone" /></el-form-item></el-col>
          <el-col :span="6"><el-form-item label="邮箱"><el-input v-model="form.contact_email" /></el-form-item></el-col>
        </el-row>
        <el-divider content-position="left">注册信息</el-divider>
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="营业执照"><el-input v-model="form.biz_license" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="税号"><el-input v-model="form.tax_id" /></el-form-item></el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="12"><el-form-item label="注册地址"><el-input v-model="form.reg_address" /></el-form-item></el-col>
          <el-col :span="12"><el-form-item label="注册资本"><el-input v-model="form.reg_capital" /></el-form-item></el-col>
        </el-row>
        <el-form-item label="备注"><el-input v-model="form.notes" type="textarea" /></el-form-item>
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
import { PROFESSION_CATEGORIES } from '../data/professions'

const list = ref<any[]>([])
const filterType = ref('')
const filterCategory = ref('')
const keyword = ref('')
const detailVisible = ref(false)
const detail = ref<any>(null)
const detailProjects = ref<any[]>([])
const blStatus = ref('')
const blLevel = ref('')
const blDate = ref('')
const blReason = ref('')
const localBL = ref<string[]>([])
const editVisible = ref(false)
const editing = ref<any>({})
const form = ref<any>({})

const regTypes = ['国内', '国外', '当地注册']
const categories = computed(() => PROFESSION_CATEGORIES.map(c => c.name))
const categoryProfessions = computed(() => {
  if (!form.value.profession_category) {
    let all: string[] = []
    PROFESSION_CATEGORIES.forEach(c => all.push(...c.professions))
    return [...new Set(all)]
  }
  const cat = PROFESSION_CATEGORIES.find(c => c.name === form.value.profession_category)
  return cat ? cat.professions : []
})

async function load() {
  const params: any = {}
  if (filterType.value) params.type = filterType.value
  if (filterCategory.value) params.category = filterCategory.value
  if (keyword.value) params.keyword = keyword.value
  try {
    const { data } = await axios.get('/api/subcontractors', { params })
    list.value = data || []
  } catch { /* ignore */ }
}

async function showDetail(row: any) {
  try {
    const { data } = await axios.get('/api/subcontractors/' + row.id)
    // New format: { base: {...}, projects: [...], blacklist_status: ... }
    // Old format: direct object (fallback)
    detail.value = data.base || data
    detailProjects.value = data.projects || []
    blStatus.value = data.blacklist_status || ''
    blLevel.value = data.restrict_level || ''
    blDate.value = data.list_date || ''
    blReason.value = data.list_reason || ''
    localBL.value = data.local_blacklisted || []
    detailVisible.value = true
  } catch { ElMessage.error('加载详情失败') }
}

function openDialog(row?: any) {
  editing.value = row ? { ...row } : {}
  form.value = row ? { ...row } : {
    short_name: '', full_name: '', registration_type: '国内', country: '',
    profession_category: '施工', profession: '', other_professions: '',
    parent_short_name: '', legal_rep_name: '', legal_rep_id: '', legal_rep_phone: '',
    contact_name: '', contact_title: '', contact_phone: '', contact_email: '',
    biz_license: '', tax_id: '', reg_address: '', reg_capital: '', notes: ''
  }
  editVisible.value = true
}

async function save() {
  const payload = { ...form.value }
  try {
    if (editing.value.id) {
      await axios.put('/api/subcontractors/' + editing.value.id, payload)
    } else {
      await axios.post('/api/subcontractors', payload)
    }
    ElMessage.success('Saved')
    editVisible.value = false
    load()
  } catch (err: any) {
    ElMessage.error(err?.response?.data || 'Save failed')
  }
}

async function doDelete() {
  await ElMessageBox.confirm('确认删除？')
  await axios.delete('/api/subcontractors/' + editing.value.id)
  ElMessage.success('Deleted')
  editVisible.value = false
  load()
}

onMounted(load)
</script>

<style scoped>
.sub-lib-page { display: flex; flex-direction: column; gap: 10px; }
.toolbar { display: flex; gap: 8px; flex-wrap: wrap; }
</style>
