# Research: WebSockets

## Question

How are live refresh events authenticated and routed?

## Source of Truth

`codex/SPEC.md`, gorilla/websocket package docs, and hub tests.

## Decision

Use the same session cookie for `/ws`, keep an in-process hub keyed by user and role, and send event IDs only.

## Risks

The in-process hub works for one backend instance only.

## Tests Added

Hub routing tests and WebSocket endpoint tests.

## Follow-Up

Production multi-instance deployments need Redis Pub/Sub, NATS, or another fanout layer.
