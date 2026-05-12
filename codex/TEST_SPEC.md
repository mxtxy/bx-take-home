# Brix Service Scheduling & Notification System — Test Specification

## 0. Purpose

This document defines the test-driven design contract for the project.

Implementation must proceed test-first. For each service or feature, write tests before implementation, run them to confirm expected failure, implement the smallest correct change, then rerun tests.

The tests are not optional; they are the proof that the implementation satisfies `SPEC.md`.

---

## 1. Test Strategy

### 1.1 Test Categories

The project uses the following test categories.

| Category | Purpose |
|---|---|
| Pure unit tests | Validate deterministic domain logic without database or HTTP |
| MySQL integration tests | Validate schema, transactions, locking, and persistence |
| Service tests | Validate service contracts and authorization decisions |
| HTTP handler tests | Validate API routing, JSON shape, status codes, and cookies |
| WebSocket tests | Validate connection auth, hub routing, and event delivery |
| Frontend tests | Validate rendering, API client behavior, and dashboard controls |
| E2E tests | Validate user flows through browser and Docker Compose environment |

### 1.2 Test Infrastructure Rules

1. Use a real MySQL database for integration tests.
2. Use deterministic fixtures.
3. Use a fake clock for completion and notification read timestamps.
4. Use a fake event publisher for service and HTTP tests where direct WebSocket testing is not required.
5. Do not rely on test ordering.
6. Reset the test database before each integration test.
7. Keep service tests narrower than E2E tests.
8. Every service error must be asserted by code, not by message string alone.

### 1.3 Fixed Test Time

Use this fixed test time unless a test states otherwise:

```txt
2026-05-12T00:00:00Z
```

### 1.4 Test Database Reset

Before each MySQL integration test:

```sql
SET FOREIGN_KEY_CHECKS = 0;
TRUNCATE TABLE notifications;
TRUNCATE TABLE jobs;
TRUNCATE TABLE quotes;
TRUNCATE TABLE technicians;
TRUNCATE TABLE managers;
TRUNCATE TABLE users;
SET FOREIGN_KEY_CHECKS = 1;
```

Then insert deterministic users, managers, technicians, and quotes.

### 1.5 Required Root Commands

The repository must support:

```bash
just test
just test-backend
just test-frontend
just test-e2e
```

Expected behavior:

```txt
just test-backend:
  go test ./...
  backend internal production package coverage is 100.0%

just test-frontend:
  npm run lint
  npm run typecheck
  npm test -- --run --coverage

just test-e2e:
  docker compose up -d --build
  playwright test
```

The final submission must pass:

```bash
docker compose down -v
docker compose up --build
just test
```

---

## 2. Database Schema Tests

File:

```txt
backend/internal/db/schema_test.go
```

These tests run after migrations have been applied to an empty test database.

### 2.1 `TestSchema_RequiredTablesExist`

Expected tables:

1. `users`
2. `managers`
3. `technicians`
4. `quotes`
5. `jobs`
6. `notifications`

### 2.2 `TestSchema_UsersEmailUnique`

Given:

1. Insert user with email `duplicate@brix.test`.
2. Insert second user with same email.

Expected:

```txt
second insert fails with duplicate key error
```

### 2.3 `TestSchema_ManagersUserUnique`

Given:

1. Insert one manager row for user 1.
2. Insert second manager row for user 1.

Expected:

```txt
second insert fails with duplicate key error
```

### 2.4 `TestSchema_TechniciansUserUnique`

Given:

1. Insert one technician row for user 3.
2. Insert second technician row for user 3.

Expected:

```txt
second insert fails with duplicate key error
```

### 2.5 `TestSchema_JobsQuoteUnique`

Given:

1. Insert job for quote 1.
2. Insert second job for quote 1.

Expected:

```txt
second insert fails with duplicate key error
```

### 2.6 `TestSchema_JobMustReferenceExistingQuote`

Given:

```txt
insert job with quote_id = 999
```

Expected:

```txt
insert fails with foreign key error
```

### 2.7 `TestSchema_JobMustReferenceExistingTechnician`

Given:

```txt
insert job with technician_id = 999
```

Expected:

```txt
insert fails with foreign key error
```

### 2.8 `TestSchema_JobMustReferenceExistingManager`

Given:

```txt
insert job with manager_id = 999
```

Expected:

```txt
insert fails with foreign key error
```

### 2.9 `TestSchema_JobWindowMustBePositive`

Given:

```txt
starts_at = 2026-05-12T12:00:00Z
ends_at = 2026-05-12T10:00:00Z
```

Expected:

