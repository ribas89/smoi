package api

import (
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"smoi/internal/database"
)

type adminUser struct {
	ID                string `json:"id"`
	Username          string `json:"username"`
	Email             string `json:"email"`
	DisplayName       string `json:"display_name"`
	Role              string `json:"role"`
	ConversationCount int64  `json:"conversation_count"`
	StorageBytes      int64  `json:"storage_bytes"`
}

type adminConversation struct {
	ID           string `json:"id"`
	UserID       string `json:"user_id"`
	Owner        string `json:"owner"`
	Title        string `json:"title"`
	UpdatedAt    int64  `json:"updated_at"`
	StorageBytes int64  `json:"storage_bytes"`
}

func (s *Server) adminStats(w http.ResponseWriter, r *http.Request, _ database.User) {
	var users, conversations, messages, attachments int64
	err := s.DB.QueryRowContext(r.Context(), `
        SELECT
            (SELECT COUNT(*) FROM users),
            (SELECT COUNT(*) FROM conversations),
            (SELECT COALESCE(SUM(LENGTH(content)), 0) FROM messages),
            (SELECT COALESCE(SUM(size), 0) FROM attachments)
    `).Scan(&users, &conversations, &messages, &attachments)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Could not load storage statistics")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]int64{
		"users":         users,
		"conversations": conversations,
		"storage_bytes": messages + attachments,
	})
}

func (s *Server) adminUsers(w http.ResponseWriter, r *http.Request, _ database.User) {
	rows, err := s.DB.QueryContext(r.Context(), `
        SELECT u.id, u.username, u.email, u.display_name, u.role,
            (SELECT COUNT(*) FROM conversations c WHERE c.user_id = u.id),
            (SELECT COALESCE(SUM(a.size), 0) FROM attachments a WHERE a.user_id = u.id) +
            (SELECT COALESCE(SUM(LENGTH(m.content)), 0)
               FROM messages m JOIN conversations c ON c.id = m.conversation_id
              WHERE c.user_id = u.id) AS total_size
        FROM users u
        ORDER BY total_size DESC, u.id
        LIMIT 500
    `)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Could not list users")
		return
	}
	defer rows.Close()

	users := make([]adminUser, 0)
	for rows.Next() {
		var user adminUser
		err = rows.Scan(
			&user.ID, &user.Username, &user.Email, &user.DisplayName,
			&user.Role, &user.ConversationCount, &user.StorageBytes,
		)
		if err != nil {
			break
		}
		users = append(users, user)
	}

	if err == nil {
		err = rows.Err()
	}
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Could not list users")
		return
	}

	jsonResponse(w, http.StatusOK, map[string]any{"data": users})
}

func (s *Server) adminConversations(w http.ResponseWriter, r *http.Request, _ database.User) {
	userFilter := strings.TrimSpace(r.URL.Query().Get("user_id"))
	rows, err := s.DB.QueryContext(r.Context(), `
        SELECT c.id, c.user_id, u.display_name, c.title, c.updated_at,
            (SELECT COALESCE(SUM(LENGTH(m.content)), 0)
               FROM messages m WHERE m.conversation_id = c.id) +
            (SELECT COALESCE(SUM(a.size), 0)
               FROM attachments a WHERE a.conversation_id = c.id) AS total_size
        FROM conversations c JOIN users u ON u.id = c.user_id
        WHERE (? = '' OR c.user_id = ?)
        ORDER BY total_size DESC, c.updated_at DESC
        LIMIT 500
    `, userFilter, userFilter)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Could not list conversations")
		return
	}
	defer rows.Close()

	conversations := make([]adminConversation, 0)
	for rows.Next() {
		var conversation adminConversation
		err = rows.Scan(
			&conversation.ID, &conversation.UserID, &conversation.Owner,
			&conversation.Title, &conversation.UpdatedAt, &conversation.StorageBytes,
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

	jsonResponse(w, http.StatusOK, map[string]any{"data": conversations})
}

func (s *Server) adminDeleteConversation(w http.ResponseWriter, r *http.Request, _ database.User) {
	conversationID := r.PathValue("id")
	var ownerID string

	err := s.DB.QueryRowContext(r.Context(), `
        SELECT user_id FROM conversations WHERE id = ?
    `, conversationID).Scan(&ownerID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, "Conversation not found")
		return
	}

	if err := s.removeConversation(r.Context(), conversationID, ownerID); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) adminDeleteUser(w http.ResponseWriter, r *http.Request, user database.User) {
	targetID := r.PathValue("id")
	if targetID == user.ID {
		writeErrorResponse(w, http.StatusForbidden, "You cannot delete your own account")
		return
	}

	var foundID string
	err := s.DB.QueryRowContext(r.Context(), `
        SELECT id FROM users WHERE id = ?
    `, targetID).Scan(&foundID)
	if errors.Is(err, sql.ErrNoRows) {
		writeErrorResponse(w, http.StatusNotFound, "User not found")
		return
	}
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Database error")
		return
	}

	directory := s.Storage.UserDir(targetID)
	err = s.Storage.DeleteWithDatabase(directory, func() error {
		result, err := s.DB.ExecContext(r.Context(), `
            DELETE FROM users WHERE id = ?
        `, targetID)
		if err != nil {
			return err
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected != 1 {
			return sql.ErrNoRows
		}
		return nil
	})
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	jsonResponse(w, http.StatusOK, map[string]bool{"ok": true})
}
