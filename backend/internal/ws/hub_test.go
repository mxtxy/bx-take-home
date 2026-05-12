package ws

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mxtxy/bx-take-home/backend/internal/auth"
	"github.com/mxtxy/bx-take-home/backend/internal/domain"
	"github.com/mxtxy/bx-take-home/backend/internal/scheduling"
	"github.com/mxtxy/bx-take-home/backend/internal/testutil"
)

func TestWebSocketHub_RegisterAndSendToUser(t *testing.T) {
	hub := NewHub()
	events, cleanup := hub.RegisterTestConnection(1, domain.RoleManager)
	defer cleanup()

	hub.SendToUser(1, ServerEvent{Type: "notification.created", NotificationID: int64Ptr(10)})

	assertReceiveEvent(t, events, "notification.created")
}

func TestWebSocketHub_SendToUser_DoesNotSendToOtherUsers(t *testing.T) {
	hub := NewHub()
	user1, cleanup1 := hub.RegisterTestConnection(1, domain.RoleManager)
	defer cleanup1()
	user2, cleanup2 := hub.RegisterTestConnection(2, domain.RoleManager)
	defer cleanup2()

	hub.SendToUser(1, ServerEvent{Type: "jobs.changed", JobID: int64Ptr(20)})

	assertReceiveEvent(t, user1, "jobs.changed")
	assertNoEvent(t, user2)
}

func TestWebSocketHub_BroadcastToManagers(t *testing.T) {
	hub := NewHub()
	manager1, cleanup1 := hub.RegisterTestConnection(1, domain.RoleManager)
	defer cleanup1()
	manager2, cleanup2 := hub.RegisterTestConnection(2, domain.RoleManager)
	defer cleanup2()
	technician1, cleanup3 := hub.RegisterTestConnection(3, domain.RoleTechnician)
	defer cleanup3()

	hub.BroadcastToRole(domain.RoleManager, ServerEvent{Type: "quotes.changed", QuoteID: int64Ptr(1)})

	assertReceiveEvent(t, manager1, "quotes.changed")
	assertReceiveEvent(t, manager2, "quotes.changed")
	assertNoEvent(t, technician1)
}

func TestWebSocketHub_BroadcastToRoleScopesByOrganization(t *testing.T) {
	hub := NewHub()
	manager1, cleanup1 := hub.RegisterTestConnection(1, domain.RoleManager, 1)
	defer cleanup1()
	manager3, cleanup2 := hub.RegisterTestConnection(5, domain.RoleManager, 2)
	defer cleanup2()

	hub.Publish([]domain.DomainEvent{{
		Type:           "quotes.changed",
		TargetRole:     rolePtr(domain.RoleManager),
		OrganizationID: 1,
		QuoteID:        int64Ptr(1),
	}})

	assertReceiveEvent(t, manager1, "quotes.changed")
	assertNoEvent(t, manager3)
}

func TestWebSocketHub_PublishIgnoresUnknownEvents(t *testing.T) {
	hub := NewHub()
	events, cleanup := hub.RegisterTestConnection(1, domain.RoleManager)
	defer cleanup()

	hub.Publish([]domain.DomainEvent{{Type: "unknown", TargetUserID: int64Ptr(1)}})

	assertNoEvent(t, events)
}

func TestSendAllDropsEventsWhenSubscriberBufferIsFull(t *testing.T) {
	sub := &subscriber{ch: make(chan ServerEvent, 1)}
	sub.ch <- ServerEvent{Type: "first"}

	sendAll([]*subscriber{sub}, ServerEvent{Type: "second"})

	if len(sub.ch) != 1 {
		t.Fatalf("buffer len = %d", len(sub.ch))
	}
}

func TestWebSocketEndpoint_RejectsUnauthenticatedConnection(t *testing.T) {
	handler, _ := newWebSocketTestHandler(t)
	server := httptest.NewServer(http.HandlerFunc(handler.ServeHTTP))
	defer server.Close()

	_, resp, err := websocket.DefaultDialer.Dial(wsURL(server.URL), nil)
	if err == nil {
		t.Fatal("expected unauthenticated dial to fail")
	}
	if resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %#v, err = %v", resp, err)
	}
}