```txt
insert fails with check constraint error
```

### 2.10 `TestSchema_JobWindowMustBeTwoHours`

Given:

```txt
starts_at = 2026-05-12T10:00:00Z
ends_at = 2026-05-12T11:00:00Z
```

Expected:

```txt
insert fails with check constraint error
```

### 2.11 `TestSeed_SeededUsersExist`

Expected users:

1. `manager1@brix.test`
2. `manager2@brix.test`
3. `technician1@brix.test`
4. `technician2@brix.test`

Expected:

```txt
all exist and password_hash is non-empty
```

### 2.12 `TestSeed_SeededQuotesExist`

Expected:

```txt
5 quotes exist
all quotes have status = unscheduled
```

---

## 3. Pure Domain Tests

File:

```txt
backend/internal/domain/time_window_test.go
```

### 3.1 `TestCalculateWindow_ReturnsExactlyTwoHours`

Given:

```txt
startsAt = 2026-05-12T10:00:00Z
```

Expected:

```txt
endsAt = 2026-05-12T12:00:00Z
duration = 120 minutes
```

### 3.2 `TestValidateStartsAt_RejectsSecondPrecision`

Given:

```txt
startsAt = 2026-05-12T10:00:01Z
```

Expected:

```txt
invalid_input
```

### 3.3 `TestValidateStartsAt_RejectsNanosecondPrecision`

Given:

```txt
startsAt = 2026-05-12T10:00:00.000000001Z
```

Expected:

```txt
invalid_input
```

### 3.4 `TestValidateStartsAt_AllowsMinutePrecision`

Given:

```txt
startsAt = 2026-05-12T10:30:00Z
```

Expected:

```txt
valid
```

### 3.5 `TestOverlaps_BoundaryCases`

Cases:

| Existing window | Requested window | Expected |
|---|---|---|
| 10:00–12:00 | 08:00–10:00 | false |
| 10:00–12:00 | 09:00–11:00 | true |
| 10:00–12:00 | 10:00–12:00 | true |
| 10:00–12:00 | 11:00–13:00 | true |
| 10:00–12:00 | 12:00–14:00 | false |

### 3.6 `TestValidateAssignJobInput_RejectsInvalidIDs`

Cases:

| quoteId | technicianId | Expected |
|---:|---:|---|
| 0 | 1 | `invalid_input` |
| -1 | 1 | `invalid_input` |
| 1 | 0 | `invalid_input` |
| 1 | -1 | `invalid_input` |

### 3.7 `TestValidateRescheduleJobInput_RejectsInvalidIDs`

Cases:

| jobId | technicianId | Expected |
|---:|---:|---|
| 0 | 1 | `invalid_input` |
| -1 | 1 | `invalid_input` |
| 1 | 0 | `invalid_input` |
| 1 | -1 | `invalid_input` |

### 3.8 `TestValidateCompleteJobInput_RejectsInvalidIDs`

Cases:

| jobId | Expected |
|---:|---|
| 0 | `invalid_input` |
| -1 | `invalid_input` |

---

## 4. AuthService Tests

File:

```txt
backend/internal/auth/auth_service_test.go
```

### 4.1 `TestAuthService_Login_ManagerSuccess`

Given:

```txt
email = manager1@brix.test
password = password123
```

Expected actor:

```txt
UserID = 1
Email = manager1@brix.test
DisplayName = Sarah Manager
Role = manager
ManagerID = 1
TechnicianID = nil
```

Expected token:

```txt
non-empty
parseable by ActorFromToken
```

### 4.2 `TestAuthService_Login_TechnicianSuccess`

Given:

```txt
email = technician1@brix.test
password = password123
```

Expected actor:

```txt
UserID = 3
Role = technician
ManagerID = nil
TechnicianID = 1
```

### 4.3 `TestAuthService_Login_NormalizesEmail`

Given:

```txt
email = " Manager1@Brix.Test "
password = password123
```

Expected:

```txt
login succeeds as manager1@brix.test
```

### 4.4 `TestAuthService_Login_InvalidPassword`

Given:

```txt
email = manager1@brix.test
password = wrong
```

Expected:

```txt
invalid_credentials
```

### 4.5 `TestAuthService_Login_UnknownEmail`

Given:

```txt
email = missing@brix.test
password = password123
```

Expected:

```txt
invalid_credentials
```

### 4.6 `TestAuthService_ActorFromToken_RejectsTamperedToken`

Given:

```txt
valid token with one character changed
```

Expected:

```txt
unauthorized
```

### 4.7 `TestAuthService_ActorFromToken_RejectsExpiredToken`

Given:

