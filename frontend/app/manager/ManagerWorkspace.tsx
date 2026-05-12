"use client";

import Alert from "@mui/joy/Alert";
import Box from "@mui/joy/Box";
import Breadcrumbs from "@mui/joy/Breadcrumbs";
import Button from "@mui/joy/Button";
import Chip from "@mui/joy/Chip";
import DialogActions from "@mui/joy/DialogActions";
import DialogContent from "@mui/joy/DialogContent";
import DialogTitle from "@mui/joy/DialogTitle";
import Divider from "@mui/joy/Divider";
import FormControl from "@mui/joy/FormControl";
import FormLabel from "@mui/joy/FormLabel";
import Input from "@mui/joy/Input";
import Link from "@mui/joy/Link";
import List from "@mui/joy/List";
import ListItem from "@mui/joy/ListItem";
import Modal from "@mui/joy/Modal";
import ModalDialog from "@mui/joy/ModalDialog";
import Option from "@mui/joy/Option";
import Select from "@mui/joy/Select";
import Sheet from "@mui/joy/Sheet";
import Snackbar from "@mui/joy/Snackbar";
import Stack from "@mui/joy/Stack";
import Table from "@mui/joy/Table";
import Textarea from "@mui/joy/Textarea";
import Typography from "@mui/joy/Typography";
import { ChevronRight, Home } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { AppShell } from "../../components/AppShell";
import { SchedulePicker } from "../../components/SchedulePicker";
import {
  Job,
  Notification,
  Quote,
  Technician,
  User,
  assignJob,
  createQuote,
  getMe,
  listJobs,
  listNotifications,
  listQuotes,
  listTechnicians,
  logout,
  markNotificationRead,
  openEventSocket,
  rescheduleJob
} from "../../lib/api";
import {
  bookedJobsForTechnician,
  canScheduleAt,
  dateHasAvailableSlot,
  formatAvailabilityRule,
  formatScheduleWindow,
  isSlotAvailable,
  scheduleInputToISOString
} from "../../lib/schedulingAvailability";

export type ManagerPageMode = "quotes" | "assign" | "jobs";

type ToastState = {
  color: "success" | "danger" | "primary";
  message: string;
};

const pageMeta = {
  quotes: { activePage: "manager-quotes" as const, title: "Quotes" },
  assign: { activePage: "manager-assign" as const, title: "Assign quotes" },
  jobs: { activePage: "manager-jobs" as const, title: "Jobs" }
};

