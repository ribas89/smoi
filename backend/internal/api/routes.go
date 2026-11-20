package api

import (
	"net/http"

	"smoi/internal/auth"
	"smoi/internal/database"
)

func (s *Server) RegisterRoutes(router *http.ServeMux) {
	router.HandleFunc("GET /api/auth/login", s.Auth.Login)
	router.HandleFunc("GET /api/auth/callback", s.Auth.Callback)
	router.HandleFunc("POST /api/auth/exchange", s.Auth.Exchange)
	router.HandleFunc("POST /api/auth/logout", s.withUser(s.logout))
	router.HandleFunc("GET /api/auth/me", s.withUser(s.currentUser))

	router.HandleFunc("GET /api/config", s.withUser(s.config))
	router.HandleFunc("GET /api/models", s.withUser(s.models))

	router.HandleFunc("GET /api/conversations", s.withUser(s.listConversations))
	router.HandleFunc("POST /api/conversations", s.withUser(s.createConversation))
	router.HandleFunc("GET /api/conversations/{id}", s.withUser(s.getConversation))
	router.HandleFunc("PATCH /api/conversations/{id}", s.withUser(s.patchConversation))
	router.HandleFunc("DELETE /api/conversations/{id}", s.withUser(s.deleteConversation))
	router.HandleFunc("GET /api/conversations/{id}/messages", s.withUser(s.listMessages))
	router.HandleFunc("POST /api/conversations/{id}/messages", s.withUser(s.sendMessage))
	router.HandleFunc("POST /api/conversations/{id}/stop", s.withUser(s.stopGeneration))

	router.HandleFunc("POST /api/attachments", s.withUser(s.uploadAttachment))
	router.HandleFunc("GET /api/attachments/{id}", s.withUser(s.getAttachment))

	router.HandleFunc("GET /api/admin/stats", s.withAdmin(s.adminStats))
	router.HandleFunc("GET /api/admin/users", s.withAdmin(s.adminUsers))
	router.HandleFunc("DELETE /api/admin/users/{id}", s.withAdmin(s.adminDeleteUser))
	router.HandleFunc("GET /api/admin/conversations", s.withAdmin(s.adminConversations))
	router.HandleFunc("DELETE /api/admin/conversations/{id}", s.withAdmin(s.adminDeleteConversation))
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request, _ database.User) {
	if err := s.Auth.Revoke(r.Context(), auth.Token(r)); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Logout failed")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) currentUser(w http.ResponseWriter, _ *http.Request, user database.User) {
	jsonResponse(w, http.StatusOK, user)
}

func (s *Server) config(w http.ResponseWriter, _ *http.Request, _ database.User) {
	jsonResponse(w, http.StatusOK, map[string]any{
		"max_upload_mb": float64(s.MaxUpload) / 1024 / 1024,
		"default_model": s.Upstream.DefaultModel,
	})
}

func (s *Server) models(w http.ResponseWriter, r *http.Request, _ database.User) {
	models, err := s.Upstream.Models(r.Context())
	if err != nil {
		writeErrorResponse(w, http.StatusBadGateway, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{
		"data":          models,
		"default_model": s.Upstream.DefaultModel,
	})
}
