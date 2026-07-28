package services

import "testing"

// 风险灯必须严格照搬细则 4.1，且取最严重的一项。
// 这张表是考核与预警的依据，判错会直接误导管理决策。
func TestJudgeLight(t *testing.T) {
	mk := func(lag int, spi float64, ms, lds bool) *ProgressData {
		d := &ProgressData{LagDays: lag, MilestoneRisk: ms, HasLDs: lds}
		if spi > 0 {
			d.PlanCum, d.ActCum = 100, spi*100
			d.SPI, d.SPIValid = spi, true
		}
		return d
	}

	cases := []struct {
		name string
		d    *ProgressData
		want string
	}{
		{"滞后0天且SPI1.0 → 绿", mk(0, 1.00, false, false), LightGreen},
		{"滞后3天且SPI0.95 → 绿（边界内）", mk(3, 0.95, false, false), LightGreen},
		{"滞后5天 → 黄", mk(5, 1.00, false, false), LightYellow},
		{"SPI0.92 → 黄", mk(0, 0.92, false, false), LightYellow},
		{"滞后10天 → 橙", mk(10, 1.00, false, false), LightOrange},
		{"SPI0.87 → 橙", mk(0, 0.87, false, false), LightOrange},
		{"付款里程碑滞后 → 橙（即便进度正常）", mk(0, 1.00, true, false), LightOrange},
		{"滞后20天 → 红", mk(20, 1.00, false, false), LightRed},
		{"SPI0.80 → 红", mk(0, 0.80, false, false), LightRed},
		{"已产生LDs → 红（即便进度全正常）", mk(0, 1.00, false, true), LightRed},
		{"多项触发取最严重：滞后5天+LDs → 红", mk(5, 1.00, false, true), LightRed},
		{"滞后10天+SPI0.80 → 红（SPI更严重）", mk(10, 0.80, false, false), LightRed},
	}
	for _, c := range cases {
		got, why := judgeLight(c.d)
		if got != c.want {
			t.Errorf("%s：期望 %s，实际 %s（原因：%s）", c.name, c.want, got, why)
		}
		if why == "" {
			t.Errorf("%s：未给出判定原因", c.name)
		}
	}
}

// 累计计划产值为 0 时 SPI 无法计算，此时不得据此判灯——
// 否则会把「没填数据」误判成「SPI=0 → 红灯」，冤枉项目部
func TestLightWithoutSPI(t *testing.T) {
	d := &ProgressData{LagDays: 2, SPIValid: false}
	got, _ := judgeLight(d)
	if got != LightGreen {
		t.Errorf("SPI 不可用且滞后 2 天时应判绿，实际 %s", got)
	}
	d2 := &ProgressData{LagDays: 20, SPIValid: false}
	if got, _ := judgeLight(d2); got != LightRed {
		t.Errorf("SPI 不可用但滞后 20 天仍应判红，实际 %s", got)
	}
}

func TestParseNum(t *testing.T) {
	cases := map[string]float64{
		"1200000": 1200000, "1,200,000": 1200000, "1，200，000": 1200000,
		"¥ 5000": 5000, "3000 元": 3000, "15 天": 15, "": 0, "abc": 0,
	}
	for in, want := range cases {
		if got := parseNum(in); got != want {
			t.Errorf("parseNum(%q) = %v，期望 %v", in, got, want)
		}
	}
}
