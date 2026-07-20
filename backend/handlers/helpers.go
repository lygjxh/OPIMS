package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"opims/models"
	"opims/services"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	olc "github.com/google/open-location-code/go"
	"github.com/xuri/excelize/v2"
)

type Handler struct {
	rootPath string
	fw       *services.FileWatcher
}

func NewHandler(rootPath string, fw *services.FileWatcher) *Handler {
	return &Handler{rootPath: rootPath, fw: fw}
}

// ---- helpers ----

func scanProjectFromRows(rows *sql.Rows) *models.Project {
	var p models.Project
	var ca, ua string
	rows.Scan(&p.ID, &p.ShortName, &p.ContractNo, &p.ProjectName, &p.ProjectType,
		&p.ProjectStatus, &p.ImplementUnit, &p.ContractAmount, &p.BudgetAmount,
		&p.ContractScope, &p.KeyPoints, &p.DomesticOverseas,
		&p.Province, &p.City, &p.Address, &p.Country,
		&p.ContractStartYear, &p.ContractStartMonth, &p.ContractEndYear, &p.ContractEndMonth,
		&p.ContractDuration, &p.ActualStartYear, &p.ActualStartMonth,
		&p.PlanEndYear, &p.PlanEndMonth, &p.ActualDuration,
		&p.CompletionYear, &p.CompletionMonth, &p.RunningStatus, &p.AbnormalReason,
		&p.ProgressStatus, &p.Issues,
		&p.CompletedOutput, &p.CompletePercent, &p.ProgressSummary,
		&p.CumReceivable, &p.CumReceived, &p.OwedAmount,
		&p.GPSLat, &p.GPSLng,
		&p.PMContract, &p.PMAppointed, &p.PMOnsite, &p.PMPhone, &p.PMBuilder, &p.PMSafetyCert,
		&p.TechLeadAppointed, &p.TechLeadOnsite, &p.TechLeadPhone, &p.TechLeadTitle,
		&p.QualityMgrAppointed, &p.QualityMgrOnsite, &p.QualityMgrPhone, &p.QualityMgrCert,
		&p.HSEAppointed, &p.HSEOnsite, &p.HSEPhone, &p.HSECert,
		&p.CostMgrAppointed, &p.CostMgrOnsite, &p.CostMgrPhone, &p.CostMgrCert,
		&p.QualityKeyProcess, &p.QualityMeasures,
		&p.SafetyCost, &p.SafetyCostSpent, &p.SafetyCostCum,
		&p.SafetyMajorHazard, &p.SafetyHazardMeasure, &p.SafetyRiskSource, &p.SafetyRiskMeasure,
		&p.OwnerUnit, &p.OwnerContact, &p.OwnerPhone,
		&p.DesignUnit, &p.DesignContact, &p.DesignPhone,
		&p.SupervisionUnit, &p.SupervisionContact, &p.SupervisionPhone,
		&p.Reporter, &p.PersonnelMgmt, &p.PersonnelLabor,
		&p.IsDeleted, &ca, &ua)
	return &p
}

func scanProject(row *sql.Row) *models.Project {
	var p models.Project
	var ca, ua string
	row.Scan(&p.ID, &p.ShortName, &p.ContractNo, &p.ProjectName, &p.ProjectType,
		&p.ProjectStatus, &p.ImplementUnit, &p.ContractAmount, &p.BudgetAmount,
		&p.ContractScope, &p.KeyPoints, &p.DomesticOverseas,
		&p.Province, &p.City, &p.Address, &p.Country,
		&p.ContractStartYear, &p.ContractStartMonth, &p.ContractEndYear, &p.ContractEndMonth,
		&p.ContractDuration, &p.ActualStartYear, &p.ActualStartMonth,
		&p.PlanEndYear, &p.PlanEndMonth, &p.ActualDuration,
		&p.CompletionYear, &p.CompletionMonth, &p.RunningStatus, &p.AbnormalReason,
		&p.ProgressStatus, &p.Issues,
		&p.CompletedOutput, &p.CompletePercent, &p.ProgressSummary,
		&p.CumReceivable, &p.CumReceived, &p.OwedAmount,
		&p.GPSLat, &p.GPSLng,
		&p.PMContract, &p.PMAppointed, &p.PMOnsite, &p.PMPhone, &p.PMBuilder, &p.PMSafetyCert,
		&p.TechLeadAppointed, &p.TechLeadOnsite, &p.TechLeadPhone, &p.TechLeadTitle,
		&p.QualityMgrAppointed, &p.QualityMgrOnsite, &p.QualityMgrPhone, &p.QualityMgrCert,
		&p.HSEAppointed, &p.HSEOnsite, &p.HSEPhone, &p.HSECert,
		&p.CostMgrAppointed, &p.CostMgrOnsite, &p.CostMgrPhone, &p.CostMgrCert,
		&p.QualityKeyProcess, &p.QualityMeasures,
		&p.SafetyCost, &p.SafetyCostSpent, &p.SafetyCostCum,
		&p.SafetyMajorHazard, &p.SafetyHazardMeasure, &p.SafetyRiskSource, &p.SafetyRiskMeasure,
		&p.OwnerUnit, &p.OwnerContact, &p.OwnerPhone,
		&p.DesignUnit, &p.DesignContact, &p.DesignPhone,
		&p.SupervisionUnit, &p.SupervisionContact, &p.SupervisionPhone,
		&p.Reporter, &p.PersonnelMgmt, &p.PersonnelLabor,
		&p.IsDeleted, &ca, &ua)
	return &p
}

