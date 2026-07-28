package services

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"opims/data"
)

// 进度管理 · 报送核查（一期 1a：只读扫描 + 状态判定）
// 依据《OPIMS 进度管理模块 需求说明书 V1.1》4.2.2–4.2.4、第 6 章。
//
// 一期范围说明：
//   - 看板以「月度周期」为单位，包含月度项与季度项；周报（WKR）按周计，
//     一个月含 4–5 周，无法在单元格内表达，故一期只统计其文件数，不做交/未交判定。
//   - L1–L4 进度计划为事件驱动，不判定交/未交，仅展示各级当前最新版本。

// 报送状态
const (
	StatusSubmitted = "已交"
	StatusLate      = "迟交"
	StatusMissing   = "缺交"
)

// SubmissionCell 看板中的一个单元格：某项目某清单项在本期的状态。
type SubmissionCell struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Required  bool   `json:"required"`
	Period    string `json:"period"`     // 该项适用的周期串，如 202607 或 2026Q3
	Deadline  string `json:"deadline"`   // 计算出的截止日 YYYY-MM-DD
	Status    string `json:"status"`     // 已交/迟交/缺交
	FileName  string `json:"file_name"`  // 命中的文件名
	FilePath  string `json:"file_path"`  // 绝对路径，供后续「打开原文件」
	SubmitAt  string `json:"submit_at"`  // 文件修改时间 YYYY-MM-DD
	DirExists bool   `json:"dir_exists"` // 存放目录是否存在

	// 以下由人工复核填充（1b）
	Reviewed  bool   `json:"reviewed"`   // 是否经人工复核
	SysStatus string `json:"sys_status"` // 被覆盖前的系统判定，便于显示「原判 X」
	Note      string `json:"note"`       // 复核备注
	Reviewer  string `json:"reviewer"`
}

// PlanInfo 某个层级进度计划的当前最新版本。
type PlanInfo struct {
	Level    string `json:"level"`
	Version  int    `json:"version"`
	FileName string `json:"file_name"`
	FilePath string `json:"file_path"`
}

// ProjectRow 看板中的一行：一个项目。
type ProjectRow struct {
	ShortName string            `json:"short_name"`
	Country   string            `json:"country"`
	Cells     []SubmissionCell  `json:"cells"`
	Plans     []PlanInfo        `json:"plans"`      // L1–L4 最新版本
	WeekCount int               `json:"week_count"` // 本月周报文件数（不判定状态）
	Compliance float64          `json:"compliance"` // 本期必交项的按时率 0–100
}

// UnmatchedFile 命名不合规、无法自动识别的文件（列入「待人工识别」）。
type UnmatchedFile struct {
	Project  string `json:"project"`
	Folder   string `json:"folder"`
	FileName string `json:"file_name"`
	FilePath string `json:"file_path"`
}

// CheckResult 一次核查的完整结果。
type CheckResult struct {
	Period    string           `json:"period"`     // 月度周期 YYYY-MM
	Checklist []data.ChecklistItem `json:"checklist"`
	Rows      []ProjectRow     `json:"rows"`
	Unmatched []UnmatchedFile  `json:"unmatched"`
	Summary   CheckSummary     `json:"summary"`
}

// CheckSummary 本期汇总。
type CheckSummary struct {
	ProjectCount int     `json:"project_count"`
	Submitted    int     `json:"submitted"`
	Late         int     `json:"late"`
	Missing      int     `json:"missing"`
	Compliance   float64 `json:"compliance"` // 整体按时率
}

// ErrProjectsDirMissing 项目文件根目录不存在。
var ErrProjectsDirMissing = fmt.Errorf("projects directory not found")

// ParsePeriod 解析 YYYY-MM 月度周期。
func ParsePeriod(s string) (time.Time, error) {
	t, err := time.Parse("2006-01", strings.TrimSpace(s))
	if err != nil {
		return time.Time{}, fmt.Errorf("周期格式应为 YYYY-MM，收到 %q", s)
	}
	return t, nil
}

// Deadline 按「基准月 + 日」规则算出实际截止日。
// 周报（BaseWeekly）在月度看板中不适用，返回零值。
func Deadline(item data.ChecklistItem, period time.Time) time.Time {
	switch item.Base {
	case data.BaseSameMonth:
		return clampDay(period.Year(), period.Month(), item.Day)
	case data.BaseNextMonth:
		n := period.AddDate(0, 1, 0)
		return clampDay(n.Year(), n.Month(), item.Day)
	case data.BaseQuarterEnd:
		// 季度末月：Q1→3月 Q2→6月 Q3→9月 Q4→12月
		q := (int(period.Month())-1)/3 + 1
		return clampDay(period.Year(), time.Month(q*3), item.Day)
	}
	return time.Time{}
}

// clampDay 处理「30 日」落在 2 月这类情况，取该月最后一天。
func clampDay(year int, month time.Month, day int) time.Time {
	last := time.Date(year, month+1, 0, 0, 0, 0, 0, time.Local).Day()
	if day > last {
		day = last
	}
	return time.Date(year, month, day, 23, 59, 59, 0, time.Local)
}

