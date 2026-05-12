# Compliance Matrix

This maps the implementation to `codex/SPEC.md` and `codex/TEST_SPEC.md`.

README usage commands are maintained in `README.md`; this matrix focuses on
implementation evidence against the spec and test plan.

| Area | Status | Evidence |
|---|---|---|
| Repository bootstrap | Complete | `justfile`, `.env.example`, `compose.yaml`, `compose.test.yaml`, `backend/`, `frontend/`, `e2e/`, `docs/` |
| Required root commands | Complete | `justfile` defines `test`, `test-backend`, `test-backend-coverage`, `test-frontend`, `test-e2e`, `up`, `down`, `reset`, `migrate`, `seed` |
| Coverage gates | Complete | `backend/scripts/check-coverage.sh` enforces 100.0% statement coverage for backend internal production packages; `frontend/vitest.config.ts` enforces 100% statements, branches, functions, and lines for `app/`, `components/`, and `lib/` |
| Docker Compose runtime | Complete | `compose.yaml` defines MySQL 8, backend, frontend, migrate, and seed services with ports 3307, 8080, and 3000 |
| Database schema | Complete | `backend/migrations/001_init.sql`; tests in `backend/internal/db/schema_test.go` cover required tables, unique keys, foreign keys, two-hour window checks |
| Seed data | Complete | `backend/migrations/002_seed.sql`; `TestSeed_SeededUsersExist`, `TestSeed_SeededQuotesExist` verify seeded users, bcrypt hashes, and five unscheduled quotes |
| Domain logic | Complete | `backend/internal/domain/domain.go`; `backend/internal/domain/time_window_test.go` covers two-hour windows, minute alignment, overlap boundaries, and input validation |
| Auth service | Complete | `backend/internal/auth/service.go`; `auth_service_test.go` covers manager/technician login, email normalization, invalid credentials, tampered/expired tokens, and missing users |
| Query services | Complete | `quotes`, `technicians`, and `jobs` packages cover manager/technician authorization, filtering, sorting, DTO fields, and empty JSON array behavior |
| Assignment scheduling | Complete | `SchedulingService.AssignJob`; tests cover success, forbidden actors, invalid input, missing quote/technician, duplicate quote, overlap conflicts, boundary windows, different technicians, and concurrency races |
| Rescheduling | Complete | `SchedulingService.RescheduleJob`; tests cover same/different technician, other-manager rejection, forbidden technician actors, invalid input, missing rows, completed-job immutability, overlap conflict, current-job exclusion, and boundary allowance |
| Completion | Complete | `SchedulingService.CompleteJob`; tests cover success, manager forbidden, wrong technician forbidden, invalid input, missing job, and already-completed conflict |
| Notifications | Complete | `backend/internal/notifications/service.go`; tests cover actor-only listing, newest-first sorting, empty arrays, mark-read success, idempotent mark-read, wrong user, missing notification, invalid ID |
| WebSockets | Complete | `backend/internal/ws`; tests cover user delivery, role broadcast, unauthenticated rejection, authenticated technician connection, assignment events, completion events |
| HTTP API | Complete | `backend/internal/httpapi/router.go`; `api_handlers_test.go` covers auth, quote, technician, job, notification, health handlers, cookies, status codes, and error JSON shape |
| Frontend API client | Complete | `frontend/lib/api.ts`; tests cover API base URL, `credentials: "include"`, server error shape mapping, and assignment requests without `endsAt` |
| Login UI | Complete | `frontend/app/login/page.tsx`; tests cover email/password inputs, login button, seeded account shortcuts, and error alert |
| Manager dashboard | Complete | `frontend/app/manager/page.tsx`; tests cover unscheduled quotes, technician selector, start-time input, assign button state, assignment reset, conflict alert, scheduled-job chip, reschedule controls, and WebSocket-triggered refetches |
| Technician dashboard | Complete | `frontend/app/technician/page.tsx`; tests cover job rendering, complete button behavior, status chips, notifications, completion refetch, and WebSocket-triggered refetches |
| E2E flows | Complete | `e2e/scheduling.spec.ts` covers manager assignment, technician job/notification view, technician completion with manager notification, overlap conflict, reschedule, and other-manager rejection |
| Repository references | Complete | Go module and imports use `github.com/mxtxy/bx-take-home/backend`; Git remote is `https://github.com/mxtxy/bx-take-home.git` |

## Verification Commands

Use the setup and command flow from `README.md`:

```bash
nix develop
npm ci
cd frontend && npm ci && cd ..
just test
```

Useful targeted checks:

```bash
just test-backend
just test-frontend
just test-e2e
```
