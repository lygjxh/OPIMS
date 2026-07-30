package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"opims/database"
	"opims/services"

	"github.com/xuri/excelize/v2"
)

// 本文件锁住 2026-07-30 一批回归缺陷，成因都是"错误被吞掉"或"字段漏映射"，
// 单看代码不明显，只有跑起来才暴露。

func setupTestDB(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := database.Init(filepath.Join(dir, "opims.db")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { database.Close() })
	return dir
}

func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	root := t.TempDir()
	return NewHandler(root, services.NewRootDir(root))
}

// 库必须真的跑在 WAL 模式：DSN 参数写成 mattn 语法时会被 modernc 驱动静默忽略，
// 表面看不出任何异常，只是 WAL 一直没生效。
func TestDatabaseRunsInWALMode(t *testing.T) {
	setupTestDB(t)
	var mode string
	if err := database.DB.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q, 期望 wal（检查 DSN 是否被驱动忽略）", mode)
	}
}

// scanProject 查不到记录时必须返回 sql.ErrNoRows，不能返回空对象充当"已存在"。
func TestScanProjectReturnsErrNoRows(t *testing.T) {
	setupTestDB(t)
	p, err := scanProject(database.DB.QueryRow(
		"SELECT * FROM projects WHERE short_name=? AND is_deleted=0", "不存在的项目"))
	if err != sql.ErrNoRows {
		t.Errorf("err = %v, 期望 sql.ErrNoRows", err)
	}
	if p != nil {
		t.Errorf("查不到记录时仍返回了对象 %+v", *p)
	}
}

// 全新项目走默认冲突策略 skip 时必须真的导入，不能被误判成同名冲突全部跳过。
func TestImportProjectsInsertsNewProject(t *testing.T) {
	setupTestDB(t)
	h := newTestHandler(t)

	xlsx := filepath.Join(t.TempDir(), "projects.xlsx")
	writeProjectImportFile(t, xlsx, "尼日利亚回归测试项目")

	body, contentType := multipartFile(t, "file", xlsx)
	req := httptest.NewRequest("POST", "/api/projects/import", body)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	h.ImportProjects(w, req)

	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var res struct {
		Imported int `json:"imported"`
		Skipped  int `json:"skipped"`
	}
	json.Unmarshal(w.Body.Bytes(), &res)
	if res.Imported != 1 {
		t.Errorf("imported=%d skipped=%d，期望 imported=1", res.Imported, res.Skipped)
	}

	var n int
	database.DB.QueryRow("SELECT COUNT(*) FROM projects WHERE project_name=?", "尼日利亚回归测试项目").Scan(&n)
	if n != 1 {
		t.Errorf("库里项目数=%d，期望 1", n)
	}
}

// 已软删除的项目查详情要返回 404，不能返回 200 + 全零值对象。
func TestProjectByIDReturns404WhenDeleted(t *testing.T) {
	setupTestDB(t)
	res, err := database.DB.Exec(
		"INSERT INTO projects (short_name, project_name, is_deleted) VALUES (?,?,1)", "已删项目", "已删项目")
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()

	h := newTestHandler(t)
	req := httptest.NewRequest("GET", "/api/projects/"+strconv.FormatInt(id, 10), nil)
	w := httptest.NewRecorder()
	h.ProjectByID(w, req)

	if w.Code != 404 {
		t.Errorf("status=%d body=%s，期望 404", w.Code, w.Body.String())
	}
}

// 账期和文件在同一个 multipart 请求里提交，后端必须都能读到。
// 用 MultipartReader 取文件会让后续 FormValue 恒为空，且与字段先后顺序无关，
// 所以这里两种顺序都测。
func TestSaveUploadFileKeepsOtherFormFields(t *testing.T) {
	xlsx := filepath.Join(t.TempDir(), "ledger.xlsx")
	if err := os.WriteFile(xlsx, []byte("dummy"), 0644); err != nil {
		t.Fatal(err)
	}

	for _, fileFirst := range []bool{true, false} {
		name := "文件在前"
		if !fileFirst {
			name = "账期在前"
		}
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			mw := multipart.NewWriter(&buf)
			writeFile := func() {
				fw, _ := mw.CreateFormFile("file", "ledger.xlsx")
				fw.Write([]byte("dummy"))
			}
			if fileFirst {
				writeFile()
				mw.WriteField("period", "2026-06")
			} else {
				mw.WriteField("period", "2026-06")
				writeFile()
			}
			mw.Close()

			req := httptest.NewRequest("POST", "/api/subcontract/import", &buf)
			req.Header.Set("Content-Type", mw.FormDataContentType())

			tmpPath, err := saveUploadFile(req, "file")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpPath)

			if got := req.FormValue("period"); got != "2026-06" {
				t.Errorf("period=%q，期望 2026-06", got)
			}
		})
	}
}

