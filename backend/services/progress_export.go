package services

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

// ExportCompliance 将年度报送合规汇总导出为 xlsx（需求 V1.1 4.5.2）。
func ExportCompliance(rep *ComplianceReport) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	sheet := "报送合规汇总"
	f.SetSheetName("Sheet1", sheet)

	// 标题与统计口径说明——导出件会被单独传阅，必须自带口径，
	// 否则收件人不知道这份数据涵盖了哪几个月。
	f.SetCellValue(sheet, "A1", rep.Year+" 年度项目报送合规汇总")
	scope := "尚无核查快照"
	if len(rep.Periods) > 0 {
		scope = fmt.Sprintf("统计周期：%s 至 %s（共 %d 期已核查）；仅统计必交项",
			rep.Periods[0], rep.Periods[len(rep.Periods)-1], len(rep.Periods))
	}
	f.SetCellValue(sheet, "A2", scope)

	headers := []string{"项目", "应交（必交项）", "按时", "迟交", "缺交", "按时率"}
	for i, h := range headers {
		f.SetCellValue(sheet, cellName(i+1, 4), h)
	}

	row := 5
	for _, r := range rep.Rows {
		vals := []interface{}{r.Project, r.Total, r.Submitted, r.Late, r.Missing,
			fmt.Sprintf("%.1f%%", r.Compliance)}
		for i, v := range vals {
			f.SetCellValue(sheet, cellName(i+1, row), v)
		}
		row++
	}

	if len(rep.Rows) > 0 {
		o := rep.Overall
		vals := []interface{}{"合计", o.Total, o.Submitted, o.Late, o.Missing,
			fmt.Sprintf("%.1f%%", o.Compliance)}
		for i, v := range vals {
			f.SetCellValue(sheet, cellName(i+1, row), v)
		}
	}

	f.SetColWidth(sheet, "A", "A", 26)
	f.SetColWidth(sheet, "B", "F", 14)

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
