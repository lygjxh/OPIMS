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
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// 分包商库单 Sheet 模板列顺序（0-based）。第 25–29 列为资质文件占位列，不入库。
var subImportHeaders = []string{
	"上报项目", "分包商编号", "分包商简称", "分包商名称", "国别", "企业性质",
	"成立日期", "注册资金", "法人及身份证", "法人联系方式", "委托代理人及身份证", "委托代理人联系方式",
	"所在地区（省市区/县）", "详细地址", "资质类别及等级", "资信等级", "分包商等级", "分包商分级",
	"经营范围", "推荐人", "上报单位", "单位负责人", "分类", "专业", "备注",
	"营业执照", "组织机构代码证", "税务登记证", "安全生产许可证", "资质证书", "关联单位",
}

// 资质文件类型（文件命名规则：分包商编号-类型*，扩展名不限）。
var qualDocTypes = []string{"营业执照", "组织机构代码证", "税务登记证", "安全生产许可证", "资质证书"}

// 入库列（26 列，与 SubcontractorBase 一一对应，不含 id/时间戳/汇总列）。
const subBaseCols = `sub_no,report_project,short_name,full_name,country,enterprise_type,
	established_date,reg_capital,legal_rep,legal_rep_phone,agent,agent_phone,region,address,
	qualification,credit_rating,grade,classification,business_scope,recommender,report_unit,
	unit_head,category,profession,notes,assoc_unit`

func subBaseArgs(b *models.SubcontractorBase) []interface{} {
	return []interface{}{
		b.SubNo, b.ReportProject, b.ShortName, b.FullName, b.Country, b.EnterpriseType,
		b.EstablishedDate, b.RegCapital, b.LegalRep, b.LegalRepPhone, b.Agent, b.AgentPhone, b.Region, b.Address,
		b.Qualification, b.CreditRating, b.Grade, b.Classification, b.BusinessScope, b.Recommender, b.ReportUnit,
		b.UnitHead, b.Category, b.Profession, b.Notes, b.AssocUnit,
	}
}

func scanSubBase(sc rowScanner, b *models.SubcontractorBase) error {
	return sc.Scan(&b.ID, &b.SubNo, &b.ReportProject, &b.ShortName, &b.FullName, &b.Country, &b.EnterpriseType,
		&b.EstablishedDate, &b.RegCapital, &b.LegalRep, &b.LegalRepPhone, &b.Agent, &b.AgentPhone, &b.Region, &b.Address,
		&b.Qualification, &b.CreditRating, &b.Grade, &b.Classification, &b.BusinessScope, &b.Recommender, &b.ReportUnit,
		&b.UnitHead, &b.Category, &b.Profession, &b.Notes, &b.AssocUnit)
}

const subBaseSelect = `SELECT id,` + subBaseCols + ` FROM subcontractors_base `

// buildSubWhere 组装列表/导出的筛选条件。
func buildSubWhere(q map[string][]string) (string, []interface{}) {
	get := func(k string) string {
		if v, ok := q[k]; ok && len(v) > 0 {
			return v[0]
		}
		return ""
	}
	where := "WHERE 1=1"
	args := []interface{}{}
	for _, f := range []struct{ param, col string }{
		{"country", "country"},
		{"category", "category"},
		{"profession", "profession"},
		{"grade", "grade"},
		{"classification", "classification"},
	} {
		if v := get(f.param); v != "" {
			where += " AND " + f.col + "=?"
			args = append(args, v)
		}
	}
	if kw := get("keyword"); kw != "" {
		where += " AND (sub_no LIKE ? OR short_name LIKE ? OR full_name LIKE ?)"
		like := "%" + kw + "%"
		args = append(args, like, like, like)
	}
	return where, args
}

// subAggregates 汇总项目分包中每个分包商（按名称）的合同数/总金额。
// 同一 contract_no 跨多账期只取最新账期一条，避免重复计。
func subAggregates() map[string]struct {
	Count  int
	Amount float64
} {
	res := map[string]struct {
		Count  int
		Amount float64
	}{}
	rows, err := database.DB.Query(`
		SELECT sub_name, COUNT(*), COALESCE(SUM(amt),0) FROM (
			SELECT sub_name, contract_no, MAX(contract_amount) amt
			FROM project_subcontract ps
			WHERE contract_no!='' AND NOT EXISTS (
				SELECT 1 FROM project_subcontract x
				WHERE x.contract_no=ps.contract_no AND x.contract_no!='' AND x.period > ps.period)
			GROUP BY sub_name, contract_no
		) GROUP BY sub_name`)
	if err != nil {
		return res
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var cnt int
		var amt float64
		rows.Scan(&name, &cnt, &amt)
		res[name] = struct {
			Count  int
			Amount float64
		}{cnt, amt}
	}
	return res
}

