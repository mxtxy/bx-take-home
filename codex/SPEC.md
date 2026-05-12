# Brix Service Scheduling & Notification System — Product, Data, and API Specification

## 0. Purpose

This document is the implementation contract for the take-home project.

The project is a small end-to-end service scheduling and notification system. Managers assign unscheduled quotes to technicians in fixed two-hour windows. The backend prevents double-booking. Assignment, update, and completion events create persisted notifications and publish WebSocket refresh events.

This specification is intentionally explicit. If implementation behavior is not described here, it is out of scope for the take-home.

---

## 1. Locked Technical Decisions

These choices are fixed for the project.

| Area | Decision |
|---|---|
| Backend language | Go |
| Backend HTTP framework | `net/http` with `github.com/go-chi/chi/v5` |
| WebSocket library | `github.com/gorilla/websocket` |
| Database | MySQL 8 |
| Database driver | `github.com/go-sql-driver/mysql` |
| Migrations | Plain SQL migrations executed by `backend/cmd/migrate` |
| Seed data | Plain SQL seed file executed by `backend/cmd/seed` |
| Authentication | Seeded users, bcrypt password hashes, HttpOnly signed session cookie |
| Commands | HTTP/JSON |
| Live updates | WebSocket event notifications only |
| Notifications | Persisted in MySQL |
| Frontend | Next.js App Router with TypeScript |
| UI | Material UI, using `@mui/material`, `@emotion/react`, `@emotion/styled` |
| Frontend package manager | npm |
| Local runtime | Docker Compose |
| Compose filename | `compose.yaml` |
| Frontend port | `3000` |
| Backend port | `8080` |
| Host MySQL port | `3307`, mapped to container port `3306` |
| In-Docker MySQL host | `mysql:3306` |

---

## 2. Explicit Non-Goals

Do not implement these items for the take-home.

1. User registration.
2. Password reset.
3. Refresh tokens.
4. Production-grade CSRF hardening.
5. Multi-tenant organization scoping.
6. Drag-and-drop calendar UI.
7. Technician availability rules.
8. Business-hours validation.
9. Travel-time calculation.
10. Email, SMS, or push delivery.
11. Distributed WebSocket fanout.
12. Redis, NATS, Kafka, or other broker infrastructure.
13. Kubernetes.
14. Production observability.
15. Next.js Server Actions for scheduling commands.
16. Scheduling commands over WebSocket.

---

## 3. Product Requirements

### 3.1 Core Requirements

1. The system contains managers, technicians, quotes, jobs, and notifications.
2. A manager can view unscheduled quotes.
3. A manager can assign an unscheduled quote to a technician.
4. A manager selects a two-hour window by selecting a start time.
5. The backend computes the end time as start time plus two hours.
6. The backend prevents overlapping jobs for the same technician.
7. A technician can view their assigned jobs.
8. A technician can mark their own scheduled job as completed.
9. A job has lifecycle states `scheduled` and `completed`.
10. Assignment creates a technician notification.
11. Reschedule or reassignment creates technician notification(s).
12. Completion creates a manager notification.
13. Notifications are persisted in MySQL.
14. WebSocket events notify connected clients that they should refetch affected data.

### 3.2 Optional Requirement Included in This Spec

The exercise includes job update notifications. To demonstrate that requirement explicitly, this project includes manager rescheduling/reassignment of scheduled jobs.

A manager can update:

1. The technician assigned to a scheduled job.
2. The start time of a scheduled job.

A completed job cannot be rescheduled or reassigned.

---

## 4. Actors and Permissions

### 4.1 Manager

A manager can:

1. Log in.
2. Log out.
3. View their own user profile.
4. View unscheduled quotes.
5. View technicians.
6. Assign an unscheduled quote to a technician.
7. View jobs that they created.
8. Reschedule or reassign jobs that they created, provided the job is still scheduled.
9. View their own notifications.
10. Mark their own notifications as read.

A manager cannot:

1. Complete jobs.
2. View or update jobs created by another manager.
3. Mark another user's notifications as read.

### 4.2 Technician

A technician can:

1. Log in.
2. Log out.
3. View their own user profile.
4. View jobs assigned to them.
5. Complete jobs assigned to them, provided the job is still scheduled.
6. View their own notifications.
7. Mark their own notifications as read.

A technician cannot:

1. View unscheduled quotes.
2. View the technician list.
3. Assign jobs.
4. Reschedule jobs.
5. Complete jobs assigned to another technician.
6. Mark another user's notifications as read.

---

## 5. Authentication Model

