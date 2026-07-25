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

		orderBy := "ps.id DESC"
		if q.Get("dim") == "sub" {
			orderBy = "ps.sub_name ASC, ps.id DESC"
		}

		rows, err := database.DB.Query(
			`SELECT ps.id, ps.seq_no, ps.branch_company, ps.project_name, ps.main_contract_amount,
			        ps.sub_name, ps.sub_tier, ps.sub_profession_raw, ps.sub_contract_profession,
			        ps.sub_controller, ps.sub_controller_phone, ps.contract_no, ps.contract_name,
			        ps.contract_amount, ps.supplement_amount, ps.contract_date, ps.progress_percent,
			        ps.entry_date, ps.exit_date, ps.evaluation_completed, ps.personnel_count,
			        ps.site_leader, ps.site_leader_approved, ps.site_leader_status,
			        ps.tech_leader, ps.tech_leader_approved, ps.tech_leader_status,
			        ps.safety_officer, ps.safety_officer_approved, ps.safety_officer_status,
			        ps.contract_compliance, ps.noncompliance_note, ps.remarks,
			        ps.project_short_name, ps.standardized_profession, ps.profession_category
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
			rows.Scan(&rec.ID, &rec.SeqNo, &rec.BranchCompany, &rec.ProjectName, &rec.MainContractAmount,
				&rec.SubName, &rec.SubTier, &rec.SubProfessionRaw, &rec.SubContractProfession,
				&rec.SubController, &rec.SubControllerPhone, &rec.ContractNo, &rec.ContractName,
				&rec.ContractAmount, &rec.SupplementAmount, &rec.ContractDate, &rec.ProgressPercent,
				&rec.EntryDate, &rec.ExitDate, &rec.EvaluationCompleted, &rec.PersonnelCount,
				&rec.SiteLeader, &rec.SiteLeaderApproved, &rec.SiteLeaderStatus,
				&rec.TechLeader, &rec.TechLeaderApproved, &rec.TechLeaderStatus,
				&rec.SafetyOfficer, &rec.SafetyOfficerApproved, &rec.SafetyOfficerStatus,
				&rec.ContractCompliance, &rec.NoncomplianceNote, &rec.Remarks,
				&rec.ProjectShortName, &rec.StandardizedProfession, &rec.ProfessionCategory)
			records = append(records, rec)
			totalAmount += rec.ContractAmount
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"records":      records,
			"total_count":  len(records),
			"total_amount": totalAmount,
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
			`INSERT INTO project_subcontract (seq_no,branch_company,project_name,main_contract_amount,
			 sub_name,sub_tier,sub_profession_raw,sub_contract_profession,sub_controller,sub_controller_phone,
			 contract_no,contract_name,contract_amount,supplement_amount,contract_date,progress_percent,
			 entry_date,exit_date,evaluation_completed,personnel_count,
			 site_leader,site_leader_approved,site_leader_status,
			 tech_leader,tech_leader_approved,tech_leader_status,
			 safety_officer,safety_officer_approved,safety_officer_status,
			 contract_compliance,noncompliance_note,remarks,
			 project_short_name,standardized_profession,profession_category)
			 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
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
			        project_short_name, standardized_profession, profession_category, is_deleted
			 FROM project_subcontract WHERE id=?`, id,
		).Scan(&rec.ID, &rec.SeqNo, &rec.BranchCompany, &rec.ProjectName, &rec.MainContractAmount,
			&rec.SubName, &rec.SubTier, &rec.SubProfessionRaw, &rec.SubContractProfession,
			&rec.SubController, &rec.SubControllerPhone, &rec.ContractNo, &rec.ContractName,
			&rec.ContractAmount, &rec.SupplementAmount, &rec.ContractDate, &rec.ProgressPercent,
			&rec.EntryDate, &rec.ExitDate, &rec.EvaluationCompleted, &rec.PersonnelCount,
			&rec.SiteLeader, &rec.SiteLeaderApproved, &rec.SiteLeaderStatus,
			&rec.TechLeader, &rec.TechLeaderApproved, &rec.TechLeaderStatus,
			&rec.SafetyOfficer, &rec.SafetyOfficerApproved, &rec.SafetyOfficerStatus,
			&rec.ContractCompliance, &rec.NoncomplianceNote, &rec.Remarks,
			&rec.ProjectShortName, &rec.StandardizedProfession, &rec.ProfessionCategory, &rec.IsDeleted)
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

	// Wipe + re-insert as one transaction: on a fatal error the old data is
	// left intact instead of being destroyed by a bad file.
	tx, err := database.DB.Begin()
	if err != nil {
		http.Error(w, fmt.Sprintf("begin tx: %v", err), http.StatusInternalServerError)
		return
	}
	if _, err := tx.Exec("DELETE FROM project_subcontract"); err != nil {
		tx.Rollback()
		http.Error(w, fmt.Sprintf("clear failed: %v", err), http.StatusInternalServerError)
		return
	}

	stmt, err := tx.Prepare(
		`INSERT INTO project_subcontract (seq_no,branch_company,project_name,main_contract_amount,
		 sub_name,sub_tier,sub_profession_raw,sub_contract_profession,sub_controller,sub_controller_phone,
		 contract_no,contract_name,contract_amount,supplement_amount,contract_date,progress_percent,
		 entry_date,exit_date,evaluation_completed,personnel_count,
		 site_leader,site_leader_approved,site_leader_status,
		 tech_leader,tech_leader_approved,tech_leader_status,
		 safety_officer,safety_officer_approved,safety_officer_status,
		 contract_compliance,noncompliance_note,remarks,
		 project_short_name,standardized_profession,profession_category)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		tx.Rollback()
		http.Error(w, fmt.Sprintf("prepare failed: %v", err), http.StatusInternalServerError)
		return
	}
	defer stmt.Close()

	imported := 0
	var errs []string
	for _, rec := range records {
		_, err := stmt.Exec(
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
			rec.ProjectShortName, rec.StandardizedProfession, rec.ProfessionCategory)
		if err != nil {
			errs = append(errs, fmt.Sprintf("row %s: %v", rec.SubName, err))
			continue
		}
		imported++
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, fmt.Sprintf("commit failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"imported": imported,
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
