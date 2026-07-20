package database

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var DB *sql.DB
var dbFilePath string

func DBPath() string { return dbFilePath }

func Init(dbPath string) error {
	dbFilePath = dbPath
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	var err error
	DB, err = sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return err
	}

	DB.SetMaxOpenConns(1)

	if err := migrate(); err != nil {
		return err
	}

	log.Println("Database initialized at", dbPath)
	return nil
}

func migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS projects (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		short_name TEXT NOT NULL UNIQUE,
		contract_no TEXT DEFAULT '',
		project_name TEXT NOT NULL,
		project_type TEXT DEFAULT '',
		project_status TEXT NOT NULL DEFAULT '在建',
		implement_unit TEXT DEFAULT '',
		contract_amount REAL DEFAULT 0,
		budget_amount REAL DEFAULT 0,
		contract_scope TEXT DEFAULT '',
		key_points TEXT DEFAULT '',
		domestic_overseas TEXT DEFAULT '',
		province TEXT DEFAULT '',
		city TEXT DEFAULT '',
		address TEXT DEFAULT '',
		country TEXT DEFAULT '',
		contract_start_year INTEGER DEFAULT 0,
		contract_start_month INTEGER DEFAULT 0,
		contract_end_year INTEGER DEFAULT 0,
		contract_end_month INTEGER DEFAULT 0,
		contract_duration INTEGER DEFAULT 0,
		actual_start_year INTEGER DEFAULT 0,
		actual_start_month INTEGER DEFAULT 0,
		plan_end_year INTEGER DEFAULT 0,
		plan_end_month INTEGER DEFAULT 0,
		actual_duration INTEGER DEFAULT 0,
		completion_year INTEGER DEFAULT 0,
		completion_month INTEGER DEFAULT 0,
		running_status TEXT DEFAULT '',
		abnormal_reason TEXT DEFAULT '',
		progress_status TEXT DEFAULT '',
		issues TEXT DEFAULT '',
		completed_output REAL DEFAULT 0,
		complete_percent TEXT DEFAULT '',
		progress_summary TEXT DEFAULT '',
		cum_receivable REAL DEFAULT 0,
		cum_received REAL DEFAULT 0,
		owed_amount REAL DEFAULT 0,
		gps_lat REAL DEFAULT 0,
		gps_lng REAL DEFAULT 0,
		pm_contract TEXT DEFAULT '',
		pm_appointed TEXT DEFAULT '',
		pm_onsite TEXT DEFAULT '',
		pm_phone TEXT DEFAULT '',
		pm_builder TEXT DEFAULT '',
		pm_safety_cert TEXT DEFAULT '',
		tech_lead_appointed TEXT DEFAULT '',
		tech_lead_onsite TEXT DEFAULT '',
		tech_lead_phone TEXT DEFAULT '',
		tech_lead_title TEXT DEFAULT '',
		quality_mgr_appointed TEXT DEFAULT '',
		quality_mgr_onsite TEXT DEFAULT '',
		quality_mgr_phone TEXT DEFAULT '',
		quality_mgr_cert TEXT DEFAULT '',
		hse_mgr_appointed TEXT DEFAULT '',
		hse_mgr_onsite TEXT DEFAULT '',
		hse_mgr_phone TEXT DEFAULT '',
		hse_mgr_cert TEXT DEFAULT '',
		cost_mgr_appointed TEXT DEFAULT '',
		cost_mgr_onsite TEXT DEFAULT '',
		cost_mgr_phone TEXT DEFAULT '',
		cost_mgr_cert TEXT DEFAULT '',
		quality_key_process TEXT DEFAULT '',
		quality_measures TEXT DEFAULT '',
		safety_cost REAL DEFAULT 0,
		safety_cost_spent REAL DEFAULT 0,
		safety_cost_cum REAL DEFAULT 0,
		safety_major_hazard TEXT DEFAULT '',
		safety_hazard_measure TEXT DEFAULT '',
		safety_risk_source TEXT DEFAULT '',
		safety_risk_measure TEXT DEFAULT '',
		owner_unit TEXT DEFAULT '',
		owner_contact TEXT DEFAULT '',
		owner_phone TEXT DEFAULT '',
		design_unit TEXT DEFAULT '',
		design_contact TEXT DEFAULT '',
		design_phone TEXT DEFAULT '',
		supervision_unit TEXT DEFAULT '',
		supervision_contact TEXT DEFAULT '',
		supervision_phone TEXT DEFAULT '',
		reporter TEXT DEFAULT '',
		personnel_mgmt INTEGER DEFAULT 0,
		personnel_labor INTEGER DEFAULT 0,
		is_deleted INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS subcontractor_blacklist (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		sub_short_name TEXT NOT NULL,
		sub_full_name TEXT NOT NULL,
		country TEXT DEFAULT '',
		related_project TEXT DEFAULT '',
		list_reason TEXT DEFAULT '',
		list_date TEXT NOT NULL,
		restrict_until TEXT DEFAULT '',
		list_reporter TEXT DEFAULT '',
		delist_reason TEXT DEFAULT '',
		delist_date TEXT DEFAULT '',
		delist_reporter TEXT DEFAULT '',
		status TEXT NOT NULL DEFAULT '列入中',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS blacklist_audit_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		blacklist_id INTEGER NOT NULL,
		action TEXT NOT NULL,
		detail TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS app_config (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);

	INSERT OR IGNORE INTO app_config (key, value) VALUES ('file_root_path', '');
	`

	_, err := DB.Exec(schema)
	return err
}

func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

func Reopen() error {
	var err error
	DB, err = sql.Open("sqlite", dbFilePath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		return err
	}
	DB.SetMaxOpenConns(1)
	return nil
}
