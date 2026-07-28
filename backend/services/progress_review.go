package services

import (
	"fmt"
	"sort"
	"time"

	"opims/database"
)

// 进度管理 · 人工复核、快照与合规率（一期 1b）
// 依据《OPIMS 进度管理模块 需求说明书 V1.1》4.2.5、4.2.6、4.5.1、4.5.2。

// Review 一条人工复核记录。
type Review struct {
	Period   string `json:"period"`
	Project  string `json:"project"`
	Code     string `json:"code"`
	Status   string `json:"status"` // 覆盖后的状态；空串表示只加备注、不改判定
	Note     string `json:"note"`
	Reviewer string `json:"reviewer"`
}

// LoadReviews 读取某期的全部复核记录，键为 "项目\x00代码"。
func LoadReviews(period string) map[string]Review {
	out := map[string]Review{}
	rows, err := database.DB.Query(
		`SELECT project, code, status, note, reviewer FROM submission_review WHERE period=?`, period)
	if err != nil {
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var r Review
		r.Period = period
		if err := rows.Scan(&r.Project, &r.Code, &r.Status, &r.Note, &r.Reviewer); err != nil {
			continue
		}
		out[r.Project+"\x00"+r.Code] = r
	}
	return out
}

// ApplyReviews 把人工复核覆盖到扫描结果上，并重算按时率与汇总。
// 系统判定保留在 SysStatus 中，便于界面显示「原判 X → 现为 Y」。
func ApplyReviews(res *CheckResult, reviews map[string]Review) {
	for i := range res.Rows {
		row := &res.Rows[i]
		var reqTotal, reqOnTime int
		for j := range row.Cells {
			c := &row.Cells[j]
			if rv, ok := reviews[row.ShortName+"\x00"+c.Code]; ok {
				c.Note = rv.Note
				c.Reviewer = rv.Reviewer
				if rv.Status != "" && rv.Status != c.Status {
					c.SysStatus = c.Status
					c.Status = rv.Status
					c.Reviewed = true
				} else if rv.Note != "" {
					c.Reviewed = true
				}
			}
			if c.Required {
				reqTotal++
				if c.Status == StatusSubmitted {
					reqOnTime++
				}
			}
		}
		row.Compliance = 0
		if reqTotal > 0 {
			row.Compliance = float64(reqOnTime) / float64(reqTotal) * 100
		}
	}
	res.Summary = summarizeCheck(res.Rows, res.Summary.ProjectCount)
}

// SaveReview 写入/更新复核，并记录留痕。
func SaveReview(r Review) error {
	if r.Period == "" || r.Project == "" || r.Code == "" {
		return fmt.Errorf("周期、项目、文件类型均不能为空")
	}
	if r.Status != "" &&
		r.Status != StatusSubmitted && r.Status != StatusLate && r.Status != StatusMissing {
		return fmt.Errorf("状态只能是 %s/%s/%s 或留空", StatusSubmitted, StatusLate, StatusMissing)
	}

	var old string
	database.DB.QueryRow(
		`SELECT status FROM submission_review WHERE period=? AND project=? AND code=?`,
		r.Period, r.Project, r.Code).Scan(&old)

	if _, err := database.DB.Exec(`
		INSERT INTO submission_review (period, project, code, status, note, reviewer)
		VALUES (?,?,?,?,?,?)
		ON CONFLICT(period, project, code) DO UPDATE SET
			status=excluded.status, note=excluded.note,
			reviewer=excluded.reviewer, updated_at=CURRENT_TIMESTAMP`,
		r.Period, r.Project, r.Code, r.Status, r.Note, r.Reviewer); err != nil {
		return err
	}

	// 留痕：需求 4.2.5 要求所有修改可追溯
	_, err := database.DB.Exec(`
		INSERT INTO submission_review_log (period, project, code, old_status, new_status, note, reviewer)
		VALUES (?,?,?,?,?,?,?)`,
		r.Period, r.Project, r.Code, old, r.Status, r.Note, r.Reviewer)
	return err
}

