<template>
  <div class="ms">
    <div class="page-head">
      <div>
        <h1 class="page-title">月度报送</h1>
        <p class="page-desc">
          核查各在建项目当期该交的文件 · 收到的文件在此改名归档
        </p>
      </div>
    </div>

    <el-tabs v-model="tab" class="tabs">
      <el-tab-pane name="check">
        <template #label><span class="tab-lb"><el-icon><Document /></el-icon>报送核查</span></template>
        <ProgressCheck v-if="loaded.check" />
      </el-tab-pane>
      <el-tab-pane name="archive">
        <template #label><span class="tab-lb"><el-icon><FolderChecked /></el-icon>收文归档</span></template>
        <ArchiveInbox v-if="loaded.archive" />
      </el-tab-pane>
    </el-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, watch, reactive } from 'vue'
import { useRoute } from 'vue-router'
import { Document, FolderChecked } from '@element-plus/icons-vue'
import ProgressCheck from './ProgressCheck.vue'
import ArchiveInbox from './ArchiveInbox.vue'

// 收文归档原本是独立页面，旧地址 /progress/archive 重定向到 ?tab=archive，
// 所以初始标签页要认这个参数，否则老书签会落到报送核查上。
const route = useRoute()
const tabOf = (q: unknown) => (q === 'archive' ? 'archive' : 'check')
const tab = ref(tabOf(route.query.tab))

// 两个标签页各要扫一遍云盘目录，首次点开才加载，避免进页面就发两组扫描请求
const loaded = reactive<Record<string, boolean>>({
  check: tab.value === 'check', archive: tab.value === 'archive',
})
watch(tab, v => { loaded[v] = true })

// 从本页跳走再带 ?tab= 回来时，路由复用同一个组件实例、setup 不会重跑，
// 只认初始值的话标签页不会切过去。必须跟着 query 走。
watch(() => route.query.tab, q => { tab.value = tabOf(q) })
</script>

<style scoped>
.ms { display: flex; flex-direction: column; gap: 12px; }
.page-title { font-size: 22px; font-weight: 700; color: var(--c-text-strong); margin: 0; }
.page-desc { font-size: 13px; color: var(--c-text-muted); margin: 4px 0 0; }
.tab-lb { display: inline-flex; align-items: center; gap: 5px; }
.tabs :deep(.el-tabs__item) { font-size: 14px; }
</style>