### 5.1 Tables

Authentication uses one `users` table. Domain identity uses separate `managers` and `technicians` tables.

Each seeded user has exactly one role:

- `manager`
- `technician`

A manager user has exactly one row in `managers`.

A technician user has exactly one row in `technicians`.

There is no registration endpoint.

### 5.2 Session Model

Successful login sets a cookie named `brix_session`.

Cookie attributes in local development:

| Attribute | Value |
|---|---|
| HttpOnly | `true` |
| SameSite | `Lax` |
| Secure | `false` |
| Path | `/` |
| MaxAge | 8 hours |

The session token is signed using HMAC with `JWT_SECRET`.

The token contains:

1. User ID.
2. Role.
3. Expiry timestamp.

The backend resolves a full actor from the token on each protected request.

### 5.3 Seeded Users

Use these exact seeded accounts.

| User ID | Domain ID | Role | Email | Password | Display Name |
|---:|---:|---|---|---|---|
| 1 | Manager ID 1 | manager | `manager1@brix.test` | `password123` | Sarah Manager |
| 2 | Manager ID 2 | manager | `manager2@brix.test` | `password123` | Alex Manager |
| 3 | Technician ID 1 | technician | `technician1@brix.test` | `password123` | Tom Technician |
| 4 | Technician ID 2 | technician | `technician2@brix.test` | `password123` | Priya Technician |

Passwords must be stored as bcrypt hashes in the seed data.

---

## 6. Time Rules

1. All backend storage is UTC.
2. All API timestamps are RFC3339 strings with timezone.
3. Example timestamp: `2026-05-12T00:00:00Z`.
4. MySQL stores timestamps in `DATETIME(6)` columns.
5. The frontend may display timestamps using browser local time.
6. The client sends only `startsAt` for assignment and rescheduling.
7. The backend computes `endsAt = startsAt + 2 hours`.
8. The backend rejects any client-provided `endsAt` field by ignoring it if decoding into strict types, or by using request structs that do not contain an `endsAt` field.
9. Past timestamps are allowed for this take-home.
10. Jobs can be scheduled at any time of day.
11. Business-hours validation is intentionally not implemented.
12. Start times must be minute-aligned.
13. Seconds and nanoseconds in `startsAt` must be zero.

Valid request timestamp:

```json
{
  "startsAt": "2026-05-12T10:30:00Z"
}
```

Invalid request timestamp:

```json
{
  "startsAt": "2026-05-12T10:30:45Z"
}
```

---

## 7. Seed Quotes

Use five unscheduled quotes.

| Quote ID | Customer Name | Description | Status |
|---:|---|---|---|
| 1 | Acme Plumbing | Replace leaking kitchen tap | unscheduled |
| 2 | Northside Dental | Repair reception lighting | unscheduled |
| 3 | Green Grocer | Service refrigeration unit | unscheduled |
| 4 | City Gym | Inspect hot water system | unscheduled |
| 5 | Harbor Cafe | Repair dishwasher drainage | unscheduled |

---

## 8. Database Specification

All tables use the InnoDB storage engine.

### 8.1 `users`

```sql
CREATE TABLE users (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  email VARCHAR(255) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  display_name VARCHAR(120) NOT NULL,
  role ENUM('manager', 'technician') NOT NULL,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
    ON UPDATE CURRENT_TIMESTAMP(6)
) ENGINE=InnoDB;
```

### 8.2 `managers`

```sql
CREATE TABLE managers (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL UNIQUE,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),

  CONSTRAINT fk_managers_user
    FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB;
```

### 8.3 `technicians`

```sql
CREATE TABLE technicians (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  user_id BIGINT NOT NULL UNIQUE,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),

  CONSTRAINT fk_technicians_user
    FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB;
```

### 8.4 `quotes`

```sql
CREATE TABLE quotes (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  customer_name VARCHAR(120) NOT NULL,
  description TEXT NOT NULL,
  status ENUM('unscheduled', 'scheduled') NOT NULL DEFAULT 'unscheduled',
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
    ON UPDATE CURRENT_TIMESTAMP(6)
) ENGINE=InnoDB;

CREATE INDEX idx_quotes_status_created
  ON quotes (status, created_at, id);
```

### 8.5 `jobs`

