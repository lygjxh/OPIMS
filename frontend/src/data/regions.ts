/**
 * 国别 → 地区（大区）映射，用于首页「合同额分布」按地区汇总。
 * 分区口径按海外工程业务习惯划分，新增国别时在对应地区追加即可。
 * 未列入的国别（含识别失败的"其他"）归入「未分类」。
 */
export const REGIONS: Record<string, string[]> = {
  东南亚: ['印尼', '越南', '泰国', '马来西亚', '新加坡', '菲律宾', '缅甸', '柬埔寨', '老挝', '文莱'],
  中东: ['阿联酋', '沙特', '伊拉克', '伊朗', '科威特', '阿曼', '卡塔尔', '约旦', '巴林', '也门', '土耳其'],
  非洲: ['尼日利亚', '纳米比亚', '埃及', '阿尔及利亚', '安哥拉', '坦桑尼亚', '莫桑比克', '刚果', '赞比亚', '几内亚', '津巴布韦'],
  中亚与俄罗斯: ['俄罗斯', '哈萨克斯坦', '乌兹别克斯坦', '白俄罗斯', '土库曼斯坦', '吉尔吉斯斯坦', '塔吉克斯坦'],
  南亚: ['巴基斯坦', '孟加拉', '印度', '斯里兰卡', '尼泊尔'],
  东北亚: ['蒙古', '韩国', '日本'],
  欧洲: ['塞尔维亚', '波兰', '匈牙利', '德国', '法国'],
  美洲: ['巴西', '智利', '秘鲁', '墨西哥', '阿根廷'],
  大洋洲: ['澳大利亚', '新西兰', '巴布亚新几内亚'],
  中国境内: ['中国'],
}

/** 反向索引：国别 → 地区 */
const COUNTRY_TO_REGION: Record<string, string> = (() => {
  const m: Record<string, string> = {}
  for (const [region, countries] of Object.entries(REGIONS)) {
    for (const c of countries) m[c] = region
  }
  return m
})()

/** 地区显示顺序（图表中同金额时的稳定次序） */
export const REGION_ORDER = [...Object.keys(REGIONS), '未分类']

export function regionOf(country: string): string {
  return COUNTRY_TO_REGION[country] || '未分类'
}
