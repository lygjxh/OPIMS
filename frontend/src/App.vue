<template>
  <el-container class="app-container">
    <el-aside width="220px" class="sidebar">
      <div class="logo">{{ $t('app.title') }}</div>
      <el-menu :default-active="route.path" router background-color="#1a1a2e" text-color="#a0a0b8" active-text-color="#4fc3f7">
        <el-menu-item index="/">
          <el-icon><DataBoard /></el-icon>
          <span>{{ $t('menu.dashboard') }}</span>
        </el-menu-item>
        <el-menu-item index="/projects">
          <el-icon><List /></el-icon>
          <span>{{ $t('menu.projects') }}</span>
        </el-menu-item>
        <el-menu-item index="/files">
          <el-icon><FolderOpened /></el-icon>
          <span>{{ $t('menu.files') }}</span>
        </el-menu-item>
        <el-menu-item index="/blacklist-sub">
          <el-icon><WarningFilled /></el-icon>
          <span>{{ $t('menu.blacklistSub') }}</span>
        </el-menu-item>
        <el-menu-item index="/blacklist-person" disabled>
          <el-icon><UserFilled /></el-icon>
          <span>{{ $t('menu.blacklistPerson') }}</span>
        </el-menu-item>
        <el-divider style="border-color:#333;margin:8px 0" />
        <el-menu-item index="/progress" disabled>
          <el-icon><TrendCharts /></el-icon>
          <span>{{ $t('menu.progress') }}</span>
        </el-menu-item>
        <el-menu-item index="/quality" disabled>
          <el-icon><CircleCheck /></el-icon>
          <span>{{ $t('menu.quality') }}</span>
        </el-menu-item>
        <el-menu-item index="/subcontract" disabled>
          <el-icon><Connection /></el-icon>
          <span>{{ $t('menu.subcontract') }}</span>
        </el-menu-item>
        <el-menu-item index="/personnel" disabled>
          <el-icon><Avatar /></el-icon>
          <span>{{ $t('menu.personnel') }}</span>
        </el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="topbar">
        <el-breadcrumb separator="/">
          <el-breadcrumb-item>{{ $t('menu.' + (route.name as string)) }}</el-breadcrumb-item>
        </el-breadcrumb>
        <el-button text @click="toggleLang" style="margin-left:auto">
          {{ $t('common.lang') }}
        </el-button>
      </el-header>
      <el-main>
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'

const route = useRoute()
const { locale } = useI18n()

function toggleLang() {
  locale.value = locale.value === 'zh' ? 'en' : 'zh'
}
</script>

<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
html, body, #app { height: 100%; font-family: 'Segoe UI', 'PingFang SC', sans-serif; }
.app-container { height: 100%; }
.sidebar { background: #1a1a2e; overflow-y: auto; }
.sidebar .logo { color: #4fc3f7; font-size: 14px; font-weight: 700; padding: 16px 12px; text-align: center; border-bottom: 1px solid #2a2a3e; }
.topbar { display: flex; align-items: center; background: #fff; border-bottom: 1px solid #e8e8e8; padding: 0 20px; height: 48px; }
.el-main { background: #f5f7fa; display: flex; flex-direction: column; overflow: hidden; }
</style>