func projectsInsertCols() string {
	return `short_name,contract_no,project_name,project_type,project_status,implement_unit,contract_amount,budget_amount,contract_scope,key_points,domestic_overseas,province,city,address,country,contract_start_year,contract_start_month,contract_end_year,contract_end_month,contract_duration,actual_start_year,actual_start_month,plan_end_year,plan_end_month,actual_duration,completion_year,completion_month,running_status,abnormal_reason,progress_status,issues,completed_output,complete_percent,progress_summary,cum_receivable,cum_received,owed_amount,gps_lat,gps_lng,pm_contract,pm_appointed,pm_onsite,pm_phone,pm_builder,pm_safety_cert,tech_lead_appointed,tech_lead_onsite,tech_lead_phone,tech_lead_title,quality_mgr_appointed,quality_mgr_onsite,quality_mgr_phone,quality_mgr_cert,hse_mgr_appointed,hse_mgr_onsite,hse_mgr_phone,hse_mgr_cert,cost_mgr_appointed,cost_mgr_onsite,cost_mgr_phone,cost_mgr_cert,quality_key_process,quality_measures,safety_cost,safety_cost_spent,safety_cost_cum,safety_major_hazard,safety_hazard_measure,safety_risk_source,safety_risk_measure,owner_unit,owner_contact,owner_phone,design_unit,design_contact,design_phone,supervision_unit,supervision_contact,supervision_phone,reporter,personnel_mgmt,personnel_labor`
}

func projectsInsertVals(p *models.Project) []interface{} {
	return []interface{}{
		p.ShortName, p.ContractNo, p.ProjectName, p.ProjectType, p.ProjectStatus,
		p.ImplementUnit, p.ContractAmount, p.BudgetAmount, p.ContractScope, p.KeyPoints,
		p.DomesticOverseas, p.Province, p.City, p.Address, p.Country,
		p.ContractStartYear, p.ContractStartMonth, p.ContractEndYear, p.ContractEndMonth, p.ContractDuration,
		p.ActualStartYear, p.ActualStartMonth, p.PlanEndYear, p.PlanEndMonth, p.ActualDuration,
		p.CompletionYear, p.CompletionMonth,
		p.RunningStatus, p.AbnormalReason, p.ProgressStatus, p.Issues,
		p.CompletedOutput, p.CompletePercent, p.ProgressSummary, p.CumReceivable, p.CumReceived, p.OwedAmount,
		p.GPSLat, p.GPSLng,
		p.PMContract, p.PMAppointed, p.PMOnsite, p.PMPhone, p.PMBuilder, p.PMSafetyCert,
		p.TechLeadAppointed, p.TechLeadOnsite, p.TechLeadPhone, p.TechLeadTitle,
		p.QualityMgrAppointed, p.QualityMgrOnsite, p.QualityMgrPhone, p.QualityMgrCert,
		p.HSEAppointed, p.HSEOnsite, p.HSEPhone, p.HSECert,
		p.CostMgrAppointed, p.CostMgrOnsite, p.CostMgrPhone, p.CostMgrCert,
		p.QualityKeyProcess, p.QualityMeasures,
		p.SafetyCost, p.SafetyCostSpent, p.SafetyCostCum,
		p.SafetyMajorHazard, p.SafetyHazardMeasure, p.SafetyRiskSource, p.SafetyRiskMeasure,
		p.OwnerUnit, p.OwnerContact, p.OwnerPhone,
		p.DesignUnit, p.DesignContact, p.DesignPhone,
		p.SupervisionUnit, p.SupervisionContact, p.SupervisionPhone,
		p.Reporter, p.PersonnelMgmt, p.PersonnelLabor,
	}
}

