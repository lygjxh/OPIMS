import { createRouter, createWebHashHistory } from 'vue-router'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    { path: '/', name: 'dashboard', component: () => import('../views/Dashboard.vue') },
    { path: '/projects', name: 'projects', component: () => import('../views/ProjectList.vue') },
    { path: '/files', name: 'files', component: () => import('../views/FileManager.vue') },
    { path: '/blacklist-sub', name: 'blacklistSub', component: () => import('../views/SubBlacklist.vue') },
    { path: '/country-profile', name: 'countryProfile', component: () => import('../views/CountryProfile.vue') },
    { path: '/progress', name: 'progress', component: () => import('../views/Placeholder.vue') },
    { path: '/quality', name: 'quality', component: () => import('../views/Placeholder.vue') },
    { path: '/subcontract', name: 'subcontract', component: () => import('../views/SubcontractList.vue') },
    { path: '/subcontractors', name: 'subcontractors', component: () => import('../views/SubcontractorLib.vue') },
    { path: '/personnel', name: 'personnel', component: () => import('../views/Placeholder.vue') },
    { path: '/blacklist-person', name: 'blacklistPerson', component: () => import('../views/Placeholder.vue') },
  ]
})

export default router
