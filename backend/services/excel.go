package services

import (
	"fmt"
	"opims/models"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

func ParseExcel(path string) ([]models.Project, error) {
	f, err := excelize.OpenFile(path)
	if err != nil {
		return nil, fmt.Errorf("open excel: %w", err)
	}
	defer f.Close()

	var projects []models.Project

	for _, sheet := range f.GetSheetList() {
		rows, err := f.GetRows(sheet)
		if err != nil || len(rows) < 5 {
			continue
		}

		status := inferStatus(sheet)

		// Detect if sheet has 项目简称/合同编号 columns
		headerRow := strings.Join(rows[1], " ")
		hasShortName := strings.Contains(headerRow, "项目简称")

		for i := 0; i < len(rows); i++ {
			row := rows[i]
			if len(row) < 3 {
				continue
			}
			firstCol := strings.TrimSpace(row[0])
			if _, err := strconv.Atoi(firstCol); err != nil {
				continue
			}

			c := func(idx int) string {
				if idx < len(row) {
					return strings.TrimSpace(row[idx])
				}
				return ""
			}
			cf := func(idx int) float64 {
				s := strings.TrimSpace(strings.ReplaceAll(c(idx), ",", ""))
				v, _ := strconv.ParseFloat(s, 64)
				return v
			}
			ci := func(idx int) int {
				v, _ := strconv.Atoi(strings.TrimSpace(c(idx)))
				return v
			}

			p := models.Project{ProjectStatus: status}

			if hasShortName {
				p.ShortName = c(1)
				p.ContractNo = c(2)
				parseZaiJianShift(&p, c, cf, ci)
			} else {
				p.ShortName = c(2) // fallback: use project name
				switch status {
				case "未开工":
					parseWeiKaiGong(&p, c, cf, ci)
				case "停工":
					parseTingGong(&p, c, cf, ci)
				case "完工":
					parseWanGong(&p, c, cf, ci)
				case "在建":
					parseZaiJianNoShift(&p, c, cf, ci)
				default:
					parseTingGong(&p, c, cf, ci)
				}
			}

			// Country
			if p.DomesticOverseas == "境外" {
				p.Country = extractCountry(p.ProjectName, p.Address)
			} else if p.DomesticOverseas == "境内" {
				p.Country = "中国"
			}

			projects = append(projects, p)
		}
	}

	return projects, nil
}

// 在建（含新中标）: C2=项目简称, C3=合同编号(shift=2)
func parseZaiJianShift(p *models.Project, c func(int) string, cf func(int) float64, ci func(int) int) {
	p.ImplementUnit = c(3)
	p.ProjectName = c(4)
	p.ContractAmount = cf(5)
	p.BudgetAmount = cf(6)
	p.ContractScope = c(7)
	p.KeyPoints = c(8)
	p.DomesticOverseas = c(9)
	p.Province = c(10)
	p.City = c(11)
	p.Address = c(12)
	p.PersonnelMgmt = ci(13)
	p.PersonnelLabor = ci(14)
	p.ContractStartYear = ci(15)
	p.ContractStartMonth = ci(16)
	p.ContractEndYear = ci(17)
	p.ContractEndMonth = ci(18)
	p.ContractDuration = ci(19)
	p.ActualStartYear = ci(20)
	p.ActualStartMonth = ci(21)
	p.PlanEndYear = ci(22)
	p.PlanEndMonth = ci(23)
	p.ActualDuration = ci(24)
	p.RunningStatus = c(25)
	p.AbnormalReason = c(26)
	p.CompletedOutput = cf(27)
	p.CompletePercent = c(28)
	p.ProgressStatus = c(29)
	p.ProgressSummary = c(30)
	p.CumReceivable = cf(31)
	p.CumReceived = cf(32)
	p.OwedAmount = cf(33)
	p.Issues = c(34)
	p.PMContract = c(35)
	p.PMAppointed = c(36)
	p.PMOnsite = c(37)
	p.PMPhone = c(38)
	p.PMBuilder = c(39)
	p.PMSafetyCert = c(40)
	p.TechLeadAppointed = c(41)
	p.TechLeadOnsite = c(42)
	p.TechLeadPhone = c(43)
	p.TechLeadTitle = c(44)
	p.QualityMgrAppointed = c(45)
	p.QualityMgrOnsite = c(46)
	p.QualityMgrPhone = c(47)
	p.QualityMgrCert = c(48)
	p.HSEAppointed = c(49)
	p.HSEOnsite = c(50)
	p.HSEPhone = c(51)
	p.HSECert = c(52)
	p.OwnerUnit = c(53)
	p.OwnerContact = c(54)
	p.OwnerPhone = c(55)
	p.DesignUnit = c(56)
	p.DesignContact = c(57)
	p.DesignPhone = c(58)
	p.SupervisionUnit = c(59)
	p.SupervisionContact = c(60)
	p.SupervisionPhone = c(61)
	p.Reporter = c(62)
}

// 在建（含新中标）- 原表无简称列
func parseZaiJianNoShift(p *models.Project, c func(int) string, cf func(int) float64, ci func(int) int) {
	p.ImplementUnit = c(1)
	p.ProjectName = c(2)
	p.ContractAmount = cf(3)
	p.BudgetAmount = cf(4)
	p.ContractScope = c(5)
	p.KeyPoints = c(6)
	p.DomesticOverseas = c(7)
	p.Province = c(8)
	p.City = c(9)
	p.Address = c(10)
	p.PersonnelMgmt = ci(11)
	p.PersonnelLabor = ci(12)
	p.ContractStartYear = ci(13)
	p.ContractStartMonth = ci(14)
	p.ContractEndYear = ci(15)
	p.ContractEndMonth = ci(16)
	p.ContractDuration = ci(17)
	p.ActualStartYear = ci(18)
	p.ActualStartMonth = ci(19)
	p.PlanEndYear = ci(20)
	p.PlanEndMonth = ci(21)
	p.ActualDuration = ci(22)
	p.RunningStatus = c(23)
	p.AbnormalReason = c(24)
	p.CompletedOutput = cf(25)
	p.CompletePercent = c(26)
	p.ProgressStatus = c(27)
	p.ProgressSummary = c(28)
	p.CumReceivable = cf(29)
	p.CumReceived = cf(30)
	p.OwedAmount = cf(31)
	p.Issues = c(32)
	p.PMContract = c(33)
	p.PMAppointed = c(34)
	p.PMOnsite = c(35)
	p.PMPhone = c(36)
	p.PMBuilder = c(37)
	p.PMSafetyCert = c(38)
	p.TechLeadAppointed = c(39)
	p.TechLeadOnsite = c(40)
	p.TechLeadPhone = c(41)
	p.TechLeadTitle = c(42)
	p.QualityMgrAppointed = c(43)
	p.QualityMgrOnsite = c(44)
	p.QualityMgrPhone = c(45)
	p.QualityMgrCert = c(46)
	p.HSEAppointed = c(47)
	p.HSEOnsite = c(48)
	p.HSEPhone = c(49)
	p.HSECert = c(50)
	p.OwnerUnit = c(51)
	p.OwnerContact = c(52)
	p.OwnerPhone = c(53)
	p.DesignUnit = c(54)
	p.DesignContact = c(55)
	p.DesignPhone = c(56)
	p.SupervisionUnit = c(57)
	p.SupervisionContact = c(58)
	p.SupervisionPhone = c(59)
	p.Reporter = c(60)
}

// 停工: NO shift
func parseTingGong(p *models.Project, c func(int) string, cf func(int) float64, ci func(int) int) {
	p.ImplementUnit = c(1)
	p.ProjectName = c(2)
	p.ContractAmount = cf(3)
	p.BudgetAmount = cf(4)
	p.ContractScope = c(5)
	p.KeyPoints = c(6)
	p.DomesticOverseas = c(7)
	p.Province = c(8)
	p.City = c(9)
	p.Address = c(10)
	p.PersonnelMgmt = ci(11)
	p.PersonnelLabor = ci(12)
	p.ContractStartYear = ci(13)
	p.ContractStartMonth = ci(14)
	p.ContractEndYear = ci(15)
	p.ContractEndMonth = ci(16)
	p.ContractDuration = ci(17)
	p.ActualStartYear = ci(18)
	p.ActualStartMonth = ci(19)
	p.PlanEndYear = ci(20)
	p.PlanEndMonth = ci(21)
	p.ActualDuration = ci(22)
	p.RunningStatus = c(23)
	p.AbnormalReason = c(24)
	p.CompletedOutput = cf(25)
	p.CompletePercent = c(26)
	p.ProgressStatus = c(27)
	p.ProgressSummary = c(28)
	p.CumReceivable = cf(29)
	p.CumReceived = cf(30)
	p.OwedAmount = cf(31)
	p.Issues = c(32)
	p.PMContract = c(33)
	p.PMAppointed = c(34)
	p.PMOnsite = c(35)
	p.PMPhone = c(36)
	p.PMBuilder = c(37)
	p.PMSafetyCert = c(38)
	p.TechLeadAppointed = c(39)
	p.TechLeadOnsite = c(40)
	p.TechLeadPhone = c(41)
	p.TechLeadTitle = c(42)
	p.QualityMgrAppointed = c(43)
	p.QualityMgrOnsite = c(44)
	p.QualityMgrPhone = c(45)
	p.QualityMgrCert = c(46)
	p.HSEAppointed = c(47)
	p.HSEOnsite = c(48)
	p.HSEPhone = c(49)
	p.HSECert = c(50)
	p.CostMgrAppointed = c(51)
	p.CostMgrOnsite = c(52)
	p.CostMgrPhone = c(53)
	p.CostMgrCert = c(54)
	p.QualityKeyProcess = c(55)
	p.QualityMeasures = c(56)
	p.SafetyCost = cf(57)
	p.SafetyCostSpent = cf(58)
	p.SafetyCostCum = cf(59)
	p.SafetyMajorHazard = c(60)
	p.SafetyHazardMeasure = c(61)
	p.SafetyRiskSource = c(62)
	p.SafetyRiskMeasure = c(63)
	p.OwnerUnit = c(64)
	p.OwnerContact = c(65)
	p.OwnerPhone = c(66)
	p.DesignUnit = c(67)
	p.DesignContact = c(68)
	p.DesignPhone = c(69)
	p.SupervisionUnit = c(70)
	p.SupervisionContact = c(71)
	p.SupervisionPhone = c(72)
	p.Reporter = c(73)
}

// 未开工: NO shift, simpler layout
func parseWeiKaiGong(p *models.Project, c func(int) string, cf func(int) float64, ci func(int) int) {
	p.ImplementUnit = c(1)
	p.ProjectName = c(2)
	p.ContractAmount = cf(3)
	p.ContractScope = c(4)
	p.KeyPoints = c(5)
	p.DomesticOverseas = c(6)
	p.Province = c(7)
	p.City = c(8)
	p.Address = c(9)
	p.ContractStartYear = ci(10)
	p.ContractStartMonth = ci(11)
	p.ContractEndYear = ci(12)
	p.ContractEndMonth = ci(13)
	p.ContractDuration = ci(14)
	p.PMContract = c(15)
	p.OwnerUnit = c(16)
	p.OwnerContact = c(17)
	p.OwnerPhone = c(18)
	p.Reporter = c(19)
}

// 完工: NO shift
func parseWanGong(p *models.Project, c func(int) string, cf func(int) float64, ci func(int) int) {
	p.ImplementUnit = c(1)
	p.ProjectName = c(2)
	p.ContractAmount = cf(3)
	p.ContractScope = c(4)
	p.DomesticOverseas = c(5)
	p.Province = c(6)
	p.City = c(7)
	p.Address = c(8)
	p.ContractStartYear = ci(9)
	p.ContractStartMonth = ci(10)
	p.ContractEndYear = ci(11)
	p.ContractEndMonth = ci(12)
	p.ContractDuration = ci(13)
	p.ActualStartYear = ci(14)
	p.ActualStartMonth = ci(15)
	p.CompletionYear = ci(16)
	p.CompletionMonth = ci(17)
	p.PMContract = c(19)
	p.PMAppointed = c(20)
	p.PMOnsite = c(21)
	p.PMPhone = c(22)
	p.PMBuilder = c(23)
	p.PMSafetyCert = c(24)
	p.TechLeadAppointed = c(25)
	p.TechLeadOnsite = c(26)
	p.TechLeadPhone = c(27)
	p.TechLeadTitle = c(28)
	p.QualityMgrAppointed = c(29)
	p.QualityMgrOnsite = c(30)
	p.QualityMgrPhone = c(31)
	p.QualityMgrCert = c(32)
	p.HSEAppointed = c(33)
	p.HSEOnsite = c(34)
	p.HSEPhone = c(35)
	p.HSECert = c(36)
	p.OwnerUnit = c(37)
	p.OwnerContact = c(38)
	p.OwnerPhone = c(39)
	p.DesignUnit = c(40)
	p.DesignContact = c(41)
	p.DesignPhone = c(42)
	p.SupervisionUnit = c(43)
	p.SupervisionContact = c(44)
	p.SupervisionPhone = c(45)
	p.Reporter = c(46)
}

func ExportExcel(projects []models.Project, allColumns bool) ([]byte, error) {
	f := excelize.NewFile()
	sheet := "项目清单"
	f.SetSheetName("Sheet1", sheet)

	if allColumns {
		return exportAllColumns(f, sheet, projects)
	}

	headers := []string{"序号", "项目简称", "合同编号", "项目名称", "项目类型", "项目状态",
		"实施单位", "合同额(万元)", "境内/境外", "国别", "省", "市", "地址"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}
	for i, p := range projects {
		row := i + 2
		vals := []interface{}{i + 1, p.ShortName, p.ContractNo, p.ProjectName, p.ProjectType, p.ProjectStatus,
			p.ImplementUnit, p.ContractAmount, p.DomesticOverseas, p.Country, p.Province, p.City, p.Address}
		for j, v := range vals {
			f.SetCellValue(sheet, cellName(j+1, row), v)
		}
	}
	return bufFromFile(f)
}

func exportAllColumns(f *excelize.File, sheet string, projects []models.Project) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func bufFromFile(f *excelize.File) ([]byte, error) {
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func cellName(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}

func inferStatus(sheet string) string {
	s := strings.TrimSpace(sheet)
	switch {
	case strings.Contains(s, "未开工"):
		return "未开工"
	case strings.Contains(s, "停工"):
		return "停工"
	case strings.Contains(s, "完工"):
		return "完工"
	case strings.Contains(s, "在建"):
		return "在建"
	default:
		return "在建"
	}
}

func extractCountry(name, addr string) string {
	keywords := map[string]string{
		"蒙古": "蒙古", "俄罗斯": "俄罗斯", "伊拉克": "伊拉克",
		"印尼": "印尼", "印度尼西亚": "印尼", "尼日利亚": "尼日利亚",
		"纳米比亚": "纳米比亚", "阿布扎比": "阿联酋", "阿联酋": "阿联酋",
		"伊朗": "伊朗", "格什姆": "伊朗", "ADNOC": "阿联酋",
		"DBN": "阿联酋", "LNG": "阿联酋", "PLF": "尼日利亚",
	}
	for k, v := range keywords {
		if strings.Contains(name+addr, k) {
			return v
		}
	}
	return "其他"
}
