package auth

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
	"smoi/internal/database"
)

type Auth struct {
	db           *database.DB
	provider     *oidc.Provider
	verifier     *oidc.IDTokenVerifier
	oauth        oauth2.Config
	issuer       string
	cookieSecure bool
	sessionHours int
}

type identityClaims struct {
	issuer      string
	subject     string
	username    string
	email       string
	displayName string
	avatar      string
}

func New(
	ctx context.Context,
	db *database.DB,
	issuer, clientID, clientSecret, redirect string,
	sessionHours int,
) (*Auth, error) {
	if issuer == "" || clientID == "" || redirect == "" {
		return nil, errors.New("OIDC_ISSUER_URL, OIDC_CLIENT_ID and OIDC_REDIRECT_URL are required")
	}

	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("OIDC discovery: %w", err)
	}

	callback, err := url.Parse(redirect)
	if err != nil || callback.Scheme == "" || callback.Host == "" {
		return nil, errors.New("OIDC_REDIRECT_URL must be absolute")
	}
	if sessionHours <= 0 {
		sessionHours = 168
	}

	return &Auth{
		db:       db,
		provider: provider,
		verifier: provider.Verifier(&oidc.Config{ClientID: clientID}),
		oauth: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Endpoint:     provider.Endpoint(),
			RedirectURL:  redirect,
			Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
		},
		issuer:       issuer,
		cookieSecure: callback.Scheme == "https",
		sessionHours: sessionHours,
	}, nil
}

func (a *Auth) Login(w http.ResponseWriter, r *http.Request) {
	state, err := randomString()
	if err != nil {
		http.Error(w, "randomness unavailable", http.StatusInternalServerError)
		return
	}
	nonce, err := randomString()
	if err != nil {
		http.Error(w, "randomness unavailable", http.StatusInternalServerError)
		return
	}
	verifier := oauth2.GenerateVerifier()

	_, err = a.db.ExecContext(r.Context(), `
        INSERT INTO oidc_flows(state_hash, nonce, verifier, expires_at)
        VALUES (?, ?, ?, ?)
    `, hash(state), nonce, verifier, time.Now().Add(10*time.Minute).Unix())
	if err != nil {
		http.Error(w, "could not create login flow", http.StatusInternalServerError)
		return
	}

	setCookie(w, "smoi_oidc_flow", state, "/api/auth/callback", 600, a.cookieSecure)
	destination := a.oauth.AuthCodeURL(
		state,
		oauth2.S256ChallengeOption(verifier),
		oauth2.SetAuthURLParam("nonce", nonce),
	)
	http.Redirect(w, r, destination, http.StatusFound)
}

func (a *Auth) Callback(w http.ResponseWriter, r *http.Request) {
	flowCookie, cookieErr := r.Cookie("smoi_oidc_flow")
	setCookie(w, "smoi_oidc_flow", "", "/api/auth/callback", -1, a.cookieSecure)

	state := r.URL.Query().Get("state")
	if cookieErr != nil || state == "" ||
		subtle.ConstantTimeCompare([]byte(flowCookie.Value), []byte(state)) != 1 {
		http.Error(w, "invalid login state", http.StatusBadRequest)
		return
	}

	nonce, verifier, err := a.consumeFlow(r.Context(), state)
	if err != nil {
		http.Error(w, "expired login state", http.StatusBadRequest)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "identity provider did not return an authorization code", http.StatusBadRequest)
		return
	}

	identity, err := a.verifyIdentity(r.Context(), code, verifier, nonce)
	if err != nil {
		http.Error(w, "OIDC verification failed", http.StatusUnauthorized)
		return
	}

	userID, err := a.provision(
		r.Context(), identity.issuer, identity.subject,
		identity.username, identity.email, identity.displayName, identity.avatar,
	)
	if err != nil {
		http.Error(w, "user provisioning failed", http.StatusInternalServerError)
		return
	}

	exchange, err := a.createLoginExchange(r.Context(), userID)
	if err != nil {
		http.Error(w, "session creation failed", http.StatusInternalServerError)
		return
	}

	setCookie(w, "smoi_login_exchange", exchange, "/api/auth/exchange", 60, a.cookieSecure)
	http.Redirect(w, r, "/?login=1", http.StatusSeeOther)
}

func (a *Auth) consumeFlow(ctx context.Context, state string) (string, string, error) {
	transaction, err := a.db.BeginTx(ctx, nil)
	if err != nil {
		return "", "", err
	}
	defer transaction.Rollback()

	var nonce, verifier string
	err = transaction.QueryRowContext(ctx, `
        SELECT nonce, verifier FROM oidc_flows
        WHERE state_hash = ? AND expires_at > ?
    `, hash(state), time.Now().Unix()).Scan(&nonce, &verifier)
	if err != nil {
		return "", "", err
	}

	if _, err := transaction.ExecContext(ctx, `
        DELETE FROM oidc_flows WHERE state_hash = ?
    `, hash(state)); err != nil {
		return "", "", err
	}

	return nonce, verifier, transaction.Commit()
}

func (a *Auth) verifyIdentity(
	ctx context.Context, code, verifier, expectedNonce string,
) (identityClaims, error) {
	var identity identityClaims

	oauthToken, err := a.oauth.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return identity, err
	}

	rawIDToken, ok := oauthToken.Extra("id_token").(string)
	if !ok {
		return identity, errors.New("missing ID token")
	}

	idToken, err := a.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return identity, err
	}

	claims := make(map[string]any)
	if err := idToken.Claims(&claims); err != nil {
		return identity, err
	}
	if claims["nonce"] != expectedNonce {
		return identity, errors.New("OIDC nonce mismatch")
	}

	avatar := avatarFromClaims(claims)
	info, err := a.provider.UserInfo(ctx, oauth2.StaticTokenSource(oauthToken))
	if err == nil {
		additional := make(map[string]any)
		if info.Claims(&additional) == nil {
			for key, value := range additional {
				if key != "sub" {
					claims[key] = value
				}
			}
			if userInfoAvatar := avatarFromClaims(additional); userInfoAvatar != "" {
				avatar = userInfoAvatar
			}
		}
	}

	identity = identityClaims{
		issuer:      idToken.Issuer,
		subject:     idToken.Subject,
		username:    firstNonEmptyClaim(claims, "preferred_username", "nickname", "email"),
		email:       firstNonEmptyClaim(claims, "email"),
		displayName: firstNonEmptyClaim(claims, "name", "given_name", "preferred_username"),
		avatar:      avatar,
	}
	if identity.username == "" {
		identity.username = idToken.Subject
	}
	if identity.displayName == "" {
		identity.displayName = identity.username
	}

	return identity, nil
}

func (a *Auth) createLoginExchange(ctx context.Context, userID string) (string, error) {
	token, err := randomString()
	if err != nil {
		return "", err
	}

	_, err = a.db.ExecContext(ctx, `
        INSERT INTO login_exchanges(token_hash, user_id, expires_at)
        VALUES (?, ?, ?)
    `, hash(token), userID, time.Now().Add(time.Minute).Unix())

	return token, err
}
