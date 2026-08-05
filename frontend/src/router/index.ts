import { createRouter, createWebHashHistory } from 'vue-router'

// 路由按侧边栏的七大类组织：
//   首页看板 / 项目清单 / 项目文件 / 国别档案 / 人员管理 / 分包管理 / 进度管理 / 质量管理
// meta.group 指向所属大类的 i18n key，供面包屑显示「首页 / 进度管理 / 指标」这样的层级。
// 顶级项自身不带 group。
const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', name: 'dashboard', component: () => import('../views/Dashboard.vue') },
    { path: '/projects', name: 'projects', component: () => import('../views/ProjectList.vue') },
    { path: '/files', name: 'files', component: () => import('../views/FileManager.vue') },
    { path: '/country-profile', name: 'countryProfile', component: () => import('../views/CountryProfile.vue') },

    // ---- 人员管理 ----
    { path: '/personnel', name: 'personnel', component: () => import('../views/Placeholder.vue') },
    {
      path: '/personnel/blacklist', name: 'blacklistPerson',
      component: () => import('../views/Placeholder.vue'), meta: { group: 'personnel' },
    },

    // ---- 分包管理 ----
    { path: '/subcontract', name: 'subcontract', component: () => import('../views/SubcontractList.vue') },
    {
      path: '/subcontract/library', name: 'subcontractors',
      component: () => import('../views/SubcontractorLib.vue'), meta: { group: 'subcontract' },
    },
    {
      path: '/subcontract/blacklist', name: 'blacklistSub',
      component: () => import('../views/SubBlacklist.vue'), meta: { group: 'subcontract' },
    },

    // ---- 进度管理 ----
    { path: '/progress', name: 'progress', component: () => import('../views/ProgressOverview.vue') },
    {
      path: '/progress/indicators', name: 'progressIndicators',
      component: () => import('../views/ProgressIndicators.vue'), meta: { group: 'progress' },
    },
    {
      path: '/progress/check', name: 'progressCheck',
      component: () => import('../views/ProgressCheck.vue'), meta: { group: 'progress' },
    },
    {
      path: '/progress/eot', name: 'progressEot',
      component: () => import('../views/EOTList.vue'), meta: { group: 'progress' },
    },
    {
      path: '/progress/timebar', name: 'timebar',
      component: () => import('../views/TimeBar.vue'), meta: { group: 'progress' },
    },
    {
      path: '/progress/archive', name: 'archive',
      component: () => import('../views/ArchiveInbox.vue'), meta: { group: 'progress' },
    },

    // ---- 质量管理 ----
    { path: '/quality', name: 'quality', component: () => import('../views/Placeholder.vue') },

    // ---- 旧路径重定向 ----
    // 2026-08-05 导航改版前的地址。用户浏览器书签、历史记录里可能还留着，
    // 直接删会变成白屏，所以保留重定向，不要清理掉。
    { path: '/subcontractors', redirect: '/subcontract/library' },
    { path: '/blacklist-sub', redirect: '/subcontract/blacklist' },
    { path: '/blacklist-person', redirect: '/personnel/blacklist' },
    { path: '/timebar', redirect: '/progress/timebar' },
    { path: '/archive', redirect: '/progress/archive' },
  ]
})

export default router