// unmatchedSubNames 返回项目分包台账里、分包商库中找不到对应「分包商名称」的名字。
func unmatchedSubNames() []string {
	rows, err := database.DB.Query(`
		SELECT DISTINCT sub_name FROM project_subcontract
		WHERE sub_name!='' AND sub_name NOT IN (SELECT full_name FROM subcontractors_base)
		ORDER BY sub_name`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var n string
		rows.Scan(&n)
		out = append(out, n)
	}
	return out
}

// SubcontractorsList handles GET (list) / POST (create) for subcontractor library.
func (h *Handler) SubcontractorsList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case "GET":
		where, args := buildSubWhere(r.URL.Query())
		rows, err := database.DB.Query(subBaseSelect+where+" ORDER BY id DESC", args...)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		list := []models.SubcontractorBase{}
		for rows.Next() {
			var b models.SubcontractorBase
			if err := scanSubBase(rows, &b); err != nil {
				continue
			}
			list = append(list, b)
		}
		// 关闭外层游标后再跑汇总查询——单连接池（MaxOpenConns=1）下，
		// 持有未关闭的游标时再发起查询会死锁。
		rows.Close()

		agg := subAggregates()
		for i := range list {
			if a, ok := agg[list[i].FullName]; ok {
				list[i].ContractCount = a.Count
				list[i].TotalAmount = a.Amount
			}
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"records":   list,
			"unmatched": unmatchedSubNames(),
		})

	case "POST":
		var b models.SubcontractorBase
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(b.SubNo) == "" {
			http.Error(w, "分包商编号不能为空", http.StatusBadRequest)
			return
		}
		ph := "?" + strings.Repeat(",?", 25)
		_, err := database.DB.Exec("INSERT INTO subcontractors_base ("+subBaseCols+") VALUES ("+ph+")", subBaseArgs(&b)...)
		if err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				http.Error(w, "分包商编号已存在", http.StatusBadRequest)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"ok": "created"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// SubcontractorsByID handles GET / PUT / DELETE for a single subcontractor.
