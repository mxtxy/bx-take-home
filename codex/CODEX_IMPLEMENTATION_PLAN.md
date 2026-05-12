# Codex Implementation Plan — Brix Service Scheduling & Notification System

## 0. Operating Mode

This plan is designed for Codex or any agentic coding assistant. It is intentionally phase-gated and test-first.

The rule for every phase is:

1. Read `SPEC.md` and `TEST_SPEC.md`.
2. Create a Git checkpoint before starting.
3. Write failing tests first.
4. Run the tests and confirm the expected failure.
5. Implement the smallest correct code to pass the tests.
6. Run phase-specific tests.
7. Run broader regression tests.
8. Update `docs/COMPLIANCE_MATRIX.md`.
9. Commit the passing phase.
10. Stop and report what changed.

Recommended Git pattern:

```bash
git status
git add .
git commit -m "checkpoint: before phase N"

# do phase work

just test-whatever
git add .
git commit -m "phase N: <summary>"
```

Do not start the next phase until the phase gate is green.

---

## 1. Phase 0 — Repository Bootstrap

### Objective

Create the repo skeleton, Docker Compose structure, documentation files, and test scaffolding. No business logic.

### Deliverables

1. `SPEC.md`
2. `TEST_SPEC.md`
3. `CODEX_IMPLEMENTATION_PLAN.md`
4. `docs/COMPLIANCE_MATRIX.md`
5. `.env.example`
6. `justfile`
7. `compose.yaml`
8. `compose.test.yaml`
9. Backend Go module skeleton.
10. Frontend Next.js/MUI skeleton.

### Codex Prompt

```txt
You are implementing Phase 0 only.

Read SPEC.md and TEST_SPEC.md.

Create the initial repository structure for a Docker Compose project with:
- Go backend
- Next.js frontend
- MySQL 8
- MUI
- test scaffolding

Add:
- docs/COMPLIANCE_MATRIX.md
- .env.example
- justfile
- compose.yaml
- compose.test.yaml placeholder
- backend Go module skeleton
- frontend Next.js/MUI skeleton

Do not implement business logic yet.

Run:
- docker compose config
- just --list

Stop after Phase 0 and report files changed plus command results.
```

### Tests and Commands

```bash
docker compose config
just --list
```

### Gate

Pass when Docker Compose config is valid and repository structure matches the specification.

---

## 2. Phase 1 — Database Migrations and Seed Data

### Objective

Implement MySQL schema and deterministic seed data.

### TDD Instruction

Write schema smoke tests before writing or finalizing migration SQL.

### Deliverables

1. `backend/cmd/migrate`
2. `backend/cmd/seed`
3. `backend/migrations/001_init.sql`
4. `backend/migrations/002_seed.sql`
5. `backend/internal/db`
6. Schema tests.
7. Seed tests.

### Codex Prompt

```txt
You are implementing Phase 1 only.

Read SPEC.md and TEST_SPEC.md.

First write schema smoke tests that fail until migrations exist.

Then implement:
- backend/cmd/migrate
- backend/cmd/seed
- backend/migrations/001_init.sql
- backend/migrations/002_seed.sql
- backend/internal/db

Seed the exact users and quotes from SPEC.md.

Run:
- docker compose up -d mysql
- just migrate
- just seed
- just test-backend

Update docs/COMPLIANCE_MATRIX.md.

Stop after Phase 1 and report test results.
```

### Tests

Required tests from `TEST_SPEC.md`:

1. Required tables exist.
2. `users.email` unique.
3. `managers.user_id` unique.
4. `technicians.user_id` unique.
5. `jobs.quote_id` unique.
6. Job foreign keys enforced.
7. Job two-hour window check enforced.
8. Seeded users exist.
9. Seeded quotes exist.

### Gate

```bash
docker compose up -d mysql
just migrate
just seed
just test-backend
```

Pass when migrations apply from an empty DB and schema/seed tests are green.

---

## 3. Phase 2 — Pure Domain Logic

### Objective

Implement deterministic domain logic with no database access.

### TDD Instruction

Write pure unit tests first.

### Deliverables