```sql
CREATE TABLE jobs (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  quote_id BIGINT NOT NULL UNIQUE,
  technician_id BIGINT NOT NULL,
  manager_id BIGINT NOT NULL,
  starts_at DATETIME(6) NOT NULL,
  ends_at DATETIME(6) NOT NULL,
  status ENUM('scheduled', 'completed') NOT NULL DEFAULT 'scheduled',
  completed_at DATETIME(6) NULL,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  updated_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6)
    ON UPDATE CURRENT_TIMESTAMP(6),

  CONSTRAINT fk_jobs_quote
    FOREIGN KEY (quote_id) REFERENCES quotes(id),

  CONSTRAINT fk_jobs_technician
    FOREIGN KEY (technician_id) REFERENCES technicians(id),

  CONSTRAINT fk_jobs_manager
    FOREIGN KEY (manager_id) REFERENCES managers(id),

  CONSTRAINT chk_job_window_positive
    CHECK (ends_at > starts_at),

  CONSTRAINT chk_job_window_two_hours
    CHECK (TIMESTAMPDIFF(MINUTE, starts_at, ends_at) = 120)
) ENGINE=InnoDB;

CREATE INDEX idx_jobs_technician_window
  ON jobs (technician_id, starts_at, ends_at);

CREATE INDEX idx_jobs_manager_status
  ON jobs (manager_id, status, starts_at, id);

CREATE INDEX idx_jobs_technician_status
  ON jobs (technician_id, status, starts_at, id);
```

### 8.6 `notifications`

```sql
CREATE TABLE notifications (
  id BIGINT PRIMARY KEY AUTO_INCREMENT,
  recipient_user_id BIGINT NOT NULL,
  actor_user_id BIGINT NULL,
  job_id BIGINT NULL,
  type ENUM(
    'job_assigned',
    'job_updated',
    'job_completed'
  ) NOT NULL,
  message VARCHAR(500) NOT NULL,
  read_at DATETIME(6) NULL,
  created_at DATETIME(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),

  CONSTRAINT fk_notifications_recipient_user
    FOREIGN KEY (recipient_user_id) REFERENCES users(id),

  CONSTRAINT fk_notifications_actor_user
    FOREIGN KEY (actor_user_id) REFERENCES users(id),

  CONSTRAINT fk_notifications_job
    FOREIGN KEY (job_id) REFERENCES jobs(id)
) ENGINE=InnoDB;

CREATE INDEX idx_notifications_recipient_created
  ON notifications (recipient_user_id, created_at DESC, id DESC);

CREATE INDEX idx_notifications_recipient_unread
  ON notifications (recipient_user_id, read_at, created_at DESC, id DESC);
```

---

## 9. Data Invariants

These invariants are mandatory.

1. A job belongs to exactly one quote.
2. A job belongs to exactly one technician.
3. A job belongs to exactly one manager.
4. A quote can have at most one job.
5. A job is exactly two hours long.
6. A technician cannot have overlapping jobs.
7. A completed job remains attached to its original quote, manager, technician, and time window.
8. Completed jobs cannot be completed again.
9. Completed jobs cannot be rescheduled.
10. Only managers can assign jobs.
11. Only managers can reschedule jobs.
12. Only the manager who created a job can reschedule that job.
13. Only the assigned technician can complete a job.
14. Notifications are persisted in MySQL.
15. WebSocket events are emitted only after the database transaction commits.
16. The frontend is not trusted for conflict prevention.
17. The backend is the source of truth for all job lifecycle changes.

---

## 10. Domain Types

Use these conceptual Go domain types.

```go
type Role string

const (
    RoleManager    Role = "manager"
    RoleTechnician Role = "technician"
)

type QuoteStatus string

const (
    QuoteUnscheduled QuoteStatus = "unscheduled"
    QuoteScheduled   QuoteStatus = "scheduled"
)

type JobStatus string

const (
    JobScheduled JobStatus = "scheduled"
    JobCompleted JobStatus = "completed"
)

type NotificationType string

const (
    NotificationJobAssigned  NotificationType = "job_assigned"
    NotificationJobUpdated   NotificationType = "job_updated"
    NotificationJobCompleted NotificationType = "job_completed"
)
```

Authenticated actor:

```go
type Actor struct {
    UserID       int64
    Email        string
    DisplayName  string
    Role         Role
    ManagerID    *int64
    TechnicianID *int64
}
```

Assignment input:

```go
type AssignJobInput struct {
    QuoteID      int64
    TechnicianID int64
    StartsAt     time.Time
}
```

Reschedule input:

```go
type RescheduleJobInput struct {
    JobID        int64
    TechnicianID int64
    StartsAt     time.Time
}
```

Completion input:

```go
type CompleteJobInput struct {
    JobID int64
}
```

---

## 11. Error Contract

