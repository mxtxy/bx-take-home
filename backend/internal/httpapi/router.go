package httpapi

import (
	"encoding/json"
	"mime"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/mxtxy/bx-take-home/backend/internal/auth"
	"github.com/mxtxy/bx-take-home/backend/internal/domain"
	"github.com/mxtxy/bx-take-home/backend/internal/jobs"
	"github.com/mxtxy/bx-take-home/backend/internal/notifications"
	"github.com/mxtxy/bx-take-home/backend/internal/quotes"
	"github.com/mxtxy/bx-take-home/backend/internal/scheduling"
	"github.com/mxtxy/bx-take-home/backend/internal/technicians"
	"github.com/mxtxy/bx-take-home/backend/internal/ws"
)

type Dependencies struct {
	Auth          *auth.Service
	Quotes        *quotes.Service
	Technicians   *technicians.Service
	Jobs          *jobs.QueryService
	Scheduling    *scheduling.Service
	Notifications *notifications.Service
	Hub           *ws.Hub
	CookieName    string
	CookieSecure  bool
	AllowedOrigin string
}

type API struct {
	deps       Dependencies
	cookieName string
}

func NewRouter(deps Dependencies) http.Handler {
	if deps.CookieName == "" {
		deps.CookieName = "brix_session"
	}
	if deps.Hub == nil {
		deps.Hub = ws.NewHub()
	}
	api := &API{deps: deps, cookieName: deps.CookieName}
	r := chi.NewRouter()
	r.Use(api.cors)
	r.Get("/healthz", api.health)
	r.Post("/api/auth/login", api.login)
	r.Post("/api/auth/logout", api.logout)
	r.Get("/api/me", api.me)
	r.Get("/api/quotes", api.listQuotes)
	r.Get("/api/technicians", api.listTechnicians)
	r.Get("/api/jobs", api.listJobs)
	r.Post("/api/jobs", api.assignJob)
	r.Patch("/api/jobs/{id}/schedule", api.rescheduleJob)
	r.Patch("/api/jobs/{id}/complete", api.completeJob)
	r.Get("/api/notifications", api.listNotifications)
	r.Patch("/api/notifications/{id}/read", api.markNotificationRead)
	r.Get("/ws", ws.NewHandler(deps.Auth, deps.Hub, deps.CookieName, deps.AllowedOrigin).ServeHTTP)
	return r
}

func (api *API) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := api.deps.AllowedOrigin
		if origin == "" {
			origin = "http://localhost:3000"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (api *API) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (api *API) login(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	actor, token, err := api.deps.Auth.Login(r.Context(), request.Email, request.Password)
	if err != nil {
		writeError(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     api.cookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int((8 * time.Hour).Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   api.deps.CookieSecure,
	})
	writeJSON(w, http.StatusOK, map[string]domain.Actor{"user": actor})
}

func (api *API) logout(w http.ResponseWriter, r *http.Request) {
	var request struct{}
	if !decodeJSON(w, r, &request) {
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     api.cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   api.deps.CookieSecure,
	})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (api *API) me(w http.ResponseWriter, r *http.Request) {
	actor, ok := api.actor(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]domain.Actor{"user": actor})
}

func (api *API) listQuotes(w http.ResponseWriter, r *http.Request) {
	actor, ok := api.actor(w, r)
	if !ok {
		return
	}
	var status *domain.QuoteStatus
	if raw := r.URL.Query().Get("status"); raw != "" {
		value := domain.QuoteStatus(raw)
		status = &value
	}
	quotes, err := api.deps.Quotes.ListQuotes(r.Context(), actor, status)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string][]domain.QuoteDTO{"quotes": quotes})
}

func (api *API) listTechnicians(w http.ResponseWriter, r *http.Request) {
	actor, ok := api.actor(w, r)
	if !ok {
		return
	}
	technicians, err := api.deps.Technicians.ListTechnicians(r.Context(), actor)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string][]domain.TechnicianDTO{"technicians": technicians})
}

func (api *API) listJobs(w http.ResponseWriter, r *http.Request) {
	actor, ok := api.actor(w, r)
	if !ok {
		return
	}
	jobs, err := api.deps.Jobs.ListJobs(r.Context(), actor)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string][]domain.JobDTO{"jobs": jobs})
}

