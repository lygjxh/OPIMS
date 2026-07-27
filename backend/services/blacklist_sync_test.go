package services

import (
	"opims/database"
	"path/filepath"
	"testing"
)

// TestOnBlacklistChangedCascade verifies that blacklisting a subcontractor
// cascades a disable (is_deleted=1) onto the project_subcontract rows of any
// subcontractor whose 关联单位(assoc_unit) points at it, and that de-listing restores them.
//
// 2026-07-26 起 subcontractors_base 改为单 Sheet 31 列结构：级联关系由
// registration_type/parent_short_name 改为 assoc_unit，本测试已同步更新。
// 注意：联动匹配的是 full_name（见 blacklist_sync.go 的 SELECT full_name），
// 因此 project_subcontract.sub_name 必须填全名。
func TestOnBlacklistChangedCascade(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "cascade.db")
	if err := database.Init(dbPath); err != nil {
		t.Fatalf("db init: %v", err)
	}
	defer database.Close()

	// 关联单位指向 PARENT1 的分包商
	_, err := database.DB.Exec(`INSERT INTO subcontractors_base
		(sub_no,short_name,full_name,country,category,profession,assoc_unit)
		VALUES ('S001','LOCAL1','Local One','印尼','施工','土建','PARENT1')`)
	if err != nil {
		t.Fatalf("insert sub: %v", err)
	}
	// 分包合同行的 sub_name 用全名，与联动逻辑一致
	_, err = database.DB.Exec(`INSERT INTO project_subcontract (sub_name) VALUES ('Local One')`)
	if err != nil {
		t.Fatalf("insert subcontract: %v", err)
	}

	getDeleted := func() int {
		var d int
		database.DB.QueryRow(`SELECT is_deleted FROM project_subcontract WHERE sub_name='Local One'`).Scan(&d)
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

// TestOnBlacklistChangedMultipleActiveEntries 覆盖同一分包商存在多条"列入中"
// 黑名单记录的情况：联动判断用的是 COUNT(*) > 0，多条记录也应正常级联
// （历史上曾把 COUNT 扫描进 bool 导致多条时失败，此测试守住该回归）。
func TestOnBlacklistChangedMultipleActiveEntries(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "cascade2.db")
	if err := database.Init(dbPath); err != nil {
		t.Fatalf("db init: %v", err)
	}
	defer database.Close()

	if _, err := database.DB.Exec(`INSERT INTO subcontractors_base
		(sub_no,short_name,full_name,country,category,profession,assoc_unit)
		VALUES ('S001','LOCAL1','Local One','印尼','施工','土建','PARENT1')`); err != nil {
		t.Fatalf("insert sub: %v", err)
	}
	if _, err := database.DB.Exec(`INSERT INTO project_subcontract (sub_name) VALUES ('Local One')`); err != nil {
		t.Fatalf("insert subcontract: %v", err)
	}

	// 同一分包商的两条"列入中"记录
	for i := 0; i < 2; i++ {
		if _, err := database.DB.Exec(`INSERT INTO subcontractor_blacklist
			(sub_short_name,sub_full_name,list_date,status) VALUES ('PARENT1','Parent','2026-07-25','列入中')`); err != nil {
			t.Fatalf("insert blacklist: %v", err)
		}
	}

	OnBlacklistChanged("PARENT1", "列入")

	var d int
	database.DB.QueryRow(`SELECT is_deleted FROM project_subcontract WHERE sub_name='Local One'`).Scan(&d)
	if d != 1 {
		t.Errorf("存在 2 条列入中记录时未级联禁用：is_deleted=%d，期望 1", d)
	}
}