Use service errors that map cleanly to HTTP responses.

```go
type ErrorCode string

const (
    ErrorInvalidInput           ErrorCode = "invalid_input"
    ErrorUnauthorized           ErrorCode = "unauthorized"
    ErrorForbidden              ErrorCode = "forbidden"
    ErrorNotFound               ErrorCode = "not_found"
    ErrorInvalidCredentials     ErrorCode = "invalid_credentials"
    ErrorQuoteAlreadyScheduled  ErrorCode = "quote_already_scheduled"
    ErrorScheduleConflict       ErrorCode = "schedule_conflict"
    ErrorCompletedJobImmutable  ErrorCode = "completed_job_immutable"
    ErrorJobAlreadyCompleted    ErrorCode = "job_already_completed"
    ErrorInternal               ErrorCode = "internal_error"
)
```

HTTP mapping:

| Error code | HTTP status |
|---|---:|
| `invalid_input` | 400 |
| `unauthorized` | 401 |
| `invalid_credentials` | 401 |
| `forbidden` | 403 |
| `not_found` | 404 |
| `quote_already_scheduled` | 409 |
| `schedule_conflict` | 409 |
| `completed_job_immutable` | 409 |
| `job_already_completed` | 409 |
| `internal_error` | 500 |

All API errors use this JSON shape:

```json
{
  "error": {
    "code": "schedule_conflict",
    "message": "Technician already has a job in that time window."
  }
}
```

---

## 12. Conflict Prevention

### 12.1 Overlap Formula

A requested job conflicts with an existing job for the same technician when:

```sql
existing.starts_at < requested.ends_at
AND
existing.ends_at > requested.starts_at
```

Given existing job:

```txt
10:00–12:00
```

Expected results:

| Requested window | Result |
|---|---|
| 08:00–10:00 | allowed |
| 09:00–11:00 | conflict |
| 10:00–12:00 | conflict |
| 11:00–13:00 | conflict |
| 12:00–14:00 | allowed |

### 12.2 Assignment Transaction

`SchedulingService.AssignJob` must run inside one MySQL transaction.

Exact order:

1. Validate actor is manager.
2. Validate `quoteId > 0`.
3. Validate `technicianId > 0`.
4. Validate `startsAt` is minute-aligned.
5. Compute `endsAt = startsAt + 2 hours`.
6. Begin transaction.
7. Select quote row `FOR UPDATE`.
8. If quote is missing, return `not_found`.
9. If quote status is not `unscheduled`, return `quote_already_scheduled`.
10. Select technician row `FOR UPDATE`.
11. If technician is missing, return `not_found`.
12. Check overlapping jobs for the technician.
13. If overlap exists, return `schedule_conflict`.
14. Insert job.
15. Update quote status to `scheduled`.
16. Insert technician notification.
17. Commit transaction.
18. Return job and post-commit events.

### 12.3 Technician Row Lock Rule

All assignment and reschedule operations for a technician must lock the target technician row before checking overlaps.

This serializes scheduling writes per technician and prevents two concurrent managers from both passing the overlap check.

### 12.4 Duplicate Quote Protection

The `jobs.quote_id` column is unique.

If duplicate quote scheduling reaches the insert statement, the service must translate the duplicate-key error into:

```txt
quote_already_scheduled
```

### 12.5 Reschedule Transaction

`SchedulingService.RescheduleJob` must run inside one MySQL transaction.

Exact order:

1. Validate actor is manager.
2. Validate `jobId > 0`.
3. Validate `technicianId > 0`.
4. Validate `startsAt` is minute-aligned.
5. Compute `endsAt = startsAt + 2 hours`.
6. Begin transaction.
7. Select job row `FOR UPDATE`.
8. If job is missing, return `not_found`.
9. If `actor.managerId != job.manager_id`, return `forbidden`.
10. If job status is `completed`, return `completed_job_immutable`.
11. Store previous technician ID.
12. Select target technician row `FOR UPDATE`.
13. If technician is missing, return `not_found`.
14. Check overlapping jobs for the target technician, excluding the current job.
15. If overlap exists, return `schedule_conflict`.
16. Update `technician_id`, `starts_at`, and `ends_at`.
17. Insert `job_updated` notification for target technician.
18. If previous technician differs from target technician, insert `job_updated` notification for previous technician.
19. Commit transaction.
20. Return job and post-commit events.

### 12.6 Completion Transaction

`SchedulingService.CompleteJob` must run inside one MySQL transaction.

Exact order:

