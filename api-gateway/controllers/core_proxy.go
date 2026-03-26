package controllers

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

func coreBaseURL() string {
	base := strings.TrimSpace(os.Getenv("AURAPANEL_CORE_URL"))
	if base == "" {
		return "http://127.0.0.1:8000"
	}
	return strings.TrimRight(base, "/")
}

func gatewayOnlyEnabled() bool {
	normalized := strings.ToLower(strings.TrimSpace(os.Getenv("AURAPANEL_GATEWAY_ONLY")))
	if normalized == "" {
		return true
	}
	return normalized == "1" || normalized == "true" || normalized == "yes" || normalized == "on"
}

func isLoopbackCoreTarget(u *url.URL) bool {
	host := strings.TrimSpace(u.Hostname())
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func NewCoreProxy() (http.Handler, error) {
	target, err := url.Parse(coreBaseURL())
	if err != nil {
		return nil, err
	}
	if gatewayOnlyEnabled() && !isLoopbackCoreTarget(target) {
		return nil, &url.Error{
			Op:  "parse",
			URL: target.String(),
			Err: ErrNonLoopbackCoreTarget,
		}
	}

	proxy := httputil.NewSingleHostReverseProxy(target)
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
		req.Header.Set("X-Forwarded-Host", req.Host)
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "error",
			"message": "core request failed: " + err.Error(),
		})
	}

	return proxy, nil
}

var ErrNonLoopbackCoreTarget = &coreProxyPolicyError{msg: "gateway-only mode requires loopback AURAPANEL_CORE_URL"}

type coreProxyPolicyError struct {
	msg string
}

func (e *coreProxyPolicyError) Error() string {
	return e.msg
}
