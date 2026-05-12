package auth

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/mxtxy/bx-take-home/backend/internal/domain"
	"github.com/mxtxy/bx-take-home/backend/internal/testutil"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	return NewService(testutil.PrepareDB(t), Options{
		Secret:     "test-secret",
		SessionTTL: 8 * time.Hour,
		Now:        testutil.FixedTime,
	})
}

func TestNewService_DefaultOptions(t *testing.T) {
	service := NewService(nil, Options{})

	if string(service.secret) != "dev_only_change_me" {
		t.Fatalf("secret = %q", string(service.secret))
	}
	if service.sessionTTL != 8*time.Hour {
		t.Fatalf("sessionTTL = %s", service.sessionTTL)
	}
	if service.now().Location() != time.UTC {
		t.Fatalf("now location = %v", service.now().Location())
	}
	if service.SessionTTL() != 8*time.Hour {
		t.Fatalf("SessionTTL() = %s", service.SessionTTL())
	}
}

func TestAuthService_Login_ManagerSuccess(t *testing.T) {
	service := newTestService(t)

	actor, token, err := service.Login(context.Background(), "manager1@brix.test", "password123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if actor.UserID != 1 || actor.Email != "manager1@brix.test" || actor.DisplayName != "Sarah Manager" || actor.Role != domain.RoleManager {
		t.Fatalf("actor = %#v", actor)
	}
	if actor.OrganizationID != 1 {
		t.Fatalf("organization id = %d", actor.OrganizationID)
	}
	if actor.ManagerID == nil || *actor.ManagerID != 1 || actor.TechnicianID != nil {
		t.Fatalf("domain identity = %#v", actor)
	}
	if token == "" {
		t.Fatal("token is empty")
	}
	parsed, err := service.ActorFromToken(context.Background(), token)
	if err != nil {
		t.Fatalf("actor from token: %v", err)
	}
	if parsed.UserID != actor.UserID || parsed.Role != actor.Role {
		t.Fatalf("parsed actor = %#v", parsed)
	}
}

func TestAuthService_Login_CreatesRevocableSession(t *testing.T) {
	database := testutil.PrepareDB(t)
	service := NewService(database, Options{
		Secret:     "test-secret",
		SessionTTL: 8 * time.Hour,
		Now:        testutil.FixedTime,
	})

	_, token, err := service.Login(context.Background(), "manager1@brix.test", "password123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if got := testutil.CountRows(t, database, `SELECT COUNT(*) FROM sessions WHERE user_id = 1 AND revoked_at IS NULL`); got != 1 {
		t.Fatalf("active sessions = %d", got)
	}

	if err := service.RevokeToken(context.Background(), token); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if got := testutil.CountRows(t, database, `SELECT COUNT(*) FROM sessions WHERE user_id = 1 AND revoked_at IS NOT NULL`); got != 1 {
		t.Fatalf("revoked sessions = %d", got)
	}
	_, err = service.ActorFromToken(context.Background(), token)
	if code := domain.CodeOf(err); code != domain.ErrorUnauthorized {
		t.Fatalf("code = %v, err = %v", code, err)
	}
}

func TestAuthService_Login_TechnicianSuccess(t *testing.T) {
	service := newTestService(t)

	actor, _, err := service.Login(context.Background(), "technician1@brix.test", "password123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if actor.UserID != 3 || actor.Role != domain.RoleTechnician {
		t.Fatalf("actor = %#v", actor)
	}
	if actor.OrganizationID != 1 {
		t.Fatalf("organization id = %d", actor.OrganizationID)
	}
	if actor.ManagerID != nil || actor.TechnicianID == nil || *actor.TechnicianID != 1 {
		t.Fatalf("domain identity = %#v", actor)
	}
}

func TestAuthService_Login_NormalizesEmail(t *testing.T) {
	service := newTestService(t)

	actor, _, err := service.Login(context.Background(), " Manager1@Brix.Test ", "password123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if actor.Email != "manager1@brix.test" {
		t.Fatalf("email = %q", actor.Email)
	}
}

func TestAuthService_Login_InvalidPassword(t *testing.T) {
	service := newTestService(t)

	_, _, err := service.Login(context.Background(), "manager1@brix.test", "wrong")
	if code := domain.CodeOf(err); code != domain.ErrorInvalidCredentials {
		t.Fatalf("code = %v, err = %v", code, err)
	}
}

func TestAuthService_Login_UnknownEmail(t *testing.T) {
	service := newTestService(t)

	_, _, err := service.Login(context.Background(), "missing@brix.test", "password123")
	if code := domain.CodeOf(err); code != domain.ErrorInvalidCredentials {
		t.Fatalf("code = %v, err = %v", code, err)
	}
}

func TestAuthService_Login_ReturnsDatabaseError(t *testing.T) {
	database := testutil.PrepareDB(t)
	service := NewService(database, Options{Secret: "test-secret", Now: testutil.FixedTime})
	if err := database.Close(); err != nil {
		t.Fatalf("close db: %v", err)
	}

	_, _, err := service.Login(context.Background(), "manager1@brix.test", "password123")
	if err == nil || domain.CodeOf(err) != domain.ErrorInternal {
		t.Fatalf("expected raw database error, got code=%v err=%v", domain.CodeOf(err), err)
	}
}

