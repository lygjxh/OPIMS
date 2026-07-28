package handlers

import (
	"encoding/json"
	"net/http"
	"opims/data"
	"opims/services"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// 进度管理 · 收文改名归档（一期 1c）
// ⚠️ 本组接口会写云盘文件，所有写操作都要求显式参数、不做默认覆盖。

// ArchiveInbox GET /api/archive/inbox?dir=xxx
// 列出待归档目录中的文件，并给出「项目/类型/周期」的猜测供界面预填。
func (h *Handler) ArchiveInbox(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	dir := strings.TrimSpace(r.URL.Query().Get("dir"))
	if dir == "" {
		// 默认收文箱：根目录下的 06.Received File，与现有目录约定一致
		dir = filepath.Join(h.root.Get(), "06.Received File")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		writeJSONError(w, http.StatusNotFound, "待归档目录不存在或不可读：可在界面上指定其它目录", dir)
		return
	}
	projects, _ := activeProjects()

	type item struct {
		Name    string `json:"name"`
		Path    string `json:"path"`
		Size    int64  `json:"size"`
		ModTime string `json:"mod_time"`
		Project string `json:"project"`
		Code    string `json:"code"`
		Period  string `json:"period"`
	}
	list := []item{}
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), "~$") || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		pj, cd, pd := services.GuessFromName(e.Name(), projects)
		list = append(list, item{
			Name: e.Name(), Path: filepath.Join(dir, e.Name()),
			Size: info.Size(), ModTime: info.ModTime().Format("2006-01-02 15:04"),
			Project: pj, Code: cd, Period: pd,
		})
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"dir": dir, "files": list,
		"projects":  projects,
		"checklist": data.DefaultChecklist,
	})
}

type archiveReq struct {
	SourcePath string `json:"source_path"`
	Project    string `json:"project"`
	Code       string `json:"code"`
	Period     string `json:"period"`
	Version    int    `json:"version"`
	Overwrite  bool   `json:"overwrite"`
	Operator   string `json:"operator"`
}

// ArchivePlan POST /api/archive/plan —— 预演，不写盘。
func (h *Handler) ArchivePlan(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var q archiveReq
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		writeJSONError(w, http.StatusBadRequest, "请求体解析失败", "")
		return
	}
	plan := services.PlanArchive(h.root.ProjectsDir(), q.SourcePath, q.Project, q.Code, q.Period, q.Version)
	json.NewEncoder(w).Encode(plan)
}

// ArchiveApply POST /api/archive/apply —— 实际执行归档（写盘）。
func (h *Handler) ArchiveApply(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var q archiveReq
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		writeJSONError(w, http.StatusBadRequest, "请求体解析失败", "")
		return
	}
	if q.SourcePath == "" || q.Project == "" || q.Code == "" || q.Period == "" {
		writeJSONError(w, http.StatusBadRequest, "源文件、项目、文件类型、周期均不能为空", "")
		return
	}
	res := services.DoArchive(h.root.ProjectsDir(), q.SourcePath, q.Project, q.Code,
		q.Period, q.Version, q.Overwrite, q.Operator)
	if !res.OK {
		writeJSONError(w, http.StatusConflict, res.Err, "")
		return
	}
	json.NewEncoder(w).Encode(res)
}

// ArchiveLog GET /api/archive/log —— 归档操作日志（可回滚项带 can_undo）。
func (h *Handler) ArchiveLog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	n, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	list, err := services.ListArchiveLog(n)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}
	if list == nil {
		list = []services.ArchiveLogEntry{}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"logs": list})
}

// ArchiveUndo POST /api/archive/undo?id=N —— 回滚一次归档。
func (h *Handler) ArchiveUndo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, _ := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "缺少有效的日志 id", "")
		return
	}
	if err := services.UndoArchive(id); err != nil {
		writeJSONError(w, http.StatusConflict, err.Error(), "")
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"ok": "undone"})
}
