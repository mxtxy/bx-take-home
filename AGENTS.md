# Repository Guidelines

## Project Structure & Module Organization

The project contract lives in `codex/SPEC.md`, test expectations in `codex/TEST_SPEC.md`, and phased implementation guidance in `codex/CODEX_IMPLEMENTATION_PLAN.md`. Keep future implementation aligned with this layout:

- `backend/`: Go 1.26 service using `net/http`, `chi`, MySQL, migrations, seed commands, and backend tests.
- `frontend/`: Next.js 16 App Router application with TypeScript, React 19, Material UI, Vitest, and Testing Library.
- `e2e/`: Playwright browser flows that exercise the Compose stack.
- `docs/`: project-facing documentation such as `COMPLIANCE_MATRIX.md`.
- `compose.yaml` and `compose.test.yaml`: local runtime and MySQL test orchestration.
- `flake.nix` and `flake.lock`: pinned development shell for Go, Node, Docker Compose, MySQL tools, and Playwright browser support.

Place backend tests beside the package they cover using Go's `*_test.go` convention. Keep frontend tests close to the components or flows they validate.

## Build, Test, and Development Commands

Use the Nix flake for development so Go, Node, Docker Compose, MySQL client tools, and Playwright browser dependencies are consistent:

- `nix develop`: enter the project development shell before running backend, frontend, Docker, or test commands.
- `npm ci`: install root Playwright dependencies after entering the shell if `node_modules/` is missing.
- `cd frontend && npm ci`: install frontend dependencies after entering the shell if `frontend/node_modules/` is missing.

Expected root commands inside `nix develop`:

- `just up`: start the Docker Compose stack with MySQL, migrations, seed data, backend, and frontend.
- `just down`: stop the Docker Compose stack.
- `just reset`: stop the stack and remove database volumes.
- `just migrate`: run backend migrations against the configured database.
- `just seed`: load deterministic seed data.
- `just test`: run backend, frontend, and E2E tests.
- `just test-backend`: run `go test ./...`.
- `just test-frontend`: run linting, type checks, and frontend tests.
- `just test-e2e`: start Docker Compose and run Playwright.
- `docker compose up --build`: run the local stack.
- `docker compose down -v`: stop services and remove database volumes.

Docker itself still needs a running host daemon; the flake provides the CLI and Compose plugin, not the daemon service.

## Coding Style & Naming Conventions

Use idiomatic Go formatting with `gofmt` and small packages under `backend/internal/...`. Prefer explicit names such as `AuthService`, `SchedulingService`, `JobRepository`, and `NotificationPublisher`. Frontend code should use TypeScript, React function components, MUI components, and filenames that match the component or route.

Keep API timestamps as RFC3339 strings, store backend times in UTC, and preserve the fixed two-hour scheduling window rule.

## Testing Guidelines

Implementation is test-first. Write failing tests before feature code, then make the smallest passing change. Use real MySQL integration tests where schema, transactions, locks, or persistence matter. Use deterministic fixtures and the fixed test time `2026-05-12T00:00:00Z`.

Backend tests should assert service error codes, not only message strings. Frontend tests should cover rendering, API behavior, and dashboard controls. E2E tests validate browser flows through Docker Compose.

After implementing any feature or bug fix, coverage should remain at 100%. Keep backend internal production package statement coverage at 100.0%, and keep frontend statement, branch, function, and line coverage at 100%.

Seeded local users all use the password `password123`: `manager1@brix.test`, `manager2@brix.test`, `technician1@brix.test`, and `technician2@brix.test`.

## Commit & Pull Request Guidelines

The current Git history only contains the initial `Create repo` commit. Continue with short, imperative subjects, for example `Add backend migrations` or `Implement scheduling service`.

For pull requests, include a concise description, tests run, linked issue or phase, and screenshots for UI changes. Update `docs/COMPLIANCE_MATRIX.md` whenever implementation changes spec coverage.

## Security & Configuration Tips

Do not commit real secrets. Add local configuration through `.env` files based on `.env.example`. Session signing must use `JWT_SECRET`, and local cookies should remain HttpOnly with `SameSite=Lax` as specified.
