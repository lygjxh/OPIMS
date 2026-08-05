package services

import (
	"testing"
	"time"
)

func day(s string) time.Time {
	t, _ := time.ParseInLocation("2006-01-02", s, time.Local)
	return t
}

// 分级是雷达的核心判断，错一档就是「今天该办的事没提示」。
func TestLevelOf(t *testing.T) {
	today := day("2026-08-05")

	cases := []struct {
		name       string
		due        string
		remindDays int
		status     string
		wantLevel  string
		wantRemain int
	}{
		{"已逾期", "2026-08-01", 0, "open", LvOverdue, -4},
		{"今天到期", "2026-08-05", 0, "open", LvDue, 0},
		{"剩3天普通条目仍是紧急档", "2026-08-08", 0, "open", LvUrgent, 3},
		{"剩6天", "2026-08-11", 0, "open", LvUrgent, 6},
		{"剩7天", "2026-08-12", 0, "open", LvSoon, 7},
		{"剩14天", "2026-08-19", 0, "open", LvSoon, 14},
		{"剩15天", "2026-08-20", 0, "open", LvNormal, 15},
		// 视为认可类：到期前 3 日就必须进 due 档强制提醒（细则 6.3.2）
		{"risk剩3天进提醒窗口", "2026-08-08", 3, "open", LvDue, 3},
		{"risk剩4天尚未进窗口", "2026-08-09", 3, "open", LvUrgent, 4},
		{"已办结不再计紧急度", "2026-08-01", 0, "done", LvClosed, 0},
		{"不构成索赔也归入已关闭", "2026-08-01", 0, "not_claim", LvClosed, 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			remain, level := levelOf(c.due, c.remindDays, c.status, today)
			if level != c.wantLevel {
				t.Errorf("紧急度 = %s, 期望 %s", level, c.wantLevel)
			}
			if level != LvClosed && remain != c.wantRemain {
				t.Errorf("剩余天数 = %d, 期望 %d", remain, c.wantRemain)
			}
		})
	}
}

// 到期日解析不了时不能让整页崩，只降级成 normal。
func TestLevelOfBadDate(t *testing.T) {
	_, level := levelOf("待定", 0, "open", day("2026-08-05"))
	if level != LvNormal {
		t.Errorf("无法解析的日期应降级为 normal，实际 %s", level)
	}
}

// risk 类没填提醒天数时必须兜底 3 日，漏提醒的代价是权利不可逆丧失。
func TestRiskRemindDaysFallback(t *testing.T) {
	rows := []RegistryRow{{
		ID: 1, Project: "甲项目", Title: "业主指令未书面反对即视为接受",
		Direction: DirRisk, DueDate: "2026-08-08", Status: "open",
		// RemindDays 故意留 0
	}}
	res := BuildRadar(rows, nil, day("2026-08-05"))
	got := res.Items[0]
	if got.RemindDays != DefaultRiskRemindDays {
		t.Errorf("RemindDays = %d, 期望兜底为 %d", got.RemindDays, DefaultRiskRemindDays)
	}
	if got.Level != LvDue {
		t.Errorf("兜底提醒窗口未生效，Level = %s, 期望 %s", got.Level, LvDue)
	}
	if res.Summary.RiskAlert != 1 {
		t.Errorf("RiskAlert = %d, 期望 1", res.Summary.RiskAlert)
	}
}

// 同一条时效两边都有时必须合成一条，否则界面会出现两个对不上的倒计时。
func TestBuildRadarDedup(t *testing.T) {
	today := day("2026-08-05")
	rows := []RegistryRow{{
		ID: 1, Project: "甲项目", Code: "T1", Title: "中心登记的版本",
		DueDate: "2026-08-10", Status: "open",
	}}
	scan := &TimeBarResult{Items: []TimeBarItem{
		// 与登记簿同项目、同代码、同到期日 —— 应合并
		{Project: "甲项目", Code: "T1", Event: "项目部报的版本",
			DueAt: "2026-08-10", SourceFile: "甲项目-TBC-202607.xlsx"},
		// 到期日不同 —— 是另一条，不合并
		{Project: "甲项目", Code: "T1", Event: "另一条时效",
			DueAt: "2026-08-20", SourceFile: "甲项目-TBC-202607.xlsx"},
	}}

	res := BuildRadar(rows, scan, today)
	if len(res.Items) != 2 {
		t.Fatalf("合并后应为 2 条，实际 %d 条", len(res.Items))
	}

	var merged *RadarItem
	for i := range res.Items {
		if res.Items[i].DueDate == "2026-08-10" {
			merged = &res.Items[i]
		}
	}
	if merged == nil {
		t.Fatal("未找到合并后的那条")
	}
	if merged.Source != SrcBoth {
		t.Errorf("来源 = %s, 期望 %s", merged.Source, SrcBoth)
	}
	// 以中心登记簿为准，标题不能被附件 E 覆盖
	if merged.Title != "中心登记的版本" {
		t.Errorf("合并后标题 = %q, 应以登记簿为准", merged.Title)
	}
	// 附件 E 带来的溯源信息该保留下来
	if merged.SourceFile == "" {
		t.Error("合并后应保留附件 E 的来源文件名")
	}
}

// 排序即优先级：先按紧急度，同档内我方将丧失权利的排在索赔机会前面。
func TestSortRadarPriority(t *testing.T) {
	today := day("2026-08-05")
	rows := []RegistryRow{
		{ID: 1, Project: "丙", Title: "还早", DueDate: "2026-09-30", Status: "open"},
		{ID: 2, Project: "甲", Title: "索赔机会今天到期", Direction: DirClaim,
			DueDate: "2026-08-05", Status: "open"},
		{ID: 3, Project: "乙", Title: "我方视为认可今天到期", Direction: DirRisk,
			RemindDays: 0, DueDate: "2026-08-05", Status: "open"},
		{ID: 4, Project: "丁", Title: "已逾期", DueDate: "2026-08-01", Status: "open"},
	}

	res := BuildRadar(rows, nil, today)
	order := []int64{}
	for _, it := range res.Items {
		order = append(order, it.ID)
	}

	// 逾期(4) → 今天到期里 risk(3) 先于 claim(2) → 其余(1)
	want := []int64{4, 3, 2, 1}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("排序 = %v, 期望 %v（同档内 risk 必须排在 claim 前）", order, want)
		}
	}
}

// 登记簿不依赖云盘：附件 E 完全缺失时雷达照样要能出结果。
func TestBuildRadarWithoutScan(t *testing.T) {
	rows := []RegistryRow{{
		ID: 1, Project: "甲项目", Title: "合同评审时登记的时效",
		DueDate: "2026-08-06", Status: "open",
	}}
	res := BuildRadar(rows, nil, day("2026-08-05"))
	if len(res.Items) != 1 {
		t.Fatalf("附件 E 缺失时应仍有 1 条，实际 %d", len(res.Items))
	}
	if res.Summary.Total != 1 || res.Summary.Projects != 1 {
		t.Errorf("汇总不对：%+v", res.Summary)
	}
}