```txt
token expiry in the past
```

Expected:

```txt
unauthorized
```

### 4.8 `TestAuthService_ActorFromToken_RejectsMissingUser`

Given:

```txt
validly signed token for user id 999
```

Expected:

```txt
unauthorized
```

---

## 5. QuoteService Tests

File:

```txt
backend/internal/quotes/quote_service_test.go
```

### 5.1 `TestQuoteService_ListQuotes_ManagerCanListUnscheduled`

Given:

```txt
actor = manager1
quotes 1-5 are unscheduled
```

When:

```go
ListQuotes(ctx, manager1, status = unscheduled)
```

Expected:

```txt
returns 5 quotes
all status = unscheduled
sorted by createdAt ASC, id ASC
```

### 5.2 `TestQuoteService_ListQuotes_ExcludesScheduledQuotes`

Given:

```txt
quote 1 has been scheduled
quotes 2-5 are unscheduled
```

Expected:

```txt
returns quotes 2-5 only
quote 1 absent
```

### 5.3 `TestQuoteService_ListQuotes_CanListScheduled`

Given:

```txt
quote 1 has been scheduled
quotes 2-5 are unscheduled
```

When:

```go
ListQuotes(ctx, manager1, status = scheduled)
```

Expected:

```txt
returns quote 1 only
```

### 5.4 `TestQuoteService_ListQuotes_CanListAllWhenStatusNil`

Given:

```txt
quote 1 scheduled
quotes 2-5 unscheduled
```

When:

```go
ListQuotes(ctx, manager1, status = nil)
```

Expected:

```txt
returns 5 quotes
```

### 5.5 `TestQuoteService_ListQuotes_TechnicianForbidden`

Given:

```txt
actor = technician1
```

Expected:

```txt
forbidden
```

### 5.6 `TestQuoteService_ListQuotes_InvalidStatus`

Given:

```txt
status = "archived"
```

Expected:

```txt
invalid_input
```

---

## 6. TechnicianService Tests

File:

```txt
backend/internal/technicians/technician_service_test.go
```

### 6.1 `TestTechnicianService_ListTechnicians_ManagerSuccess`

Given:

```txt
actor = manager1
```

Expected:

```txt
returns Tom Technician and Priya Technician
each item includes technician id, user id, display name, email
```

### 6.2 `TestTechnicianService_ListTechnicians_SortedByDisplayNameThenID`

Expected order:

```txt
Priya Technician
Tom Technician
```

Sort rule:

```txt
display_name ASC, id ASC
```

### 6.3 `TestTechnicianService_ListTechnicians_TechnicianForbidden`

Given:

```txt
actor = technician1
```

Expected:

```txt
forbidden
```

---

## 7. JobQueryService Tests

File:

```txt
backend/internal/jobs/job_query_service_test.go
```

### 7.1 `TestJobQueryService_ListJobs_ManagerSeesOwnJobsOnly`

Given:

```txt
job 1 created by manager1
job 2 created by manager2
```

When:

```go
ListJobs(ctx, manager1)
```

Expected:

```txt
returns job 1 only
does not return job 2
```

### 7.2 `TestJobQueryService_ListJobs_TechnicianSeesAssignedJobsOnly`

Given:

```txt
job 1 assigned to technician1
job 2 assigned to technician2
```

When:

```go
ListJobs(ctx, technician1)
```

Expected:

```txt
returns job 1 only
does not return job 2
```

### 7.3 `TestJobQueryService_ListJobs_SortedByStartsAtThenID`

Given jobs:

```txt
job A starts 14:00
job B starts 10:00
job C starts 10:00 with higher id
```

Expected order:

```txt
job B
job C
job A
```

### 7.4 `TestJobQueryService_ListJobs_IncludesQuoteTechnicianManagerNames`

Expected each job DTO includes:

1. `quoteCustomerName`
2. `quoteDescription`
3. `technicianName`
4. `managerName`

### 7.5 `TestJobQueryService_ListJobs_RejectsMalformedActor`

Given:

```txt
actor role = manager
managerId = nil
```

Expected:

```txt
unauthorized or forbidden
```

Given:

```txt
actor role = technician
technicianId = nil
```

Expected:

```txt
unauthorized or forbidden
```

Implementation must choose one code and use it consistently. Preferred: `unauthorized` for malformed session-derived actors.

---

## 8. SchedulingService Assignment Tests

File:

```txt
backend/internal/scheduling/scheduling_assign_test.go
```

### 8.1 `TestSchedulingService_AssignJob_Success`

Given:

