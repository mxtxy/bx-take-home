package httpapi

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mxtxy/bx-take-home/backend/internal/auth"
	"github.com/mxtxy/bx-take-home/backend/internal/domain"
	"github.com/mxtxy/bx-take-home/backend/internal/jobs"
	"github.com/mxtxy/bx-take-home/backend/internal/notifications"
	"github.com/mxtxy/bx-take-home/backend/internal/quotes"
	"github.com/mxtxy/bx-take-home/backend/internal/scheduling"
	"github.com/mxtxy/bx-take-home/backend/internal/technicians"
	"github.com/mxtxy/bx-take-home/backend/internal/testutil"
	"github.com/mxtxy/bx-take-home/backend/internal/ws"
)

func newTestAPI(t *testing.T) (*sql.DB, http.Handler) {
	t.Helper()
	database := testutil.PrepareDB(t)
	authService := auth.NewService(database, auth.Options{Secret: "test-secret", Now: testutil.FixedTime})
	hub := ws.NewHub()
	return database, NewRouter(Dependencies{
		Auth:          authService,
		Quotes:        quotes.NewService(database),
		Technicians:   technicians.NewService(database),
		Jobs:          jobs.NewQueryService(database),
		Scheduling:    scheduling.NewService(database, scheduling.Options{Clock: testutil.FixedTime}),
		Notifications: notifications.NewService(database, notifications.Options{Clock: testutil.FixedTime}),
		Hub:           hub,
		CookieName:    "brix_session",
		CookieSecure:  false,
	})
}

func TestAuthHandlers(t *testing.T) {
	_, handler := newTestAPI(t)

	manager := requestJSON(t, handler, http.MethodPost, "/api/auth/login", `{"email":"manager1@brix.test","password":"password123"}`, nil)
	if manager.Code != http.StatusOK || findCookie(manager, "brix_session") == nil {
		t.Fatalf("manager login status=%d cookies=%#v body=%s", manager.Code, manager.Result().Cookies(), manager.Body.String())
	}
	technician := requestJSON(t, handler, http.MethodPost, "/api/auth/login", `{"email":"technician1@brix.test","password":"password123"}`, nil)
	if technician.Code != http.StatusOK || findCookie(technician, "brix_session") == nil {
		t.Fatalf("technician login status=%d cookies=%#v", technician.Code, technician.Result().Cookies())
	}
	wrongPassword := requestJSON(t, handler, http.MethodPost, "/api/auth/login", `{"email":"manager1@brix.test","password":"wrong"}`, nil)
	assertErrorResponse(t, wrongPassword, http.StatusUnauthorized, domain.ErrorInvalidCredentials)
	malformed := requestJSON(t, handler, http.MethodPost, "/api/auth/login", `{`, nil)
	assertErrorResponse(t, malformed, http.StatusBadRequest, domain.ErrorInvalidInput)
	meMissing := requestJSON(t, handler, http.MethodGet, "/api/me", ``, nil)
	assertErrorResponse(t, meMissing, http.StatusUnauthorized, domain.ErrorUnauthorized)
	meValid := requestJSON(t, handler, http.MethodGet, "/api/me", ``, []*http.Cookie{findCookie(manager, "brix_session")})
	if meValid.Code != http.StatusOK || !bytes.Contains(meValid.Body.Bytes(), []byte("Sarah Manager")) {
		t.Fatalf("me response status=%d body=%s", meValid.Code, meValid.Body.String())
	}
	logout := requestJSON(t, handler, http.MethodPost, "/api/auth/logout", `{}`, []*http.Cookie{findCookie(manager, "brix_session")})
	if logout.Code != http.StatusOK || findCookie(logout, "brix_session").MaxAge >= 0 {
		t.Fatalf("logout status=%d cookies=%#v", logout.Code, logout.Result().Cookies())
	}
}

