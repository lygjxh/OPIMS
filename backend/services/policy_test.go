package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 构造一个与真实政策库同构的临时目录：辅助文件 + 国别档案 + .obsidian 隐藏目录。
func makeVault(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// .obsidian 配置目录（必须被忽略）
	if err := os.MkdirAll(filepath.Join(dir, ".obsidian"), 0755); err != nil {
		t.Fatal(err)
	}

	files := map[string]string{
		"00 出入境政策一览.base": "irrelevant",
		"01 模板.md":         "---\n国家: \"\"\n---\n# 模板\n",
		"02 更新与检索规则.md":    "# 规则\n",
		"03 公司内部管理制度摘要.md": "# 内部制度\n\n审批需 5 个工作日。\n",
		// 国别名与 OPIMS 标准名不一致，且风险等级带括号说明
		"印度尼西亚.md": "---\n" +
			"国家: \"印度尼西亚\"\n" +
			"更新日期: \"2026-07-26\"\n" +
			"人员类型: [\"管理人员\", \"技术人员\"]\n" +
			"签证类型: \"VITAS（限期居留签证）→ KITAS\"\n" +
			"预计办理周期: \"约4-8周\"\n" +
			"风险等级: \"高（重要提示：普通劳务无合法路径）\"\n" +
			"信息来源: \"https://evisa.imigrasi.go.id\"\n" +
			"---\n\n# 印度尼西亚 出入境政策\n\n" +
			"## 三、所需材料清单\n- [ ] 护照\n- [ ] 邀请函\n\n" +
			"## 六、费用明细\n| 项目 | 金额 |\n|---|---|\n| 签证费 | 1200 |\n",
		// 两侧写法一致、风险等级为纯字符、日期很旧（应判为过期）
		"越南.md": "---\n" +
			"国家: \"越南\"\n" +
			"更新日期: \"2020-01-01\"\n" +
			"签证类型: \"工作签证\"\n" +
			"风险等级: \"中\"\n" +
			"---\n\n# 越南 出入境政策\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestListPolicies(t *testing.T) {
	list, err := ListPolicies(makeVault(t))
	if err != nil {
		t.Fatalf("ListPolicies: %v", err)
	}

	// 只应识别出 2 个国别档案：辅助文件(00-03)、.base、.obsidian 都要排除
	if len(list) != 2 {
		names := []string{}
		for _, s := range list {
			names = append(names, s.FileName)
		}
		t.Fatalf("期望 2 个国别档案，实际 %d 个：%v", len(list), names)
	}

	// 高风险应排在前面
	if list[0].RiskLevel != "高" {
		t.Errorf("期望高风险排首位，实际 %q(%s)", list[0].RiskLevel, list[0].Country)
	}

	idn := list[0]
	// 国别归一化：印度尼西亚 → 印尼（否则无法与 OPIMS 项目数据关联）
	if idn.Country != "印尼" {
		t.Errorf("国别归一化失败：期望「印尼」，实际 %q", idn.Country)
	}
	if idn.CountryRaw != "印度尼西亚" {
		t.Errorf("原始国名应保留：实际 %q", idn.CountryRaw)
	}
	// 风险等级从「高（重要提示：…）」中提取，完整文本保留在 RiskNote
	if idn.RiskLevel != "高" {
		t.Errorf("风险等级提取失败：实际 %q", idn.RiskLevel)
	}
	if !strings.Contains(idn.RiskNote, "重要提示") {
		t.Errorf("风险说明原文丢失：%q", idn.RiskNote)
	}
	if idn.IsStale {
		t.Errorf("2026-07-26 更新的档案不应判为过期（当前 %d 天）", idn.StaleDays)
	}

	// 越南：日期久远应判为过期
	var vn *PolicySummary
	for i := range list {
		if list[i].Country == "越南" {
			vn = &list[i]
		}
	}
	if vn == nil {
		t.Fatal("未找到越南档案")
	}
	if !vn.IsStale {
		t.Errorf("2020-01-01 的档案应判为过期，StaleDays=%d", vn.StaleDays)
	}
}

func TestGetPolicy(t *testing.T) {
	dir := makeVault(t)

	// 用归一化名查找
	d, err := GetPolicy(dir, "印尼")
	if err != nil {
		t.Fatalf("用标准名「印尼」查找失败: %v", err)
	}
	if d.Meta.VisaType == "" {
		t.Error("frontmatter 未解析出签证类型")
	}
	if len(d.Meta.PersonTypes) != 2 {
		t.Errorf("人员类型数组解析失败：%v", d.Meta.PersonTypes)
	}
	// GFM：表格与任务列表都要渲染出来
	if !strings.Contains(d.BodyHTML, "<table") {
		t.Error("费用明细表格未渲染为 <table>")
	}
	if !strings.Contains(d.BodyHTML, "checkbox") {
		t.Error("材料清单未渲染为任务列表")
	}

	// 用原始名同样能查到
	if _, err := GetPolicy(dir, "印度尼西亚"); err != nil {
		t.Errorf("用原始名「印度尼西亚」查找失败: %v", err)
	}

	// 不存在的国别应报错而不是返回空档案
	if _, err := GetPolicy(dir, "火星"); err == nil {
		t.Error("查询不存在的国别时应返回错误")
	}
}

func TestGetAuxDoc(t *testing.T) {
	html, err := GetAuxDoc(makeVault(t), "03")
	if err != nil {
		t.Fatalf("GetAuxDoc: %v", err)
	}
	if !strings.Contains(html, "内部制度") {
		t.Errorf("内部制度文档内容不对: %q", html)
	}
}

func TestPolicyDirMissing(t *testing.T) {
	_, err := ListPolicies(filepath.Join(t.TempDir(), "不存在的目录"))
	if err != ErrPolicyDirMissing {
		t.Errorf("目录不存在时应返回 ErrPolicyDirMissing，实际 %v", err)
	}
}
