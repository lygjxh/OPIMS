<template>
  <div class="placeholder">
    <div class="ph-card">
      <div class="ph-icon"><el-icon :size="34"><Tools /></el-icon></div>
      <h2>模块建设中</h2>
      <p>该功能模块正在规划开发，敬请期待</p>
      <el-button type="primary" plain @click="$router.push('/')">返回首页看板</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Tools } from '@element-plus/icons-vue'
</script>

<style scoped>
.placeholder { display: flex; align-items: center; justify-content: center; min-height: 70vh; }
.ph-card {
  text-align: center; background: var(--c-surface);
  border: 1px solid var(--c-border); border-radius: var(--radius);
  box-shadow: var(--shadow-card); padding: 48px 64px;
}
.ph-icon {
  width: 78px; height: 78px; margin: 0 auto 18px; border-radius: 20px;
  background: var(--c-accent-soft); color: var(--c-accent);
  display: flex; align-items: center; justify-content: center;
}
.ph-card h2 { font-size: 20px; color: var(--c-text-strong); margin: 0 0 8px; font-weight: 700; }
.ph-card p { font-size: 14px; color: var(--c-text-muted); margin: 0 0 22px; }
</style>
