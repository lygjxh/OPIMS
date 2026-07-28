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
		restrict_level TEXT DEFAULT '',
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

	-- project_subcontract: mirrors the management ledger (32 columns A-AF)
	CREATE TABLE IF NOT EXISTS project_subcontract (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		period TEXT DEFAULT '',
		seq_no TEXT DEFAULT '',
		branch_company TEXT DEFAULT '',
		project_name TEXT DEFAULT '',
		main_contract_amount REAL DEFAULT 0,
		sub_name TEXT NOT NULL,
		sub_tier TEXT DEFAULT '',
		sub_profession_raw TEXT DEFAULT '',
		sub_contract_profession TEXT DEFAULT '',
		sub_controller TEXT DEFAULT '',
		sub_controller_phone TEXT DEFAULT '',
		contract_no TEXT DEFAULT '',
		contract_name TEXT DEFAULT '',
		contract_amount REAL DEFAULT 0,
		supplement_amount REAL DEFAULT 0,
		contract_date TEXT DEFAULT '',
		progress_percent TEXT DEFAULT '',
		entry_date TEXT DEFAULT '',
		exit_date TEXT DEFAULT '',
		evaluation_completed TEXT DEFAULT '',
		personnel_count INTEGER DEFAULT 0,
		site_leader TEXT DEFAULT '',
		site_leader_approved TEXT DEFAULT '',
		site_leader_status TEXT DEFAULT '',
		tech_leader TEXT DEFAULT '',
		tech_leader_approved TEXT DEFAULT '',
		tech_leader_status TEXT DEFAULT '',
		safety_officer TEXT DEFAULT '',
		safety_officer_approved TEXT DEFAULT '',
		safety_officer_status TEXT DEFAULT '',
		contract_compliance TEXT DEFAULT '',
		noncompliance_note TEXT DEFAULT '',
		remarks TEXT DEFAULT '',
		project_short_name TEXT DEFAULT '',
		standardized_profession TEXT DEFAULT '',
		profession_category TEXT DEFAULT '',
		is_deleted INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- subcontractors_base: subcontractor library master data
	-- 2026-07-26 重构：单 Sheet 31 列模板，以「分包商编号 sub_no」为唯一键。
	CREATE TABLE IF NOT EXISTS subcontractors_base (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		sub_no TEXT NOT NULL UNIQUE,
		report_project TEXT DEFAULT '',
		short_name TEXT DEFAULT '',
		full_name TEXT NOT NULL,
		country TEXT DEFAULT '',
		enterprise_type TEXT DEFAULT '',
		established_date TEXT DEFAULT '',
		reg_capital TEXT DEFAULT '',
		legal_rep TEXT DEFAULT '',
		legal_rep_phone TEXT DEFAULT '',
		agent TEXT DEFAULT '',
		agent_phone TEXT DEFAULT '',
		region TEXT DEFAULT '',
		address TEXT DEFAULT '',
		qualification TEXT DEFAULT '',
		credit_rating TEXT DEFAULT '',
		grade TEXT DEFAULT '',
		classification TEXT DEFAULT '',
		business_scope TEXT DEFAULT '',
		recommender TEXT DEFAULT '',
		report_unit TEXT DEFAULT '',
		unit_head TEXT DEFAULT '',
		category TEXT DEFAULT '',
		profession TEXT DEFAULT '',
		notes TEXT DEFAULT '',
		assoc_unit TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- subcontractor_projects: unified cooperation history (Plan B)
	CREATE TABLE IF NOT EXISTS subcontractor_projects (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		sub_short_name TEXT NOT NULL,
		project_short_name TEXT DEFAULT '',
		project_name TEXT DEFAULT '',
		start_date TEXT DEFAULT '',
		end_date TEXT DEFAULT '',
		contract_no TEXT DEFAULT '',
		contract_amount REAL DEFAULT 0,
		scope TEXT DEFAULT '',
		profession_category TEXT DEFAULT '',
		profession TEXT DEFAULT '',
		other_professions TEXT DEFAULT '',
		project_status TEXT DEFAULT '',
		is_manual INTEGER DEFAULT 0,
		notes TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_ps_period ON project_subcontract(period);
	CREATE INDEX IF NOT EXISTS idx_ps_contract ON project_subcontract(contract_no);

	-- 进度管理 · 人工复核（需求 V1.1 4.2.5）
	-- 系统按文件判定难免有偏差（如云盘同步重写了 mtime、个别文件未按流程归档），
	-- 管理员可修正判定并留备注。一个「周期+项目+文件类型」至多一条复核记录。
	CREATE TABLE IF NOT EXISTS submission_review (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		period TEXT NOT NULL,
		project TEXT NOT NULL,
		code TEXT NOT NULL,
		status TEXT NOT NULL DEFAULT '',
		note TEXT DEFAULT '',
		reviewer TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(period, project, code)
	);

	-- 进度管理 · 复核操作留痕（需求 V1.1 4.2.5「所有修改留痕」）
	CREATE TABLE IF NOT EXISTS submission_review_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		period TEXT NOT NULL,
		project TEXT NOT NULL,
		code TEXT NOT NULL,
		old_status TEXT DEFAULT '',
		new_status TEXT DEFAULT '',
		note TEXT DEFAULT '',
		reviewer TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- 进度管理 · 每期核查快照（需求 V1.1 4.2.6）
	-- 快照是考核的客观数据源：文件事后可能被删改，快照锁定核查当时的事实。
	CREATE TABLE IF NOT EXISTS submission_snapshot (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		period TEXT NOT NULL,
		project TEXT NOT NULL,
		code TEXT NOT NULL,
		required INTEGER DEFAULT 1,
		status TEXT NOT NULL,
		file_name TEXT DEFAULT '',
		submit_at TEXT DEFAULT '',
		deadline TEXT DEFAULT '',
		reviewed INTEGER DEFAULT 0,
		note TEXT DEFAULT '',
		taken_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(period, project, code)
	);

	CREATE INDEX IF NOT EXISTS idx_snap_period ON submission_snapshot(period);

	-- 进度管理 · 收文改名归档操作日志（需求 V1.1 4.4，一期 1c）
	-- 这是 OPIMS 第一个写云盘文件的功能，每次操作必须留痕以便回滚与追溯。
	CREATE TABLE IF NOT EXISTS archive_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		operator TEXT DEFAULT '',
		source_path TEXT NOT NULL,
		target_path TEXT NOT NULL,
		project TEXT DEFAULT '',
		code TEXT DEFAULT '',
		period TEXT DEFAULT '',
		overwrite INTEGER DEFAULT 0,
		undone INTEGER DEFAULT 0,
		undone_at TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	INSERT OR IGNORE INTO app_config (key, value) VALUES ('file_root_path', '');
	`

	// project_subcontract.period must exist before the CREATE INDEX above runs on
	// an already-populated database, so migrate the column first.
	addColumnIfMissing("project_subcontract", "period", "TEXT DEFAULT ''")

	// 分包商库 2026-07-26 重构：旧结构（含 registration_type 列）与新单 Sheet 结构
	// 不兼容，旧库仅为测试数据，直接重建（CREATE TABLE IF NOT EXISTS 随后重新建表）。
	if hasColumn("subcontractors_base", "registration_type") {
		DB.Exec("DROP TABLE subcontractors_base")
	}

	if _, err := DB.Exec(schema); err != nil {
		return err
	}

	// Safe addition of restrict_level column for existing databases
	addColumnIfMissing("subcontractor_blacklist", "restrict_level", "TEXT DEFAULT ''")
	return nil
}

// addColumnIfMissing adds a column only if it doesn't already exist in the table.
func addColumnIfMissing(table, column, colDef string) {
	if hasColumn(table, column) {
		return
	}
	DB.Exec("ALTER TABLE " + table + " ADD COLUMN " + column + " " + colDef)
}

// hasColumn reports whether the given table currently has the given column.
func hasColumn(table, column string) bool {
	var count int
	row := DB.QueryRow("SELECT COUNT(*) FROM pragma_table_info(?) WHERE name=?", table, column)
	if err := row.Scan(&count); err != nil {
		return false
	}
	return count > 0
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