func TestWebSocketEndpoint_AllowsAuthenticatedTechnician(t *testing.T) {
	handler, authService := newWebSocketTestHandler(t)
	server := httptest.NewServer(http.HandlerFunc(handler.ServeHTTP))
	defer server.Close()
	token, err := authService.SignToken(3, domain.RoleTechnician, testutil.FixedTime().Add(time.Hour))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	headers := http.Header{}
	headers.Add("Cookie", "brix_session="+token)

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), headers)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	_ = conn.Close()
}

func TestWebSocketEndpoint_RejectsDisallowedOrigin(t *testing.T) {
	handler, authService := newWebSocketTestHandler(t)
	server := httptest.NewServer(http.HandlerFunc(handler.ServeHTTP))
	defer server.Close()
	token, err := authService.SignToken(3, domain.RoleTechnician, testutil.FixedTime().Add(time.Hour))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	headers := http.Header{}
	headers.Add("Cookie", "brix_session="+token)
	headers.Add("Origin", "http://evil.test")

	_, resp, err := websocket.DefaultDialer.Dial(wsURL(server.URL), headers)
	if err == nil {
		t.Fatal("expected disallowed origin dial to fail")
	}
	if resp == nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %#v, err = %v", resp, err)
	}
}

func TestWebSocketEndpoint_AllowsConfiguredOrigin(t *testing.T) {
	handler, authService := newWebSocketTestHandler(t)
	server := httptest.NewServer(http.HandlerFunc(handler.ServeHTTP))
	defer server.Close()
	token, err := authService.SignToken(3, domain.RoleTechnician, testutil.FixedTime().Add(time.Hour))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	headers := http.Header{}
	headers.Add("Cookie", "brix_session="+token)
	headers.Add("Origin", "http://localhost:3000")

	conn, _, err := websocket.DefaultDialer.Dial(wsURL(server.URL), headers)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	_ = conn.Close()
}

func TestWebSocketEndpoint_DefaultCookieName(t *testing.T) {
	database := testutil.PrepareDB(t)
	authService := auth.NewService(database, auth.Options{Secret: "test-secret", Now: testutil.FixedTime})
	handler := NewHandler(authService, NewHub(), "")
	server := httptest.NewServer(http.HandlerFunc(handler.ServeHTTP))
	defer server.Close()
	token, _ := authService.SignToken(3, domain.RoleTechnician, testutil.FixedTime().Add(time.Hour))

	conn := dialWithToken(t, server.URL, token)
	_ = conn.Close()
}

func TestWebSocketEndpoint_RejectsInvalidToken(t *testing.T) {
	handler, _ := newWebSocketTestHandler(t)
	request := httptest.NewRequest(http.MethodGet, "/ws", nil)
	request.AddCookie(&http.Cookie{Name: "brix_session", Value: "bad-token"})
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func TestWebSocketEndpoint_UnregistersWhenUpgradeFails(t *testing.T) {
	handler, authService := newWebSocketTestHandler(t)
	token, _ := authService.SignToken(3, domain.RoleTechnician, testutil.FixedTime().Add(time.Hour))
	request := httptest.NewRequest(http.MethodGet, "/ws", nil)
	request.AddCookie(&http.Cookie{Name: "brix_session", Value: token})
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", recorder.Code)
	}
	assertSubscriberCount(t, handler.hub, 0)
}

func TestWebSocketEndpoint_UnregistersClientOnDisconnect(t *testing.T) {
	database := testutil.PrepareDB(t)
	authService := auth.NewService(database, auth.Options{Secret: "test-secret", Now: testutil.FixedTime})
	hub := NewHub()
	handler := NewHandler(authService, hub, "brix_session")
	server := httptest.NewServer(http.HandlerFunc(handler.ServeHTTP))
	t.Cleanup(func() {
		_ = server.Listener.Close()
		server.CloseClientConnections()
	})
	token, _ := authService.SignToken(3, domain.RoleTechnician, testutil.FixedTime().Add(time.Hour))
	conn := dialWithToken(t, server.URL, token)

	assertSubscriberCount(t, hub, 1)
	if err := conn.Close(); err != nil {
		t.Fatalf("close websocket: %v", err)
	}
	assertSubscriberCount(t, hub, 0)
}

