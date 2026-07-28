package services

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"opims/data"
	"opims/database"
)

// 进度管理 · 收文改名归档（一期 1c）
// 依据《需求说明书 V1.1》4.4 与决策 13。
//
// ⚠️ 这是整个 OPIMS 第一个「写云盘文件」的功能，因此附加三条硬性要求：
//   1. 先复制到目标位置并校验，确认成功后才删除原件——不用 rename 一步到位，
//      跨盘符/云盘同步目录下 rename 可能失败并留下半成品
//   2. 每次操作写日志，可据此回滚
//   3. 目标文件已存在时绝不静默覆盖，一律报冲突交由人工决定

// ArchivePlan 一次归档的预演结果（不落盘，供界面确认）。
type ArchivePlan struct {
	SourcePath string `json:"source_path"`
	SourceName string `json:"source_name"`
	Project    string `json:"project"`
	Code       string `json:"code"`
	CodeName   string `json:"code_name"`
	Period     string `json:"period"`
	Version    int    `json:"version"`
	TargetDir  string `json:"target_dir"`
	TargetName string `json:"target_name"`
	TargetPath string `json:"target_path"`
	Conflict   bool   `json:"conflict"`  // 目标已存在
	Suggested  string `json:"suggested"` // 冲突时建议的新版本名
	Err        string `json:"error"`
}

// ArchiveResult 实际执行结果。
type ArchiveResult struct {
	OK         bool   `json:"ok"`
	SourcePath string `json:"source_path"`
	TargetPath string `json:"target_path"`
	LogID      int64  `json:"log_id"`
	Err        string `json:"error"`
}

// PlanArchive 预演：根据用户选定的项目/类型/周期算出目标路径，检查冲突。
// 不做任何文件写入。
func PlanArchive(projectsDir, srcPath, project, code, period string, version int) ArchivePlan {
	p := ArchivePlan{
		SourcePath: srcPath, SourceName: filepath.Base(srcPath),
		Project: project, Code: code, Period: period, Version: version,
	}

	st, err := os.Stat(srcPath)
	if err != nil {
		p.Err = "源文件不存在或不可读：" + err.Error()
		return p
	}
	if st.IsDir() {
		p.Err = "源路径是目录，不是文件"
		return p
	}

	item, ok := data.FindChecklistItem(code)
	if !ok {
		p.Err = "未知的文件类型代码：" + code
		return p
	}
	p.CodeName = item.Name

	p.TargetDir = filepath.Join(projectsDir, project, filepath.FromSlash(item.Folder))
	ext := filepath.Ext(srcPath)
	base := project + "-" + code + "-" + period
	if version > 1 {
		base += "-v" + fmt.Sprint(version)
	}
	p.TargetName = base + ext
	p.TargetPath = filepath.Join(p.TargetDir, p.TargetName)

	if _, err := os.Stat(p.TargetPath); err == nil {
		p.Conflict = true
		// 建议下一个可用版本号，而不是覆盖
		for v := max2(version, 1) + 1; v <= 99; v++ {
			cand := filepath.Join(p.TargetDir, project+"-"+code+"-"+period+"-v"+fmt.Sprint(v)+ext)
			if _, err := os.Stat(cand); os.IsNotExist(err) {
				p.Suggested = filepath.Base(cand)
				break
			}
		}
	}
	return p
}

// DoArchive 执行归档：复制 → 校验 → 删原件 → 写日志。
// overwrite 为 true 时才允许覆盖已存在的目标（由界面二次确认后传入）。
func DoArchive(projectsDir, srcPath, project, code, period string, version int, overwrite bool, operator string) ArchiveResult {
	res := ArchiveResult{SourcePath: srcPath}

	plan := PlanArchive(projectsDir, srcPath, project, code, period, version)
	if plan.Err != "" {
		res.Err = plan.Err
		return res
	}
	res.TargetPath = plan.TargetPath
	if plan.Conflict && !overwrite {
		res.Err = "目标文件已存在：" + plan.TargetName + "。请改用新版本号，或显式确认覆盖"
		return res
	}

	if err := os.MkdirAll(plan.TargetDir, 0755); err != nil {
		res.Err = "创建目标目录失败：" + err.Error()
		return res
	}

	// 1) 先复制。不用 os.Rename——跨盘符或云盘同步目录下可能失败并留下半成品
	if err := copyFileVerified(srcPath, plan.TargetPath); err != nil {
		res.Err = "复制失败：" + err.Error()
		return res
	}

	// 2) 复制成功后才删原件。删除失败不算整体失败——文件已安全到位，
	//    只是源目录多留了一份，提示用户手工清理即可
	delWarn := ""
	if err := os.Remove(srcPath); err != nil {
		delWarn = "（原件未能删除，需手工清理：" + err.Error() + "）"
	}

	// 3) 写日志，供回滚与追溯
	id, _ := logArchive(operator, srcPath, plan.TargetPath, plan.Project, plan.Code, plan.Period, overwrite)
	res.OK = true
	res.LogID = id
	res.Err = delWarn
	return res
}