func placeholders(n int) string { return strings.Repeat("?,", n-1) + "?" }

type nameMapping struct{ ShortName, ContractNo string }

func loadNameMapping(rootPath string) map[string]nameMapping {
	mapping := map[string]nameMapping{}
	paths := []string{
		filepath.Join(rootPath, "海外项目简称.xlsx"),
		`D:\WPS云盘\186113660\WPS云盘\OneDrive - 中国化学工程股份有限公司\海外运营中心\06.Received File\01.项目管理情况汇总表\海外项目简称.xlsx`,
	}
	var f *excelize.File
	var err error
	for _, p := range paths {
		if f, err = excelize.OpenFile(p); err == nil {
			break
		}
	}
	if err != nil {
		return mapping
	}
	defer f.Close()
	if rows, err := f.GetRows(f.GetSheetList()[0]); err == nil {
		for i := 1; i < len(rows); i++ {
			row := rows[i]
			if len(row) < 4 {
				continue
			}
			name := strings.TrimSpace(row[3])
			if name != "" && strings.TrimSpace(row[1]) != "" {
				cn := ""
				if len(row) >= 3 {
					cn = strings.TrimSpace(row[2])
				}
				mapping[name] = nameMapping{ShortName: strings.TrimSpace(row[1]), ContractNo: cn}
			}
		}
	}
	return mapping
}

func parseGPS(input string, p *models.Project) {
	input = strings.TrimSpace(input)
	if input == "" {
		return
	}
	if parts := strings.Split(input, ","); len(parts) == 2 {
		if lat, e1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64); e1 == nil {
			if lng, e2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); e2 == nil {
				p.GPSLat, p.GPSLng = lat, lng
				return
			}
		}
	}
	if lat, lng, ok := parseDMS(input); ok {
		p.GPSLat, p.GPSLng = lat, lng
		return
	}
	if code, err := olc.Decode(input); err == nil {
		p.GPSLat = code.LatLo + (code.LatHi-code.LatLo)/2
		p.GPSLng = code.LngLo + (code.LngHi-code.LngLo)/2
	}
}

func parseDMS(s string) (lat, lng float64, ok bool) {
	re := regexp.MustCompile(`(\d+)[° ]\s*(\d+)['′ ]\s*([\d.]+)["″ ]\s*([NS])\s+(\d+)[° ]\s*(\d+)['′ ]\s*([\d.]+)["″ ]\s*([EW])`)
	m := re.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return 0, 0, false
	}
	d1, _ := strconv.ParseFloat(m[1], 64)
	d2, _ := strconv.ParseFloat(m[2], 64)
	d3, _ := strconv.ParseFloat(m[3], 64)
	lat = d1 + d2/60 + d3/3600
	if m[4] == "S" {
		lat = -lat
	}
	d4, _ := strconv.ParseFloat(m[5], 64)
	d5, _ := strconv.ParseFloat(m[6], 64)
	d6, _ := strconv.ParseFloat(m[7], 64)
	lng = d4 + d5/60 + d6/3600
	if m[8] == "W" {
		lng = -lng
	}
	return lat, lng, true
}

func extractTrailingNumber(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] >= '0' && s[i] <= '9' {
			j := i
			for j > 0 && s[j-1] >= '0' && s[j-1] <= '9' {
				j--
			}
			n, _ := strconv.Atoi(s[j : i+1])
			return n
		}
	}
	return 0
}

func chineseNumber(n int) string {
	if n <= 0 || n > 99 {
		return ""
	}
	if n <= 10 {
		return []string{"", "一", "二", "三", "四", "五", "六", "七", "八", "九", "十"}[n]
	}
	if n < 20 {
		return "十" + chineseNumber(n-10)
	}
	if n%10 == 0 {
		return chineseNumber(n/10) + "十"
	}
	return chineseNumber(n/10) + "十" + chineseNumber(n%10)
}

func jsonOK(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
