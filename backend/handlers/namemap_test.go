package handlers

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

// 简称映射表的实际存放位置是 <根>/06.Received File/01.项目管理情况汇总表/。
// 根目录曾从「项目文件目录」上移一层到「海外运营中心」，当时漏改了这里的查找路径，
// 导致导入时匹配不到项目简称与合同编号。本测试锁住该路径，防止再次失配。
func TestLoadNameMappingFindsReceivedFileLocation(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "06.Received File", "01.项目管理情况汇总表")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	f := excelize.NewFile()
	defer f.Close()
	sheet := f.GetSheetName(0)
	// 列序：序号 / 项目简称 / 合同编号 / 项目名称
	rows := [][]string{
		{"序号", "项目简称", "合同编号", "项目名称"},
		{"1", "尼日利亚PLF项目", "14HJ-2023JH097", "尼日利亚PLF-LNG项目"},
		{"2", "阿布扎比LNG项目", "14HJ-2024JH011", "阿布扎比LNG储罐项目"},
	}
	for r, row := range rows {
		for c, v := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+1)
			f.SetCellValue(sheet, cell, v)
		}
	}
	if err := f.SaveAs(filepath.Join(dir, "海外项目简称.xlsx")); err != nil {
		t.Fatal(err)
	}

	m := loadNameMapping(root)
	if len(m) != 2 {
		t.Fatalf("应从 06.Received File/01.项目管理情况汇总表/ 读到 2 条映射，实际 %d 条", len(m))
	}
	got, ok := m["尼日利亚PLF-LNG项目"]
	if !ok {
		t.Fatal("未按项目名称建立映射")
	}
	if got.ShortName != "尼日利亚PLF项目" || got.ContractNo != "14HJ-2023JH097" {
		t.Errorf("映射内容不对：%+v", got)
	}
}

// 找不到映射表时应返回空 map 而不是崩溃——导入仍可继续，只是简称取项目全名
func TestLoadNameMappingMissing(t *testing.T) {
	if m := loadNameMapping(t.TempDir()); len(m) != 0 {
		t.Errorf("无映射表时应返回空 map，实际 %d 条", len(m))
	}
}
