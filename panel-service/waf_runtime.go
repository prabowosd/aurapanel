package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

type wafInspectionInput struct {
	Method    string `json:"method"`
	Path      string `json:"path"`
	Query     string `json:"query"`
	Body      string `json:"body"`
	UserAgent string `json:"user_agent"`
	IP        string `json:"ip"`
}

type wafInspectionResult struct {
	Allowed bool   `json:"allowed"`
	Score   int    `json:"score"`
	Reason  string `json:"reason"`
}

func detectModSecurityActive() bool {
	if !fileExists("/usr/local/lsws/modules/mod_security.so") || !fileExists(olsHTTPDConfigPath) {
		return false
	}
	raw, err := os.ReadFile(olsHTTPDConfigPath)
	if err != nil {
		return false
	}
	content := string(raw)
	return strings.Contains(content, "module mod_security") &&
		strings.Contains(content, "modsecurity  on") &&
		strings.Contains(content, "ls_enabled              1")
}

func inspectWAFPayload(input wafInspectionInput) wafInspectionResult {
	normalized := strings.ToLower(strings.Join([]string{
		strings.TrimSpace(input.Path),
		strings.TrimSpace(input.Query),
		strings.TrimSpace(input.Body),
		strings.TrimSpace(input.UserAgent),
	}, "\n"))

	signatures := []struct {
		Name     string
		Patterns []string
		Score    int
	}{
		{Name: "XSS payload", Patterns: []string{"<script", "javascript:", "onerror=", "onload=", "%3cscript"}, Score: 95},
		{Name: "SQL injection payload", Patterns: []string{" union select ", "' or 1=1", "\" or 1=1", "sleep(", "benchmark(", "information_schema"}, Score: 98},
		{Name: "LFI payload", Patterns: []string{"../", "..%2f", "/etc/passwd", "boot.ini"}, Score: 90},
		{Name: "RCE payload", Patterns: []string{";wget ", ";curl ", "$(", "`id`", "|bash", "cmd.exe"}, Score: 99},
		{Name: "Scanner signature", Patterns: []string{"sqlmap", "nikto", "acunetix", "nmap"}, Score: 85},
	}

	for _, signature := range signatures {
		for _, pattern := range signature.Patterns {
			if strings.Contains(normalized, pattern) {
				reason := fmt.Sprintf("%s tespit edildi (%s).", signature.Name, pattern)
				if detectModSecurityActive() {
					reason = "ModSecurity/OWASP CRS aktif: " + reason
				} else {
					reason = "ModSecurity pasif ama istek riskli: " + reason
				}
				return wafInspectionResult{
					Allowed: false,
					Score:   signature.Score,
					Reason:  reason,
				}
			}
		}
	}

	reason := "Istek, AuraPanel WAF heuristiklerine gore temiz."
	if detectModSecurityActive() {
		reason = "ModSecurity/OWASP CRS aktif ve istek temel imzalara takilmadi."
	}
	return wafInspectionResult{
		Allowed: true,
		Score:   5,
		Reason:  reason,
	}
}

func (s *service) handleSecurityWAF(w http.ResponseWriter, r *http.Request) {
	var payload wafInspectionInput
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid WAF payload.")
		return
	}
	result := inspectWAFPayload(payload)
	writeJSON(w, http.StatusOK, apiResponse{
		Status:  "success",
		Allowed: result.Allowed,
		Score:   result.Score,
		Reason:  result.Reason,
		Data:    result,
	})
}
