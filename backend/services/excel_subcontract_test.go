package services

import (
	"opims/data"
	"testing"
)

func TestParseSubcontractExcel(t *testing.T) {
	path := `D:\WPS云盘\186113660\WPS云盘\OneDrive - 中国化学工程股份有限公司\海外运营中心\06.Received File\02.在建项目分包商及分包合同管理台账\在建项目分包商及分包合同管理台账3.24（五公司).xlsx`
	records, err := ParseSubcontractExcel(path)
	if err != nil {
		t.Fatalf("ParseSubcontractExcel failed: %v", err)
	}

	t.Logf("Total records: %d", len(records))
	if len(records) < 700 {
		t.Errorf("Expected >= 900 records, got %d", len(records))
	}

	// Validate first record
	if len(records) > 0 {
		r := records[0]
		t.Logf("First: sub_name=%s tier=%s raw=%s std=%s cat=%s",
			r.SubName, r.SubTier, r.SubProfessionRaw, r.StandardizedProfession, r.ProfessionCategory)

		if r.SubName == "" {
			t.Error("First record has empty SubName")
		}
		if r.StandardizedProfession == "" || r.StandardizedProfession == "其他" && r.SubProfessionRaw != "" {
			// Check if raw value exists in keywords
			if kw, ok := data.ProfessionKeywords[r.SubProfessionRaw]; ok {
				if kw == "其他" {
					t.Logf("Raw profession '%s' explicitly maps to 其他", r.SubProfessionRaw)
				}
			} else {
				t.Logf("Raw profession '%s' not in keyword map, defaults to 其他", r.SubProfessionRaw)
			}
		}
	}

	// Distribution
	profCount := map[string]int{}
	catCount := map[string]int{}
	tierCount := map[string]int{}
	for _, r := range records {
		profCount[r.StandardizedProfession]++
		catCount[r.ProfessionCategory]++
		tierCount[r.SubTier]++
	}
	t.Log("Profession distribution:")
	for k, v := range profCount {
		t.Logf("  %s: %d", k, v)
	}
	t.Log("Category distribution:")
	for k, v := range catCount {
		t.Logf("  %s: %d", k, v)
	}
	t.Log("Tier distribution:")
	for k, v := range tierCount {
		t.Logf("  %s: %d", k, v)
	}
}
