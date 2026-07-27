package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"opims/services"
	"strings"
)

// policyDirHint 是政策库目录缺失时给用户的统一提示，
// 指明应去「项目文件 → 设置根目录」检查，而不是让用户找不着北。
const policyDirHint = "未找到政策库目录，请确认根目录设置正确，且根目录下存在 08.Policy/出入境政策 文件夹"

// PolicyCountries GET /api/policy/countries —— 各国别档案摘要列表。
func (h *Handler) PolicyCountries(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	list, err := services.ListPolicies(h.root.EntryExitPolicyDir())
	if err != nil {
		if errors.Is(err, services.ErrPolicyDirMissing) {
			writeJSONError(w, http.StatusNotFound, policyDirHint, h.root.EntryExitPolicyDir())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, err.Error(), "")
		return
	}
	if list == nil {
		list = []services.PolicySummary{}
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"countries": list,
		"dir":       h.root.EntryExitPolicyDir(),
	})
}

// PolicyCountryByName GET /api/policy/country/{name} —— 单个国别完整档案。
// name 可以是归一化后的标准名（印尼），也可以是原始名（印度尼西亚）。
func (h *Handler) PolicyCountryByName(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	name := strings.TrimPrefix(r.URL.Path, "/api/policy/country/")
	if decoded, err := url.PathUnescape(name); err == nil {
		name = decoded
	}
	if name == "" {
		writeJSONError(w, http.StatusBadRequest, "缺少国别参数", "")
		return
	}

	detail, err := services.GetPolicy(h.root.EntryExitPolicyDir(), name)
	if err != nil {
		if errors.Is(err, services.ErrPolicyDirMissing) {
			writeJSONError(w, http.StatusNotFound, policyDirHint, h.root.EntryExitPolicyDir())
			return
		}
		writeJSONError(w, http.StatusNotFound, err.Error(), "")
		return
	}
	json.NewEncoder(w).Encode(detail)
}

// PolicyInternal GET /api/policy/internal —— 公司内部管理制度摘要。
// 该文档说明公司内部审批流程，与目标国外部政策叠加才是真实排期，
// 故单独提供入口。
func (h *Handler) PolicyInternal(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	html, err := services.GetAuxDoc(h.root.EntryExitPolicyDir(), "03")
	if err != nil {
		if errors.Is(err, services.ErrPolicyDirMissing) {
			writeJSONError(w, http.StatusNotFound, policyDirHint, h.root.EntryExitPolicyDir())
			return
		}
		writeJSONError(w, http.StatusNotFound, err.Error(), "")
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"body_html": html})
}

// writeJSONError 以 JSON 返回错误，便于前端区分「目录没配好」与其它失败，
// 并把期望路径回传，方便用户对照检查。
func writeJSONError(w http.ResponseWriter, code int, msg, dir string) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg, "expected_dir": dir})
}
