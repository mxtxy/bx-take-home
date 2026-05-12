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
| Database schema | Complete | `backend/migrations/001_init.sql`; tests in `backend/internal/db/schema_test.go` cover organizations, tenant-scoped users/quotes/jobs/notifications, sessions, technician availability rules, schedule audit logs, unique keys, foreign keys, and two-hour window checks |
| Seed data | Complete | `backend/migrations/002_seed.sql`; seed tests verify seeded organization, users, bcrypt hashes, five unscheduled quotes, and weekday technician availability interpreted as Australia/Sydney local time |
| Domain logic | Complete | `backend/internal/domain/domain.go`; `backend/internal/domain/time_window_test.go` covers two-hour windows, minute alignment, overlap boundaries, quote creation validation, technician availability errors, and input validation |
| Auth service | Complete | `backend/internal/auth/service.go`; `auth_service_test.go` covers manager/technician login, email normalization, invalid credentials, tampered/expired tokens, organization mismatch, revocable sessions, logout revocation, missing users, and session failures |
| Organization tenancy | Complete | `organization_id` is carried through actors, DTOs, schema, query services, scheduling, notifications, HTTP auth, and WebSocket events; backend tests cover cross-organization data isolation |
| Query services | Complete | `quotes`, `technicians`, and `jobs` packages cover manager/technician authorization, tenant filtering, sorting, DTO fields, and empty JSON array behavior |
| Quote creation | Complete | `quotes.Service.CreateQuote`, `POST /api/quotes`, manager dashboard quote form, and E2E quote creation flow are covered by backend, HTTP, frontend, and Playwright tests |
| Assignment scheduling | Complete | `SchedulingService.AssignJob`; tests cover success, forbidden actors, invalid input, missing quote/technician, duplicate quote, overlap conflicts, Sydney-local technician availability rules, boundary windows, different technicians, tenant isolation, audit logs, and concurrency races |
| Rescheduling | Complete | `SchedulingService.RescheduleJob`; tests cover same/different technician, other-manager rejection, forbidden technician actors, invalid input, missing rows, completed-job immutability, overlap conflict, Sydney-local technician availability rules, current-job exclusion, tenant isolation, and audit logs |
| Completion | Complete | `SchedulingService.CompleteJob`; tests cover success, manager forbidden, wrong technician forbidden, invalid input, missing job, already-completed conflict, tenant isolation, and audit logs |
| Notifications | Complete | `backend/internal/notifications/service.go`, `frontend/components/AppShell.tsx`; tests cover actor-only and organization-only listing, unread counts, newest-first sorting, empty arrays, mark-read success, idempotent mark-read, wrong user, missing notification, invalid ID, and shared drawer mark-as-read behavior |
| WebSockets | Complete | `backend/internal/ws`; tests cover user delivery, organization-scoped role broadcast, unauthenticated rejection, authenticated technician connection, assignment events, completion events, and reconnect behavior through the frontend API client and E2E tests |
| HTTP API | Complete | `backend/internal/httpapi/router.go`; `api_handlers_test.go` covers auth, quote, technician, job, notification, health handlers, revocable cookies, status codes, error JSON shape, request IDs, and `Traceparent` propagation/generation |
| Frontend API client | Complete | `frontend/lib/api.ts`; tests cover API base URL, `credentials: "include"`, server error shape mapping, quote creation, unread notification responses, assignment requests without `endsAt`, and WebSocket reconnect cleanup |
| Joy UI shell and theming | Complete | `frontend/components/AppProviders.tsx`, `frontend/components/AppShell.tsx`, `frontend/app/layout.tsx`; tests cover Joy provider wiring, default Joy theme usage without custom accents, reference-style sidebar navigation, mobile header open/close behavior, dark-mode toggle, role-scoped navigation, user display, notification drawer, unread count, notification timestamps, mark-as-read controls, and logout |
| Login UI | Complete | `frontend/app/login/page.tsx`; tests cover email/password inputs, login button, seeded account shortcuts, and error alert with Joy UI controls |
| Manager pages | Complete | `frontend/app/manager/quotes/page.tsx`, `frontend/app/manager/assign/page.tsx`, `frontend/app/manager/jobs/page.tsx`, `frontend/app/manager/ManagerWorkspace.tsx`, `frontend/components/SchedulePicker.tsx`; tests cover quote creation and existing quote viewing on the quotes page, quote assignment on the assign page, all-job viewing/rescheduling on the jobs page, Joy dropdown placeholders and option selection, calendar scheduling controls, start-time input compatibility, assign button state, assignment reset, popup success/error notifications, scheduled-job chip, reschedule dialog behavior, focus refresh for stale data, and WebSocket-triggered notification/job/quote refetches |
| Technician dashboard | Complete | `frontend/app/technician/page.tsx`; tests cover assigned job rendering, complete button behavior, status chips, notification drawer, mark-as-read, popup success/error notifications, completion refetch, focus refresh for stale data, and WebSocket-triggered refetches in the shared Joy shell |
| E2E flows | Complete | `e2e/scheduling.spec.ts` covers manager assignment on `/manager/assign`, quote creation on `/manager/quotes`, manager job viewing/reschedule on `/manager/jobs`, technician job/notification drawer view, notification mark-as-read, technician completion with manager notification drawer, overlap conflict, reschedule, other-manager rejection, stale UI refresh on focus, and WebSocket reconnect behavior |
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