// TakeSnapshot 将本期核查结果存为快照（覆盖同期旧快照）。
// 快照是考核依据：文件事后可能被删改，快照锁定核查当时的事实。
func TakeSnapshot(res *CheckResult) (int, error) {
	tx, err := database.DB.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM submission_snapshot WHERE period=?`, res.Period); err != nil {
		return 0, err
	}
	stmt, err := tx.Prepare(`
		INSERT INTO submission_snapshot
			(period, project, code, required, status, file_name, submit_at, deadline, reviewed, note)
		VALUES (?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	n := 0
	for _, row := range res.Rows {
		for _, c := range row.Cells {
			req := 0
			if c.Required {
				req = 1
			}
			rev := 0
			if c.Reviewed {
				rev = 1
			}
			if _, err := stmt.Exec(res.Period, row.ShortName, c.Code, req,
				c.Status, c.FileName, c.SubmitAt, c.Deadline, rev, c.Note); err != nil {
				return 0, err
			}
			n++
		}
	}
	return n, tx.Commit()
}

// ComplianceRow 某项目在某年度的报送合规统计。
type ComplianceRow struct {
	Project    string  `json:"project"`
	Total      int     `json:"total"`      // 必交项总数
	Submitted  int     `json:"submitted"`  // 按时
	Late       int     `json:"late"`
	Missing    int     `json:"missing"`
	Compliance float64 `json:"compliance"` // 按时率
}

// ComplianceReport 年度报送合规汇总。
type ComplianceReport struct {
	Year    string          `json:"year"`
	Periods []string        `json:"periods"` // 已存快照的周期
	Rows    []ComplianceRow `json:"rows"`
	Overall ComplianceRow   `json:"overall"`
}

// Compliance 基于快照统计年度合规率。
// 只统计已存快照的周期——没核查过的月份不应算作缺交，否则会冤枉项目部。
func Compliance(year string) (*ComplianceReport, error) {
	if len(year) != 4 {
		return nil, fmt.Errorf("年度格式应为 YYYY，收到 %q", year)
	}
	rep := &ComplianceReport{Year: year}

	pr, err := database.DB.Query(
		`SELECT DISTINCT period FROM submission_snapshot WHERE period LIKE ? ORDER BY period`, year+"-%")
	if err != nil {
		return nil, err
	}
	for pr.Next() {
		var p string
		if pr.Scan(&p) == nil {
			rep.Periods = append(rep.Periods, p)
		}
	}
	pr.Close()
	if len(rep.Periods) == 0 {
		return rep, nil // 尚无快照，返回空报表而非报错
	}

	rows, err := database.DB.Query(`
		SELECT project,
		       COUNT(*),
		       SUM(CASE WHEN status=? THEN 1 ELSE 0 END),
		       SUM(CASE WHEN status=? THEN 1 ELSE 0 END),
		       SUM(CASE WHEN status=? THEN 1 ELSE 0 END)
		FROM submission_snapshot
		WHERE period LIKE ? AND required=1
		GROUP BY project`,
		StatusSubmitted, StatusLate, StatusMissing, year+"-%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var r ComplianceRow
		if err := rows.Scan(&r.Project, &r.Total, &r.Submitted, &r.Late, &r.Missing); err != nil {
			continue
		}
		if r.Total > 0 {
			r.Compliance = float64(r.Submitted) / float64(r.Total) * 100
		}
		rep.Rows = append(rep.Rows, r)
		rep.Overall.Total += r.Total
		rep.Overall.Submitted += r.Submitted
		rep.Overall.Late += r.Late
		rep.Overall.Missing += r.Missing
	}
	if rep.Overall.Total > 0 {
		rep.Overall.Compliance = float64(rep.Overall.Submitted) / float64(rep.Overall.Total) * 100
	}
	// 合规率低的排前面，便于一眼看到需要督办的项目
	sort.Slice(rep.Rows, func(i, j int) bool { return rep.Rows[i].Compliance < rep.Rows[j].Compliance })
	return rep, nil
}

// CurrentYear 返回当前年份字符串，供 handler 缺省取值。
func CurrentYear() string { return time.Now().Format("2006") }