func (api *API) assignJob(w http.ResponseWriter, r *http.Request) {
	actor, ok := api.actor(w, r)
	if !ok {
		return
	}
	var request struct {
		QuoteID      int64  `json:"quoteId"`
		TechnicianID int64  `json:"technicianId"`
		StartsAt     string `json:"startsAt"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	startsAt, err := parseStartsAt(request.StartsAt)
	if err != nil {
		writeError(w, err)
		return
	}
	job, events, err := api.deps.Scheduling.AssignJob(r.Context(), actor, domain.AssignJobInput{
		QuoteID:      request.QuoteID,
		TechnicianID: request.TechnicianID,
		StartsAt:     startsAt,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	api.deps.Hub.Publish(events)
	writeJSON(w, http.StatusCreated, map[string]domain.JobDTO{"job": job})
}

func (api *API) rescheduleJob(w http.ResponseWriter, r *http.Request) {
	actor, ok := api.actor(w, r)
	if !ok {
		return
	}
	jobID, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}
	var request struct {
		TechnicianID int64  `json:"technicianId"`
		StartsAt     string `json:"startsAt"`
	}
	if !decodeJSON(w, r, &request) {
		return
	}
	startsAt, err := parseStartsAt(request.StartsAt)
	if err != nil {
		writeError(w, err)
		return
	}
	job, events, err := api.deps.Scheduling.RescheduleJob(r.Context(), actor, domain.RescheduleJobInput{
		JobID:        jobID,
		TechnicianID: request.TechnicianID,
		StartsAt:     startsAt,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	api.deps.Hub.Publish(events)
	writeJSON(w, http.StatusOK, map[string]domain.JobDTO{"job": job})
}

func (api *API) completeJob(w http.ResponseWriter, r *http.Request) {
	actor, ok := api.actor(w, r)
	if !ok {
		return
	}
	jobID, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}
	job, events, err := api.deps.Scheduling.CompleteJob(r.Context(), actor, domain.CompleteJobInput{JobID: jobID})
	if err != nil {
		writeError(w, err)
		return
	}
	api.deps.Hub.Publish(events)
	writeJSON(w, http.StatusOK, map[string]domain.JobDTO{"job": job})
}

func (api *API) listNotifications(w http.ResponseWriter, r *http.Request) {
	actor, ok := api.actor(w, r)
	if !ok {
		return
	}
	notifications, err := api.deps.Notifications.ListNotifications(r.Context(), actor)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string][]domain.NotificationDTO{"notifications": notifications})
}

func (api *API) markNotificationRead(w http.ResponseWriter, r *http.Request) {
	actor, ok := api.actor(w, r)
	if !ok {
		return
	}
	notificationID, ok := parsePathID(w, r, "id")
	if !ok {
		return
	}
	notification, err := api.deps.Notifications.MarkRead(r.Context(), actor, notificationID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]domain.NotificationDTO{"notification": notification})
}

func (api *API) actor(w http.ResponseWriter, r *http.Request) (domain.Actor, bool) {
	cookie, err := r.Cookie(api.cookieName)
	if err != nil || cookie.Value == "" {
		writeError(w, domain.Errorf(domain.ErrorUnauthorized))
		return domain.Actor{}, false
	}
	actor, err := api.deps.Auth.ActorFromToken(r.Context(), cookie.Value)
	if err != nil {
		writeError(w, domain.Errorf(domain.ErrorUnauthorized))
		return domain.Actor{}, false
	}
	return actor, true
}

func parsePathID(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	value, err := strconv.ParseInt(chi.URLParam(r, name), 10, 64)
	if err != nil {
		writeError(w, domain.Errorf(domain.ErrorInvalidInput))
		return 0, false
	}
	return value, true
}

func parseStartsAt(value string) (time.Time, error) {
	startsAt, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, domain.Errorf(domain.ErrorInvalidInput)
	}
	return startsAt.UTC(), nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(w, domain.Errorf(domain.ErrorInvalidInput))
		return false
	}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(target); err != nil {
		writeError(w, domain.Errorf(domain.ErrorInvalidInput))
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, err error) {
	code := domain.CodeOf(err)
	status := statusForCode(code)
	writeJSON(w, status, map[string]map[string]string{
		"error": {
			"code":    string(code),
			"message": domain.ErrorMessage(code),
		},
	})
}

func statusForCode(code domain.ErrorCode) int {
	switch code {
	case domain.ErrorInvalidInput:
		return http.StatusBadRequest
	case domain.ErrorUnauthorized, domain.ErrorInvalidCredentials:
		return http.StatusUnauthorized
	case domain.ErrorForbidden:
		return http.StatusForbidden
	case domain.ErrorNotFound:
		return http.StatusNotFound
	case domain.ErrorQuoteAlreadyScheduled, domain.ErrorScheduleConflict, domain.ErrorCompletedJobImmutable, domain.ErrorJobAlreadyCompleted:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
