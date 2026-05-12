package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/json"
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
	UserID int64       `json:"uid"`
	Role   domain.Role `json:"role"`
	Exp    int64       `json:"exp"`
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
	token, _ := s.SignToken(actor.UserID, actor.Role, s.now().UTC().Add(s.sessionTTL))
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
	return actor, nil
}

func (s *Service) SignToken(userID int64, role domain.Role, expiresAt time.Time) (string, error) {
	payload, _ := json.Marshal(tokenPayload{
		UserID: userID,
		Role:   role,
		Exp:    expiresAt.UTC().Unix(),
	})
	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := s.sign(encodedPayload)
	return encodedPayload + "." + signature, nil
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

func (s *Service) actorByEmail(ctx context.Context, email string) (domain.Actor, string, error) {
	return s.scanActor(ctx, `
		SELECT u.id, u.email, u.password_hash, u.display_name, u.role, m.id, t.id
		FROM users u
		LEFT JOIN managers m ON m.user_id = u.id
		LEFT JOIN technicians t ON t.user_id = u.id
		WHERE u.email = ?
	`, email)
}

func (s *Service) actorByID(ctx context.Context, userID int64) (domain.Actor, error) {
	actor, _, err := s.scanActor(ctx, `
		SELECT u.id, u.email, u.password_hash, u.display_name, u.role, m.id, t.id
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
