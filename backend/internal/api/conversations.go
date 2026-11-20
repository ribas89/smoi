package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"smoi/internal/database"
	"smoi/internal/storage"
)

const conversationPageSize = 50

type createConversationRequest struct {
	Title string `json:"title"`
	Model string `json:"model"`
}

type updateConversationRequest struct {
	Title *string `json:"title"`
	Model *string `json:"model"`
}

func (s *Server) listConversations(w http.ResponseWriter, r *http.Request, user database.User) {
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if offset < 0 {
		offset = 0
	}

	searchFilter := "%" + strings.TrimSpace(r.URL.Query().Get("q")) + "%"
	rows, err := s.DB.QueryContext(r.Context(), `
        SELECT c.id, c.user_id, c.title, c.model, c.created_at, c.updated_at
        FROM conversations c
        WHERE c.user_id = ?
          AND (
            c.title LIKE ?
            OR EXISTS (
              SELECT 1
              FROM messages m
              WHERE m.conversation_id = c.id AND m.content LIKE ?
            )
          )
        ORDER BY c.updated_at DESC, c.id DESC
        LIMIT ? OFFSET ?
    `, user.ID, searchFilter, searchFilter, conversationPageSize, offset)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Could not list conversations")
		return
	}
	defer rows.Close()

	conversations := make([]database.Conversation, 0)
	for rows.Next() {
		var conversation database.Conversation
		err = rows.Scan(
			&conversation.ID, &conversation.UserID, &conversation.Title,
			&conversation.Model, &conversation.CreatedAt, &conversation.UpdatedAt,
		)
		if err != nil {
			break
		}
		conversations = append(conversations, conversation)
	}

	if err == nil {
		err = rows.Err()
	}
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Could not list conversations")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{
		"data":     conversations,
		"has_more": len(conversations) == conversationPageSize,
	})
}

func (s *Server) createConversation(w http.ResponseWriter, r *http.Request, user database.User) {
	var input createConversationRequest
	if err := decodeJSONRequest(r, &input); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid conversation payload")
		return
	}

	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		input.Title = "New chat"
	}
	if input.Model == "" {
		input.Model = s.Upstream.DefaultModel
	}
	if len(input.Title) > 160 || len(input.Model) > 250 {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid conversation title or model")
		return
	}

	now := time.Now().Unix()
	conversation := database.Conversation{
		ID:        storage.NewID(),
		UserID:    user.ID,
		Title:     input.Title,
		Model:     input.Model,
		CreatedAt: now,
		UpdatedAt: now,
	}

	_, err := s.DB.ExecContext(r.Context(), `
        INSERT INTO conversations(id, user_id, title, model, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?)
    `, conversation.ID, user.ID, conversation.Title, conversation.Model, now, now)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Could not create conversation")
		return
	}

	jsonResponse(w, http.StatusCreated, conversation)
}

func (s *Server) ownedConversation(
	w http.ResponseWriter, r *http.Request, user database.User,
) (database.Conversation, bool) {
	conversation, err := s.DB.GetConversation(r.Context(), r.PathValue("id"), user.ID)
	if errors.Is(err, sql.ErrNoRows) {
		writeErrorResponse(w, http.StatusNotFound, "Conversation not found")
		return conversation, false
	}
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Database error")
		return conversation, false
	}

	return conversation, true
}

func (s *Server) getConversation(w http.ResponseWriter, r *http.Request, user database.User) {
	if conversation, ok := s.ownedConversation(w, r, user); ok {
		jsonResponse(w, http.StatusOK, conversation)
	}
}

func (s *Server) patchConversation(w http.ResponseWriter, r *http.Request, user database.User) {
	conversation, ok := s.ownedConversation(w, r, user)
	if !ok {
		return
	}

	var input updateConversationRequest
	if err := decodeJSONRequest(r, &input); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid update payload")
		return
	}

	if input.Title != nil {
		conversation.Title = strings.TrimSpace(*input.Title)
		if len(conversation.Title) == 0 || len(conversation.Title) > 160 {
			writeErrorResponse(w, http.StatusBadRequest, "Title must be 1 to 160 characters")
			return
		}
	}
	if input.Model != nil {
		conversation.Model = strings.TrimSpace(*input.Model)
		if len(conversation.Model) > 250 {
			writeErrorResponse(w, http.StatusBadRequest, "Model name too long")
			return
		}
	}

	conversation.UpdatedAt = time.Now().Unix()
	_, err := s.DB.ExecContext(r.Context(), `
        UPDATE conversations
        SET title = ?, model = ?, updated_at = ?
        WHERE id = ? AND user_id = ?
    `, conversation.Title, conversation.Model, conversation.UpdatedAt, conversation.ID, user.ID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Could not update conversation")
		return
	}

	jsonResponse(w, http.StatusOK, conversation)
}

func (s *Server) removeConversation(ctx context.Context, conversationID, userID string) error {
	s.Jobs.Stop(conversationID)
	directory := s.Storage.ConversationDir(userID, conversationID)

	return s.Storage.DeleteWithDatabase(directory, func() error {
		result, err := s.DB.ExecContext(ctx, `
            DELETE FROM conversations WHERE id = ? AND user_id = ?
        `, conversationID, userID)
		if err != nil {
			return err
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return sql.ErrNoRows
		}
		return nil
	})
}

func (s *Server) deleteConversation(w http.ResponseWriter, r *http.Request, user database.User) {
	conversation, ok := s.ownedConversation(w, r, user)
	if !ok {
		return
	}

	if err := s.removeConversation(r.Context(), conversation.ID, user.ID); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) listMessages(w http.ResponseWriter, r *http.Request, user database.User) {
	conversation, ok := s.ownedConversation(w, r, user)
	if !ok {
		return
	}

	messages, err := s.DB.Messages(r.Context(), conversation.ID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Could not load messages")
		return
	}

	jobs, err := s.conversationJobs(r.Context(), conversation.ID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Could not load generation state")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{"data": messages, "jobs": jobs})
}
