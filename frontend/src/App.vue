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
        <template v-for="m in menu" :key="m.path">
          <div class="nav-row">
            <RouterLink v-if="!m.disabled" :to="m.path" class="nav-item">
              <el-icon class="nav-ic"><component :is="m.icon" /></el-icon>
              <span>{{ $t('menu.' + m.key) }}</span>
            </RouterLink>
            <button v-else type="button" class="nav-item pending" @click="notReady($t('menu.' + m.key))">
              <el-icon class="nav-ic"><component :is="m.icon" /></el-icon>
              <span>{{ $t('menu.' + m.key) }}</span>
              <span class="soon">待建</span>
            </button>
            <!-- 展开箭头独立成键，避免与父级本身的跳转抢点击 -->
            <button v-if="m.children" type="button" class="caret" :class="{ open: isOpen(m.key) }"
              :aria-expanded="isOpen(m.key)"
              :aria-label="(isOpen(m.key) ? '收起 ' : '展开 ') + $t('menu.' + m.key)"
              @click="toggle(m.key)">
              <el-icon><ArrowRight /></el-icon>
            </button>
          </div>

          <div v-if="m.children && isOpen(m.key)" class="sub">
            <template v-for="c in m.children" :key="c.path">
              <RouterLink v-if="!c.disabled" :to="c.path" class="nav-item sub-item">
                <span class="dot" />
                <span>{{ $t('menu.' + c.key) }}</span>
              </RouterLink>
              <button v-else type="button" class="nav-item sub-item pending"
                @click="notReady($t('menu.' + c.key))">
                <span class="dot" />
                <span>{{ $t('menu.' + c.key) }}</span>
                <span class="soon">待建</span>
              </button>
            </template>
          </div>
        </template>
      </nav>

      <div class="side-foot">v0.1 · for-claude</div>
    </aside>

    <!-- 主区 -->
    <div class="main-wrap">
      <header class="topbar">
        <div class="crumb">
          <span class="crumb-home">首页</span>
          <template v-if="route.meta.group">
            <span class="crumb-sep">/</span>
            <span class="crumb-home">{{ $t('menu.' + (route.meta.group as string)) }}</span>
          </template>
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
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import {
  DataBoard, List, FolderOpened, TrendCharts, CircleCheck,
  Connection, Avatar, Switch, Flag, ArrowRight,
} from '@element-plus/icons-vue'

const route = useRoute()
const { locale } = useI18n()

// 七大类导航。父级本身就是原来的那个页面（进度管理是新增的总览页），
// 子菜单挂在下面；只有二级，不做三级。
const menu = [
  { path: '/', key: 'dashboard', icon: DataBoard, disabled: false },
  { path: '/projects', key: 'projects', icon: List, disabled: false },
  { path: '/files', key: 'files', icon: FolderOpened, disabled: false },
  { path: '/country-profile', key: 'countryProfile', icon: Flag, disabled: false },
  {
    path: '/personnel', key: 'personnel', icon: Avatar, disabled: true,
    children: [
      { path: '/personnel/blacklist', key: 'blacklistPerson', disabled: true },
    ],
  },
  {
    path: '/subcontract', key: 'subcontract', icon: Connection, disabled: false,
    children: [
      { path: '/subcontract/library', key: 'subcontractors', disabled: false },
      { path: '/subcontract/blacklist', key: 'blacklistSub', disabled: false },
    ],
  },
  {
    path: '/progress', key: 'progress', icon: TrendCharts, disabled: false,
    children: [
      { path: '/progress/indicators', key: 'progressIndicators', disabled: false },
      { path: '/progress/check', key: 'progressCheck', disabled: false },
      { path: '/progress/eot', key: 'progressEot', disabled: false },
      { path: '/progress/timebar', key: 'timebar', disabled: false },
      { path: '/progress/archive', key: 'archive', disabled: false },
    ],
  },
  { path: '/quality', key: 'quality', icon: CircleCheck, disabled: true },
]

// 展开的分组。用户手动收起后不强行掰回来，只在进入某个分组的页面时自动展开它。
const opened = ref<string[]>([])
const isOpen = (k: string) => opened.value.includes(k)
function toggle(k: string) {
  const i = opened.value.indexOf(k)
  if (i >= 0) opened.value.splice(i, 1)
  else opened.value.push(k)
}
watch(
  () => route.path,
  () => {
    // 子页面看 meta.group，父级页面看自己的 key
    const g = (route.meta.group as string) || menu.find(m => m.path === route.path && m.children)?.key
    if (g && !opened.value.includes(g)) opened.value.push(g)
  },
  { immediate: true },
)

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
/* 父级一行：菜单项占满，展开箭头贴右 */
.nav-row { display: flex; align-items: center; }
.nav-row > .nav-item { flex: 1; min-width: 0; }
.caret {
  flex-shrink: 0; width: 26px; height: 30px;
  display: flex; align-items: center; justify-content: center;
  background: none; border: 0; cursor: pointer; padding: 0;
  color: #7d90b3; border-radius: 6px;
  transition: transform .18s, background .18s, color .18s;
}
.caret:hover { background: rgba(255, 255, 255, .08); color: #fff; }
.caret:focus-visible { outline: 2px solid var(--c-secondary); outline-offset: 1px; }
.caret.open { transform: rotate(90deg); }
.caret .el-icon { font-size: 13px; }

/* 子菜单：靠缩进和小圆点区分层级，不再用分组标题。
   选择器必须带 .nav-item —— 单写 .sub-item 特异性与 .nav-item 相同，
   会被后面 .nav-item 的 padding/font-size 简写覆盖掉。 */
.sub { margin: 1px 0 4px; }
.nav-item.sub-item { padding-left: 34px; font-size: 13.5px; }
.dot {
  width: 4px; height: 4px; border-radius: 50%;
  background: currentColor; opacity: .55; flex-shrink: 0;
}
.nav-item.sub-item.router-link-exact-active .dot { opacity: 1; }

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
