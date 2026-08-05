package services

import (
	"sort"
	"strings"
	"time"
)

// 进度管理 · 时限雷达
//
// 把两个来源合成一份「今天必须动什么」的清单：
//   1. 中心登记簿 time_bar_registry —— 合同评审阶段就录进去，不等项目部报表
//   2. 各项目附件 E 扫描结果        —— 项目部按月报上来的
//
// 依据：细则 5.1（Time Bar）、6.3（视为认可防范）、附件 G 第十节（缺失型信号）。
//
// 为什么要合并而不是各看各的：同一条时效可能两边都有（中心录过、项目部又报了一次），
// 分两个页面看必然出现「这边显示还剩 3 天、那边显示已逾期」的对不上账。
// 合并后以中心登记簿为准，并标出该条是否两边都有。

// 方向。搞反了后果完全不同，见 db.go 中 time_bar_registry 的注释。
const (
	DirClaim = "claim" // 业主行为给了我方索赔机会，逾期丧失索赔权
	DirRisk  = "risk"  // 我方沉默将使自己丧失权利（6.3 视为认可 / 附件 G G10.5）
)

// 来源
const (
	SrcRegistry = "registry" // 中心登记簿
	SrcAttachE  = "attach_e" // 项目部附件 E
	SrcBoth     = "both"     // 两边都有，以登记簿为准
)

// 紧急度。比原时效预警多一档 due（今天到期或提醒日已到），
// 因为「今天必须动」和「这周要留意」在驾驶舱里是两种行为。
const (
	LvOverdue = "overdue" // 已逾期
	LvDue     = "due"     // 今天到期，或已进入强制提醒窗口
	LvUrgent  = "urgent"  // 剩余 <7 天
	LvSoon    = "soon"    // 剩余 7–14 天
	LvNormal  = "normal"  // 剩余 >14 天
	LvClosed  = "closed"  // 已办结 / 经复核不构成索赔
)

// 细则 6.3.2：「视为认可」期限到期前 3 日发出提醒。
// 登记时没填 remind_days 的，按这个兜底——漏提醒的代价是权利不可逆丧失，
// 宁可早提醒也不能不提醒。
const DefaultRiskRemindDays = 3

// ParseFlexDate 对外暴露日期解析，供 handler 校验录入的日期。
// 内部实现与附件 E 解析共用同一套宽松规则（见 timebar.go），
// 手录和表格解析必须认同一批格式，否则同一个日期两边算出不同结果。
func ParseFlexDate(s string) (time.Time, bool) { return parseFlexDate(s) }

// RadarItem 雷达上的一条。字段取两个来源的并集，缺的留空。
type RadarItem struct {
	ID          int64  `json:"id"` // 登记簿主键；纯附件 E 来源为 0
	Source      string `json:"source"`
	Project     string `json:"project"`
	Code        string `json:"code"`
	CodeName    string `json:"code_name"`
	SignalCode  string `json:"signal_code"` // 附件 G 信号码，如 G10.5
	Title       string `json:"title"`
	Clause      string `json:"clause"`
	Direction   string `json:"direction"`
	TriggerDate string `json:"trigger_date"`
	DueDays     int    `json:"due_days"`
	DueDate     string `json:"due_date"`
	RemindDays  int    `json:"remind_days"`
	Remaining   int    `json:"remaining"` // 距到期日天数，负数=已逾期
	Level       string `json:"level"`
	Status      string `json:"status"`
	Owner       string `json:"owner"` // 附件 E 才有；中心登记簿不设此字段
	Remark      string `json:"remark"`
	SourceFile  string `json:"source_file"`
}

// RadarSummary 驾驶舱红线区要的几个数。
type RadarSummary struct {
	Total     int `json:"total"`
	Overdue   int `json:"overdue"`
	Due       int `json:"due"`
	Urgent    int `json:"urgent"`
	Soon      int `json:"soon"`
	Closed    int `json:"closed"`
	Projects  int `json:"projects"`
	RiskAlert int `json:"risk_alert"` // direction=risk 且已进入提醒窗口——最危险的一类
}

// RadarResult 雷达全量结果。
type RadarResult struct {
	Items    []RadarItem  `json:"items"`
	Warnings []string     `json:"warnings"`
	Summary  RadarSummary `json:"summary"`
}

// RegistryRow 登记簿的一行，由 handler 从数据库读出后传进来。
// service 层不碰数据库，方便单测。
type RegistryRow struct {
	ID          int64
	Project     string
	Code        string
	SignalCode  string
	Title       string
	Clause      string
	Direction   string
	TriggerDate string
	DueDays     int
	DueDate     string
	RemindDays  int
	Status      string
	Remark      string
}

