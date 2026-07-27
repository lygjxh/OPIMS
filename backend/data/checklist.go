package data

// 报送清单：定义每类文件的周期、截止规则与存放位置。
// 依据《OPIMS 进度管理模块 需求说明书 V1.1》4.1.2。
//
// 一期先内置于代码，后续若需管理员在界面增删改，把本表迁入数据库即可，
// 上层扫描逻辑不依赖清单来源。

// 周期类型
const (
	CycleMonth   = "月"
	CycleQuarter = "季"
	CycleWeek    = "周"
)

// 截止日基准月
const (
	BaseSameMonth   = "当月"
	BaseNextMonth   = "次月"
	BaseQuarterEnd  = "季度末月"
	BaseWeekly      = "每周" // 周报：按周内某天，不看基准月
)

// ChecklistItem 一个报送清单项。
type ChecklistItem struct {
	Code     string `json:"code"`      // 文件类型代码，如 MPR
	Name     string `json:"name"`      // 中文名
	Cycle    string `json:"cycle"`     // 周期：月/季/周
	Base     string `json:"base"`      // 截止日基准月
	Day      int    `json:"day"`       // 截止日：几号（周报时为周几，5=周五）
	Required bool   `json:"required"`  // 是否必交
	Folder   string `json:"folder"`    // 相对项目文件夹的存放路径
	Note     string `json:"note"`      // 备注（如"是否纳入考核待定"）
}

// DefaultChecklist 默认报送清单。Folder 用 / 分隔，使用时经 filepath.FromSlash 转换。
var DefaultChecklist = []ChecklistItem{
	{Code: "MPR", Name: "月度进度分析报告", Cycle: CycleMonth, Base: BaseNextMonth, Day: 5, Required: true,
		Folder: "06.Report/Monthly Report"},
	{Code: "SIX", Name: "月度管理问题及建议（六方面梳理）", Cycle: CycleMonth, Base: BaseSameMonth, Day: 30, Required: true,
		Folder: "06.Report/Monthly Report"},
	{Code: "RP3", Name: "三月滚动计划", Cycle: CycleMonth, Base: BaseSameMonth, Day: 25, Required: true,
		Folder: "05.Schedule"},
	{Code: "TBC", Name: "合同时效日历", Cycle: CycleMonth, Base: BaseNextMonth, Day: 5, Required: true,
		Folder: "05.Schedule/Time Bar"},
	{Code: "EOT", Name: "工期索赔跟踪台账", Cycle: CycleMonth, Base: BaseNextMonth, Day: 5, Required: true,
		Folder: "05.Schedule/EOT"},
	{Code: "QOV", Name: "季度产值分解表", Cycle: CycleQuarter, Base: BaseQuarterEnd, Day: 25, Required: true,
		Folder: "06.Report/Monthly Report"},
	{Code: "WKR", Name: "项目周报", Cycle: CycleWeek, Base: BaseWeekly, Day: 5, Required: false,
		Folder: "06.Report/Weekly Report",
		Note:   "截止每周周五；是否纳入报送核查与考核待确认，当前不计入合规率"},
}

// PlanLevels 进度计划的四个层级。事件驱动、无固定周期，
// 不参与「已交/迟交/缺交」判定，仅在看板中展示各级当前最新版本。
var PlanLevels = []string{"L1", "L2", "L3", "L4"}

// PlanFolder 进度计划的存放目录（相对项目文件夹）。
const PlanFolder = "05.Schedule"

// FindChecklistItem 按代码查找清单项。
func FindChecklistItem(code string) (ChecklistItem, bool) {
	for _, it := range DefaultChecklist {
		if it.Code == code {
			return it, true
		}
	}
	return ChecklistItem{}, false
}
