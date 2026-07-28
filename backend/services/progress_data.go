package services

import (
	"path/filepath"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// 进度管理 · 进度指标与风险灯 —— 二期 2c
// 解析各项目附件 B-1（月度进度数据表），按细则 4.1 自动判定风险灯。
// 依据《需求说明书 V1.1》6A.1。
//
// 关键设计：SPI 由系统统一按「累计实际产值 ÷ 累计计划产值」计算，项目部不填。
// 各项目自行计算口径不同，算出的数字无法横向比较，而中心的价值正在于横向比较。

// 风险灯（细则 4.1）
const (
	LightGreen  = "绿"
	LightYellow = "黄"
	LightOrange = "橙"
	LightRed    = "红"
)

// 细则 4.1 对应的响应要求——看到灯要知道该做什么，否则灯没有意义
var lightActions = map[string]string{
	LightGreen:  "正常监控，周报记录",
	LightYellow: "项目部内部讨论，10 日内出纠偏方案",
	LightOrange: "项目部出正式纠偏计划，报中心备案",
	LightRed:    "中心介入，组织专题分析会",
}

// ProgressData 一个项目某期的进度指标。
type ProgressData struct {
	Project     string  `json:"project"`
	Period      string  `json:"period"`
	DataDate    string  `json:"data_date"`
	Reporter    string  `json:"reporter"`
	LagDays     int     `json:"lag_days"`      // 关键路径滞后天数
	MilestoneRisk bool  `json:"milestone_risk"` // 付款里程碑预计滞后
	HasLDs      bool    `json:"has_lds"`       // 已产生误期违约金
	SelfLight   string  `json:"self_light"`    // 项目自评
	Currency    string  `json:"currency"`
	PlanPeriod  float64 `json:"plan_period"`
	ActPeriod   float64 `json:"act_period"`
	PlanCum     float64 `json:"plan_cum"`
	ActCum      float64 `json:"act_cum"`

	// 以下由系统计算
	RatePeriod float64 `json:"rate_period"` // 本期完成率
	RateCum    float64 `json:"rate_cum"`    // 累计完成率
	SPI        float64 `json:"spi"`
	SPIValid   bool    `json:"spi_valid"`   // 分母为 0 时无效，不据此判灯
	Light      string  `json:"light"`       // 系统判定
	LightWhy   string  `json:"light_why"`   // 触发该灯的具体条件
	Action     string  `json:"action"`      // 细则 4.1 要求的响应
	Mismatch   bool    `json:"mismatch"`    // 自评与系统判定不一致
	Milestones []Milestone `json:"milestones"`
	SourceFile string  `json:"source_file"`
}

type Milestone struct {
	Name       string `json:"name"`
	IsPayment  bool   `json:"is_payment"`
	ContractAt string `json:"contract_at"`
	ForecastAt string `json:"forecast_at"`
	Status     string `json:"status"`
}

type ProgressDataResult struct {
	Items    []ProgressData `json:"items"`
	Warnings []string       `json:"warnings"`
	Summary  struct {
		Total    int `json:"total"`
		Green    int `json:"green"`
		Yellow   int `json:"yellow"`
		Orange   int `json:"orange"`
		Red      int `json:"red"`
		Mismatch int `json:"mismatch"`
	} `json:"summary"`
}

// ScanProgressData 扫描各项目某期的附件 B-1。
func ScanProgressData(projectsDir string, projects []ProjectBrief, period string) *ProgressDataResult {
	res := &ProgressDataResult{}
	compact := strings.ReplaceAll(period, "-", "") // 2026-07 → 202607

	for _, p := range projects {
		dir := filepath.Join(projectsDir, p.ShortName, filepath.FromSlash("06.Report/Monthly Report"))
		name := p.ShortName + "-MPD-" + compact + ".xlsx"
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err != nil {
			continue // 未提交，报送核查里已体现为缺交，这里不重复告警
		}
		d, warns := parseProgressData(path, p.ShortName)
		res.Warnings = append(res.Warnings, warns...)
		if d != nil {
			res.Items = append(res.Items, *d)
		}
	}

	// 风险高的排前面
	rank := map[string]int{LightRed: 0, LightOrange: 1, LightYellow: 2, LightGreen: 3}
	sort.SliceStable(res.Items, func(i, j int) bool {
		return rank[res.Items[i].Light] < rank[res.Items[j].Light]
	})

	for _, it := range res.Items {
		res.Summary.Total++
		switch it.Light {
		case LightRed:
			res.Summary.Red++
		case LightOrange:
			res.Summary.Orange++
		case LightYellow:
			res.Summary.Yellow++
		default:
			res.Summary.Green++
		}
		if it.Mismatch {
			res.Summary.Mismatch++
		}
	}
	return res
}

// parseProgressData 解析附件 B-1：Sheet「进度数据」按 A 列标签取 B 列值，
// Sheet「里程碑」按表头定位。不写死行号（需求 6A.0）。
func parseProgressData(path, project string) (*ProgressData, []string) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, []string{project + "：进度数据表打不开：" + err.Error()}
	}
	defer f.Close()

	var warns []string
	rows, err := f.GetRows(pickSheet(f, "进度数据"))
	if err != nil || len(rows) == 0 {
		return nil, []string{project + "：进度数据表内容为空"}
	}

	// A 列标签 → B 列值
	kv := map[string]string{}
	for _, row := range rows {
		if len(row) < 2 {
			continue
		}
		k := strings.TrimSpace(row[0])
		v := strings.TrimSpace(row[1])
		if k != "" && v != "" && v != "系统计算" {
			kv[k] = v
		}
	}

	d := &ProgressData{
		Project: project, SourceFile: filepath.Base(path),
		Period: kv["报告周期"], DataDate: kv["数据日期"], Reporter: kv["填报人"],
		Currency: kv["币种"], SelfLight: kv["项目自评风险灯"],
	}
	if v, err := strconv.Atoi(kv["关键路径滞后天数"]); err == nil {
		d.LagDays = v
	} else if kv["关键路径滞后天数"] != "" {
		warns = append(warns, project+"：关键路径滞后天数不是整数（"+kv["关键路径滞后天数"]+"）")
	}
	d.MilestoneRisk = kv["付款里程碑预计滞后"] == "是"
	d.HasLDs = kv["本期已产生 LDs"] == "是" || kv["本期已产生LDs"] == "是"
	d.PlanPeriod = parseNum(kv["本期计划产值"])
	d.ActPeriod = parseNum(kv["本期实际产值"])
	d.PlanCum = parseNum(kv["累计计划产值"])
	d.ActCum = parseNum(kv["累计实际产值"])

	if d.PlanPeriod > 0 {
		d.RatePeriod = d.ActPeriod / d.PlanPeriod * 100
	}
	if d.PlanCum > 0 {
		d.RateCum = d.ActCum / d.PlanCum * 100
		d.SPI = d.ActCum / d.PlanCum
		d.SPIValid = true
	} else {
		warns = append(warns, project+"：累计计划产值为 0 或缺失，SPI 无法计算，风险灯仅依据关键路径滞后天数判定")
	}

	d.Light, d.LightWhy = judgeLight(d)
	d.Action = lightActions[d.Light]
	d.Mismatch = d.SelfLight != "" && d.SelfLight != d.Light

	// 里程碑
	if mr, err := f.GetRows(pickSheet(f, "里程碑")); err == nil {
		if hdr, hdrRow := findHeaderRow(mr, "里程碑名称", "合同约定日期"); hdrRow >= 0 {
			get := func(row []string, k string) string {
				i, ok := hdr[k]
				if !ok || i >= len(row) {
					return ""
				}
				return strings.TrimSpace(row[i])
			}
			for r := hdrRow + 1; r < len(mr); r++ {
				if isBlankRow(mr[r]) {
					continue
				}
				name := get(mr[r], "里程碑名称")
				if name == "" {
					continue
				}
				d.Milestones = append(d.Milestones, Milestone{
					Name: name, IsPayment: get(mr[r], "是否付款节点") == "是",
					ContractAt: get(mr[r], "合同约定日期"),
					ForecastAt: get(mr[r], "当前预计/实际完成日期"),
					Status:     get(mr[r], "状态"),
				})
			}
		}
	}
	return d, warns
}

