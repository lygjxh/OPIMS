package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"opims/database"
	"opims/models"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// SubcontractorsList handles GET (list) / POST (create) for subcontractor library.
func (h *Handler) SubcontractorsList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		q := r.URL.Query()
		where := "WHERE 1=1"
		args := []interface{}{}

		for _, f := range []struct{ param, col string }{
			{"type", "registration_type"},
			{"country", "country"},
			{"category", "profession_category"},
			{"profession", "profession"},
			{"parent", "parent_short_name"},
		} {
			if v := q.Get(f.param); v != "" {
				where += " AND " + f.col + "=?"
				args = append(args, v)
			}
		}
		if kw := q.Get("keyword"); kw != "" {
			where += " AND (short_name LIKE ? OR full_name LIKE ?)"
			like := "%" + kw + "%"
			args = append(args, like, like)
		}

		rows, err := database.DB.Query(
			`SELECT sb.id, sb.short_name, sb.full_name, sb.registration_type, sb.country,
			        sb.profession_category, sb.profession, sb.other_professions,
			        sb.parent_short_name, sb.legal_rep_name, sb.legal_rep_id, sb.legal_rep_phone,
			        sb.contact_name, sb.contact_title, sb.contact_phone, sb.contact_email,
			        sb.biz_license, sb.tax_id, sb.reg_address, sb.reg_capital, sb.notes,
			        COALESCE((SELECT COUNT(*) FROM subcontractor_projects WHERE sub_short_name=sb.short_name), 0) as project_count,
			        COALESCE((SELECT SUM(contract_amount) FROM subcontractor_projects WHERE sub_short_name=sb.short_name), 0) as total_amount
			 FROM subcontractors_base sb `+where+" ORDER BY sb.id DESC", args...)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		type row struct {
			models.SubcontractorBase
			ProjectCount float64 `json:"project_count"`
			TotalAmount  float64 `json:"total_amount"`
		}
		var list []row
		for rows.Next() {
			var r row
			rows.Scan(&r.ID, &r.ShortName, &r.FullName, &r.RegistrationType, &r.Country,
				&r.ProfessionCategory, &r.Profession, &r.OtherProfessions,
				&r.ParentShortName, &r.LegalRepName, &r.LegalRepID, &r.LegalRepPhone,
				&r.ContactName, &r.ContactTitle, &r.ContactPhone, &r.ContactEmail,
				&r.BizLicense, &r.TaxID, &r.RegAddress, &r.RegCapital, &r.Notes,
				&r.ProjectCount, &r.TotalAmount)
			list = append(list, r)
		}
		json.NewEncoder(w).Encode(list)

	case "POST":
		var b models.SubcontractorBase
		json.NewDecoder(r.Body).Decode(&b)
		_, err := database.DB.Exec(
			`INSERT INTO subcontractors_base (short_name,full_name,registration_type,country,
			 profession_category,profession,other_professions,parent_short_name,
			 legal_rep_name,legal_rep_id,legal_rep_phone,
			 contact_name,contact_title,contact_phone,contact_email,
			 biz_license,tax_id,reg_address,reg_capital,notes)
			 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			b.ShortName, b.FullName, b.RegistrationType, b.Country,
			b.ProfessionCategory, b.Profession, b.OtherProfessions, b.ParentShortName,
			b.LegalRepName, b.LegalRepID, b.LegalRepPhone,
			b.ContactName, b.ContactTitle, b.ContactPhone, b.ContactEmail,
			b.BizLicense, b.TaxID, b.RegAddress, b.RegCapital, b.Notes)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"ok": "created"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// SubcontractorsByID handles GET / PUT / DELETE for a single subcontractor.
func (h *Handler) SubcontractorsByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/api/subcontractors/"))

	switch r.Method {
	case "GET":
		var b models.SubcontractorBase
		err := database.DB.QueryRow(
			`SELECT id, short_name, full_name, registration_type, country,
			        profession_category, profession, other_professions, parent_short_name,
			        legal_rep_name, legal_rep_id, legal_rep_phone,
			        contact_name, contact_title, contact_phone, contact_email,
			        biz_license, tax_id, reg_address, reg_capital, notes
			 FROM subcontractors_base WHERE id=?`, id,
		).Scan(&b.ID, &b.ShortName, &b.FullName, &b.RegistrationType, &b.Country,
			&b.ProfessionCategory, &b.Profession, &b.OtherProfessions, &b.ParentShortName,
			&b.LegalRepName, &b.LegalRepID, &b.LegalRepPhone,
			&b.ContactName, &b.ContactTitle, &b.ContactPhone, &b.ContactEmail,
			&b.BizLicense, &b.TaxID, &b.RegAddress, &b.RegCapital, &b.Notes)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		// Fetch cooperation history
		pRows, err := database.DB.Query(
			`SELECT project_short_name, project_name, start_date, end_date,
			        contract_no, contract_amount, scope, profession_category, profession,
			        other_professions, project_status
			 FROM subcontractor_projects WHERE sub_short_name=? ORDER BY start_date DESC`, b.ShortName)
		projects := []models.SubcontractorProject{}
		if err == nil {
			defer pRows.Close()
			for pRows.Next() {
				var sp models.SubcontractorProject
				pRows.Scan(&sp.ProjectShortName, &sp.ProjectName, &sp.StartDate, &sp.EndDate,
					&sp.ContractNo, &sp.ContractAmount, &sp.Scope,
					&sp.ProfessionCategory, &sp.Profession, &sp.OtherProfessions, &sp.ProjectStatus)
				projects = append(projects, sp)
			}
		}

		// Check blacklist status
		var blStatus, blLevel, blDate, blReason string
		database.DB.QueryRow(
			`SELECT status, restrict_level, list_date, list_reason FROM subcontractor_blacklist WHERE sub_full_name=? ORDER BY id DESC LIMIT 1`,
			b.FullName).Scan(&blStatus, &blLevel, &blDate, &blReason)

		// Check linked local subsidiaries blacklist status
		var localBlacklisted []string
		if b.RegistrationType == "国内" {
			lRows, _ := database.DB.Query(
				`SELECT sb.short_name FROM subcontractors_base sb
				 JOIN subcontractor_blacklist bl ON bl.sub_full_name=sb.full_name AND bl.status='列入中'
				 WHERE sb.parent_short_name=? AND sb.registration_type='当地注册'`, b.ShortName)
			if lRows != nil {
				defer lRows.Close()
				for lRows.Next() {
					var n string
					lRows.Scan(&n)
					localBlacklisted = append(localBlacklisted, n)
				}
			}
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"base":              b,
			"projects":          projects,
			"blacklist_status":  blStatus,
			"restrict_level":    blLevel,
			"list_date":         blDate,
			"list_reason":       blReason,
			"local_blacklisted": localBlacklisted,
		})

	case "PUT":
		var b models.SubcontractorBase
		json.NewDecoder(r.Body).Decode(&b)
		_, err := database.DB.Exec(
			`UPDATE subcontractors_base SET short_name=?,full_name=?,registration_type=?,country=?,
			 profession_category=?,profession=?,other_professions=?,parent_short_name=?,
			 legal_rep_name=?,legal_rep_id=?,legal_rep_phone=?,
			 contact_name=?,contact_title=?,contact_phone=?,contact_email=?,
			 biz_license=?,tax_id=?,reg_address=?,reg_capital=?,notes=?,
			 updated_at=CURRENT_TIMESTAMP WHERE id=?`,
			b.ShortName, b.FullName, b.RegistrationType, b.Country,
			b.ProfessionCategory, b.Profession, b.OtherProfessions, b.ParentShortName,
			b.LegalRepName, b.LegalRepID, b.LegalRepPhone,
			b.ContactName, b.ContactTitle, b.ContactPhone, b.ContactEmail,
			b.BizLicense, b.TaxID, b.RegAddress, b.RegCapital, b.Notes, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"ok": "updated"})

	case "DELETE":
		database.DB.Exec("DELETE FROM subcontractors_base WHERE id=?", id)
		json.NewEncoder(w).Encode(map[string]string{"ok": "deleted"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// SubcontractorsImport parses the 2-sheet Excel template and imports data.
// POST /api/subcontractors/import — multipart form with file field "file".
func (h *Handler) SubcontractorsImport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "file too large", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	f, err := excelize.OpenReader(file)
	if err != nil {
		http.Error(w, "invalid excel: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer f.Close()

	type importResult struct {
		Base    int `json:"base"`
		History int `json:"history"`
	}
	var result importResult
	var errors []map[string]string

	// ---- Sheet 1: 分包商库基础信息 ----
	if rows, err := f.GetRows(f.GetSheetName(0)); err == nil && len(rows) > 1 {
		// rows[0] = header, rows[1:] = data rows
		for i, row := range rows[1:] {
			// Skip empty rows
			if len(row) < 2 || strings.TrimSpace(row[0]) == "" {
				continue
			}
			name := func(j int) string {
				if j < len(row) {
					return strings.TrimSpace(row[j])
				}
				return ""
			}
			shortName := name(0)
			fullName := name(1)
			regType := name(2)
			country := name(3)
			cat := name(4)
			prof := name(5)
			otherProf := name(6)
			parent := name(7)
			legalName := name(8)
			legalID := name(9)
			legalPhone := name(10)
			contactName := name(11)
			contactTitle := name(12)
			contactPhone := name(13)
			contactEmail := name(14)
			bizLic := name(15)
			taxID := name(16)
			regAddr := name(17)
			regCap := name(18)
			notes := name(19)

			// Validate required fields
			missing := []string{}
			if shortName == "" {
				missing = append(missing, "分包商简称")
			}
			if fullName == "" {
				missing = append(missing, "分包商名称")
			}
			if regType == "" {
				missing = append(missing, "注册类型")
			}
			if country == "" {
				missing = append(missing, "国别")
			}
			if cat == "" {
				missing = append(missing, "主专业分类")
			}
			if prof == "" {
				missing = append(missing, "主专业")
			}
			if legalName == "" {
				missing = append(missing, "法代名称")
			}
			if legalPhone == "" {
				missing = append(missing, "法代电话")
			}
			if len(missing) > 0 {
				errors = append(errors, map[string]string{
					"row": fmt.Sprintf("Sheet1-%d", i+2), "field": strings.Join(missing, ","), "msg": "缺少必填字段",
				})
				continue
			}

			// Validate profession category
			catValid := false
			for _, c := range []string{"采购", "施工", "设计", "咨询"} {
				if cat == c {
					catValid = true
					break
				}
			}
			if !catValid {
				errors = append(errors, map[string]string{
					"row": fmt.Sprintf("Sheet1-%d", i+2), "field": "主专业分类", "msg": "必须是采购/施工/设计/咨询之一",
				})
				continue
			}

			_, err := database.DB.Exec(
				`INSERT INTO subcontractors_base
				 (short_name,full_name,registration_type,country,profession_category,profession,
				  other_professions,parent_short_name,legal_rep_name,legal_rep_id,legal_rep_phone,
				  contact_name,contact_title,contact_phone,contact_email,
				  biz_license,tax_id,reg_address,reg_capital,notes)
				 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
				shortName, fullName, regType, country, cat, prof, otherProf, parent,
				legalName, legalID, legalPhone,
				contactName, contactTitle, contactPhone, contactEmail,
				bizLic, taxID, regAddr, regCap, notes)
			if err != nil {
				errors = append(errors, map[string]string{
					"row": fmt.Sprintf("Sheet1-%d", i+2), "field": shortName, "msg": err.Error(),
				})
				continue
			}
			result.Base++
		}
	}

	// ---- Sheet 2: 合作历史补充 ----
	if f.SheetCount > 1 {
		if rows, err := f.GetRows(f.GetSheetName(1)); err == nil && len(rows) > 1 {
			for _, row := range rows[1:] {
				if len(row) < 2 || strings.TrimSpace(row[0]) == "" {
					continue
				}
				name := func(j int) string {
					if j < len(row) { return strings.TrimSpace(row[j]) }
					return ""
				}
				subShort := name(0)
				projShort := name(1)
				projName := name(2)
				startDate := name(3)
				endDate := name(4)
				contractNo := name(5)
				amtStr := name(6)
				scope := name(7)
				pCat := name(8)
				pProf := name(9)
				otherP := name(10)
				pStatus := name(11)
				pNotes := name(12)

				amt := 0.0
				if amtStr != "" {
					fmt.Sscanf(amtStr, "%f", &amt)
				}

				database.DB.Exec(
					`INSERT INTO subcontractor_projects
					 (sub_short_name,project_short_name,project_name,start_date,end_date,
					  contract_no,contract_amount,scope,profession_category,profession,
					  other_professions,project_status,is_manual,notes)
					 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,1,?)`,
					subShort, projShort, projName, startDate, endDate,
					contractNo, amt, scope, pCat, pProf, otherP, pStatus, pNotes)
				result.History++
			}
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"imported": result,
		"errors":   errors,
	})
}

