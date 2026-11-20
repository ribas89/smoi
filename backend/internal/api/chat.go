package api

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"smoi/internal/database"
	"smoi/internal/storage"
)

type postMessageRequest struct {
	Content       string   `json:"content"`
	AttachmentIDs []string `json:"attachment_ids"`
	RequestID     string   `json:"request_id"`
}

type messageSaveError struct {
	status  int
	message string
}

func (err messageSaveError) Error() string {
	return err.message
}

func validateMessage(input *postMessageRequest) error {
	input.Content = strings.TrimSpace(input.Content)

	if len(input.Content) > 120000 || len(input.AttachmentIDs) > 20 {
		return errors.New("Message too large or too many attachments")
	}
	if input.Content == "" && len(input.AttachmentIDs) == 0 {
		return errors.New("Message cannot be empty")
	}
	if len(input.RequestID) < 8 || len(input.RequestID) > 128 {
		return errors.New("Invalid request ID")
	}

	return nil
}

func (s *Server) sendMessage(w http.ResponseWriter, r *http.Request, user database.User) {
	conversation, ok := s.ownedConversation(w, r, user)
	if !ok {
		return
	}

	var input postMessageRequest
	if err := decodeJSONRequest(r, &input); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid message body")
		return
	}
	if err := validateMessage(&input); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	job, err := s.saveMessage(r.Context(), conversation, user.ID, input)
	if err != nil {
		var saveError messageSaveError
		if errors.As(err, &saveError) {
			writeErrorResponse(w, saveError.status, saveError.message)
		} else {
			slog.Error("message submission failed", "conversation", conversation.ID, "error", err)
			writeErrorResponse(w, http.StatusInternalServerError, "Could not save message")
		}
		return
	}

	slog.Debug("incoming message", "user", user.ID, "conversation", conversation.ID,
		"job", job.ID, "content", input.Content, "attachments", input.AttachmentIDs)
	s.Jobs.Supersede(job)
	jsonResponse(w, http.StatusAccepted, job)
}

func (s *Server) saveMessage(
	ctx context.Context, conversation database.Conversation, userID string, input postMessageRequest,
) (GenerationJob, error) {
	transaction, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return GenerationJob{}, err
	}
	defer transaction.Rollback()

	now := time.Now().UnixMicro()
	var latest int64
	if err := transaction.QueryRowContext(ctx, `
        SELECT COALESCE(MAX(created_at), 0) FROM messages WHERE conversation_id = ?
    `, conversation.ID).Scan(&latest); err != nil {
		return GenerationJob{}, err
	}
	if now <= latest+1 {
		now = latest + 2
	}

	userMessageID := storage.NewID()
	assistantMessageID := storage.NewID()
	job := GenerationJob{
		ID:                 storage.NewID(),
		ConversationID:     conversation.ID,
		UserMessageID:      userMessageID,
		AssistantMessageID: assistantMessageID,
		Status:             "pending",
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	_, err = transaction.ExecContext(ctx, `
        UPDATE generation_jobs
        SET status = 'stopped', error = 'Interrupted by a newer message', updated_at = ?
        WHERE conversation_id = ? AND status = 'pending'
    `, now, conversation.ID)
	if err != nil {
		return GenerationJob{}, err
	}

	_, err = transaction.ExecContext(ctx, `
        INSERT INTO messages(id, conversation_id, role, content, client_request_id, created_at)
        VALUES (?, ?, 'user', ?, ?, ?)
    `, userMessageID, conversation.ID, input.Content, input.RequestID, now)
	if err != nil {
		return GenerationJob{}, messageSaveError{
			status:  http.StatusConflict,
			message: "This message was already submitted. Reload the conversation before retrying.",
		}
	}

	if err := s.attachFilesToMessage(ctx, transaction, userMessageID, userID, conversation.ID, input.AttachmentIDs); err != nil {
		return GenerationJob{}, messageSaveError{status: http.StatusBadRequest, message: err.Error()}
	}

	_, err = transaction.ExecContext(ctx, `
        INSERT INTO messages(id, conversation_id, role, content, created_at)
        VALUES (?, ?, 'assistant', '', ?)
    `, assistantMessageID, conversation.ID, now+1)
	if err != nil {
		return GenerationJob{}, err
	}

	_, err = transaction.ExecContext(ctx, `
        INSERT INTO generation_jobs(id, conversation_id, user_message_id, assistant_message_id,
                                    model, status, error, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, 'pending', '', ?, ?)
    `, job.ID, conversation.ID, userMessageID, assistantMessageID, conversation.Model, now, now)
	if err != nil {
		return GenerationJob{}, err
	}

	title := conversationTitle(conversation, input.Content)
	_, err = transaction.ExecContext(ctx, `
        UPDATE conversations SET title = ?, updated_at = ? WHERE id = ? AND user_id = ?
    `, title, time.Now().Unix(), conversation.ID, userID)
	if err != nil {
		return GenerationJob{}, err
	}

	return job, transaction.Commit()
}

func (s *Server) attachFilesToMessage(
	ctx context.Context,
	transaction *sql.Tx,
	messageID, userID, conversationID string,
	attachmentIDs []string,
) error {
	for _, attachmentID := range attachmentIDs {
		result, err := transaction.ExecContext(ctx, `
            UPDATE attachments SET message_id = ?
            WHERE id = ? AND user_id = ? AND conversation_id = ? AND message_id IS NULL
        `, messageID, attachmentID, userID, conversationID)
		if err != nil {
			return err
		}

		changed, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if changed != 1 {
			return errors.New("attachment is unavailable or already used")
		}
	}

	return nil
}

func conversationTitle(conversation database.Conversation, content string) string {
	if conversation.Title != "New chat" || content == "" {
		return conversation.Title
	}

	title := []rune(strings.ReplaceAll(content, "\n", " "))
	if len(title) > 55 {
		title = title[:55]
	}
	return string(title)
}
