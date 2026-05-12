# Brix Scheduler Take Home Exercise

## Project Setup

Prerequisites:

- Nix with flakes enabled, for the pinned development shell.
- Docker with a running Docker daemon, for MySQL and the local application stack.

Clone the repository and enter the project:

```sh
git clone https://github.com/mxtxy/bx-take-home
cd bx-take-home
```

Enter the development shell and install Node dependencies:

```sh
nix develop
npm ci
cd frontend && npm ci && cd ..
```

Start the full application stack:

```sh
just up
```

The Compose stack starts MySQL 8.4, runs migrations, loads deterministic seed data, starts the Go backend, and starts the Next.js frontend. Once the services are healthy, open:

- Frontend: http://localhost:3000
- Backend health check: http://localhost:8080/healthz

For non-development environments, set `APP_ENV=production` and provide a unique `JWT_SECRET` of at least 32 characters. The development default is only accepted when `APP_ENV` is `development`, `test`, or `local`.

Seeded login accounts all use password `password123`:

- `manager1@brix.test`
- `manager2@brix.test`
- `technician1@brix.test`
- `technician2@brix.test`

Useful development commands:

```sh
just down          # stop the stack
just reset         # stop the stack and remove database volumes
just test-backend  # run Go tests and backend coverage gate
just test-frontend # run lint, typecheck, and Vitest coverage gate
just test-e2e      # run Playwright against Docker Compose
just test          # run all test suites
```

## Decisions and Trade offs

I approached this exercise by treating the scheduling rules as backend invariants first, then building the UI around those constraints. The most important requirement was preventing technicians from being double booked, especially when multiple managers attempt to schedule work at the same time.

Although Phoenix LiveView would have been a strong fit for this sort of workflow, I chose a stack that better matched the role and expected production environment:

- Go HTTP API for transactional backend commands 
- MySQL 8 as the single source of truth
- Next.js, React, TypeScript, and MUI for the frontend
- WebSockets for live update signals
- Docker Compose for local development and review

The core design principle was that the backend is authoritative. The frontend can make scheduling easier for users, but it is not trusted to enforce correctness. Assignment, rescheduling, completion, authorisation, and conflict prevention are all handled by the Go backend.

### Backend and Scheduling Correctness

I used Go for the backend because it keeps the business logic explicit, testable, and easy to reason about. The scheduling service owns the core workflows:

- assigning a quote to a technician 
- rescheduling or reassigning a job
- completing a job
- creating notifications
- producing post commit events for WebSocket delivery

The most important rule is technician availability. A requested job conflicts with an existing job when:

```sql
existing.starts_at < requested.ends_at 
AND existing.ends_at > requested.starts_at
```

The backend computes the 2 hour job window from the submitted start time. The client does not submit `endsAt`.

To prevent race conditions, scheduling writes run inside MySQL transactions. The assignment flow locks the relevant quote and technician rows, checks for overlapping jobs, inserts the job, updates the quote, creates a notification, and commits. This ensures that concurrent manager requests cannot both schedule overlapping work for the same technician.  

I also used database constraints to protect the model:

* foreign keys for jobs, quotes, managers, technicians, users, and notifications
* a unique constraint on `jobs.quote_id` so a quote cannot be scheduled twice
* a check constraint to ensure job windows are exactly two hours  
* indexes for technician schedule lookups and notification queries

This means the database still protects important invariants even if an application level mistake is introduced later.

### Frontend

The frontend uses Next.js, React, TypeScript, and MUI. I kept the UI intentionally simple: managers can view unscheduled quotes, choose a technician, select a start time, assign or reschedule jobs, and see notifications. Technicians can view assigned jobs, complete them, and see notifications.  

I chose forms and lists rather than a full calendar because the timebox was 3–5 hours and the key evaluation point was scheduling correctness, not calendar UX. The UI still demonstrates the required flows without spending unnecessary time on drag and drop or complex visual scheduling.

MUI provided production like components quickly without requiring custom design work.

### WebSockets and Notifications