// 分包台账详情要带上账期，写入什么就读出什么。
func TestSubcontractByIDReturnsPeriod(t *testing.T) {
	setupTestDB(t)
	res, err := database.DB.Exec(
		"INSERT INTO project_subcontract (project_name, sub_name, period) VALUES (?,?,?)",
		"回归测试项目", "回归测试分包商", "2099-01")
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()

	h := newTestHandler(t)
	req := httptest.NewRequest("GET", "/api/subcontract/"+strconv.FormatInt(id, 10), nil)
	w := httptest.NewRecorder()
	h.SubcontractByID(w, req)

	if w.Code != 200 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var rec map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &rec)
	if rec["period"] != "2099-01" {
		t.Errorf("period=%v，期望 2099-01", rec["period"])
	}
}

// 备份 → 写入 → 恢复 → 查询：恢复后必须回到备份时的内容。
func TestBackupRestoreDropsWritesAfterBackup(t *testing.T) {
	dir := setupTestDB(t)
	h := newTestHandler(t)

	database.DB.Exec("INSERT INTO projects (short_name, project_name) VALUES (?,?)", "基线项目", "基线项目")

	backupDir := filepath.Join(dir, "bak")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(map[string]string{"path": backupDir})
	w := httptest.NewRecorder()
	h.Backup(w, httptest.NewRequest("POST", "/api/config/backup", bytes.NewReader(body)))
	if w.Code != 200 {
		t.Fatalf("备份失败 status=%d body=%s", w.Code, w.Body.String())
	}
	var backup struct {
		Path string `json:"path"`
	}
	json.Unmarshal(w.Body.Bytes(), &backup)

	// 备份之后写入的探针数据，恢复后应当消失
	database.DB.Exec("INSERT INTO projects (short_name, project_name) VALUES (?,?)", "探针项目", "探针项目")

	body2, _ := json.Marshal(map[string]string{"path": backup.Path})
	w2 := httptest.NewRecorder()
	h.Restore(w2, httptest.NewRequest("POST", "/api/config/restore", bytes.NewReader(body2)))
	if w2.Code != 200 {
		t.Fatalf("恢复失败 status=%d body=%s", w2.Code, w2.Body.String())
	}

	var probe, baseline int
	database.DB.QueryRow("SELECT COUNT(*) FROM projects WHERE short_name='探针项目'").Scan(&probe)
	database.DB.QueryRow("SELECT COUNT(*) FROM projects WHERE short_name='基线项目'").Scan(&baseline)
	if probe != 0 {
		t.Errorf("恢复后探针项目仍有 %d 条，期望 0", probe)
	}
	if baseline != 1 {
		t.Errorf("恢复后基线项目有 %d 条，期望 1（备份内容丢失）", baseline)
	}
}

// 同一分钟内连续备份两次不能互相覆盖。
func TestBackupTwiceKeepsBothFiles(t *testing.T) {
	dir := setupTestDB(t)
	h := newTestHandler(t)
	backupDir := filepath.Join(dir, "bak")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}

	names := map[string]bool{}
	for i := 0; i < 2; i++ {
		body, _ := json.Marshal(map[string]string{"path": backupDir})
		w := httptest.NewRecorder()
		h.Backup(w, httptest.NewRequest("POST", "/api/config/backup", bytes.NewReader(body)))
		if w.Code != 200 {
			t.Fatalf("备份失败 status=%d body=%s", w.Code, w.Body.String())
		}
		var res struct {
			Filename string `json:"filename"`
		}
		json.Unmarshal(w.Body.Bytes(), &res)
		names[res.Filename] = true
	}
	if len(names) != 2 {
		t.Errorf("两次备份文件名重复（%v），后一次会覆盖前一次", names)
	}
}

// writeProjectImportFile 造一份符合 ParseExcel 规则的最小导入文件：
// 第 2 行是表头（含"项目简称"即走带简称的列序），数据行首列必须是序号数字，
// 第 10 列须为"境外"才会进入导入范围。
func writeProjectImportFile(t *testing.T, path, projectName string) {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()
	sheet := "在建项目"
	f.SetSheetName(f.GetSheetName(0), sheet)

	rows := [][]string{
		{"境外在建项目情况表"},
		{"序号", "项目简称", "合同编号", "实施单位", "项目名称", "合同额", "预算额", "承包范围", "工程重点", "境内外"},
		{"", "", "", "", "", "", "", "", "", ""},
		{"1", "回归测试简称", "14HJ-2026JH001", "海外运营中心", projectName, "1000", "900", "施工总承包", "无", "境外"},
		{"合计", "", "", "", "", "1000", "900", "", "", ""},
	}
	for r, row := range rows {
		cell, _ := excelize.CoordinatesToCellName(1, r+1)
		if err := f.SetSheetRow(sheet, cell, &row); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
}

// multipartFile 把磁盘上的文件包成 multipart 请求体。
func multipartFile(t *testing.T, field, path string) (*bytes.Buffer, string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, err := mw.CreateFormFile(field, filepath.Base(path))
	if err != nil {
		t.Fatal(err)
	}
	fw.Write(data)
	mw.Close()
	return &buf, mw.FormDataContentType()
}
