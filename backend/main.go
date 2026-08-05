package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"opims/database"
	"opims/handlers"
	"opims/services"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const defaultPort = "8080"

// 端口默认 8080，可用环境变量 OPIMS_PORT 改。
// 用途：本地要同时跑「正在用的正式实例」和「待验证的新版本」时，
// 后者换个端口即可，不必先把前者关掉。日常使用不需要设这个变量。
var port = func() string {
	if p := strings.TrimSpace(os.Getenv("OPIMS_PORT")); p != "" {
		return p
	}
	return defaultPort
}()

func main() {
	// Check if already running — if so, just open browser and exit
	if isPortInUse(port) {
		openBrowser()
		fmt.Println("OPIMS is already running. Opening browser...")
		fmt.Println("Press Enter to exit this launcher.")
		fmt.Scanln()
		return
	}

	execDir, _ := os.Executable()
	baseDir := filepath.Dir(execDir)

	if _, err := os.Stat(filepath.Join(baseDir, "opims.db")); os.IsNotExist(err) {
		baseDir, _ = os.Getwd()
	}

	dbPath := filepath.Join(baseDir, "opims.db")

	if err := database.Init(dbPath); err != nil {
		log.Fatal("Database init failed:", err)
	}
	defer database.Close()

	var rootPath string
	row := database.DB.QueryRow("SELECT value FROM app_config WHERE key='file_root_path'")
	if err := row.Scan(&rootPath); err != nil || rootPath == "" {
		// 首次运行：尚未配置根目录，先落在 exe 所在目录，
		// 由用户通过界面「项目文件 → 设置根目录」指向实际位置。
		rootPath = baseDir
	}

	rootDir := services.NewRootDir(rootPath)

	mux := http.NewServeMux()
	h := handlers.NewHandler(rootPath, rootDir)

	mux.HandleFunc("/api/projects/import", h.ImportProjects)
	mux.HandleFunc("/api/projects/export", h.ExportProjects)
	mux.HandleFunc("/api/projects", h.Projects)
	mux.HandleFunc("/api/projects/", h.ProjectByID)
	mux.HandleFunc("/api/files/scan", h.ScanFiles)
	mux.HandleFunc("/api/files/root", h.FileRoot)
	mux.HandleFunc("/api/files/contracts", h.ContractsInfo)
	mux.HandleFunc("/api/files/view", h.ViewFile)
	mux.HandleFunc("/api/files/", h.Files)
	mux.HandleFunc("/api/blacklist/subcontractor", h.SubBlacklist)
	mux.HandleFunc("/api/blacklist/subcontractor/", h.SubBlacklistByID)
	mux.HandleFunc("/api/subcontract/import", h.SubcontractImport)
	mux.HandleFunc("/api/subcontract/export", h.SubcontractExport)
	mux.HandleFunc("/api/subcontract/", h.SubcontractByID)
	mux.HandleFunc("/api/subcontract", h.SubcontractList)
	mux.HandleFunc("/api/subcontractors/export/template", h.SubcontractorsExportTemplate)
	mux.HandleFunc("/api/subcontractors/import", h.SubcontractorsImport)
	mux.HandleFunc("/api/subcontractors/export", h.SubcontractorsExport)
	mux.HandleFunc("/api/subcontractors/quals", h.SubcontractorsQuals)
	mux.HandleFunc("/api/subcontractors/qual/open", h.SubcontractorsQualOpen)
	mux.HandleFunc("/api/subcontractors/", h.SubcontractorsByID)
	mux.HandleFunc("/api/subcontractors", h.SubcontractorsList)
	mux.HandleFunc("/api/dashboard", h.Dashboard)
	mux.HandleFunc("/api/dashboard/contract-distribution", h.ContractDistribution)
	mux.HandleFunc("/api/dashboard/region-detail/", h.RegionDetail)
	mux.HandleFunc("/api/policy/countries", h.PolicyCountries)
	mux.HandleFunc("/api/policy/country/", h.PolicyCountryByName)
	mux.HandleFunc("/api/policy/internal", h.PolicyInternal)
	mux.HandleFunc("/api/progress/check", h.ProgressCheck)
	mux.HandleFunc("/api/progress/review", h.ProgressReview)
	mux.HandleFunc("/api/progress/snapshot", h.ProgressSnapshot)
	mux.HandleFunc("/api/progress/compliance", h.ProgressCompliance)
	mux.HandleFunc("/api/progress/compliance/export", h.ProgressComplianceExport)
	mux.HandleFunc("/api/timebar", h.TimeBarList)
	// 时限雷达：注册顺序从具体到宽泛，ServeMux 前缀匹配下 /log 必须在 /{id} 之前
	mux.HandleFunc("/api/radar/registry/log", h.RadarRegistryLog)
	mux.HandleFunc("/api/radar/registry/", h.RadarRegistryByID)
	mux.HandleFunc("/api/radar/registry", h.RadarRegistry)
	mux.HandleFunc("/api/radar", h.Radar)
	mux.HandleFunc("/api/eot", h.EOTList)
	mux.HandleFunc("/api/progress/indicators", h.ProgressIndicators)
	mux.HandleFunc("/api/archive/inbox", h.ArchiveInbox)
	mux.HandleFunc("/api/archive/plan", h.ArchivePlan)
	mux.HandleFunc("/api/archive/apply", h.ArchiveApply)
	mux.HandleFunc("/api/archive/log", h.ArchiveLog)
	mux.HandleFunc("/api/archive/undo", h.ArchiveUndo)
	mux.HandleFunc("/api/config/backup", h.Backup)
	mux.HandleFunc("/api/config/restore", h.Restore)

	// serve frontend
	fs := http.FileServer(http.Dir(filepath.Join(baseDir, "frontend-dist")))
	mux.Handle("/", fs)

	server := &http.Server{Addr: ":" + port, Handler: mux}

	// Start server in background
	go func() {
		fmt.Printf("OPIMS running at http://localhost:%s\n", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	// Wait for server to be ready
	time.Sleep(300 * time.Millisecond)
	openBrowser()

	fmt.Println("╔══════════════════════════════════╗")
	fmt.Println("║  OPIMS is running               ║")
	fmt.Println("║  Open: http://localhost:" + port + "      ║")
	fmt.Println("║  Press ENTER in this window     ║")
	fmt.Println("║  to stop the server.            ║")
	fmt.Println("╚══════════════════════════════════╝")

	// Wait for Enter to exit
	fmt.Scanln()
	fmt.Println("Shutting down...")
	server.Close()
}

func isPortInUse(port string) bool {
	timeout := 200 * time.Millisecond
	conn, err := net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", port), timeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func openBrowser() {
	url := "http://localhost:" + port
	switch runtime.GOOS {
	case "windows":
		exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		exec.Command("open", url).Start()
	default:
		exec.Command("xdg-open", url).Start()
	}
}
