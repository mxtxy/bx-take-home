"use client";

import {
  Alert,
  AppBar,
  Box,
  Button,
  Chip,
  Container,
  FormControl,
  InputLabel,
  List,
  ListItem,
  ListItemText,
  NativeSelect,
  Paper,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TextField,
  Toolbar,
  Typography
} from "@mui/material";
import { useCallback, useEffect, useState } from "react";
import {
  Job,
  Notification,
  Quote,
  Technician,
  User,
  assignJob,
  getMe,
  listJobs,
  listNotifications,
  listQuotes,
  listTechnicians,
  logout,
  openEventSocket,
  rescheduleJob
} from "../../lib/api";

export default function ManagerPage() {
  const [user, setUser] = useState<User | null>(null);
  const [quotes, setQuotes] = useState<Quote[]>([]);
  const [technicians, setTechnicians] = useState<Technician[]>([]);
  const [jobs, setJobs] = useState<Job[]>([]);
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [quoteID, setQuoteID] = useState("");
  const [technicianID, setTechnicianID] = useState("");
  const [startsAt, setStartsAt] = useState("");
  const [alert, setAlert] = useState("");
  const [success, setSuccess] = useState("");
  const [rescheduleState, setRescheduleState] = useState<Record<number, { technicianID: string; startsAt: string }>>({});

  const refreshQuotes = useCallback(async () => {
    setQuotes((await listQuotes("unscheduled")).quotes);
  }, []);
  const refreshJobs = useCallback(async () => {
    setJobs((await listJobs()).jobs);
  }, []);
  const refreshNotifications = useCallback(async () => {
    setNotifications((await listNotifications()).notifications);
  }, []);
  const refreshAll = useCallback(async () => {
    const [me, quoteData, technicianData, jobData, notificationData] = await Promise.all([
      getMe(),
      listQuotes("unscheduled"),
      listTechnicians(),
      listJobs(),
      listNotifications()
    ]);
    setUser(me.user);
    setQuotes(quoteData.quotes);
    setTechnicians(technicianData.technicians);
    setJobs(jobData.jobs);
    setNotifications(notificationData.notifications);
  }, []);

  useEffect(() => {
    refreshAll().catch(() => {
      window.location.href = "/login";
    });
  }, [refreshAll]);

  useEffect(() => {
    return openEventSocket((event) => {
      if (event.type === "notification.created") void refreshNotifications();
      if (event.type === "jobs.changed") void refreshJobs();
      if (event.type === "quotes.changed") void refreshQuotes();
    });
  }, [refreshJobs, refreshNotifications, refreshQuotes]);

  async function handleAssign() {
    setAlert("");
    setSuccess("");
    try {
      await assignJob({
        quoteId: Number(quoteID),
        technicianId: Number(technicianID),
        startsAt: new Date(startsAt).toISOString()
      });
      setQuoteID("");
      setTechnicianID("");
      setStartsAt("");
      setSuccess("Job assigned.");
      await Promise.all([refreshQuotes(), refreshJobs(), refreshNotifications()]);
    } catch (err) {
      setSuccess("");
      setAlert(errorMessage(err, "Assignment failed."));
    }
  }

  async function handleReschedule(job: Job) {
    const state = rescheduleState[job.id];
    if (!state?.technicianID || !state.startsAt) return;
    setAlert("");
    setSuccess("");
    try {
      await rescheduleJob(job.id, {
        technicianId: Number(state.technicianID),
        startsAt: new Date(state.startsAt).toISOString()
      });
      setSuccess("Job rescheduled.");
      await Promise.all([refreshJobs(), refreshNotifications()]);
    } catch (err) {
      setSuccess("");
      setAlert(errorMessage(err, "Reschedule failed."));
    }
  }

  return (
    <Box sx={{ minHeight: "100vh", bgcolor: "background.default" }}>
      <AppBar position="static" color="primary">
        <Toolbar>
          <Typography sx={{ flex: 1 }} variant="h6">
            Brix Scheduler
          </Typography>
          <Typography sx={{ mr: 2 }}>{user?.displayName}</Typography>
          <Button color="inherit" onClick={() => logout().finally(() => (window.location.href = "/login"))}>
            Logout
          </Button>
        </Toolbar>
      </AppBar>
      <Container sx={{ py: 3 }} maxWidth="lg">
        <Stack spacing={3}>
          {success ? <Alert severity="success">{success}</Alert> : null}
          {alert ? <Alert severity="error">{alert}</Alert> : null}
          <Paper sx={{ p: 2 }}>
            <Typography variant="h6">Notifications</Typography>
            <List dense>
              {notifications.map((notification) => (
                <ListItem key={notification.id}>
                  <ListItemText primary={notification.message} />
                </ListItem>
              ))}
              {notifications.length === 0 ? <ListItem><ListItemText primary="No notifications" /></ListItem> : null}
            </List>
          </Paper>
          <Paper sx={{ p: 2 }}>
            <Typography variant="h6" sx={{ mb: 2 }}>
              Unscheduled quotes
            </Typography>
            <Stack spacing={2}>
              <FormControl fullWidth>
                <InputLabel variant="standard" htmlFor="quote-select">
                  Quote
                </InputLabel>
                <NativeSelect id="quote-select" value={quoteID} onChange={(event) => setQuoteID(event.target.value)}>
                  <option value="" />
                  {quotes.map((quote) => (
                    <option key={quote.id} value={quote.id}>
                      {quote.customerName}
                    </option>
                  ))}
                </NativeSelect>
              </FormControl>
              <List dense aria-label="Unscheduled quote list">
                {quotes.map((quote) => (
                  <ListItem key={quote.id} disablePadding>
                    <ListItemText primary={quote.customerName} secondary={quote.description} />
                  </ListItem>
                ))}
              </List>
              <FormControl fullWidth>
                <InputLabel variant="standard" htmlFor="technician-select">
                  Technician
                </InputLabel>
                <NativeSelect id="technician-select" value={technicianID} onChange={(event) => setTechnicianID(event.target.value)}>
                  <option value="" />
                  {technicians.map((technician) => (
                    <option key={technician.id} value={technician.id}>
                      {technician.displayName}
                    </option>
                  ))}
                </NativeSelect>
              </FormControl>
              <TextField
                label="Start time"
                type="datetime-local"
                value={startsAt}
                onChange={(event) => setStartsAt(event.target.value)}
                slotProps={{ inputLabel: { shrink: true } }}
              />
              <Button variant="contained" disabled={!quoteID || !technicianID || !startsAt} onClick={handleAssign}>
                Assign
              </Button>
            </Stack>
          </Paper>
          <Paper sx={{ p: 2 }}>
            <Typography variant="h6" sx={{ mb: 2 }}>
              Manager jobs
            </Typography>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>Quote</TableCell>
                  <TableCell>Technician</TableCell>
                  <TableCell>Window</TableCell>
                  <TableCell>Status</TableCell>
                  <TableCell>Schedule</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {jobs.map((job) => (
                  <TableRow key={job.id}>
                    <TableCell>{job.quoteCustomerName}</TableCell>
                    <TableCell>{job.technicianName}</TableCell>
                    <TableCell>{formatWindow(job)}</TableCell>
                    <TableCell><Chip size="small" label={job.status} /></TableCell>
                    <TableCell>
                      {job.status === "scheduled" ? (
                        <Stack direction="row" spacing={1}>
                          <NativeSelect
                            inputProps={{ "aria-label": `Reschedule assignee ${job.id}` }}
                            value={rescheduleState[job.id]?.technicianID || String(job.technicianId)}
                            onChange={(event) =>
                              setRescheduleState((current) => ({
                                ...current,
                                [job.id]: { technicianID: event.target.value, startsAt: current[job.id]?.startsAt || "" }
                              }))
                            }
                          >
                            {technicians.map((technician) => (
                              <option key={technician.id} value={technician.id}>
                                {technician.displayName}
                              </option>
                            ))}
                          </NativeSelect>
                          <TextField
                            type="datetime-local"
                            size="small"
                            value={rescheduleState[job.id]?.startsAt || ""}
                            onChange={(event) =>
                              setRescheduleState((current) => ({
                                ...current,
                                [job.id]: { technicianID: current[job.id]?.technicianID || String(job.technicianId), startsAt: event.target.value }
                              }))
                            }
                            slotProps={{ htmlInput: { "aria-label": `Reschedule start ${job.id}` } }}
                          />
                          <Button size="small" variant="outlined" onClick={() => handleReschedule(job)}>
                            Reschedule
                          </Button>
                        </Stack>
                      ) : null}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </Paper>
        </Stack>
      </Container>
    </Box>
  );
}

function formatWindow(job: Job): string {
  return `${new Date(job.startsAt).toLocaleString()} - ${new Date(job.endsAt).toLocaleString()}`;
}

function errorMessage(err: unknown, fallback: string): string {
  if (err instanceof Error) return err.message;
  if (typeof err === "object" && err !== null && "message" in err && typeof err.message === "string") {
    return err.message;
  }
  return fallback;
}
