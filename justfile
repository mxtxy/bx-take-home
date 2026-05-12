set shell := ["bash", "-cu"]

_default:
    @just --list

# Show available project commands
help:
    @just --list

# Start the Docker Compose stack
up:
    docker compose up --build

# Stop the Docker Compose stack
down:
    docker compose down

# Stop the stack and remove volumes
reset:
    docker compose down -v

# Run backend migrations
migrate:
    cd backend && go run ./cmd/migrate

# Run deterministic seed data
seed:
    cd backend && go run ./cmd/seed

# Run backend, frontend, and E2E tests
test: test-backend test-frontend test-e2e

# Run Go backend tests
test-backend:
    docker compose -f compose.test.yaml up -d --wait mysql
    cd backend && CGO_ENABLED=0 go test ./...
    backend/scripts/check-coverage.sh /tmp/bx-backend.cover

# Run Go backend coverage gate
test-backend-coverage:
    backend/scripts/check-coverage.sh /tmp/bx-backend.cover

# Run frontend lint, typecheck, and tests
test-frontend:
    cd frontend && npm run lint
    cd frontend && npm run typecheck
    cd frontend && npm test -- --run --coverage

# Start Compose and run Playwright tests
test-e2e:
    docker compose down -v
    docker compose up -d --build
    npx playwright install chromium
    npx playwright test
