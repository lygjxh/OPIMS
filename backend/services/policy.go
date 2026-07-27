package services

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"opims/data"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"gopkg.in/yaml.v3"
)

// StaleAfterDays 超过该天数未更新即提示核查。出入境政策变化快，过期信息有实际风险。
const StaleAfterDays = 90

// PolicyMeta 是国别档案 YAML frontmatter 的结构化表示。
// 字段名与 Obsidian 笔记中的中文键一一对应。
type PolicyMeta struct {
	Country       string   `yaml:"国家"          json:"country_raw"`
	UpdatedAt     string   `yaml:"更新日期"       json:"updated_at"`
	PersonTypes   []string `yaml:"人员类型"       json:"person_types"`
	VisaType      string   `yaml:"签证类型"       json:"visa_type"`
	Authority     string   `yaml:"办理机构"       json:"authority"`
	Method        string   `yaml:"办理方式"       json:"method"`
	Duration      string   `yaml:"办理时限"       json:"duration"`
	TotalCycle    string   `yaml:"预计办理周期"    json:"total_cycle"`
	StayValidity  string   `yaml:"停留/有效期限"   json:"stay_validity"`
	NeedHealth    string   `yaml:"是否需健康证明"  json:"need_health"`
	NeedNoCrime   string   `yaml:"是否需无犯罪记录证明" json:"need_no_crime"`
	SeparatePermit string  `yaml:"工作许可是否单独办理" json:"separate_permit"`
	FamilyPolicy  string   `yaml:"家属随行政策"    json:"family_policy"`
	Cost          string   `yaml:"签证大致费用"    json:"cost"`
	RiskRaw       string   `yaml:"风险等级"       json:"risk_raw"`
	Source        string   `yaml:"信息来源"       json:"source"`
}

// PolicySummary 是列表页所需的国别档案摘要。
type PolicySummary struct {
	Country     string `json:"country"`      // 归一化后的 OPIMS 标准国别名
	CountryRaw  string `json:"country_raw"`  // 政策库中的原始写法
	FileName    string `json:"file_name"`
	UpdatedAt   string `json:"updated_at"`
	RiskLevel   string `json:"risk_level"`   // 高/中/低，从 RiskRaw 开头提取
	RiskNote    string `json:"risk_note"`    // 风险等级字段的完整原文
	VisaType    string `json:"visa_type"`
	TotalCycle  string `json:"total_cycle"`
	IsStale     bool   `json:"is_stale"`     // 更新日期超过 StaleAfterDays
	StaleDays   int    `json:"stale_days"`   // 距今天数，无法解析日期时为 -1
}

// PolicyDetail 是详情页所需的完整内容。
type PolicyDetail struct {
	PolicySummary
	Meta     PolicyMeta `json:"meta"`
	BodyHTML string     `json:"body_html"`
}

// ErrPolicyDirMissing 表示政策库目录不存在，供 handler 区分「目录没配好」与「确实没档案」。
var ErrPolicyDirMissing = fmt.Errorf("policy directory not found")

// isAuxFile 判断是否为辅助文件（00 一览 / 01 模板 / 02 规则 / 03 内部制度），
// 这些不是国别档案。约定：以数字开头的文件均为辅助文件。
func isAuxFile(name string) bool {
	return len(name) > 0 && name[0] >= '0' && name[0] <= '9'
}

// ListPolicies 扫描出入境政策目录，返回各国别档案摘要（按风险等级、国名排序）。
func ListPolicies(dir string) ([]PolicySummary, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, ErrPolicyDirMissing
	}

	var list []PolicySummary
	for _, e := range entries {
		// 跳过子目录与隐藏项（08.Policy 下有 .obsidian 配置目录）
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".md") || isAuxFile(e.Name()) {
			continue
		}
		meta, _, err := parsePolicyFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue // 单个文件损坏不应导致整个列表失败
		}
		list = append(list, summarize(meta, e.Name()))
	}

	// 高风险排前面，同级按国名排序，便于一眼看到需要重点关注的国别
	rank := map[string]int{"高": 0, "中": 1, "低": 2}
	sort.Slice(list, func(i, j int) bool {
		ri, oki := rank[list[i].RiskLevel]
		rj, okj := rank[list[j].RiskLevel]
		if !oki {
			ri = 3
		}
		if !okj {
			rj = 3
		}
		if ri != rj {
			return ri < rj
		}
		return list[i].Country < list[j].Country
	})
	return list, nil
}

