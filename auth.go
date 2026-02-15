package main

import (
	"context"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
)

var alphanumeric = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

type contextKey string

const userContextKey contextKey = "username"

type Auth struct {
	mu        sync.RWMutex
	sessions  map[string]string // token -> username
	usersFile string
}

func NewAuth(usersFile string) *Auth {
	return &Auth{
		sessions:  make(map[string]string),
		usersFile: usersFile,
	}
}

func (a *Auth) checkCredentials(username, password string) bool {
	f, err := os.Open(a.usersFile)
	if err != nil {
		return false
	}
	defer f.Close()

	r := csv.NewReader(f)
	rows, err := r.ReadAll()
	if err != nil {
		return false
	}
	for i, row := range rows {
		if i == 0 { // skip header
			continue
		}
		if len(row) >= 2 && row[0] == username && row[1] == password {
			return true
		}
	}
	return false
}

func (a *Auth) createSession(username string) string {
	b := make([]byte, 16)
	rand.Read(b)
	token := hex.EncodeToString(b)

	a.mu.Lock()
	a.sessions[token] = username
	a.mu.Unlock()
	return token
}

func (a *Auth) getUsername(token string) (string, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	name, ok := a.sessions[token]
	return name, ok
}

func (a *Auth) middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(header, "Bearer ")
		username, ok := a.getUsername(token)
		if !ok {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), userContextKey, username)
		next(w, r.WithContext(ctx))
	}
}

func getUser(r *http.Request) string {
	if u, ok := r.Context().Value(userContextKey).(string); ok {
		return u
	}
	return ""
}

func (a *Auth) userCount() int {
	f, err := os.Open(a.usersFile)
	if err != nil {
		return 0
	}
	defer f.Close()
	r := csv.NewReader(f)
	rows, _ := r.ReadAll()
	count := len(rows) - 1 // subtract header
	if count < 0 {
		return 0
	}
	return count
}

func (a *Auth) userExists(username string) bool {
	f, err := os.Open(a.usersFile)
	if err != nil {
		return false
	}
	defer f.Close()
	r := csv.NewReader(f)
	rows, _ := r.ReadAll()
	for i, row := range rows {
		if i == 0 {
			continue
		}
		if len(row) >= 1 && row[0] == username {
			return true
		}
	}
	return false
}

func (a *Auth) addUser(username, password string) error {
	// create file with header if it doesn't exist
	if _, err := os.Stat(a.usersFile); os.IsNotExist(err) {
		if err := os.WriteFile(a.usersFile, []byte("username,password\n"), 0644); err != nil {
			return err
		}
	}
	f, err := os.OpenFile(a.usersFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	w.Write([]string{username, password})
	w.Flush()
	return w.Error()
}

func handleRegister(auth *Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
			return
		}
		username := strings.TrimSpace(body.Username)
		if username == "" || body.Password == "" {
			http.Error(w, `{"error":"username and password required"}`, http.StatusBadRequest)
			return
		}
		if len(username) < 3 {
			http.Error(w, `{"error":"username must be at least 3 characters"}`, http.StatusBadRequest)
			return
		}
		if len(body.Password) < 4 {
			http.Error(w, `{"error":"password must be at least 4 characters"}`, http.StatusBadRequest)
			return
		}
		if !alphanumeric.MatchString(username) {
			http.Error(w, `{"error":"username must be alphanumeric only"}`, http.StatusBadRequest)
			return
		}
		if !alphanumeric.MatchString(body.Password) {
			http.Error(w, `{"error":"password must be alphanumeric only"}`, http.StatusBadRequest)
			return
		}
		if auth.userCount() >= maxUsers {
			http.Error(w, `{"error":"registration closed — user limit reached"}`, http.StatusForbidden)
			return
		}
		if auth.userExists(username) {
			http.Error(w, `{"error":"username already taken"}`, http.StatusConflict)
			return
		}
		if err := auth.addUser(username, body.Password); err != nil {
			http.Error(w, `{"error":"registration failed"}`, http.StatusInternalServerError)
			return
		}
		token := auth.createSession(username)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"token":    token,
			"username": username,
		})
	}
}

const maxUsers = 10

func handleRegisterInfo(auth *Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		remaining := maxUsers - auth.userCount()
		if remaining < 0 {
			remaining = 0
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]int{"remaining": remaining})
	}
}

func handleLogin(auth *Auth) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
			return
		}
		if !auth.checkCredentials(body.Username, body.Password) {
			http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
			return
		}
		token := auth.createSession(body.Username)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"token":    token,
			"username": body.Username,
		})
	}
}