func TestQuoteHandlers(t *testing.T) {
	_, handler := newTestAPI(t)
	managerCookie := loginCookie(t, handler, "manager1@brix.test")
	technicianCookie := loginCookie(t, handler, "technician1@brix.test")

	manager := requestJSON(t, handler, http.MethodGet, "/api/quotes?status=unscheduled", ``, []*http.Cookie{managerCookie})
	if manager.Code != http.StatusOK || !bytes.Contains(manager.Body.Bytes(), []byte("Acme Plumbing")) {
		t.Fatalf("manager quotes status=%d body=%s", manager.Code, manager.Body.String())
	}
	technician := requestJSON(t, handler, http.MethodGet, "/api/quotes?status=unscheduled", ``, []*http.Cookie{technicianCookie})
	assertErrorResponse(t, technician, http.StatusForbidden, domain.ErrorForbidden)
	invalid := requestJSON(t, handler, http.MethodGet, "/api/quotes?status=invalid", ``, []*http.Cookie{managerCookie})
	assertErrorResponse(t, invalid, http.StatusBadRequest, domain.ErrorInvalidInput)
	all := requestJSON(t, handler, http.MethodGet, "/api/quotes", ``, []*http.Cookie{managerCookie})
	if all.Code != http.StatusOK || !bytes.Contains(all.Body.Bytes(), []byte("Harbor Cafe")) {
		t.Fatalf("all quotes status=%d body=%s", all.Code, all.Body.String())
	}
}

func TestTechnicianHandlers(t *testing.T) {
	_, handler := newTestAPI(t)
	managerCookie := loginCookie(t, handler, "manager1@brix.test")
	technicianCookie := loginCookie(t, handler, "technician1@brix.test")

	manager := requestJSON(t, handler, http.MethodGet, "/api/technicians", ``, []*http.Cookie{managerCookie})
	if manager.Code != http.StatusOK || !bytes.Contains(manager.Body.Bytes(), []byte("Priya Technician")) {
		t.Fatalf("technicians status=%d body=%s", manager.Code, manager.Body.String())
	}
	technician := requestJSON(t, handler, http.MethodGet, "/api/technicians", ``, []*http.Cookie{technicianCookie})
	assertErrorResponse(t, technician, http.StatusForbidden, domain.ErrorForbidden)
}

