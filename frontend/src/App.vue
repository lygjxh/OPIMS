<template>
  <div class="shell">
    <!-- 侧边栏 -->
    <aside class="sidebar">
      <div class="brand">
        <div class="brand-mark">OP</div>
        <div class="brand-text">
          <div class="brand-name">OPIMS</div>
          <div class="brand-sub">海外项目综合管理</div>
        </div>
      </div>

      <nav class="nav">
        <div class="nav-group">主业务</div>
        <template v-for="m in mainMenu" :key="m.path">
          <RouterLink v-if="!m.disabled" :to="m.path" class="nav-item">
            <el-icon class="nav-ic"><component :is="m.icon" /></el-icon>
            <span>{{ $t('menu.' + m.key) }}</span>
          </RouterLink>
          <button v-else type="button" class="nav-item pending" @click="notReady($t('menu.' + m.key))">
            <el-icon class="nav-ic"><component :is="m.icon" /></el-icon>
            <span>{{ $t('menu.' + m.key) }}</span>
            <span class="soon">待建</span>
          </button>
        </template>

        <div class="nav-group">扩展模块</div>
        <template v-for="m in extMenu" :key="m.path">
          <RouterLink v-if="!m.disabled" :to="m.path" class="nav-item">
            <el-icon class="nav-ic"><component :is="m.icon" /></el-icon>
            <span>{{ $t('menu.' + m.key) }}</span>
          </RouterLink>
          <button v-else type="button" class="nav-item pending" @click="notReady($t('menu.' + m.key))">
            <el-icon class="nav-ic"><component :is="m.icon" /></el-icon>
            <span>{{ $t('menu.' + m.key) }}</span>
            <span class="soon">待建</span>
          </button>
        </template>
      </nav>

      <div class="side-foot">v0.1 · for-claude</div>
    </aside>

    <!-- 主区 -->
    <div class="main-wrap">
      <header class="topbar">
        <div class="crumb">
          <span class="crumb-home">首页</span>
          <span class="crumb-sep">/</span>
          <span class="crumb-cur">{{ $t('menu.' + (route.name as string)) }}</span>
        </div>
        <div class="top-actions">
          <el-button text class="lang-btn" @click="toggleLang">
            <el-icon><Switch /></el-icon>{{ $t('common.lang') }}
          </el-button>
          <div class="user">
            <div class="avatar">运</div>
            <span class="user-name">海外运营中心</span>
          </div>
        </div>
      </header>

      <main class="content">
        <router-view />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import {
  DataBoard, List, FolderOpened, WarningFilled, UserFilled,
  TrendCharts, CircleCheck, Connection, Avatar, Switch, OfficeBuilding, Flag,
} from '@element-plus/icons-vue'

const route = useRoute()
const { locale } = useI18n()

const mainMenu = [
  { path: '/', key: 'dashboard', icon: DataBoard, disabled: false },
  { path: '/projects', key: 'projects', icon: List, disabled: false },
  { path: '/files', key: 'files', icon: FolderOpened, disabled: false },
  { path: '/blacklist-sub', key: 'blacklistSub', icon: WarningFilled, disabled: false },
  { path: '/country-profile', key: 'countryProfile', icon: Flag, disabled: false },
  { path: '/blacklist-person', key: 'blacklistPerson', icon: UserFilled, disabled: true },
]
const extMenu = [
  { path: '/subcontract', key: 'subcontract', icon: Connection, disabled: false },
  { path: '/subcontractors', key: 'subcontractors', icon: OfficeBuilding, disabled: false },
  { path: '/progress', key: 'progress', icon: TrendCharts, disabled: false },
  { path: '/quality', key: 'quality', icon: CircleCheck, disabled: true },
  { path: '/personnel', key: 'personnel', icon: Avatar, disabled: true },
]

function toggleLang() {
  locale.value = locale.value === 'zh' ? 'en' : 'zh'
}

function notReady(name: string) {
  ElMessage.info(`${name} 模块建设中`)
}
</script>

<style scoped>
.shell { display: flex; height: 100%; }

