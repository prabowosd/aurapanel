package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

func generateTOTPSecret() string {
	secret := make([]byte, 20)
	if _, err := rand.Read(secret); err != nil {
		return strings.ToUpper(generateSecret(20))
	}
	return strings.TrimRight(base32.StdEncoding.EncodeToString(secret), "=")
}

func normalizeTOTPToken(token string) string {
	token = strings.TrimSpace(token)
	token = strings.ReplaceAll(token, " ", "")
	return token
}

func verifyStoredTOTPSecret(secret, token string, now time.Time) bool {
	secret = strings.TrimSpace(secret)
	token = normalizeTOTPToken(token)
	if secret == "" || token == "" {
		return false
	}
	for offset := -1; offset <= 1; offset++ {
		if computeTOTP(secret, now.Add(time.Duration(offset)*30*time.Second)) == token {
			return true
		}
	}
	return false
}

func computeTOTP(secret string, now time.Time) string {
	normalized := strings.ToUpper(strings.TrimSpace(secret))
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(normalized)
	if err != nil {
		return ""
	}
	counter := uint64(now.UTC().Unix() / 30)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, counter)

	sum := hmac.New(sha1.New, key)
	sum.Write(buf)
	hash := sum.Sum(nil)
	offset := hash[len(hash)-1] & 0x0f
	code := (int(hash[offset])&0x7f)<<24 |
		(int(hash[offset+1])&0xff)<<16 |
		(int(hash[offset+2])&0xff)<<8 |
		(int(hash[offset+3]) & 0xff)
	code %= 1000000
	return fmt.Sprintf("%06d", code)
}

func (s *service) handleTOTPSetup(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		AccountName string `json:"account_name"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid 2FA setup payload.")
		return
	}
	account := strings.TrimSpace(payload.AccountName)
	if account == "" {
		account = "admin"
	}

	secret := generateTOTPSecret()

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state.TwoFASecrets == nil {
		s.state.TwoFASecrets = map[string]string{}
	}
	s.state.TwoFASecrets[strings.ToLower(account)] = secret
	writeJSON(w, http.StatusOK, apiResponse{
		Status: "success",
		Data: map[string]interface{}{
			"secret":    secret,
			"qr_base64": "",
		},
	})
}

func (s *service) handleTOTPVerify(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Token string `json:"token"`
	}
	if err := decodeJSON(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid 2FA verify payload.")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	for account, secret := range s.state.TwoFASecrets {
		if !verifyStoredTOTPSecret(secret, payload.Token, now) {
			continue
		}
		for i := range s.state.Users {
			if strings.EqualFold(s.state.Users[i].Username, account) || strings.EqualFold(s.state.Users[i].Email, account) {
				s.state.Users[i].TwoFAEnabled = true
				s.state.TwoFASecrets[s.state.Users[i].Username] = secret
			}
		}
		writeJSON(w, http.StatusOK, apiResponse{Status: "success", Valid: true})
		return
	}

	writeJSON(w, http.StatusOK, apiResponse{Status: "success", Valid: false})
}

func (s *service) handleImmutableStatus(w http.ResponseWriter) {
	supported := fileExists("/run/ostree-booted") || commandExists("rpm-ostree") || commandExists("bootc")
	mode := "mutable"
	if supported {
		mode = "immutable-capable"
	}
	writeJSON(w, http.StatusOK, apiResponse{
		Status: "success",
		Data: map[string]interface{}{
			"supported": supported,
			"mode":      mode,
		},
	})
}

func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}
