import { createI18n } from 'vue-i18n'

const messages = {
  zh: {
    app: { title: 'OPIMS - 海外项目综合管理系统' },
    menu: {
      dashboard: '首页看板',
      projects: '项目清单',
      files: '项目文件',
      blacklistSub: '分包商黑名单',
      blacklistPerson: '人员黑名单',
      subcontractors: '分包商库',
      countryProfile: '国别档案',
      progress: '项目进度',
      timebar: '合同时效预警',
      quality: '项目质量',
      subcontract: '项目分包',
      personnel: '项目人员',
    },
    common: {
      save: '保存', cancel: '取消', delete: '删除', edit: '编辑',
      create: '新建', import: '导入', export: '导出', refresh: '刷新',
      search: '搜索', filter: '筛选', backup: '备份', restore: '恢复',
      confirm: '确认', confirmDelete: '确认删除？',
      success: '操作成功', error: '操作失败',
      lang: 'EN',
    },
    project: {
      shortName: '项目简称', contractNo: '合同编号', projectName: '项目名称',
      projectType: '项目类型', projectStatus: '项目状态',
      implementUnit: '实施单位', contractAmount: '合同额(万元)',
      domesticOverseas: '境内/境外', country: '国别',
      all: '全部', EPC: 'EPC', PC: 'PC', C: 'C',
      notStarted: '未开工', inProgress: '在建', suspended: '停工', completed: '完工',
    },
    file: {
      rootPath: '文件根目录', setRoot: '设置根目录',
      scan: '扫描文件', openFile: '打开文件',
    },
    blacklist: {
      subShortName: '分包商简称', subFullName: '分包商名称',
      listDate: '列入日期', delist: '拉出', listed: '列入中', delisted: '已拉出',
    },
    dashboard: {
      total: '项目总数', shortcuts: '快捷入口',
    }
  },
  en: {
    app: { title: 'OPIMS - Oversea Project Integrated Management System' },
    menu: {
      dashboard: 'Dashboard',
      projects: 'Project List',
      files: 'Project Files',
      blacklistSub: 'Subcontractor Blacklist',
      blacklistPerson: 'Personnel Blacklist',
      subcontractors: 'Subcontractor Library',
      countryProfile: 'Country Profile',
      progress: 'Progress',
      timebar: 'Time Bar Alerts',
      quality: 'Quality',
      subcontract: 'Subcontract',
      personnel: 'Personnel',
    },
    common: {
      save: 'Save', cancel: 'Cancel', delete: 'Delete', edit: 'Edit',
      create: 'Create', import: 'Import', export: 'Export', refresh: 'Refresh',
      search: 'Search', filter: 'Filter', backup: 'Backup', restore: 'Restore',
      confirm: 'Confirm', confirmDelete: 'Confirm delete?',
      success: 'Success', error: 'Error',
      lang: '中文',
    },
    project: {
      shortName: 'Short Name', contractNo: 'Contract No', projectName: 'Project Name',
      projectType: 'Type', projectStatus: 'Status',
      implementUnit: 'Unit', contractAmount: 'Amount(10k)',
      domesticOverseas: 'Domestic/Overseas', country: 'Country',
      all: 'All', EPC: 'EPC', PC: 'PC', C: 'C',
      notStarted: 'Not Started', inProgress: 'In Progress', suspended: 'Suspended', completed: 'Completed',
    },
    file: {
      rootPath: 'File Root', setRoot: 'Set Root',
      scan: 'Scan Files', openFile: 'Open File',
    },
    blacklist: {
      subShortName: 'Sub Short Name', subFullName: 'Sub Full Name',
      listDate: 'List Date', delist: 'Delist', listed: 'Listed', delisted: 'Delisted',
    },
    dashboard: {
      total: 'Total Projects', shortcuts: 'Shortcuts',
    }
  }
}

const i18n = createI18n({
  legacy: false,
  locale: 'zh',
  fallbackLocale: 'en',
  messages,
})

export default i18n
