package models

type Project struct {
	ID                  int     `json:"id"`
	ShortName           string  `json:"short_name"`
	ContractNo          string  `json:"contract_no"`
	ProjectName         string  `json:"project_name"`
	ProjectType         string  `json:"project_type"`
	ProjectStatus       string  `json:"project_status"`
	ImplementUnit       string  `json:"implement_unit"`
	ContractAmount      float64 `json:"contract_amount"`
	BudgetAmount        float64 `json:"budget_amount"`
	ContractScope       string  `json:"contract_scope"`
	KeyPoints           string  `json:"key_points"`
	DomesticOverseas    string  `json:"domestic_overseas"`
	Province            string  `json:"province"`
	City                string  `json:"city"`
	Address             string  `json:"address"`
	Country             string  `json:"country"`
	// Dates
	ContractStartYear   int `json:"contract_start_year"`
	ContractStartMonth  int `json:"contract_start_month"`
	ContractEndYear     int `json:"contract_end_year"`
	ContractEndMonth    int `json:"contract_end_month"`
	ContractDuration    int `json:"contract_duration"`
	ActualStartYear     int `json:"actual_start_year"`
	ActualStartMonth    int `json:"actual_start_month"`
	PlanEndYear         int `json:"plan_end_year"`
	PlanEndMonth        int `json:"plan_end_month"`
	ActualDuration      int `json:"actual_duration"`
	CompletionYear      int `json:"completion_year"`
	CompletionMonth     int `json:"completion_month"`
	// Status
	RunningStatus       string  `json:"running_status"`
	AbnormalReason      string  `json:"abnormal_reason"`
	ProgressStatus      string  `json:"progress_status"`
	Issues              string  `json:"issues"`
	CompletedOutput     float64 `json:"completed_output"`
	CompletePercent     string  `json:"complete_percent"`
	ProgressSummary     string  `json:"progress_summary"`
	CumReceivable       float64 `json:"cum_receivable"`
	CumReceived         float64 `json:"cum_received"`
	OwedAmount          float64 `json:"owed_amount"`
	// GPS
	GPSLat              float64 `json:"gps_lat"`
	GPSLng              float64 `json:"gps_lng"`
	// Personnel
	PMContract          string `json:"pm_contract"`
	PMAppointed         string `json:"pm_appointed"`
	PMOnsite            string `json:"pm_onsite"`
	PMPhone             string `json:"pm_phone"`
	PMBuilder           string `json:"pm_builder"`
	PMSafetyCert        string `json:"pm_safety_cert"`
	TechLeadAppointed   string `json:"tech_lead_appointed"`
	TechLeadOnsite      string `json:"tech_lead_onsite"`
	TechLeadPhone       string `json:"tech_lead_phone"`
	TechLeadTitle       string `json:"tech_lead_title"`
	QualityMgrAppointed string `json:"quality_mgr_appointed"`
	QualityMgrOnsite    string `json:"quality_mgr_onsite"`
	QualityMgrPhone     string `json:"quality_mgr_phone"`
	QualityMgrCert      string `json:"quality_mgr_cert"`
	HSEAppointed        string `json:"hse_mgr_appointed"`
	HSEOnsite           string `json:"hse_mgr_onsite"`
	HSEPhone            string `json:"hse_mgr_phone"`
	HSECert             string `json:"hse_mgr_cert"`
	CostMgrAppointed    string `json:"cost_mgr_appointed"`
	CostMgrOnsite       string `json:"cost_mgr_onsite"`
	CostMgrPhone        string `json:"cost_mgr_phone"`
	CostMgrCert         string `json:"cost_mgr_cert"`
	// Quality/Safety
	QualityKeyProcess   string  `json:"quality_key_process"`
	QualityMeasures     string  `json:"quality_measures"`
	SafetyCost          float64 `json:"safety_cost"`
	SafetyCostSpent     float64 `json:"safety_cost_spent"`
	SafetyCostCum       float64 `json:"safety_cost_cum"`
	SafetyMajorHazard   string  `json:"safety_major_hazard"`
	SafetyHazardMeasure string  `json:"safety_hazard_measure"`
	SafetyRiskSource    string  `json:"safety_risk_source"`
	SafetyRiskMeasure   string  `json:"safety_risk_measure"`
	// Owner/Design/Supervision
	OwnerUnit           string `json:"owner_unit"`
	OwnerContact        string `json:"owner_contact"`
	OwnerPhone          string `json:"owner_phone"`
	DesignUnit          string `json:"design_unit"`
	DesignContact       string `json:"design_contact"`
	DesignPhone         string `json:"design_phone"`
	SupervisionUnit     string `json:"supervision_unit"`
	SupervisionContact  string `json:"supervision_contact"`
	SupervisionPhone    string `json:"supervision_phone"`
	Reporter            string `json:"reporter"`
	// Meta
	PersonnelMgmt       int    `json:"personnel_mgmt"`
	PersonnelLabor      int    `json:"personnel_labor"`
	IsDeleted           int    `json:"is_deleted"`
}

