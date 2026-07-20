package handlers

import (
	"encoding/json"
	"net/http"
	"opims/database"
	"opims/models"
	"strconv"
	"strings"
)

// SubBlacklist handles GET (list) / POST (create) for subcontractor blacklist.
func (h *Handler) SubBlacklist(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		q := r.URL.Query()
		where := "WHERE 1=1"
		args := []interface{}{}

		for _, f := range []struct{ param, col string }{
			{"country", "country"},
			{"project", "related_project"},
			{"status", "status"},
		} {
			if v := q.Get(f.param); v != "" {
				where += " AND " + f.col + "=?"
				args = append(args, v)
			}
		}

		rows, err := database.DB.Query(
			"SELECT * FROM subcontractor_blacklist "+where+" ORDER BY id DESC", args...)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var list []models.SubBlacklist
		for rows.Next() {
			var b models.SubBlacklist
			rows.Scan(&b.ID, &b.SubShortName, &b.SubFullName, &b.Country, &b.RelatedProject,
				&b.ListReason, &b.ListDate, &b.RestrictUntil, &b.ListReporter,
				&b.DelistReason, &b.DelistDate, &b.DelistReporter, &b.Status,
				new(string), new(string))
			list = append(list, b)
		}
		json.NewEncoder(w).Encode(list)

	case "POST":
		var b models.SubBlacklist
		json.NewDecoder(r.Body).Decode(&b)
		b.Status = "列入中"
		res, err := database.DB.Exec(
			`INSERT INTO subcontractor_blacklist (sub_short_name,sub_full_name,country,related_project,list_reason,list_date,restrict_until,list_reporter,status)
			 VALUES (?,?,?,?,?,?,?,?,?)`,
			b.SubShortName, b.SubFullName, b.Country, b.RelatedProject,
			b.ListReason, b.ListDate, b.RestrictUntil, b.ListReporter, b.Status)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		id, _ := res.LastInsertId()
		database.DB.Exec("INSERT INTO blacklist_audit_log (blacklist_id,action,detail) VALUES (?,'列入','')", id)
		json.NewEncoder(w).Encode(map[string]string{"ok": "created"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// SubBlacklistByID handles single-entry PUT / DELETE.
func (h *Handler) SubBlacklistByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/api/blacklist/subcontractor/"))

	switch r.Method {
	case "PUT":
		var b models.SubBlacklist
		json.NewDecoder(r.Body).Decode(&b)

		if b.DelistDate != "" && b.Status == "" {
			b.Status = "已拉出"
		}

		var cur string
		database.DB.QueryRow("SELECT status FROM subcontractor_blacklist WHERE id=?", id).Scan(&cur)
		if cur == "已拉出" {
			http.Error(w, "已拉出的记录不可修改", http.StatusForbidden)
			return
		}

		_, err := database.DB.Exec(
			`UPDATE subcontractor_blacklist SET sub_short_name=?,sub_full_name=?,country=?,related_project=?,
			 list_reason=?,list_date=?,restrict_until=?,list_reporter=?,
			 delist_reason=?,delist_date=?,delist_reporter=?,status=?,updated_at=CURRENT_TIMESTAMP WHERE id=?`,
			b.SubShortName, b.SubFullName, b.Country, b.RelatedProject,
			b.ListReason, b.ListDate, b.RestrictUntil, b.ListReporter,
			b.DelistReason, b.DelistDate, b.DelistReporter, b.Status, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		action := "编辑"
		if b.Status == "已拉出" {
			action = "拉出"
		}
		database.DB.Exec("INSERT INTO blacklist_audit_log (blacklist_id,action,detail) VALUES (?,?,'')", id, action)
		json.NewEncoder(w).Encode(map[string]string{"ok": "updated"})

	case "DELETE":
		var status string
		database.DB.QueryRow("SELECT status FROM subcontractor_blacklist WHERE id=?", id).Scan(&status)
		if status == "已拉出" {
			http.Error(w, "已拉出的记录不可删除", http.StatusForbidden)
			return
		}
		database.DB.Exec("DELETE FROM subcontractor_blacklist WHERE id=?", id)
		database.DB.Exec("INSERT INTO blacklist_audit_log (blacklist_id,action,detail) VALUES (?,'删除','')", id)
		json.NewEncoder(w).Encode(map[string]string{"ok": "deleted"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
