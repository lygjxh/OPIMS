// Package data provides the region-country mapping for dashboard contract distribution.
// When adding a new country, update this file AND frontend/src/data/regions.ts.
package data

// RegionCountries maps region name to its member countries.
// The UNCLASSIFIED catch-all is handled programmatically: any country not found
// in any region is assigned "未分类".
// Source: SDD "地区口径" table, synced with frontend/src/data/regions.ts.
var RegionCountries = map[string][]string{
	"东南亚":    {"印尼", "越南", "泰国", "马来西亚", "新加坡", "菲律宾", "缅甸", "柬埔寨", "老挝", "文莱"},
	"中东":     {"阿联酋", "沙特", "伊拉克", "伊朗", "科威特", "阿曼", "卡塔尔", "约旦", "巴林", "也门", "土耳其"},
	"非洲":     {"尼日利亚", "纳米比亚", "埃及", "阿尔及利亚", "安哥拉", "坦桑尼亚", "莫桑比克", "刚果", "赞比亚", "几内亚", "津巴布韦"},
	"中亚与俄罗斯": {"俄罗斯", "哈萨克斯坦", "乌兹别克斯坦", "白俄罗斯", "土库曼斯坦", "吉尔吉斯斯坦", "塔吉克斯坦"},
	"南亚":     {"巴基斯坦", "孟加拉", "印度", "斯里兰卡", "尼泊尔"},
	"东北亚":    {"蒙古", "韩国", "日本"},
	"欧洲":     {"塞尔维亚", "波兰", "匈牙利", "德国", "法国"},
	"美洲":     {"巴西", "智利", "秘鲁", "墨西哥", "阿根廷"},
	"大洋洲":    {"澳大利亚", "新西兰", "巴布亚新几内亚"},
	"中国境内":   {"中国"},
}

// REGION_ORDER defines the display order for regions in charts (stable ordering).
var REGION_ORDER = func() []string {
	order := make([]string, 0, len(RegionCountries))
	for region := range RegionCountries {
		order = append(order, region)
	}
	// Append UNCLASSIFIED at the end
	order = append(order, "未分类")
	return order
}()

// countryToRegion is a reverse index built at init time.
var countryToRegion map[string]string

func init() {
	countryToRegion = make(map[string]string)
	for region, countries := range RegionCountries {
		for _, c := range countries {
			countryToRegion[c] = region
		}
	}
}

// RegionOf returns the region name for a given country.
// Returns "未分类" if the country is not found in any region.
func RegionOf(country string) string {
	if r, ok := countryToRegion[country]; ok {
		return r
	}
	return "未分类"
}
