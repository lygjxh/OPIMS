package handlers

import (
	"encoding/json"
	"net/http"
	"opims/services"
	"os"
	"time"
)

// TimeBarList GET /api/timebar —— 合同时效预警看板（二期 2a）。
// 汇总各在建项目的附件 E，按日历日倒计时，最紧急的排在最前。
func (h *Handler) TimeBarList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	projects, err := activeProjects()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(projects) == 0 {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"items": []interface{}{}, "warnings": []string{},
			"summary": services.TimeBarSummary{},
			"warning": "没有「在建」状态的项目，时效预警仅针对在建项目",
		})
		return
	}

	dir := h.root.ProjectsDir()
	if _, err := os.Stat(dir); err != nil {
		writeJSONError(w, http.StatusNotFound,
			"未找到项目文件目录，请确认根目录设置正确，且根目录下存在 01.Project Files 文件夹", dir)
		return
	}

	// 以当天零点为基准计算剩余天数，避免同一天内因时分不同得出不同结果
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	res := services.ScanTimeBars(dir, projects, today)
	if res.Items == nil {
		res.Items = []services.TimeBarItem{}
	}
	if res.Warnings == nil {
		res.Warnings = []string{}
	}
	json.NewEncoder(w).Encode(res)
}
