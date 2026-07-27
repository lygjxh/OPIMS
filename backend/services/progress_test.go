package services

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"opims/data"
)

func mustItem(t *testing.T, code string) data.ChecklistItem {
	t.Helper()
	it, ok := data.FindChecklistItem(code)
	if !ok {
		t.Fatalf("清单项 %s 不存在", code)
	}
	return it
}

// 截止日是「基准月 + 日」规则算出来的，不是固定日期——这是 V1.1 的核心调整。
func TestDeadlineRules(t *testing.T) {
	p, _ := ParsePeriod("2026-07")
	cases := []struct{ code, want string }{
		{"MPR", "2026-08-05"}, // 次月 5 日
		{"SIX", "2026-07-30"}, // 当月 30 日
		{"RP3", "2026-07-25"}, // 当月 25 日
		{"QOV", "2026-09-25"}, // 2026-07 属 Q3，季度末月为 9 月
	}
	for _, c := range cases {
		got := Deadline(mustItem(t, c.code), p).Format("2006-01-02")
		if got != c.want {
			t.Errorf("%s 截止日 = %s，期望 %s", c.code, got, c.want)
		}
	}

	// 「当月 30 日」落在 2 月时应取该月最后一天，不能溢出到 3 月
	feb, _ := ParsePeriod("2026-02")
	if got := Deadline(mustItem(t, "SIX"), feb).Format("2006-01-02"); got != "2026-02-28" {
		t.Errorf("2 月的「30 日」应收敛为 2026-02-28，实际 %s", got)
	}
}

func TestPeriodString(t *testing.T) {
	p, _ := ParsePeriod("2026-07")
	if got := PeriodString(mustItem(t, "MPR"), p); got != "202607" {
		t.Errorf("月度周期串 = %s，期望 202607", got)
	}
	if got := PeriodString(mustItem(t, "QOV"), p); got != "2026Q3" {
		t.Errorf("季度周期串 = %s，期望 2026Q3", got)
	}
}

// 搭一个与真实云盘同构的项目目录
func makeProjectDir(t *testing.T) (root string, proj string) {
	t.Helper()
	root = t.TempDir()
	proj = "尼日利亚PLF项目"
	for _, d := range []string{
		"05.Schedule", "05.Schedule/Time Bar", "05.Schedule/EOT",
		"06.Report/Monthly Report", "06.Report/Weekly Report",
	} {
		if err := os.MkdirAll(filepath.Join(root, proj, filepath.FromSlash(d)), 0755); err != nil {
			t.Fatal(err)
		}
	}
	return root, proj
}

