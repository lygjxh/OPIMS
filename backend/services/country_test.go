package services

import "testing"

// TestExtractCountry 覆盖关键词表的顺序敏感场景：
// 长国名必须先于其包含的短国名匹配，否则会被抢先误判。
func TestExtractCountry(t *testing.T) {
	cases := []struct {
		name, addr, want string
	}{
		// 基本匹配
		{"俄罗斯波罗的海天然气化工项目", "", "俄罗斯"},
		{"越南龙山油气工业开发区低温乙烷储罐项目", "", "越南"},
		{"纳米比亚能源枢纽港工程EPC项目", "", "纳米比亚"},
		{"斯里兰卡首都机场改扩建项目", "", "斯里兰卡"},
		{"老挝彭下-农波矿区钾肥项目", "", "老挝"},

		// 顺序敏感：长国名优先
		{"印度尼西亚某项目", "", "印尼"},          // 不能被"印度"抢先
		{"白俄罗斯某项目", "", "白俄罗斯"},         // 不能被"俄罗斯"抢先
		{"印度某炼油项目", "", "印度"},

		// 项目专有缩写
		{"尼日利亚PLF-LNG-EPCC项目", "", "尼日利亚"}, // 国名优先于缩写
		{"阿布扎比 ADNOC LNG 公用工程", "", "阿联酋"},
		{"德村天然气液化项目", "格什姆岛", "伊朗"},      // 靠地址命中

		// LNG 是工艺缩写，不得用于判断国别
		{"某国LNG接收站项目", "", "其他"},

		// 无命中
		{"未知地区某项目", "", "其他"},
	}

	for _, c := range cases {
		if got := extractCountry(c.name, c.addr); got != c.want {
			t.Errorf("extractCountry(%q, %q) = %q, want %q", c.name, c.addr, got, c.want)
		}
	}
}
