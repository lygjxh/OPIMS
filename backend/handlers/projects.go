package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"opims/database"
	"opims/models"
	"opims/services"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Projects dispatches GET (list) / POST (create) / PUT (update).
func (h *Handler) Projects(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case "GET":
		h.listProjects(w, r)
	case "POST":
		h.createProject(w, r)
	case "PUT":
		h.updateProject(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// ProjectByID handles single-project GET/PUT/DELETE.
func (h *Handler) ProjectByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	idStr := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	if idStr == "" || idStr == "import" || idStr == "export" {
		return
	}
	id, _ := strconv.Atoi(idStr)

	switch r.Method {
	case "GET":
		p := scanProject(database.DB.QueryRow("SELECT * FROM projects WHERE id=? AND is_deleted=0", id))
		json.NewEncoder(w).Encode(p)
	case "PUT":
		h.updateProject(w, r)
	case "DELETE":
		database.DB.Exec("UPDATE projects SET is_deleted=1, updated_at=CURRENT_TIMESTAMP WHERE id=?", id)
		json.NewEncoder(w).Encode(map[string]string{"ok": "deleted"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) listProjects(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	where := "WHERE is_deleted=0"
	args := []interface{}{}

	// status=all 表示显式请求全部状态（不加状态条件，也不套用下面的默认过滤）
	statusParam := q.Get("status")
	allStatuses := statusParam == "all"

	for _, f := range []struct{ param, col string }{
		{"status", "project_status"},
		{"type", "project_type"},
		{"country", "country"},
		{"domestic_overseas", "domestic_overseas"},
	} {
		v := q.Get(f.param)
		if v == "" || (f.param == "status" && allStatuses) {
			continue
		}
		where += " AND " + f.col + "=?"
		args = append(args, v)
	}

	if kw := q.Get("keyword"); kw != "" {
		pat := "%" + kw + "%"
		where += " AND (short_name LIKE ? OR project_name LIKE ? OR contract_no LIKE ?)"
		args = append(args, pat, pat, pat)
	}

	// Default: only 在建 + 未开工 when no status filter (status=all 跳过该默认)
	if statusParam == "" {
		where += " AND project_status IN ('在建','未开工')"
	}

	rows, err := database.DB.Query(
		"SELECT * FROM projects "+where+" ORDER BY CASE project_status WHEN '在建' THEN 0 WHEN '未开工' THEN 1 ELSE 2 END, country ASC",
		args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		if p := scanProject(rows); p != nil {
			projects = append(projects, *p)
		}
	}

	// Distinct countries for filter dropdown
	countrySet := map[string]bool{}
	if dr, _ := database.DB.Query("SELECT DISTINCT country FROM projects WHERE is_deleted=0 AND country != ''"); dr != nil {
		defer dr.Close()
		for dr.Next() {
			var c string
			dr.Scan(&c)
			countrySet[c] = true
		}
	}
	countries := make([]string, 0, len(countrySet))
	for c := range countrySet {
		countries = append(countries, c)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{"projects": projects, "countries": countries})
}

func (h *Handler) createProject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		models.Project
		GPSInput string `json:"gps_input"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	p := body.Project
	parseGPS(body.GPSInput, &p)

	_, err := database.DB.Exec(
		"INSERT INTO projects ("+projectsInsertCols()+") VALUES ("+placeholders(82)+")",
		projectsInsertVals(&p)...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"ok": "created"})
}

func (h *Handler) updateProject(w http.ResponseWriter, r *http.Request) {
	var body struct {
		models.Project
		GPSInput string `json:"gps_input"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	p := body.Project
	parseGPS(body.GPSInput, &p)

	_, err := database.DB.Exec(
		"UPDATE projects SET "+updateSetCols()+" updated_at=CURRENT_TIMESTAMP WHERE id=?",
		append(projectsInsertVals(&p), p.ID)...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"ok": "updated"})
}

// ImportProjects imports Excel and applies conflict resolution.
func (h *Handler) ImportProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseMultipartForm(32 << 20)
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "No file uploaded", http.StatusBadRequest)
		return
	}
	defer file.Close()

	tmpPath := filepath.Join(os.TempDir(), "opims_import.xlsx")
	dst, _ := os.Create(tmpPath)
	io.Copy(dst, file)
	dst.Close()
	defer os.Remove(tmpPath)

	conflictMode := r.FormValue("conflict")
	if conflictMode == "" {
		conflictMode = "skip"
	}

	projects, err := services.ParseExcel(tmpPath)
	if err != nil {
		http.Error(w, "Parse error: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Filter overseas only
	var overseas []models.Project
	for _, p := range projects {
		if p.DomesticOverseas == "境外" {
			overseas = append(overseas, p)
		}
	}

	nameMap := loadNameMapping(h.root.Get())
	imported, skipped := 0, 0
	type conflict struct {
		ShortName string             `json:"short_name"`
		Existing  models.Project     `json:"existing"`
		Incoming  models.Project     `json:"incoming"`
		Diffs     []map[string]string `json:"diffs"`
	}
	var conflictList []conflict

	for _, p := range overseas {
		if mapped, ok := nameMap[p.ProjectName]; ok {
			p.ShortName = mapped.ShortName
			p.ContractNo = mapped.ContractNo
		}
		if p.ShortName == "" {
			p.ShortName = p.ProjectName
		}

		var existing models.Project
		err := database.DB.QueryRow("SELECT * FROM projects WHERE short_name=? AND is_deleted=0", p.ShortName).Scan(
			&existing.ID, &existing.ShortName, &existing.ContractNo, &existing.ProjectName,
			&existing.ProjectType, &existing.ProjectStatus, &existing.ImplementUnit,
			&existing.ContractAmount, &existing.BudgetAmount, &existing.ContractScope,
			&existing.KeyPoints, &existing.DomesticOverseas, &existing.Province, &existing.City,
			&existing.Address, &existing.Country,
			&existing.ContractStartYear, &existing.ContractStartMonth,
			&existing.ContractEndYear, &existing.ContractEndMonth, &existing.ContractDuration,
			&existing.ActualStartYear, &existing.ActualStartMonth,
			&existing.PlanEndYear, &existing.PlanEndMonth, &existing.ActualDuration,
			&existing.CompletionYear, &existing.CompletionMonth,
			&existing.RunningStatus, &existing.AbnormalReason, &existing.ProgressStatus,
			&existing.Issues, &existing.CompletedOutput, &existing.CompletePercent,
			&existing.ProgressSummary, &existing.CumReceivable, &existing.CumReceived,
			&existing.OwedAmount,
			&existing.GPSLat, &existing.GPSLng,
			&existing.PMContract, &existing.PMAppointed, &existing.PMOnsite, &existing.PMPhone,
			&existing.PMBuilder, &existing.PMSafetyCert,
			&existing.TechLeadAppointed, &existing.TechLeadOnsite, &existing.TechLeadPhone,
			&existing.TechLeadTitle,
			&existing.QualityMgrAppointed, &existing.QualityMgrOnsite, &existing.QualityMgrPhone,
			&existing.QualityMgrCert,
			&existing.HSEAppointed, &existing.HSEOnsite, &existing.HSEPhone, &existing.HSECert,
			&existing.CostMgrAppointed, &existing.CostMgrOnsite, &existing.CostMgrPhone,
			&existing.CostMgrCert,
			&existing.QualityKeyProcess, &existing.QualityMeasures,
			&existing.SafetyCost, &existing.SafetyCostSpent, &existing.SafetyCostCum,
			&existing.SafetyMajorHazard, &existing.SafetyHazardMeasure,
			&existing.SafetyRiskSource, &existing.SafetyRiskMeasure,
			&existing.OwnerUnit, &existing.OwnerContact, &existing.OwnerPhone,
			&existing.DesignUnit, &existing.DesignContact, &existing.DesignPhone,
			&existing.SupervisionUnit, &existing.SupervisionContact, &existing.SupervisionPhone,
			&existing.Reporter,
			&existing.PersonnelMgmt, &existing.PersonnelLabor,
			&existing.IsDeleted,
		)

		if err == nil {
			// Conflict detected
			switch conflictMode {
			case "skip":
				skipped++
				continue
			case "overwrite":
				database.DB.Exec("DELETE FROM projects WHERE id=?", existing.ID)
			case "keep_both":
				// Generate unique short name with suffix
				suffix := 1
				origShort := p.ShortName
				for {
					var c int
					database.DB.QueryRow("SELECT COUNT(*) FROM projects WHERE short_name=?", p.ShortName).Scan(&c)
					if c == 0 {
						break
					}
					suffix++
					p.ShortName = origShort + "_" + strconv.Itoa(suffix)
				}
			}

			// Build field-level diffs for conflict reporting
			if conflictMode == "skip" || conflictMode == "keep_both" {
				var diffs []map[string]string
				for _, pair := range []struct{ field, a, b string }{
					{"project_name", existing.ProjectName, p.ProjectName},
					{"contract_amount", formatFloat(existing.ContractAmount), formatFloat(p.ContractAmount)},
					{"project_status", existing.ProjectStatus, p.ProjectStatus},
					{"country", existing.Country, p.Country},
				} {
					if pair.a != pair.b {
						diffs = append(diffs, map[string]string{"field": pair.field, "old": pair.a, "new": pair.b})
					}
				}
				conflictList = append(conflictList, conflict{
					ShortName: existing.ShortName,
					Existing:  existing,
					Incoming:  p,
					Diffs:     diffs,
				})
				if conflictMode == "skip" {
					continue
				}
			}
		}

		_, err = database.DB.Exec(
			"INSERT INTO projects ("+projectsInsertCols()+") VALUES ("+placeholders(82)+")",
			projectsInsertVals(&p)...)
		if err != nil {
			skipped++
			continue
		}
		imported++
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"imported": imported, "skipped": skipped, "conflicts": conflictList,
	})
}

// formatFloat formats a float64 to string for comparison display.
func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}

// ExportProjects exports current filter results to .xlsx.
func (h *Handler) ExportProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	where := "WHERE is_deleted=0"
	args := []interface{}{}

	if v := q.Get("status"); v != "" {
		where += " AND project_status=?"
		args = append(args, v)
	}
	if v := q.Get("type"); v != "" {
		where += " AND project_type=?"
		args = append(args, v)
	}
	if v := q.Get("country"); v != "" {
		where += " AND country=?"
		args = append(args, v)
	}

	rows, err := database.DB.Query("SELECT * FROM projects "+where+" ORDER BY id", args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		if p := scanProject(rows); p != nil {
			projects = append(projects, *p)
		}
	}

	data, err := services.ExportExcel(projects, false)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=projects_export.xlsx")
	w.Write(data)
}

// updateSetCols returns "col=?," pairs for the UPDATE clause.
func updateSetCols() string {
	cols := strings.Split(projectsInsertCols(), ",")
	for i, c := range cols {
		cols[i] = c + "=?"
	}
	return strings.Join(cols, ",") + ","
}