Notifications are persisted in MySQL rather than existing only in memory. This means users can still see notifications if they were offline or disconnected when an event occurred.  

WebSockets are used for live refresh signals, not for business commands. Scheduling actions are performed through HTTP endpoints because they need validation, authorisation, transaction handling, and clear HTTP error responses.

WebSocket payloads are intentionally minimal. They contain event types and IDs, not full domain objects. When the frontend receives an event, it refetches the relevant data over HTTP. 

For example:

* `notification.created` triggers a notification refetch
* `jobs.changed` triggers a job list refetch 
* `quotes.changed` triggers an unscheduled quote refetch for managers

This keeps client state simple and avoids duplicating business rules in WebSocket message handling.

The WebSocket hub is in process, which is appropriate for this take home exercise. In production, a multi instance deployment would need Redis Pub/Sub, NATS, or another fanout mechanism.

### Authentication and Authorisation 

Auth is deliberately simple. The app uses seeded users, email/password login, and an HttpOnly session cookie. This is enough to demonstrate role based access control without spending the timebox on user registration, password reset, refresh tokens, or account management.

Authorisation is enforced server side:  

* only managers can assign and reschedule jobs
* only the manager who created a job can reschedule it
* only technicians can complete jobs
* only the assigned technician can complete a job 
* users can only read and update their own notifications

In production, I would extend this with stronger session management, CSRF protection, password reset, audit logs, tenancy, and organisation level authorisation.

### Testing and Test Driven Development

Testing was central to the implementation. Before building the system, I used GPT-5.5 Pro with extended reasoning to help produce a comprehensive implementation spec, data spec, API contract, and test specification. I then used Codex to implement the project in a test driven sequence: write failing tests first, implement the smallest correct change, then rerun the relevant test suite.

This approach was valuable because it forced the business rules to be made explicit before implementation. The tests cover the scheduling behaviour at multiple layers:

* pure overlap and time window logic
* service level assignment, rescheduling, and completion flows  
* MySQL integration tests for transactions and constraints
* concurrency tests for simultaneous manager scheduling attempts
* HTTP API behaviour and error responses
* notification behaviour 
* WebSocket event delivery
* frontend and end to end user flows

The concurrency tests were especially important. They verify that overlapping assignment attempts against the same technician result in exactly one successful job creation, with the other request rejected as a scheduling conflict.

### AI Assisted Development 

AI tools were used as an accelerator, not as a substitute for architectural judgment. 

I used GPT-5.5 Pro to help clarify the system design, generate a detailed spec, and enumerate edge cases. Codex was then used to implement against those specs using test driven development.

The important architecture decisions were deliberate:

* backend commands over HTTP rather than WebSocket  
* MySQL as the scheduling source of truth
* transaction based conflict prevention
* row locking for concurrent scheduling safety
* persisted notifications
* minimal WebSocket event payloads 
* simple auth scoped to the take home requirements

The AI generated work was constrained by tests and reviewed against the business invariants. The final confidence comes from the transaction design and test coverage, not from the generated code itself.

### Trade offs

To stay within the intended scope, I intentionally kept several areas simple:

* Auth uses seeded users and an HttpOnly cookie, but does not include registration, password reset, or production grade session lifecycle management.
* The UI uses forms and lists instead of a full calendar.
* WebSocket messages are refresh signals rather than full real time state synchronisation.  
* The WebSocket hub is in process rather than distributed.
* Notifications are stored and displayed in app, but not sent by email or SMS.
* There is no tenant or organisation model.
* There are no technician availability, travel time, or business hour rules.

These were deliberate trade offs to keep the implementation focussed on the core domain problem: assigning quotes to technicians safely and correctly.

### What I Would Add Next

With more time, I would add:

* organisation tenancy
* a proper calendar interface 
* technician availability rules
* audit logs for scheduling changes
* distributed WebSocket fanout
* email or SMS notifications  
* better observability and tracing
* stronger auth and session hardening
* richer end to end tests around reconnects and stale UI state
