package services

import (
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// 进度管理 · 工期索赔（EOT）跟踪台账 —— 二期 2b
// 解析各项目附件 D，跨项目汇总索赔状态与「请求 vs 批准」。
// 依据《进度管理实施细则 V1.1》5.4、《需求说明书 V1.1》6A.3。

// eotCategories 索赔机会点分类，与附件 D 一致。
var eotCategories = map[string]string{
	"C1": "业主的行为", "C2": "业主代表的行为", "C3": "业主设计变更",
	"C4": "合同文件缺陷", "C5": "施工条件变化", "C6": "所在国政策法规变更",
	"C7": "不可抗力事件", "C8": "不可预见因素", "C9": "业主指定分包商违约",
}

// EOTItem 一条索赔记录。
type EOTItem struct {
	Project    string  `json:"project"`
	SeqNo      string  `json:"seq_no"`
	Subject    string  `json:"subject"`      // 索赔事项
	Category   string  `json:"category"`     // C1..C9
	CatName    string  `json:"cat_name"`
	Clause     string  `json:"clause"`       // 合同依据条款
	FoundAt    string  `json:"found_at"`     // 发现/发生日期
	CNAt       string  `json:"cn_at"`        // CN 发出日
	CNDueAt    string  `json:"cn_due_at"`    // 系统算：发现日 + 10 日历日
	PCODueAt   string  `json:"pco_due_at"`   // 系统算：CN 发出日 + 21 日历日
	PCOAt      string  `json:"pco_at"`       // PCO 实交日
	Days       float64 `json:"days"`         // 累计影响天数
	Amount     float64 `json:"amount"`       // 累计索赔金额
	Status     string  `json:"status"`
	Owner      string  `json:"owner"`
	Overdue    string  `json:"overdue"`      // 系统算：逾期说明（空=未逾期）
	SourceFile string  `json:"source_file"`
}

// EOTResult 汇总结果。
type EOTResult struct {
	Items    []EOTItem  `json:"items"`
	Warnings []string   `json:"warnings"`
	Summary  EOTSummary `json:"summary"`
	ByStatus []NameCount `json:"by_status"`
	ByCat    []NameCount `json:"by_cat"`
}

type NameCount struct {
	Name  string  `json:"name"`
	Count int     `json:"count"`
	Days  float64 `json:"days"`
	Amount float64 `json:"amount"`
}

type EOTSummary struct {
	Total       int     `json:"total"`
	Projects    int     `json:"projects"`
	OverdueCN   int     `json:"overdue_cn"`   // CN 已逾期未发
	OverduePCO  int     `json:"overdue_pco"`  // PCO 已逾期未交
	TotalDays   float64 `json:"total_days"`   // 累计请求天数
	TotalAmount float64 `json:"total_amount"` // 累计请求金额
	ClosedCount int     `json:"closed_count"`
}

// EOT 时限（日历日），与细则 5.1 一致
const (
	cnDueDays  = 10
	pcoDueDays = 21
)

// ScanEOT 扫描各项目 05.Schedule/EOT/ 下的附件 D。
func ScanEOT(projectsDir string, projects []ProjectBrief, today time.Time) *EOTResult {
	res := &EOTResult{}
	seen := map[string]bool{}

	for _, p := range projects {
		dir := filepath.Join(projectsDir, p.ShortName, filepath.FromSlash("05.Schedule/EOT"))
		f := latestPeriodFile(dir, p.ShortName, "EOT")
		if f == "" {
			continue
		}
		items, warns := parseEOTFile(filepath.Join(dir, f), p.ShortName, today)
		res.Items = append(res.Items, items...)
		res.Warnings = append(res.Warnings, warns...)
		if len(items) > 0 {
			seen[p.ShortName] = true
		}
	}

	// 逾期的排最前，其次按索赔金额降序——先看该催的，再看金额大的
	sort.SliceStable(res.Items, func(i, j int) bool {
		a, b := res.Items[i], res.Items[j]
		if (a.Overdue != "") != (b.Overdue != "") {
			return a.Overdue != ""
		}
		return a.Amount > b.Amount
	})

	statusAgg := map[string]*NameCount{}
	catAgg := map[string]*NameCount{}
	res.Summary.Projects = len(seen)
	for _, it := range res.Items {
		res.Summary.Total++
		res.Summary.TotalDays += it.Days
		res.Summary.TotalAmount += it.Amount
		if strings.Contains(it.Overdue, "CN") {
			res.Summary.OverdueCN++
		}
		if strings.Contains(it.Overdue, "PCO") {
			res.Summary.OverduePCO++
		}
		if strings.Contains(it.Status, "关闭") {
			res.Summary.ClosedCount++
		}
		addAgg(statusAgg, orDash(it.Status), it)
		addAgg(catAgg, orDash(it.CatName), it)
	}
	res.ByStatus = sortAgg(statusAgg)
	res.ByCat = sortAgg(catAgg)
	return res
}

func addAgg(m map[string]*NameCount, key string, it EOTItem) {
	c, ok := m[key]
	if !ok {
		c = &NameCount{Name: key}
		m[key] = c
	}
	c.Count++
	c.Days += it.Days
	c.Amount += it.Amount
}

func sortAgg(m map[string]*NameCount) []NameCount {
	out := make([]NameCount, 0, len(m))
	for _, v := range m {
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Count > out[j].Count })
	return out
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "未填写"
	}
	return s
}