func TestJobHandlers(t *testing.T) {
	database, handler := newTestAPI(t)
	managerCookie := loginCookie(t, handler, "manager1@brix.test")
	manager2Cookie := loginCookie(t, handler, "manager2@brix.test")
	technicianCookie := loginCookie(t, handler, "technician1@brix.test")
	technician2Cookie := loginCookie(t, handler, "technician2@brix.test")
	job1 := testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T08:00:00Z")
	_ = testutil.InsertScheduledJob(t, database, 2, 2, 2, "2026-05-12T08:00:00Z")

	managerJobs := requestJSON(t, handler, http.MethodGet, "/api/jobs", ``, []*http.Cookie{managerCookie})
	if managerJobs.Code != http.StatusOK || !bytes.Contains(managerJobs.Body.Bytes(), []byte(`"id":`)) || bytes.Contains(managerJobs.Body.Bytes(), []byte("Northside Dental")) {
		t.Fatalf("manager jobs status=%d body=%s", managerJobs.Code, managerJobs.Body.String())
	}
	technicianJobs := requestJSON(t, handler, http.MethodGet, "/api/jobs", ``, []*http.Cookie{technicianCookie})
	if technicianJobs.Code != http.StatusOK || !bytes.Contains(technicianJobs.Body.Bytes(), []byte("Acme Plumbing")) || bytes.Contains(technicianJobs.Body.Bytes(), []byte("Northside Dental")) {
		t.Fatalf("technician jobs status=%d body=%s", technicianJobs.Code, technicianJobs.Body.String())
	}
	create := requestJSON(t, handler, http.MethodPost, "/api/jobs", `{"quoteId":3,"technicianId":1,"startsAt":"2026-05-12T12:00:00Z"}`, []*http.Cookie{managerCookie})
	if create.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", create.Code, create.Body.String())
	}
	techCreate := requestJSON(t, handler, http.MethodPost, "/api/jobs", `{"quoteId":4,"technicianId":1,"startsAt":"2026-05-12T14:00:00Z"}`, []*http.Cookie{technicianCookie})
	assertErrorResponse(t, techCreate, http.StatusForbidden, domain.ErrorForbidden)
	malformed := requestJSON(t, handler, http.MethodPost, "/api/jobs", `{`, []*http.Cookie{managerCookie})
	assertErrorResponse(t, malformed, http.StatusBadRequest, domain.ErrorInvalidInput)
	invalidTime := requestJSON(t, handler, http.MethodPost, "/api/jobs", `{"quoteId":4,"technicianId":1,"startsAt":"not-a-time"}`, []*http.Cookie{managerCookie})
	assertErrorResponse(t, invalidTime, http.StatusBadRequest, domain.ErrorInvalidInput)
	overlap := requestJSON(t, handler, http.MethodPost, "/api/jobs", `{"quoteId":4,"technicianId":1,"startsAt":"2026-05-12T09:00:00Z"}`, []*http.Cookie{managerCookie})
	assertErrorResponse(t, overlap, http.StatusConflict, domain.ErrorScheduleConflict)
	duplicate := requestJSON(t, handler, http.MethodPost, "/api/jobs", `{"quoteId":1,"technicianId":1,"startsAt":"2026-05-12T16:00:00Z"}`, []*http.Cookie{managerCookie})
	assertErrorResponse(t, duplicate, http.StatusConflict, domain.ErrorQuoteAlreadyScheduled)
	reschedule := requestJSON(t, handler, http.MethodPatch, "/api/jobs/"+itoa(job1)+"/schedule", `{"technicianId":2,"startsAt":"2026-05-12T14:00:00Z"}`, []*http.Cookie{managerCookie})
	if reschedule.Code != http.StatusOK {
		t.Fatalf("reschedule status=%d body=%s", reschedule.Code, reschedule.Body.String())
	}
	otherManager := requestJSON(t, handler, http.MethodPatch, "/api/jobs/"+itoa(job1)+"/schedule", `{"technicianId":2,"startsAt":"2026-05-12T16:00:00Z"}`, []*http.Cookie{manager2Cookie})
	assertErrorResponse(t, otherManager, http.StatusForbidden, domain.ErrorForbidden)
	complete := requestJSON(t, handler, http.MethodPatch, "/api/jobs/"+itoa(job1)+"/complete", `{}`, []*http.Cookie{technician2Cookie})
	if complete.Code != http.StatusOK {
		t.Fatalf("complete status=%d body=%s", complete.Code, complete.Body.String())
	}
	completedReschedule := requestJSON(t, handler, http.MethodPatch, "/api/jobs/"+itoa(job1)+"/schedule", `{"technicianId":2,"startsAt":"2026-05-12T18:00:00Z"}`, []*http.Cookie{managerCookie})
	assertErrorResponse(t, completedReschedule, http.StatusConflict, domain.ErrorCompletedJobImmutable)
	wrongTech := requestJSON(t, handler, http.MethodPatch, "/api/jobs/"+itoa(job1)+"/complete", `{}`, []*http.Cookie{technicianCookie})
	assertErrorResponse(t, wrongTech, http.StatusForbidden, domain.ErrorForbidden)
	managerComplete := requestJSON(t, handler, http.MethodPatch, "/api/jobs/"+itoa(job1)+"/complete", `{}`, []*http.Cookie{managerCookie})
	assertErrorResponse(t, managerComplete, http.StatusForbidden, domain.ErrorForbidden)
	alreadyCompleted := requestJSON(t, handler, http.MethodPatch, "/api/jobs/"+itoa(job1)+"/complete", `{}`, []*http.Cookie{technician2Cookie})
	assertErrorResponse(t, alreadyCompleted, http.StatusConflict, domain.ErrorJobAlreadyCompleted)
}