1. Validate actor is technician.
2. Validate `jobId > 0`.
3. Begin transaction.
4. Select job row `FOR UPDATE`.
5. If job is missing, return `not_found`.
6. If `actor.technicianId != job.technician_id`, return `forbidden`.
7. If job status is `completed`, return `job_already_completed`.
8. Update job status to `completed`.
9. Set `completed_at` to current UTC clock time.
10. Insert `job_completed` notification for the manager user.
11. Commit transaction.
12. Return job and post-commit events.

---

## 13. Service Architecture

Implement these backend services.

1. `AuthService`
2. `QuoteService`
3. `TechnicianService`
4. `JobQueryService`
5. `SchedulingService`
6. `NotificationService`
7. `EventPublisher` / `WebSocketHub`

### 13.1 `AuthService`

Responsibilities:

1. Login.
2. Verify bcrypt password.
3. Create signed session token.
4. Parse signed session token.
5. Build `Actor` from user row.

Required public methods:

```go
Login(ctx context.Context, email string, password string) (Actor, string, error)
ActorFromToken(ctx context.Context, token string) (Actor, error)
```

### 13.2 `QuoteService`

Responsibilities:

1. List quotes visible to a manager.
2. Validate quote status filter.
3. Reject technician access.

Required public method:

```go
ListQuotes(ctx context.Context, actor Actor, status *QuoteStatus) ([]QuoteDTO, error)
```

### 13.3 `TechnicianService`

Responsibilities:

1. List technicians for manager assignment UI.
2. Reject technician access.

Required public method:

```go
ListTechnicians(ctx context.Context, actor Actor) ([]TechnicianDTO, error)
```

### 13.4 `JobQueryService`

Responsibilities:

1. List jobs visible to current actor.
2. Managers see jobs they created.
3. Technicians see jobs assigned to them.

Required public method:

```go
ListJobs(ctx context.Context, actor Actor) ([]JobDTO, error)
```

### 13.5 `SchedulingService`

Responsibilities:

1. Assign job.
2. Reschedule or reassign job.
3. Complete job.
4. Enforce conflicts.
5. Create notifications.
6. Return post-commit events.

Required public methods:

```go
AssignJob(ctx context.Context, actor Actor, input AssignJobInput) (JobDTO, []DomainEvent, error)

RescheduleJob(ctx context.Context, actor Actor, input RescheduleJobInput) (JobDTO, []DomainEvent, error)

CompleteJob(ctx context.Context, actor Actor, input CompleteJobInput) (JobDTO, []DomainEvent, error)
```

### 13.6 `NotificationService`

Responsibilities:

1. List notifications for actor.
2. Mark actor's notification as read.
3. Reject access to other users' notifications.

Required public methods:

```go
ListNotifications(ctx context.Context, actor Actor) ([]NotificationDTO, error)

MarkRead(ctx context.Context, actor Actor, notificationID int64) (NotificationDTO, error)
```

### 13.7 `EventPublisher`

The scheduling service returns domain events. HTTP handlers publish them after successful service return.

```go
type DomainEvent struct {
    Type           string
    TargetUserID   *int64
    TargetRole     *Role
    JobID          *int64
    QuoteID        *int64
    NotificationID *int64
}
```

Rules:

1. `TargetUserID` is used for direct user events.
2. `TargetRole = manager` is used for manager broadcast events.
3. Events are never published before transaction commit.

---

## 14. API Specification

### 14.1 Common Rules

| Item | Rule |
|---|---|
| Base URL | `http://localhost:8080` |
| Content type | `application/json` |
| Auth cookie | `brix_session` |
| Protected endpoints | Require `brix_session` |
| Timestamp response format | RFC3339 UTC |

### 14.2 Health

#### `GET /healthz`

Status: `200`

Response:

```json
{
  "ok": true
}
```

---

## 15. Auth API

### 15.1 `POST /api/auth/login`

Request:

```json
{
  "email": "manager1@brix.test",
  "password": "password123"
}
```

Rules:

1. Email is trimmed.
2. Email is lowercased before lookup.
3. Password is compared using bcrypt.
4. Successful login sets `brix_session` cookie.

Success status: `200`

Response:

```json
{
  "user": {
    "id": 1,
    "email": "manager1@brix.test",
    "displayName": "Sarah Manager",
    "role": "manager",
    "managerId": 1,
    "technicianId": null
  }
}
```

Invalid credentials status: `401`

Response:

```json
{
  "error": {
    "code": "invalid_credentials",
    "message": "Invalid email or password."
  }
}
```

### 15.2 `POST /api/auth/logout`

