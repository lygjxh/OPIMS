package services

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// osReadDir 抽出以便测试替换，同时避免与其它文件的同名导入冲突。
var osReadDir = os.ReadDir

// 进度管理 · 合同时效日历（Time Bar）—— 二期 2a
// 解析各项目的附件 E，按日历日倒计时并给出预警。
// 依据《进度管理实施细则 V1.1》5.1 与《需求说明书 V1.1》6A.2。
//
// 为什么优先做这块：细则 5.1 明确「违反将导致不可逆的索赔权丧失」，
// 时效漏掉直接损失索赔权，比进度看板更硬。且不依赖 .mpp 解析。

// 预警灯（剩余天数阈值，见需求 6A.2）
const (
	TBRed  = "红"
	TBAmber = "黄"
	TBGreen = "绿"
	TBDone  = "完成"
)

// TimeBarItem 一条时效记录。
type TimeBarItem struct {
	Project    string `json:"project"`
	SeqNo      string `json:"seq_no"`
	Clause     string `json:"clause"`      // 合同条款号
	Code       string `json:"code"`        // T1..T9
	CodeName   string `json:"code_name"`   // 时效事项中文名
	Event      string `json:"event"`       // 触发事件描述
	TriggerAt  string `json:"trigger_at"`  // 触发日期
	Days       int    `json:"days"`        // 时限天数（日历日）
	DueAt      string `json:"due_at"`      // 系统算：到期日
	Remaining  int    `json:"remaining"`   // 系统算：剩余天数（负数=已逾期）
	Light      string `json:"light"`       // 系统算：预警灯
	DoneAt     string `json:"done_at"`     // 实际完成日
	Status     string `json:"status"`
	Owner      string `json:"owner"`
	Note       string `json:"note"`
	SourceFile string `json:"source_file"`
}

// TimeBarResult 全部项目的时效汇总。
type TimeBarResult struct {
	Items    []TimeBarItem `json:"items"`
	Warnings []string      `json:"warnings"` // 解析告警，需在界面可见（需求 6A.5）
	Summary  TimeBarSummary `json:"summary"`
}

type TimeBarSummary struct {
	Total    int `json:"total"`
	Overdue  int `json:"overdue"`  // 已逾期
	Red      int `json:"red"`      // 剩余<7天
	Amber    int `json:"amber"`    // 7-14天
	Green    int `json:"green"`
	Done     int `json:"done"`
	Projects int `json:"projects"`
}

// tbCodeNames 时效代码中文名，与附件 E「代码说明」一致。
var tbCodeNames = map[string]string{
	"T1": "变更/延误通知（Change Notice）",
	"T2": "初步变更令（PCO）提交",
	"T3": "不可抗力通知",
	"T4": "索赔详细报告",
	"T5": "滚动 PCO 更新",
	"T6": "业主答复期届满",
	"T9": "其他合同专有时效",
}

// ScanTimeBars 扫描各项目 05.Schedule/Time Bar/ 下的附件 E，解析并计算倒计时。
// 每个项目取周期最新的一个文件。
func ScanTimeBars(projectsDir string, projects []ProjectBrief, today time.Time) *TimeBarResult {
	res := &TimeBarResult{}
	seen := map[string]bool{}

	for _, p := range projects {
		dir := filepath.Join(projectsDir, p.ShortName, filepath.FromSlash("05.Schedule/Time Bar"))
		f := latestPeriodFile(dir, p.ShortName, "TBC")
		if f == "" {
			continue
		}
		items, warns := parseTimeBarFile(filepath.Join(dir, f), p.ShortName, today)
		res.Items = append(res.Items, items...)
		res.Warnings = append(res.Warnings, warns...)
		if len(items) > 0 {
			seen[p.ShortName] = true
		}
	}

	// 最紧急的排最前：已逾期 → 剩余天数升序；已完成沉底
	sort.SliceStable(res.Items, func(i, j int) bool {
		a, b := res.Items[i], res.Items[j]
		if (a.Light == TBDone) != (b.Light == TBDone) {
			return b.Light == TBDone
		}
		return a.Remaining < b.Remaining
	})

	res.Summary.Projects = len(seen)
	for _, it := range res.Items {
		res.Summary.Total++
		switch it.Light {
		case TBDone:
			res.Summary.Done++
		case TBRed:
			res.Summary.Red++
			if it.Remaining < 0 {
				res.Summary.Overdue++
			}
		case TBAmber:
			res.Summary.Amber++
		default:
			res.Summary.Green++
		}
	}
	return res
}