export function ManagerWorkspace({ mode }: { mode: ManagerPageMode }) {
  const [user, setUser] = useState<User | null>(null);
  const [quotes, setQuotes] = useState<Quote[]>([]);
  const [technicians, setTechnicians] = useState<Technician[]>([]);
  const [jobs, setJobs] = useState<Job[]>([]);
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const [quoteID, setQuoteID] = useState("");
  const [newQuoteCustomerName, setNewQuoteCustomerName] = useState("");
  const [newQuoteDescription, setNewQuoteDescription] = useState("");
  const [technicianID, setTechnicianID] = useState("");
  const [startsAt, setStartsAt] = useState("");
  const [rescheduleError, setRescheduleError] = useState("");
  const [rescheduleState, setRescheduleState] = useState<Record<number, { technicianID: string; startsAt: string }>>({});
  const [activeRescheduleJob, setActiveRescheduleJob] = useState<Job | null>(null);
  const [toast, setToast] = useState<ToastState | null>(null);
  const meta = pageMeta[mode];
  const selectedTechnician = technicians.find((technician) => String(technician.id) === technicianID) || null;
  const assignmentStartAvailable = !startsAt || !selectedTechnician || canScheduleAt(selectedTechnician, jobs, startsAt);

  const showToast = useCallback((message: string, color: ToastState["color"] = "primary") => {
    setToast({ message, color });
  }, []);

  const applyNotifications = useCallback((data: { notifications: Notification[]; unreadCount?: number }) => {
    setNotifications(data.notifications);
    setUnreadCount(data.unreadCount ?? data.notifications.filter((notification) => notification.readAt === null).length);
    return data.notifications;
  }, []);

  const refreshQuotes = useCallback(async () => {
    setQuotes((await listQuotes(mode === "assign" ? "unscheduled" : undefined)).quotes);
  }, [mode]);

  const refreshJobs = useCallback(async () => {
    setJobs((await listJobs()).jobs);
  }, []);

  const refreshNotifications = useCallback(async () => {
    return applyNotifications(await listNotifications());
  }, [applyNotifications]);

  const refreshAll = useCallback(async () => {
    const [me, quoteData, technicianData, jobData, notificationData] = await Promise.all([
      getMe(),
      listQuotes(mode === "assign" ? "unscheduled" : undefined),
      listTechnicians(),
      listJobs(),
      listNotifications()
    ]);
    setUser(me.user);
    setQuotes(quoteData.quotes);
    setTechnicians(technicianData.technicians);
    setJobs(jobData.jobs);
    applyNotifications(notificationData);
  }, [applyNotifications, mode]);

  useEffect(() => {
    refreshAll().catch(() => {
      window.location.href = "/login";
    });
  }, [refreshAll]);

  useEffect(() => {
    return openEventSocket((event) => {
      if (event.type === "notification.created") {
        void refreshNotifications().then((items) => {
          if (items[0]) showToast(items[0].message, "primary");
        });
      }
      if (event.type === "jobs.changed") void refreshJobs();
      if (event.type === "quotes.changed") void refreshQuotes();
    });
  }, [refreshJobs, refreshNotifications, refreshQuotes, showToast]);

  useEffect(() => {
    const refreshStaleData = () => {
      void refreshAll();
    };
    window.addEventListener("focus", refreshStaleData);
    return () => window.removeEventListener("focus", refreshStaleData);
  }, [refreshAll]);

  async function handleAssign() {
    try {
      await assignJob({
        quoteId: Number(quoteID),
        technicianId: Number(technicianID),
        startsAt: scheduleInputToISOString(startsAt)
      });
      setQuoteID("");
      setTechnicianID("");
      setStartsAt("");
      showToast("Job assigned.", "success");
      await Promise.all([refreshQuotes(), refreshJobs(), refreshNotifications()]);
    } catch (err) {
      showToast(errorMessage(err, "Assignment failed."), "danger");
    }
  }

  async function handleCreateQuote() {
    try {
      await createQuote({
        customerName: newQuoteCustomerName,
        description: newQuoteDescription
      });
      setNewQuoteCustomerName("");
      setNewQuoteDescription("");
      showToast("Quote created.", "success");
      await refreshQuotes();
    } catch (err) {
      showToast(errorMessage(err, "Quote creation failed."), "danger");
    }
  }

  async function handleReschedule(job: Job) {
    const state = rescheduleState[job.id]!;
    setRescheduleError("");
    try {
      await rescheduleJob(job.id, {
        technicianId: Number(state.technicianID),
        startsAt: scheduleInputToISOString(state.startsAt)
      });
      showToast("Job rescheduled.", "success");
      setActiveRescheduleJob(null);
      await Promise.all([refreshJobs(), refreshNotifications()]);
    } catch (err) {
      const message = errorMessage(err, "Reschedule failed.");
      setRescheduleError(message);
      showToast(message, "danger");
    }
  }

  async function handleMarkNotificationRead(notificationID: number) {
    await markNotificationRead(notificationID);
    await refreshNotifications();
  }

  return (
    <AppShell
      activePage={meta.activePage}
      notificationCount={unreadCount}
      notifications={notifications}
      onLogout={() => logout().finally(() => (window.location.href = "/login"))}
      onMarkNotificationRead={handleMarkNotificationRead}
      roleLabel="Manager workspace"
      unreadCount={unreadCount}
      userName={user?.displayName || "Loading"}
    >
      <Stack spacing={2}>
        <PageHeader title={meta.title} />
        {mode === "quotes" ? renderQuotesPage() : null}
        {mode === "assign" ? renderAssignPage() : null}
        {mode === "jobs" ? renderJobsPage() : null}
      </Stack>
      {renderRescheduleDialog()}
      <Snackbar autoHideDuration={4000} color={toast?.color || "primary"} onClose={setToast.bind(null, null)} open={toast !== null} variant="soft">
        <Alert aria-label="Notification popup" color={toast?.color || "primary"} role="alert" variant="soft">
          {toast?.message}
        </Alert>
      </Snackbar>
    </AppShell>
  );

  function renderQuotesPage() {
    return (
      <Stack spacing={2}>
        <Sheet variant="outlined" sx={{ p: 2 }}>
          <Stack spacing={2}>
            <Typography level="title-md">Create quote</Typography>
            <Stack direction={{ xs: "column", md: "row" }} spacing={1.5}>
              <FormControl sx={{ flex: 1 }}>
                <FormLabel>Customer name</FormLabel>
                <Input value={newQuoteCustomerName} onChange={(event) => setNewQuoteCustomerName(event.target.value)} />
              </FormControl>
              <FormControl sx={{ flex: 2 }}>
                <FormLabel>Quote description</FormLabel>
                <Textarea minRows={1} value={newQuoteDescription} onChange={(event) => setNewQuoteDescription(event.target.value)} />
              </FormControl>
              <Button disabled={!newQuoteCustomerName.trim() || !newQuoteDescription.trim()} onClick={handleCreateQuote} sx={{ alignSelf: { xs: "stretch", md: "end" } }}>
                Create quote
              </Button>
            </Stack>
          </Stack>
        </Sheet>

        <Sheet variant="outlined" sx={{ overflow: "auto" }}>
          <Table aria-label="Existing quotes" stickyHeader hoverRow sx={tableSx}>
            <thead>
              <tr>
                <th>Customer</th>
                <th>Description</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {quotes.map((quote) => (
                <tr key={quote.id}>
                  <td>{quote.customerName}</td>
                  <td>{quote.description}</td>
                  <td>
                    <Chip size="sm" color={quote.status === "unscheduled" ? "primary" : "success"} variant="soft">
                      {quote.status}
                    </Chip>
                  </td>
                </tr>
              ))}
            </tbody>
          </Table>
        </Sheet>
      </Stack>
    );
  }

  function renderAssignPage() {
    return (
      <Stack direction={{ xs: "column", lg: "row" }} spacing={2} sx={{ alignItems: "flex-start" }}>
        <Sheet variant="outlined" sx={{ p: 2, width: "100%", maxWidth: 520 }}>
          <Stack spacing={2}>
            <FormControl>
              <FormLabel id="quote-select-label">Quote</FormLabel>
              <Select aria-labelledby="quote-select-label" onChange={(_, value) => setQuoteID(value as string)} placeholder="Select a quote" value={quoteID || null}>
                {quotes.map((quote) => (
                  <Option key={quote.id} value={String(quote.id)}>
                    {quote.customerName}
                  </Option>
                ))}
              </Select>
            </FormControl>

            <FormControl>
              <FormLabel id="technician-select-label">Technician</FormLabel>
              <Select aria-labelledby="technician-select-label" onChange={(_, value) => setTechnicianID(value as string)} placeholder="Select a technician" value={technicianID || null}>
                {technicians.map((technician) => (
                  <Option key={technician.id} value={String(technician.id)}>
                    {technician.displayName}
                  </Option>
                ))}
              </Select>
            </FormControl>

            <SchedulePicker
              label="Start time"
              value={startsAt}
              onChange={setStartsAt}
              isDateDisabled={(date) => (selectedTechnician ? !dateHasAvailableSlot(selectedTechnician, jobs, date) : false)}
              isTimeDisabled={(date, time) => (selectedTechnician ? !isSlotAvailable(selectedTechnician, jobs, date, time) : false)}
            />

            {renderTechnicianAvailability()}

            <Button disabled={!quoteID || !technicianID || !startsAt || !assignmentStartAvailable} onClick={handleAssign}>
              Assign
            </Button>
          </Stack>
        </Sheet>

        <Sheet variant="outlined" sx={{ p: 2, flex: 1, width: "100%" }}>
          <Typography level="title-md" sx={{ mb: 1 }}>
            Unscheduled quotes
          </Typography>
          <List aria-label="Unscheduled quote list" size="sm" sx={{ "--ListItem-paddingY": "0.5rem" }}>
            {quotes.map((quote) => (
              <ListItem key={quote.id} sx={{ display: "block" }}>
                <Typography level="title-sm">{quote.customerName}</Typography>
                <Typography level="body-sm" color="neutral">
                  {quote.description}
                </Typography>
              </ListItem>
            ))}
          </List>
        </Sheet>
      </Stack>
    );
  }

  function renderJobsPage() {
    return (
      <Sheet variant="outlined" sx={{ overflow: "auto" }}>
        <Table aria-label="Manager jobs" stickyHeader hoverRow sx={tableSx}>
          <thead>
            <tr>
              <th>Quote</th>
              <th>Technician</th>
              <th>Window</th>
              <th>Status</th>
              <th>Schedule</th>
            </tr>
          </thead>
          <tbody>
            {jobs.map((job) => (
              <tr key={job.id}>
                <td>{job.quoteCustomerName}</td>
                <td>{job.technicianName}</td>
                <td>{formatScheduleWindow(job)}</td>
                <td>
                  <Chip size="sm" color={job.status === "scheduled" ? "primary" : "success"} variant="soft">
                    {job.status}
                  </Chip>
                </td>
                <td>
                  {job.status === "scheduled" ? (
                    <Button
                      aria-label={`Reschedule ${job.quoteCustomerName}`}
                      color="neutral"
                      variant="outlined"
                      onClick={() => {
                        setRescheduleState((current) => ({
                          ...current,
                          [job.id]: { technicianID: String(job.technicianId), startsAt: "" }
                        }));
                        setRescheduleError("");
                        setActiveRescheduleJob(job);
                      }}
                    >
                      Reschedule
                    </Button>
                  ) : null}
                </td>
              </tr>
            ))}
          </tbody>
        </Table>
      </Sheet>
    );
  }

  function renderTechnicianAvailability() {
    if (!selectedTechnician) {
      return (
        <Stack role="region" aria-label="Technician availability" spacing={0.5} sx={{ borderTop: "1px solid", borderColor: "divider", pt: 1 }}>
          <Typography level="title-sm">Availability</Typography>
          <Typography level="body-sm" color="neutral">
            Select a technician to view availability and booked windows.
          </Typography>
        </Stack>
      );
    }

    const availability = selectedTechnician.availability ?? [];
    const bookings = bookedJobsForTechnician(jobs, selectedTechnician.id);

    return (
      <Stack role="region" aria-label={`${selectedTechnician.displayName} availability`} spacing={1} sx={{ borderTop: "1px solid", borderColor: "divider", pt: 1 }}>
        <Box>
          <Typography level="title-sm">Availability</Typography>
          <List size="sm" sx={{ "--ListItem-paddingY": "0.25rem" }}>
            {availability.map((rule) => (
              <ListItem key={`${rule.weekday}-${rule.startsAt}`} sx={{ display: "block" }}>
                <Typography level="body-sm">{formatAvailabilityRule(rule)}</Typography>
              </ListItem>
            ))}
            {availability.length === 0 ? (
              <ListItem>
                <Typography level="body-sm" color="neutral">
                  No availability rules.
                </Typography>
              </ListItem>
            ) : null}
          </List>
        </Box>
        <Box>
          <Typography level="title-sm">Booked windows</Typography>
          <List size="sm" sx={{ "--ListItem-paddingY": "0.25rem" }}>
            {bookings.map((job) => (
              <ListItem key={job.id} sx={{ display: "block" }}>
                <Typography level="body-sm">{job.quoteCustomerName || `Job #${job.id}`}</Typography>
                <Typography level="body-xs" color="neutral">
                  {formatScheduleWindow(job)}
                </Typography>
              </ListItem>
            ))}
            {bookings.length === 0 ? (
              <ListItem>
                <Typography level="body-sm" color="neutral">
                  No booked windows.
                </Typography>
              </ListItem>
            ) : null}
          </List>
        </Box>
      </Stack>
    );
  }

  function renderRescheduleDialog() {
    return (
        <Modal open={activeRescheduleJob !== null} onClose={() => setActiveRescheduleJob(null)}>
        <ModalDialog
          aria-labelledby="reschedule-dialog-title"
          sx={{
            maxWidth: 560,
            width: "calc(100% - 32px)",
            borderRadius: "var(--brix-ui-corner-radius)",
            "--Card-radius": "var(--brix-ui-corner-radius)"
          }}
        >
          {activeRescheduleJob ? (
            <Stack spacing={2}>
              <DialogTitle id="reschedule-dialog-title">Reschedule {activeRescheduleJob.quoteCustomerName}</DialogTitle>
              <DialogContent>
                <Stack spacing={2}>
                  {rescheduleError ? <Alert color="danger">{rescheduleError}</Alert> : null}
                  <FormControl>
                    <FormLabel id="new-technician-label">New technician</FormLabel>
                    <Select
                      aria-labelledby="new-technician-label"
                      onChange={(_, value) =>
                        setRescheduleState((current) => ({
                          ...current,
                          [activeRescheduleJob.id]: { technicianID: value as string, startsAt: current[activeRescheduleJob.id]!.startsAt }
                        }))
                      }
                      value={rescheduleState[activeRescheduleJob.id]!.technicianID}
                    >
                      {technicians.map((technician) => (
                        <Option key={technician.id} value={String(technician.id)}>
                          {technician.displayName}
                        </Option>
                      ))}
                    </Select>
                  </FormControl>
                  <SchedulePicker
                    calendarLabel={`Reschedule ${activeRescheduleJob.id} calendar`}
                    label="New start time"
                    timeSlotsLabel={`Reschedule ${activeRescheduleJob.id} time slots`}
                    value={rescheduleState[activeRescheduleJob.id]!.startsAt}
                    onChange={(value) =>
                      setRescheduleState((current) => ({
                        ...current,
                        [activeRescheduleJob.id]: {
                          technicianID: current[activeRescheduleJob.id]!.technicianID,
                          startsAt: value
                        }
                      }))
                    }
                  />
                </Stack>
              </DialogContent>
              <DialogActions>
                <Button variant="plain" color="neutral" onClick={() => setActiveRescheduleJob(null)}>
                  Cancel
                </Button>
                <Button
                  disabled={!rescheduleState[activeRescheduleJob.id]!.technicianID || !rescheduleState[activeRescheduleJob.id]!.startsAt}
                  onClick={() => handleReschedule(activeRescheduleJob)}
                >
                  Save changes
                </Button>
              </DialogActions>
            </Stack>
          ) : null}
        </ModalDialog>
      </Modal>
    );
  }
}

