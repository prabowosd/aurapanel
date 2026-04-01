package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type webmailTokenView struct {
	TokenPreview string `json:"token_preview"`
	Address      string `json:"address"`
	Domain       string `json:"domain"`
	ExpiresAt    int64  `json:"expires_at"`
	Expired      bool   `json:"expired"`
}

func normalizeDMARCPolicy(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "none":
		return "none"
	case "reject":
		return "reject"
	default:
		return "quarantine"
	}
}

func normalizeDMARCAlignment(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "s":
		return "s"
	default:
		return "r"
	}
}

func parseBoundedInt(raw string, fallback, minValue, maxValue int) int {
	value := strings.TrimSpace(raw)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	if parsed < minValue {
		return minValue
	}
	if parsed > maxValue {
		return maxValue
	}
	return parsed
}

func emailDomain(address string) string {
	address = strings.TrimSpace(strings.ToLower(address))
	parts := strings.SplitN(address, "@", 2)
	if len(parts) != 2 {
		return ""
	}
	return normalizeDomain(parts[1])
}

func maskToken(token string) string {
	token = strings.TrimSpace(token)
	if len(token) <= 8 {
		return token
	}
	return token[:4] + "..." + token[len(token)-4:]
}

func sanitizeEmailURI(value string, fallbackDomain string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "mailto:postmaster@" + fallbackDomain
	}
	if strings.HasPrefix(strings.ToLower(value), "mailto:") {
		return value
	}
	if strings.Contains(value, "@") {
		return "mailto:" + value
	}
	return "mailto:" + value + "@" + fallbackDomain
}

func buildSPFRecord() string {
	parts := []string{"v=spf1", "mx", "a"}
	publicIP := strings.TrimSpace(os.Getenv("AURAPANEL_PUBLIC_IP"))
	if publicIP != "" && net.ParseIP(publicIP) != nil {
		if strings.Contains(publicIP, ":") {
			parts = append(parts, "ip6:"+publicIP)
		} else {
			parts = append(parts, "ip4:"+publicIP)
		}
	}
	includeHost := strings.TrimSpace(os.Getenv("AURAPANEL_MAIL_SPF_INCLUDE"))
	if includeHost != "" {
		parts = append(parts, "include:"+includeHost)
	}
	parts = append(parts, "~all")
	return strings.Join(parts, " ")
}

func buildDMARCRecord(policy, rua, ruf, adkim, aspf string, pct int) string {
	parts := []string{
		"v=DMARC1",
		"p=" + normalizeDMARCPolicy(policy),
		"adkim=" + normalizeDMARCAlignment(adkim),
		"aspf=" + normalizeDMARCAlignment(aspf),
		"pct=" + strconv.Itoa(pct),
	}
	if strings.TrimSpace(rua) != "" {
		parts = append(parts, "rua="+rua)
	}
	if strings.TrimSpace(ruf) != "" {
		parts = append(parts, "ruf="+ruf)
	}
	return strings.Join(parts, "; ")
}

func (s *service) ensureMailAuthRecordLocked(domain string) MailAuthRecord {
	domain = normalizeDomain(domain)
	now := time.Now().UTC().UnixMilli()
	record, ok := s.modules.MailAuthRecords[domain]
	if ok {
		record.UpdatedAt = now
		s.modules.MailAuthRecords[domain] = record
		return record
	}
	record = MailAuthRecord{
		Domain:      domain,
		SPFHost:     "@",
		SPFValue:    buildSPFRecord(),
		DMARCHost:   "_dmarc." + domain,
		DMARCValue:  buildDMARCRecord("quarantine", "mailto:postmaster@"+domain, "", "r", "r", 100),
		Policy:      "quarantine",
		RUA:         "mailto:postmaster@" + domain,
		GeneratedAt: now,
		UpdatedAt:   now,
	}
	s.modules.MailAuthRecords[domain] = record
	return record
}

func (s *service) handleMailAuthGet(w http.ResponseWriter, r *http.Request) {
	domain := normalizeDomain(r.URL.Query().Get("domain"))
	if domain == "" {
		writeError(w, http.StatusBadRequest, "Domain is required.")
		return
	}

	s.mu.Lock()
	record := s.ensureMailAuthRecordLocked(domain)
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, apiResponse{Status: "success", Data: record})
}