func TestWebSocketEndpoint_AssignmentSendsNotificationToTechnician(t *testing.T) {
	database := testutil.PrepareDB(t)
	authService := auth.NewService(database, auth.Options{Secret: "test-secret", Now: testutil.FixedTime})
	hub := NewHub()
	handler := NewHandler(authService, hub, "brix_session")
	server := httptest.NewServer(http.HandlerFunc(handler.ServeHTTP))
	defer server.Close()
	token, _ := authService.SignToken(3, domain.RoleTechnician, testutil.FixedTime().Add(time.Hour))
	conn := dialWithToken(t, server.URL, token)
	defer conn.Close()

	service := scheduling.NewService(database, scheduling.Options{Clock: testutil.FixedTime})
	_, events, err := service.AssignJob(context.Background(), testutil.Manager1(), domain.AssignJobInput{
		QuoteID:      1,
		TechnicianID: 1,
		StartsAt:     testutil.MustTime(t, "2026-05-12T00:00:00Z"),
	})
	if err != nil {
		t.Fatalf("assign: %v", err)
	}
	hub.Publish(events)

	assertReadSocketEvent(t, conn, "notification.created")
	assertReadSocketEvent(t, conn, "jobs.changed")
}

func TestWebSocketEndpoint_QuoteChangeDoesNotReachOtherOrganizationManager(t *testing.T) {
	database := testutil.PrepareDB(t)
	testutil.InsertOtherOrganizationFixture(t, database)
	authService := auth.NewService(database, auth.Options{Secret: "test-secret", Now: testutil.FixedTime})
	hub := NewHub()
	handler := NewHandler(authService, hub, "brix_session")
	server := httptest.NewServer(http.HandlerFunc(handler.ServeHTTP))
	defer server.Close()
	token, _ := authService.SignToken(5, domain.RoleManager, testutil.FixedTime().Add(time.Hour))
	conn := dialWithToken(t, server.URL, token)
	defer conn.Close()

	service := scheduling.NewService(database, scheduling.Options{Clock: testutil.FixedTime})
	_, events, err := service.AssignJob(context.Background(), testutil.Manager1(), domain.AssignJobInput{
		QuoteID:      1,
		TechnicianID: 1,
		StartsAt:     testutil.MustTime(t, "2026-05-12T00:00:00Z"),
	})
	if err != nil {
		t.Fatalf("assign: %v", err)
	}
	hub.Publish(events)

	assertNoSocketEvent(t, conn)
}

func TestWebSocketEndpoint_CompletionSendsNotificationToManager(t *testing.T) {
	database := testutil.PrepareDB(t)
	authService := auth.NewService(database, auth.Options{Secret: "test-secret", Now: testutil.FixedTime})
	hub := NewHub()
	handler := NewHandler(authService, hub, "brix_session")
	server := httptest.NewServer(http.HandlerFunc(handler.ServeHTTP))
	defer server.Close()
	token, _ := authService.SignToken(1, domain.RoleManager, testutil.FixedTime().Add(time.Hour))
	conn := dialWithToken(t, server.URL, token)
	defer conn.Close()

	jobID := testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z")
	service := scheduling.NewService(database, scheduling.Options{Clock: testutil.FixedTime})
	_, events, err := service.CompleteJob(context.Background(), testutil.Technician1(), domain.CompleteJobInput{JobID: jobID})
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	hub.Publish(events)

	assertReadSocketEvent(t, conn, "notification.created")
	assertReadSocketEvent(t, conn, "jobs.changed")
}

func TestPongHandlerExtendsReadDeadline(t *testing.T) {
	conn := &fakeDeadlineSetter{}

	if err := pongHandler(conn)("pong"); err != nil {
		t.Fatalf("pong handler: %v", err)
	}
	if conn.calls != 1 {
		t.Fatalf("deadline calls = %d", conn.calls)
	}
}

func TestPongHandlerReturnsDeadlineError(t *testing.T) {
	conn := &fakeDeadlineSetter{err: errors.New("deadline failed")}

	if err := pongHandler(conn)("pong"); err == nil {
		t.Fatal("expected deadline error")
	}
}

