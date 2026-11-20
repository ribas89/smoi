package api

import (
	"context"
	"log/slog"
	"os"
	"time"
)

type pendingAttachment struct {
	id   string
	path string
}

func (s *Server) Housekeeping(ctx context.Context) {
	s.Storage.CleanTemporary(24 * time.Hour)
	s.removeAbandonedAttachments(ctx)

	expiration := time.Now().Unix()
	for _, table := range []string{"sessions", "login_exchanges", "oidc_flows"} {
		query := "DELETE FROM " + table + " WHERE expires_at < ?"
		if _, err := s.DB.ExecContext(ctx, query, expiration); err != nil {
			slog.Warn("housekeeping failed", "table", table, "error", err)
		}
	}
}

func (s *Server) removeAbandonedAttachments(ctx context.Context) {
	cutoff := time.Now().Add(-24 * time.Hour).Unix()
	rows, err := s.DB.QueryContext(ctx, `
        SELECT id, storage_reference
        FROM attachments
        WHERE message_id IS NULL AND created_at < ?
    `, cutoff)
	if err != nil {
		slog.Warn("could not list abandoned attachments", "error", err)
		return
	}

	attachments := make([]pendingAttachment, 0)
	for rows.Next() {
		var attachment pendingAttachment
		if err := rows.Scan(&attachment.id, &attachment.path); err == nil {
			attachments = append(attachments, attachment)
		}
	}
	_ = rows.Close()

	for _, attachment := range attachments {
		_, err := s.DB.ExecContext(ctx, `
            DELETE FROM attachments WHERE id = ? AND message_id IS NULL
        `, attachment.id)
		if err != nil {
			continue
		}

		if path, err := s.Storage.Path(attachment.path); err == nil {
			_ = os.Remove(path)
		}
	}
}
