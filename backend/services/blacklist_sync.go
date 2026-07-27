package services

import (
	"opims/database"
)

// OnBlacklistChanged is called after any blacklist create/update/delete.
// 2026-07-26：级联关系由「国内母公司-当地公司」改为「关联单位」关系——
// 即某分包商的「关联单位」指向本次变更的分包商（按简称或全名匹配）时，联动禁用/恢复。
func OnBlacklistChanged(subShortName string, action string) {
	// 找出"关联单位"指向该分包商（按简称，或其黑名单记录里的全名）的其它分包商
	rows, err := database.DB.Query(
		`SELECT full_name FROM subcontractors_base
		 WHERE assoc_unit!='' AND (
		   assoc_unit=? OR
		   assoc_unit IN (SELECT sub_full_name FROM subcontractor_blacklist WHERE sub_short_name=?)
		 )`, subShortName, subShortName)
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

	// 该分包商当前是否处于"列入中"
	var activeCount int
	database.DB.QueryRow(
		`SELECT COUNT(*) FROM subcontractor_blacklist
		 WHERE sub_short_name=? AND status='列入中'`, subShortName,
	).Scan(&activeCount)

	if action == "列入" || action == "编辑" {
		if activeCount > 0 {
			for _, local := range locals {
				database.DB.Exec(`UPDATE project_subcontract SET is_deleted=1 WHERE sub_name=?`, local)
			}
		}
	} else if action == "拉出" || action == "删除" {
		if activeCount == 0 {
			for _, local := range locals {
				database.DB.Exec(`UPDATE project_subcontract SET is_deleted=0 WHERE sub_name=?`, local)
			}
		}
	}
}