// BuildRadar 合并登记簿与附件 E，算出紧急度并排序。
// today 由调用方传当天零点，避免同一天内因时分不同得出不同结果。
func BuildRadar(rows []RegistryRow, scan *TimeBarResult, today time.Time) *RadarResult {
	res := &RadarResult{Items: []RadarItem{}, Warnings: []string{}}
	if scan != nil {
		res.Warnings = append(res.Warnings, scan.Warnings...)
	}

	// 登记簿优先入表，并记下去重键
	seen := map[string]int{} // key -> res.Items 下标
	for _, r := range rows {
		it := radarFromRegistry(r, today)
		seen[dedupKey(it.Project, it.Code, it.DueDate)] = len(res.Items)
		res.Items = append(res.Items, it)
	}

	// 附件 E：命中去重键的只把来源标成 both，不覆盖登记簿的内容
	if scan != nil {
		for _, s := range scan.Items {
			k := dedupKey(s.Project, s.Code, s.DueAt)
			if i, ok := seen[k]; ok {
				res.Items[i].Source = SrcBoth
				// 登记簿没有的字段用附件 E 补齐，已有的不覆盖（以登记簿为准）
				if res.Items[i].SourceFile == "" {
					res.Items[i].SourceFile = s.SourceFile
				}
				if res.Items[i].Owner == "" {
					res.Items[i].Owner = s.Owner
				}
				continue
			}
			res.Items = append(res.Items, radarFromScan(s, today))
		}
	}

	projects := map[string]bool{}
	for i := range res.Items {
		it := &res.Items[i]
		projects[it.Project] = true
		res.Summary.Total++
		switch it.Level {
		case LvOverdue:
			res.Summary.Overdue++
		case LvDue:
			res.Summary.Due++
		case LvUrgent:
			res.Summary.Urgent++
		case LvSoon:
			res.Summary.Soon++
		case LvClosed:
			res.Summary.Closed++
		}
		if it.Direction == DirRisk && it.Level != LvClosed &&
			(it.Level == LvOverdue || it.Level == LvDue) {
			res.Summary.RiskAlert++
		}
	}
	res.Summary.Projects = len(projects)

	sortRadar(res.Items)
	return res
}

func radarFromRegistry(r RegistryRow, today time.Time) RadarItem {
	it := RadarItem{
		ID: r.ID, Source: SrcRegistry, Project: r.Project,
		Code: r.Code, CodeName: tbCodeNames[r.Code], SignalCode: r.SignalCode,
		Title: r.Title, Clause: r.Clause, Direction: r.Direction,
		TriggerDate: r.TriggerDate, DueDays: r.DueDays, DueDate: r.DueDate,
		RemindDays: r.RemindDays, Status: r.Status, Remark: r.Remark,
	}
	if it.Direction == "" {
		it.Direction = DirClaim
	}
	// risk 类没填提醒天数就兜底 3 日（细则 6.3.2）
	if it.Direction == DirRisk && it.RemindDays <= 0 {
		it.RemindDays = DefaultRiskRemindDays
	}
	it.Remaining, it.Level = levelOf(it.DueDate, it.RemindDays, it.Status, today)
	return it
}

func radarFromScan(s TimeBarItem, today time.Time) RadarItem {
	it := RadarItem{
		Source: SrcAttachE, Project: s.Project,
		Code: s.Code, CodeName: s.CodeName,
		Title: s.Event, Clause: s.Clause, Direction: DirClaim,
		TriggerDate: s.TriggerAt, DueDays: s.Days, DueDate: s.DueAt,
		Status: s.Status, Owner: s.Owner, Remark: s.Note, SourceFile: s.SourceFile,
	}
	if s.DoneAt != "" || s.Status == "已完成" {
		it.Status = "done"
	}
	it.Remaining, it.Level = levelOf(it.DueDate, 0, it.Status, today)
	return it
}

// levelOf 由到期日和状态算剩余天数与紧急度。
// 到期日解析不了时返回 normal，不让一条坏数据把整页拖垮——
// 解析失败会另行进 warnings，界面上看得到。
func levelOf(dueDate string, remindDays int, status string, today time.Time) (int, string) {
	if isClosedStatus(status) {
		return 0, LvClosed
	}
	due, ok := parseFlexDate(dueDate)
	if !ok {
		return 0, LvNormal
	}
	remaining := int(due.Sub(today).Hours() / 24)
	switch {
	case remaining < 0:
		return remaining, LvOverdue
	case remaining == 0, remindDays > 0 && remaining <= remindDays:
		return remaining, LvDue
	case remaining < 7:
		return remaining, LvUrgent
	case remaining <= 14:
		return remaining, LvSoon
	default:
		return remaining, LvNormal
	}
}

func isClosedStatus(s string) bool {
	switch strings.TrimSpace(s) {
	case "done", "not_claim", "已完成", "已办结":
		return true
	}
	return false
}

func dedupKey(project, code, due string) string {
	return strings.TrimSpace(project) + "|" + strings.TrimSpace(code) + "|" + strings.TrimSpace(due)
}

// sortRadar 排序即优先级：先按紧急度，同档内 risk 排在 claim 前面
// （risk 是我方要丧失权利，比错过一次索赔机会更不可挽回），再按剩余天数升序。
func sortRadar(items []RadarItem) {
	rank := map[string]int{
		LvOverdue: 0, LvDue: 1, LvUrgent: 2, LvSoon: 3, LvNormal: 4, LvClosed: 5,
	}
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if rank[a.Level] != rank[b.Level] {
			return rank[a.Level] < rank[b.Level]
		}
		if a.Direction != b.Direction {
			return a.Direction == DirRisk
		}
		if a.Remaining != b.Remaining {
			return a.Remaining < b.Remaining
		}
		return a.Project < b.Project
	})
}
