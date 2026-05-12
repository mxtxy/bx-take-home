# Joy UI Migration Design

## Personas

- Managers need fast quote assignment, visible scheduling errors, and enough job context to reschedule without navigating away.
- Technicians need a focused queue of assigned jobs, clear status, notifications, and a low-friction completion action.
- Reviewers need consistent navigation and predictable seeded-account access while exercising the take-home flows.

## Navigation

The redesigned frontend uses one shared application shell for authenticated pages. The shell follows the Joy UI dashboard reference with a persistent sidebar, compact mobile header, role-specific navigation, notification drawer, dark-mode toggle, current user, and logout action. Workspace switching and accent switching were intentionally removed: manager and technician users are routed to their own role-specific pages, and Joy's default theme now owns the palette.

Managers land on `/manager/quotes`. Quote creation and existing quote viewing live on `/manager/quotes`, quote assignment lives on `/manager/assign`, and scheduled job viewing/rescheduling lives on `/manager/jobs`. The assignment page keeps the Joy `Select` placeholders, technician selection, and calendar-oriented `SchedulePicker`; the jobs page keeps rescheduling in a dialog instead of rendering dense inline controls in the table.

Technicians land on `/technician`, where the page prioritizes the assigned jobs table. Scheduled jobs expose a single completion action; completed jobs keep status visible and remove the completion affordance. The shared notification drawer now exposes a mark-as-read action for unread notifications and refreshes the unread count after the API call.

## UI Approach

Material UI imports were replaced with Joy UI components. `AppProviders` now uses Joy `CssVarsProvider` and `CssBaseline` without custom palette, radius, button, sheet, or shadow overrides. `RootLayout` includes `InitColorSchemeScript` and `suppressHydrationWarning` for Joy color-scheme hydration.

The authenticated dashboards use Joy `Sheet`, `Stack`, `Typography`, `Select`, `Option`, `List`, `Table`, `Button`, `Alert`, `Modal`, and `Chip` for a quiet operational layout. Primary actions use Joy's default solid primary `Button`; secondary actions use Joy neutral `outlined` or `plain` variants. Containers rely on Joy's default surface, border, radius, and shadow tokens instead of app-specific styling.

## Calendar Interface

The native datetime value is still preserved because the API contract sends a `startsAt` timestamp and computes the two-hour end time on the backend. The new `SchedulePicker` adds a calendar day grid and common two-hour time-slot buttons while retaining direct minute-aligned input for exact starts.

This keeps the UI simple, avoids drag-and-drop scheduling, and stays within the spec's non-goals while giving managers a proper calendar-oriented scheduling surface.