func (h *Handler) SubcontractorsByID(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	id, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/api/subcontractors/"))

	switch r.Method {
	case "GET":
		var b models.SubcontractorBase
		if err := scanSubBase(database.DB.QueryRow(subBaseSelect+"WHERE id=?", id), &b); err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		if a, ok := subAggregates()[b.FullName]; ok {
			b.ContractCount = a.Count
			b.TotalAmount = a.Amount
		}

		// 黑名单状态（按分包商名称匹配）
		var blStatus, blLevel, blDate, blReason string
		database.DB.QueryRow(
			`SELECT status, restrict_level, list_date, list_reason FROM subcontractor_blacklist WHERE sub_full_name=? ORDER BY id DESC LIMIT 1`,
			b.FullName).Scan(&blStatus, &blLevel, &blDate, &blReason)

		// 关联单位是否在黑名单（列入中）
		assocBlacklisted := false
		if strings.TrimSpace(b.AssocUnit) != "" {
			var cnt int
			database.DB.QueryRow(
				`SELECT COUNT(*) FROM subcontractor_blacklist WHERE (sub_full_name=? OR sub_short_name=?) AND status='列入中'`,
				b.AssocUnit, b.AssocUnit).Scan(&cnt)
			assocBlacklisted = cnt > 0
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"base":              b,
			"blacklist_status":  blStatus,
			"restrict_level":    blLevel,
			"list_date":         blDate,
			"list_reason":       blReason,
			"assoc_blacklisted": assocBlacklisted,
			"quals":             h.scanQuals(b.SubNo),
		})

	case "PUT":
		var b models.SubcontractorBase
		if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		set := `sub_no=?,report_project=?,short_name=?,full_name=?,country=?,enterprise_type=?,
			established_date=?,reg_capital=?,legal_rep=?,legal_rep_phone=?,agent=?,agent_phone=?,region=?,address=?,
			qualification=?,credit_rating=?,grade=?,classification=?,business_scope=?,recommender=?,report_unit=?,
			unit_head=?,category=?,profession=?,notes=?,assoc_unit=?,updated_at=CURRENT_TIMESTAMP`
		args := append(subBaseArgs(&b), id)
		if _, err := database.DB.Exec("UPDATE subcontractors_base SET "+set+" WHERE id=?", args...); err != nil {
			if strings.Contains(err.Error(), "UNIQUE") {
				http.Error(w, "分包商编号已存在", http.StatusBadRequest)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"ok": "updated"})

	case "DELETE":
		database.DB.Exec("DELETE FROM subcontractors_base WHERE id=?", id)
		json.NewEncoder(w).Encode(map[string]string{"ok": "deleted"})

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// SubcontractorsImport parses the single-sheet Excel template and upserts by 分包商编号.
// POST /api/subcontractors/import — multipart form with file field "file".
func (h *Handler) SubcontractorsImport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, "file too large", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	f, err := excelize.OpenReader(file)
	if err != nil {
		http.Error(w, "invalid excel: "+err.Error(), http.StatusBadRequest)
		return
	}
	defer f.Close()

	rows, err := f.GetRows(f.GetSheetName(0))
	if err != nil || len(rows) < 2 {
		http.Error(w, "解析结果为空（请检查文件是否为分包商库模板）", http.StatusBadRequest)
		return
	}

	added, updated := 0, 0
	var errs []map[string]string
	for i, row := range rows[1:] {
		cell := func(j int) string {
			if j < len(row) {
				return strings.TrimSpace(row[j])
			}
			return ""
		}
		if cell(1) == "" && cell(3) == "" { // 空行
			continue
		}
		b := models.SubcontractorBase{
			ReportProject: cell(0), SubNo: cell(1), ShortName: cell(2), FullName: cell(3),
			Country: cell(4), EnterpriseType: cell(5), EstablishedDate: cell(6), RegCapital: cell(7),
			LegalRep: cell(8), LegalRepPhone: cell(9), Agent: cell(10), AgentPhone: cell(11),
			Region: cell(12), Address: cell(13), Qualification: cell(14), CreditRating: cell(15),
			Grade: cell(16), Classification: cell(17), BusinessScope: cell(18), Recommender: cell(19),
			ReportUnit: cell(20), UnitHead: cell(21), Category: cell(22), Profession: cell(23),
			Notes: cell(24), AssocUnit: cell(30),
		}
		if b.SubNo == "" || b.FullName == "" {
			errs = append(errs, map[string]string{"row": fmt.Sprintf("第%d行", i+2), "msg": "缺分包商编号或名称，已跳过"})
			continue
		}
		var already int
		database.DB.QueryRow("SELECT COUNT(*) FROM subcontractors_base WHERE sub_no=?", b.SubNo).Scan(&already)
		ph := "?" + strings.Repeat(",?", 25)
		_, err := database.DB.Exec(
			"INSERT INTO subcontractors_base ("+subBaseCols+") VALUES ("+ph+") "+
				`ON CONFLICT(sub_no) DO UPDATE SET
				  report_project=excluded.report_project, short_name=excluded.short_name, full_name=excluded.full_name,
				  country=excluded.country, enterprise_type=excluded.enterprise_type, established_date=excluded.established_date,
				  reg_capital=excluded.reg_capital, legal_rep=excluded.legal_rep, legal_rep_phone=excluded.legal_rep_phone,
				  agent=excluded.agent, agent_phone=excluded.agent_phone, region=excluded.region, address=excluded.address,
				  qualification=excluded.qualification, credit_rating=excluded.credit_rating, grade=excluded.grade,
				  classification=excluded.classification, business_scope=excluded.business_scope, recommender=excluded.recommender,
				  report_unit=excluded.report_unit, unit_head=excluded.unit_head, category=excluded.category,
				  profession=excluded.profession, notes=excluded.notes, assoc_unit=excluded.assoc_unit,
				  updated_at=CURRENT_TIMESTAMP`,
			subBaseArgs(&b)...)
		if err != nil {
			errs = append(errs, map[string]string{"row": fmt.Sprintf("第%d行", i+2), "msg": err.Error()})
			continue
		}
		if already > 0 {
			updated++
		} else {
			added++
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"added": added, "updated": updated, "skipped": len(errs), "errors": errs,
	})
}

