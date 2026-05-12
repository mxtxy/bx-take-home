# Research: MySQL Locking

## Question

How does the scheduler prevent concurrent double-booking and duplicate quote scheduling?

## Source of Truth

`codex/SPEC.md`, MySQL `SELECT ... FOR UPDATE`, and service integration tests.

## Decision

Assignment and reschedule transactions lock quote or job rows plus the target technician row before checking overlaps.

## Risks

Skipping row locks allows two transactions to pass the overlap check before either commits.

## Tests Added

Concurrent assignment tests cover overlapping jobs and duplicate quote scheduling.

## Follow-Up

For higher write volumes, add more explicit isolation-level documentation and deadlock retry policy.