1. Domain types.
2. Error types.
3. Time-window calculation.
4. Overlap detection.
5. Input validation.

### Codex Prompt

```txt
You are implementing Phase 2 only.

Read SPEC.md and TEST_SPEC.md.

Write failing unit tests first for:
- two-hour window calculation
- start-time validation
- overlap detection
- assignment input validation
- reschedule input validation
- completion input validation

Then implement the pure domain code.

Do not write database logic in this phase.

Run:
- cd backend && go test ./internal/domain ./internal/scheduling

Update docs/COMPLIANCE_MATRIX.md.

Stop after Phase 2 and report test results.
```

### Gate

```bash
cd backend && go test ./internal/domain ./internal/scheduling
```

Pass when pure backend unit tests are green.

---

## 4. Phase 3 — AuthService

### Objective

Implement seeded-user login, signed session token creation, actor resolution, and role guards.

### TDD Instruction

Write tests for login, token validation, and token rejection first.

### Deliverables

1. `AuthService.Login`
2. `AuthService.ActorFromToken`
3. Token signer/parser.
4. Password verification.
5. Auth middleware.
6. Manager-only guard.
7. Technician-only guard.

### Codex Prompt

```txt
You are implementing Phase 3 only.

Read SPEC.md and TEST_SPEC.md.

Write tests first for:
- valid manager login
- valid technician login
- normalized email login
- invalid password
- unknown email
- tampered token rejection
- expired token rejection
- missing user rejection

Then implement AuthService and auth middleware.

Run:
- just test-backend

Update docs/COMPLIANCE_MATRIX.md.

Stop after Phase 3 and report test results.
```

### Gate

```bash
just test-backend
```

Pass when auth tests are green and manual curl login works.

Manual check:

```bash
curl -i \
  -X POST http://localhost:8080/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"manager1@brix.test","password":"password123"}'
```

---

## 5. Phase 4 — Query Services

### Objective

Implement read-only service layer for quotes, technicians, and jobs.

### TDD Instruction

Write service tests before implementation.

### Deliverables

1. `QuoteService.ListQuotes`
2. `TechnicianService.ListTechnicians`
3. `JobQueryService.ListJobs`

### Codex Prompt

```txt
You are implementing Phase 4 only.

Read SPEC.md and TEST_SPEC.md.

Write tests first for:
- QuoteService list/filter/authorization behavior
- TechnicianService list/sort/authorization behavior
- JobQueryService manager visibility
- JobQueryService technician visibility
- JobQueryService sorting
- JobQueryService joined display fields

Then implement the query services.

Run:
- just test-backend

Update docs/COMPLIANCE_MATRIX.md.

Stop after Phase 4 and report test results.
```

### Gate

```bash
just test-backend
```

Pass when query service tests are green.

---

## 6. Phase 5 — SchedulingService Assignment

### Objective

Implement transactional job assignment and backend-enforced conflict prevention.

### TDD Instruction

Write MySQL integration tests before implementation. Include concurrency tests.

### Deliverables

1. `SchedulingService.AssignJob`
2. Transaction wrapper.
3. Quote row lock.
4. Technician row lock.
5. Overlap check.
6. Job insert.
7. Quote status update.
8. Technician notification insert.
9. Post-commit domain events.

### Codex Prompt

```txt
You are implementing Phase 5 only.

Read SPEC.md and TEST_SPEC.md, especially conflict handling and transaction rules.

Write MySQL integration tests first for:
- successful assignment
- technician actor forbidden
- malformed manager actor forbidden
- invalid input
- missing quote
- missing technician
- quote already scheduled
- overlap conflict
- exact overlap conflict
- boundary before allowed
- boundary after allowed
- different technician same window allowed
- concurrent overlapping assignment with exactly one success
- concurrent same quote assignment with exactly one success

Then implement SchedulingService.AssignJob using MySQL transactions and SELECT ... FOR UPDATE.

Run:
- just test-backend

Update docs/COMPLIANCE_MATRIX.md.

Stop after Phase 5 and report test results.
```

### Gate

```bash
just test-backend
```

Pass when assignment tests, including concurrency tests, are green.