// GetPolicy 按归一化国别名查找并返回完整档案。
func GetPolicy(dir, country string) (*PolicyDetail, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, ErrPolicyDirMissing
	}
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") ||
			!strings.HasSuffix(e.Name(), ".md") || isAuxFile(e.Name()) {
			continue
		}
		path := filepath.Join(dir, e.Name())
		meta, body, err := parsePolicyFile(path)
		if err != nil {
			continue
		}
		s := summarize(meta, e.Name())
		if s.Country != country && s.CountryRaw != country {
			continue
		}
		html, err := renderMarkdown(body)
		if err != nil {
			return nil, err
		}
		return &PolicyDetail{PolicySummary: s, Meta: meta, BodyHTML: html}, nil
	}
	return nil, fmt.Errorf("未找到国别档案：%s", country)
}

// GetAuxDoc 读取指定辅助文档（如「03 公司内部管理制度摘要.md」）并渲染为 HTML。
func GetAuxDoc(dir, prefix string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", ErrPolicyDirMissing
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") ||
			!strings.HasPrefix(e.Name(), prefix) {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return "", err
		}
		_, body := splitFrontmatter(raw)
		return renderMarkdown(body)
	}
	return "", fmt.Errorf("未找到文档：%s", prefix)
}

// parsePolicyFile 读取单个国别档案，返回 frontmatter 与正文。
func parsePolicyFile(path string) (PolicyMeta, []byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return PolicyMeta{}, nil, err
	}
	front, body := splitFrontmatter(raw)

	var meta PolicyMeta
	if len(front) > 0 {
		if err := yaml.Unmarshal(front, &meta); err != nil {
			// frontmatter 格式有误时仍返回正文，不让整个档案不可用
			return meta, body, nil
		}
	}
	return meta, body, nil
}

// splitFrontmatter 拆分 YAML frontmatter 与正文。
// 无 frontmatter 时返回 (nil, 全文)。
func splitFrontmatter(raw []byte) (front, body []byte) {
	// 兼容 CRLF 与文件开头的 BOM
	s := bytes.TrimPrefix(raw, []byte("\xef\xbb\xbf"))
	s = bytes.ReplaceAll(s, []byte("\r\n"), []byte("\n"))
	if !bytes.HasPrefix(s, []byte("---\n")) {
		return nil, s
	}
	rest := s[4:]
	end := bytes.Index(rest, []byte("\n---"))
	if end < 0 {
		return nil, s
	}
	front = rest[:end]
	body = rest[end+4:]
	return front, bytes.TrimLeft(body, "\n")
}

// renderMarkdown 将正文渲染为 HTML。启用 GFM 以支持表格与任务列表
// （档案中的费用明细/政策变化记录是表格，材料清单是 - [ ] 任务列表）。
func renderMarkdown(body []byte) (string, error) {
	md := goldmark.New(goldmark.WithExtensions(extension.GFM))
	var buf bytes.Buffer
	if err := md.Convert(body, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// summarize 由 frontmatter 生成列表摘要，处理国别归一化、风险等级提取与时效判断。
func summarize(m PolicyMeta, fileName string) PolicySummary {
	raw := strings.TrimSpace(m.Country)
	if raw == "" {
		// frontmatter 缺「国家」时退回用文件名（去掉 .md）
		raw = strings.TrimSuffix(fileName, ".md")
	}
	level, days, stale := riskLevel(m.RiskRaw), -1, false
	days, stale = staleness(m.UpdatedAt)

	return PolicySummary{
		Country:    data.NormalizeCountry(raw),
		CountryRaw: raw,
		FileName:   fileName,
		UpdatedAt:  strings.TrimSpace(m.UpdatedAt),
		RiskLevel:  level,
		RiskNote:   strings.TrimSpace(m.RiskRaw),
		VisaType:   strings.TrimSpace(m.VisaType),
		TotalCycle: strings.TrimSpace(m.TotalCycle),
		IsStale:    stale,
		StaleDays:  days,
	}
}

// riskLevel 从「风险等级」字段开头提取等级。
// 实际值形如「高（重要提示：……）」，不能用等号精确匹配。
func riskLevel(s string) string {
	t := strings.TrimSpace(s)
	for _, lv := range []string{"高", "中", "低"} {
		if strings.HasPrefix(t, lv) {
			return lv
		}
	}
	return ""
}

// staleness 返回距更新日期的天数与是否过期。日期无法解析时返回 (-1, false)。
func staleness(updated string) (int, bool) {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(updated))
	if err != nil {
		return -1, false
	}
	days := int(time.Since(t).Hours() / 24)
	return days, days > StaleAfterDays
}
