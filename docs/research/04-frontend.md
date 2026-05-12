# Research: Frontend

## Question

How should the browser app fetch data and react to live events?

## Source of Truth

`codex/SPEC.md`, Next.js App Router, MUI, Vitest, and React Testing Library.

## Decision

Use HTTP API calls with `credentials: "include"` and WebSocket events as refetch triggers.

## Risks

The UI depends on backend authorization and does not attempt to resolve scheduling conflicts locally.

## Tests Added

API client, login, manager dashboard, and technician dashboard tests.

## Follow-Up

Production UX would add better loading states, accessibility audits, and richer date/time handling.