---

## 7. Phase 6 — SchedulingService Reschedule

### Objective

Implement transactional reschedule/reassignment for scheduled jobs.

### TDD Instruction

Write reschedule integration tests before implementation.

### Deliverables

1. `SchedulingService.RescheduleJob`
2. Job row lock.
3. Manager ownership check.
4. Completed-job immutability check.
5. Target technician row lock.
6. Overlap check excluding current job.
7. Job update.
8. Technician update notifications.
9. Post-commit domain events.

### Codex Prompt

```txt
You are implementing Phase 6 only.

Read SPEC.md and TEST_SPEC.md.

Write tests first for:
- same technician reschedule success
- different technician reassignment success
- manager cannot update another manager's job
- technician actor forbidden
- invalid input
- missing job
- missing target technician
- completed job immutable
- overlap conflict
- current job excluded from overlap check
- boundary allowed

Then implement SchedulingService.RescheduleJob.

Run:
- just test-backend

Update docs/COMPLIANCE_MATRIX.md.

Stop after Phase 6 and report test results.
```

### Gate

```bash
just test-backend
```

Pass when reschedule tests are green.

---

## 8. Phase 7 — SchedulingService Completion

### Objective

Implement technician job completion.

### TDD Instruction

Write completion integration tests before implementation.

### Deliverables

1. `SchedulingService.CompleteJob`
2. Technician authorization check.
3. Job row lock.
4. Completed-job conflict check.
5. Status update.
6. `completed_at` update from fake/injected clock.
7. Manager notification.
8. Post-commit domain events.

### Codex Prompt

```txt
You are implementing Phase 7 only.

Read SPEC.md and TEST_SPEC.md.

Write tests first for:
- successful completion
- manager actor forbidden
- wrong technician forbidden
- invalid input
- missing job
- already completed conflict

Then implement SchedulingService.CompleteJob.

Run:
- just test-backend

Update docs/COMPLIANCE_MATRIX.md.

Stop after Phase 7 and report test results.
```

### Gate

```bash
just test-backend
```

Pass when completion tests are green.

---

## 9. Phase 8 — NotificationService

### Objective

Implement notification listing and marking read.

### TDD Instruction

Write notification service tests before implementation.

### Deliverables

1. `NotificationService.ListNotifications`
2. `NotificationService.MarkRead`
3. Ownership enforcement.
4. Idempotent read behavior.

### Codex Prompt

```txt
You are implementing Phase 8 only.

Read SPEC.md and TEST_SPEC.md.

Write tests first for:
- list returns only actor notifications
- list sorted newest first
- mark read success
- already-read mark read is idempotent
- wrong user forbidden
- missing notification
- invalid notification ID

Then implement NotificationService.

Run:
- just test-backend

Update docs/COMPLIANCE_MATRIX.md.

Stop after Phase 8 and report test results.
```

### Gate

```bash
just test-backend
```

Pass when notification tests are green.

---

## 10. Phase 9 — WebSocketHub

### Objective

Implement authenticated WebSocket connections and event routing.

### TDD Instruction

Write WebSocket hub and endpoint tests first.

### Deliverables

1. `GET /ws`
2. Authenticated WebSocket upgrade.
3. In-process connection registry keyed by user ID.
4. Role-indexed connection registry for manager broadcasts.
5. `SendToUser`.
6. `BroadcastToRole`.
7. Disconnect cleanup.
8. Optional heartbeat/ping-pong.

### Codex Prompt

```txt
You are implementing Phase 9 only.

Read SPEC.md and TEST_SPEC.md WebSocket sections.

Write tests first for:
- hub register and send to user
- send to user does not send to other users
- broadcast to managers does not send to technicians
- unauthenticated /ws rejected
- authenticated technician /ws accepted
- assignment sends notification.created to technician
- completion sends notification.created to manager

Then implement the WebSocket hub and endpoint.

Do not move scheduling commands to WebSocket.

Run:
- just test-backend

Update docs/COMPLIANCE_MATRIX.md.

Stop after Phase 9 and report test results.
```

### Gate

```bash
just test-backend
```