/* ---------- 侧边栏 ---------- */
.sidebar {
  width: 232px;
  flex-shrink: 0;
  background: linear-gradient(180deg, var(--c-side-bg) 0%, var(--c-side-bg-2) 100%);
  display: flex;
  flex-direction: column;
  color: var(--c-side-text);
}
.brand {
  display: flex; align-items: center; gap: 12px;
  padding: 20px 18px;
  border-bottom: 1px solid rgba(255, 255, 255, .08);
}
.brand-mark {
  width: 38px; height: 38px; border-radius: 10px;
  background: linear-gradient(135deg, var(--c-secondary), var(--c-primary));
  color: #fff; font-weight: 800; font-size: 15px;
  display: flex; align-items: center; justify-content: center;
  box-shadow: 0 4px 12px rgba(59, 130, 246, .4);
}
.brand-name { color: #fff; font-weight: 700; font-size: 18px; letter-spacing: .5px; }
.brand-sub { font-size: 11px; color: var(--c-side-text); margin-top: 2px; }

.nav { flex: 1; overflow-y: auto; padding: 12px 10px; }
.nav-group {
  font-size: 11px; color: #5f7398; letter-spacing: 1px;
  padding: 14px 10px 6px;
}
.nav-item {
  display: flex; align-items: center; gap: 11px;
  padding: 10px 12px; margin: 2px 0;
  border-radius: 9px; color: var(--c-side-text);
  text-decoration: none; font-size: 14px;
  transition: background .18s, color .18s;
  position: relative;
}
.nav-item:hover { background: rgba(255, 255, 255, .06); color: #fff; }
.nav-item.router-link-exact-active {
  background: var(--c-side-active-bg);
  color: #fff; font-weight: 600;
}
.nav-item.router-link-exact-active::before {
  content: ''; position: absolute; left: -10px; top: 8px; bottom: 8px;
  width: 3px; border-radius: 0 3px 3px 0; background: var(--c-accent);
}
.nav-ic { font-size: 17px; }
/* 待建模块：可点击并给出提示，而不是死链 */
.nav-item.pending {
  width: 100%; font: inherit; font-size: 14px; text-align: left;
  background: none; border: 0; cursor: pointer; opacity: .62;
}
.nav-item.pending:hover { opacity: .95; background: rgba(255, 255, 255, .06); }
.nav-item:focus-visible { outline: 2px solid var(--c-secondary); outline-offset: 1px; }
.soon {
  margin-left: auto; font-size: 10px; padding: 1px 6px;
  border-radius: 5px; background: rgba(255, 255, 255, .1); color: #8ea3c7;
}
.side-foot {
  padding: 12px 18px; font-size: 11px; color: #4d6188;
  border-top: 1px solid rgba(255, 255, 255, .07);
}

/* ---------- 主区 ---------- */
.main-wrap { flex: 1; display: flex; flex-direction: column; min-width: 0; }
.topbar {
  height: 58px; flex-shrink: 0;
  background: var(--c-surface);
  border-bottom: 1px solid var(--c-border);
  display: flex; align-items: center; padding: 0 22px;
}
.crumb { display: flex; align-items: center; gap: 8px; font-size: 14px; }
.crumb-home { color: var(--c-text-muted); }
.crumb-sep { color: var(--c-border-strong); }
.crumb-cur { color: var(--c-text-strong); font-weight: 600; }
.top-actions { margin-left: auto; display: flex; align-items: center; gap: 14px; }
.lang-btn { color: var(--c-text-muted); font-weight: 500; }
.user { display: flex; align-items: center; gap: 9px; }
.user-name { font-size: 13px; color: var(--c-text); font-weight: 500; }
.avatar {
  width: 34px; height: 34px; border-radius: 50%;
  background: linear-gradient(135deg, var(--c-primary), var(--c-primary-700));
  color: #fff; font-size: 13px; font-weight: 600;
  display: flex; align-items: center; justify-content: center;
}
.content { flex: 1; overflow-y: auto; padding: 22px; background: var(--c-bg); }
</style>
