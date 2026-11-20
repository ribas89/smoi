package api

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"smoi/internal/auth"
	"smoi/internal/database"
	"smoi/internal/storage"
	"smoi/internal/upstream"
)

type Server struct {
	DB        *database.DB
	Auth      *auth.Auth
	Storage   *storage.Store
	Upstream  *upstream.Client
	MaxUpload int64
	Jobs      *JobManager
}

type userHandler func(http.ResponseWriter, *http.Request, database.User)

func jsonResponse(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeErrorResponse(w http.ResponseWriter, status int, message string) {
	jsonResponse(w, status, map[string]string{"error": message})
}

func decodeJSONRequest(r *http.Request, output any) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(output); err != nil {
		return err
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errors.New("unexpected trailing JSON")
	}

	return nil
}

func (s *Server) withUser(next userHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := s.Auth.Current(r.Context(), auth.Token(r))
		if err != nil {
			writeErrorResponse(w, http.StatusUnauthorized, "Authentication required or session expired")
			return
		}

		started := time.Now()
		slog.Debug("client request", "user", user.ID, "method", r.Method, "path", r.URL.Path)
		next(w, r, user)
		slog.Debug("client request finished", "user", user.ID, "method", r.Method,
			"path", r.URL.Path, "duration", time.Since(started).String())
	}
}

func (s *Server) withAdmin(next userHandler) http.HandlerFunc {
	return s.withUser(func(w http.ResponseWriter, r *http.Request, user database.User) {
		if user.Role != "admin" {
			writeErrorResponse(w, http.StatusForbidden, "Administrator access required")
			return
		}

		next(w, r, user)
	})
}
