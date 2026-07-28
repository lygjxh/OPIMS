package services

import (
	"path/filepath"
	"testing"

	"opims/database"
)

func initTestDB(t *testing.T) {
	t.Helper()
	if err := database.Init(filepath.Join(t.TempDir(), "p.db")); err != nil {
		t.Fatalf("db init: %v", err)
	}
	t.Cleanup(func() { database.Close() })
}

// 人工复核应覆盖系统判定，并保留原判用于界面显示；按时率随之重算。
func TestApplyReviewsOverridesStatus(t *testing.T) {
	initTestDB(t)

	res := &CheckResult{
		Period: "2026-07",
		Rows: []ProjectRow{{
			ShortName: "A项目",
			Cells: []SubmissionCell{
				{Code: "MPR", Required: true, Status: StatusMissing},
				{Code: "SIX", Required: true, Status: StatusSubmitted},
			},
		}},
		Summary: CheckSummary{ProjectCount: 1},
	}

	// 管理员认定 MPR 其实已交（如文件未按流程归档，已电话确认）
	if err := SaveReview(Review{
		Period: "2026-07", Project: "A项目", Code: "MPR",
		Status: StatusSubmitted, Note: "已邮件收到，未归档", Reviewer: "张三",
	}); err != nil {
		t.Fatalf("SaveReview: %v", err)
	}

	ApplyReviews(res, LoadReviews("2026-07"))

	c := res.Rows[0].Cells[0]
	if c.Status != StatusSubmitted {
		t.Errorf("复核后状态应为已交，实际 %s", c.Status)
	}
	if c.SysStatus != StatusMissing {
		t.Errorf("应保留系统原判「缺交」，实际 %q", c.SysStatus)
	}
	if !c.Reviewed || c.Note != "已邮件收到，未归档" {
		t.Errorf("复核标记/备注丢失：reviewed=%v note=%q", c.Reviewed, c.Note)
	}
	if res.Rows[0].Compliance != 100 {
		t.Errorf("两项必交均已交，按时率应为 100，实际 %.0f", res.Rows[0].Compliance)
	}

	// 留痕：需求 4.2.5 要求所有修改可追溯
	var logs int
	database.DB.QueryRow(`SELECT COUNT(*) FROM submission_review_log WHERE period='2026-07'`).Scan(&logs)
	if logs != 1 {
		t.Errorf("应留下 1 条复核日志，实际 %d", logs)
	}

	// 同一格再次复核：记录被更新而非重复插入，日志累加
	if err := SaveReview(Review{
		Period: "2026-07", Project: "A项目", Code: "MPR",
		Status: StatusLate, Note: "核实为迟交", Reviewer: "李四",
	}); err != nil {
		t.Fatal(err)
	}
	var reviews int
	database.DB.QueryRow(`SELECT COUNT(*) FROM submission_review WHERE period='2026-07'`).Scan(&reviews)
	if reviews != 1 {
		t.Errorf("同一格应只有 1 条复核记录，实际 %d", reviews)
	}
	database.DB.QueryRow(`SELECT COUNT(*) FROM submission_review_log WHERE period='2026-07'`).Scan(&logs)
	if logs != 2 {
		t.Errorf("日志应累加为 2 条，实际 %d", logs)
	}
}

func TestSnapshotAndCompliance(t *testing.T) {
	initTestDB(t)

	mk := func(period string, missing int) *CheckResult {
		cells := []SubmissionCell{{Code: "MPR", Required: true, Status: StatusSubmitted}}
		if missing > 0 {
			cells = append(cells, SubmissionCell{Code: "SIX", Required: true, Status: StatusMissing})
		} else {
			cells = append(cells, SubmissionCell{Code: "SIX", Required: true, Status: StatusSubmitted})
		}
		return &CheckResult{Period: period,
			Rows:    []ProjectRow{{ShortName: "A项目", Cells: cells}},
			Summary: CheckSummary{ProjectCount: 1}}
	}

	if _, err := TakeSnapshot(mk("2026-06", 0)); err != nil {
		t.Fatalf("TakeSnapshot: %v", err)
	}
	if _, err := TakeSnapshot(mk("2026-07", 1)); err != nil {
		t.Fatalf("TakeSnapshot: %v", err)
	}

	rep, err := Compliance("2026")
	if err != nil {
		t.Fatalf("Compliance: %v", err)
	}
	if len(rep.Periods) != 2 {
		t.Fatalf("应统计 2 个已核查周期，实际 %v", rep.Periods)
	}
	if len(rep.Rows) != 1 {
		t.Fatalf("应有 1 个项目，实际 %d", len(rep.Rows))
	}
	r := rep.Rows[0]
	// 2 期 × 2 必交项 = 4；其中 3 按时 1 缺交
	if r.Total != 4 || r.Submitted != 3 || r.Missing != 1 {
		t.Errorf("统计不对：总 %d 按时 %d 缺交 %d", r.Total, r.Submitted, r.Missing)
	}
	if got := int(r.Compliance + 0.5); got != 75 {
		t.Errorf("按时率应为 75%%，实际 %.1f%%", r.Compliance)
	}

	// 重复存同期快照应覆盖而非累加
	if _, err := TakeSnapshot(mk("2026-07", 1)); err != nil {
		t.Fatal(err)
	}
	rep2, _ := Compliance("2026")
	if rep2.Rows[0].Total != 4 {
		t.Errorf("重复存快照后总数应仍为 4，实际 %d", rep2.Rows[0].Total)
	}
}

// 没有任何快照时应返回空报表，而不是报错或把未核查月份算成缺交
func TestComplianceWithoutSnapshot(t *testing.T) {
	initTestDB(t)
	rep, err := Compliance("2026")
	if err != nil {
		t.Fatalf("无快照时不应报错：%v", err)
	}
	if len(rep.Periods) != 0 || len(rep.Rows) != 0 {
		t.Errorf("无快照应返回空报表，实际 %+v", rep)
	}
}