func TestNotificationHandlers(t *testing.T) {
	database, handler := newTestAPI(t)
	managerCookie := loginCookie(t, handler, "manager1@brix.test")
	technicianCookie := loginCookie(t, handler, "technician1@brix.test")
	jobID := testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z")
	managerNotificationID := insertHTTPNotification(t, database, 1, jobID)
	technicianNotificationID := insertHTTPNotification(t, database, 3, jobID)

	list := requestJSON(t, handler, http.MethodGet, "/api/notifications", ``, []*http.Cookie{managerCookie})
	if list.Code != http.StatusOK || !bytes.Contains(list.Body.Bytes(), []byte(`"notifications"`)) || bytes.Contains(list.Body.Bytes(), []byte(`"id":`+itoa(technicianNotificationID))) {
		t.Fatalf("notifications status=%d body=%s", list.Code, list.Body.String())
	}
	read := requestJSON(t, handler, http.MethodPatch, "/api/notifications/"+itoa(managerNotificationID)+"/read", `{}`, []*http.Cookie{managerCookie})
	if read.Code != http.StatusOK {
		t.Fatalf("read status=%d body=%s", read.Code, read.Body.String())
	}
	wrongUser := requestJSON(t, handler, http.MethodPatch, "/api/notifications/"+itoa(managerNotificationID)+"/read", `{}`, []*http.Cookie{technicianCookie})
	assertErrorResponse(t, wrongUser, http.StatusForbidden, domain.ErrorForbidden)
	missing := requestJSON(t, handler, http.MethodPatch, "/api/notifications/999/read", `{}`, []*http.Cookie{managerCookie})
	assertErrorResponse(t, missing, http.StatusNotFound, domain.ErrorNotFound)
	invalid := requestJSON(t, handler, http.MethodPatch, "/api/notifications/0/read", `{}`, []*http.Cookie{managerCookie})
	assertErrorResponse(t, invalid, http.StatusBadRequest, domain.ErrorInvalidInput)
}

func TestHealthHandler(t *testing.T) {
	_, handler := newTestAPI(t)
	response := requestJSON(t, handler, http.MethodGet, "/healthz", ``, nil)
	if response.Code != http.StatusOK || !bytes.Contains(response.Body.Bytes(), []byte(`"ok":true`)) {
		t.Fatalf("health status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestRouterDefaultsAndCORSPreflight(t *testing.T) {
	database := testutil.PrepareDB(t)
	handler := NewRouter(Dependencies{
		Auth:          auth.NewService(database, auth.Options{Secret: "test-secret", Now: testutil.FixedTime}),
		Quotes:        quotes.NewService(database),
		Technicians:   technicians.NewService(database),
		Jobs:          jobs.NewQueryService(database),
		Scheduling:    scheduling.NewService(database, scheduling.Options{Clock: testutil.FixedTime}),
		Notifications: notifications.NewService(database, notifications.Options{Clock: testutil.FixedTime}),
		AllowedOrigin: "http://frontend.test",
	})
	request := httptest.NewRequest(http.MethodOptions, "/api/jobs", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d", recorder.Code)
	}
	if got := recorder.Header().Get("Access-Control-Allow-Origin"); got != "http://frontend.test" {
		t.Fatalf("origin = %q", got)
	}
}

func TestDecodeJSON_RejectsNonJSONContentType(t *testing.T) {
	_, handler := newTestAPI(t)
	request := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader([]byte(`{"email":"manager1@brix.test","password":"password123"}`)))
	request.Header.Set("Content-Type", "text/plain")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusBadRequest, domain.ErrorInvalidInput)
}

func TestLogout_RejectsNonJSONContentType(t *testing.T) {
	_, handler := newTestAPI(t)
	request := httptest.NewRequest(http.MethodPost, "/api/auth/logout", bytes.NewReader([]byte(`{}`)))
	request.Header.Set("Content-Type", "text/plain")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	assertErrorResponse(t, recorder, http.StatusBadRequest, domain.ErrorInvalidInput)
}

func TestAuthenticatedHandlers_RequireCookie(t *testing.T) {
	_, handler := newTestAPI(t)
	tests := []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodGet, "/api/quotes", ""},
		{http.MethodGet, "/api/technicians", ""},
		{http.MethodGet, "/api/jobs", ""},
		{http.MethodPost, "/api/jobs", `{"quoteId":1,"technicianId":1,"startsAt":"2026-05-12T10:00:00Z"}`},
		{http.MethodPatch, "/api/jobs/1/schedule", `{"technicianId":1,"startsAt":"2026-05-12T10:00:00Z"}`},
		{http.MethodPatch, "/api/jobs/1/complete", `{}`},
		{http.MethodGet, "/api/notifications", ""},
		{http.MethodPatch, "/api/notifications/1/read", `{}`},
	}
	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			response := requestJSON(t, handler, tt.method, tt.path, tt.body, nil)
			assertErrorResponse(t, response, http.StatusUnauthorized, domain.ErrorUnauthorized)
		})
	}
}