```txt
actor = manager1
quoteId = 1
technicianId = 1
startsAt = 2026-05-12T10:00:00Z
```

Expected job:

```txt
quoteId = 1
technicianId = 1
managerId = 1
startsAt = 2026-05-12T10:00:00Z
endsAt = 2026-05-12T12:00:00Z
status = scheduled
completedAt = nil
```

Expected DB state:

```txt
jobs count = 1
quotes.id=1 status = scheduled
notifications count for technician1 user = 1
notification type = job_assigned
```

Expected events:

```txt
notification.created to technician1 user
jobs.changed to technician1 user
quotes.changed to manager role
jobs.changed to manager1 user
```

### 8.2 `TestSchedulingService_AssignJob_TechnicianActorForbidden`

Given:

```txt
actor = technician1
```

Expected:

```txt
forbidden
no job inserted
no quote update
no notification inserted
no events returned
```

### 8.3 `TestSchedulingService_AssignJob_MalformedManagerActorForbidden`

Given:

```txt
actor role = manager
actor.managerId = nil
```

Expected:

```txt
forbidden or unauthorized
no mutation
```

Implementation must choose one code and use it consistently. Preferred: `forbidden` when the actor role cannot perform the operation due to missing domain identity.

### 8.4 `TestSchedulingService_AssignJob_InvalidInput`

Cases:

1. `quoteId = 0`.
2. `technicianId = 0`.
3. `startsAt` has seconds precision.

Expected:

```txt
invalid_input
no mutation
```

### 8.5 `TestSchedulingService_AssignJob_MissingQuote`

Given:

```txt
quoteId = 999
```

Expected:

```txt
not_found
```

### 8.6 `TestSchedulingService_AssignJob_MissingTechnician`

Given:

```txt
technicianId = 999
```

Expected:

```txt
not_found
```

### 8.7 `TestSchedulingService_AssignJob_QuoteAlreadyScheduled`

Given:

```txt
quote 1 already has a job
```

When:

```txt
assign quote 1 again
```

Expected:

```txt
quote_already_scheduled
jobs count for quote 1 remains 1
```

### 8.8 `TestSchedulingService_AssignJob_OverlapConflict`

Given existing job:

```txt
technician1
10:00–12:00
```

When requested:

```txt
technician1
11:00–13:00
```

Expected:

```txt
schedule_conflict
no new job
quote remains unscheduled
no notification
no events
```

### 8.9 `TestSchedulingService_AssignJob_ExactOverlapConflict`

Existing:

```txt
10:00–12:00
```

Requested:

```txt
10:00–12:00
```

Expected:

```txt
schedule_conflict
```

### 8.10 `TestSchedulingService_AssignJob_BoundaryBeforeAllowed`

Existing:

```txt
10:00–12:00
```

Requested:

```txt
08:00–10:00
```

Expected:

```txt
success
```

### 8.11 `TestSchedulingService_AssignJob_BoundaryAfterAllowed`

Existing:

```txt
10:00–12:00
```

Requested:

```txt
12:00–14:00
```

Expected:

```txt
success
```

### 8.12 `TestSchedulingService_AssignJob_DifferentTechnicianSameWindowAllowed`

Existing:

```txt
technician1
10:00–12:00
```

Requested:

```txt
technician2
10:00–12:00
```

Expected:

```txt
success
```

### 8.13 `TestSchedulingService_AssignJob_ConcurrentOverlapExactlyOneSuccess`

Given two unscheduled quotes:

```txt
quote 1
quote 2
```

Run two goroutines at the same time:

```txt
manager1 assigns quote1 to technician1 at 10:00
manager2 assigns quote2 to technician1 at 11:00
```

Expected:

```txt
success count = 1
schedule_conflict count = 1
jobs for technician1 overlapping 10:00–13:00 = 1
only one technician notification created
only one quote changed to scheduled
```

This test proves backend concurrency handling.

### 8.14 `TestSchedulingService_AssignJob_ConcurrentSameQuoteExactlyOneSuccess`

Given one unscheduled quote:

```txt
quote 1
```

Run two goroutines at the same time:

```txt
manager1 assigns quote1 to technician1 at 10:00
manager2 assigns quote1 to technician2 at 10:00
```

Expected:

```txt
success count = 1
quote_already_scheduled count = 1
jobs for quote 1 = 1
```

---

## 9. SchedulingService Reschedule Tests

File:

```txt
backend/internal/scheduling/scheduling_reschedule_test.go
```

### 9.1 `TestSchedulingService_RescheduleJob_SuccessSameTechnician`

Given:

```txt
manager1 created job1
technician1
10:00–12:00
status = scheduled
```

