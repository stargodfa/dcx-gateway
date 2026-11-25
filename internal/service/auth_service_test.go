package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"dcx-gateway/internal/cache"
	"dcx-gateway/internal/config"
	"dcx-gateway/internal/model/entity"
	"dcx-gateway/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

func TestAuthServiceLoginSuccessWritesSession(t *testing.T) {
	hash := bcryptHash(t, "test-password")
	users := &fakeUserRepo{byUsername: map[string]entity.UEUser{
		"operator": {ID: 1, Username: "operator", PasswordHash: hash, Role: "operator", Enabled: true},
	}}
	sessions := newFakeAuthCache()
	svc := NewAuthService(users, sessions, config.SecurityConfig{SessionTTLSeconds: 7200})

	resp, err := svc.Login(context.Background(), "operator", "test-password")
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if resp.AccessToken == "" || resp.TokenType != "Bearer" || resp.ExpiresIn != 7200 {
		t.Fatalf("unexpected login response: %#v", resp)
	}
	tokenHash := TokenHash(resp.AccessToken)
	if _, ok := sessions.sessions[tokenHash]; !ok {
		t.Fatalf("session was not written using token hash")
	}
}

func TestAuthServiceLoginFailures(t *testing.T) {
	hash := bcryptHash(t, "test-password")
	users := &fakeUserRepo{byUsername: map[string]entity.UEUser{
		"operator": {ID: 1, Username: "operator", PasswordHash: hash, Role: "operator", Enabled: true},
		"disabled": {ID: 2, Username: "disabled", PasswordHash: hash, Role: "operator", Enabled: false},
	}}
	svc := NewAuthService(users, newFakeAuthCache(), config.SecurityConfig{SessionTTLSeconds: 7200})

	if _, err := svc.Login(context.Background(), "operator", "wrong"); !errors.Is(err, ErrAuthUnauthorized) {
		t.Fatalf("wrong password err = %v, want unauthorized", err)
	}
	if _, err := svc.Login(context.Background(), "missing", "test-password"); !errors.Is(err, ErrAuthUnauthorized) {
		t.Fatalf("missing user err = %v, want unauthorized", err)
	}
	if _, err := svc.Login(context.Background(), "disabled", "test-password"); !errors.Is(err, ErrAuthForbidden) {
		t.Fatalf("disabled user err = %v, want forbidden", err)
	}
}

func TestAuthServiceLoginRedisFailureReturnsError(t *testing.T) {
	hash := bcryptHash(t, "test-password")
	users := &fakeUserRepo{byUsername: map[string]entity.UEUser{
		"operator": {ID: 1, Username: "operator", PasswordHash: hash, Role: "operator", Enabled: true},
	}}
	sessions := newFakeAuthCache()
	sessions.setErr = errors.New("redis down")
	svc := NewAuthService(users, sessions, config.SecurityConfig{SessionTTLSeconds: 7200})

	if _, err := svc.Login(context.Background(), "operator", "test-password"); err == nil {
		t.Fatalf("Login err = nil, want redis error")
	}
}

func TestAuthServiceValidateLogoutAndMe(t *testing.T) {
	hash := bcryptHash(t, "test-password")
	users := &fakeUserRepo{byID: map[uint64]entity.UEUser{
		1: {ID: 1, Username: "operator", PasswordHash: hash, Role: "operator", Enabled: true},
	}}
	sessions := newFakeAuthCache()
	token := "token"
	sessions.sessions[TokenHash(token)] = cache.AuthSession{
		UserID:    1,
		Username:  "operator",
		Role:      "operator",
		ExpiresAt: time.Now().Add(time.Hour),
	}
	svc := NewAuthService(users, sessions, config.SecurityConfig{SessionTTLSeconds: 7200})

	principal, err := svc.ValidateBearerToken(context.Background(), token)
	if err != nil {
		t.Fatalf("ValidateBearerToken returned error: %v", err)
	}
	if principal.Username != "operator" || principal.Role != "operator" || principal.AuthType != "bearer" {
		t.Fatalf("unexpected principal: %#v", principal)
	}
	if err := svc.Logout(context.Background(), token); err != nil {
		t.Fatalf("Logout returned error: %v", err)
	}
	if _, ok := sessions.sessions[TokenHash(token)]; ok {
		t.Fatalf("session was not deleted")
	}
}

func bcryptHash(t *testing.T, password string) string {
	t.Helper()
	b, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword returned error: %v", err)
	}
	return string(b)
}

type fakeUserRepo struct {
	byUsername map[string]entity.UEUser
	byID       map[uint64]entity.UEUser
}

func (r *fakeUserRepo) FindByUsername(ctx context.Context, username string) (*entity.UEUser, error) {
	user, ok := r.byUsername[username]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &user, nil
}

func (r *fakeUserRepo) FindByID(ctx context.Context, id uint64) (*entity.UEUser, error) {
	user, ok := r.byID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return &user, nil
}

type fakeAuthCache struct {
	sessions  map[string]cache.AuthSession
	setErr    error
	getErr    error
	deleteErr error
}

func newFakeAuthCache() *fakeAuthCache {
	return &fakeAuthCache{sessions: make(map[string]cache.AuthSession)}
}

func (c *fakeAuthCache) SetSession(ctx context.Context, tokenHash string, session cache.AuthSession, ttl time.Duration) error {
	if c.setErr != nil {
		return c.setErr
	}
	c.sessions[tokenHash] = session
	return nil
}

func (c *fakeAuthCache) GetSession(ctx context.Context, tokenHash string) (*cache.AuthSession, bool, error) {
	if c.getErr != nil {
		return nil, false, c.getErr
	}
	session, ok := c.sessions[tokenHash]
	if !ok {
		return nil, false, nil
	}
	return &session, true, nil
}

func (c *fakeAuthCache) DeleteSession(ctx context.Context, tokenHash string) error {
	if c.deleteErr != nil {
		return c.deleteErr
	}
	delete(c.sessions, tokenHash)
	return nil
}
