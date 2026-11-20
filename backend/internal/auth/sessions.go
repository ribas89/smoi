package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"smoi/internal/database"
	"smoi/internal/storage"
)

func randomString() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func hash(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func setCookie(
	w http.ResponseWriter, name, value, path string, maxAge int, secure bool,
) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (a *Auth) Exchange(w http.ResponseWriter, r *http.Request) {
	exchangeCookie, err := r.Cookie("smoi_login_exchange")
	setCookie(w, "smoi_login_exchange", "", "/api/auth/exchange", -1, a.cookieSecure)
	if err != nil || exchangeCookie.Value == "" {
		http.Error(w, "login exchange expired", http.StatusUnauthorized)
		return
	}

	token, expiration, err := a.exchangeForSession(r.Context(), exchangeCookie.Value)
	if err != nil {
		http.Error(w, "session unavailable or exchange expired", http.StatusUnauthorized)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"token":      token,
		"expires_at": expiration,
	})
}

func (a *Auth) exchangeForSession(
	ctx context.Context, exchange string,
) (string, int64, error) {
	transaction, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return "", 0, err
	}
	defer transaction.Rollback()

	var userID string
	err = transaction.QueryRowContext(ctx, `
        SELECT user_id FROM login_exchanges
        WHERE token_hash = ? AND expires_at > ?
    `, hash(exchange), time.Now().Unix()).Scan(&userID)
	if err != nil {
		return "", 0, err
	}

	if _, err := transaction.ExecContext(ctx, `
        DELETE FROM login_exchanges WHERE token_hash = ?
    `, hash(exchange)); err != nil {
		return "", 0, err
	}

	token, err := randomString()
	if err != nil {
		return "", 0, err
	}

	now := time.Now().Unix()
	expiration := now + int64(a.sessionHours)*3600
	_, err = transaction.ExecContext(ctx, `
        INSERT INTO sessions(id, user_id, session_token_hash, expires_at, created_at)
        VALUES (?, ?, ?, ?, ?)
    `, storage.NewID(), userID, hash(token), expiration, now)
	if err != nil {
		return "", 0, err
	}

	if err := transaction.Commit(); err != nil {
		return "", 0, err
	}
	return token, expiration, nil
}

func (a *Auth) Current(ctx context.Context, bearer string) (database.User, error) {
	var user database.User
	if bearer == "" {
		return user, sql.ErrNoRows
	}

	err := a.db.QueryRowContext(ctx, `
        SELECT u.id, u.username, u.email, u.display_name, u.avatar_url, u.role
        FROM sessions s JOIN users u ON u.id = s.user_id
        WHERE s.session_token_hash = ? AND s.expires_at > ?
    `, hash(bearer), time.Now().Unix()).Scan(
		&user.ID, &user.Username, &user.Email, &user.DisplayName,
		&user.AvatarURL, &user.Role,
	)
	return user, err
}

func Token(r *http.Request) string {
	authorization := r.Header.Get("Authorization")
	if !strings.HasPrefix(authorization, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer "))
}

func (a *Auth) Revoke(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}

	_, err := a.db.ExecContext(ctx, `
        DELETE FROM sessions WHERE session_token_hash = ?
    `, hash(token))
	return err
}
