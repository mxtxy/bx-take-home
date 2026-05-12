import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, test, vi } from "vitest";
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

describe("manager dashboard", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    api.getMe.mockResolvedValue({ user: managerUser });
    api.listQuotes.mockResolvedValue({ quotes: [quote] });
    api.listTechnicians.mockResolvedValue({ technicians });
    api.listJobs.mockResolvedValue({ jobs: [scheduledJob] });
    api.listNotifications.mockResolvedValue({ notifications: [], unreadCount: 0 });
    api.markNotificationRead.mockResolvedValue({ notification: managerNotification });
    api.assignJob.mockResolvedValue({ job: scheduledJob });
    api.createQuote.mockResolvedValue({ quote: createdQuote });
    api.rescheduleJob.mockResolvedValue({ job: scheduledJob });
    api.openEventSocket.mockReturnValue(() => undefined);
  });

  test("Manager dashboard renders unscheduled quotes section", async () => {
    renderManagerPage();
    expect(await screen.findByText(/unscheduled quotes/i)).toBeInTheDocument();
    expect(screen.getAllByText("Acme Plumbing").length).toBeGreaterThan(0);
  });

  test("Manager dashboard renders technician selector", async () => {
    renderManagerPage();
    expect(await screen.findByRole("combobox", { name: "Technician" })).toHaveTextContent("Select a technician");
  });

  test("Manager dashboard renders Joy dropdowns with clear placeholders", async () => {
    renderManagerPage();

    expect(await screen.findByRole("combobox", { name: "Quote" })).toHaveTextContent("Select a quote");
    expect(screen.getByRole("combobox", { name: "Technician" })).toHaveTextContent("Select a technician");
  });

  test("Manager dashboard renders calendar scheduling controls", async () => {
    renderManagerPage();
    expect(await screen.findByLabelText("Start time")).toBeInTheDocument();
    expect(screen.getByRole("group", { name: "Start time calendar" })).toBeInTheDocument();
  });

  test("Assign page displays selected technician availability and booked windows", async () => {
    api.listJobs.mockResolvedValue({
      jobs: [
        scheduledJob,
        {
          ...scheduledJob,
          id: 2,
          quoteCustomerName: "Northside Dental",
          startsAt: "2026-05-12T00:00:00Z",
          endsAt: "2026-05-12T02:00:00Z"
        }
      ]
    });
    renderManagerPage();

    await chooseJoyOption("Technician", "Tom Technician");

    const availability = await screen.findByRole("region", { name: "Tom Technician availability" });
    expect(availability).toHaveTextContent("Availability");
    expect(availability).toHaveTextContent("Tue 08:00-18:00 Sydney");
    expect(availability).toHaveTextContent("Booked windows");
    expect(availability).toHaveTextContent("Acme Plumbing");
    expect(availability).toHaveTextContent("2026-05-12 08:00 - 2026-05-12 10:00 Sydney");
    expect(within(availability).getByText("Acme Plumbing").compareDocumentPosition(within(availability).getByText("Northside Dental"))).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
  });

  test("Assign page handles technicians without availability or booked windows", async () => {
    api.listTechnicians.mockResolvedValue({ technicians: [technicianWithoutAvailability] });
    api.listJobs.mockResolvedValue({ jobs: [] });
    renderManagerPage();

    await chooseJoyOption("Technician", "No Availability");

    const availability = await screen.findByRole("region", { name: "No Availability availability" });
    expect(availability).toHaveTextContent("No availability rules.");
    expect(availability).toHaveTextContent("No booked windows.");
  });

  test("Assign page labels bookings without a quote customer name", async () => {
    api.listTechnicians.mockResolvedValue({ technicians: [{ ...technicianWithoutAvailability, availability: weekdayAvailability }] });
    api.listJobs.mockResolvedValue({
      jobs: [{ ...scheduledJob, id: 9, quoteCustomerName: undefined, technicianId: 3 }]
    });
    renderManagerPage();

    await chooseJoyOption("Technician", "No Availability");

    expect(await screen.findByRole("region", { name: "No Availability availability" })).toHaveTextContent("Job #9");
  });

  test("Assign page disables booked slots and fully unavailable dates", async () => {
    api.listJobs.mockResolvedValue({
      jobs: [
        {
          ...scheduledJob,
          startsAt: "2026-05-12T00:00:00Z",
          endsAt: "2026-05-12T02:00:00Z"
        }
      ]
    });
    renderManagerPage();

    await chooseJoyOption("Quote", "Acme Plumbing");
    await chooseJoyOption("Technician", "Tom Technician");

    await waitFor(() => expect(screen.getByRole("button", { name: "Select 2026-05-16" })).toBeDisabled());
    expect(screen.getByRole("button", { name: "08:00" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "10:00" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "16:00" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "18:00" })).toBeDisabled();

    await userEvent.type(screen.getByLabelText("Start time"), "2026-05-12T10:00");
    expect(screen.getByRole("button", { name: /^assign$/i })).toBeDisabled();
  });

  test("Assign button is disabled until quote, technician, and start time are selected", async () => {
    renderManagerPage();
    const assign = await screen.findByRole("button", { name: /assign/i });
    expect(assign).toBeDisabled();
    await chooseJoyOption("Quote", "Acme Plumbing");
    await chooseJoyOption("Technician", "Tom Technician");
    await userEvent.type(screen.getByLabelText("Start time"), "2026-05-12T12:00");
    expect(assign).toBeEnabled();
  });

  test("Successful assignment resets assignment form", async () => {
    renderManagerPage();
    await screen.findByRole("combobox", { name: "Quote" });
    await chooseJoyOption("Quote", "Acme Plumbing");
    await chooseJoyOption("Technician", "Tom Technician");
    await userEvent.type(screen.getByLabelText("Start time"), "2026-05-12T12:00");
    await userEvent.click(screen.getByRole("button", { name: /assign/i }));

    await waitFor(() =>
      expect(api.assignJob).toHaveBeenCalledWith({
        quoteId: 1,
        technicianId: 1,
        startsAt: "2026-05-12T02:00:00.000Z"
      })
    );
    expect(screen.getByRole("combobox", { name: "Quote" })).toHaveTextContent("Select a quote");
    expect(screen.getByRole("combobox", { name: "Technician" })).toHaveTextContent("Select a technician");
    expect(screen.getByLabelText("Start time")).toHaveValue("");
  });

  test("Conflict response displays visible MUI Alert", async () => {
    api.assignJob.mockRejectedValue({ message: "Technician already has a job in that time window." });
    renderManagerPage();
    await screen.findByRole("combobox", { name: "Quote" });
    await chooseJoyOption("Quote", "Acme Plumbing");
    await chooseJoyOption("Technician", "Tom Technician");
    await userEvent.type(screen.getByLabelText("Start time"), "2026-05-12T12:00");
    await userEvent.click(screen.getByRole("button", { name: /assign/i }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Technician already has a job in that time window.");
  });

  test("Assignment Error instance message renders in alert", async () => {
    api.assignJob.mockRejectedValue(new Error("Backend is unavailable."));
    renderManagerPage();
    await screen.findByRole("combobox", { name: "Quote" });
    await chooseJoyOption("Quote", "Acme Plumbing");
    await chooseJoyOption("Technician", "Tom Technician");
    await userEvent.type(screen.getByLabelText("Start time"), "2026-05-12T12:00");
    await userEvent.click(screen.getByRole("button", { name: /assign/i }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Backend is unavailable.");
  });

  test("Assignment fallback error renders when thrown value has no message", async () => {
    api.assignJob.mockRejectedValue("failed");
    renderManagerPage();
    await screen.findByRole("combobox", { name: "Quote" });
    await chooseJoyOption("Quote", "Acme Plumbing");
    await chooseJoyOption("Technician", "Tom Technician");
    await userEvent.type(screen.getByLabelText("Start time"), "2026-05-12T12:00");
    await userEvent.click(screen.getByRole("button", { name: /assign/i }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Assignment failed.");
  });

  test("Assignment errors render through the notification popup", async () => {
    api.assignJob.mockRejectedValue(new Error("Technician is not available in that time window."));
    renderManagerPage();
    await screen.findByRole("combobox", { name: "Quote" });
    await chooseJoyOption("Quote", "Acme Plumbing");
    await chooseJoyOption("Technician", "Tom Technician");
    await userEvent.type(screen.getByLabelText("Start time"), "2026-05-12T12:00");
    await userEvent.click(screen.getByRole("button", { name: /assign/i }));

    expect(await screen.findByLabelText("Notification popup")).toHaveTextContent("Technician is not available in that time window.");
  });

  test("Manager can create a new unscheduled quote", async () => {
    api.listQuotes
      .mockResolvedValueOnce({ quotes: [quote] })
      .mockResolvedValueOnce({ quotes: [quote, createdQuote] });
    renderManagerPage("quotes");
    await screen.findByRole("heading", { name: "Quotes" });
    await userEvent.type(screen.getByLabelText("Customer name"), "Sydney Bakery");
    await userEvent.type(screen.getByLabelText("Quote description"), "Install replacement oven circuit");
    await userEvent.click(screen.getByRole("button", { name: "Create quote" }));

    await waitFor(() =>
      expect(api.createQuote).toHaveBeenCalledWith({
        customerName: "Sydney Bakery",
        description: "Install replacement oven circuit"
      })
    );
    expect(api.listQuotes).toHaveBeenCalledTimes(2);
    expect(await screen.findByLabelText("Notification popup")).toHaveTextContent("Quote created.");
    expect(screen.getByRole("table", { name: "Existing quotes" })).toHaveTextContent("Sydney Bakery");
  });

  test("Quote creation errors render through the notification popup", async () => {
    api.createQuote.mockRejectedValue(new Error("Customer name is required."));
    renderManagerPage("quotes");
    await screen.findByRole("heading", { name: "Quotes" });
    await userEvent.type(screen.getByLabelText("Customer name"), "Sydney Bakery");
    await userEvent.type(screen.getByLabelText("Quote description"), "Install replacement oven circuit");
    await userEvent.click(screen.getByRole("button", { name: "Create quote" }));

    expect(await screen.findByLabelText("Notification popup")).toHaveTextContent("Customer name is required.");
  });

  test("Scheduled jobs render with status chip", async () => {
    renderManagerPage("jobs");
    const table = await screen.findByRole("table", { name: "Manager jobs" });
    expect(table).toHaveTextContent("2026-05-12 08:00 - 2026-05-12 10:00 Sydney");
    expect(await screen.findByText("scheduled")).toBeInTheDocument();
  });

  test("Reschedule controls render only for scheduled jobs", async () => {
    api.listJobs.mockResolvedValue({ jobs: [scheduledJob, { ...scheduledJob, id: 2, status: "completed" }] });
    renderManagerPage("jobs");
    expect(await screen.findAllByRole("button", { name: /reschedule acme plumbing/i })).toHaveLength(1);
  });

  test("Reschedule opens a dialog instead of rendering inline row controls", async () => {
    renderManagerPage("jobs");
    await userEvent.click(await screen.findByRole("button", { name: /reschedule acme plumbing/i }));

    expect(screen.queryByLabelText("Reschedule start 1")).not.toBeInTheDocument();
    expect(screen.getByRole("dialog", { name: "Reschedule Acme Plumbing" })).toBeInTheDocument();
    expect(screen.getByRole("combobox", { name: "New technician" })).toHaveTextContent("Tom Technician");
    expect(screen.getByRole("button", { name: "Save changes" })).toBeDisabled();
    expect(api.rescheduleJob).not.toHaveBeenCalled();
  });

  test("Reschedule dialog closes from Escape and cancel actions", async () => {
    renderManagerPage("jobs");
    await userEvent.click(await screen.findByRole("button", { name: /reschedule acme plumbing/i }));
    await userEvent.keyboard("{Escape}");
    await waitFor(() => expect(screen.queryByRole("dialog", { name: "Reschedule Acme Plumbing" })).not.toBeInTheDocument());

    await userEvent.click(screen.getByRole("button", { name: /reschedule acme plumbing/i }));
    await userEvent.click(screen.getByRole("button", { name: "Cancel" }));

    await waitFor(() => expect(screen.queryByRole("dialog", { name: "Reschedule Acme Plumbing" })).not.toBeInTheDocument());
  });

  test("Successful reschedule sends selected technician and start time", async () => {
    const localStart = "2026-05-12T13:00";
    renderManagerPage("jobs");
    await userEvent.click(await screen.findByRole("button", { name: /reschedule acme plumbing/i }));
    await chooseJoyOption("New technician", "Priya Technician");
    await userEvent.type(screen.getByLabelText("New start time"), localStart);
    await userEvent.click(screen.getByRole("button", { name: "Save changes" }));

    await waitFor(() =>
      expect(api.rescheduleJob).toHaveBeenCalledWith(1, {
        technicianId: 2,
        startsAt: "2026-05-12T03:00:00.000Z"
      })
    );
    expect(await screen.findByRole("alert")).toHaveTextContent("Job rescheduled.");
  });

  test("Reschedule fallback error renders when thrown value has no message", async () => {
    api.rescheduleJob.mockRejectedValue("failed");
    renderManagerPage("jobs");
    await userEvent.click(await screen.findByRole("button", { name: /reschedule acme plumbing/i }));
    await userEvent.type(screen.getByLabelText("New start time"), "2026-05-12T13:00");
    await userEvent.click(screen.getByRole("button", { name: "Save changes" }));

    expect(await screen.findByRole("alert")).toHaveTextContent("Reschedule failed.");
  });

  test("Reschedule errors render through the notification popup", async () => {
    api.rescheduleJob.mockRejectedValue(new Error("Technician already has a job in that time window."));
    renderManagerPage("jobs");
    await userEvent.click(await screen.findByRole("button", { name: /reschedule acme plumbing/i }));
    await userEvent.type(screen.getByLabelText("New start time"), "2026-05-12T13:00");
    await userEvent.click(screen.getByRole("button", { name: "Save changes" }));

    expect(await screen.findByLabelText("Notification popup")).toHaveTextContent("Technician already has a job in that time window.");
  });

  test("Manager unread notification count renders when present", async () => {
    api.listNotifications.mockResolvedValue({ notifications: [managerNotification], unreadCount: 1 });
    renderManagerPage("quotes");

    expect(await screen.findByText("1 unread")).toBeInTheDocument();
  });

  test("Manager unread notification count falls back to unread notifications when count is omitted", async () => {
    api.listNotifications.mockResolvedValue({ notifications: [managerNotification] });
    renderManagerPage("quotes");

    expect(await screen.findByText("1 unread")).toBeInTheDocument();
  });

  test("Notification menu opens as a slide-out drawer with unread count and notification time", async () => {
    api.listNotifications.mockResolvedValue({ notifications: [managerNotification], unreadCount: 1 });
    renderManagerPage("quotes");

    await userEvent.click(await screen.findByRole("button", { name: "Open notifications" }));

    expect(screen.getByRole("dialog", { name: "Notifications" })).toBeInTheDocument();
    expect(screen.getByRole("dialog", { name: "Notifications" })).toHaveTextContent("1 unread");
    expect(screen.getByRole("dialog", { name: "Notifications" })).toHaveTextContent("Job #1 has been updated.");
    expect(screen.getByRole("dialog", { name: "Notifications" })).toHaveTextContent("12 May");
  });

  test("Manager load failure redirects to login", async () => {
    api.getMe.mockRejectedValue(new Error("unauthorized"));
    renderManagerPage("quotes");

    await waitFor(() => expect(api.getMe).toHaveBeenCalled());
  });

  test("Logout invokes API before redirecting to login", async () => {
    api.logout.mockResolvedValue({ ok: true });
    renderManagerPage("quotes");
    await userEvent.click(await screen.findByRole("button", { name: /logout/i }));

    await waitFor(() => expect(api.logout).toHaveBeenCalled());
  });

  test("notification.created WebSocket event triggers notifications refetch", async () => {
    let listener: (event: { type: string }) => void = () => undefined;
    api.openEventSocket.mockImplementation((callback) => {
      listener = callback;
      return () => undefined;
    });
    renderManagerPage();
    await screen.findByText(/unscheduled quotes/i);
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
      .mockResolvedValueOnce({ notifications: [managerNotification], unreadCount: 1 });
    renderManagerPage();
    await screen.findByText(/unscheduled quotes/i);

    listener({ type: "notification.created" });

    expect(await screen.findByLabelText("Notification popup")).toHaveTextContent("Job #1 has been updated.");
  });

  test("jobs.changed WebSocket event triggers jobs refetch", async () => {
    let listener: (event: { type: string }) => void = () => undefined;
    api.openEventSocket.mockImplementation((callback) => {
      listener = callback;
      return () => undefined;
    });
    renderManagerPage("jobs");
    await screen.findByRole("heading", { name: "Jobs" });
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
    renderManagerPage();
    await screen.findByText(/unscheduled quotes/i);
    api.listQuotes.mockClear();

    listener({ type: "quotes.changed" });

    await waitFor(() => expect(api.listQuotes).toHaveBeenCalledWith("unscheduled"));
  });

  test("window focus refreshes stale manager data", async () => {
    renderManagerPage();
    await screen.findByText(/unscheduled quotes/i);
    api.listQuotes.mockClear();
    api.listJobs.mockClear();
    api.listNotifications.mockClear();

    window.dispatchEvent(new Event("focus"));

    await waitFor(() => expect(api.listQuotes).toHaveBeenCalledWith("unscheduled"));
    expect(api.listJobs).toHaveBeenCalled();
    expect(api.listNotifications).toHaveBeenCalled();
  });
});

function renderManagerPage(page: "assign" | "jobs" | "quotes" = "assign") {
  const Component = page === "assign" ? ManagerAssignPage : page === "jobs" ? ManagerJobsPage : ManagerQuotesPage;
  render(
    <AppProviders>
      <Component />
    </AppProviders>
  );
}

async function chooseJoyOption(label: string, optionName: string) {
  await userEvent.click(screen.getByRole("combobox", { name: label }));
  await userEvent.click(await screen.findByRole("option", { name: optionName }));
}

const managerUser = { id: 1, displayName: "Sarah Manager", role: "manager" };
const quote = { id: 1, customerName: "Acme Plumbing", description: "Replace leaking kitchen tap", status: "unscheduled" };
const createdQuote = { id: 6, customerName: "Sydney Bakery", description: "Install replacement oven circuit", status: "unscheduled" };
const weekdayAvailability = [
  { weekday: 0, startsAt: "08:00", endsAt: "18:00" },
  { weekday: 1, startsAt: "08:00", endsAt: "18:00" },
  { weekday: 2, startsAt: "08:00", endsAt: "18:00" },
  { weekday: 3, startsAt: "08:00", endsAt: "18:00" },
  { weekday: 4, startsAt: "08:00", endsAt: "18:00" }
];
const technicians = [
  { id: 1, userId: 3, displayName: "Tom Technician", email: "technician1@brix.test", availability: weekdayAvailability },
  { id: 2, userId: 4, displayName: "Priya Technician", email: "technician2@brix.test", availability: weekdayAvailability }
];
const technicianWithoutAvailability = { id: 3, userId: 5, displayName: "No Availability", email: "none@brix.test" };
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