When:

```txt
reschedule to technician1 at 14:00–16:00
```

Expected:

```txt
job startsAt updated
job endsAt updated
technician unchanged
job_updated notification to technician1
jobs.changed to technician1
jobs.changed to manager1
```

### 9.2 `TestSchedulingService_RescheduleJob_SuccessDifferentTechnician`

Given:

```txt
job1 assigned to technician1
```

When:

```txt
manager1 reassigns to technician2 at 14:00
```

Expected:

```txt
job technicianId = 2
job_updated notification to technician2
job_updated notification to technician1
jobs.changed to technician1
jobs.changed to technician2
jobs.changed to manager1
```

### 9.3 `TestSchedulingService_RescheduleJob_ManagerCannotUpdateOtherManagersJob`

Given:

```txt
job1 created by manager1
actor = manager2
```

Expected:

```txt
forbidden
job unchanged
no notifications
no events
```

### 9.4 `TestSchedulingService_RescheduleJob_TechnicianActorForbidden`

Given:

```txt
actor = technician1
```

Expected:

```txt
forbidden
```

### 9.5 `TestSchedulingService_RescheduleJob_InvalidInput`

Cases:

1. `jobId = 0`.
2. `technicianId = 0`.
3. `startsAt` has seconds precision.

Expected:

```txt
invalid_input
no mutation
```

### 9.6 `TestSchedulingService_RescheduleJob_MissingJob`

Given:

```txt
jobId = 999
```

Expected:

```txt
not_found
```

### 9.7 `TestSchedulingService_RescheduleJob_MissingTargetTechnician`

Given:

```txt
technicianId = 999
```

Expected:

```txt
not_found
```

### 9.8 `TestSchedulingService_RescheduleJob_CompletedJobImmutable`

Given:

```txt
job1 status = completed
```

Expected:

```txt
completed_job_immutable
job unchanged
```

### 9.9 `TestSchedulingService_RescheduleJob_OverlapConflict`

Given:

```txt
job1: technician1, 08:00–10:00
job2: technician1, 10:00–12:00
```

When:

```txt
reschedule job1 to 11:00–13:00
```

Expected:

```txt
schedule_conflict
job1 unchanged
```

### 9.10 `TestSchedulingService_RescheduleJob_ExcludesCurrentJobFromOverlapCheck`

Given:

```txt
job1: technician1, 10:00–12:00
```

When:

```txt
reschedule job1 to same technician and same 10:00–12:00
```

Expected:

```txt
success
```

### 9.11 `TestSchedulingService_RescheduleJob_TargetTechnicianBoundaryAllowed`

Given:

```txt
job1: technician1, 08:00–10:00
job2: technician1, 10:00–12:00
```

When:

```txt
reschedule job1 to 12:00–14:00
```

Expected:

```txt
success
```

---

## 10. SchedulingService Completion Tests

File:

```txt
backend/internal/scheduling/scheduling_complete_test.go
```

### 10.1 `TestSchedulingService_CompleteJob_Success`

Given:

```txt
job1 assigned to technician1
status = scheduled
actor = technician1
fake clock = 2026-05-12T16:00:00Z
```

Expected job:

```txt
status = completed
completedAt = 2026-05-12T16:00:00Z
```

Expected DB state:

```txt
manager1 receives one job_completed notification
job remains attached to original quote, manager, technician
```

Expected events:

```txt
notification.created to manager1 user
jobs.changed to manager1 user
jobs.changed to technician1 user
```

### 10.2 `TestSchedulingService_CompleteJob_ManagerActorForbidden`

Given:

```txt
actor = manager1
```

Expected:

```txt
forbidden
job remains scheduled
```

### 10.3 `TestSchedulingService_CompleteJob_WrongTechnicianForbidden`

Given:

```txt
job1 assigned to technician1
actor = technician2
```

Expected:

```txt
forbidden
job remains scheduled
no notification
no events
```

### 10.4 `TestSchedulingService_CompleteJob_InvalidInput`

Given:

```txt
jobId = 0
```

Expected:

```txt
invalid_input
```

### 10.5 `TestSchedulingService_CompleteJob_MissingJob`

Given:

```txt
jobId = 999
```

Expected:

```txt
not_found
```

### 10.6 `TestSchedulingService_CompleteJob_AlreadyCompletedConflict`

Given:

```txt
job1 status = completed
completedAt = 2026-05-12T15:00:00Z
```

Expected:

```txt
job_already_completed
completedAt unchanged
no duplicate notification
```

---

## 11. NotificationService Tests

File:

```txt
backend/internal/notifications/notification_service_test.go
```

