package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"smoi/internal/storage"
)

func (a *Auth) provision(
	ctx context.Context,
	issuer, subject, username, email, displayName, avatar string,
) (string, error) {
	transaction, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer transaction.Rollback()

	now := time.Now().Unix()
	var userID string
	err = transaction.QueryRowContext(ctx, `
        SELECT id FROM users WHERE oidc_issuer = ? AND oidc_subject = ?
    `, issuer, subject).Scan(&userID)

	switch {
	case err == nil:
		err = updateExistingUser(ctx, transaction, userID, username, email, displayName, avatar, now)
	case errors.Is(err, sql.ErrNoRows):
		userID, err = createUser(ctx, transaction, issuer, subject, username, email, displayName, avatar, now)
	}
	if err != nil {
		return "", err
	}

	if err := transaction.Commit(); err != nil {
		return "", err
	}
	return userID, nil
}

func updateExistingUser(
	ctx context.Context, transaction *sql.Tx,
	userID, username, email, displayName, avatar string, now int64,
) error {
	_, err := transaction.ExecContext(ctx, `
        UPDATE users
        SET username = ?, email = ?, display_name = ?,
            avatar_url = COALESCE(NULLIF(?, ''), avatar_url),
            updated_at = ?, last_login_at = ?
        WHERE id = ?
    `, username, email, displayName, avatar, now, now, userID)
	return err
}

func createUser(
	ctx context.Context, transaction *sql.Tx,
	issuer, subject, username, email, displayName, avatar string, now int64,
) (string, error) {
	var userCount int
	err := transaction.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&userCount)
	if err != nil {
		return "", err
	}

	role := "user"
	if userCount == 0 {
		role = "admin"
	}

	userID := storage.NewID()
	_, err = transaction.ExecContext(ctx, `
        INSERT INTO users(
            id, oidc_issuer, oidc_subject, username, email,
            display_name, avatar_url, role, created_at, updated_at, last_login_at
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `, userID, issuer, subject, username, email, displayName, avatar, role, now, now, now)

	return userID, err
}
