package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"opims/database"
	"opims/models"
	"os"
	"path/filepath"
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

	timestamp := time.Now().Format("20060102_1504")
	filename := fmt.Sprintf("opims_backup_%s.db", timestamp)
	dest := filepath.Join(body.Path, filename)

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