### 11.1 `TestNotificationService_ListNotifications_ReturnsOnlyActorNotifications`

Given:

```txt
notification A recipient = manager1 user
notification B recipient = technician1 user
```

When:

```go
ListNotifications(ctx, manager1)
```

Expected:

```txt
returns notification A only
```

### 11.2 `TestNotificationService_ListNotifications_SortedNewestFirst`

Given:

```txt
notification A created at 10:00
notification B created at 11:00
notification C created at 11:00 with higher id
```

Expected order:

```txt
C
B
A
```

Sort:

```txt
created_at DESC, id DESC
```

### 11.3 `TestNotificationService_MarkRead_Success`

Given:

```txt
notification1 recipient = manager1 user
read_at = null
fake clock = 2026-05-12T16:00:00Z
```

Expected:

```txt
readAt = 2026-05-12T16:00:00Z
```

### 11.4 `TestNotificationService_MarkRead_AlreadyReadIsIdempotent`

Given:

```txt
notification1 readAt = 2026-05-12T15:00:00Z
```

When:

```txt
mark read again
```

Expected:

```txt
success
readAt remains 2026-05-12T15:00:00Z
```

### 11.5 `TestNotificationService_MarkRead_WrongUserForbidden`

Given:

```txt
notification1 recipient = manager1 user
actor = technician1
```

Expected:

```txt
forbidden
readAt unchanged
```

### 11.6 `TestNotificationService_MarkRead_MissingNotification`

Given:

```txt
notificationId = 999
```

Expected:

```txt
not_found
```

### 11.7 `TestNotificationService_MarkRead_InvalidID`

Given:

```txt
notificationId = 0
```

Expected:

```txt
invalid_input
```

---

## 12. EventPublisher and WebSocketHub Tests

File:

```txt
backend/internal/ws/hub_test.go
```

### 12.1 `TestWebSocketHub_RegisterAndSendToUser`

Given:

```txt
user1 connection registered
```

When:

```txt
SendToUser(user1, notification.created)
```

Expected:

```txt
user1 receives event
```

### 12.2 `TestWebSocketHub_SendToUser_DoesNotSendToOtherUsers`

Given:

```txt
user1 connection
user2 connection
```

When:

```txt
SendToUser(user1, jobs.changed)
```

Expected:

```txt
user1 receives event
user2 receives nothing
```

### 12.3 `TestWebSocketHub_BroadcastToManagers`

Given:

```txt
manager1 connection
manager2 connection
technician1 connection
```

When:

```txt
BroadcastToRole(manager, quotes.changed)
```

Expected:

```txt
manager1 receives event
manager2 receives event
technician1 receives nothing
```

### 12.4 `TestWebSocketEndpoint_RejectsUnauthenticatedConnection`

When:

```txt
connect to /ws without cookie
```

Expected:

```txt
HTTP 401 before upgrade
```

### 12.5 `TestWebSocketEndpoint_AllowsAuthenticatedTechnician`

Given:

```txt
valid technician1 session cookie
```

Expected:

```txt
WebSocket connection succeeds
```

### 12.6 `TestWebSocketEndpoint_AssignmentSendsNotificationToTechnician`

Given:

```txt
technician1 is connected over WebSocket
manager1 assigns quote1 to technician1
```

Expected:

```txt
technician1 receives notification.created event
technician1 receives jobs.changed event
```

### 12.7 `TestWebSocketEndpoint_CompletionSendsNotificationToManager`

Given:

```txt
manager1 is connected over WebSocket
technician1 completes manager1's job
```

Expected:

```txt
manager1 receives notification.created event
manager1 receives jobs.changed event
```

---

## 13. HTTP API Handler Tests

File:

```txt
backend/internal/httpapi/api_handlers_test.go
```

These tests verify that the service contract is exposed correctly over HTTP.

### 13.1 Auth Handler Tests

Required tests:

1. `POST /api/auth/login` valid manager returns `200` and sets cookie.
2. `POST /api/auth/login` valid technician returns `200` and sets cookie.
3. `POST /api/auth/login` wrong password returns `401`.
4. `POST /api/auth/login` malformed JSON returns `400`.
5. `GET /api/me` without cookie returns `401`.
6. `GET /api/me` with valid cookie returns actor.
7. `POST /api/auth/logout` clears cookie.

### 13.2 Quote Handler Tests

Required tests:

1. `GET /api/quotes?status=unscheduled` as manager returns `200`.
2. `GET /api/quotes?status=unscheduled` as technician returns `403`.
3. `GET /api/quotes?status=invalid` as manager returns `400`.
4. `GET /api/quotes` as manager returns all quotes.

