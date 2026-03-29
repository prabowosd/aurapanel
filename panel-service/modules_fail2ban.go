package main

import (
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strings"
)

func (s *service) handleFail2banList(w http.ResponseWriter) {
	cmd := exec.Command("fail2ban-client", "status")
	output, err := cmd.CombinedOutput()
	if err != nil {
		writeJSON(w, http.StatusOK, apiResponse{Status: "success", Data: map[string]interface{}{"status": "not installed or inactive", "raw": string(output)}})
		return
	}
	writeJSON(w, http.StatusOK, apiResponse{Status: "success", Data: map[string]interface{}{"status": "active", "raw": string(output)}})
}

func (s *service) handleFail2banUnban(w http.ResponseWriter, r *http.Request) {
	ip := strings.TrimSpace(r.URL.Query().Get("ip"))
	if ip == "" {
		writeError(w, http.StatusBadRequest, "IP is required.")
		return
	}
	if net.ParseIP(ip) == nil {
		writeError(w, http.StatusBadRequest, "Invalid IP address.")
		return
	}

	// Preferred command for most Fail2Ban versions/jails.
	if output, err := exec.Command("fail2ban-client", "set", "sshd", "unbanip", ip).CombinedOutput(); err == nil {
		writeJSON(w, http.StatusOK, apiResponse{Status: "success", Message: fmt.Sprintf("IP %s unbanned successfully from sshd.", ip)})
		return
	} else if strings.TrimSpace(string(output)) != "" {
		// keep trying below; this output is only used if every fallback fails.
	}

	// Fallback for versions supporting global unban command.
	if output, err := exec.Command("fail2ban-client", "unban", ip).CombinedOutput(); err == nil {
		writeJSON(w, http.StatusOK, apiResponse{Status: "success", Message: fmt.Sprintf("IP %s unbanned successfully.", ip)})
		return
	} else if strings.TrimSpace(string(output)) != "" {
		// keep trying below; this output is only used if every fallback fails.
	}

	// Final fallback: enumerate jails and try unban in each one.
	statusOutput, statusErr := exec.Command("fail2ban-client", "status").CombinedOutput()
	if statusErr != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to inspect fail2ban status: %s", strings.TrimSpace(string(statusOutput))))
		return
	}
	jails := parseFail2BanJails(string(statusOutput))
	if len(jails) == 0 {
		writeError(w, http.StatusInternalServerError, "No fail2ban jail found for unban operation.")
		return
	}

	unbannedFrom := make([]string, 0, len(jails))
	lastErrors := make([]string, 0, len(jails))
	for _, jail := range jails {
		output, err := exec.Command("fail2ban-client", "set", jail, "unbanip", ip).CombinedOutput()
		if err == nil {
			unbannedFrom = append(unbannedFrom, jail)
			continue
		}
		msg := strings.TrimSpace(string(output))
		if msg == "" {
			msg = err.Error()
		}
		lastErrors = append(lastErrors, fmt.Sprintf("%s: %s", jail, msg))
	}

	if len(unbannedFrom) > 0 {
		writeJSON(w, http.StatusOK, apiResponse{
			Status:  "success",
			Message: fmt.Sprintf("IP %s unbanned from jail(s): %s", ip, strings.Join(unbannedFrom, ", ")),
			Data: map[string]interface{}{
				"jails":      unbannedFrom,
				"partialErr": lastErrors,
			},
		})
		return
	}

	writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to unban IP in detected jails: %s", strings.Join(lastErrors, " | ")))
}

func parseFail2BanJails(statusRaw string) []string {
	lines := strings.Split(statusRaw, "\n")
	for _, line := range lines {
		lower := strings.ToLower(line)
		if !strings.Contains(lower, "jail list:") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		rawJails := strings.Split(parts[1], ",")
		jails := make([]string, 0, len(rawJails))
		for _, jail := range rawJails {
			name := strings.TrimSpace(jail)
			if name != "" {
				jails = append(jails, name)
			}
		}
		return jails
	}
	return nil
}
