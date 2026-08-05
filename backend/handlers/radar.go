package handlers

import (
	"encoding/json"
	"net/http"
	"opims/database"
	"opims/services"
	"os"
	"strconv"
	"strings"
	"time"
)

// 进度管理 · 时限雷达（细则 5.1 / 6.3、附件 G 第十节）
//
// 与原 /api/timebar 的区别：那个只读项目部报上来的附件 E，本接口把中心登记簿
// 也合进来。中心在合同评审阶段就能登记时效条款，不必等第一次月报。

// Radar GET /api/radar —— 合并后的雷达清单。
func (h *Handler) Radar(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	rows, err := loadRegistry("")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	// 附件 E 扫描是尽力而为：目录没设好或项目部没报，不该让整个雷达失效——
	// 中心登记簿那部分本来就不依赖云盘。
	var scan *services.TimeBarResult
	var scanNote string
	projects, _ := activeProjects()
	if len(projects) == 0 {
		scanNote = "没有「在建」状态的项目，附件 E 未扫描；下列条目来自中心登记簿"
	} else if dir := h.root.ProjectsDir(); dir == "" {
		scanNote = "未设置项目文件根目录，附件 E 未扫描；下列条目来自中心登记簿"
	} else if _, err := os.Stat(dir); err != nil {
		scanNote = "项目文件目录不可读，附件 E 未扫描；下列条目来自中心登记簿"
	} else {
		scan = services.ScanTimeBars(dir, projects, today)
	}

	res := services.BuildRadar(rows, scan, today)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items": res.Items, "warnings": res.Warnings,
		"summary": res.Summary, "scan_note": scanNote,
		"today": today.Format("2006-01-02"),
	})
}