Pass when WebSocket tests are green.

---

## 11. Phase 10 — HTTP API Handlers

### Objective

Expose all services through the specified REST API.

### TDD Instruction

Write HTTP handler tests before implementation.

### Deliverables

1. `GET /healthz`
2. `POST /api/auth/login`
3. `POST /api/auth/logout`
4. `GET /api/me`
5. `GET /api/quotes`
6. `GET /api/technicians`
7. `GET /api/jobs`
8. `POST /api/jobs`
9. `PATCH /api/jobs/{id}/schedule`
10. `PATCH /api/jobs/{id}/complete`
11. `GET /api/notifications`
12. `PATCH /api/notifications/{id}/read`
13. Error-shape mapping.
14. Cookie handling.
15. Post-service event publication.

### Codex Prompt

```txt
You are implementing Phase 10 only.

Read SPEC.md API section and TEST_SPEC.md HTTP handler section.

Write HTTP handler tests first for:
- auth handlers
- quote handlers
- technician handlers
- job handlers
- notification handlers
- representative error shapes

Then implement HTTP routes and handlers using chi.

Map service errors to the exact JSON error shape from SPEC.md.

Publish returned domain events only after the service call succeeds.

Run:
- just test-backend

Update docs/COMPLIANCE_MATRIX.md.

Stop after Phase 10 and report test results.
```

### Gate

```bash
just test-backend
```

Pass when HTTP handler tests are green.

---

## 12. Phase 11 — Frontend App Shell and API Client

### Objective

Implement the Next.js/MUI shell, API client, auth-aware routing, and login UI.

### TDD Instruction

Write frontend tests before implementation.

### Deliverables

1. MUI theme/provider setup.
2. API client using `credentials: "include"`.
3. Error mapping.
4. Login page.
5. Seeded login buttons.
6. Logout action.
7. Root route redirect logic.
8. Basic protected-route behavior.

### Codex Prompt

```txt
You are implementing Phase 11 only.

Read SPEC.md frontend section and TEST_SPEC.md frontend section.

Write frontend tests first for:
- API client uses credentials include
- API client maps error shape
- assignment request does not send endsAt
- login page fields render
- seeded login buttons render
- login error alert renders

Then implement:
- MUI app shell
- API client
- login page
- logout
- root redirect
- protected-route behavior

Run:
- cd frontend && npm run lint
- cd frontend && npm run typecheck
- cd frontend && npm test -- --run

Update docs/COMPLIANCE_MATRIX.md.

Stop after Phase 11 and report test results.
```

### Gate

```bash
cd frontend && npm run lint
cd frontend && npm run typecheck
cd frontend && npm test -- --run
```

Pass when frontend auth shell tests are green.

---

## 13. Phase 12 — Manager Dashboard

### Objective

Implement manager scheduling UI.

### TDD Instruction

Write manager dashboard tests before implementation.

### Deliverables

1. Fetch current user.
2. Fetch unscheduled quotes.
3. Fetch technicians.
4. Fetch manager jobs.
5. Fetch manager notifications.
6. Assignment form.
7. Conflict error alert.
8. Reschedule controls.
9. Notifications panel.
10. WebSocket-triggered refetches.

### Codex Prompt

```txt
You are implementing Phase 12 only.

Read SPEC.md manager dashboard section and TEST_SPEC.md manager dashboard tests.

Write tests first for:
- unscheduled quote list renders
- technician selector renders
- start-time input renders
- assign button disabled until valid
- successful assignment resets form
- conflict response displays alert
- scheduled jobs render with status chip
- reschedule controls render only for scheduled jobs
- WebSocket events trigger relevant refetches

Then implement manager dashboard.

Run:
- cd frontend && npm run lint
- cd frontend && npm run typecheck
- cd frontend && npm test -- --run

Update docs/COMPLIANCE_MATRIX.md.

Stop after Phase 12 and report test results.
```

### Gate

```bash
just test-frontend
```

Pass when manager UI tests, lint, and typecheck are green.

---

## 14. Phase 13 — Technician Dashboard

### Objective

Implement technician job lifecycle UI.