// copyFileVerified 复制并校验字节数一致，避免复制到一半就当成功。
func copyFileVerified(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	si, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	n, cerr := io.Copy(out, in)
	if err := out.Close(); err != nil && cerr == nil {
		cerr = err
	}
	if cerr != nil {
		os.Remove(dst) // 复制失败要清掉半成品，不能留残骸
		return cerr
	}
	if n != si.Size() {
		os.Remove(dst)
		return fmt.Errorf("字节数不一致：源 %d，写入 %d", si.Size(), n)
	}
	return nil
}

func logArchive(operator, src, dst, project, code, period string, overwrite bool) (int64, error) {
	ov := 0
	if overwrite {
		ov = 1
	}
	r, err := database.DB.Exec(`
		INSERT INTO archive_log (operator, source_path, target_path, project, code, period, overwrite, undone)
		VALUES (?,?,?,?,?,?,?,0)`,
		operator, src, dst, project, code, period, ov)
	if err != nil {
		return 0, err
	}
	return r.LastInsertId()
}

// ArchiveLogEntry 一条归档日志。
type ArchiveLogEntry struct {
	ID         int64  `json:"id"`
	Operator   string `json:"operator"`
	SourcePath string `json:"source_path"`
	TargetPath string `json:"target_path"`
	Project    string `json:"project"`
	Code       string `json:"code"`
	Period     string `json:"period"`
	Undone     bool   `json:"undone"`
	CreatedAt  string `json:"created_at"`
	CanUndo    bool   `json:"can_undo"` // 目标还在、源位置未被占用
}

// ListArchiveLog 返回最近的归档记录（默认 50 条）。
func ListArchiveLog(limit int) ([]ArchiveLogEntry, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := database.DB.Query(`
		SELECT id, operator, source_path, target_path, project, code, period, undone, created_at
		FROM archive_log ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ArchiveLogEntry
	for rows.Next() {
		var e ArchiveLogEntry
		var undone int
		if err := rows.Scan(&e.ID, &e.Operator, &e.SourcePath, &e.TargetPath,
			&e.Project, &e.Code, &e.Period, &undone, &e.CreatedAt); err != nil {
			continue
		}
		e.Undone = undone == 1
		if !e.Undone {
			_, tErr := os.Stat(e.TargetPath)
			_, sErr := os.Stat(e.SourcePath)
			e.CanUndo = tErr == nil && os.IsNotExist(sErr)
		}
		out = append(out, e)
	}
	return out, nil
}

// UndoArchive 回滚一次归档：把文件从目标位置搬回原位置。
func UndoArchive(id int64) error {
	var src, dst string
	var undone int
	err := database.DB.QueryRow(
		`SELECT source_path, target_path, undone FROM archive_log WHERE id=?`, id).
		Scan(&src, &dst, &undone)
	if err != nil {
		return fmt.Errorf("未找到归档记录 #%d", id)
	}
	if undone == 1 {
		return fmt.Errorf("该记录已回滚过")
	}
	if _, err := os.Stat(dst); err != nil {
		return fmt.Errorf("目标文件已不在原处，无法回滚：%s", dst)
	}
	if _, err := os.Stat(src); err == nil {
		return fmt.Errorf("源位置已存在同名文件，回滚会覆盖，请先手工处理：%s", src)
	}

	if err := os.MkdirAll(filepath.Dir(src), 0755); err != nil {
		return err
	}
	if err := copyFileVerified(dst, src); err != nil {
		return fmt.Errorf("回滚复制失败：%w", err)
	}
	if err := os.Remove(dst); err != nil {
		return fmt.Errorf("文件已复制回原位，但目标处未能删除，需手工清理：%w", err)
	}
	_, err = database.DB.Exec(
		`UPDATE archive_log SET undone=1, undone_at=? WHERE id=?`,
		time.Now().Format("2006-01-02 15:04:05"), id)
	return err
}

// GuessFromName 由文件名猜测项目/类型/周期，供界面预填。
// 猜不出就返回空，由人工选择——不做过度猜测（需求 6A.5）。
func GuessFromName(name string, projects []ProjectBrief) (project, code, period string) {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	if m := fileRe.FindStringSubmatch(base); m != nil {
		guess := m[fileRe.SubexpIndex("name")]
		for _, p := range projects {
			if p.ShortName == guess {
				project = p.ShortName
				break
			}
		}
		code = m[fileRe.SubexpIndex("code")]
		period = m[fileRe.SubexpIndex("period")]
		return
	}
	// 文件名里含某个项目简称也算线索
	for _, p := range projects {
		if strings.Contains(base, p.ShortName) {
			project = p.ShortName
			break
		}
	}
	return
}

func max2(a, b int) int {
	if a > b {
		return a
	}
	return b
}
