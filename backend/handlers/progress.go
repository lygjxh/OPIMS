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
	json.NewEncoder(w).Encode(res)
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
