package services

// countryKeywords maps project-name/address keywords to country names.
// Edit this list when new countries or project-specific abbreviations appear.
var countryKeywords = map[string]string{
	// country names (Chinese)
	"蒙古":   "蒙古",
	"俄罗斯":  "俄罗斯",
	"伊拉克":  "伊拉克",
	"印尼":   "印尼",
	"印度尼西亚": "印尼",
	"尼日利亚":  "尼日利亚",
	"纳米比亚":  "纳米比亚",
	"阿布扎比":  "阿联酋",
	"阿联酋":   "阿联酋",
	"伊朗":   "伊朗",

	// project-specific abbreviations
	"格什姆":  "伊朗",
	"ADNOC": "阿联酋",
	"DBN":   "阿联酋",
	"LNG":   "阿联酋",
	"PLF":   "尼日利亚",
}
