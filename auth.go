package main

import (
	"context"
	"crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"sync"
)

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