// writeSubSheet 把表头 + 数据写入一个工作表（导出/模板共用）。
func writeSubSheet(f *excelize.File, sheet string, records []models.SubcontractorBase) {
	for i, hdr := range subImportHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, hdr)
	}
	for ri, b := range records {
		vals := []interface{}{
			b.ReportProject, b.SubNo, b.ShortName, b.FullName, b.Country, b.EnterpriseType,
			b.EstablishedDate, b.RegCapital, b.LegalRep, b.LegalRepPhone, b.Agent, b.AgentPhone,
			b.Region, b.Address, b.Qualification, b.CreditRating, b.Grade, b.Classification,
			b.BusinessScope, b.Recommender, b.ReportUnit, b.UnitHead, b.Category, b.Profession, b.Notes,
			"", "", "", "", "", b.AssocUnit, // 25–29 资质文件占位列留空
		}
		for j, v := range vals {
			cell, _ := excelize.CoordinatesToCellName(j+1, ri+2)
			f.SetCellValue(sheet, cell, v)
		}
	}
}

// SubcontractorsExportTemplate 下载单 Sheet 模板（表头 + 一行示例）。
// GET /api/subcontractors/export/template
func (h *Handler) SubcontractorsExportTemplate(w http.ResponseWriter, r *http.Request) {
	f := excelize.NewFile()
	defer f.Close()
	sheet := "分包商库"
	f.SetSheetName("Sheet1", sheet)
	sample := []models.SubcontractorBase{{
		ReportProject: "印尼CAA项目", SubNo: "202204-00001", ShortName: "南京泰然",
		FullName: "南京泰然工程建设有限公司", Country: "中国", EnterpriseType: "有限责任公司",
		EstablishedDate: "2022-01-14", RegCapital: "3000000", LegalRep: "张三 3201XXXXXXXXXXXXXX",
		LegalRepPhone: "13800000000", Region: "江苏省 南京市", Address: "示例地址",
		Grade: "正常使用", Classification: "核心层分包商", Category: "施工", Profession: "管道安装",
	}}
	writeSubSheet(f, sheet, sample)
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=subcontractor_template.xlsx")
	f.Write(w)
}

// SubcontractorsExport 导出全部或当前筛选结果。
// GET /api/subcontractors/export?scope=all|filtered&<筛选参数...>
func (h *Handler) SubcontractorsExport(w http.ResponseWriter, r *http.Request) {
	where, args := "WHERE 1=1", []interface{}{}
	if r.URL.Query().Get("scope") != "all" {
		where, args = buildSubWhere(r.URL.Query())
	}
	rows, err := database.DB.Query(subBaseSelect+where+" ORDER BY id DESC", args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()
	var records []models.SubcontractorBase
	for rows.Next() {
		var b models.SubcontractorBase
		if scanSubBase(rows, &b) == nil {
			records = append(records, b)
		}
	}

	f := excelize.NewFile()
	defer f.Close()
	sheet := "分包商库"
	f.SetSheetName("Sheet1", sheet)
	writeSubSheet(f, sheet, records)
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=subcontractors_export.xlsx")
	f.Write(w)
}

// scanQuals 扫描资质文件目录，返回 5 类资质文件的实际文件名（存在才有值）。
func (h *Handler) scanQuals(subNo string) map[string]string {
	out := map[string]string{}
	if strings.TrimSpace(subNo) == "" {
		return out
	}
	entries, err := os.ReadDir(h.root.QualDir())
	if err != nil {
		return out
	}
	for _, t := range qualDocTypes {
		prefix := subNo + "-" + t
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			if strings.HasPrefix(e.Name(), prefix) {
				out[t] = e.Name()
				break
			}
		}
	}
	return out
}

// SubcontractorsQuals 返回某分包商的资质文件存在情况。
// GET /api/subcontractors/quals?no=<分包商编号>
func (h *Handler) SubcontractorsQuals(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.scanQuals(r.URL.Query().Get("no")))
}

// SubcontractorsQualOpen 用本机默认程序打开某分包商的某类资质文件。
// GET /api/subcontractors/qual/open?no=<编号>&type=<资质类型>
func (h *Handler) SubcontractorsQualOpen(w http.ResponseWriter, r *http.Request) {
	no := strings.TrimSpace(r.URL.Query().Get("no"))
	typ := strings.TrimSpace(r.URL.Query().Get("type"))
	if no == "" || typ == "" {
		http.Error(w, "missing no or type", http.StatusBadRequest)
		return
	}
	fname := h.scanQuals(no)[typ]
	if fname == "" {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	dir := h.root.QualDir()
	target := filepath.Join(dir, fname)
	// 根目录边界校验，挡住 .. 越权
	absDir, _ := filepath.Abs(dir)
	absTarget, _ := filepath.Abs(target)
	if !strings.HasPrefix(absTarget, absDir+string(os.PathSeparator)) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if _, err := os.Stat(target); err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	exec.Command("rundll32", "url.dll,FileProtocolHandler", target).Start()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"ok": "opened"})
}
