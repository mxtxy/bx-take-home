import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, test, vi } from "vitest";
import ManagerPage from "../app/manager/page";

const api = vi.hoisted(() => ({
  getMe: vi.fn(),
  listQuotes: vi.fn(),
  listTechnicians: vi.fn(),
  listJobs: vi.fn(),
  listNotifications: vi.fn(),
  assignJob: vi.fn(),
  rescheduleJob: vi.fn(),
  logout: vi.fn(),
  openEventSocket: vi.fn()
}));

vi.mock("../lib/api", () => api);

describe("manager dashboard", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    api.getMe.mockResolvedValue({ user: managerUser });
    api.listQuotes.mockResolvedValue({ quotes: [quote] });
    api.listTechnicians.mockResolvedValue({ technicians });
    api.listJobs.mockResolvedValue({ jobs: [scheduledJob] });
    api.listNotifications.mockResolvedValue({ notifications: [] });
    api.assignJob.mockResolvedValue({ job: scheduledJob });
    api.rescheduleJob.mockResolvedValue({ job: scheduledJob });
    api.openEventSocket.mockReturnValue(() => undefined);
  });

  test("Manager dashboard renders unscheduled quotes section", async () => {
    render(<ManagerPage />);
    expect(await screen.findByText(/unscheduled quotes/i)).toBeInTheDocument();
    expect(screen.getAllByText("Acme Plumbing").length).toBeGreaterThan(0);
  });

  test("Manager dashboard renders technician selector", async () => {
    render(<ManagerPage />);
    expect(await screen.findByLabelText("Technician")).toBeInTheDocument();
  });

  test("Manager dashboard renders start-time input", async () => {
    render(<ManagerPage />);
    expect(await screen.findByLabelText("Start time")).toBeInTheDocument();
  });

  test("Assign button is disabled until quote, technician, and start time are selected", async () => {
    render(<ManagerPage />);
    const assign = await screen.findByRole("button", { name: /assign/i });
    expect(assign).toBeDisabled();
    await userEvent.selectOptions(screen.getByLabelText("Quote"), "1");
    await userEvent.selectOptions(screen.getByLabelText("Technician"), "1");
    await userEvent.type(screen.getByLabelText("Start time"), "2026-05-12T10:00");
    expect(assign).toBeEnabled();
  });

  test("Successful assignment resets assignment form", async () => {
    render(<ManagerPage />);
    await userEvent.selectOptions(await screen.findByLabelText("Quote"), "1");
    await userEvent.selectOptions(screen.getByLabelText("Technician"), "1");
    await userEvent.type(screen.getByLabelText("Start time"), "2026-05-12T10:00");
    await userEvent.click(screen.getByRole("button", { name: /assign/i }));

    await waitFor(() => expect(api.assignJob).toHaveBeenCalled());
    expect(screen.getByLabelText("Quote")).toHaveValue("");
    expect(screen.getByLabelText("Technician")).toHaveValue("");
    expect(screen.getByLabelText("Start time")).toHaveValue("");
  });

  test("Conflict response displays visible MUI Alert", async () => {
    api.assignJob.mockRejectedValue({ message: "Technician already has a job in that time window." });
    render(<ManagerPage />);
    await userEvent.selectOptions(await screen.findByLabelText("Quote"), "1");
    await userEvent.selectOptions(screen.getByLabelText("Technician"), "1");
    await userEvent.type(screen.getByLabelText("Start time"), "2026-05-12T10:00");
    await userEvent.click(screen.getByRole("button", { name: /assign/i }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Technician already has a job in that time window.");
  });

  test("Assignment Error instance message renders in alert", async () => {
    api.assignJob.mockRejectedValue(new Error("Backend is unavailable."));
    render(<ManagerPage />);
    await userEvent.selectOptions(await screen.findByLabelText("Quote"), "1");
    await userEvent.selectOptions(screen.getByLabelText("Technician"), "1");
    await userEvent.type(screen.getByLabelText("Start time"), "2026-05-12T10:00");
    await userEvent.click(screen.getByRole("button", { name: /assign/i }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Backend is unavailable.");
  });

  test("Assignment fallback error renders when thrown value has no message", async () => {
    api.assignJob.mockRejectedValue("failed");
    render(<ManagerPage />);
    await userEvent.selectOptions(await screen.findByLabelText("Quote"), "1");
    await userEvent.selectOptions(screen.getByLabelText("Technician"), "1");
    await userEvent.type(screen.getByLabelText("Start time"), "2026-05-12T10:00");
    await userEvent.click(screen.getByRole("button", { name: /assign/i }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Assignment failed.");
  });

  test("Scheduled jobs render with status chip", async () => {
    render(<ManagerPage />);
    expect(await screen.findByText("scheduled")).toBeInTheDocument();
  });

  test("Reschedule controls render only for scheduled jobs", async () => {
    api.listJobs.mockResolvedValue({ jobs: [scheduledJob, { ...scheduledJob, id: 2, status: "completed" }] });
    render(<ManagerPage />);
    expect(await screen.findAllByRole("button", { name: /reschedule/i })).toHaveLength(1);
  });

  test("Reschedule without a selected start time does not call API", async () => {
    render(<ManagerPage />);
    await userEvent.click(await screen.findByRole("button", { name: /reschedule/i }));

    expect(api.rescheduleJob).not.toHaveBeenCalled();
  });

  test("Successful reschedule sends selected technician and start time", async () => {
    const localStart = "2026-05-12T13:00";
    render(<ManagerPage />);
    await userEvent.selectOptions(await screen.findByLabelText("Reschedule assignee 1"), "2");
    await userEvent.type(screen.getByLabelText("Reschedule start 1"), localStart);
    await userEvent.click(screen.getByRole("button", { name: /reschedule/i }));

    await waitFor(() =>
      expect(api.rescheduleJob).toHaveBeenCalledWith(1, {
        technicianId: 2,
        startsAt: new Date(localStart).toISOString()
      })
    );
    expect(await screen.findByRole("alert")).toHaveTextContent("Job rescheduled.");
  });

  test("Reschedule fallback error renders when thrown value has no message", async () => {
    api.rescheduleJob.mockRejectedValue("failed");
    render(<ManagerPage />);
    await userEvent.type(await screen.findByLabelText("Reschedule start 1"), "2026-05-12T13:00");
    await userEvent.click(screen.getByRole("button", { name: /reschedule/i }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Reschedule failed.");
  });

  test("Manager notifications render when present", async () => {
    api.listNotifications.mockResolvedValue({ notifications: [managerNotification] });
    render(<ManagerPage />);

    expect(await screen.findByText("Job #1 has been updated.")).toBeInTheDocument();
  });

  test("Manager load failure redirects to login", async () => {
    api.getMe.mockRejectedValue(new Error("unauthorized"));
    render(<ManagerPage />);

    await waitFor(() => expect(api.getMe).toHaveBeenCalled());
  });

  test("Logout invokes API before redirecting to login", async () => {
    api.logout.mockResolvedValue({ ok: true });
    render(<ManagerPage />);
    await userEvent.click(await screen.findByRole("button", { name: /logout/i }));

    await waitFor(() => expect(api.logout).toHaveBeenCalled());
  });

  test("notification.created WebSocket event triggers notifications refetch", async () => {
    let listener: (event: { type: string }) => void = () => undefined;
    api.openEventSocket.mockImplementation((callback) => {
      listener = callback;
      return () => undefined;
    });
    render(<ManagerPage />);
    await screen.findByText(/unscheduled quotes/i);
    api.listNotifications.mockClear();

    listener({ type: "notification.created" });

    await waitFor(() => expect(api.listNotifications).toHaveBeenCalled());
  });

  test("jobs.changed WebSocket event triggers jobs refetch", async () => {
    let listener: (event: { type: string }) => void = () => undefined;
    api.openEventSocket.mockImplementation((callback) => {
      listener = callback;
      return () => undefined;
    });
    render(<ManagerPage />);
    await screen.findByText(/unscheduled quotes/i);
    api.listJobs.mockClear();

    listener({ type: "jobs.changed" });

    await waitFor(() => expect(api.listJobs).toHaveBeenCalled());
  });

  test("quotes.changed WebSocket event triggers quotes refetch", async () => {
    let listener: (event: { type: string }) => void = () => undefined;
    api.openEventSocket.mockImplementation((callback) => {
      listener = callback;
      return () => undefined;
    });
    render(<ManagerPage />);
    await screen.findByText(/unscheduled quotes/i);
    api.listQuotes.mockClear();

    listener({ type: "quotes.changed" });

    await waitFor(() => expect(api.listQuotes).toHaveBeenCalledWith("unscheduled"));
  });
});

const managerUser = { id: 1, displayName: "Sarah Manager", role: "manager" };
const quote = { id: 1, customerName: "Acme Plumbing", description: "Replace leaking kitchen tap", status: "unscheduled" };
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
  startsAt: "2026-05-12T10:00:00Z",
  endsAt: "2026-05-12T12:00:00Z",
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