type SubBlacklist struct {
	ID              int    `json:"id"`
	SubShortName    string `json:"sub_short_name"`
	SubFullName     string `json:"sub_full_name"`
	Country         string `json:"country"`
	RelatedProject  string `json:"related_project"`
	ListReason      string `json:"list_reason"`
	ListDate        string `json:"list_date"`
	RestrictLevel   string `json:"restrict_level"`
	RestrictUntil   string `json:"restrict_until"`
	ListReporter    string `json:"list_reporter"`
	DelistReason    string `json:"delist_reason"`
	DelistDate      string `json:"delist_date"`
	DelistReporter  string `json:"delist_reporter"`
	Status          string `json:"status"`
}

// SubcontractRecord mirrors one row from the management ledger (project_subcontract table).
type SubcontractRecord struct {
	ID                     int     `json:"id"`
	Period                 string  `json:"period"`
	SeqNo                  string  `json:"seq_no"`
	BranchCompany          string  `json:"branch_company"`
	ProjectName            string  `json:"project_name"`
	MainContractAmount     float64 `json:"main_contract_amount"`
	SubName                string  `json:"sub_name"`
	SubTier                string  `json:"sub_tier"`
	SubProfessionRaw       string  `json:"sub_profession_raw"`
	SubContractProfession  string  `json:"sub_contract_profession"`
	SubController          string  `json:"sub_controller"`
	SubControllerPhone     string  `json:"sub_controller_phone"`
	ContractNo             string  `json:"contract_no"`
	ContractName           string  `json:"contract_name"`
	ContractAmount         float64 `json:"contract_amount"`
	SupplementAmount       float64 `json:"supplement_amount"`
	ContractDate           string  `json:"contract_date"`
	ProgressPercent        string  `json:"progress_percent"`
	EntryDate              string  `json:"entry_date"`
	ExitDate               string  `json:"exit_date"`
	EvaluationCompleted    string  `json:"evaluation_completed"`
	PersonnelCount         int     `json:"personnel_count"`
	SiteLeader             string  `json:"site_leader"`
	SiteLeaderApproved     string  `json:"site_leader_approved"`
	SiteLeaderStatus       string  `json:"site_leader_status"`
	TechLeader             string  `json:"tech_leader"`
	TechLeaderApproved     string  `json:"tech_leader_approved"`
	TechLeaderStatus       string  `json:"tech_leader_status"`
	SafetyOfficer          string  `json:"safety_officer"`
	SafetyOfficerApproved  string  `json:"safety_officer_approved"`
	SafetyOfficerStatus    string  `json:"safety_officer_status"`
	ContractCompliance     string  `json:"contract_compliance"`
	NoncomplianceNote      string  `json:"noncompliance_note"`
	Remarks                string  `json:"remarks"`
	ProjectShortName       string  `json:"project_short_name"`
	StandardizedProfession string  `json:"standardized_profession"`
	ProfessionCategory     string  `json:"profession_category"`
	IsDeleted              int     `json:"is_deleted"`
	Blacklisted            bool    `json:"blacklisted"`
}

