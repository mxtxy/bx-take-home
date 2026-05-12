import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { test, expect, type Browser, type Page } from "@playwright/test";

const backendURL = "http://localhost:8080";
const quoteNames: Record<string, string> = {
  "1": "Acme Plumbing",
  "2": "Northside Dental"
};
const technicianNames: Record<string, string> = {
  "1": "Tom Technician",
  "2": "Priya Technician"
};

test.beforeEach(() => {
  resetDatabase();
});

test("manager can assign quote to technician", async ({ page }) => {
  await login(page, "manager1@brix.test", "manager");
  await openManagerPage(page, "Assign quotes", "/manager/assign");
  await expect(page.getByLabel("Unscheduled quote list").getByText("Acme Plumbing")).toBeVisible();
  await chooseJoyOption(page, "Quote", "Acme Plumbing");
  await chooseJoyOption(page, "Technician", "Tom Technician");
  await page.getByLabel("Start time", { exact: true }).fill(dateTimeLocal("2026-05-12T10:00:00Z"));
  await page.getByRole("button", { name: "Assign" }).click();
  await expect(page.getByText("Job assigned.")).toBeVisible();
  await expect(page.getByLabel("Unscheduled quote list").getByText("Acme Plumbing")).toHaveCount(0);
  await openManagerPage(page, "Jobs", "/manager/jobs");
  await expect(page.getByRole("cell", { name: "Tom Technician", exact: true })).toBeVisible();
  await expect(page.getByRole("table", { name: "Manager jobs" })).toContainText("2026-05-12 10:00 - 2026-05-12 12:00 Sydney");
});

test("technician sees assigned job and notification", async ({ page }) => {
  await createAssignment(page, 1, 1, "2026-05-12T10:00:00Z");
  await login(page, "technician1@brix.test", "technician");
  await expect(page.getByRole("cell", { name: "Acme Plumbing" })).toBeVisible();
  await expect(page.getByRole("table", { name: "Assigned jobs" })).toContainText("2026-05-12 10:00 - 2026-05-12 12:00 Sydney");
  await page.getByRole("button", { name: "Open notifications" }).click();
  await expect(page.getByRole("dialog", { name: "Notifications" })).toContainText(/assigned job #1 for Acme Plumbing/i);
  await expect(page.getByRole("dialog", { name: "Notifications" })).toContainText("12 May");
  await page.getByRole("button", { name: /Mark notification .* as read/ }).click();
  await expect(page.getByRole("dialog", { name: "Notifications" })).toContainText("0 unread");
});

test("technician completes job and manager sees notification", async ({ page }) => {
  await createAssignment(page, 1, 1, "2026-05-12T10:00:00Z");
  await login(page, "technician1@brix.test", "technician");
  await page.getByRole("button", { name: "Complete" }).click();
  await expect(page.getByRole("row", { name: /Acme Plumbing.*completed/i })).toBeVisible();
  await login(page, "manager1@brix.test", "manager");
  await page.getByRole("button", { name: "Open notifications" }).click();
  await expect(page.getByRole("dialog", { name: "Notifications" })).toContainText(/Acme Plumbing was completed/i);
});

test("manager can create a quote", async ({ page }) => {
  await login(page, "manager1@brix.test", "manager");
  await page.getByLabel("Customer name").fill("Sydney Bakery");
  await page.getByLabel("Quote description").fill("Install replacement oven circuit");
  await page.getByRole("button", { name: "Create quote" }).click();
  await expect(page.getByLabel("Notification popup")).toContainText("Quote created.");
  await expect(page.getByRole("table", { name: "Existing quotes" }).getByText("Sydney Bakery")).toBeVisible();
});

test("manager cannot submit overlapping assignment", async ({ page }) => {
  await login(page, "manager1@brix.test", "manager");
  await openManagerPage(page, "Assign quotes", "/manager/assign");
  await assignVisible(page, "1", "1", "2026-05-12T10:00:00Z");
  await expect(page.getByText("Job assigned.")).toBeVisible();
  await chooseJoyOption(page, "Quote", "Northside Dental");
  await chooseJoyOption(page, "Technician", "Tom Technician");
  await page.getByLabel("Start time", { exact: true }).fill(dateTimeLocal("2026-05-12T11:00:00Z"));
  await expect(page.getByRole("button", { name: "Assign" })).toBeDisabled();
  await expect(page.getByLabel("Unscheduled quote list").getByText("Northside Dental")).toBeVisible();
});

test("manager can reschedule job", async ({ page }) => {
  await createAssignment(page, 1, 1, "2026-05-12T10:00:00Z");
  await login(page, "manager1@brix.test", "manager");
  await openManagerPage(page, "Jobs", "/manager/jobs");
  await page.getByRole("button", { name: "Reschedule Acme Plumbing" }).click();
  await expect(page.getByRole("dialog", { name: "Reschedule Acme Plumbing" })).toBeVisible();
  await chooseJoyOption(page, "New technician", "Priya Technician");
  await page.getByLabel("New start time", { exact: true }).fill(dateTimeLocal("2026-05-12T14:00:00Z"));
  await page.getByRole("button", { name: "Save changes" }).click();
  await expect(page.getByText("Job rescheduled.")).toBeVisible();
  await expect(page.getByRole("cell", { name: "Priya Technician", exact: true })).toBeVisible();
  await login(page, "technician2@brix.test", "technician");
  await expect(page.getByRole("cell", { name: "Acme Plumbing" })).toBeVisible();
});

test("other manager cannot reschedule job", async ({ page }) => {
  await createAssignment(page, 1, 1, "2026-05-12T10:00:00Z");
  await login(page, "manager1@brix.test", "manager");
  const jobID = await firstJobID(page);
  await login(page, "manager2@brix.test", "manager");
  await openManagerPage(page, "Jobs", "/manager/jobs");
  await expect(page.getByText("Acme Plumbing")).toHaveCount(0);
  const status = await page.evaluate(
    async ({ backend, id }) => {
      const response = await fetch(`${backend}/api/jobs/${id}/schedule`, {
        method: "PATCH",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ technicianId: 2, startsAt: "2026-05-12T14:00:00Z" })
      });
      return response.status;
    },
    { backend: backendURL, id: jobID }
  );
  expect(status).toBe(403);
});

