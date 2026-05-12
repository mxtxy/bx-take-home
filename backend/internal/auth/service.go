package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mxtxy/bx-take-home/backend/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type Options struct {
	Secret     string
	SessionTTL time.Duration
	Now        func() time.Time
}

type Service struct {
	db         *sql.DB
	secret     []byte
	sessionTTL time.Duration
	now        func() time.Time
}

type tokenPayload struct {
	UserID         int64       `json:"uid"`
	OrganizationID int64       `json:"org,omitempty"`
	Role           domain.Role `json:"role"`
	SessionID      string      `json:"sid,omitempty"`
	IssuedAt       int64       `json:"iat,omitempty"`
	Exp            int64       `json:"exp"`
}

func NewService(db *sql.DB, options Options) *Service {
	if options.Secret == "" {
		options.Secret = "dev_only_change_me"
	}
	if options.SessionTTL == 0 {
		options.SessionTTL = 8 * time.Hour
	}
	if options.Now == nil {
		options.Now = func() time.Time { return time.Now().UTC() }
	}
	return &Service{
		db:         db,
		secret:     []byte(options.Secret),
		sessionTTL: options.SessionTTL,
		now:        options.Now,
	}
}

func (s *Service) Login(ctx context.Context, email string, password string) (domain.Actor, string, error) {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	actor, passwordHash, err := s.actorByEmail(ctx, normalizedEmail)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.Actor{}, "", domain.Errorf(domain.ErrorInvalidCredentials)
		}
		return domain.Actor{}, "", err
	}
	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) != nil {
		return domain.Actor{}, "", domain.Errorf(domain.ErrorInvalidCredentials)
	}
	expiresAt := s.now().UTC().Add(s.sessionTTL)
	sessionID, err := s.createSession(ctx, actor.UserID, actor.OrganizationID, expiresAt)
	if err != nil {
		return domain.Actor{}, "", err
	}
	token := s.signToken(tokenPayload{
		UserID:         actor.UserID,
		OrganizationID: actor.OrganizationID,
		Role:           actor.Role,
		SessionID:      sessionID,
		IssuedAt:       s.now().UTC().Unix(),
		Exp:            expiresAt.UTC().Unix(),
	})
	return actor, token, nil
}

func (s *Service) ActorFromToken(ctx context.Context, token string) (domain.Actor, error) {
	payload, err := s.parseToken(token)
	if err != nil {
		return domain.Actor{}, domain.Errorf(domain.ErrorUnauthorized)
	}
	if !s.now().UTC().Before(time.Unix(payload.Exp, 0).UTC()) {
		return domain.Actor{}, domain.Errorf(domain.ErrorUnauthorized)
	}
	actor, err := s.actorByID(ctx, payload.UserID)
	if err != nil {
		return domain.Actor{}, domain.Errorf(domain.ErrorUnauthorized)
	}
	if actor.Role != payload.Role {
		return domain.Actor{}, domain.Errorf(domain.ErrorUnauthorized)
	}
	if payload.OrganizationID != 0 && actor.OrganizationID != payload.OrganizationID {
		return domain.Actor{}, domain.Errorf(domain.ErrorUnauthorized)
	}
	if payload.SessionID != "" {
		if err := s.validateSession(ctx, payload); err != nil {
			return domain.Actor{}, domain.Errorf(domain.ErrorUnauthorized)
		}
	}
	return actor, nil
}

func (s *Service) RevokeToken(ctx context.Context, token string) error {
	payload, err := s.parseToken(token)
	if err != nil {
		return domain.Errorf(domain.ErrorUnauthorized)
	}
	if payload.SessionID == "" {
		return nil
	}
	_, err = s.db.ExecContext(ctx, `
		UPDATE sessions
		SET revoked_at = ?
		WHERE id = ? AND user_id = ? AND revoked_at IS NULL
	`, s.now().UTC(), payload.SessionID, payload.UserID)
	return err
}

func (s *Service) SessionTTL() time.Duration {
	return s.sessionTTL
}

