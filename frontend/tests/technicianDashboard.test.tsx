import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, test, vi } from "vitest";
import TechnicianPage from "../app/technician/page";
import { AppProviders } from "../components/AppProviders";

const api = vi.hoisted(() => ({
  getMe: vi.fn(),
  listJobs: vi.fn(),
  listNotifications: vi.fn(),
  markNotificationRead: vi.fn(),
  completeJob: vi.fn(),
  logout: vi.fn(),
  openEventSocket: vi.fn()
}));

vi.mock("../lib/api", () => api);

describe("technician dashboard", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    api.getMe.mockResolvedValue({ user: technicianUser });
    api.listJobs.mockResolvedValue({ jobs: [scheduledJob] });
    api.listNotifications.mockResolvedValue({ notifications: [assignmentNotification], unreadCount: 1 });
    api.markNotificationRead.mockResolvedValue({ notification: { ...assignmentNotification, readAt: "2026-05-12T11:00:00Z" } });
    api.completeJob.mockResolvedValue({ job: { ...scheduledJob, status: "completed" } });
    api.openEventSocket.mockReturnValue(() => undefined);
  });

  test("Technician dashboard renders jobs list", async () => {
    renderTechnicianPage();
    const table = await screen.findByRole("table", { name: "Assigned jobs" });

    expect(table).toHaveTextContent("Acme Plumbing");
    expect(table).toHaveTextContent("2026-05-12 08:00 - 2026-05-12 10:00 Sydney");
  });

  test("Technician jobs table matches the manager table interface", async () => {
    renderTechnicianPage();

    const table = await screen.findByRole("table", { name: "Assigned jobs" });
    const heading = screen.getByRole("heading", { name: "Assigned jobs" });

    expect(heading.compareDocumentPosition(table)).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
    expect(table.parentElement).not.toHaveTextContent("Assigned jobs");
    expect(table).toHaveAttribute("data-table-interface", "manager");
    expect(within(table).getByRole("columnheader", { name: "Quote" })).toBeInTheDocument();
  });

  test("Scheduled job renders Complete button", async () => {
    renderTechnicianPage();
    expect(await screen.findByRole("button", { name: /complete/i })).toBeInTheDocument();
  });

  test("Completed job does not render Complete button", async () => {
    api.listJobs.mockResolvedValue({ jobs: [{ ...scheduledJob, status: "completed", completedAt: "2026-05-12T16:00:00Z" }] });
    renderTechnicianPage();
    await screen.findByText("completed");
    expect(screen.queryByRole("button", { name: /complete/i })).not.toBeInTheDocument();
  });

  test("Completed job renders completed status chip", async () => {
    api.listJobs.mockResolvedValue({ jobs: [{ ...scheduledJob, status: "completed", completedAt: "2026-05-12T16:00:00Z" }] });
    renderTechnicianPage();
    expect(await screen.findByText("completed")).toBeInTheDocument();
  });

  test("Assignment notification renders in the notification drawer", async () => {
    renderTechnicianPage();
    await userEvent.click(await screen.findByRole("button", { name: "Open notifications" }));

    expect(screen.getByRole("dialog", { name: "Notifications" })).toHaveTextContent("You have been assigned job #1 for Acme Plumbing.");
    expect(screen.getByRole("dialog", { name: "Notifications" })).toHaveTextContent("12 May");
  });

  test("Technician unread notification count falls back to unread notifications when count is omitted", async () => {
    api.listNotifications.mockResolvedValue({ notifications: [assignmentNotification] });
    renderTechnicianPage();

    expect(await screen.findByText("1 unread")).toBeInTheDocument();
  });

  test("Technician marks notifications as read from the drawer", async () => {
    api.listNotifications
      .mockResolvedValueOnce({ notifications: [assignmentNotification], unreadCount: 1 })
      .mockResolvedValueOnce({ notifications: [{ ...assignmentNotification, readAt: "2026-05-12T11:00:00Z" }], unreadCount: 0 });
    renderTechnicianPage();

    await userEvent.click(await screen.findByRole("button", { name: "Open notifications" }));
    await userEvent.click(screen.getByRole("button", { name: "Mark notification 1 as read" }));

    await waitFor(() => expect(api.markNotificationRead).toHaveBeenCalledWith(1));
    await waitFor(() => expect(screen.getAllByText("0 unread").length).toBeGreaterThan(0));
  });

  test("Successful completion triggers jobs refetch", async () => {
    renderTechnicianPage();
    await userEvent.click(await screen.findByRole("button", { name: /complete/i }));
    await waitFor(() => expect(api.completeJob).toHaveBeenCalledWith(1));
    expect(api.listJobs).toHaveBeenCalledTimes(2);
  });

  test("Completion errors render through the notification popup", async () => {
    api.completeJob.mockRejectedValue(new Error("Job has already been completed."));
    renderTechnicianPage();
    await userEvent.click(await screen.findByRole("button", { name: /complete/i }));

    expect(await screen.findByLabelText("Notification popup")).toHaveTextContent("Job has already been completed.");
  });

  test("Completion object error messages render through the notification popup", async () => {
    api.completeJob.mockRejectedValue({ message: "Job is no longer assigned to you." });
    renderTechnicianPage();
    await userEvent.click(await screen.findByRole("button", { name: /complete/i }));

    expect(await screen.findByLabelText("Notification popup")).toHaveTextContent("Job is no longer assigned to you.");
  });

  test("Completion fallback errors render through the notification popup", async () => {
    api.completeJob.mockRejectedValue("failed");
    renderTechnicianPage();
    await userEvent.click(await screen.findByRole("button", { name: /complete/i }));

    expect(await screen.findByLabelText("Notification popup")).toHaveTextContent("Completion failed.");
  });

  test("Technician load failure redirects to login", async () => {
    api.getMe.mockRejectedValue(new Error("unauthorized"));
    renderTechnicianPage();

    await waitFor(() => expect(api.getMe).toHaveBeenCalled());
  });

  test("Logout invokes API before redirecting to login", async () => {
    api.logout.mockResolvedValue({ ok: true });
    renderTechnicianPage();
    await userEvent.click(await screen.findByRole("button", { name: /logout/i }));

    await waitFor(() => expect(api.logout).toHaveBeenCalled());
  });

  test("notification.created WebSocket event triggers notifications refetch", async () => {
    let listener: (event: { type: string }) => void = () => undefined;
    api.openEventSocket.mockImplementation((callback) => {
      listener = callback;
      return () => undefined;
    });
    renderTechnicianPage();
    await screen.findByText("Acme Plumbing");
    api.listNotifications.mockClear();

    listener({ type: "notification.created" });

    await waitFor(() => expect(api.listNotifications).toHaveBeenCalled());
  });

  test("notification.created WebSocket event shows live notification popup", async () => {
    let listener: (event: { type: string }) => void = () => undefined;
    api.openEventSocket.mockImplementation((callback) => {
      listener = callback;
      return () => undefined;
    });
    api.listNotifications
      .mockResolvedValueOnce({ notifications: [], unreadCount: 0 })
      .mockResolvedValueOnce({ notifications: [assignmentNotification], unreadCount: 1 });
    renderTechnicianPage();
    await screen.findByText("Acme Plumbing");

    listener({ type: "notification.created" });

    expect(await screen.findByLabelText("Notification popup")).toHaveTextContent("You have been assigned job #1 for Acme Plumbing.");
  });

  test("notification.created WebSocket event does not popup when no notification is returned", async () => {
    let listener: (event: { type: string }) => void = () => undefined;
    api.openEventSocket.mockImplementation((callback) => {
      listener = callback;
      return () => undefined;
    });
    api.listNotifications
      .mockResolvedValueOnce({ notifications: [], unreadCount: 0 })
      .mockResolvedValueOnce({ notifications: [], unreadCount: 0 });
    renderTechnicianPage();
    await screen.findByText("Acme Plumbing");

    listener({ type: "notification.created" });

    await waitFor(() => expect(api.listNotifications).toHaveBeenCalledTimes(2));
    expect(screen.queryByLabelText("Notification popup")).not.toBeInTheDocument();
  });

  test("jobs.changed WebSocket event triggers jobs refetch", async () => {
    let listener: (event: { type: string }) => void = () => undefined;
    api.openEventSocket.mockImplementation((callback) => {
      listener = callback;
      return () => undefined;
    });
    renderTechnicianPage();
    await screen.findByText("Acme Plumbing");
    api.listJobs.mockClear();

    listener({ type: "jobs.changed" });

    await waitFor(() => expect(api.listJobs).toHaveBeenCalled());
  });

  test("window focus refreshes stale technician data", async () => {
    renderTechnicianPage();
    await screen.findByText("Acme Plumbing");
    api.listJobs.mockClear();
    api.listNotifications.mockClear();

    window.dispatchEvent(new Event("focus"));

    await waitFor(() => expect(api.listJobs).toHaveBeenCalled());
    expect(api.listNotifications).toHaveBeenCalled();
  });
});

function renderTechnicianPage() {
  render(
    <AppProviders>
      <TechnicianPage />
    </AppProviders>
  );
}

const technicianUser = { id: 3, displayName: "Tom Technician", role: "technician" };
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
const assignmentNotification = {
  id: 1,
  type: "job_assigned",
  message: "You have been assigned job #1 for Acme Plumbing.",
  jobId: 1,
  readAt: null,
  createdAt: "2026-05-12T10:00:00Z"
};