func TestActorRejectsInvalidCookie(t *testing.T) {
	_, handler := newTestAPI(t)
	response := requestJSON(t, handler, http.MethodGet, "/api/me", "", []*http.Cookie{{Name: "brix_session", Value: "bad-token"}})

	assertErrorResponse(t, response, http.StatusUnauthorized, domain.ErrorUnauthorized)
}

func TestJobHandlerInvalidPathAndBodyBranches(t *testing.T) {
	_, handler := newTestAPI(t)
	managerCookie := loginCookie(t, handler, "manager1@brix.test")
	technicianCookie := loginCookie(t, handler, "technician1@brix.test")

	invalidRescheduleID := requestJSON(t, handler, http.MethodPatch, "/api/jobs/not-a-number/schedule", `{"technicianId":1,"startsAt":"2026-05-12T10:00:00Z"}`, []*http.Cookie{managerCookie})
	assertErrorResponse(t, invalidRescheduleID, http.StatusBadRequest, domain.ErrorInvalidInput)
	malformedReschedule := requestJSON(t, handler, http.MethodPatch, "/api/jobs/1/schedule", `{`, []*http.Cookie{managerCookie})
	assertErrorResponse(t, malformedReschedule, http.StatusBadRequest, domain.ErrorInvalidInput)
	invalidRescheduleTime := requestJSON(t, handler, http.MethodPatch, "/api/jobs/1/schedule", `{"technicianId":1,"startsAt":"bad-time"}`, []*http.Cookie{managerCookie})
	assertErrorResponse(t, invalidRescheduleTime, http.StatusBadRequest, domain.ErrorInvalidInput)

	invalidCompleteID := requestJSON(t, handler, http.MethodPatch, "/api/jobs/not-a-number/complete", `{}`, []*http.Cookie{technicianCookie})
	assertErrorResponse(t, invalidCompleteID, http.StatusBadRequest, domain.ErrorInvalidInput)
}

func TestNotificationHandlerInvalidPathBranch(t *testing.T) {
	_, handler := newTestAPI(t)
	managerCookie := loginCookie(t, handler, "manager1@brix.test")

	response := requestJSON(t, handler, http.MethodPatch, "/api/notifications/not-a-number/read", `{}`, []*http.Cookie{managerCookie})
	assertErrorResponse(t, response, http.StatusBadRequest, domain.ErrorInvalidInput)
}

func TestListHandlersWriteServiceErrors(t *testing.T) {
	authService := newHTTPAuthService(t)
	tests := []struct {
		name    string
		path    string
		mockDep func(*testing.T, Dependencies) (Dependencies, func())
	}{
		{
			name: "quotes",
			path: "/api/quotes",
			mockDep: func(t *testing.T, deps Dependencies) (Dependencies, func()) {
				database, mock := newHTTPMockDB(t)
				mock.ExpectQuery("SELECT id, customer_name").WillReturnError(errors.New("query failed"))
				deps.Quotes = quotes.NewService(database)
				return deps, func() { assertHTTPMock(t, mock, database) }
			},
		},
		{
			name: "technicians",
			path: "/api/technicians",
			mockDep: func(t *testing.T, deps Dependencies) (Dependencies, func()) {
				database, mock := newHTTPMockDB(t)
				mock.ExpectQuery("SELECT t.id, u.id").WillReturnError(errors.New("query failed"))
				deps.Technicians = technicians.NewService(database)
				return deps, func() { assertHTTPMock(t, mock, database) }
			},
		},
		{
			name: "jobs",
			path: "/api/jobs",
			mockDep: func(t *testing.T, deps Dependencies) (Dependencies, func()) {
				database, mock := newHTTPMockDB(t)
				mock.ExpectQuery("SELECT").WillReturnError(errors.New("query failed"))
				deps.Jobs = jobs.NewQueryService(database)
				return deps, func() { assertHTTPMock(t, mock, database) }
			},
		},
		{
			name: "notifications",
			path: "/api/notifications",
			mockDep: func(t *testing.T, deps Dependencies) (Dependencies, func()) {
				database, mock := newHTTPMockDB(t)
				mock.ExpectQuery("SELECT id, type, message").WillReturnError(errors.New("query failed"))
				deps.Notifications = notifications.NewService(database, notifications.Options{})
				return deps, func() { assertHTTPMock(t, mock, database) }
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps, cleanup := tt.mockDep(t, Dependencies{Auth: authService, Hub: ws.NewHub(), CookieName: "brix_session"})
			defer cleanup()
			handler := NewRouter(deps)
			response := requestJSON(t, handler, http.MethodGet, tt.path, "", []*http.Cookie{authCookie(t, authService, 1, domain.RoleManager)})

			assertErrorResponse(t, response, http.StatusInternalServerError, domain.ErrorInternal)
		})
	}
}

