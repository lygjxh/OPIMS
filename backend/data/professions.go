// Package data provides the profession classification system for subcontractors.
// Two-tier classification: Category (采购/施工/设计/咨询) → Profession (13 standardized values).
// This file is the single source of truth for profession keyword mapping.
// When keywords change, re-import data for the mapping to take effect on existing records.
package data

// ProfessionCategory represents the first-tier classification.
type ProfessionCategory struct {
	Name        string   // 采购 / 施工 / 设计 / 咨询
	Professions []string // second-tier standardized names
}

// Categories is the ordered list of profession categories (first tier).
var Categories = []ProfessionCategory{
	{
		Name: "采购",
		Professions: []string{
			"材料",
			"机械",
			"人力",
			"检测",
		},
	},
	{
		Name: "施工",
		Professions: []string{
			"土建",
			"市政",
			"钢结构",
			"管道",
			"电气仪表",
			"暖通消防",
			"防腐保温",
			"设备安装",
			"装饰装修",
			"地基处理",
			"劳务",
			"检测",
			"其他",
		},
	},
	{
		Name: "设计",
		Professions: []string{
			"设计",
		},
	},
	{
		Name: "咨询",
		Professions: []string{
			"咨询",
		},
	},
}

// ProfessionKeywords maps raw management-ledger values to standardized second-tier professions.
// Keys are lowercase-normalized raw values; values are the standardized profession name.
// If a raw value is not found, it falls back to "其他".
var ProfessionKeywords = map[string]string{
	// === 土建 ===
	"工业建筑":   "土建",
	"房屋建筑":   "土建",
	"土建":     "土建",
	"建筑":     "土建",
	"建筑工程":   "土建",
	"土建工程":   "土建",
	"建筑安装":   "土建",
	"建筑装饰工程": "土建",

	// === 市政 ===
	"市政工程":   "市政",
	"公路工程":   "市政",
	"城市道路照明": "市政",
	"环保工程":   "市政",
	"道路换填":   "市政",
	"总图工程":   "市政",

	// === 钢结构 ===
	"钢结构":   "钢结构",
	"钢结构工程": "钢结构",

	// === 管道 ===
	"管道安装":     "管道",
	"管道":       "管道",
	"长输管道":     "管道",
	"工艺管道二标段": "管道",

	// === 电气仪表 ===
	"电气仪表": "电气仪表",
	"电仪":   "电气仪表",
	"智能化":  "电气仪表",
	"电力工程": "电气仪表",
	"电气电仪": "电气仪表",
	"机电工程": "电气仪表",
	"机电安装": "电气仪表",
	"标段一电仪": "电气仪表",
	"标段三电仪": "电气仪表",

	// === 暖通消防 ===
	"暖通消防":     "暖通消防",
	"暖通":       "暖通消防",
	"厂房给排水采暖": "暖通消防",

	// === 防腐保温 ===
	"防腐保温":     "防腐保温",
	"防腐":       "防腐保温",
	"保冷":       "防腐保温",
	"热处理":      "防腐保温",
	"保冷绝热标段一": "防腐保温",
	"保冷绝热标段二": "防腐保温",
	"防腐二标段":   "防腐保温",

	// === 设备安装 ===
	"设备安装":       "设备安装",
	"低温储罐":       "设备安装",
	"非标制作":       "设备安装",
	"设备":         "设备安装",
	"安装":         "设备安装",
	"安装工程":       "设备安装",
	"储罐":         "设备安装",
	"储罐、球罐、低温罐": "设备安装",
	"内罐":         "设备安装",

	// === 装饰装修 ===
	"装饰装修": "装饰装修",
	"幕墙工程": "装饰装修",

	// === 地基处理 ===
	"地基处理": "地基处理",
	"桩基工程": "地基处理",
	"模板脚手架": "地基处理",

	// === 劳务 ===
	"劳务公司": "劳务",
	"建筑劳务": "劳务",

	// === 检测 ===
	"检测类":  "检测",
	"无损检测": "检测",

	// === 其他 (explicit) ===
	"其他专业": "其他",
	"其他":   "其他",
	"园林绿化": "其他",
	"桥梁工程": "其他",

	// === composite values (pick the first match) ===
	"管道、钢结构":           "管道",
	"管道制作安装、设备安装": "管道",
	"土建、钢结构、安装":     "土建",
	"钢结构、工艺管道":       "钢结构",
}

// MapProfession maps a raw profession string to its standardized name.
// Returns the standardized profession name, or "其他" if no match.
func MapProfession(raw string) string {
	if raw == "" {
		return "其他"
	}
	if v, ok := ProfessionKeywords[raw]; ok {
		return v
	}
	return "其他"
}

// CategoryOf returns the first-tier category for a given standardized profession.
func CategoryOf(profession string) string {
	for _, cat := range Categories {
		for _, p := range cat.Professions {
			if p == profession {
				return cat.Name
			}
		}
	}
	return "施工" // default fallback
}