test("manager refreshes stale UI state on focus", async ({ page }) => {
  await login(page, "manager1@brix.test", "manager");
  await openManagerPage(page, "Assign quotes", "/manager/assign");
  await expect(page.getByLabel("Unscheduled quote list").getByText("Acme Plumbing")).toBeVisible();

  insertAssignmentDirectly(1, 1, 1, "2026-05-11 22:00:00", "2026-05-12 00:00:00");
  await page.evaluate(() => window.dispatchEvent(new Event("focus")));

  await expect(page.getByLabel("Unscheduled quote list").getByText("Acme Plumbing")).toHaveCount(0);
});

test("technician receives live updates after WebSocket reconnect", async ({ page, browser }) => {
  await login(page, "technician1@brix.test", "technician");
  await expect(page.getByRole("heading", { name: "Assigned jobs" })).toBeVisible();

  await page.context().setOffline(true);
  await page.context().setOffline(false);
  await page.waitForTimeout(1500);

  await createAssignmentInNewManagerContext(browser, 1, 1, "2026-05-12T10:00:00Z");

  await expect(page.getByRole("cell", { name: "Acme Plumbing" })).toBeVisible({ timeout: 7000 });
  await expect(page.getByLabel("Notification popup")).toContainText("You have been assigned job");
});

async function login(page: Page, email: string, role: "manager" | "technician") {
  await page.goto("/login");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill("password123");
  await page.getByRole("button", { name: "Login" }).click();
  await page.waitForURL(role === "manager" ? "**/manager/quotes" : "**/technician");
}

async function createAssignment(page: Page, quoteID: number, technicianID: number, startsAtUTC: string) {
  await login(page, "manager1@brix.test", "manager");
  await openManagerPage(page, "Assign quotes", "/manager/assign");
  await assignVisible(page, String(quoteID), String(technicianID), startsAtUTC);
  await expect(page.getByText("Job assigned.")).toBeVisible();
}

async function openManagerPage(page: Page, name: "Assign quotes" | "Jobs" | "Quotes", path: string) {
  await page.getByRole("link", { name }).click();
  await page.waitForURL(`**${path}`);
}

async function assignVisible(page: Page, quoteID: string, technicianID: string, startsAtUTC: string) {
  await chooseJoyOption(page, "Quote", quoteNames[quoteID]);
  await chooseJoyOption(page, "Technician", technicianNames[technicianID]);
  await page.getByLabel("Start time", { exact: true }).fill(dateTimeLocal(startsAtUTC));
  await expect(page.getByRole("button", { name: "Assign" })).toBeEnabled();
  await page.getByRole("button", { name: "Assign" }).click();
}

async function chooseJoyOption(page: Page, label: string, optionName: string) {
  await page.getByRole("combobox", { name: label }).click();
  await page.getByRole("option", { name: optionName }).click();
}

async function firstJobID(page: Page): Promise<number> {
  const data = await page.evaluate(async (backend) => {
    const response = await fetch(`${backend}/api/jobs`, { credentials: "include" });
    return response.json();
  }, backendURL);
  return data.jobs[0].id as number;
}

async function createAssignmentInNewManagerContext(browser: Browser, quoteID: number, technicianID: number, startsAt: string) {
  const context = await browser.newContext();
  const page = await context.newPage();
  try {
    await createAssignment(page, quoteID, technicianID, startsAt);
  } finally {
    await context.close();
  }
}

function dateTimeLocal(iso: string): string {
  const date = new Date(iso);
  const pad = (value: number) => String(value).padStart(2, "0");
  return `${date.getUTCFullYear()}-${pad(date.getUTCMonth() + 1)}-${pad(date.getUTCDate())}T${pad(date.getUTCHours())}:${pad(date.getUTCMinutes())}`;
}

function resetDatabase() {
  const seed = readFileSync("backend/migrations/002_seed.sql", "utf8");
  const reset = `
SET FOREIGN_KEY_CHECKS = 0;
TRUNCATE TABLE schedule_audit_logs;
TRUNCATE TABLE notifications;
TRUNCATE TABLE sessions;
TRUNCATE TABLE jobs;
TRUNCATE TABLE quotes;
TRUNCATE TABLE technician_availability_rules;
TRUNCATE TABLE technicians;
TRUNCATE TABLE managers;
TRUNCATE TABLE users;
TRUNCATE TABLE organizations;
SET FOREIGN_KEY_CHECKS = 1;
${seed}
`;
  execFileSync("docker", ["compose", "exec", "-T", "mysql", "mysql", "-ubrix", "-pbrix_password", "brix_scheduler"], {
    input: reset
  });
}

function insertAssignmentDirectly(quoteID: number, technicianID: number, managerID: number, startsAt: string, endsAt: string) {
  const sql = `
INSERT INTO jobs (organization_id, quote_id, technician_id, manager_id, starts_at, ends_at, status)
VALUES (1, ${quoteID}, ${technicianID}, ${managerID}, '${startsAt}', '${endsAt}', 'scheduled');
UPDATE quotes SET status = 'scheduled' WHERE id = ${quoteID};
`;
  execFileSync("docker", ["compose", "exec", "-T", "mysql", "mysql", "-ubrix", "-pbrix_password", "brix_scheduler"], {
    input: sql
  });
}
