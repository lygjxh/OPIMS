package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"opims/database"
	"opims/services"
	"time"
)

// 进度管理 · 报送核查（一期 1a）
// 依据《OPIMS 进度管理模块 需求说明书 V1.1》4.2.2–4.2.4。

// ProgressCheck GET /api/progress/check?period=YYYY-MM
// 扫描各在建项目的报送情况，返回看板数据。period 缺省为上一个月
// （通常在次月初核查上月报送，默认上月更贴合实际使用）。
func (h *Handler) ProgressCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	period := r.URL.Query().Get("period")
	if period == "" {
		period = time.Now().AddDate(0, -1, 0).Format("2006-01")
	}

	projects, err := activeProjects()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(projects) == 0 {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"period": period, "rows": []interface{}{},
			"warning": "没有「在建」状态的项目，报送核查仅针对在建项目",
		})
		return
	}

	res, err := services.CheckSubmissions(h.root.ProjectsDir(), projects, period)
	if err != nil {
		if errors.Is(err, services.ErrProjectsDirMissing) {
			writeJSONError(w, http.StatusNotFound,
				"未找到项目文件目录，请确认根目录设置正确，且根目录下存在 01.Project Files 文件夹",
				h.root.ProjectsDir())
			return
		}
		writeJSONError(w, http.StatusBadRequest, err.Error(), "")
		return
	}
	// 叠加人工复核：系统按文件判定难免有偏差，复核结果优先
	services.ApplyReviews(res, services.LoadReviews(res.Period))
	json.NewEncoder(w).Encode(res)
}

// ProgressReview POST /api/progress/review —— 保存人工复核（修正判定 + 备注）。
func (h *Handler) ProgressReview(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var rv services.Review
	if err := json.NewDecoder(r.Body).Decode(&rv); err != nil {
		writeJSONError(w, http.StatusBadRequest, "请求体解析失败: "+err.Error(), "")
		return
	}
	if err := services.SaveReview(rv); err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error(), "")
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"ok": "saved"})
}

// ProgressSnapshot POST /api/progress/snapshot?period=YYYY-MM
// 将本期核查结果（含人工复核）存为快照，作为年度考核的客观数据源。
func (h *Handler) ProgressSnapshot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	period := r.URL.Query().Get("period")
	if period == "" {
		period = time.Now().AddDate(0, -1, 0).Format("2006-01")
	}
	projects, err := activeProjects()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	res, err := services.CheckSubmissions(h.root.ProjectsDir(), projects, period)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error(), "")
		return
	}
	services.ApplyReviews(res, services.LoadReviews(res.Period))

	n, err := services.TakeSnapshot(res)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "保存快照失败: "+err.Error(), "")
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok": "saved", "period": res.Period, "records": n,
	})
}

// ProgressCompliance GET /api/progress/compliance?year=YYYY —— 年度报送合规率。
func (h *Handler) ProgressCompliance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	year := r.URL.Query().Get("year")
	if year == "" {
		year = services.CurrentYear()
	}
	rep, err := services.Compliance(year)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error(), "")
		return
	}
	json.NewEncoder(w).Encode(rep)
}

// ProgressComplianceExport GET /api/progress/compliance/export?year=YYYY —— 导出 Excel。
func (h *Handler) ProgressComplianceExport(w http.ResponseWriter, r *http.Request) {
	year := r.URL.Query().Get("year")
	if year == "" {
		year = services.CurrentYear()
	}
	rep, err := services.Compliance(year)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	buf, err := services.ExportCompliance(rep)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=compliance_"+year+".xlsx")
	w.Write(buf)
}

// activeProjects 取「在建」项目。报送核查只针对在建项目——
// 库中完工项目占多数，全量生成会产生大量假缺交（需求 V1.1 决策 10）。
func activeProjects() ([]services.ProjectBrief, error) {
	rows, err := database.DB.Query(
		`SELECT short_name, country FROM projects
		 WHERE is_deleted=0 AND project_status='在建' AND short_name!=''
		 ORDER BY country, short_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []services.ProjectBrief
	for rows.Next() {
		var p services.ProjectBrief
		if err := rows.Scan(&p.ShortName, &p.Country); err != nil {
			continue
		}
		list = append(list, p)
	}
	return list, nil
}