func (s *Service) SignToken(userID int64, role domain.Role, expiresAt time.Time) (string, error) {
	return s.signToken(tokenPayload{
		UserID: userID,
		Role:   role,
		Exp:    expiresAt.UTC().Unix(),
	}), nil
}

func (s *Service) signToken(payload tokenPayload) string {
	payloadBytes, _ := json.Marshal(payload)
	encodedPayload := base64.RawURLEncoding.EncodeToString(payloadBytes)
	signature := s.sign(encodedPayload)
	return encodedPayload + "." + signature
}

func (s *Service) parseToken(token string) (tokenPayload, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return tokenPayload{}, domain.Errorf(domain.ErrorUnauthorized)
	}
	expected := s.sign(parts[0])
	if subtle.ConstantTimeCompare([]byte(expected), []byte(parts[1])) != 1 {
		return tokenPayload{}, domain.Errorf(domain.ErrorUnauthorized)
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return tokenPayload{}, err
	}
	var payload tokenPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return tokenPayload{}, err
	}
	return payload, nil
}

func (s *Service) sign(payload string) string {
	mac := hmac.New(sha256.New, s.secret)
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s *Service) createSession(ctx context.Context, userID int64, organizationID int64, expiresAt time.Time) (string, error) {
	sessionID := newSessionID()
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (id, organization_id, user_id, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, sessionID, organizationID, userID, expiresAt.UTC(), s.now().UTC())
	if err != nil {
		return "", err
	}
	return sessionID, nil
}

func (s *Service) validateSession(ctx context.Context, payload tokenPayload) error {
	var userID int64
	var organizationID int64
	var expiresAt time.Time
	var revokedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT user_id, organization_id, expires_at, revoked_at
		FROM sessions
		WHERE id = ?
	`, payload.SessionID).Scan(&userID, &organizationID, &expiresAt, &revokedAt)
	if err != nil {
		return err
	}
	if userID != payload.UserID || organizationID != payload.OrganizationID || revokedAt.Valid || !s.now().UTC().Before(expiresAt.UTC()) {
		return domain.Errorf(domain.ErrorUnauthorized)
	}
	_, err = s.db.ExecContext(ctx, `UPDATE sessions SET last_seen_at = ? WHERE id = ?`, s.now().UTC(), payload.SessionID)
	return err
}

func newSessionID() string {
	var bytes [16]byte
	_, _ = rand.Read(bytes[:])
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16])
}

func (s *Service) actorByEmail(ctx context.Context, email string) (domain.Actor, string, error) {
	return s.scanActor(ctx, `
		SELECT u.id, u.organization_id, u.email, u.password_hash, u.display_name, u.role, m.id, t.id
		FROM users u
		LEFT JOIN managers m ON m.user_id = u.id
		LEFT JOIN technicians t ON t.user_id = u.id
		WHERE u.email = ?
	`, email)
}

func (s *Service) actorByID(ctx context.Context, userID int64) (domain.Actor, error) {
	actor, _, err := s.scanActor(ctx, `
		SELECT u.id, u.organization_id, u.email, u.password_hash, u.display_name, u.role, m.id, t.id
		FROM users u
		LEFT JOIN managers m ON m.user_id = u.id
		LEFT JOIN technicians t ON t.user_id = u.id
		WHERE u.id = ?
	`, userID)
	return actor, err
}

func (s *Service) scanActor(ctx context.Context, query string, arg any) (domain.Actor, string, error) {
	var actor domain.Actor
	var passwordHash string
	var role string
	var managerID sql.NullInt64
	var technicianID sql.NullInt64
	err := s.db.QueryRowContext(ctx, query, arg).Scan(
		&actor.UserID,
		&actor.OrganizationID,
		&actor.Email,
		&passwordHash,
		&actor.DisplayName,
		&role,
		&managerID,
		&technicianID,
	)
	if err != nil {
		return domain.Actor{}, "", err
	}
	actor.Role = domain.Role(role)
	if managerID.Valid {
		value := managerID.Int64
		actor.ManagerID = &value
	}
	if technicianID.Valid {
		value := technicianID.Int64
		actor.TechnicianID = &value
	}
	return actor, passwordHash, nil
}