Success status: `200`

Side effect: clears `brix_session` cookie.

Response:

```json
{
  "ok": true
}
```

### 15.3 `GET /api/me`

Protected.

Success status: `200`

Manager response:

```json
{
  "user": {
    "id": 1,
    "email": "manager1@brix.test",
    "displayName": "Sarah Manager",
    "role": "manager",
    "managerId": 1,
    "technicianId": null
  }
}
```

Technician response:

```json
{
  "user": {
    "id": 3,
    "email": "technician1@brix.test",
    "displayName": "Tom Technician",
    "role": "technician",
    "managerId": null,
    "technicianId": 1
  }
}
```

Unauthenticated status: `401`.

---

## 16. Quote API

### 16.1 `GET /api/quotes?status=unscheduled`

Manager-only.

Valid `status` values:

1. `unscheduled`
2. `scheduled`

If `status` is omitted, return all quotes.

Sort order:

```txt
created_at ASC, id ASC
```

Success status: `200`

Response:

```json
{
  "quotes": [
    {
      "id": 1,
      "customerName": "Acme Plumbing",
      "description": "Replace leaking kitchen tap",
      "status": "unscheduled",
      "createdAt": "2026-05-11T00:00:00Z",
      "updatedAt": "2026-05-11T00:00:00Z"
    }
  ]
}
```

Technician access status: `403`.

Invalid status value status: `400`.

---

## 17. Technician API

### 17.1 `GET /api/technicians`

Manager-only.

Sort order:

```txt
display_name ASC, id ASC
```

Success status: `200`

Response:

```json
{
  "technicians": [
    {
      "id": 2,
      "userId": 4,
      "displayName": "Priya Technician",
      "email": "technician2@brix.test"
    },
    {
      "id": 1,
      "userId": 3,
      "displayName": "Tom Technician",
      "email": "technician1@brix.test"
    }
  ]
}
```

Technician access status: `403`.

---

## 18. Job API

### 18.1 `GET /api/jobs`

Protected.

Manager rule:

```txt
Return jobs where jobs.manager_id = actor.manager_id.
```

Technician rule:

```txt
Return jobs where jobs.technician_id = actor.technician_id.
```

Sort order:

```txt
starts_at ASC, id ASC
```

Success status: `200`

Response:

```json
{
  "jobs": [
    {
      "id": 1,
      "quoteId": 1,
      "quoteCustomerName": "Acme Plumbing",
      "quoteDescription": "Replace leaking kitchen tap",
      "technicianId": 1,
      "technicianName": "Tom Technician",
      "managerId": 1,
      "managerName": "Sarah Manager",
      "startsAt": "2026-05-12T00:00:00Z",
      "endsAt": "2026-05-12T02:00:00Z",
      "status": "scheduled",
      "completedAt": null,
      "createdAt": "2026-05-11T00:00:00Z",
      "updatedAt": "2026-05-11T00:00:00Z"
    }
  ]
}
```

### 18.2 `POST /api/jobs`

Manager-only.

Request:

```json
{
  "quoteId": 1,
  "technicianId": 1,
  "startsAt": "2026-05-12T00:00:00Z"
}
```

Success status: `201`

Response:

```json
{
  "job": {
    "id": 1,
    "quoteId": 1,
    "technicianId": 1,
    "managerId": 1,
    "startsAt": "2026-05-12T00:00:00Z",
    "endsAt": "2026-05-12T02:00:00Z",
    "status": "scheduled",
    "completedAt": null
  }
}
```

Overlap conflict status: `409`

Response:

```json
{
  "error": {
    "code": "schedule_conflict",
    "message": "Technician already has a job in that time window."
  }
}
```

Duplicate quote status: `409`

Response:

```json
{
  "error": {
    "code": "quote_already_scheduled",
    "message": "Quote has already been scheduled."
  }
}
```

### 18.3 `PATCH /api/jobs/{id}/schedule`

Manager-only.

Request:

```json
{
  "technicianId": 2,
  "startsAt": "2026-05-12T04:00:00Z"
}
```

Success status: `200`

Response:

```json
{
  "job": {
    "id": 1,
    "quoteId": 1,
    "technicianId": 2,
    "managerId": 1,
    "startsAt": "2026-05-12T04:00:00Z",
    "endsAt": "2026-05-12T06:00:00Z",
    "status": "scheduled",
    "completedAt": null
  }
}
```

Rules:

1. Only the manager who created the job can reschedule it.
2. Completed jobs cannot be rescheduled.
3. Rescheduling must respect the same overlap rule as assignment.