// PeriodString 返回该清单项在此月度周期下对应的周期串。
// 月度项 → 202607；季度项 → 2026Q3。
func PeriodString(item data.ChecklistItem, period time.Time) string {
	if item.Cycle == data.CycleQuarter {
		q := (int(period.Month())-1)/3 + 1
		return fmt.Sprintf("%dQ%d", period.Year(), q)
	}
	return period.Format("200601")
}

// fileRe 匹配 {项目简称}-{代码}-{周期}[-v{n}].{ext}
// 简称中可能含「-」，故以代码与周期为锚点从右侧解析。
var fileRe = regexp.MustCompile(`^(?P<name>.+)-(?P<code>[A-Z0-9]{2,4})-(?P<period>\d{6}|\d{4}Q[1-4]|\d{4}W\d{1,2})(?:-v(?P<ver>\d+))?$`)

// planRe 匹配 {项目简称}-{L1..L4}-v{n}.{ext}
var planRe = regexp.MustCompile(`^(?P<name>.+)-(?P<level>L[1-4])-v(?P<ver>\d+)$`)

// CheckSubmissions 扫描指定项目在某月度周期的报送情况。
// projects 为参与核查的项目（调用方已按「在建」过滤）。
func CheckSubmissions(projectsDir string, projects []ProjectBrief, periodStr string) (*CheckResult, error) {
	period, err := ParsePeriod(periodStr)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(projectsDir); err != nil {
		return nil, ErrProjectsDirMissing
	}

	res := &CheckResult{
		Period:    period.Format("2006-01"),
		Checklist: data.DefaultChecklist,
	}
	matched := map[string]bool{} // 已被识别的文件绝对路径

	for _, p := range projects {
		projDir := filepath.Join(projectsDir, p.ShortName)
		row := ProjectRow{ShortName: p.ShortName, Country: p.Country}

		var reqTotal, reqOnTime int
		for _, item := range data.DefaultChecklist {
			if item.Cycle == data.CycleWeek {
				continue // 周报不进月度看板，见文件头说明
			}
			cell := buildCell(projDir, p.ShortName, item, period, matched)
			if item.Required {
				reqTotal++
				if cell.Status == StatusSubmitted {
					reqOnTime++
				}
			}
			row.Cells = append(row.Cells, cell)
		}
		if reqTotal > 0 {
			row.Compliance = float64(reqOnTime) / float64(reqTotal) * 100
		}

		row.Plans = scanPlans(projDir, p.ShortName, matched)
		row.WeekCount = countWeeklyFiles(projDir, period, matched)
		res.Rows = append(res.Rows, row)
	}

	res.Unmatched = collectUnmatched(projectsDir, projects, matched)
	res.Summary = summarizeCheck(res.Rows, len(projects))
	return res, nil
}

// ProjectBrief 参与核查的项目最小信息（由 handler 从 projects 表取）。
type ProjectBrief struct {
	ShortName string
	Country   string
}

// buildCell 判定某项目某清单项在本期的状态。
func buildCell(projDir, shortName string, item data.ChecklistItem, period time.Time, matched map[string]bool) SubmissionCell {
	ps := PeriodString(item, period)
	dl := Deadline(item, period)
	cell := SubmissionCell{
		Code: item.Code, Name: item.Name, Required: item.Required,
		Period: ps, Deadline: dl.Format("2006-01-02"), Status: StatusMissing,
	}

	dir := filepath.Join(projDir, filepath.FromSlash(item.Folder))
	entries, err := os.ReadDir(dir)
	if err != nil {
		// 目录不存在：记为未建目录 + 缺交，不报错、不中断（需求 7.5）
		return cell
	}
	cell.DirExists = true

	want := shortName + "-" + item.Code + "-" + ps
	var best os.DirEntry
	var bestVer = -1
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		base := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		m := fileRe.FindStringSubmatch(base)
		if m == nil {
			continue
		}
		if base != want && !strings.HasPrefix(base, want+"-v") {
			continue
		}
		// 命名合规即视为「已识别」，被新版本取代的旧版也算，
		// 否则旧版会被误列入「待人工识别」。
		matched[filepath.Join(dir, e.Name())] = true

		ver := 0
		if v := m[fileRe.SubexpIndex("ver")]; v != "" {
			ver, _ = strconv.Atoi(v)
		}
		if ver > bestVer { // 同期多版本取版本号最大者
			bestVer, best = ver, e
		}
	}
	if best == nil {
		return cell
	}

	full := filepath.Join(dir, best.Name())
	cell.FileName = best.Name()
	cell.FilePath = full

	// 提交时间用文件修改时间。注意：云盘同步可能重写 mtime，
	// 判定为迟交时管理员可在复核中修正（1b 提供）。
	if info, err := best.Info(); err == nil {
		cell.SubmitAt = info.ModTime().Format("2006-01-02")
		if info.ModTime().After(dl) {
			cell.Status = StatusLate
		} else {
			cell.Status = StatusSubmitted
		}
	} else {
		cell.Status = StatusSubmitted
	}
	return cell
}

