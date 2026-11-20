package api

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"smoi/internal/database"
	"smoi/internal/storage"
)

func (s *Server) uploadAttachment(w http.ResponseWriter, r *http.Request, user database.User) {
	r.Body = http.MaxBytesReader(w, r.Body, s.MaxUpload+1024*1024)

	reader, err := r.MultipartReader()
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Multipart upload required")
		return
	}

	var conversationID string
	var uploaded *database.Attachment

	for {
		part, readErr := reader.NextPart()
		if errors.Is(readErr, io.EOF) {
			break
		}
		if readErr != nil {
			writeErrorResponse(w, http.StatusBadRequest, "Malformed multipart upload")
			return
		}

		switch part.FormName() {
		case "conversation_id":
			conversationID, err = s.readUploadConversation(r, part, user)
		case "file":
			if conversationID == "" || uploaded != nil {
				err = errors.New("provide conversation_id before one file")
			} else {
				uploaded, err = s.saveUploadedAttachment(r, part, user, conversationID)
			}
		}
		_ = part.Close()

		if err != nil {
			writeErrorResponse(w, uploadErrorStatus(err), err.Error())
			return
		}
	}

	if uploaded == nil {
		writeErrorResponse(w, http.StatusBadRequest, "No file uploaded")
		return
	}

	jsonResponse(w, http.StatusCreated, uploaded)
}

func (s *Server) readUploadConversation(
	r *http.Request, part *multipart.Part, user database.User,
) (string, error) {
	bytes, err := io.ReadAll(io.LimitReader(part, 129))
	if err != nil || len(bytes) > 128 || len(bytes) == 0 {
		return "", errors.New("invalid conversation ID")
	}

	conversationID := string(bytes)
	if _, err := s.DB.GetConversation(r.Context(), conversationID, user.ID); err != nil {
		return "", errors.New("conversation not found")
	}

	return conversationID, nil
}

func (s *Server) saveUploadedAttachment(
	r *http.Request, part *multipart.Part, user database.User, conversationID string,
) (*database.Attachment, error) {
	filename := filepath.Base(strings.ReplaceAll(part.FileName(), "\\", "/"))
	if filename == "" || filename == "." || len(filename) > 255 {
		return nil, errors.New("invalid filename")
	}

	mimeType, _, mimeErr := mime.ParseMediaType(part.Header.Get("Content-Type"))
	if mimeErr != nil || mimeType == "" || mimeType == "application/octet-stream" {
		mimeType = mimeFromFilename(filename)
	}

	attachmentID := storage.NewID()
	reference, size, err := s.Storage.Save(
		user.ID, conversationID, attachmentID, filename, part, s.MaxUpload,
	)
	if err != nil {
		return nil, fmt.Errorf("file upload failed: %w", err)
	}

	attachment := &database.Attachment{
		ID:               attachmentID,
		UserID:           user.ID,
		ConversationID:   conversationID,
		Filename:         filename,
		MIMEType:         mimeType,
		Size:             size,
		StorageReference: reference,
		CreatedAt:        time.Now().Unix(),
	}

	_, err = s.DB.ExecContext(r.Context(), `
        INSERT INTO attachments(
            id, user_id, conversation_id, message_id, filename,
            mime_type, size, storage_reference, created_at
        ) VALUES (?, ?, ?, NULL, ?, ?, ?, ?, ?)
    `, attachment.ID, user.ID, conversationID, filename, mimeType,
		size, reference, attachment.CreatedAt)
	if err != nil {
		if absolute, pathErr := s.Storage.Path(reference); pathErr == nil {
			_ = os.Remove(absolute)
		}
		return nil, errors.New("could not save attachment metadata")
	}

	slog.Debug("attachment received", "user", user.ID, "conversation", conversationID,
		"file", attachment.ID, "name", filename, "mime", mimeType, "bytes", size)
	return attachment, nil
}

func uploadErrorStatus(err error) int {
	message := err.Error()

	switch {
	case strings.Contains(message, "exceeds"):
		return http.StatusRequestEntityTooLarge
	case strings.Contains(message, "conversation not found"):
		return http.StatusNotFound
	case strings.Contains(message, "could not save attachment metadata"):
		return http.StatusInternalServerError
	default:
		return http.StatusBadRequest
	}
}

func (s *Server) getAttachment(w http.ResponseWriter, r *http.Request, user database.User) {
	attachment, err := s.findOwnedAttachment(r, user)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, "Attachment not found")
		return
	}

	absolute, err := s.Storage.Path(attachment.StorageReference)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Invalid attachment path")
		return
	}

	file, err := os.Open(absolute)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, "Stored file unavailable")
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Stored file unreadable")
		return
	}

	disposition := "attachment"
	if r.URL.Query().Get("inline") == "1" && strings.HasPrefix(attachment.MIMEType, "image/") {
		disposition = "inline"
	}

	w.Header().Set("Content-Type", attachment.MIMEType)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
	w.Header().Set("Content-Disposition", fmt.Sprintf(
		`%s; filename*=UTF-8''%s`, disposition, encodeDownloadFilename(attachment.Filename),
	))
	w.Header().Set("Cache-Control", "private, no-store")
	http.ServeContent(w, r, attachment.Filename, info.ModTime(), file)
}

func (s *Server) findOwnedAttachment(
	r *http.Request, user database.User,
) (database.Attachment, error) {
	var attachment database.Attachment
	err := s.DB.QueryRowContext(r.Context(), `
        SELECT a.id, a.user_id, a.conversation_id, a.message_id,
               a.filename, a.mime_type, a.size, a.storage_reference, a.created_at
        FROM attachments a
        JOIN conversations c ON c.id = a.conversation_id
        WHERE a.id = ? AND a.user_id = ? AND c.user_id = ?
              AND a.message_id IS NOT NULL
    `, r.PathValue("id"), user.ID, user.ID).Scan(
		&attachment.ID, &attachment.UserID, &attachment.ConversationID,
		&attachment.MessageID, &attachment.Filename, &attachment.MIMEType,
		&attachment.Size, &attachment.StorageReference, &attachment.CreatedAt,
	)

	return attachment, err
}

func encodeDownloadFilename(name string) string {
	var encoded strings.Builder

	for _, char := range []byte(name) {
		allowed := char >= 'a' && char <= 'z' ||
			char >= 'A' && char <= 'Z' ||
			char >= '0' && char <= '9' ||
			char == '-' || char == '_' || char == '.'

		if allowed {
			encoded.WriteByte(char)
		} else {
			_, _ = fmt.Fprintf(&encoded, "%%%02X", char)
		}
	}

	return encoded.String()
}