function PageHeader({ title }: { title: string }) {
  return (
    <Box>
      <Breadcrumbs aria-label="breadcrumbs" separator={<ChevronRight size={14} />} size="sm" sx={{ pl: 0 }}>
        <Link aria-label="Home" color="neutral" href="/manager/quotes" underline="none">
          <Home size={16} />
        </Link>
        <Link color="neutral" href="/manager/quotes" sx={{ fontSize: 12, fontWeight: 500 }} underline="hover">
          Manager
        </Link>
        <Typography color="primary" sx={{ fontSize: 12, fontWeight: 500 }}>
          {title}
        </Typography>
      </Breadcrumbs>
      <Stack direction={{ xs: "column", sm: "row" }} spacing={1} sx={{ alignItems: { xs: "flex-start", sm: "center" }, justifyContent: "space-between" }}>
        <Typography component="h1" level="h2">
          {title}
        </Typography>
      </Stack>
      <Divider sx={{ mt: 1 }} />
    </Box>
  );
}

function errorMessage(err: unknown, fallback: string): string {
  if (err instanceof Error) return err.message;
  if (typeof err === "object" && err !== null && "message" in err && typeof err.message === "string") {
    return err.message;
  }
  return fallback;
}

const tableSx = {
  "--TableCell-headBackground": "var(--joy-palette-background-level1)",
  "--Table-headerUnderlineThickness": "1px",
  "--TableRow-hoverBackground": "var(--joy-palette-background-level1)",
  "--TableCell-paddingY": "8px",
  "--TableCell-paddingX": "10px"
};