func TestWriteSocketEventsWritesUntilDone(t *testing.T) {
	sub := &subscriber{ch: make(chan ServerEvent, 1), done: make(chan struct{})}
	writer := &fakeEventWriter{written: make(chan ServerEvent, 1)}
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		writeSocketEvents(writer, sub)
	}()

	sub.ch <- ServerEvent{Type: "jobs.changed", JobID: int64Ptr(1)}
	assertReceiveEvent(t, writer.written, "jobs.changed")
	close(sub.done)

	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("write loop did not exit")
	}
}

func TestWriteSocketEventsReturnsOnWriteError(t *testing.T) {
	sub := &subscriber{ch: make(chan ServerEvent, 1), done: make(chan struct{})}
	writer := &fakeEventWriter{written: make(chan ServerEvent, 1), err: errors.New("write failed")}
	sub.ch <- ServerEvent{Type: "jobs.changed", JobID: int64Ptr(1)}

	writeSocketEvents(writer, sub)

	if len(writer.written) != 1 {
		t.Fatalf("written count = %d", len(writer.written))
	}
}

func newWebSocketTestHandler(t *testing.T) (*Handler, *auth.Service) {
	t.Helper()
	database := testutil.PrepareDB(t)
	authService := auth.NewService(database, auth.Options{Secret: "test-secret", Now: testutil.FixedTime})
	return NewHandler(authService, NewHub(), "brix_session", "http://localhost:3000"), authService
}

func wsURL(httpURL string) string {
	return "ws" + strings.TrimPrefix(httpURL, "http")
}

func dialWithToken(t *testing.T, serverURL string, token string) *websocket.Conn {
	t.Helper()
	headers := http.Header{}
	headers.Add("Cookie", "brix_session="+token)
	conn, _, err := websocket.DefaultDialer.Dial(wsURL(serverURL), headers)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	return conn
}

func assertReceiveEvent(t *testing.T, events <-chan ServerEvent, eventType string) {
	t.Helper()
	select {
	case event := <-events:
		if event.Type != eventType {
			t.Fatalf("event type = %s, want %s", event.Type, eventType)
		}
	case <-time.After(time.Second):
		t.Fatalf("timeout waiting for %s", eventType)
	}
}

func assertNoEvent(t *testing.T, events <-chan ServerEvent) {
	t.Helper()
	select {
	case event := <-events:
		t.Fatalf("unexpected event: %#v", event)
	case <-time.After(50 * time.Millisecond):
	}
}

func assertSubscriberCount(t *testing.T, hub *Hub, want int) {
	t.Helper()
	deadline := time.Now().Add(500 * time.Millisecond)
	for {
		hub.mu.RLock()
		got := 0
		for _, subscribers := range hub.users {
			got += len(subscribers)
		}
		hub.mu.RUnlock()
		if got == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("subscriber count = %d, want %d", got, want)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func assertReadSocketEvent(t *testing.T, conn *websocket.Conn, eventType string) {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}
	var event ServerEvent
	if err := conn.ReadJSON(&event); err != nil {
		t.Fatalf("read event: %v", err)
	}
	if event.Type != eventType {
		t.Fatalf("event type = %s, want %s", event.Type, eventType)
	}
}

func assertNoSocketEvent(t *testing.T, conn *websocket.Conn) {
	t.Helper()
	if err := conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond)); err != nil {
		t.Fatalf("set deadline: %v", err)
	}
	var event ServerEvent
	err := conn.ReadJSON(&event)
	if err == nil {
		t.Fatalf("unexpected event: %#v", event)
	}
}

func rolePtr(value domain.Role) *domain.Role {
	return &value
}

type fakeDeadlineSetter struct {
	calls int
	err   error
}

func (f *fakeDeadlineSetter) SetReadDeadline(time.Time) error {
	f.calls++
	return f.err
}

type fakeEventWriter struct {
	written chan ServerEvent
	err     error
}

func (f *fakeEventWriter) WriteJSON(value any) error {
	f.written <- value.(ServerEvent)
	return f.err
}
