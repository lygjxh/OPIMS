package services

import (
	"opims/database"
)

// OnBlacklistChanged is called after any blacklist create/update/delete.
// It syncs the cascading disable state for linked local subsidiaries.
func OnBlacklistChanged(subShortName string, action string) {
	// Find linked local subsidiaries of this domestic parent
	rows, err := database.DB.Query(
		`SELECT short_name FROM subcontractors_base 
		 WHERE parent_short_name=? AND registration_type='当地注册'`, subShortName)
	if err != nil {
		return
	}
	defer rows.Close()

	var locals []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		locals = append(locals, name)
	}

	if len(locals) == 0 {
		return
	}

	// Check if parent is currently blacklisted (status=列入中)
	var activeCount int
	database.DB.QueryRow(
		`SELECT COUNT(*) FROM subcontractor_blacklist
		 WHERE sub_short_name=? AND status='列入中'`, subShortName,
	).Scan(&activeCount)

	if action == "列入" || action == "编辑" {
		if activeCount > 0 {
			// Mark local subsidiaries as disabled in project_subcontract
			for _, local := range locals {
				database.DB.Exec(
					`UPDATE project_subcontract SET is_deleted=1 WHERE sub_name=?`, local)
			}
		}
	} else if action == "拉出" || action == "删除" {
		// Check if there are any other active blacklist entries for this parent
		var remaining int
		database.DB.QueryRow(
			`SELECT COUNT(*) FROM subcontractor_blacklist 
			 WHERE sub_short_name=? AND status='列入中'`, subShortName,
		).Scan(&remaining)
		if remaining == 0 {
			// Restore local subsidiaries
			for _, local := range locals {
				database.DB.Exec(
					`UPDATE project_subcontract SET is_deleted=0 WHERE sub_name=?`, local)
			}
		}
	}
}
