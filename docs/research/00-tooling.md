# Research: Tooling and Docker Compose

## Question

Does the repository define repeatable commands and a Docker Compose topology for MySQL, backend, and frontend?

## Source of Truth

`codex/SPEC.md`, `codex/TEST_SPEC.md`, Docker Compose documentation, and command output.

## Decision

Use root `justfile` recipes and `compose.yaml` with `mysql`, `migrate`, `seed`, `backend`, and `frontend`.

## Risks

The backend and tests depend on MySQL readiness; health checks and Compose dependencies keep startup deterministic.

## Tests Added

`docker compose config` and `just --list`.

## Follow-Up

For production, split migration/seed execution from app startup and manage secrets externally.