### TDD Instruction

Write technician dashboard tests before implementation.

### Deliverables

1. Fetch current user.
2. Fetch technician jobs.
3. Fetch technician notifications.
4. Complete job action.
5. Status display.
6. Notifications panel.
7. WebSocket-triggered refetches.

### Codex Prompt

```txt
You are implementing Phase 13 only.

Read SPEC.md technician dashboard section and TEST_SPEC.md technician dashboard tests.

Write tests first for:
- technician jobs list renders
- scheduled job shows Complete button
- completed job hides Complete button
- completed status chip renders
- notifications panel renders
- successful completion triggers jobs refetch
- WebSocket events trigger relevant refetches

Then implement technician dashboard.

Run:
- cd frontend && npm run lint
- cd frontend && npm run typecheck
- cd frontend && npm test -- --run

Update docs/COMPLIANCE_MATRIX.md.

Stop after Phase 13 and report test results.
```

### Gate

```bash
just test-frontend
```

Pass when technician UI tests, lint, and typecheck are green.

---

## 15. Phase 14 — End-to-End Tests

### Objective

Prove the complete system works through the browser.

### TDD Instruction

Write E2E tests before polishing implementation.

### Deliverables

1. Playwright setup.
2. Manager assignment flow.
3. Technician assignment notification flow.
4. Technician completion flow.
5. Manager completion notification flow.
6. Conflict error flow.
7. Reschedule flow.
8. Unauthorized manager update rejection flow.

### Codex Prompt

```txt
You are implementing Phase 14 only.

Read SPEC.md and TEST_SPEC.md E2E sections.

Write Playwright tests for:
- manager assigns quote to technician
- technician sees assigned job and notification
- technician completes job
- manager sees completion notification
- manager sees conflict error for overlapping job
- manager can reschedule job
- other manager cannot reschedule job

Then fix implementation issues discovered by E2E tests.

Run:
- just test-e2e
- just test

Update docs/COMPLIANCE_MATRIX.md.

Stop after Phase 14 and report test results.
```

### Gate

```bash
just test-e2e
just test
```

Pass when full system tests are green.

---

## 16. Phase 15 — README, Compliance Matrix, and Final Submission Pass

### Objective

Make the repo reviewer-ready.

### Deliverables

1. Complete `README.md`.
2. Complete `docs/COMPLIANCE_MATRIX.md`.
3. Complete research notes.
4. Final test pass.
5. Fresh Docker Compose startup pass.

### Required README Sections

1. Overview.
2. Tech stack.
3. Running locally.
4. Seeded users.
5. User flows.
6. API summary.
7. Conflict handling.
8. Concurrency handling.
9. Notifications.
10. WebSockets.
11. Auth.
12. Tests.
13. Trade-offs.
14. What I would do next.
15. AI usage.

### Codex Prompt

```txt
You are implementing Phase 15 only.

Read SPEC.md, TEST_SPEC.md, CODEX_IMPLEMENTATION_PLAN.md, docs/COMPLIANCE_MATRIX.md, and current code.

Write README.md with:
- overview
- stack
- setup
- seeded users
- run commands
- test commands
- conflict handling
- concurrency handling
- notification design
- WebSocket trade-offs
- auth trade-offs
- production improvements
- AI usage statement

Finalize docs/COMPLIANCE_MATRIX.md.

Run:
- just test
- docker compose down -v
- docker compose up --build

Stop after Phase 15 and report final test results.
```

### Final Gate

```bash
just test
docker compose down -v
docker compose up --build
git status
```

Expected:

1. Full test suite passes.
2. Fresh Docker Compose startup works.
3. No uncommitted changes.
4. README is complete.
5. Compliance matrix is complete.

---

## 17. Superpowers Research Plan

This plan prevents guessing. Each research item must produce a short note before implementation.

Create these files:

```txt
docs/research/00-tooling.md
docs/research/01-mysql-locking.md
docs/research/02-auth.md
docs/research/03-websockets.md
docs/research/04-frontend.md
docs/research/05-e2e.md
```

Each file must use this format:

```md
# Research: <topic>

## Question

What exact implementation detail are we validating?

## Source of Truth

Official docs, package docs, current code, or this spec.

## Decision

The implementation choice.

## Risks

Known failure modes.

## Tests Added

List the tests that prove the decision.

## Follow-Up

Deferred production improvements.
```

### 17.1 Tooling and Docker Compose

Questions:

1. Does `compose.yaml` define `mysql`, `backend`, and `frontend` correctly?
2. Are health checks present?
3. Can migrations run against an empty DB?
4. Can `docker compose down -v` followed by `docker compose up --build` produce a working app?

Tests:

```bash
docker compose config
docker compose down -v
docker compose up --build
```

### 17.2 MySQL Locking and Concurrency

Questions:

1. Does `SELECT ... FOR UPDATE` lock the technician row during scheduling?
2. Does locking the technician row serialize overlapping assignment attempts?
3. Does the unique `quote_id` constraint prevent duplicate quote scheduling?
4. Does the concurrent assignment test fail without locking and pass with locking?

Tests:

1. Concurrent overlapping assignment test.
2. Concurrent same quote assignment test.
3. Duplicate quote scheduling test.
4. Boundary overlap tests.

### 17.3 Auth

Questions:

1. Does login set the HttpOnly cookie?
2. Does `/api/me` derive identity from the cookie?
3. Do manager-only and technician-only guards work?
4. Does WebSocket auth use the same cookie?

Tests:

1. Login success.
2. Login failure.
3. Role guard tests.
4. `/ws` unauthenticated rejection.
5. `/ws` authenticated connection.

### 17.4 WebSockets

Questions:

1. Are WebSocket events emitted only after DB commit?
2. Are messages scoped to authorized users?
3. Does assignment notify the technician?
4. Does completion notify the manager?
5. Does the frontend refetch after event receipt?

Tests:

1. Assignment WebSocket event test.
2. Completion WebSocket event test.
3. Hub role broadcast tests.
4. Frontend refresh tests.

### 17.5 Frontend and MUI

Questions:

1. Does the frontend send API requests with credentials included?
2. Does login redirect by role?
3. Does manager UI show unscheduled quotes and technicians?
4. Does technician UI show assigned jobs?
5. Are conflict errors visible to the user?

Tests:

1. API client credentials test.
2. Login page test.
3. Manager dashboard tests.
4. Technician dashboard tests.

### 17.6 End-to-End Flow

Questions:

1. Can a reviewer run the system from Docker Compose?
2. Can a manager assign a quote through the browser?
3. Can a technician complete the job through the browser?
4. Are notifications visible?
5. Is conflict prevention observable through the UI?

Tests:

1. Playwright happy path.
2. Playwright conflict path.
3. Playwright reschedule path.
4. Playwright unauthorized update path.

---

## 18. Final Submission Checklist

Before sending the GitHub or GitLab link:

```bash
docker compose down -v
docker compose up --build
just test
git status
```

Final checklist:

1. Fresh Docker Compose startup works.
2. Frontend loads at `localhost:3000`.
3. Backend health check passes.
4. Manager login works.
5. Technician login works.
6. Assignment works.
7. Conflict rejection works.
8. Completion works.
9. Notifications work.
10. WebSocket refresh works.
11. README is complete.
12. Compliance matrix is complete.
13. Tests are green.
14. No uncommitted changes.

---

## 19. One-Shot Codex Starter Prompt

Use this only to bootstrap the repository. After that, use phase-specific prompts.

```txt
Read SPEC.md, TEST_SPEC.md, and CODEX_IMPLEMENTATION_PLAN.md.

Implement this project test-first.

Do not implement production code before writing the relevant failing tests.

Start with Phase 0 only:
- repo skeleton
- Docker Compose files
- justfile
- backend module skeleton
- frontend skeleton
- compliance matrix

For every later phase:
- write tests first
- run tests and confirm expected failure
- implement the smallest correct code
- rerun tests
- update docs/COMPLIANCE_MATRIX.md
- stop and report results

Do not skip phases.
Do not implement UI before scheduling service tests pass.
Do not perform scheduling commands over WebSocket.
```