func writeFile(t *testing.T, root, proj, rel, name string, mod time.Time) string {
	t.Helper()
	p := filepath.Join(root, proj, filepath.FromSlash(rel), name)
	if err := os.WriteFile(p, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, mod, mod); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCheckSubmissions(t *testing.T) {
	root, proj := makeProjectDir(t)
	mk := func(rel, name string, mod time.Time) { writeFile(t, root, proj, rel, name, mod) }

	// 按时提交（截止 2026-08-05）
	mk("06.Report/Monthly Report", proj+"-MPR-202607.pdf", time.Date(2026, 8, 3, 10, 0, 0, 0, time.Local))
	// 迟交（截止 2026-07-30）
	mk("06.Report/Monthly Report", proj+"-SIX-202607.docx", time.Date(2026, 8, 10, 10, 0, 0, 0, time.Local))
	// 同期多版本，应取 v2
	mk("05.Schedule/Time Bar", proj+"-TBC-202607.xlsx", time.Date(2026, 8, 1, 10, 0, 0, 0, time.Local))
	mk("05.Schedule/Time Bar", proj+"-TBC-202607-v2.xlsx", time.Date(2026, 8, 2, 10, 0, 0, 0, time.Local))
	// 进度计划：L2 有 v1/v3，应取 v3
	mk("05.Schedule", proj+"-L2-v1.mpp", time.Now())
	mk("05.Schedule", proj+"-L2-v3.mpp", time.Now())
	mk("05.Schedule", proj+"-L1-v1.mpp", time.Now())
	// 命名不合规 → 待人工识别
	mk("05.Schedule", "随手起的计划名 6.23.mpp", time.Now())
	// 周报：本月 1 份
	mk("06.Report/Weekly Report", proj+"-WKR-2026W28.docx", time.Now())

	res, err := CheckSubmissions(root, []ProjectBrief{{ShortName: proj, Country: "尼日利亚"}}, "2026-07")
	if err != nil {
		t.Fatalf("CheckSubmissions: %v", err)
	}
	if len(res.Rows) != 1 {
		t.Fatalf("期望 1 行，实际 %d", len(res.Rows))
	}
	row := res.Rows[0]

	cell := func(code string) SubmissionCell {
		for _, c := range row.Cells {
			if c.Code == code {
				return c
			}
		}
		t.Fatalf("看板中无 %s 列", code)
		return SubmissionCell{}
	}

	if c := cell("MPR"); c.Status != StatusSubmitted {
		t.Errorf("MPR 应为已交，实际 %s", c.Status)
	}
	if c := cell("SIX"); c.Status != StatusLate {
		t.Errorf("SIX 超过 2026-07-30 应为迟交，实际 %s", c.Status)
	}
	if c := cell("EOT"); c.Status != StatusMissing {
		t.Errorf("EOT 无文件应为缺交，实际 %s", c.Status)
	}
	// 同期多版本取版本号最大者
	if c := cell("TBC"); c.FileName != proj+"-TBC-202607-v2.xlsx" {
		t.Errorf("TBC 应取 v2，实际 %s", c.FileName)
	}
	// 季度项：2026-07 落在 Q3
	if c := cell("QOV"); c.Period != "2026Q3" {
		t.Errorf("QOV 周期应为 2026Q3，实际 %s", c.Period)
	}
	// 周报不进看板单元格
	for _, c := range row.Cells {
		if c.Code == "WKR" {
			t.Error("周报不应作为月度看板单元格")
		}
	}
	if row.WeekCount != 1 {
		t.Errorf("本月周报数应为 1，实际 %d", row.WeekCount)
	}

	// 进度计划取各级最新版本
	if len(row.Plans) != 2 {
		t.Fatalf("应识别出 L1、L2 两级计划，实际 %d", len(row.Plans))
	}
	for _, p := range row.Plans {
		if p.Level == "L2" && p.Version != 3 {
			t.Errorf("L2 应取 v3，实际 v%d", p.Version)
		}
	}

	// 命名不合规的文件进入待人工识别，且不影响扫描
	if len(res.Unmatched) != 1 || res.Unmatched[0].FileName != "随手起的计划名 6.23.mpp" {
		t.Errorf("待人工识别清单不对：%+v", res.Unmatched)
	}

	// 汇总：6 个必交项中 MPR 按时、SIX 迟交、TBC 按时，其余缺交
	if res.Summary.Submitted != 2 || res.Summary.Late != 1 {
		t.Errorf("汇总不对：已交 %d 迟交 %d 缺交 %d",
			res.Summary.Submitted, res.Summary.Late, res.Summary.Missing)
	}
}

// 命名规范但属于其它周期的文件，不应被列入「待人工识别」——
// 该清单只表示「命名不合规」，混入其它周期文件会误导管理员去改本来就对的文件。
func TestOtherPeriodFilesNotUnmatched(t *testing.T) {
	root, proj := makeProjectDir(t)
	// 本期（2026-07）无文件；下列文件命名规范但属于别的周期
	writeFile(t, root, proj, "06.Report/Monthly Report", proj+"-MPR-202608.pdf", time.Now())
	writeFile(t, root, proj, "05.Schedule", proj+"-L3-v2.mpp", time.Now())
	// 这个才是真正命名不合规的
	writeFile(t, root, proj, "06.Report/Monthly Report", "六月月报最终版.docx", time.Now())

	res, err := CheckSubmissions(root, []ProjectBrief{{ShortName: proj}}, "2026-07")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Unmatched) != 1 {
		names := []string{}
		for _, u := range res.Unmatched {
			names = append(names, u.FileName)
		}
		t.Fatalf("待人工识别应只含 1 个不合规文件，实际 %d 个：%v", len(res.Unmatched), names)
	}
	if res.Unmatched[0].FileName != "六月月报最终版.docx" {
		t.Errorf("应识别出的不合规文件不对：%s", res.Unmatched[0].FileName)
	}
}

// 目录缺失时应记为缺交并标记未建目录，而不是报错中断（需求 7.5）
func TestMissingFolderTolerated(t *testing.T) {
	root := t.TempDir()
	proj := "空项目"
	if err := os.MkdirAll(filepath.Join(root, proj), 0755); err != nil {
		t.Fatal(err)
	}
	res, err := CheckSubmissions(root, []ProjectBrief{{ShortName: proj}}, "2026-07")
	if err != nil {
		t.Fatalf("目录缺失不应报错：%v", err)
	}
	for _, c := range res.Rows[0].Cells {
		if c.DirExists {
			t.Errorf("%s 的目录不存在，DirExists 应为 false", c.Code)
		}
		if c.Status != StatusMissing {
			t.Errorf("%s 应为缺交，实际 %s", c.Code, c.Status)
		}
	}
}
