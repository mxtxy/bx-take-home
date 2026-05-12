# Research: Auth

## Question

How are seeded users authenticated and resolved for role-based authorization?

## Source of Truth

`codex/SPEC.md`, bcrypt package docs, and auth service/handler tests.

## Decision

Use seeded bcrypt hashes and an HMAC-signed HttpOnly session cookie containing user ID, role, and expiry.

## Risks

The signed token is intentionally simple for this take-home and lacks production JWT/key rotation features.

## Tests Added

Login, invalid credential, tampered token, expired token, missing user, and protected-route tests.

## Follow-Up

Production hardening would add CSRF defenses, key rotation, and stricter cookie settings behind HTTPS.