// RadarRegistry GET/POST /api/radar/registry —— 中心登记簿的读取与新增。
func (h *Handler) RadarRegistry(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		rows, err := loadRegistry(r.URL.Query().Get("project"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out := make([]map[string]interface{}, 0, len(rows))
		for _, x := range rows {
			out = append(out, registryJSON(x))
		}
		json.NewEncoder(w).Encode(map[string]interface{}{"items": out})

	case http.MethodPost:
		var q registryInput
		if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
			writeJSONError(w, http.StatusBadRequest, "请求体解析失败", "")
			return
		}
		if msg := q.normalize(); msg != "" {
			writeJSONError(w, http.StatusBadRequest, msg, "")
			return
		}
		res, err := database.DB.Exec(
			`INSERT INTO time_bar_registry
			 (project, code, signal_code, title, clause, direction,
			  trigger_date, due_days, due_date, remind_days, status, remark)
			 VALUES (?,?,?,?,?,?,?,?,?,?,'open',?)`,
			q.Project, q.Code, q.SignalCode, q.Title, q.Clause, q.Direction,
			q.TriggerDate, q.DueDays, q.DueDate, q.RemindDays, q.Remark)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		id, _ := res.LastInsertId()
		logRegistry(id, "create", "", "open", "", "")
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": "created", "id": id})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// RadarRegistryByID PUT /api/radar/registry/{id} —— 修改内容或变更状态。
//
// ⚠️ 故意不实现 DELETE。附件 G 十二.4：经复核不构成索赔的记录不得删除，
// 只能改状态并写明理由——它是我方「已尽注意义务」的证明。
func (h *Handler) RadarRegistryByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	idStr := strings.TrimPrefix(r.URL.Path, "/api/radar/registry/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeJSONError(w, http.StatusBadRequest, "无效的记录编号", "")
		return
	}

	if r.Method == http.MethodDelete {
		writeJSONError(w, http.StatusMethodNotAllowed,
			"时限登记记录不可删除。经复核不构成索赔的，请改为「不构成索赔」并注明理由——"+
				"记录本身是我方已尽注意义务的证明（附件 G 十二.4）", "")
		return
	}
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var cur struct{ Status string }
	if err := database.DB.QueryRow(
		`SELECT status FROM time_bar_registry WHERE id=?`, id).Scan(&cur.Status); err != nil {
		writeJSONError(w, http.StatusNotFound, "记录不存在", "")
		return
	}

	var q registryInput
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		writeJSONError(w, http.StatusBadRequest, "请求体解析失败", "")
		return
	}
	if msg := q.normalize(); msg != "" {
		writeJSONError(w, http.StatusBadRequest, msg, "")
		return
	}
	// 判为「不构成索赔」必须写理由，否则这条记录日后无法自证判断依据
	if q.Status == "not_claim" && strings.TrimSpace(q.ClosedReason) == "" {
		writeJSONError(w, http.StatusBadRequest,
			"判为「不构成索赔」时必须填写理由（附件 G 十二.4）", "")
		return
	}

	_, err = database.DB.Exec(
		`UPDATE time_bar_registry SET
		   project=?, code=?, signal_code=?, title=?, clause=?, direction=?,
		   trigger_date=?, due_days=?, due_date=?, remind_days=?,
		   status=?, closed_reason=?, remark=?, updated_at=CURRENT_TIMESTAMP
		 WHERE id=?`,
		q.Project, q.Code, q.SignalCode, q.Title, q.Clause, q.Direction,
		q.TriggerDate, q.DueDays, q.DueDate, q.RemindDays,
		q.Status, q.ClosedReason, q.Remark, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	action := "update"
	if q.Status != cur.Status {
		action = "status"
	}
	logRegistry(id, action, cur.Status, q.Status, q.ClosedReason, "")
	json.NewEncoder(w).Encode(map[string]string{"ok": "saved"})
}

// RadarRegistryLog GET /api/radar/registry/log?id= —— 某条登记的变更留痕。
func (h *Handler) RadarRegistryLog(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id, _ := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	rows, err := database.DB.Query(
		`SELECT action, from_status, to_status, reason, operator, created_at
		   FROM time_bar_registry_log WHERE registry_id=? ORDER BY id DESC`, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	list := []map[string]string{}
	for rows.Next() {
		var a, f, t, rs, op, at string
		if err := rows.Scan(&a, &f, &t, &rs, &op, &at); err != nil {
			continue
		}
		list = append(list, map[string]string{
			"action": a, "from_status": f, "to_status": t,
			"reason": rs, "operator": op, "created_at": at,
		})
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"items": list})
}

// ---- 内部 ----

type registryInput struct {
	Project      string `json:"project"`
	Code         string `json:"code"`
	SignalCode   string `json:"signal_code"`
	Title        string `json:"title"`
	Clause       string `json:"clause"`
	Direction    string `json:"direction"`
	TriggerDate  string `json:"trigger_date"`
	DueDays      int    `json:"due_days"`
	DueDate      string `json:"due_date"`
	RemindDays   int    `json:"remind_days"`
	Status       string `json:"status"`
	ClosedReason string `json:"closed_reason"`
	Remark       string `json:"remark"`
}

// normalize 校验并补全。返回非空字符串表示校验不通过。
func (q *registryInput) normalize() string {
	q.Project = strings.TrimSpace(q.Project)
	q.Title = strings.TrimSpace(q.Title)
	q.TriggerDate = strings.TrimSpace(q.TriggerDate)
	q.DueDate = strings.TrimSpace(q.DueDate)

	if q.Project == "" {
		return "请选择项目"
	}
	if q.Title == "" {
		return "请填写事项名称"
	}
	if q.Direction != services.DirRisk {
		q.Direction = services.DirClaim
	}
	if q.Status == "" {
		q.Status = "open"
	}
	switch q.Status {
	case "open", "done", "not_claim":
	default:
		return "无效的状态值"
	}

	// 到期日可以直接填，也可以由起算日 + 时限天数算出来。
	// 两者都没有就无法倒计时，这条记录也就没有意义了。
	if q.DueDate == "" {
		if q.TriggerDate == "" || q.DueDays <= 0 {
			return "请填写到期日，或同时填写起算日与时限天数"
		}
		td, ok := services.ParseFlexDate(q.TriggerDate)
		if !ok {
			return "起算日格式无法识别，请用 YYYY-MM-DD"
		}
		// 按日历日推算，不扣周末与节假日（细则 5.1）
		q.DueDate = td.AddDate(0, 0, q.DueDays).Format("2006-01-02")
	} else if _, ok := services.ParseFlexDate(q.DueDate); !ok {
		return "到期日格式无法识别，请用 YYYY-MM-DD"
	}

	if q.Direction == services.DirRisk && q.RemindDays <= 0 {
		q.RemindDays = services.DefaultRiskRemindDays
	}
	if q.RemindDays < 0 {
		q.RemindDays = 0
	}
	return ""
}

func loadRegistry(project string) ([]services.RegistryRow, error) {
	q := `SELECT id, project, code, signal_code, title, clause, direction,
	             trigger_date, due_days, due_date, remind_days, status, remark
	        FROM time_bar_registry`
	args := []interface{}{}
	if p := strings.TrimSpace(project); p != "" {
		q += ` WHERE project=?`
		args = append(args, p)
	}
	q += ` ORDER BY due_date`

	rows, err := database.DB.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	list := []services.RegistryRow{}
	for rows.Next() {
		var x services.RegistryRow
		if err := rows.Scan(&x.ID, &x.Project, &x.Code, &x.SignalCode, &x.Title,
			&x.Clause, &x.Direction, &x.TriggerDate, &x.DueDays, &x.DueDate,
			&x.RemindDays, &x.Status, &x.Remark); err != nil {
			continue
		}
		list = append(list, x)
	}
	return list, rows.Err()
}

func registryJSON(x services.RegistryRow) map[string]interface{} {
	return map[string]interface{}{
		"id": x.ID, "project": x.Project, "code": x.Code,
		"signal_code": x.SignalCode, "title": x.Title, "clause": x.Clause,
		"direction": x.Direction, "trigger_date": x.TriggerDate,
		"due_days": x.DueDays, "due_date": x.DueDate,
		"remind_days": x.RemindDays, "status": x.Status, "remark": x.Remark,
	}
}

func logRegistry(id int64, action, from, to, reason, operator string) {
	database.DB.Exec(
		`INSERT INTO time_bar_registry_log
		 (registry_id, action, from_status, to_status, reason, operator)
		 VALUES (?,?,?,?,?,?)`, id, action, from, to, reason, operator)
}
