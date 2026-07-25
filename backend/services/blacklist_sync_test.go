package services

import (
	"opims/database"
	"path/filepath"
	"testing"
)

// TestOnBlacklistChangedCascade verifies that blacklisting a domestic parent
// cascades a disable (is_deleted=1) onto its 当地注册 local subsidiaries'
// project_subcontract rows, and that de-listing restores them.
func TestOnBlacklistChangedCascade(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "cascade.db")
	if err := database.Init(dbPath); err != nil {
		t.Fatalf("db init: %v", err)
	}
	defer database.Close()

	// Local subsidiary of domestic parent PARENT1.
	_, err := database.DB.Exec(`INSERT INTO subcontractors_base
		(short_name,full_name,registration_type,profession_category,profession,parent_short_name,legal_rep_name,legal_rep_phone)
		VALUES ('LOCAL1','Local One','当地注册','施工','土建','PARENT1','rep','123')`)
	if err != nil {
		t.Fatalf("insert sub: %v", err)
	}
	// A subcontract row whose sub_name = the local's short_name.
	_, err = database.DB.Exec(`INSERT INTO project_subcontract (sub_name) VALUES ('LOCAL1')`)
	if err != nil {
		t.Fatalf("insert subcontract: %v", err)
	}

	getDeleted := func() int {
		var d int
		database.DB.QueryRow(`SELECT is_deleted FROM project_subcontract WHERE sub_name='LOCAL1'`).Scan(&d)
		return d
	}

	if getDeleted() != 0 {
		t.Fatalf("precondition: expected is_deleted=0")
	}

	// List parent -> should cascade disable local.
	res, _ := database.DB.Exec(`INSERT INTO subcontractor_blacklist
		(sub_short_name,sub_full_name,list_date,status) VALUES ('PARENT1','Parent','2026-07-25','列入中')`)
	blID, _ := res.LastInsertId()
	OnBlacklistChanged("PARENT1", "列入")
	if got := getDeleted(); got != 1 {
		t.Errorf("after 列入: expected is_deleted=1, got %d", got)
	}

	// De-list parent -> should restore local.
	database.DB.Exec(`UPDATE subcontractor_blacklist SET status='已拉出' WHERE id=?`, blID)
	OnBlacklistChanged("PARENT1", "拉出")
	if got := getDeleted(); got != 0 {
		t.Errorf("after 拉出: expected is_deleted=0, got %d", got)
	}
}

// TestOnBlacklistChangedMultipleActiveEntries reproduces the COUNT(*)->bool edge:
// when a parent has >1 active blacklist rows, scanning COUNT into a bool fails.
func TestOnBlacklistChangedMultipleActiveEntries(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "cascade2.db")
	if err := database.Init(dbPath); err != nil {
		t.Fatalf("db init: %v", err)
	}
	defer database.Close()

	database.DB.Exec(`INSERT INTO subcontractors_base
		(short_name,full_name,registration_type,profession_category,profession,parent_short_name,legal_rep_name,legal_rep_phone)
		VALUES ('LOCAL1','Local One','当地注册','施工','土建','PARENT1','rep','123')`)
	database.DB.Exec(`INSERT INTO project_subcontract (sub_name) VALUES ('LOCAL1')`)

	// Two active blacklist entries for the same parent.
	database.DB.Exec(`INSERT INTO subcontractor_blacklist (sub_short_name,sub_full_name,list_date,status) VALUES ('PARENT1','Parent','2026-07-25','列入中')`)
	database.DB.Exec(`INSERT INTO subcontractor_blacklist (sub_short_name,sub_full_name,list_date,status) VALUES ('PARENT1','Parent','2026-07-25','列入中')`)

	OnBlacklistChanged("PARENT1", "列入")

	var d int
	database.DB.QueryRow(`SELECT is_deleted FROM project_subcontract WHERE sub_name='LOCAL1'`).Scan(&d)
	t.Logf("with 2 active blacklist rows, is_deleted=%d (want 1)", d)
	if d != 1 {
		t.Errorf("BUG: COUNT(*)->bool scan fails when >1 active entries; is_deleted=%d, cascade skipped", d)
	}
}