func TestAuthService_Login_ReturnsSessionCreateError(t *testing.T) {
	database := testutil.PrepareDB(t)
	service := NewService(database, Options{Secret: "test-secret", Now: testutil.FixedTime})
	if _, err := database.Exec(`DROP TABLE sessions`); err != nil {
		t.Fatalf("drop sessions: %v", err)
	}

	_, _, err := service.Login(context.Background(), "manager1@brix.test", "password123")
	if err == nil {
		t.Fatal("expected session create error")
	}
}

func TestAuthService_ActorFromToken_RejectsTamperedToken(t *testing.T) {
	service := newTestService(t)
	_, token, err := service.Login(context.Background(), "manager1@brix.test", "password123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	tampered := token[:len(token)-1] + "x"

	_, err = service.ActorFromToken(context.Background(), tampered)
	if code := domain.CodeOf(err); code != domain.ErrorUnauthorized {
		t.Fatalf("code = %v, err = %v", code, err)
	}
}

func TestAuthService_ActorFromToken_RejectsExpiredToken(t *testing.T) {
	service := newTestService(t)
	token, err := service.SignToken(1, domain.RoleManager, testutil.FixedTime().Add(-time.Minute))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	_, err = service.ActorFromToken(context.Background(), token)
	if code := domain.CodeOf(err); code != domain.ErrorUnauthorized {
		t.Fatalf("code = %v, err = %v", code, err)
	}
}

func TestAuthService_ActorFromToken_RejectsMissingUser(t *testing.T) {
	service := newTestService(t)
	token, err := service.SignToken(999, domain.RoleManager, testutil.FixedTime().Add(time.Hour))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	_, err = service.ActorFromToken(context.Background(), token)
	if code := domain.CodeOf(err); code != domain.ErrorUnauthorized {
		t.Fatalf("code = %v, err = %v", code, err)
	}
}

func TestAuthService_ActorFromToken_RejectsRoleMismatch(t *testing.T) {
	service := newTestService(t)
	token, err := service.SignToken(1, domain.RoleTechnician, testutil.FixedTime().Add(time.Hour))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	_, err = service.ActorFromToken(context.Background(), token)
	if code := domain.CodeOf(err); code != domain.ErrorUnauthorized {
		t.Fatalf("code = %v, err = %v", code, err)
	}
}

func TestAuthService_ActorFromToken_RejectsOrganizationMismatch(t *testing.T) {
	service := newTestService(t)
	token := service.signToken(tokenPayload{
		UserID:         1,
		OrganizationID: 2,
		Role:           domain.RoleManager,
		Exp:            testutil.FixedTime().Add(time.Hour).Unix(),
	})

	_, err := service.ActorFromToken(context.Background(), token)
	if code := domain.CodeOf(err); code != domain.ErrorUnauthorized {
		t.Fatalf("code = %v, err = %v", code, err)
	}
}

func TestAuthService_ActorFromToken_RejectsMissingSession(t *testing.T) {
	database := testutil.PrepareDB(t)
	service := NewService(database, Options{Secret: "test-secret", SessionTTL: 8 * time.Hour, Now: testutil.FixedTime})
	_, token, err := service.Login(context.Background(), "manager1@brix.test", "password123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if _, err := database.Exec(`DROP TABLE sessions`); err != nil {
		t.Fatalf("drop sessions: %v", err)
	}

	_, err = service.ActorFromToken(context.Background(), token)
	if code := domain.CodeOf(err); code != domain.ErrorUnauthorized {
		t.Fatalf("code = %v, err = %v", code, err)
	}
}

func TestAuthService_RevokeToken_RejectsMalformedToken(t *testing.T) {
	service := newTestService(t)

	err := service.RevokeToken(context.Background(), "malformed")
	if code := domain.CodeOf(err); code != domain.ErrorUnauthorized {
		t.Fatalf("code = %v, err = %v", code, err)
	}
}

func TestAuthService_RevokeToken_IgnoresTokenWithoutSession(t *testing.T) {
	service := newTestService(t)
	token, err := service.SignToken(1, domain.RoleManager, testutil.FixedTime().Add(time.Hour))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	if err := service.RevokeToken(context.Background(), token); err != nil {
		t.Fatalf("revoke legacy token: %v", err)
	}
}

func TestParseToken_RejectsMalformedTokens(t *testing.T) {
	service := NewService(nil, Options{Secret: "test-secret"})
	tests := []struct {
		name  string
		token string
	}{
		{"missing signature", "payload"},
		{"bad base64 payload", "%%." + service.sign("%%")},
		{"bad json payload", base64.RawURLEncoding.EncodeToString([]byte("{")) + "." + service.sign(base64.RawURLEncoding.EncodeToString([]byte("{")))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := service.parseToken(tt.token); err == nil {
				t.Fatal("expected parse error")
			}
		})
	}
}