// SubcontractorBase represents a subcontractor in the library.
// 2026-07-26 重构：以「分包商编号 sub_no」为唯一键，单 Sheet 31 列模板。
type SubcontractorBase struct {
	ID              int    `json:"id"`
	ReportProject   string `json:"report_project"`   // 上报项目
	SubNo           string `json:"sub_no"`           // 分包商编号（唯一键）
	ShortName       string `json:"short_name"`       // 分包商简称
	FullName        string `json:"full_name"`        // 分包商名称
	Country         string `json:"country"`          // 国别
	EnterpriseType  string `json:"enterprise_type"`  // 企业性质
	EstablishedDate string `json:"established_date"` // 成立日期
	RegCapital      string `json:"reg_capital"`      // 注册资金
	LegalRep        string `json:"legal_rep"`        // 法人及身份证
	LegalRepPhone   string `json:"legal_rep_phone"`  // 法人联系方式
	Agent           string `json:"agent"`            // 委托代理人及身份证
	AgentPhone      string `json:"agent_phone"`      // 委托代理人联系方式
	Region          string `json:"region"`           // 所在地区
	Address         string `json:"address"`          // 详细地址
	Qualification   string `json:"qualification"`    // 资质类别及等级
	CreditRating    string `json:"credit_rating"`    // 资信等级
	Grade           string `json:"grade"`            // 分包商等级
	Classification  string `json:"classification"`   // 分包商分级
	BusinessScope   string `json:"business_scope"`   // 经营范围
	Recommender     string `json:"recommender"`      // 推荐人
	ReportUnit      string `json:"report_unit"`      // 上报单位
	UnitHead        string `json:"unit_head"`        // 单位负责人
	Category        string `json:"category"`         // 分类
	Profession      string `json:"profession"`       // 专业
	Notes           string `json:"notes"`            // 备注
	AssocUnit       string `json:"assoc_unit"`       // 关联单位
	// 汇总字段（来自项目分包，非入库列）
	ContractCount int     `json:"contract_count"`
	TotalAmount   float64 `json:"total_amount"`
}

// SubcontractorProject is a single cooperation history entry.
type SubcontractorProject struct {
	ID                 int     `json:"id"`
	SubShortName       string  `json:"sub_short_name"`
	ProjectShortName   string  `json:"project_short_name"`
	ProjectName        string  `json:"project_name"`
	StartDate          string  `json:"start_date"`
	EndDate            string  `json:"end_date"`
	ContractNo         string  `json:"contract_no"`
	ContractAmount     float64 `json:"contract_amount"`
	Scope              string  `json:"scope"`
	ProfessionCategory string  `json:"profession_category"`
	Profession         string  `json:"profession"`
	OtherProfessions   string  `json:"other_professions"`
	ProjectStatus      string  `json:"project_status"`
	IsManual           int     `json:"is_manual"`
	Notes              string  `json:"notes"`
}

type FileNode struct {
	Name     string     `json:"name"`
	Path     string     `json:"path"`
	IsDir    bool       `json:"is_dir"`
	Size     int64      `json:"size"`
	ModTime  string     `json:"mod_time"`
	Children []FileNode `json:"children,omitempty"`
}

type DashboardData struct {
	TotalProjects  int            `json:"total_projects"`
	StatusCounts   map[string]int `json:"status_counts"`
	ProjectMarkers []MapMarker    `json:"project_markers"`
}

type MapMarker struct {
	ShortName string  `json:"short_name"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	Country   string  `json:"country"`
	Status    string  `json:"status"`
}

// ContractSlice is one data item in the contract distribution donut chart.
type ContractSlice struct {
	Name       string  `json:"name"`
	Amount     float64 `json:"amount"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

// ContractDistributionResponse is the response for GET /api/dashboard/contract-distribution.
type ContractDistributionResponse struct {
	Slices      []ContractSlice `json:"slices"`
	TotalAmount float64         `json:"total_amount"`
	TotalCount  int             `json:"total_count"`
}

// RegionCountryDetail is one country's data in a region drill-down.
type RegionCountryDetail struct {
	Name       string  `json:"name"`
	Amount     float64 `json:"amount"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}
