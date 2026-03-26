package controllers

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aurapanel/api-gateway/middleware"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type adminCredentials struct {
	email        string
	passwordHash string
	passwordText string
}

type loginAttempt struct {
	Failures    int
	FirstFail   time.Time
	LockedUntil time.Time
}

var (
	loginAttemptsMu sync.Mutex
	loginAttempts   = map[string]loginAttempt{}
)

const (
	maxFailedAttempts = 5
	failureWindow     = 10 * time.Minute
	lockDuration      = 15 * time.Minute
)

func devSimulationEnabled() bool {
	normalized := strings.ToLower(strings.TrimSpace(os.Getenv("AURAPANEL_DEV_SIMULATION")))
	return normalized == "1" || normalized == "true" || normalized == "yes" || normalized == "on"
}

func loadAdminCredentials() (adminCredentials, error) {
	creds := adminCredentials{
		email:        strings.TrimSpace(os.Getenv("AURAPANEL_ADMIN_EMAIL")),
		passwordHash: strings.TrimSpace(os.Getenv("AURAPANEL_ADMIN_PASSWORD_BCRYPT")),
		passwordText: strings.TrimSpace(os.Getenv("AURAPANEL_ADMIN_PASSWORD")),
	}

	if creds.email == "" {
		creds.email = "admin@server.com"
	}

	if creds.passwordHash == "" && creds.passwordText == "" {
		passwordFile := "/opt/aurapanel/logs/initial_password.txt"
		if raw, err := os.ReadFile(passwordFile); err == nil {
			creds.passwordText = strings.TrimSpace(string(raw))
		}
	}

	if creds.passwordHash == "" && creds.passwordText == "" {
		if devSimulationEnabled() {
			creds.passwordText = "password123"
			return creds, nil
		}
		return creds, errors.New("admin credentials are not configured")
	}

	return creds, nil
}

func verifyPassword(input string, creds adminCredentials) bool {
	if creds.passwordHash != "" {
		return bcrypt.CompareHashAndPassword([]byte(creds.passwordHash), []byte(input)) == nil
	}

	return subtle.ConstantTimeCompare([]byte(input), []byte(creds.passwordText)) == 1
}

type gatewayClaims struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
	jwt.RegisteredClaims
}

func issueToken(user User) (string, error) {
	now := time.Now().UTC()
	claims := gatewayClaims{
		Email: user.Email,
		Name:  user.Name,
		Role:  user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.Email,
			Issuer:    middleware.JwtIssuer(),
			Audience:  jwt.ClaimStrings{middleware.JwtAudience()},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(12 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(middleware.JwtSecret()))
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func clientIP(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func attemptKey(r *http.Request, email string) string {
	return strings.ToLower(strings.TrimSpace(clientIP(r) + "|" + email))
}

func isLoginBlocked(key string) (bool, time.Duration) {
	loginAttemptsMu.Lock()
	defer loginAttemptsMu.Unlock()

	attempt, ok := loginAttempts[key]
	if !ok {
		return false, 0
	}
	if attempt.LockedUntil.After(time.Now()) {
		return true, time.Until(attempt.LockedUntil)
	}
	if !attempt.LockedUntil.IsZero() {
		delete(loginAttempts, key)
	}
	return false, 0
}

func recordLoginFailure(key string) {
	loginAttemptsMu.Lock()
	defer loginAttemptsMu.Unlock()

	now := time.Now()
	attempt := loginAttempts[key]

	if attempt.FirstFail.IsZero() || now.Sub(attempt.FirstFail) > failureWindow {
		attempt = loginAttempt{Failures: 0, FirstFail: now}
	}

	attempt.Failures++
	if attempt.Failures >= maxFailedAttempts {
		attempt.LockedUntil = now.Add(lockDuration)
	}
	loginAttempts[key] = attempt
}

func clearLoginAttempts(key string) {
	loginAttemptsMu.Lock()
	defer loginAttemptsMu.Unlock()
	delete(loginAttempts, key)
}

// Login handles user authentication and JWT token generation
func Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, r, http.StatusBadRequest, "AUTH_BAD_REQUEST", "Invalid request payload")
		return
	}

	email := strings.TrimSpace(req.Email)
	if email == "" || strings.TrimSpace(req.Password) == "" {
		middleware.WriteError(w, r, http.StatusBadRequest, "AUTH_MISSING_CREDENTIALS", "Email and password are required")
		return
	}

	key := attemptKey(r, email)
	if blocked, remaining := isLoginBlocked(key); blocked {
		middleware.WriteError(w, r, http.StatusTooManyRequests, "AUTH_RATE_LIMIT", "Too many failed attempts. Try again in "+remaining.Round(time.Second).String())
		return
	}

	creds, err := loadAdminCredentials()
	if err != nil {
		middleware.WriteError(w, r, http.StatusInternalServerError, "AUTH_NOT_CONFIGURED", err.Error())
		return
	}

	if !strings.EqualFold(email, creds.email) || !verifyPassword(req.Password, creds) {
		recordLoginFailure(key)
		middleware.WriteError(w, r, http.StatusUnauthorized, "AUTH_INVALID_CREDENTIALS", "Invalid credentials")
		return
	}
	clearLoginAttempts(key)

	user := User{
		ID:    1,
		Name:  "System Administrator",
		Email: creds.email,
		Role:  "admin",
	}

	token, err := issueToken(user)
	if err != nil {
		middleware.WriteError(w, r, http.StatusInternalServerError, "AUTH_TOKEN_ERROR", "Token generation failed")
		return
	}

	writeJSON(w, http.StatusOK, AuthResponse{Token: token, User: user})
}

// Me returns current logged in user details
func Me(w http.ResponseWriter, r *http.Request) {
	authUser, ok := middleware.GetAuthUser(r.Context())
	if !ok {
		middleware.WriteError(w, r, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Unauthorized")
		return
	}

	writeJSON(w, http.StatusOK, User{
		ID:    1,
		Name:  authUser.Name,
		Email: authUser.Email,
		Role:  authUser.Role,
	})
}
