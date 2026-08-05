package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"opims/data"
	"opims/database"
	"opims/models"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Dashboard returns aggregate stats + map markers.
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var data models.DashboardData
	data.StatusCounts = make(map[string]int)

	if rows, err := database.DB.Query(
		"SELECT project_status, COUNT(*) FROM projects WHERE is_deleted=0 GROUP BY project_status"); err == nil {
		defer rows.Close()
		for rows.Next() {
			var s string
			var c int
			rows.Scan(&s, &c)
			data.StatusCounts[s] = c
			data.TotalProjects += c
		}
	}

	if mRows, err := database.DB.Query(
		"SELECT short_name, gps_lat, gps_lng, country, project_status FROM projects WHERE is_deleted=0"); err == nil {
		defer mRows.Close()
		for mRows.Next() {
			var m models.MapMarker
			mRows.Scan(&m.ShortName, &m.Lat, &m.Lng, &m.Country, &m.Status)
			if m.Lat == 0 && m.Lng == 0 {
				continue
			}
			data.ProjectMarkers = append(data.ProjectMarkers, m)
		}
	}

	json.NewEncoder(w).Encode(data)
}

// Backup copies the database to a user-chosen folder.
func (h *Handler) Backup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Path == "" {
		http.Error(w, "path required", http.StatusBadRequest)
		return
	}

	// 时间戳原本只到分钟，同一分钟内备份两次会静默覆盖前一份。改到秒，
	// 再对同秒重名的情况追加序号，确保任何一次备份都不会顶掉已有文件。
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("opims_backup_%s.db", timestamp)
	dest := filepath.Join(body.Path, filename)
	for i := 2; ; i++ {
		if _, err := os.Stat(dest); os.IsNotExist(err) {
			break
		}
		filename = fmt.Sprintf("opims_backup_%s_%d.db", timestamp, i)
		dest = filepath.Join(body.Path, filename)
	}

	// 数据库运行在 WAL 模式，最近的写入可能还在 -wal 文件里没落盘。
	// 备份只复制 .db 主文件，故先做 checkpoint 把 WAL 内容合并进主库，
	// 否则备份会缺少最新几条改动。
	if _, err := database.DB.Exec("PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		http.Error(w, "checkpoint failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := copyFile(database.DBPath(), dest); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"path": dest, "filename": filename})
}