// latestPeriodFile 在目录中找 {简称}-{code}-{周期}.xlsx，返回周期最大的那个文件名。
func latestPeriodFile(dir, shortName, code string) string {
	entries, err := readDirSafe(dir)
	if err != nil {
		return ""
	}
	prefix := shortName + "-" + code + "-"
	best, bestKey := "", ""
	for _, name := range entries {
		if !strings.HasSuffix(strings.ToLower(name), ".xlsx") || strings.HasPrefix(name, "~$") {
			continue
		}
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		base := strings.TrimSuffix(name, filepath.Ext(name))
		key := strings.TrimPrefix(base, prefix) // 202607 或 202607-v2
		if key > bestKey {
			best, bestKey = name, key
		}
	}
	return best
}

// parseTimeBarFile 解析单个附件 E。
// 按表头文字定位列，不写死列序——项目部调整列顺序不应导致解析错位（需求 6A.0）。
func parseTimeBarFile(path, project string, today time.Time) ([]TimeBarItem, []string) {
	var warns []string
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, []string{project + "：时效日历打不开（" + filepath.Base(path) + "）：" + err.Error()}
	}
	defer f.Close()

	sheet := pickSheet(f, "时效日历")
	rows, err := f.GetRows(sheet)
	if err != nil || len(rows) == 0 {
		return nil, []string{project + "：时效日历内容为空"}
	}

	hdr, hdrRow := findHeaderRow(rows, "合同条款号", "触发日期")
	if hdrRow < 0 {
		return nil, []string{project + "：时效日历未找到表头行（需包含「合同条款号」「触发日期」）"}
	}

	col := func(row []string, name string) string {
		i, ok := hdr[name]
		if !ok || i >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[i])
	}

	var items []TimeBarItem
	for r := hdrRow + 1; r < len(rows); r++ {
		row := rows[r]
		if isBlankRow(row) {
			continue
		}
		trigger := col(row, "触发日期")
		daysStr := col(row, "时限天数")
		if trigger == "" && daysStr == "" {
			continue // 空白模板行
		}

		it := TimeBarItem{
			Project: project, SeqNo: col(row, "序号"),
			Clause: col(row, "合同条款号"), Code: strings.ToUpper(col(row, "时效代码")),
			Event: col(row, "触发事件描述"), TriggerAt: trigger,
			DoneAt: col(row, "实际完成日"), Status: col(row, "状态"),
			Owner: col(row, "责任人"), Note: col(row, "备注"),
			SourceFile: filepath.Base(path),
		}
		it.CodeName = tbCodeNames[it.Code]

		td, ok := parseFlexDate(trigger)
		if !ok {
			warns = append(warns, project+"：第"+strconv.Itoa(r+1)+"行触发日期无法解析（"+trigger+"）")
			continue
		}
		days, err := strconv.Atoi(strings.TrimSpace(daysStr))
		if err != nil {
			warns = append(warns, project+"：第"+strconv.Itoa(r+1)+"行时限天数无法解析（"+daysStr+"）")
			continue
		}
		it.Days = days

		// 按日历日计算，不扣除周末与节假日（细则 V1.1 5.1）
		due := td.AddDate(0, 0, days)
		it.DueAt = due.Format("2006-01-02")
		it.Remaining = int(due.Sub(today).Hours() / 24)

		switch {
		case it.DoneAt != "" || it.Status == "已完成":
			it.Light = TBDone
		case it.Remaining < 7:
			it.Light = TBRed
		case it.Remaining <= 14:
			it.Light = TBAmber
		default:
			it.Light = TBGreen
		}
		items = append(items, it)
	}
	return items, warns
}

// ---- 通用小工具 ----

func readDirSafe(dir string) ([]string, error) {
	entries, err := osReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out, nil
}

// pickSheet 优先取指定名字的 sheet，找不到则用第一个。
func pickSheet(f *excelize.File, want string) string {
	for _, s := range f.GetSheetList() {
		if strings.Contains(s, want) {
			return s
		}
	}
	list := f.GetSheetList()
	if len(list) > 0 {
		return list[0]
	}
	return "Sheet1"
}

// findHeaderRow 找到同时含指定关键列的表头行，返回「列名→列号」与行号。
// 按表头文字定位，不写死行号（需求 6A.0）。
func findHeaderRow(rows [][]string, must ...string) (map[string]int, int) {
	for r, row := range rows {
		m := map[string]int{}
		for i, cell := range row {
			c := strings.TrimSpace(cell)
			if c != "" {
				m[c] = i
			}
		}
		ok := true
		for _, k := range must {
			if _, has := m[k]; !has {
				ok = false
				break
			}
		}
		if ok {
			return m, r
		}
	}
	return nil, -1
}

func isBlankRow(row []string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

// parseFlexDate 容忍几种常见写法，但不做过度猜测。
func parseFlexDate(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{"2006-01-02", "2006/01/02", "2006.01.02", "20060102"} {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, true
		}
	}
	// Excel 序列号日期
	if n, err := strconv.ParseFloat(s, 64); err == nil && n > 20000 && n < 80000 {
		if t, err := excelize.ExcelDateToTime(n, false); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
