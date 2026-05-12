package ws

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mxtxy/bx-take-home/backend/internal/domain"
)

type ServerEvent struct {
	Type           string `json:"type"`
	NotificationID *int64 `json:"notificationId,omitempty"`
	JobID          *int64 `json:"jobId,omitempty"`
	QuoteID        *int64 `json:"quoteId,omitempty"`
}

type subscriber struct {
	userID int64
	role   domain.Role
	ch     chan ServerEvent
	done   chan struct{}
}

type Hub struct {
	mu     sync.RWMutex
	users  map[int64]map[*subscriber]struct{}
	roles  map[domain.Role]map[*subscriber]struct{}
	closed bool
}

func NewHub() *Hub {
	return &Hub{
		users: map[int64]map[*subscriber]struct{}{},
		roles: map[domain.Role]map[*subscriber]struct{}{},
	}
}

func (h *Hub) RegisterTestConnection(userID int64, role domain.Role) (<-chan ServerEvent, func()) {
	sub := h.register(userID, role)
	return sub.ch, func() { h.unregister(sub) }
}

func (h *Hub) SendToUser(userID int64, event ServerEvent) {
	h.mu.RLock()
	var recipients []*subscriber
	for sub := range h.users[userID] {
		recipients = append(recipients, sub)
	}
	h.mu.RUnlock()
	sendAll(recipients, event)
}

func (h *Hub) BroadcastToRole(role domain.Role, event ServerEvent) {
	h.mu.RLock()
	var recipients []*subscriber
	for sub := range h.roles[role] {
		recipients = append(recipients, sub)
	}
	h.mu.RUnlock()
	sendAll(recipients, event)
}

func (h *Hub) Publish(events []domain.DomainEvent) {
	for _, event := range events {
		serverEvent := ServerEvent{Type: event.Type}
		switch event.Type {
		case "notification.created":
			serverEvent.NotificationID = event.NotificationID
			serverEvent.JobID = event.JobID
		case "jobs.changed":
			serverEvent.JobID = event.JobID
		case "quotes.changed":
			serverEvent.QuoteID = event.QuoteID
		default:
			continue
		}
		if event.TargetUserID != nil {
			h.SendToUser(*event.TargetUserID, serverEvent)
		}
		if event.TargetRole != nil {
			h.BroadcastToRole(*event.TargetRole, serverEvent)
		}
	}
}

func (h *Hub) register(userID int64, role domain.Role) *subscriber {
	sub := &subscriber{
		userID: userID,
		role:   role,
		ch:     make(chan ServerEvent, 16),
		done:   make(chan struct{}),
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.users[userID] == nil {
		h.users[userID] = map[*subscriber]struct{}{}
	}
	if h.roles[role] == nil {
		h.roles[role] = map[*subscriber]struct{}{}
	}
	h.users[userID][sub] = struct{}{}
	h.roles[role][sub] = struct{}{}
	return sub
}

func (h *Hub) unregister(sub *subscriber) {
	h.mu.Lock()
	if subscribers := h.users[sub.userID]; subscribers != nil {
		delete(subscribers, sub)
		if len(subscribers) == 0 {
			delete(h.users, sub.userID)
		}
	}
	if subscribers := h.roles[sub.role]; subscribers != nil {
		delete(subscribers, sub)
		if len(subscribers) == 0 {
			delete(h.roles, sub.role)
		}
	}
	h.mu.Unlock()
	select {
	case <-sub.done:
	default:
		close(sub.done)
	}
}

func sendAll(recipients []*subscriber, event ServerEvent) {
	for _, sub := range recipients {
		select {
		case sub.ch <- event:
		default:
		}
	}
}

type AuthService interface {
	ActorFromToken(ctx context.Context, token string) (domain.Actor, error)
}

type Handler struct {
	auth          AuthService
	hub           *Hub
	cookieName    string
	allowedOrigin string
	upgrader      websocket.Upgrader
}

func NewHandler(authService AuthService, hub *Hub, cookieName string, allowedOrigins ...string) *Handler {
	if cookieName == "" {
		cookieName = "brix_session"
	}
	allowedOrigin := "http://localhost:3000"
	if len(allowedOrigins) > 0 && allowedOrigins[0] != "" {
		allowedOrigin = allowedOrigins[0]
	}
	handler := &Handler{
		auth:          authService,
		hub:           hub,
		cookieName:    cookieName,
		allowedOrigin: allowedOrigin,
	}
	handler.upgrader = websocket.Upgrader{
		CheckOrigin: handler.checkOrigin,
	}
	return handler
}

func (h *Handler) checkOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	return origin == "" || origin == h.allowedOrigin
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(h.cookieName)
	if err != nil || cookie.Value == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	actor, err := h.auth.ActorFromToken(r.Context(), cookie.Value)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	sub := h.hub.register(actor.UserID, actor.Role)
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.hub.unregister(sub)
		return
	}
	defer conn.Close()
	defer h.hub.unregister(sub)

	conn.SetReadLimit(1024)
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(pongHandler(conn))

	writeDone := make(chan struct{})
	go func() {
		defer close(writeDone)
		writeSocketEvents(conn, sub)
	}()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
	h.hub.unregister(sub)
	<-writeDone
}

type deadlineSetter interface {
	SetReadDeadline(time.Time) error
}

func pongHandler(conn deadlineSetter) func(string) error {
	return func(string) error {
		return conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	}
}

type eventWriter interface {
	WriteJSON(any) error
}

func writeSocketEvents(writer eventWriter, sub *subscriber) {
	for {
		select {
		case event := <-sub.ch:
			if err := writer.WriteJSON(event); err != nil {
				return
			}
		case <-sub.done:
			return
		}
	}
}

func int64Ptr(value int64) *int64 {
	return &value
}
