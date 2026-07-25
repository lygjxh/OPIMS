/**
 * 分包商专业分类体系 — 前端数据源。
 * 两层分类：专业分类（采购/施工/设计/咨询）→ 具体专业。
 * 与 backend/data/professions.go 保持同步。
 *
 * 进口管理台账数据时，基于 professionKeywords 自动映射。
 * 映射不到的记为「其他」。
 */

export interface ProfessionCategory {
  name: string
  professions: string[]
}

export const PROFESSION_CATEGORIES: ProfessionCategory[] = [
  {
    name: '采购',
    professions: ['材料', '机械', '人力', '检测'],
  },
  {
    name: '施工',
    professions: [
      '土建',
      '市政',
      '钢结构',
      '管道',
      '电气仪表',
      '暖通消防',
      '防腐保温',
      '设备安装',
      '装饰装修',
      '地基处理',
      '劳务',
      '检测',
      '其他',
    ],
  },
  {
    name: '设计',
    professions: ['设计'],
  },
  {
    name: '咨询',
    professions: ['咨询'],
  },
]

/**
 * 专业关键词 → 标准化专业名映射表。
 * 与 backend/data/professions.go 的 ProfessionKeywords 保持同步。
 */
export const professionKeywords: Record<string, string> = {
  // === 土建 ===
  工业建筑: '土建',
  房屋建筑: '土建',
  土建: '土建',
  建筑: '土建',
  建筑工程: '土建',
  土建工程: '土建',
  建筑安装: '土建',
  建筑装饰工程: '土建',

  // === 市政 ===
  市政工程: '市政',
  公路工程: '市政',
  城市道路照明: '市政',
  环保工程: '市政',
  道路换填: '市政',
  总图工程: '市政',

  // === 钢结构 ===
  钢结构: '钢结构',
  钢结构工程: '钢结构',

  // === 管道 ===
  管道安装: '管道',
  管道: '管道',
  长输管道: '管道',
  工艺管道二标段: '管道',

  // === 电气仪表 ===
  电气仪表: '电气仪表',
  电仪: '电气仪表',
  智能化: '电气仪表',
  电力工程: '电气仪表',
  电气电仪: '电气仪表',
  机电工程: '电气仪表',
  机电安装: '电气仪表',
  标段一电仪: '电气仪表',
  标段三电仪: '电气仪表',

  // === 暖通消防 ===
  暖通消防: '暖通消防',
  暖通: '暖通消防',
  厂房给排水采暖: '暖通消防',

  // === 防腐保温 ===
  防腐保温: '防腐保温',
  防腐: '防腐保温',
  保冷: '防腐保温',
  热处理: '防腐保温',
  保冷绝热标段一: '防腐保温',
  保冷绝热标段二: '防腐保温',
  防腐二标段: '防腐保温',

  // === 设备安装 ===
  设备安装: '设备安装',
  低温储罐: '设备安装',
  非标制作: '设备安装',
  设备: '设备安装',
  安装: '设备安装',
  安装工程: '设备安装',
  储罐: '设备安装',
  '储罐、球罐、低温罐': '设备安装',
  内罐: '设备安装',

  // === 装饰装修 ===
  装饰装修: '装饰装修',
  幕墙工程: '装饰装修',

  // === 地基处理 ===
  地基处理: '地基处理',
  桩基工程: '地基处理',
  模板脚手架: '地基处理',

  // === 劳务 ===
  劳务公司: '劳务',
  建筑劳务: '劳务',

  // === 检测 ===
  检测类: '检测',
  无损检测: '检测',

  // === 其他 ===
  其他专业: '其他',
  其他: '其他',
  园林绿化: '其他',
  桥梁工程: '其他',

  // === 复合值（取首个匹配） ===
  '管道、钢结构': '管道',
  '管道制作安装、设备安装': '管道',
  '土建、钢结构、安装': '土建',
  '钢结构、工艺管道': '钢结构',
}

/** 分包商层级（6 级，来自管理台账 F 列） */
export const SUBCONTRACTOR_TIERS = [
  '核心层',
  '紧密层',
  '普通层',
  '黑名单',
  '限制使用',
  '已退库',
] as const

export type SubcontractorTier = (typeof SUBCONTRACTOR_TIERS)[number]

/** 注册类型 */
export const REGISTRATION_TYPES = ['国内', '国外', '当地注册'] as const
export type RegistrationType = (typeof REGISTRATION_TYPES)[number]

/**
 * Map a raw profession string to its standardized name.
 * Falls back to '其他' when no match.
 */
export function mapProfession(raw: string): string {
  if (!raw) return '其他'
  return professionKeywords[raw] || '其他'
}

/**
 * Return the first-tier category for a standardized profession.
 */
export function categoryOf(profession: string): string {
  for (const cat of PROFESSION_CATEGORIES) {
    if (cat.professions.includes(profession)) return cat.name
  }
  return '施工'
}
