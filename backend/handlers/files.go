package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"opims/database"
	"opims/models"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Files returns directory tree for a project's file folder.
func (h *Handler) Files(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	projectName := strings.TrimPrefix(r.URL.Path, "/api/files/")
	projectName = strings.TrimSuffix(projectName, "/")
	if projectName == "scan" || projectName == "root" {
		return
	}

	projDir := filepath.Join(h.root.Get(), projectName)
	entries, err := os.ReadDir(projDir)
	if err != nil {
		json.NewEncoder(w).Encode([]models.FileNode{})
		return
	}

	var nodes []models.FileNode
	for _, e := range entries {
		info, _ := e.Info()
		node := models.FileNode{
			Name:    e.Name(),
			Path:    filepath.Join(projDir, e.Name()),
			IsDir:   e.IsDir(),
			ModTime: info.ModTime().Format("2006-01-02 15:04"),
		}
		if !e.IsDir() {
			node.Size = info.Size()
		} else {
			if sub, _ := os.ReadDir(node.Path); sub != nil {
				for _, se := range sub {
					si, _ := se.Info()
					child := models.FileNode{
						Name:  se.Name(),
						Path:  filepath.Join(node.Path, se.Name()),
						IsDir: se.IsDir(),
					}
					child.ModTime = si.ModTime().Format("2006-01-02 15:04")
					if !se.IsDir() {
						child.Size = si.Size()
					}
					node.Children = append(node.Children, child)
				}
			}
		}
		nodes = append(nodes, node)
	}
	json.NewEncoder(w).Encode(nodes)
}

// ContractsInfo lists contract files for a project.
func (h *Handler) ContractsInfo(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	if project == "" {
		http.Error(w, "missing project", http.StatusBadRequest)
		return
	}

	dir := filepath.Join(h.root.Get(), project, "01.Contract")
	entries, err := os.ReadDir(dir)
	if err != nil {
		json.NewEncoder(w).Encode([]struct{}{})
		return
	}

	type cInfo struct {
		Name  string `json:"name"`
		Label string `json:"label"`
		Path  string `json:"path"`
	}
	var main, supp []cInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		lower := strings.ToLower(e.Name())
		if strings.Contains(lower, "supplement") {
			label := "补充合同"
			base := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
			if num := extractTrailingNumber(base); num > 0 {
				label = fmt.Sprintf("补充合同%s", chineseNumber(num))
			}
			supp = append(supp, cInfo{e.Name(), label, filepath.Join(dir, e.Name())})
		} else if strings.Contains(lower, "contract") {
			main = append(main, cInfo{e.Name(), "主合同", filepath.Join(dir, e.Name())})
		}
	}
	if len(main) == 0 {
		for _, e := range entries {
			if e.IsDir() || strings.Contains(strings.ToLower(e.Name()), "supplement") {
				continue
			}
			main = append(main, cInfo{e.Name(), "合同文件", filepath.Join(dir, e.Name())})
		}
	}
	result := append(main, supp...)
	if result == nil {
		result = []cInfo{}
	}
	json.NewEncoder(w).Encode(result)
}

// ViewFile opens a file with the OS default program.
func (h *Handler) ViewFile(w http.ResponseWriter, r *http.Request) {
	project := r.URL.Query().Get("project")
	folder := r.URL.Query().Get("folder")
	fileName := r.URL.Query().Get("file")
	if project == "" || folder == "" {
		http.Error(w, "missing project or folder", http.StatusBadRequest)
		return
	}

	dir := filepath.Join(h.root.Get(), project, folder)
	var target string
	if fileName != "" {
		target = filepath.Join(dir, fileName)
		if _, err := os.Stat(target); err != nil {
			http.Error(w, "file not found", http.StatusNotFound)
			return
		}
	} else {
		if entries, err := os.ReadDir(dir); err == nil {
			for _, e := range entries {
				if e.IsDir() {
					continue
				}
				target = filepath.Join(dir, e.Name())
				if strings.Contains(strings.ToLower(e.Name()), "contract") {
					break
				}
			}
		}
	}
	if target == "" {
		http.Error(w, "no file found", http.StatusNotFound)
		return
	}

	exec.Command("rundll32", "url.dll,FileProtocolHandler", target).Start()
	json.NewEncoder(w).Encode(map[string]string{"ok": "opened"})
}

// ScanFiles scans the root directory and returns file tree.
func (h *Handler) ScanFiles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	root := h.root.Get()

	entries, err := os.ReadDir(root)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"root": root, "files": []models.FileNode{}})
		return
	}

	var files []models.FileNode
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		info, _ := e.Info()
		node := models.FileNode{
			Name:    e.Name(),
			Path:    filepath.Join(root, e.Name()),
			IsDir:   true,
			ModTime: info.ModTime().Format("2006-01-02 15:04"),
		}
		if sub, _ := os.ReadDir(node.Path); sub != nil {
			for _, se := range sub {
				si, _ := se.Info()
				child := models.FileNode{
					Name:  se.Name(),
					Path:  filepath.Join(node.Path, se.Name()),
					IsDir: se.IsDir(),
				}
				child.ModTime = si.ModTime().Format("2006-01-02 15:04")
				if !se.IsDir() {
					child.Size = si.Size()
				}
				node.Children = append(node.Children, child)
			}
		}
		files = append(files, node)
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"root": root, "files": files})
}

// FileRoot gets/sets the file root path.
func (h *Handler) FileRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		json.NewEncoder(w).Encode(map[string]string{"path": h.root.Get()})
	case "POST", "PUT":
		var body struct {
			Root string `json:"root"`
			Path string `json:"path"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		newRoot := body.Root
		if newRoot == "" {
			newRoot = body.Path
		}
		if newRoot != "" {
			h.root.Set(newRoot)
			database.DB.Exec("UPDATE app_config SET value=? WHERE key='file_root_path'", newRoot)
		}
		json.NewEncoder(w).Encode(map[string]string{"ok": "updated"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