// SubcontractorsExportTemplate generates an empty 2-sheet Excel template.
// GET /api/subcontractors/export/template
func (h *Handler) SubcontractorsExportTemplate(w http.ResponseWriter, r *http.Request) {
	f := excelize.NewFile()
	defer f.Close()

	// Sheet 1: 分包商库基础信息
	f.SetSheetName("Sheet1", "分包商库基础信息")
	headers1 := []string{
		"分包商简称", "分包商名称", "注册类型", "国别", "主专业分类", "主专业",
		"其他专业", "母公司简称", "法代名称", "法代身份证号", "法代电话",
		"联系人名称", "联系人职务", "联系人电话", "联系人邮箱",
		"营业执照号", "税号", "注册地址", "注册资本", "备注",
	}
	for i, h := range headers1 {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue("分包商库基础信息", cell, h)
	}

	// Sheet 2: 合作历史补充
	f.NewSheet("合作历史补充")
	headers2 := []string{
		"分包商简称", "项目简称", "项目名称", "开工日期", "完工日期",
		"合同号", "合同额（万元）", "承揽范围", "主专业分类", "主专业",
		"其他专业", "项目状态", "备注",
	}
	for i, h := range headers2 {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue("合作历史补充", cell, h)
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=分包商库导入模板.xlsx")
	f.Write(w)
}

// SubcontractorsExportProjects exports a subcontractor's project cooperation list.
// GET /api/subcontractors/:id/export-projects
func (h *Handler) SubcontractorsExportProjects(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	// Get the subcontractor's short_name
	var shortName string
	err = database.DB.QueryRow("SELECT short_name FROM subcontractors_base WHERE id=?", id).Scan(&shortName)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	rows, err := database.DB.Query(
		`SELECT project_short_name, project_name, start_date, end_date,
		        contract_no, contract_amount, scope, profession_category, profession,
		        other_professions, project_status
		 FROM subcontractor_projects WHERE sub_short_name=? ORDER BY start_date DESC`, shortName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	f := excelize.NewFile()
	defer f.Close()
	f.SetSheetName("Sheet1", "合作清单")

	headers := []string{
		"项目简称", "项目名称", "开工日期", "完工日期",
		"合同号", "合同额（万元）", "承揽范围", "主专业分类", "主专业",
		"其他专业", "项目状态",
	}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue("合作清单", cell, h)
	}

	idx := 2
	for rows.Next() {
		var projShort, projName, startDate, endDate, contractNo string
		var contractAmt float64
		var scope, pCat, pProf, otherP, pStatus string
		rows.Scan(&projShort, &projName, &startDate, &endDate,
			&contractNo, &contractAmt, &scope, &pCat, &pProf, &otherP, &pStatus)

		vals := []interface{}{projShort, projName, startDate, endDate, contractNo, contractAmt, scope, pCat, pProf, otherP, pStatus}
		for j, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(j+1, idx)
			f.SetCellValue("合作清单", cell, v)
		}
		idx++
	}

	fullName := shortName
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s_合作清单.xlsx", fullName))
	f.Write(w)
}
