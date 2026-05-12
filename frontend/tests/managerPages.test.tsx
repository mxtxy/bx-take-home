import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ReactElement } from "react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import ManagerPage from "../app/manager/page";
import ManagerAssignPage from "../app/manager/assign/page";
import ManagerJobsPage from "../app/manager/jobs/page";
import ManagerQuotesPage from "../app/manager/quotes/page";
import { AppProviders } from "../components/AppProviders";

const api = vi.hoisted(() => ({
  getMe: vi.fn(),
  listQuotes: vi.fn(),
  listTechnicians: vi.fn(),
  listJobs: vi.fn(),
  listNotifications: vi.fn(),
  markNotificationRead: vi.fn(),
  assignJob: vi.fn(),
  createQuote: vi.fn(),
  rescheduleJob: vi.fn(),
  logout: vi.fn(),
  openEventSocket: vi.fn()
}));

vi.mock("../lib/api", () => api);

describe("manager pages", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    api.getMe.mockResolvedValue({ user: managerUser });
    api.listQuotes.mockResolvedValue({ quotes: [quote] });
    api.listTechnicians.mockResolvedValue({ technicians });
    api.listJobs.mockResolvedValue({ jobs: [scheduledJob] });
    api.listNotifications.mockResolvedValue({ notifications: [managerNotification], unreadCount: 1 });
    api.markNotificationRead.mockResolvedValue({ notification: { ...managerNotification, readAt: "2026-05-12T11:00:00Z" } });
    api.assignJob.mockResolvedValue({ job: scheduledJob });
    api.createQuote.mockResolvedValue({ quote: createdQuote });
    api.rescheduleJob.mockResolvedValue({ job: scheduledJob });
    api.openEventSocket.mockReturnValue(() => undefined);
  });

  test("Quotes page owns quote creation and existing quote viewing", async () => {
    api.listQuotes.mockResolvedValue({ quotes: [quote, { ...createdQuote, status: "scheduled" }] });
    renderManagerPage(<ManagerQuotesPage />);

    expect(await screen.findByRole("heading", { name: "Quotes" })).toBeInTheDocument();
    expect(screen.getByLabelText("Customer name")).toBeInTheDocument();
    expect(screen.getByLabelText("Quote description")).toBeInTheDocument();
    expect(screen.getByRole("table", { name: "Existing quotes" })).toHaveTextContent("Acme Plumbing");
    expect(screen.getByRole("table", { name: "Existing quotes" })).toHaveTextContent("scheduled");
    expect(screen.queryByRole("button", { name: "Assign" })).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Jobs" })).not.toBeInTheDocument();
  });

  test("Manager index opens the quotes page", async () => {
    renderManagerPage(<ManagerPage />);

    expect(await screen.findByRole("heading", { name: "Quotes" })).toBeInTheDocument();
  });

  test("Assign quotes page owns quote assignment", async () => {
    renderManagerPage(<ManagerAssignPage />);

    expect(await screen.findByRole("heading", { name: "Assign quotes" })).toBeInTheDocument();
    expect(screen.getByRole("combobox", { name: "Quote" })).toHaveTextContent("Select a quote");
    expect(screen.getByRole("combobox", { name: "Technician" })).toHaveTextContent("Select a technician");
    expect(screen.getByLabelText("Start time")).toBeInTheDocument();
    expect(screen.queryByLabelText("Customer name")).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Jobs" })).not.toBeInTheDocument();
  });

  test("Jobs page owns scheduled job viewing and rescheduling", async () => {
    api.listJobs.mockResolvedValue({
      jobs: [{ ...scheduledJob, startsAt: "2026-05-11T22:00:00Z", endsAt: "2026-05-12T00:00:00Z" }]
    });
    renderManagerPage(<ManagerJobsPage />);

    expect(await screen.findByRole("heading", { name: "Jobs" })).toBeInTheDocument();
    expect(screen.getByRole("table", { name: "Manager jobs" })).toHaveTextContent("Acme Plumbing");
    expect(screen.getByRole("table", { name: "Manager jobs" })).toHaveTextContent("2026-05-12 08:00 - 2026-05-12 10:00 Sydney");
    expect(screen.getByRole("button", { name: "Reschedule Acme Plumbing" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Assign" })).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Customer name")).not.toBeInTheDocument();
  });

  test("Manager pages mark notifications as read and refresh the drawer count", async () => {
    api.listNotifications
      .mockResolvedValueOnce({ notifications: [managerNotification], unreadCount: 1 })
      .mockResolvedValueOnce({ notifications: [{ ...managerNotification, readAt: "2026-05-12T11:00:00Z" }], unreadCount: 0 });
    renderManagerPage(<ManagerQuotesPage />);

    await userEvent.click(await screen.findByRole("button", { name: "Open notifications" }));
    await userEvent.click(screen.getByRole("button", { name: "Mark notification 1 as read" }));

    await waitFor(() => expect(api.markNotificationRead).toHaveBeenCalledWith(1));
    await waitFor(() => expect(screen.getAllByText("0 unread").length).toBeGreaterThan(0));
  });
});

function renderManagerPage(page: ReactElement) {
  render(<AppProviders>{page}</AppProviders>);
}

const managerUser = { id: 1, displayName: "Sarah Manager", role: "manager" };
const quote = { id: 1, customerName: "Acme Plumbing", description: "Replace leaking kitchen tap", status: "unscheduled" };
const createdQuote = { id: 6, customerName: "Sydney Bakery", description: "Install replacement oven circuit", status: "unscheduled" };
const technicians = [
  { id: 1, userId: 3, displayName: "Tom Technician", email: "technician1@brix.test" },
  { id: 2, userId: 4, displayName: "Priya Technician", email: "technician2@brix.test" }
];
const scheduledJob = {
  id: 1,
  quoteId: 1,
  quoteCustomerName: "Acme Plumbing",
  quoteDescription: "Replace leaking kitchen tap",
  technicianId: 1,
  technicianName: "Tom Technician",
  managerId: 1,
  managerName: "Sarah Manager",
  startsAt: "2026-05-11T22:00:00Z",
  endsAt: "2026-05-12T00:00:00Z",
  status: "scheduled",
  completedAt: null
};
const managerNotification = {
  id: 1,
  type: "job_updated",
  message: "Job #1 has been updated.",
  jobId: 1,
  readAt: null,
  createdAt: "2026-05-12T10:00:00Z"
};
