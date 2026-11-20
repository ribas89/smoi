package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
	"smoi/migrations"
)

type DB struct {
	*sql.DB
}

type User struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	AvatarURL   string `json:"avatar_url"`
	Role        string `json:"role"`
}

type Conversation struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Title     string `json:"title"`
	Model     string `json:"model"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
}

type Attachment struct {
	ID               string         `json:"id"`
	UserID           string         `json:"user_id"`
	ConversationID   string         `json:"conversation_id"`
	MessageID        sql.NullString `json:"-"`
	Filename         string         `json:"filename"`
	MIMEType         string         `json:"mime_type"`
	Size             int64          `json:"size"`
	StorageReference string         `json:"-"`
	CreatedAt        int64          `json:"created_at"`
}

type Message struct {
	ID             string       `json:"id"`
	ConversationID string       `json:"conversation_id"`
	Role           string       `json:"role"`
	Content        string       `json:"content"`
	CreatedAt      int64        `json:"created_at"`
	Attachments    []Attachment `json:"attachments"`
}

func Open(path string) (*DB, error) {
	connection, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	connection.SetMaxOpenConns(1)
	connection.SetMaxIdleConns(1)
	connection.SetConnMaxLifetime(0)

	if err := configureSQLite(connection); err != nil {
		_ = connection.Close()
		return nil, err
	}
	if err := applyMigrations(connection); err != nil {
		_ = connection.Close()
		return nil, err
	}

	return &DB{DB: connection}, nil
}

func configureSQLite(connection *sql.DB) error {
	settings := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA busy_timeout=5000",
		"PRAGMA synchronous=NORMAL",
	}

	for _, setting := range settings {
		if _, err := connection.Exec(setting); err != nil {
			return fmt.Errorf("sqlite %s: %w", setting, err)
		}
	}
	return nil
}

func applyMigrations(connection *sql.DB) error {
	_, err := connection.Exec(`
        CREATE TABLE IF NOT EXISTS schema_migrations (
            version TEXT PRIMARY KEY,
            applied_at INTEGER NOT NULL
        )
    `)
	if err != nil {
		return err
	}

	entries, err := migrations.Files.ReadDir(".")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		if err := applyMigration(connection, entry.Name()); err != nil {
			return err
		}
	}
	return nil
}

func applyMigration(connection *sql.DB, filename string) error {
	var alreadyApplied int
	err := connection.QueryRow(`
        SELECT COUNT(*) FROM schema_migrations WHERE version = ?
    `, filename).Scan(&alreadyApplied)
	if err != nil || alreadyApplied != 0 {
		return err
	}

	sqlText, err := migrations.Files.ReadFile(filename)
	if err != nil {
		return err
	}

	transaction, err := connection.BeginTx(context.Background(), nil)
	if err != nil {
		return err
	}
	defer transaction.Rollback()

	for _, statement := range strings.Split(string(sqlText), ";") {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}
		if _, err := transaction.Exec(statement); err != nil {
			return fmt.Errorf("migration %s: %w", filename, err)
		}
	}

	_, err = transaction.Exec(`
        INSERT INTO schema_migrations(version, applied_at) VALUES (?, ?)
    `, filename, time.Now().Unix())
	if err != nil {
		return err
	}

	return transaction.Commit()
}

func (db *DB) GetConversation(
	ctx context.Context, conversationID, userID string,
) (Conversation, error) {
	var conversation Conversation
	err := db.QueryRowContext(ctx, `
        SELECT id, user_id, title, model, created_at, updated_at
        FROM conversations
        WHERE id = ? AND user_id = ?
    `, conversationID, userID).Scan(
		&conversation.ID, &conversation.UserID, &conversation.Title,
		&conversation.Model, &conversation.CreatedAt, &conversation.UpdatedAt,
	)

	return conversation, err
}

func (db *DB) Messages(ctx context.Context, conversationID string) ([]Message, error) {
	rows, err := db.QueryContext(ctx, `
        SELECT id, conversation_id, role, content, created_at
        FROM messages
        WHERE conversation_id = ?
        ORDER BY created_at, id
    `, conversationID)
	if err != nil {
		return nil, err
	}

	messages := make([]Message, 0)
	for rows.Next() {
		var message Message
		message.Attachments = []Attachment{}
		err = rows.Scan(
			&message.ID, &message.ConversationID, &message.Role,
			&message.Content, &message.CreatedAt,
		)
		if err != nil {
			break
		}
		messages = append(messages, message)
	}
	if err == nil {
		err = rows.Err()
	}
	_ = rows.Close()
	if err != nil {
		return nil, err
	}

	for index := range messages {
		attachments, err := db.messageAttachments(ctx, messages[index].ID)
		if err != nil {
			return nil, err
		}
		messages[index].Attachments = attachments
	}

	return messages, nil
}

func (db *DB) messageAttachments(ctx context.Context, messageID string) ([]Attachment, error) {
	rows, err := db.QueryContext(ctx, `
        SELECT id, user_id, conversation_id, message_id,
               filename, mime_type, size, storage_reference, created_at
        FROM attachments
        WHERE message_id = ?
        ORDER BY created_at, id
    `, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	attachments := make([]Attachment, 0)
	for rows.Next() {
		var attachment Attachment
		err = rows.Scan(
			&attachment.ID, &attachment.UserID, &attachment.ConversationID,
			&attachment.MessageID, &attachment.Filename, &attachment.MIMEType,
			&attachment.Size, &attachment.StorageReference, &attachment.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, attachment)
	}

	return attachments, rows.Err()
}
