package services

import (
	"os"
	"path/filepath"
	"testing"
)

func setupArchive(t *testing.T) (root, inbox, proj string) {
	t.Helper()
	initTestDB(t)
	base := t.TempDir()
	root = filepath.Join(base, "01.Project Files")
	inbox = filepath.Join(base, "06.Received File")
	proj = "尼日利亚PLF项目"
	for _, d := range []string{
		filepath.Join(root, proj, "06.Report", "Monthly Report"),
		filepath.Join(root, proj, "05.Schedule", "Time Bar"),
		inbox,
	} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}
	return
}

func writeSrc(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestArchiveHappyPath(t *testing.T) {
	root, inbox, proj := setupArchive(t)
	src := writeSrc(t, inbox, "七月月报最终版.pdf", "月报正文内容")

	plan := PlanArchive(root, src, proj, "MPR", "202607", 1)
	if plan.Err != "" {
		t.Fatalf("预演出错：%s", plan.Err)
	}
	want := proj + "-MPR-202607.pdf"
	if plan.TargetName != want {
		t.Errorf("目标文件名应为 %s，实际 %s", want, plan.TargetName)
	}
	if plan.Conflict {
		t.Error("目标不存在时不应报冲突")
	}

	res := DoArchive(root, src, proj, "MPR", "202607", 1, false, "张三")
	if !res.OK {
		t.Fatalf("归档失败：%s", res.Err)
	}
	// 目标已就位
	got, err := os.ReadFile(res.TargetPath)
	if err != nil || string(got) != "月报正文内容" {
		t.Fatalf("目标文件内容不对：%v %q", err, string(got))
	}
	// 原件已删除
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Error("归档后原件应已删除")
	}
	// 有日志可回滚
	logs, _ := ListArchiveLog(10)
	if len(logs) != 1 || !logs[0].CanUndo {
		t.Fatalf("应留下 1 条可回滚日志，实际 %+v", logs)
	}
}

// 目标已存在时绝不静默覆盖——这是需求 4.4 的硬性要求
func TestArchiveRefusesOverwrite(t *testing.T) {
	root, inbox, proj := setupArchive(t)
	dst := filepath.Join(root, proj, "06.Report", "Monthly Report", proj+"-MPR-202607.pdf")
	if err := os.WriteFile(dst, []byte("已有的重要文件"), 0644); err != nil {
		t.Fatal(err)
	}
	src := writeSrc(t, inbox, "新月报.pdf", "新内容")

	plan := PlanArchive(root, src, proj, "MPR", "202607", 1)
	if !plan.Conflict {
		t.Fatal("目标已存在时应报冲突")
	}
	if plan.Suggested != proj+"-MPR-202607-v2.pdf" {
		t.Errorf("应建议下一个版本号，实际 %q", plan.Suggested)
	}

	// 未显式确认覆盖 → 必须拒绝，且不得动任何文件
	res := DoArchive(root, src, proj, "MPR", "202607", 1, false, "张三")
	if res.OK {
		t.Fatal("未确认覆盖时不应执行归档")
	}
	if b, _ := os.ReadFile(dst); string(b) != "已有的重要文件" {
		t.Error("拒绝归档时不得改动已有目标文件")
	}
	if _, err := os.Stat(src); err != nil {
		t.Error("拒绝归档时不得删除源文件")
	}

	// 改用 v2 版本号即可正常归档，不覆盖原件
	res2 := DoArchive(root, src, proj, "MPR", "202607", 2, false, "张三")
	if !res2.OK {
		t.Fatalf("改用 v2 应能归档：%s", res2.Err)
	}
	if b, _ := os.ReadFile(dst); string(b) != "已有的重要文件" {
		t.Error("归档 v2 后原 v1 文件应保持不变")
	}
}

func TestArchiveUndo(t *testing.T) {
	root, inbox, proj := setupArchive(t)
	src := writeSrc(t, inbox, "待归档.xlsx", "台账内容")

	res := DoArchive(root, src, proj, "TBC", "202607", 1, false, "李四")
	if !res.OK {
		t.Fatalf("归档失败：%s", res.Err)
	}
	// 归档到了 Time Bar 子目录（清单里 TBC 的 Folder）
	if !filepath.IsAbs(res.TargetPath) || filepath.Base(filepath.Dir(res.TargetPath)) != "Time Bar" {
		t.Errorf("TBC 应归档到 05.Schedule/Time Bar，实际 %s", res.TargetPath)
	}

	if err := UndoArchive(res.LogID); err != nil {
		t.Fatalf("回滚失败：%v", err)
	}
	// 文件回到原处、目标处已清空
	if b, err := os.ReadFile(src); err != nil || string(b) != "台账内容" {
		t.Errorf("回滚后源文件应恢复：%v", err)
	}
	if _, err := os.Stat(res.TargetPath); !os.IsNotExist(err) {
		t.Error("回滚后目标处文件应已移除")
	}
	// 重复回滚要拒绝
	if err := UndoArchive(res.LogID); err == nil {
		t.Error("重复回滚应报错")
	}
}

// 源位置已被占用时不能回滚，否则会覆盖用户新放的文件
func TestUndoRefusesWhenSourceOccupied(t *testing.T) {
	root, inbox, proj := setupArchive(t)
	src := writeSrc(t, inbox, "待归档.pdf", "原内容")
	res := DoArchive(root, src, proj, "MPR", "202607", 1, false, "王五")
	if !res.OK {
		t.Fatal(res.Err)
	}
	// 用户又往原位置放了个同名文件
	writeSrc(t, inbox, "待归档.pdf", "用户新放的文件")

	if err := UndoArchive(res.LogID); err == nil {
		t.Fatal("源位置已被占用时应拒绝回滚")
	}
	if b, _ := os.ReadFile(src); string(b) != "用户新放的文件" {
		t.Error("拒绝回滚时不得覆盖用户新放的文件")
	}
}

func TestGuessFromName(t *testing.T) {
	projects := []ProjectBrief{{ShortName: "尼日利亚PLF项目"}, {ShortName: "阿布扎比LNG项目"}}

	// 已规范命名 → 三项都能猜出
	p, c, d := GuessFromName("尼日利亚PLF项目-MPR-202607.pdf", projects)
	if p != "尼日利亚PLF项目" || c != "MPR" || d != "202607" {
		t.Errorf("规范文件名解析不对：%q %q %q", p, c, d)
	}
	// 只含项目简称 → 只猜项目，不瞎猜类型和周期
	p, c, d = GuessFromName("尼日利亚PLF项目 六月进度汇报.pptx", projects)
	if p != "尼日利亚PLF项目" || c != "" || d != "" {
		t.Errorf("应只猜出项目：%q %q %q", p, c, d)
	}
	// 完全无线索 → 全空，交人工选择
	p, c, d = GuessFromName("扫描件0001.pdf", projects)
	if p != "" || c != "" || d != "" {
		t.Errorf("无线索时不应猜测：%q %q %q", p, c, d)
	}
}