// Restore replaces the current database with a backup file.
func (h *Handler) Restore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Path == "" {
		http.Error(w, "path required", http.StatusBadRequest)
		return
	}
	if _, err := os.Stat(body.Path); os.IsNotExist(err) {
		http.Error(w, "backup file not found", http.StatusNotFound)
		return
	}

	// Close current DB, replace file, reopen
	if err := database.Close(); err != nil {
		http.Error(w, "failed to close db: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 备份文件只含主库。库跑在 WAL 模式时，旧的 -wal 里可能还有备份之后写入的改动，
	// 换掉主库却留着它，重新打开时会把这些改动回放回来，恢复就等于没做。
	for _, suffix := range []string{"-wal", "-shm"} {
		if err := os.Remove(database.DBPath() + suffix); err != nil && !os.IsNotExist(err) {
			http.Error(w, "failed to clear "+suffix+": "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	if err := copyFile(body.Path, database.DBPath()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := database.Reopen(); err != nil {
		http.Error(w, "database replaced but reopen failed; please restart: "+err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func copyFile(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer d.Close()

	_, err = io.Copy(d, s)
	return err
}

// parseStatusFilter 解析 status 查询参数（逗号分隔），仅保留合法状态值；
// 传入空、"全部" 或全部非法时返回 nil（表示不加状态过滤，即"全部"）。
func parseStatusFilter(raw string) []string {
	if raw == "" || raw == "全部" {
		return nil
	}
	valid := map[string]bool{"未开工": true, "在建": true, "停工": true, "完工": true}
	var out []string
	for _, p := range strings.Split(raw, ",") {
		p = strings.TrimSpace(p)
		if valid[p] {
			out = append(out, p)
		}
	}
	return out
}

// ContractDistribution returns contract amount grouped by region or country.
// GET /api/dashboard/contract-distribution?dim=region&status=在建,未开工
func (h *Handler) ContractDistribution(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// dim 仅接受 region / country，其余（含空、非法值）一律回落到默认的 region。
	dim := r.URL.Query().Get("dim")
	if dim != "country" {
		dim = "region"
	}
	rawStatus := r.URL.Query().Get("status")

	conditions := []string{"is_deleted=0", "domestic_overseas='境外'"}
	args := []interface{}{}
	if validStatuses := parseStatusFilter(rawStatus); len(validStatuses) > 0 {
		ph := make([]string, len(validStatuses))
		for i, p := range validStatuses {
			ph[i] = "?"
			args = append(args, p)
		}
		conditions = append(conditions, "project_status IN ("+strings.Join(ph, ",")+")")
	}
	where := strings.Join(conditions, " AND ")

	rows, err := database.DB.Query("SELECT country, contract_amount FROM projects WHERE "+where, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type entry struct{ name string; amount float64; count int }
	agg := make(map[string]*entry)

	for rows.Next() {
		var country string
		var amt float64
		if err := rows.Scan(&country, &amt); err != nil {
			continue
		}
		key := country
		if dim == "region" {
			key = data.RegionOf(country)
		}
		if e, ok := agg[key]; ok {
			e.amount += amt
			e.count++
		} else {
			agg[key] = &entry{name: key, amount: amt, count: 1}
		}
	}

	slices := make([]models.ContractSlice, 0, len(agg))
	var totalAmount float64
	var totalCount int
	for _, e := range agg {
		totalAmount += e.amount
		totalCount += e.count
		slices = append(slices, models.ContractSlice{
			Name: e.name, Amount: math.Round(e.amount*100) / 100, Count: e.count,
		})
	}

	sort.Slice(slices, func(i, j int) bool { return slices[i].Amount > slices[j].Amount })

	if len(slices) > 7 {
		keep := slices[:7]
		var ta float64
		var tc int
		for _, s := range slices[7:] {
			ta += s.Amount
			tc += s.Count
		}
		keep = append(keep, models.ContractSlice{
			Name: fmt.Sprintf("其他 %d 项", tc), Amount: math.Round(ta*100) / 100, Count: tc,
		})
		slices = keep
	}

	for i := range slices {
		if totalAmount > 0 {
			slices[i].Percentage = math.Round(slices[i].Amount/totalAmount*10000) / 100
		}
	}

	json.NewEncoder(w).Encode(models.ContractDistributionResponse{
		Slices: slices, TotalAmount: math.Round(totalAmount*100) / 100, TotalCount: totalCount,
	})
}

// RegionDetail returns per-country breakdown for a given region.
// GET /api/dashboard/region-detail/{region}?status=在建,未开工
func (h *Handler) RegionDetail(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	region := strings.TrimPrefix(r.URL.Path, "/api/dashboard/region-detail/")
	region = strings.TrimSuffix(region, "/")
	if region == "" {
		http.Error(w, "region required", http.StatusBadRequest)
		return
	}

	var countries []string
	for reg, cs := range data.RegionCountries {
		if reg == region { countries = cs; break }
	}
	if countries == nil {
		http.Error(w, "region not found", http.StatusNotFound)
		return
	}

	rawStatus := r.URL.Query().Get("status")
	conditions := []string{"is_deleted=0", "domestic_overseas='境外'"}
	args := []interface{}{}
	if validStatuses := parseStatusFilter(rawStatus); len(validStatuses) > 0 {
		ph := make([]string, len(validStatuses))
		for i, p := range validStatuses {
			ph[i] = "?"
			args = append(args, p)
		}
		conditions = append(conditions, "project_status IN ("+strings.Join(ph, ",")+")")
	}

	cph := make([]string, len(countries))
	for i := range countries {
		cph[i] = "?"
		args = append(args, countries[i])
	}
	conditions = append(conditions, "country IN ("+strings.Join(cph, ",")+")")
	where := strings.Join(conditions, " AND ")

	rows, err := database.DB.Query("SELECT country, SUM(contract_amount), COUNT(*) FROM projects WHERE "+where+" GROUP BY country", args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var details []models.RegionCountryDetail
	var totalAmount float64
	var totalCount int
	for rows.Next() {
		var d models.RegionCountryDetail
		if err := rows.Scan(&d.Name, &d.Amount, &d.Count); err != nil { continue }
		d.Amount = math.Round(d.Amount*100) / 100
		totalAmount += d.Amount
		totalCount += d.Count
		details = append(details, d)
	}

	for i := range details {
		if totalAmount > 0 {
			details[i].Percentage = math.Round(details[i].Amount/totalAmount*10000) / 100
		}
	}

	json.NewEncoder(w).Encode(details)
}