// parseEOTFile 解析单个附件 D 的主台账。
func parseEOTFile(path, project string, today time.Time) ([]EOTItem, []string) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, []string{project + "：索赔台账打不开（" + filepath.Base(path) + "）：" + err.Error()}
	}
	defer f.Close()

	var warns []string
	// 主台账可能在任意 sheet，逐个找含关键列的表头
	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err != nil || len(rows) == 0 {
			continue
		}
		hdr, hdrRow := findHeaderRow(rows, "索赔事项", "类别代码")
		if hdrRow < 0 {
			continue
		}
		col := func(row []string, name string) string {
			i, ok := hdr[name]
			if !ok || i >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[i])
		}

		var items []EOTItem
		for r := hdrRow + 1; r < len(rows); r++ {
			row := rows[r]
			if isBlankRow(row) {
				continue
			}
			subject := col(row, "索赔事项")
			if subject == "" {
				continue // 空白模板行
			}
			it := EOTItem{
				Project: project, SeqNo: col(row, "序号"), Subject: subject,
				Category: strings.ToUpper(col(row, "类别代码")),
				Clause:   col(row, "合同依据条款"),
				FoundAt:  col(row, "发现/发生日期"),
				CNAt:     col(row, "CN 发出日"),
				PCOAt:    col(row, "PCO 实交日"),
				Status:   col(row, "当前状态"), Owner: col(row, "责任人"),
				SourceFile: filepath.Base(path),
			}
			if it.CNAt == "" {
				it.CNAt = col(row, "CN发出日") // 容忍中间无空格的写法
			}
			if it.PCOAt == "" {
				it.PCOAt = col(row, "PCO实交日")
			}
			it.CatName = eotCategories[it.Category]
			it.Days = parseNum(col(row, "累计影响天数"))
			it.Amount = parseNum(col(row, "累计索赔金额"))

			// 时限按日历日计算，与细则 5.1 及时效日历同一口径
			if d, ok := parseFlexDate(it.FoundAt); ok {
				due := d.AddDate(0, 0, cnDueDays)
				it.CNDueAt = due.Format("2006-01-02")
				if it.CNAt == "" && today.After(due) {
					it.Overdue = "CN 已逾期 " + strconv.Itoa(int(today.Sub(due).Hours()/24)) + " 天"
				}
			}
			if d, ok := parseFlexDate(it.CNAt); ok {
				due := d.AddDate(0, 0, pcoDueDays)
				it.PCODueAt = due.Format("2006-01-02")
				if it.PCOAt == "" && today.After(due) {
					if it.Overdue != "" {
						it.Overdue += "；"
					}
					it.Overdue += "PCO 已逾期 " + strconv.Itoa(int(today.Sub(due).Hours()/24)) + " 天"
				}
			}
			items = append(items, it)
		}
		return items, warns
	}
	return nil, []string{project + "：索赔台账未找到主台账表头（需包含「索赔事项」「类别代码」）"}
}

// parseNum 去掉千分位、币种符号与空白后转数字；失败返回 0。
func parseNum(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	r := strings.NewReplacer(",", "", "，", "", " ", "", "¥", "", "$", "", "元", "", "天", "")
	v, err := strconv.ParseFloat(r.Replace(s), 64)
	if err != nil {
		return 0
	}
	return v
}
