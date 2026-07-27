package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"opims/database"
	"opims/data"
	"opims/models"
	"opims/services"
	"os"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// SubcontractList handles GET (list) / POST (create).
func (h *Handler) SubcontractList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		q := r.URL.Query()
		where := "WHERE 1=1"
		args := []interface{}{}

		for _, f := range []struct{ param, col string }{
			{"project", "ps.project_short_name"},
			{"sub_name", "ps.sub_name"},
			{"tier", "ps.sub_tier"},
			{"category", "ps.profession_category"},
			{"profession", "ps.standardized_profession"},
		} {
			if v := q.Get(f.param); v != "" {
				where += " AND " + f.col + "=?"
				args = append(args, v)
			}
		}
		if kw := q.Get("keyword"); kw != "" {
			where += " AND (ps.contract_no LIKE ? OR ps.sub_name LIKE ?)"
			like := "%" + kw + "%"
			args = append(args, like, like)
		}

		// 账期（月份）快照：收集所有可选账期（倒序），默认取最新账期；
		// period=all 表示跨账期汇总，同一 contract_no 只保留最新账期的那条。
		periods := []string{}
		if prows, perr := database.DB.Query("SELECT DISTINCT period FROM project_subcontract WHERE period!='' ORDER BY period DESC"); perr == nil {
			for prows.Next() {
				var p string
				prows.Scan(&p)
				periods = append(periods, p)
			}
			prows.Close()
		}
		period := q.Get("period")
		if period == "" {
			if len(periods) > 0 {
				period = periods[0] // 默认最新账期
			}
		}
		if period == "all" {
			where += ` AND NOT EXISTS (SELECT 1 FROM project_subcontract x
				WHERE x.contract_no=ps.contract_no AND x.contract_no!='' AND x.period > ps.period)`
		} else {
			where += " AND ps.period=?"
			args = append(args, period)
		}

		orderBy := "ps.id DESC"
		if q.Get("dim") == "sub" {
			orderBy = "ps.sub_name ASC, ps.id DESC"
		}

		rows, err := database.DB.Query(
			`SELECT ps.id, ps.period, ps.seq_no, ps.branch_company, ps.project_name, ps.main_contract_amount,
			        ps.sub_name, ps.sub_tier, ps.sub_profession_raw, ps.sub_contract_profession,
			        ps.sub_controller, ps.sub_controller_phone, ps.contract_no, ps.contract_name,
			        ps.contract_amount, ps.supplement_amount, ps.contract_date, ps.progress_percent,
			        ps.entry_date, ps.exit_date, ps.evaluation_completed, ps.personnel_count,
			        ps.site_leader, ps.site_leader_approved, ps.site_leader_status,
			        ps.tech_leader, ps.tech_leader_approved, ps.tech_leader_status,
			        ps.safety_officer, ps.safety_officer_approved, ps.safety_officer_status,
			        ps.contract_compliance, ps.noncompliance_note, ps.remarks,
			        ps.project_short_name, ps.standardized_profession, ps.profession_category,
			        COALESCE((SELECT 1 FROM subcontractor_blacklist WHERE sub_full_name=ps.sub_name AND status='列入中' LIMIT 1), 0) as blacklisted
			 FROM project_subcontract ps `+where+" ORDER BY "+orderBy, args...)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var records []models.SubcontractRecord
		var totalAmount float64
		for rows.Next() {
			var rec models.SubcontractRecord
			rows.Scan(&rec.ID, &rec.Period, &rec.SeqNo, &rec.BranchCompany, &rec.ProjectName, &rec.MainContractAmount,
				&rec.SubName, &rec.SubTier, &rec.SubProfessionRaw, &rec.SubContractProfession,
				&rec.SubController, &rec.SubControllerPhone, &rec.ContractNo, &rec.ContractName,
				&rec.ContractAmount, &rec.SupplementAmount, &rec.ContractDate, &rec.ProgressPercent,
				&rec.EntryDate, &rec.ExitDate, &rec.EvaluationCompleted, &rec.PersonnelCount,
				&rec.SiteLeader, &rec.SiteLeaderApproved, &rec.SiteLeaderStatus,
				&rec.TechLeader, &rec.TechLeaderApproved, &rec.TechLeaderStatus,
				&rec.SafetyOfficer, &rec.SafetyOfficerApproved, &rec.SafetyOfficerStatus,
				&rec.ContractCompliance, &rec.NoncomplianceNote, &rec.Remarks,
				&rec.ProjectShortName, &rec.StandardizedProfession, &rec.ProfessionCategory,
				&rec.Blacklisted)
			records = append(records, rec)
			totalAmount += rec.ContractAmount
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"records":      records,
			"total_count":  len(records),
			"total_amount": totalAmount,
			"periods":      periods,
			"period":       period,
		})

	case "POST":
		var rec models.SubcontractRecord
		if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		rec.StandardizedProfession = data.MapProfession(rec.SubProfessionRaw)
		rec.ProfessionCategory = data.CategoryOf(rec.StandardizedProfession)

		res, err := database.DB.Exec(
			`INSERT INTO project_subcontract (period,seq_no,branch_company,project_name,main_contract_amount,
			 sub_name,sub_tier,sub_profession_raw,sub_contract_profession,sub_controller,sub_controller_phone,
			 contract_no,contract_name,contract_amount,supplement_amount,contract_date,progress_percent,
			 entry_date,exit_date,evaluation_completed,personnel_count,
			 site_leader,site_leader_approved,site_leader_status,
			 tech_leader,tech_leader_approved,tech_leader_status,
			 safety_officer,safety_officer_approved,safety_officer_status,
			 contract_compliance,noncompliance_note,remarks,
			 project_short_name,standardized_profession,profession_category)
			 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			rec.Period, rec.SeqNo, rec.BranchCompany, rec.ProjectName, rec.MainContractAmount,
			rec.SubName, rec.SubTier, rec.SubProfessionRaw, rec.SubContractProfession,
			rec.SubController, rec.SubControllerPhone,
			rec.ContractNo, rec.ContractName, rec.ContractAmount, rec.SupplementAmount,
			rec.ContractDate, rec.ProgressPercent,
			rec.EntryDate, rec.ExitDate, rec.EvaluationCompleted, rec.PersonnelCount,
			rec.SiteLeader, rec.SiteLeaderApproved, rec.SiteLeaderStatus,
			rec.TechLeader, rec.TechLeaderApproved, rec.TechLeaderStatus,
			rec.SafetyOfficer, rec.SafetyOfficerApproved, rec.SafetyOfficerStatus,
			rec.ContractCompliance, rec.NoncomplianceNote, rec.Remarks,
			rec.ProjectShortName, rec.StandardizedProfession, rec.ProfessionCategory)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		id, _ := res.LastInsertId()
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": "created", "id": id})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// SubcontractByID handles GET / PUT / DELETE.
func (h *Handler) SubcontractByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/api/subcontract/"))

	switch r.Method {
	case "GET":
		var rec models.SubcontractRecord
		err := database.DB.QueryRow(
			`SELECT id, seq_no, branch_company, project_name, main_contract_amount,
			        sub_name, sub_tier, sub_profession_raw, sub_contract_profession,
			        sub_controller, sub_controller_phone, contract_no, contract_name,
			        contract_amount, supplement_amount, contract_date, progress_percent,
			        entry_date, exit_date, evaluation_completed, personnel_count,
			        site_leader, site_leader_approved, site_leader_status,
			        tech_leader, tech_leader_approved, tech_leader_status,
			        safety_officer, safety_officer_approved, safety_officer_status,
			        contract_compliance, noncompliance_note, remarks,
			        project_short_name, standardized_profession, profession_category, is_deleted,
			        COALESCE((SELECT 1 FROM subcontractor_blacklist WHERE sub_full_name=ps.sub_name AND status='列入中' LIMIT 1), 0) as blacklisted
			 FROM project_subcontract ps WHERE id=?`, id,
		).Scan(&rec.ID, &rec.SeqNo, &rec.BranchCompany, &rec.ProjectName, &rec.MainContractAmount,
			&rec.SubName, &rec.SubTier, &rec.SubProfessionRaw, &rec.SubContractProfession,
			&rec.SubController, &rec.SubControllerPhone, &rec.ContractNo, &rec.ContractName,
			&rec.ContractAmount, &rec.SupplementAmount, &rec.ContractDate, &rec.ProgressPercent,
			&rec.EntryDate, &rec.ExitDate, &rec.EvaluationCompleted, &rec.PersonnelCount,
			&rec.SiteLeader, &rec.SiteLeaderApproved, &rec.SiteLeaderStatus,
			&rec.TechLeader, &rec.TechLeaderApproved, &rec.TechLeaderStatus,
			&rec.SafetyOfficer, &rec.SafetyOfficerApproved, &rec.SafetyOfficerStatus,
			&rec.ContractCompliance, &rec.NoncomplianceNote, &rec.Remarks,
			&rec.ProjectShortName, &rec.StandardizedProfession, &rec.ProfessionCategory, &rec.IsDeleted,
			&rec.Blacklisted)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(rec)

	case "PUT":
		var rec models.SubcontractRecord
		if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		rec.StandardizedProfession = data.MapProfession(rec.SubProfessionRaw)
		rec.ProfessionCategory = data.CategoryOf(rec.StandardizedProfession)

		_, err := database.DB.Exec(
			`UPDATE project_subcontract SET seq_no=?,branch_company=?,project_name=?,main_contract_amount=?,
			 sub_name=?,sub_tier=?,sub_profession_raw=?,sub_contract_profession=?,sub_controller=?,sub_controller_phone=?,
			 contract_no=?,contract_name=?,contract_amount=?,supplement_amount=?,contract_date=?,progress_percent=?,
			 entry_date=?,exit_date=?,evaluation_completed=?,personnel_count=?,
			 site_leader=?,site_leader_approved=?,site_leader_status=?,
			 tech_leader=?,tech_leader_approved=?,tech_leader_status=?,
			 safety_officer=?,safety_officer_approved=?,safety_officer_status=?,
			 contract_compliance=?,noncompliance_note=?,remarks=?,
			 project_short_name=?,standardized_profession=?,profession_category=?,
			 updated_at=CURRENT_TIMESTAMP WHERE id=?`,
			rec.SeqNo, rec.BranchCompany, rec.ProjectName, rec.MainContractAmount,
			rec.SubName, rec.SubTier, rec.SubProfessionRaw, rec.SubContractProfession,
			rec.SubController, rec.SubControllerPhone,
			rec.ContractNo, rec.ContractName, rec.ContractAmount, rec.SupplementAmount,
			rec.ContractDate, rec.ProgressPercent,
			rec.EntryDate, rec.ExitDate, rec.EvaluationCompleted, rec.PersonnelCount,
			rec.SiteLeader, rec.SiteLeaderApproved, rec.SiteLeaderStatus,
			rec.TechLeader, rec.TechLeaderApproved, rec.TechLeaderStatus,
			rec.SafetyOfficer, rec.SafetyOfficerApproved, rec.SafetyOfficerStatus,
			rec.ContractCompliance, rec.NoncomplianceNote, rec.Remarks,
			rec.ProjectShortName, rec.StandardizedProfession, rec.ProfessionCategory, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"ok": "updated"})

	case "DELETE":
		database.DB.Exec("DELETE FROM project_subcontract WHERE id=?", id)
		json.NewEncoder(w).Encode(map[string]string{"ok": "deleted"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// SubcontractImport handles POST multipart upload of the management ledger Excel.
func (h *Handler) SubcontractImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	tmpPath, err := saveUploadFile(r, "file")
	if err != nil {
		http.Error(w, fmt.Sprintf("upload failed: %v", err), http.StatusBadRequest)
		return
	}
	defer os.Remove(tmpPath)

	// 账期（年月）由导入弹窗手动选择，台账文件名不一定带日期。
	period := strings.TrimSpace(r.FormValue("period"))
	if period == "" {
		http.Error(w, "请选择账期（年月，如 2026-06）", http.StatusBadRequest)
		return
	}

	records, err := services.ParseSubcontractExcel(tmpPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("parse failed: %v", err), http.StatusBadRequest)
		return
	}

	// Refuse to wipe existing data when the file parsed to nothing.
	if len(records) == 0 {
		http.Error(w, "解析结果为空，未替换现有数据（请检查文件是否为管理台账）", http.StatusBadRequest)
		return
	}

	// 按账期分快照：只覆盖本账期的数据，其它月份不受影响。整个过程放在一个事务里，
	// 出错则回滚，避免坏文件破坏该账期的既有数据。
	tx, err := database.DB.Begin()
	if err != nil {
		http.Error(w, fmt.Sprintf("begin tx: %v", err), http.StatusInternalServerError)
		return
	}

	// 记录本账期覆盖前已存在的合同编号，用于统计"新增/更新"。
	existing := map[string]bool{}
	if erows, e := tx.Query("SELECT contract_no FROM project_subcontract WHERE period=? AND contract_no!=''", period); e == nil {
		for erows.Next() {
			var c string
			erows.Scan(&c)
			existing[c] = true
		}
		erows.Close()
	}

	if _, err := tx.Exec("DELETE FROM project_subcontract WHERE period=?", period); err != nil {
		tx.Rollback()
		http.Error(w, fmt.Sprintf("clear failed: %v", err), http.StatusInternalServerError)
		return
	}

	stmt, err := tx.Prepare(
		`INSERT INTO project_subcontract (period,seq_no,branch_company,project_name,main_contract_amount,
		 sub_name,sub_tier,sub_profession_raw,sub_contract_profession,sub_controller,sub_controller_phone,
		 contract_no,contract_name,contract_amount,supplement_amount,contract_date,progress_percent,
		 entry_date,exit_date,evaluation_completed,personnel_count,
		 site_leader,site_leader_approved,site_leader_status,
		 tech_leader,tech_leader_approved,tech_leader_status,
		 safety_officer,safety_officer_approved,safety_officer_status,
		 contract_compliance,noncompliance_note,remarks,
		 project_short_name,standardized_profession,profession_category)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		tx.Rollback()
		http.Error(w, fmt.Sprintf("prepare failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	// Match project_short_name against projects table (best-effort).
	nameToShort := loadProjectNameMapping()

	added, updated := 0, 0
	var errs []string
	for _, rec := range records {
		if strings.TrimSpace(rec.ContractNo) == "" {
			errs = append(errs, fmt.Sprintf("行[%s]：缺分包合同编号，已跳过", rec.SubName))
			continue
		}
		if rec.ProjectName != "" {
			rec.ProjectShortName = matchProject(rec.ProjectName, nameToShort)
		}
		_, err := stmt.Exec(
			period, rec.SeqNo, rec.BranchCompany, rec.ProjectName, rec.MainContractAmount,
			rec.SubName, rec.SubTier, rec.SubProfessionRaw, rec.SubContractProfession,
			rec.SubController, rec.SubControllerPhone,
			rec.ContractNo, rec.ContractName, rec.ContractAmount, rec.SupplementAmount,
			rec.ContractDate, rec.ProgressPercent,
			rec.EntryDate, rec.ExitDate, rec.EvaluationCompleted, rec.PersonnelCount,
			rec.SiteLeader, rec.SiteLeaderApproved, rec.SiteLeaderStatus,
			rec.TechLeader, rec.TechLeaderApproved, rec.TechLeaderStatus,
			rec.SafetyOfficer, rec.SafetyOfficerApproved, rec.SafetyOfficerStatus,
			rec.ContractCompliance, rec.NoncomplianceNote, rec.Remarks,
			rec.ProjectShortName, rec.StandardizedProfession, rec.ProfessionCategory)
		if err != nil {
			errs = append(errs, fmt.Sprintf("行[%s]：%v", rec.SubName, err))
			continue
		}
		if existing[rec.ContractNo] {
			updated++
		} else {
			added++
		}
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, fmt.Sprintf("commit failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"period":   period,
		"added":    added,
		"updated":  updated,
		"skipped":  len(errs),
		"imported": added + updated,
		"errors":   errs,
	})
}

// SubcontractExport exports current filtered records as .xlsx.
func (h *Handler) SubcontractExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	where := "WHERE 1=1"
	args := []interface{}{}
	for _, f := range []struct{ param, col string }{
		{"project", "ps.project_short_name"},
		{"sub_name", "ps.sub_name"},
		{"tier", "ps.sub_tier"},
		{"category", "ps.profession_category"},
		{"profession", "ps.standardized_profession"},
	} {
		if v := q.Get(f.param); v != "" {
			where += " AND " + f.col + "=?"
			args = append(args, v)
		}
	}
	if kw := q.Get("keyword"); kw != "" {
		where += " AND (ps.contract_no LIKE ? OR ps.sub_name LIKE ?)"
		like := "%" + kw + "%"
		args = append(args, like, like)
	}

	rows, err := database.DB.Query(
		`SELECT seq_no,branch_company,project_name,sub_name,sub_tier,
		        sub_profession_raw,standardized_profession,profession_category,
		        sub_contract_profession,sub_controller,sub_controller_phone,
		        contract_no,contract_name,contract_amount,supplement_amount,
		        contract_date,progress_percent,entry_date,exit_date,
		        evaluation_completed,personnel_count,
		        site_leader,site_leader_approved,site_leader_status,
		        tech_leader,tech_leader_approved,tech_leader_status,
		        safety_officer,safety_officer_approved,safety_officer_status,
		        contract_compliance,noncompliance_note,remarks,project_short_name
		 FROM project_subcontract ps `+where+" ORDER BY ps.id", args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	f := excelize.NewFile()
	sheet := "Sheet1"
	// Headers
	headers := []string{"序号","分公司","项目名称","分包商名称","层级","原始专业","标准化专业","专业分类","合同专业","实控人","实控人电话","合同编号","合同名称","合同额","补充协议金额","合同日期","施工状态","进场时间","撤场时间","完工评价","人员数","现场负责人","审批负责人","负责人状态","技术负责人","审批技术","技术状态","安全员","审批安全员","安全员状态","是否一致","不一致说明","备注","项目简称"}
	for i, h := range headers {
		f.SetCellValue(sheet, cellName(i+1, 1), h)
	}
	rowIdx := 2
	for rows.Next() {
		var vals [35]string
		var v0, v1, v2, v3, v4, v5, v6, v7, v8, v9, v10, v11 string
		var v12, v13, v14, v15, v16, v17, v18 float64
		var v19 int
		var v20, v21, v22, v23, v24, v25, v26, v27, v28, v29, v30, v31, v32, v33 string
		rows.Scan(&v0, &v1, &v2, &v3, &v4, &v5, &v6, &v7, &v8, &v9, &v10,
			&v11, &v12, &v13, &v14, &v15, &v16, &v17, &v18, &v19,
			&v20, &v21, &v22, &v23, &v24, &v25, &v26, &v27, &v28,
			&v29, &v30, &v31, &v32, &v33)
		vals = [35]string{v0, v1, v2, v3, v4, v5, v6, v7, v8, v9, v10,
			v11, fmt.Sprintf("%.2f", v12), fmt.Sprintf("%.2f", v13),
			fmt.Sprintf("%.2f", v14), fmt.Sprintf("%.2f", v15),
			fmt.Sprintf("%.2f", v16), fmt.Sprintf("%.2f", v17),
			fmt.Sprintf("%.2f", v18), fmt.Sprintf("%d", v19),
			v20, v21, v22, v23, v24, v25, v26, v27, v28,
			v29, v30, v31, v32, v33}
		for i, v := range vals {
			f.SetCellValue(sheet, cellName(i+1, rowIdx), v)
		}
		rowIdx++
	}

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=subcontract_export.xlsx")
	buf, _ := f.WriteToBuffer()
	w.Write(buf.Bytes())
}

// saveUploadFile saves a multipart upload to a temp file.
func saveUploadFile(r *http.Request, field string) (string, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, 50<<20)
	mr, err := r.MultipartReader()
	if err != nil {
		return "", err
	}
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if part.FormName() == field {
			tmpFile, err := os.CreateTemp("", "opims-subcontract-*.xlsx")
			if err != nil {
				return "", err
			}
			defer tmpFile.Close()
			io.Copy(tmpFile, part)
			return tmpFile.Name(), nil
		}
	}
	return "", fmt.Errorf("field %s not found in upload", field)
}

func cellName(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}

// loadProjectNameMapping loads project_name → short_name from the projects table.
func loadProjectNameMapping() map[string]string {
	m := map[string]string{}
	rows, err := database.DB.Query("SELECT project_name, short_name FROM projects WHERE is_deleted=0")
	if err != nil {
		return m
	}
	defer rows.Close()
	for rows.Next() {
		var name, short string
		rows.Scan(&name, &short)
		m[name] = short
	}
	return m
}

// matchProject finds the best-matching project short_name for a given name from the ledger.
func matchProject(name string, db map[string]string) string {
	if short, ok := db[name]; ok {
		return short
	}
	for pn, short := range db {
		if strings.Contains(name, pn) || strings.Contains(pn, name) {
		return short
		}
	}
	return ""
}
