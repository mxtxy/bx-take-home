import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, test, vi } from "vitest";
import TechnicianPage from "../app/technician/page";

const api = vi.hoisted(() => ({
  getMe: vi.fn(),
  listJobs: vi.fn(),
  listNotifications: vi.fn(),
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
    api.listNotifications.mockResolvedValue({ notifications: [assignmentNotification] });
    api.completeJob.mockResolvedValue({ job: { ...scheduledJob, status: "completed" } });
    api.openEventSocket.mockReturnValue(() => undefined);
  });

  test("Technician dashboard renders jobs list", async () => {
    render(<TechnicianPage />);
    expect(await screen.findByText("Acme Plumbing")).toBeInTheDocument();
  });

  test("Scheduled job renders Complete button", async () => {
    render(<TechnicianPage />);
    expect(await screen.findByRole("button", { name: /complete/i })).toBeInTheDocument();
  });

  test("Completed job does not render Complete button", async () => {
    api.listJobs.mockResolvedValue({ jobs: [{ ...scheduledJob, status: "completed", completedAt: "2026-05-12T16:00:00Z" }] });
    render(<TechnicianPage />);
    await screen.findByText("completed");
    expect(screen.queryByRole("button", { name: /complete/i })).not.toBeInTheDocument();
  });

  test("Completed job renders completed status chip", async () => {
    api.listJobs.mockResolvedValue({ jobs: [{ ...scheduledJob, status: "completed", completedAt: "2026-05-12T16:00:00Z" }] });
    render(<TechnicianPage />);
    expect(await screen.findByText("completed")).toBeInTheDocument();
  });

  test("Assignment notification renders in notifications panel", async () => {
    render(<TechnicianPage />);
    expect(await screen.findByText(/assigned job/i)).toBeInTheDocument();
  });

  test("Successful completion triggers jobs refetch", async () => {
    render(<TechnicianPage />);
    await userEvent.click(await screen.findByRole("button", { name: /complete/i }));
    await waitFor(() => expect(api.completeJob).toHaveBeenCalledWith(1));
    expect(api.listJobs).toHaveBeenCalledTimes(2);
  });

  test("Technician load failure redirects to login", async () => {
    api.getMe.mockRejectedValue(new Error("unauthorized"));
    render(<TechnicianPage />);

    await waitFor(() => expect(api.getMe).toHaveBeenCalled());
  });

  test("Logout invokes API before redirecting to login", async () => {
    api.logout.mockResolvedValue({ ok: true });
    render(<TechnicianPage />);
    await userEvent.click(await screen.findByRole("button", { name: /logout/i }));

    await waitFor(() => expect(api.logout).toHaveBeenCalled());
  });

  test("notification.created WebSocket event triggers notifications refetch", async () => {
    let listener: (event: { type: string }) => void = () => undefined;
    api.openEventSocket.mockImplementation((callback) => {
      listener = callback;
      return () => undefined;
    });
    render(<TechnicianPage />);
    await screen.findByText("Acme Plumbing");
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
    render(<TechnicianPage />);
    await screen.findByText("Acme Plumbing");
    api.listJobs.mockClear();

    listener({ type: "jobs.changed" });

    await waitFor(() => expect(api.listJobs).toHaveBeenCalled());
  });
});

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
  startsAt: "2026-05-12T10:00:00Z",
  endsAt: "2026-05-12T12:00:00Z",
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
