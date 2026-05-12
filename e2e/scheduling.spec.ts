import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { test, expect, type Page } from "@playwright/test";

const backendURL = "http://localhost:8080";

test.beforeEach(() => {
  resetDatabase();
});

test("manager can assign quote to technician", async ({ page }) => {
  await login(page, "manager1@brix.test", "manager");
  await expect(page.getByLabel("Unscheduled quote list").getByText("Acme Plumbing")).toBeVisible();
  await page.getByLabel("Quote", { exact: true }).selectOption("1");
  await page.getByLabel("Technician", { exact: true }).selectOption("1");
  await page.getByLabel("Start time", { exact: true }).fill("2026-05-12T10:00");
  await page.getByRole("button", { name: "Assign" }).click();
  await expect(page.getByText("Job assigned.")).toBeVisible();
  await expect(page.getByRole("option", { name: "Acme Plumbing" })).toHaveCount(0);
  await expect(page.getByRole("cell", { name: "Tom Technician", exact: true })).toBeVisible();
});

test("technician sees assigned job and notification", async ({ page }) => {
  await createAssignment(page, 1, 1, "2026-05-12T10:00");
  await login(page, "technician1@brix.test", "technician");
  await expect(page.getByRole("cell", { name: "Acme Plumbing" })).toBeVisible();
  await expect(page.getByText(/assigned job #1 for Acme Plumbing/i)).toBeVisible();
});

test("technician completes job and manager sees notification", async ({ page }) => {
  await createAssignment(page, 1, 1, "2026-05-12T10:00");
  await login(page, "technician1@brix.test", "technician");
  await page.getByRole("button", { name: "Complete" }).click();
  await expect(page.getByText("completed")).toBeVisible();
  await login(page, "manager1@brix.test", "manager");
  await expect(page.getByText(/Acme Plumbing was completed/i)).toBeVisible();
});

test("manager sees conflict error for overlapping job", async ({ page }) => {
  await login(page, "manager1@brix.test", "manager");
  await assignVisible(page, "1", "1", "2026-05-12T10:00");
  await expect(page.getByText("Job assigned.")).toBeVisible();
  await assignVisible(page, "2", "1", "2026-05-12T11:00");
  await expect(page.getByRole("alert").filter({ hasText: "Technician already has a job in that time window." })).toBeVisible();
  await expect(page.getByLabel("Unscheduled quote list").getByText("Northside Dental")).toBeVisible();
});

test("manager can reschedule job", async ({ page }) => {
  await createAssignment(page, 1, 1, "2026-05-12T10:00");
  await login(page, "manager1@brix.test", "manager");
  const jobID = await firstJobID(page);
  await page.getByLabel(`Reschedule assignee ${jobID}`).selectOption("2");
  await page.getByLabel(`Reschedule start ${jobID}`).fill("2026-05-12T14:00");
  await page.getByRole("button", { name: "Reschedule" }).click();
  await expect(page.getByText("Job rescheduled.")).toBeVisible();
  await expect(page.getByRole("cell", { name: "Priya Technician", exact: true })).toBeVisible();
  await login(page, "technician2@brix.test", "technician");
  await expect(page.getByRole("cell", { name: "Acme Plumbing" })).toBeVisible();
});

test("other manager cannot reschedule job", async ({ page }) => {
  await createAssignment(page, 1, 1, "2026-05-12T10:00");
  await login(page, "manager1@brix.test", "manager");
  const jobID = await firstJobID(page);
  await login(page, "manager2@brix.test", "manager");
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

async function login(page: Page, email: string, role: "manager" | "technician") {
  await page.goto("/login");
  await page.getByLabel("Email").fill(email);
  await page.getByLabel("Password").fill("password123");
  await page.getByRole("button", { name: "Login" }).click();
  await page.waitForURL(`**/${role}`);
}

async function createAssignment(page: Page, quoteID: number, technicianID: number, startsAt: string) {
  await login(page, "manager1@brix.test", "manager");
  await assignVisible(page, String(quoteID), String(technicianID), startsAt);
}

async function assignVisible(page: Page, quoteID: string, technicianID: string, startsAt: string) {
  await page.getByLabel("Quote", { exact: true }).selectOption(quoteID);
  await page.getByLabel("Technician", { exact: true }).selectOption(technicianID);
  await page.getByLabel("Start time", { exact: true }).fill(startsAt);
  await expect(page.getByRole("button", { name: "Assign" })).toBeEnabled();
  await page.getByRole("button", { name: "Assign" }).click();
}

async function firstJobID(page: Page): Promise<number> {
  const data = await page.evaluate(async (backend) => {
    const response = await fetch(`${backend}/api/jobs`, { credentials: "include" });
    return response.json();
  }, backendURL);
  return data.jobs[0].id as number;
}

function resetDatabase() {
  const seed = readFileSync("backend/migrations/002_seed.sql", "utf8");
  const reset = `
SET FOREIGN_KEY_CHECKS = 0;
TRUNCATE TABLE notifications;
TRUNCATE TABLE jobs;
TRUNCATE TABLE quotes;
TRUNCATE TABLE technicians;
TRUNCATE TABLE managers;
TRUNCATE TABLE users;
SET FOREIGN_KEY_CHECKS = 1;
${seed}
`;
  execFileSync("docker", ["compose", "exec", "-T", "mysql", "mysql", "-ubrix", "-pbrix_password", "brix_scheduler"], {
    input: reset
  });
}