// scanPlans 取每个层级版本号最大的进度计划。
func scanPlans(projDir, shortName string, matched map[string]bool) []PlanInfo {
	dir := filepath.Join(projDir, filepath.FromSlash(data.PlanFolder))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	best := map[string]PlanInfo{}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		base := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		m := planRe.FindStringSubmatch(base)
		if m == nil || m[planRe.SubexpIndex("name")] != shortName {
			continue
		}
		lv := m[planRe.SubexpIndex("level")]
		ver, _ := strconv.Atoi(m[planRe.SubexpIndex("ver")])
		full := filepath.Join(dir, e.Name())
		matched[full] = true
		if cur, ok := best[lv]; !ok || ver > cur.Version {
			best[lv] = PlanInfo{Level: lv, Version: ver, FileName: e.Name(), FilePath: full}
		}
	}
	var out []PlanInfo
	for _, lv := range data.PlanLevels {
		if p, ok := best[lv]; ok {
			out = append(out, p)
		}
	}
	return out
}

// countWeeklyFiles 统计本月的周报文件数（一期不判定状态）。
func countWeeklyFiles(projDir string, period time.Time, matched map[string]bool) int {
	item, ok := data.FindChecklistItem("WKR")
	if !ok {
		return 0
	}
	dir := filepath.Join(projDir, filepath.FromSlash(item.Folder))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		base := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		m := fileRe.FindStringSubmatch(base)
		if m == nil || m[fileRe.SubexpIndex("code")] != "WKR" {
			continue
		}
		// 周期形如 2026W30，判断该周是否落在本月
		if wk := m[fileRe.SubexpIndex("period")]; strings.Contains(wk, "W") {
			if t, ok := weekStart(wk); ok && t.Year() == period.Year() && t.Month() == period.Month() {
				matched[filepath.Join(dir, e.Name())] = true
				n++
			}
		}
	}
	return n
}

// weekStart 由 2026W30 求该 ISO 周的周一。
func weekStart(s string) (time.Time, bool) {
	parts := strings.SplitN(s, "W", 2)
	if len(parts) != 2 {
		return time.Time{}, false
	}
	y, err1 := strconv.Atoi(parts[0])
	w, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || w < 1 || w > 53 {
		return time.Time{}, false
	}
	// ISO：1月4日必属第1周
	jan4 := time.Date(y, 1, 4, 0, 0, 0, 0, time.Local)
	off := (int(jan4.Weekday()) + 6) % 7 // 周一=0
	week1Mon := jan4.AddDate(0, 0, -off)
	return week1Mon.AddDate(0, 0, (w-1)*7), true
}

// isWellNamed 判断文件名是否符合命名规范（不论属于哪个周期）。
// 「待人工识别」只应包含命名不合规的文件；命名规范但属于其它周期的文件
// 不算问题，不能列进去误导管理员。
func isWellNamed(fileName, shortName string) bool {
	base := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	if m := planRe.FindStringSubmatch(base); m != nil {
		return m[planRe.SubexpIndex("name")] == shortName
	}
	if m := fileRe.FindStringSubmatch(base); m != nil {
		if m[fileRe.SubexpIndex("name")] != shortName {
			return false
		}
		// 代码须在清单内，避免任意大写串被当成合规
		_, ok := data.FindChecklistItem(m[fileRe.SubexpIndex("code")])
		return ok
	}
	return false
}

// collectUnmatched 收集各项目相关目录下命名不合规的文件。
func collectUnmatched(projectsDir string, projects []ProjectBrief, matched map[string]bool) []UnmatchedFile {
	folders := map[string]bool{}
	for _, it := range data.DefaultChecklist {
		folders[it.Folder] = true
	}
	folders[data.PlanFolder] = true

	var out []UnmatchedFile
	for _, p := range projects {
		for f := range folders {
			dir := filepath.Join(projectsDir, p.ShortName, filepath.FromSlash(f))
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				full := filepath.Join(dir, e.Name())
				// 本期已命中的、以及命名规范但属于其它周期的，都不算「待人工识别」
				if matched[full] || isWellNamed(e.Name(), p.ShortName) {
					continue
				}
				out = append(out, UnmatchedFile{
					Project: p.ShortName, Folder: f, FileName: e.Name(), FilePath: full,
				})
			}
		}
	}
	return out
}

func summarizeCheck(rows []ProjectRow, projectCount int) CheckSummary {
	s := CheckSummary{ProjectCount: projectCount}
	total, onTime := 0, 0
	for _, r := range rows {
		for _, c := range r.Cells {
			if !c.Required {
				continue
			}
			total++
			switch c.Status {
			case StatusSubmitted:
				s.Submitted++
				onTime++
			case StatusLate:
				s.Late++
			default:
				s.Missing++
			}
		}
	}
	if total > 0 {
		s.Compliance = float64(onTime) / float64(total) * 100
	}
	return s
}