### 18.4 `PATCH /api/jobs/{id}/complete`

Technician-only.

Request:

```json
{}
```

Success status: `200`

Response:

```json
{
  "job": {
    "id": 1,
    "status": "completed",
    "completedAt": "2026-05-12T06:30:00Z"
  }
}
```

Rules:

1. Only the assigned technician can complete the job.
2. A completed job cannot be completed twice.

---

## 19. Notification API

### 19.1 `GET /api/notifications`

Protected.

Returns notifications for the authenticated user.

Sort order:

```txt
created_at DESC, id DESC
```

Success status: `200`

Response:

```json
{
  "notifications": [
    {
      "id": 1,
      "type": "job_assigned",
      "message": "You have been assigned job #1 for Acme Plumbing.",
      "jobId": 1,
      "readAt": null,
      "createdAt": "2026-05-11T00:00:00Z"
    }
  ]
}
```

### 19.2 `PATCH /api/notifications/{id}/read`

Protected.

Rules:

1. Only the recipient can mark a notification as read.
2. Marking an already-read notification succeeds and returns the existing `readAt`.

Success status: `200`

Response:

```json
{
  "notification": {
    "id": 1,
    "readAt": "2026-05-11T00:05:00Z"
  }
}
```

---

## 20. WebSocket API

### 20.1 Endpoint

```txt
GET /ws
```

Auth:

1. Uses the same `brix_session` cookie as HTTP endpoints.
2. Unauthenticated attempts are rejected before upgrade.

### 20.2 Event Shape

Server-to-client events:

```ts
type ServerEvent =
  | {
      type: "notification.created";
      notificationId: number;
      jobId: number | null;
    }
  | {
      type: "jobs.changed";
      jobId: number;
    }
  | {
      type: "quotes.changed";
      quoteId: number;
    };
```

No full job, quote, user, or notification payloads are pushed over WebSocket.

Frontend refetch behavior:

| Event | Frontend action |
|---|---|
| `notification.created` | Refetch `/api/notifications` |
| `jobs.changed` | Refetch `/api/jobs` |
| `quotes.changed` | If actor is manager, refetch `/api/quotes?status=unscheduled` |

### 20.3 Broadcast Rules

Job assigned, after commit:

1. Send `notification.created` to assigned technician user.
2. Send `jobs.changed` to assigned technician user.
3. Send `quotes.changed` to all connected managers.
4. Send `jobs.changed` to assigning manager user.

Job rescheduled or reassigned, after commit:

1. Send `notification.created` to target technician user.
2. If technician changed, send `notification.created` to previous technician user.
3. Send `jobs.changed` to target technician user.
4. If technician changed, send `jobs.changed` to previous technician user.
5. Send `jobs.changed` to manager user.

Job completed, after commit:

1. Send `notification.created` to manager user.
2. Send `jobs.changed` to manager user.
3. Send `jobs.changed` to technician user.

### 20.4 Hub Deployment Rule

The WebSocket hub is in-process.

The in-process hub is appropriate for a single API instance. A production multi-instance deployment would need Redis Pub/Sub, NATS, or another fanout mechanism.

---

## 21. Frontend Specification

### 21.1 Routes

Use Next.js App Router.

| Route | Behavior |
|---|---|
| `/` | Checks `/api/me`; redirects unauthenticated users to `/login`; redirects managers to `/manager`; redirects technicians to `/technician` |
| `/login` | Login form and seeded account helper buttons |
| `/manager` | Manager dashboard |
| `/technician` | Technician dashboard |

### 21.2 Login UI

The login page must include:

1. Email input.
2. Password input.
3. Login button.
4. Seeded manager login shortcut.
5. Seeded technician login shortcut.
6. Error alert.

### 21.3 Manager Dashboard

The manager dashboard must show:

1. Current user display.
2. Logout button.
3. Notifications panel.
4. Unscheduled quotes list.
5. Technician selector.
6. Start-time input.
7. Assign button.
8. Manager jobs list.
9. Reschedule controls for scheduled jobs.
10. Visible conflict error when scheduling fails.

Use MUI components such as:

1. `Container`
2. `AppBar`
3. `Toolbar`
4. `Typography`
5. `Card`
6. `Table`
7. `Select`
8. `TextField`
9. `Button`
10. `Alert`
11. `Snackbar`
12. `Chip`

Do not use MUI X Data Grid.

### 21.4 Technician Dashboard

The technician dashboard must show:

1. Current user display.
2. Logout button.
3. Notifications panel.
4. Assigned jobs list.
5. Scheduled/completed status chip.
6. Complete button for scheduled jobs.

### 21.5 Frontend State Rules

1. The frontend fetches API data over HTTP.
2. The frontend includes credentials on API requests.
3. The frontend connects to WebSocket after successful auth.
4. WebSocket events trigger HTTP refetches.
5. The frontend does not attempt complex client-side state reconciliation from WebSocket payloads.

### 21.6 Frontend Time Handling

For a `datetime-local` input:

```ts
const startsAtIso = new Date(localDateTimeValue).toISOString();
```

For display:

```ts
new Date(job.startsAt).toLocaleString();
```

---

## 22. Docker Compose Specification

### 22.1 Required Services

The project must define these services in `compose.yaml`:

1. `mysql`
2. `backend`
3. `frontend`

Optional services:

1. `migrate`
2. `seed`

### 22.2 Ports

| Service | Host port | Container port |
|---|---:|---:|
| frontend | 3000 | 3000 |
| backend | 8080 | 8080 |
| mysql | 3307 | 3306 |

### 22.3 `.env.example`

Root `.env.example` must include:

```env
MYSQL_DATABASE=brix_scheduler
MYSQL_USER=brix
MYSQL_PASSWORD=brix_password
MYSQL_ROOT_PASSWORD=root_password

BACKEND_PORT=8080
FRONTEND_PORT=3000

APP_ENV=development
JWT_SECRET=dev_only_change_me
COOKIE_NAME=brix_session
COOKIE_SECURE=false
CORS_ALLOWED_ORIGIN=http://localhost:3000

NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_WS_URL=ws://localhost:8080/ws
```

### 22.4 Required Commands

The repo must support:

```bash
docker compose up --build
docker compose down -v
```

The repo should support:

```bash
just up
just down
just reset
just migrate
just seed
just test
just test-backend
just test-frontend
just test-e2e
```

---

## 23. Repository Structure

Use this structure.

```txt
.
├── SPEC.md
├── TEST_SPEC.md
├── CODEX_IMPLEMENTATION_PLAN.md
├── compose.yaml
├── compose.test.yaml
├── justfile
├── .env.example
│
├── docs
│   ├── COMPLIANCE_MATRIX.md
│   └── research
│       ├── 00-tooling.md
│       ├── 01-mysql-locking.md
│       ├── 02-auth.md
│       ├── 03-websockets.md
│       ├── 04-frontend.md
│       └── 05-e2e.md
│
├── backend
│   ├── Dockerfile
│   ├── go.mod
│   ├── go.sum
│   ├── cmd
│   │   ├── api
│   │   │   └── main.go
│   │   ├── migrate
│   │   │   └── main.go
│   │   └── seed
│   │       └── main.go
│   ├── migrations
│   │   ├── 001_init.sql
│   │   └── 002_seed.sql
│   └── internal
│       ├── auth
│       ├── config
│       ├── db
│       ├── domain
│       ├── httpapi
│       ├── jobs
│       ├── notifications
│       ├── quotes
│       ├── scheduling
│       ├── technicians
│       └── ws
│
├── frontend
│   ├── Dockerfile
│   ├── package.json
│   ├── package-lock.json
│   ├── next.config.ts
│   ├── tsconfig.json
│   ├── app
│   │   ├── layout.tsx
│   │   ├── page.tsx
│   │   ├── login
│   │   │   └── page.tsx
│   │   ├── manager
│   │   │   └── page.tsx
│   │   └── technician
│   │       └── page.tsx
│   ├── components
│   ├── lib
│   └── tests
│
└── e2e
    └── scheduling.spec.ts
```

---

## 24. Acceptance Criteria

The project is acceptable only if all items below are true.

1. `docker compose up --build` starts MySQL, backend, and frontend.
2. Frontend is available at `http://localhost:3000`.
3. Backend health check is available at `http://localhost:8080/healthz`.
4. Seeded manager can log in.
5. Seeded technician can log in.
6. Manager can view unscheduled quotes.
7. Manager can assign quote to technician.
8. Job window is exactly two hours.
9. Backend rejects overlapping jobs.
10. Backend rejects duplicate quote scheduling.
11. Technician can view assigned jobs.
12. Technician can complete assigned scheduled job.
13. Manager receives notification when job is completed.
14. Technician receives notification when job is assigned.
15. Technician receives notification when job is updated.
16. WebSocket events trigger UI refresh.
17. Tests cover assignment, conflict prevention, completion, and notifications.
18. `just test` passes before final submission.