func (s *service) handleMailAuthBootstrap(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Domain string `json:"domain"`
		Policy string `json:"policy"`
		RUA    string `json:"rua"`
		RUF    string `json:"ruf"`
		PCT    string `json:"pct"`
		ADKIM  string `json:"adkim"`
		ASPF   string `json:"aspf"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid mail auth bootstrap payload.")
		return
	}

	domain := normalizeDomain(payload.Domain)
	if domain == "" {
		writeError(w, http.StatusBadRequest, "Domain is required.")
		return
	}

	policy := normalizeDMARCPolicy(payload.Policy)
	rua := sanitizeEmailURI(payload.RUA, domain)
	ruf := ""
	if strings.TrimSpace(payload.RUF) != "" {
		ruf = sanitizeEmailURI(payload.RUF, domain)
	}
	pct := parseBoundedInt(payload.PCT, 100, 1, 100)
	adkim := normalizeDMARCAlignment(payload.ADKIM)
	aspf := normalizeDMARCAlignment(payload.ASPF)
	now := time.Now().UTC().UnixMilli()

	record := MailAuthRecord{
		Domain:      domain,
		SPFHost:     "@",
		SPFValue:    buildSPFRecord(),
		DMARCHost:   "_dmarc." + domain,
		DMARCValue:  buildDMARCRecord(policy, rua, ruf, adkim, aspf, pct),
		Policy:      policy,
		RUA:         rua,
		RUF:         ruf,
		GeneratedAt: now,
		UpdatedAt:   now,
	}

	s.mu.Lock()
	if existing, ok := s.modules.MailAuthRecords[domain]; ok && existing.GeneratedAt > 0 {
		record.GeneratedAt = existing.GeneratedAt
	}
	s.modules.MailAuthRecords[domain] = record
	s.appendActivityLocked("system", "mail_auth_bootstrap", fmt.Sprintf("SPF/DMARC bootstrap updated for %s.", domain), "")
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, apiResponse{
		Status:  "success",
		Message: "SPF/DMARC bootstrap completed.",
		Data:    record,
	})
}

func (s *service) handleMailDeliverability(w http.ResponseWriter, r *http.Request) {
	domain := normalizeDomain(r.URL.Query().Get("domain"))
	if domain == "" {
		writeError(w, http.StatusBadRequest, "Domain is required.")
		return
	}

	now := time.Now().UTC()
	s.mu.RLock()
	record, hasAuth := s.modules.MailAuthRecords[domain]
	dkim, hasDKIM := s.modules.MailDKIM[domain]

	mailboxes := 0
	forwards := 0
	routingRules := 0
	catchAllEnabled := false
	webmailTokens := 0

	for _, mailbox := range s.modules.Mailboxes {
		if normalizeDomain(mailbox.Domain) == domain {
			mailboxes++
		}
	}
	for _, forward := range s.modules.MailForwards {
		if normalizeDomain(forward.Domain) == domain {
			forwards++
		}
	}
	for _, rule := range s.modules.MailRouting {
		if normalizeDomain(rule.Domain) == domain {
			routingRules++
		}
	}
	if catchAll, ok := s.modules.MailCatchAll[domain]; ok {
		catchAllEnabled = catchAll.Enabled
	}
	for _, token := range s.modules.WebmailTokens {
		if token.ExpiresAt.Before(now) {
			continue
		}
		if emailDomain(token.Address) == domain {
			webmailTokens++
		}
	}
	s.mu.RUnlock()

	score := 40
	recommendations := make([]string, 0, 6)
	checks := map[string]bool{
		"dkim":          hasDKIM && strings.TrimSpace(dkim.PublicKey) != "",
		"spf_dmarc":     hasAuth && strings.TrimSpace(record.SPFValue) != "" && strings.TrimSpace(record.DMARCValue) != "",
		"mailbox_ready": mailboxes > 0,
		"catch_all":     !catchAllEnabled,
	}

	if checks["dkim"] {
		score += 20
	} else {
		recommendations = append(recommendations, "Enable DKIM for this domain.")
	}

	if checks["spf_dmarc"] {
		score += 25
	} else {
		recommendations = append(recommendations, "Bootstrap SPF and DMARC records.")
	}

	if checks["mailbox_ready"] {
		score += 10
	} else {
		score -= 10
		recommendations = append(recommendations, "Create at least one mailbox for sender reputation.")
	}

	if checks["catch_all"] {
		score += 5
	} else {
		score -= 15
		recommendations = append(recommendations, "Disable catch-all to reduce spam noise.")
	}

	if forwards > 0 || routingRules > 0 {
		score += 5
	}
	if webmailTokens > 10 {
		score -= 5
		recommendations = append(recommendations, "High active webmail SSO token count, consider revocation/cleanup.")
	}

	score = clampScore(score)
	risk := "low"
	if score < 80 {
		risk = "medium"
	}
	if score < 60 {
		risk = "high"
	}

	writeJSON(w, http.StatusOK, apiResponse{
		Status: "success",
		Data: map[string]interface{}{
			"domain": domain,
			"score":  score,
			"risk":   risk,
			"checks": checks,
			"observability": map[string]interface{}{
				"mailboxes":             mailboxes,
				"forwards":              forwards,
				"routing_rules":         routingRules,
				"catch_all_enabled":     catchAllEnabled,
				"active_webmail_tokens": webmailTokens,
				"dkim_selector":         dkim.Selector,
				"dmarc_policy":          record.Policy,
			},
			"recommendations": recommendations,
		},
	})
}

func (s *service) webmailTokenStatsLocked(domain string) (int, int, int) {
	now := time.Now().UTC()
	active := 0
	expired := 0
	total := 0
	for _, token := range s.modules.WebmailTokens {
		tokenDomain := emailDomain(token.Address)
		if domain != "" && tokenDomain != domain {
			continue
		}
		total++
		if token.ExpiresAt.Before(now) {
			expired++
		} else {
			active++
		}
	}
	return total, active, expired
}

func (s *service) handleMailWebmailOpsStatus(w http.ResponseWriter, r *http.Request) {
	domain := normalizeDomain(r.URL.Query().Get("domain"))
	s.mu.RLock()
	total, active, expired := s.webmailTokenStatsLocked(domain)
	baseURL := s.resolveWebmailBaseURL(r)
	s.mu.RUnlock()

	writeJSON(w, http.StatusOK, apiResponse{
		Status: "success",
		Data: map[string]interface{}{
			"domain":         domain,
			"base_url":       baseURL,
			"tokens_total":   total,
			"tokens_active":  active,
			"tokens_expired": expired,
		},
	})
}

func (s *service) handleMailWebmailOpsTokens(w http.ResponseWriter, r *http.Request) {
	domain := normalizeDomain(r.URL.Query().Get("domain"))
	status := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))
	if status == "" {
		status = "all"
	}
	now := time.Now().UTC()

	s.mu.RLock()
	items := make([]webmailTokenView, 0, len(s.modules.WebmailTokens))
	for token, item := range s.modules.WebmailTokens {
		tokenDomain := emailDomain(item.Address)
		if domain != "" && tokenDomain != domain {
			continue
		}
		expired := item.ExpiresAt.Before(now)
		if status == "active" && expired {
			continue
		}
		if status == "expired" && !expired {
			continue
		}
		items = append(items, webmailTokenView{
			TokenPreview: maskToken(token),
			Address:      item.Address,
			Domain:       tokenDomain,
			ExpiresAt:    item.ExpiresAt.UnixMilli(),
			Expired:      expired,
		})
	}
	s.mu.RUnlock()

	sort.Slice(items, func(i, j int) bool { return items[i].ExpiresAt > items[j].ExpiresAt })
	writeJSON(w, http.StatusOK, apiResponse{Status: "success", Data: items})
}

func (s *service) handleMailWebmailOpsCleanup(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Domain string `json:"domain"`
	}
	_ = decodeJSON(r, &payload)
	domain := normalizeDomain(payload.Domain)
	now := time.Now().UTC()
	s.mu.Lock()
	removed := 0
	for token, item := range s.modules.WebmailTokens {
		if domain != "" && emailDomain(item.Address) != domain {
			continue
		}
		if item.ExpiresAt.Before(now) {
			delete(s.modules.WebmailTokens, token)
			removed++
		}
	}
	s.appendActivityLocked("system", "webmail_token_cleanup", fmt.Sprintf("Webmail cleanup removed %d expired token(s).", removed), "")
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, apiResponse{
		Status:  "success",
		Message: fmt.Sprintf("Expired webmail tokens cleaned up: %d", removed),
		Data: map[string]interface{}{
			"removed": removed,
		},
	})
}

func (s *service) handleMailWebmailOpsRevoke(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Token         string `json:"token"`
		Address       string `json:"address"`
		Domain        string `json:"domain"`
		RevokeExpired bool   `json:"revoke_expired"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid webmail revoke payload.")
		return
	}

	tokenKey := strings.TrimSpace(payload.Token)
	address := strings.TrimSpace(strings.ToLower(payload.Address))
	domain := normalizeDomain(payload.Domain)
	now := time.Now().UTC()

	s.mu.Lock()
	revoked := 0
	if tokenKey != "" {
		if _, ok := s.modules.WebmailTokens[tokenKey]; ok {
			delete(s.modules.WebmailTokens, tokenKey)
			revoked++
		}
	} else {
		for token, item := range s.modules.WebmailTokens {
			if address != "" && strings.TrimSpace(strings.ToLower(item.Address)) != address {
				continue
			}
			if domain != "" && emailDomain(item.Address) != domain {
				continue
			}
			if payload.RevokeExpired && !item.ExpiresAt.Before(now) {
				continue
			}
			delete(s.modules.WebmailTokens, token)
			revoked++
		}
	}
	s.appendActivityLocked("system", "webmail_token_revoke", fmt.Sprintf("Webmail token revoke removed %d token(s).", revoked), "")
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, apiResponse{
		Status:  "success",
		Message: fmt.Sprintf("Webmail tokens revoked: %d", revoked),
		Data: map[string]interface{}{
			"revoked": revoked,
		},
	})
}