// judgeLight 严格按细则 4.1 判定，取最严重的一项，并说明触发原因。
func judgeLight(d *ProgressData) (string, string) {
	// 红：滞后 >14 天 或 SPI <0.85 或 LDs 已产生
	if d.HasLDs {
		return LightRed, "已产生误期违约金（LDs）"
	}
	if d.LagDays > 14 {
		return LightRed, "关键路径滞后 " + strconv.Itoa(d.LagDays) + " 天（>14 天）"
	}
	if d.SPIValid && d.SPI < 0.85 {
		return LightRed, "SPI " + fmtSPI(d.SPI) + "（<0.85）"
	}
	// 橙：滞后 7-14 天 或 SPI 0.85-0.90 或 付款里程碑预计滞后
	if d.MilestoneRisk {
		return LightOrange, "付款里程碑预计滞后"
	}
	if d.LagDays > 7 {
		return LightOrange, "关键路径滞后 " + strconv.Itoa(d.LagDays) + " 天（7–14 天）"
	}
	if d.SPIValid && d.SPI < 0.90 {
		return LightOrange, "SPI " + fmtSPI(d.SPI) + "（0.85–0.90）"
	}
	// 黄：滞后 3-7 天 或 SPI 0.90-0.95
	if d.LagDays > 3 {
		return LightYellow, "关键路径滞后 " + strconv.Itoa(d.LagDays) + " 天（3–7 天）"
	}
	if d.SPIValid && d.SPI < 0.95 {
		return LightYellow, "SPI " + fmtSPI(d.SPI) + "（0.90–0.95）"
	}
	return LightGreen, "关键路径滞后 " + strconv.Itoa(d.LagDays) + " 天且 SPI 达标"
}

func fmtSPI(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) }
