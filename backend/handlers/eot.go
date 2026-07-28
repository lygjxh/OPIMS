package handlers

import (
	"encoding/json"
	"net/http"
	"opims/services"
	"os"
	"time"
)

// EOTList GET /api/eot —— 工期索赔跟踪汇总（二期 2b）。
func (h *Handler) EOTList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	projects, err := activeProjects()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(projects) == 0 {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"items": []interface{}{}, "warnings": []string{},
			"summary": services.EOTSummary{},
			"warning": "没有「在建」状态的项目，索赔跟踪仅针对在建项目",
		})
		return
	}
	dir := h.root.ProjectsDir()
	if _, err := os.Stat(dir); err != nil {
		writeJSONError(w, http.StatusNotFound,
			"未找到项目文件目录，请确认根目录设置正确，且根目录下存在 01.Project Files 文件夹", dir)
		return
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	res := services.ScanEOT(dir, projects, today)
	if res.Items == nil {
		res.Items = []services.EOTItem{}
	}
	if res.Warnings == nil {
		res.Warnings = []string{}
	}
	json.NewEncoder(w).Encode(res)
}

// ProgressIndicators GET /api/progress/indicators?period=YYYY-MM
// 解析各项目附件 B-1，按细则 4.1 判定风险灯（二期 2c）。
func (h *Handler) ProgressIndicators(w http.ResponseWriter, r *http.Request) {
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
	dir := h.root.ProjectsDir()
	if _, err := os.Stat(dir); err != nil {
		writeJSONError(w, http.StatusNotFound,
			"未找到项目文件目录，请确认根目录设置正确，且根目录下存在 01.Project Files 文件夹", dir)
		return
	}

	res := services.ScanProgressData(dir, projects, period)
	if res.Items == nil {
		res.Items = []services.ProgressData{}
	}
	if res.Warnings == nil {
		res.Warnings = []string{}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"period": period, "items": res.Items,
		"warnings": res.Warnings, "summary": res.Summary,
		"project_count": len(projects),
	})
}
