package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"dcx-gateway/internal/cache"
	"dcx-gateway/internal/config"
	"dcx-gateway/internal/model/dto"
	"dcx-gateway/internal/model/entity"
	"dcx-gateway/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrAuthUnauthorized = errors.New("unauthorized")
	ErrAuthForbidden    = errors.New("forbidden")
	ErrAuthUnavailable  = errors.New("auth unavailable")
)

// dummyBcryptHash is only used to equalize login timing for unknown users.
// It must not correspond to any real account password.
const dummyBcryptHash = "$2a$10$BHgXxpcQGOCYRJLMI80O/OJLHXgaTOAPuiS.SvukHhBENmTj/XL5K"

type AuthPrincipal struct {
	UserID   uint64
	Username string
	Role     string
	AuthType string
}

type AuthService interface {
	Login(ctx context.Context, username string, password string) (*dto.LoginResponse, error)
	Logout(ctx context.Context, token string) error
	ValidateBearerToken(ctx context.Context, token string) (*AuthPrincipal, error)
}

type authService struct {
	users      repository.UserRepository
	sessions   cache.AuthCache
	sessionTTL time.Duration
	expiresIn  int
}

func NewAuthService(users repository.UserRepository, sessions cache.AuthCache, cfg config.SecurityConfig) AuthService {
	seconds := cfg.SessionTTLSeconds
	if seconds <= 0 {
		seconds = config.DefaultSessionTTLSeconds
	}
	return &authService{
		users:      users,
		sessions:   sessions,
		sessionTTL: time.Duration(seconds) * time.Second,
		expiresIn:  seconds,
	}
}

func (s *authService) Login(ctx context.Context, username string, password string) (*dto.LoginResponse, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil, ErrAuthUnauthorized
	}
	user, err := s.users.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			_ = bcrypt.CompareHashAndPassword([]byte(dummyBcryptHash), []byte(password))
			return nil, ErrAuthUnauthorized
		}
		return nil, err
	}
	if !user.Enabled {
		return nil, ErrAuthForbidden
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrAuthUnauthorized
	}
	token, err := generateAccessToken()
	if err != nil {
		return nil, err
	}
	session := cache.AuthSession{
		UserID:    user.ID,
		Username:  user.Username,
		Role:      user.Role,
		ExpiresAt: time.Now().UTC().Add(s.sessionTTL),
	}
	if err := s.sessions.SetSession(ctx, TokenHash(token), session, s.sessionTTL); err != nil {
		return nil, err
	}
	return &dto.LoginResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   s.expiresIn,
		User: dto.AuthUser{
			Username: user.Username,
			Role:     user.Role,
		},
	}, nil
}

func (s *authService) Logout(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil
	}
	return s.sessions.DeleteSession(ctx, TokenHash(token))
}

func (s *authService) ValidateBearerToken(ctx context.Context, token string) (*AuthPrincipal, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrAuthUnauthorized
	}
	session, ok, err := s.sessions.GetSession(ctx, TokenHash(token))
	if err != nil {
		return nil, ErrAuthUnavailable
	}
	if !ok || session == nil {
		return nil, ErrAuthUnauthorized
	}
	if !session.ExpiresAt.IsZero() && time.Now().UTC().After(session.ExpiresAt) {
		_ = s.sessions.DeleteSession(ctx, TokenHash(token))
		return nil, ErrAuthUnauthorized
	}
	user, err := s.users.FindByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrAuthUnauthorized
		}
		return nil, ErrAuthUnavailable
	}
	if !user.Enabled {
		return nil, ErrAuthForbidden
	}
	return principalFromUser(user), nil
}

func TokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func principalFromUser(user *entity.UEUser) *AuthPrincipal {
	return &AuthPrincipal{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		AuthType: "bearer",
	}
}

func generateAccessToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}