### 13.3 Technician Handler Tests

Required tests:

1. `GET /api/technicians` as manager returns `200`.
2. `GET /api/technicians` as technician returns `403`.

### 13.4 Job Handler Tests

Required tests:

1. `GET /api/jobs` as manager returns only manager jobs.
2. `GET /api/jobs` as technician returns only assigned jobs.
3. `POST /api/jobs` as manager succeeds with status `201`.
4. `POST /api/jobs` as technician returns `403`.
5. `POST /api/jobs` malformed JSON returns `400`.
6. `POST /api/jobs` invalid timestamp returns `400 invalid_input`.
7. `POST /api/jobs` overlap returns `409 schedule_conflict`.
8. `POST /api/jobs` duplicate quote returns `409 quote_already_scheduled`.
9. `PATCH /api/jobs/{id}/schedule` as owning manager succeeds.
10. `PATCH /api/jobs/{id}/schedule` as other manager returns `403`.
11. `PATCH /api/jobs/{id}/schedule` completed job returns `409 completed_job_immutable`.
12. `PATCH /api/jobs/{id}/complete` as assigned technician succeeds.
13. `PATCH /api/jobs/{id}/complete` as wrong technician returns `403`.
14. `PATCH /api/jobs/{id}/complete` as manager returns `403`.
15. `PATCH /api/jobs/{id}/complete` already-completed job returns `409 job_already_completed`.

### 13.5 Notification Handler Tests

Required tests:

1. `GET /api/notifications` returns only actor notifications.
2. `PATCH /api/notifications/{id}/read` as owner succeeds.
3. `PATCH /api/notifications/{id}/read` as other user returns `403`.
4. `PATCH /api/notifications/999/read` returns `404`.
5. `PATCH /api/notifications/0/read` returns `400`.

### 13.6 Error Shape Tests

For every representative error response, assert shape:

```json
{
  "error": {
    "code": "some_code",
    "message": "some message"
  }
}
```

Required representative cases:

1. `400 invalid_input`
2. `401 unauthorized`
3. `403 forbidden`
4. `404 not_found`
5. `409 schedule_conflict`
6. `409 quote_already_scheduled`
7. `409 completed_job_immutable`
8. `409 job_already_completed`

---

## 14. Frontend Test Specification

Use Vitest and React Testing Library unless the implementation chooses a compatible equivalent.

### 14.1 Frontend API Client Tests

File:

```txt
frontend/tests/apiClient.test.ts
```

Required tests:

1. API client uses `NEXT_PUBLIC_API_URL`.
2. API client sends `credentials: "include"`.
3. API client maps error response shape to readable error object.
4. API client serializes assignment request without `endsAt`.

### 14.2 Login Page Tests

File:

```txt
frontend/tests/login.test.tsx
```

Required tests:

1. Login page renders email input.
2. Login page renders password input.
3. Login page renders login button.
4. Seeded manager login button exists.
5. Seeded technician login button exists.
6. Error alert renders when login fails.

### 14.3 Manager Dashboard Tests

File:

```txt
frontend/tests/managerDashboard.test.tsx
```

Required tests:

1. Manager dashboard renders unscheduled quotes section.
2. Manager dashboard renders technician selector.
3. Manager dashboard renders start-time input.
4. Assign button is disabled until quote, technician, and start time are selected.
5. Successful assignment resets assignment form.
6. Conflict response displays visible MUI Alert.
7. Scheduled jobs render with status chip.
8. Reschedule controls render only for scheduled jobs.
9. `notification.created` WebSocket event triggers notifications refetch.
10. `jobs.changed` WebSocket event triggers jobs refetch.
11. `quotes.changed` WebSocket event triggers quotes refetch.

### 14.4 Technician Dashboard Tests

File:

```txt
frontend/tests/technicianDashboard.test.tsx
```

Required tests:

1. Technician dashboard renders jobs list.
2. Scheduled job renders Complete button.
3. Completed job does not render Complete button.
4. Completed job renders completed status chip.
5. Assignment notification renders in notifications panel.
6. Successful completion triggers jobs refetch.
7. `notification.created` WebSocket event triggers notifications refetch.
8. `jobs.changed` WebSocket event triggers jobs refetch.

### 14.5 Frontend Static Checks

Required commands:

```bash
npm run lint
npm run typecheck
npm test -- --run
```

---

## 15. End-to-End Acceptance Tests

Use Playwright.

File:

```txt
e2e/scheduling.spec.ts
```

### 15.1 `manager can assign quote to technician`