func TestMarkNotificationReadWritesServiceError(t *testing.T) {
	authService := newHTTPAuthService(t)
	database, mock := newHTTPMockDB(t)
	defer assertHTTPMock(t, mock, database)
	mock.ExpectBegin().WillReturnError(errors.New("begin failed"))
	handler := NewRouter(Dependencies{
		Auth:          authService,
		Notifications: notifications.NewService(database, notifications.Options{}),
		Hub:           ws.NewHub(),
		CookieName:    "brix_session",
	})

	response := requestJSON(t, handler, http.MethodPatch, "/api/notifications/1/read", `{}`, []*http.Cookie{authCookie(t, authService, 1, domain.RoleManager)})

	assertErrorResponse(t, response, http.StatusInternalServerError, domain.ErrorInternal)
}

func TestStatusForCode_DefaultsToInternalServerError(t *testing.T) {
	if got := statusForCode(domain.ErrorCode("unknown")); got != http.StatusInternalServerError {
		t.Fatalf("status = %d", got)
	}
}

func loginCookie(t *testing.T, handler http.Handler, email string) *http.Cookie {
	t.Helper()
	response := requestJSON(t, handler, http.MethodPost, "/api/auth/login", `{"email":"`+email+`","password":"password123"}`, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("login %s: status=%d body=%s", email, response.Code, response.Body.String())
	}
	return findCookie(response, "brix_session")
}

func requestJSON(t *testing.T, handler http.Handler, method string, path string, body string, cookies []*http.Cookie) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body == "" {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader([]byte(body))
	}
	request := httptest.NewRequest(method, path, reader)
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	for _, cookie := range cookies {
		if cookie != nil {
			request.AddCookie(cookie)
		}
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func findCookie(response *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

func assertErrorResponse(t *testing.T, response *httptest.ResponseRecorder, status int, code domain.ErrorCode) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d, want %d, body=%s", response.Code, status, response.Body.String())
	}
	var parsed struct {
		Error struct {
			Code    domain.ErrorCode `json:"code"`
			Message string           `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("parse error body: %v body=%s", err, response.Body.String())
	}
	if parsed.Error.Code != code || parsed.Error.Message == "" {
		t.Fatalf("error body = %#v", parsed)
	}
}

func insertHTTPNotification(t *testing.T, database *sql.DB, recipientUserID int64, jobID int64) int64 {
	t.Helper()
	result, err := database.Exec(`
		INSERT INTO notifications (recipient_user_id, actor_user_id, job_id, type, message, created_at)
		VALUES (?, 1, ?, 'job_assigned', 'notification', '2026-05-12 10:00:00')
	`, recipientUserID, jobID)
	if err != nil {
		t.Fatalf("insert notification: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}
	return id
}

func itoa(value int64) string {
	return strconv.FormatInt(value, 10)
}

func newHTTPAuthService(t *testing.T) *auth.Service {
	t.Helper()
	database := testutil.PrepareDB(t)
	return auth.NewService(database, auth.Options{Secret: "test-secret", Now: testutil.FixedTime})
}

func authCookie(t *testing.T, authService *auth.Service, userID int64, role domain.Role) *http.Cookie {
	t.Helper()
	token, err := authService.SignToken(userID, role, testutil.FixedTime().Add(time.Hour))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return &http.Cookie{Name: "brix_session", Value: token}
}

func newHTTPMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	return database, mock
}

func assertHTTPMock(t *testing.T, mock sqlmock.Sqlmock, database *sql.DB) {
	t.Helper()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
	mock.ExpectClose()
	if err := database.Close(); err != nil {
		t.Fatalf("close mock db: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("close expectation: %v", err)
	}
}
