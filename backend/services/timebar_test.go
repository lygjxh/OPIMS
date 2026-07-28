package services

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

// 造一个与附件 E 模板同构的 xlsx：表头在第 5 行，数据自第 6 行起。
func makeTimeBarFile(t *testing.T, dir, project string, rows [][]string) {
	t.Helper()
	f := excelize.NewFile()
	sheet := "时效日历"
	f.SetSheetName("Sheet1", sheet)

	f.SetCellValue(sheet, "A1", "附件 E  合同时效日历（Time Bar Calendar）")
	f.SetCellValue(sheet, "A3", "项目简称")
	f.SetCellValue(sheet, "B3", project)

	hdr := []string{"序号", "合同条款号", "时效代码", "触发事件描述", "触发日期", "时限天数",
		"到期日", "剩余天数", "预警灯", "实际完成日", "状态", "责任人", "备注"}
	for i, h := range hdr {
		c, _ := excelize.CoordinatesToCellName(i+1, 5)
		f.SetCellValue(sheet, c, h)
	}
	for r, row := range rows {
		for i, v := range row {
			c, _ := excelize.CoordinatesToCellName(i+1, 6+r)
			f.SetCellValue(sheet, c, v)
		}
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := f.SaveAs(filepath.Join(dir, project+"-TBC-202607.xlsx")); err != nil {
		t.Fatal(err)
	}
}

func TestScanTimeBars(t *testing.T) {
	root := t.TempDir()
	proj := "尼日利亚PLF项目"
	dir := filepath.Join(root, proj, filepath.FromSlash("05.Schedule/Time Bar"))

	today := time.Date(2026, 7, 27, 0, 0, 0, 0, time.Local)
	// 触发日 + 时限天数 → 到期日；剩余天数相对 today 计算
	makeTimeBarFile(t, dir, proj, [][]string{
		// 已逾期：7/10 + 10 = 7/20，早于 today
		{"1", "GC 20.1", "T1", "业主口头指示增加防腐等级", "2026-07-10", "10", "", "", "", "", "进行中", "张三", ""},
		// 红灯：7/25 + 7 = 8/1，剩 5 天
		{"2", "GC 19.2", "T3", "港口罢工", "2026-07-25", "7", "", "", "", "", "进行中", "李四", ""},
		// 黄灯：7/20 + 21 = 8/10，剩 14 天
		{"3", "GC 20.1", "T2", "CN 已发，待提交 PCO", "2026-07-20", "21", "", "", "", "", "进行中", "王五", ""},
		// 绿灯：7/26 + 28 = 8/23，剩 27 天
		{"4", "GC 20.2", "T4", "索赔详细报告", "2026-07-26", "28", "", "", "", "", "未启动", "赵六", ""},
		// 已完成：即便早已过期也不再倒计时
		{"5", "GC 20.1", "T1", "已完成的通知", "2026-06-01", "10", "", "", "", "2026-06-08", "已完成", "张三", ""},
		// 触发日期无法解析 → 告警，不中断
		{"6", "GC 9.9", "T9", "日期写错", "去年七月", "10", "", "", "", "", "", "", ""},
	})

	res := ScanTimeBars(root, []ProjectBrief{{ShortName: proj}}, today)

	if len(res.Items) != 5 {
		t.Fatalf("应解析出 5 条有效记录（第 6 行日期无效被跳过），实际 %d", len(res.Items))
	}
	if len(res.Warnings) != 1 {
		t.Errorf("应有 1 条解析告警，实际 %d：%v", len(res.Warnings), res.Warnings)
	}

	by := map[string]TimeBarItem{}
	for _, it := range res.Items {
		by[it.SeqNo] = it
	}

	// 到期日按日历日计算，不跳过周末
	if got := by["1"].DueAt; got != "2026-07-20" {
		t.Errorf("1 号到期日应为 2026-07-20，实际 %s", got)
	}
	if by["1"].Remaining != -7 || by["1"].Light != TBRed {
		t.Errorf("1 号应已逾期 7 天且为红灯，实际 剩余%d 灯%s", by["1"].Remaining, by["1"].Light)
	}
	if by["2"].Remaining != 5 || by["2"].Light != TBRed {
		t.Errorf("2 号剩 5 天应为红灯，实际 剩余%d 灯%s", by["2"].Remaining, by["2"].Light)
	}
	if by["3"].Remaining != 14 || by["3"].Light != TBAmber {
		t.Errorf("3 号剩 14 天应为黄灯，实际 剩余%d 灯%s", by["3"].Remaining, by["3"].Light)
	}
	if by["4"].Light != TBGreen {
		t.Errorf("4 号剩 27 天应为绿灯，实际 %s", by["4"].Light)
	}
	// 已完成的不再按剩余天数判灯
	if by["5"].Light != TBDone {
		t.Errorf("5 号已完成应为完成态，实际 %s", by["5"].Light)
	}
	// 代码中文名要能带出来，界面才不用再维护一份映射
	if by["1"].CodeName == "" {
		t.Error("时效代码中文名未填充")
	}

	// 排序：最紧急在前，已完成沉底
	if res.Items[0].SeqNo != "1" {
		t.Errorf("逾期最久的应排首位，实际首位是 %s 号", res.Items[0].SeqNo)
	}
	if res.Items[len(res.Items)-1].Light != TBDone {
		t.Error("已完成的应排在最后")
	}

	s := res.Summary
	if s.Total != 5 || s.Overdue != 1 || s.Red != 2 || s.Amber != 1 || s.Green != 1 || s.Done != 1 {
		t.Errorf("汇总不对：%+v", s)
	}
}

// 表头行号变化（项目部在上方插了行）不应导致解析失败——按表头文字定位
func TestTimeBarHeaderRowShift(t *testing.T) {
	root := t.TempDir()
	proj := "测试项目"
	dir := filepath.Join(root, proj, filepath.FromSlash("05.Schedule/Time Bar"))

	f := excelize.NewFile()
	sheet := "时效日历"
	f.SetSheetName("Sheet1", sheet)
	hdr := []string{"序号", "合同条款号", "时效代码", "触发事件描述", "触发日期", "时限天数",
		"到期日", "剩余天数", "预警灯", "实际完成日", "状态", "责任人", "备注"}
	// 表头放到第 9 行（比模板多插了 4 行）
	for i, h := range hdr {
		c, _ := excelize.CoordinatesToCellName(i+1, 9)
		f.SetCellValue(sheet, c, h)
	}
	vals := []string{"1", "GC 1.1", "T1", "事件", "2026-07-20", "10"}
	for i, v := range vals {
		c, _ := excelize.CoordinatesToCellName(i+1, 10)
		f.SetCellValue(sheet, c, v)
	}
	os.MkdirAll(dir, 0755)
	if err := f.SaveAs(filepath.Join(dir, proj+"-TBC-202607.xlsx")); err != nil {
		t.Fatal(err)
	}

	res := ScanTimeBars(root, []ProjectBrief{{ShortName: proj}},
		time.Date(2026, 7, 27, 0, 0, 0, 0, time.Local))
	if len(res.Items) != 1 {
		t.Fatalf("表头下移后仍应能解析，实际 %d 条，告警：%v", len(res.Items), res.Warnings)
	}
	if res.Items[0].DueAt != "2026-07-30" {
		t.Errorf("到期日应为 2026-07-30，实际 %s", res.Items[0].DueAt)
	}
}
