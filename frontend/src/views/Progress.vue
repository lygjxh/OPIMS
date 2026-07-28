<template>
  <div class="pg">
    <div class="page-head">
      <div>
        <h1 class="page-title">项目进度</h1>
        <p class="page-desc">报送核查 · 进度指标与风险灯 · 工期索赔跟踪</p>
      </div>
    </div>

    <el-tabs v-model="tab" class="tabs">
      <el-tab-pane name="indicators">
        <template #label><span class="tab-lb"><el-icon><TrendCharts /></el-icon>进度指标</span></template>
        <ProgressIndicators v-if="loaded.indicators" />
      </el-tab-pane>
      <el-tab-pane name="check">
        <template #label><span class="tab-lb"><el-icon><Document /></el-icon>报送核查</span></template>
        <ProgressCheck v-if="loaded.check" />
      </el-tab-pane>
      <el-tab-pane name="eot">
        <template #label><span class="tab-lb"><el-icon><Money /></el-icon>工期索赔</span></template>
        <EOTList v-if="loaded.eot" />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, reactive } from 'vue'
import { TrendCharts, Document, Money } from '@element-plus/icons-vue'
import ProgressIndicators from './ProgressIndicators.vue'
import ProgressCheck from './ProgressCheck.vue'
import EOTList from './EOTList.vue'

const tab = ref('indicators')
// 各标签页首次点开才加载，避免一次性发起三组扫描请求
const loaded = reactive<Record<string, boolean>>({ indicators: true, check: false, eot: false })
watch(tab, v => { loaded[v] = true })
</script>

<style scoped>
.pg { display: flex; flex-direction: column; gap: 12px; }
.page-title { font-size: 22px; font-weight: 700; color: var(--c-text-strong); margin: 0; }
.page-desc { font-size: 13px; color: var(--c-text-muted); margin: 4px 0 0; }
.tab-lb { display: inline-flex; align-items: center; gap: 5px; }
.tabs :deep(.el-tabs__item) { font-size: 14px; }
</style>