Flow:

1. Login as `manager1@brix.test`.
2. Go to `/manager`.
3. Confirm unscheduled quotes are visible.
4. Select Quote 1.
5. Select Tom Technician.
6. Select `startsAt = 2026-05-12T10:00` local equivalent.
7. Click Assign.
8. Confirm success message.
9. Confirm Quote 1 disappears from unscheduled list.
10. Confirm job appears in manager jobs list.

### 15.2 `technician sees assigned job and notification`

Flow:

1. Login as `technician1@brix.test`.
2. Go to `/technician`.
3. Confirm job for Acme Plumbing is visible.
4. Confirm notification panel includes assignment notification.

### 15.3 `technician completes job and manager sees notification`

Flow:

1. Login as `technician1@brix.test`.
2. Click Complete on scheduled job.
3. Confirm job status is completed.
4. Login as `manager1@brix.test`.
5. Confirm manager notification includes completed job.

### 15.4 `manager sees conflict error for overlapping job`

Flow:

1. Login as `manager1@brix.test`.
2. Assign Quote 1 to technician1 at 10:00.
3. Assign Quote 2 to technician1 at 11:00.
4. Confirm visible error: `Technician already has a job in that time window.`
5. Confirm Quote 2 remains unscheduled.

### 15.5 `manager can reschedule job`

Flow:

1. Login as `manager1@brix.test`.
2. Assign Quote 1 to technician1 at 10:00.
3. Reschedule the job to technician2 at 14:00.
4. Confirm job now shows Priya Technician.
5. Login as technician2.
6. Confirm job appears for technician2.

### 15.6 `other manager cannot reschedule job`

Flow:

1. Login as `manager1@brix.test`.
2. Assign Quote 1 to technician1 at 10:00.
3. Login as `manager2@brix.test`.
4. Confirm manager2 does not see manager1's job in normal jobs list.
5. Attempt direct API reschedule of manager1's job using manager2 session.
6. Confirm response is `403`.

---

## 16. Service Test Implementation Order

Implement tests and code in this order.

1. Database schema tests.
2. Database migrations and seed data.
3. Pure domain tests.
4. Pure domain logic.
5. AuthService tests.
6. AuthService implementation.
7. QuoteService tests.
8. QuoteService implementation.
9. TechnicianService tests.
10. TechnicianService implementation.
11. JobQueryService tests.
12. JobQueryService implementation.
13. SchedulingService assignment tests.
14. SchedulingService assignment implementation.
15. SchedulingService reschedule tests.
16. SchedulingService reschedule implementation.
17. SchedulingService completion tests.
18. SchedulingService completion implementation.
19. NotificationService tests.
20. NotificationService implementation.
21. WebSocketHub tests.
22. WebSocketHub implementation.
23. HTTP API handler tests.
24. HTTP API handlers.
25. Frontend API client tests.
26. Frontend API client implementation.
27. Login page tests.
28. Login page implementation.
29. Manager dashboard tests.
30. Manager dashboard implementation.
31. Technician dashboard tests.
32. Technician dashboard implementation.
33. E2E tests.
34. E2E fixes.
35. Compliance matrix.

Do not implement UI before scheduling service tests pass.

---

## 17. Phase Gates

### 17.1 Backend Domain Gate

Pass condition:

```bash
cd backend && go test ./internal/domain ./internal/scheduling -run 'TestCalculate|TestValidate|TestOverlaps'
```

### 17.2 Backend Service Gate

Pass condition:

```bash
just test-backend
```

Required passing areas:

1. Domain tests.
2. Schema tests.
3. Auth tests.
4. Quote tests.
5. Technician tests.
6. Job query tests.
7. Scheduling assignment tests.
8. Scheduling reschedule tests.
9. Scheduling completion tests.
10. Notification tests.
11. WebSocket tests.
12. HTTP handler tests.

### 17.3 Frontend Gate

Pass condition:

```bash
just test-frontend
```

Required passing areas:

1. Lint.
2. Typecheck.
3. API client tests.
4. Login tests.
5. Manager dashboard tests.
6. Technician dashboard tests.

### 17.4 E2E Gate

Pass condition:

```bash
just test-e2e
```

Required passing flows:

1. Assignment.
2. Technician notification.
3. Completion.
4. Manager notification.
5. Conflict error.
6. Reschedule.
7. Unauthorized manager update rejection.

### 17.5 Final Gate

Pass condition:

```bash
docker compose down -v
docker compose up --build
just test
git status
```

Expected final state:

```txt
all tests pass
no uncommitted changes
COMPLIANCE_MATRIX complete
```
