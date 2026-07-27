package data

import "strings"

// 政策库中的国别写法与 OPIMS 标准国别名不完全一致：
//   - Obsidian 笔记用全称或带地区后缀（印度尼西亚、阿联酋-阿布扎比）
//   - OPIMS 项目数据用简称（印尼、阿联酋，见 services/country.go 的 countryRules）
// 不归一化就无法把政策与项目关联起来，故在此维护映射表。
//
// 新增国别档案时，若其文件名/「国家」字段与 OPIMS 标准名不同，在此补一条即可。
var policyCountryAlias = map[string]string{
	"印度尼西亚":   "印尼",
	"阿联酋-阿布扎比": "阿联酋",
	"阿布扎比":    "阿联酋",
	"俄罗斯联邦":   "俄罗斯",
	"越南社会主义共和国": "越南",
	"尼日利亚联邦共和国": "尼日利亚",
	"纳米比亚共和国": "纳米比亚",
}

// NormalizeCountry 将政策库中的国别写法归一化为 OPIMS 标准国别名。
// 无映射时原样返回（说明两侧写法本就一致，如「俄罗斯」「越南」）。
func NormalizeCountry(name string) string {
	n := strings.TrimSpace(name)
	if std, ok := policyCountryAlias[n]; ok {
		return std
	}
	return n
}
