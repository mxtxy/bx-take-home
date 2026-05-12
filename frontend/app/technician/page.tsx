"use client";

import {
  AppBar,
  Box,
  Button,
  Chip,
  Container,
  List,
  ListItem,
  ListItemText,
  Paper,
  Stack,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  Toolbar,
  Typography
} from "@mui/material";
import { useCallback, useEffect, useState } from "react";
import {
  Job,
  Notification,
  User,
  completeJob,
  getMe,
  listJobs,
  listNotifications,
  logout,
  openEventSocket
} from "../../lib/api";

export default function TechnicianPage() {
  const [user, setUser] = useState<User | null>(null);
  const [jobs, setJobs] = useState<Job[]>([]);
  const [notifications, setNotifications] = useState<Notification[]>([]);

  const refreshJobs = useCallback(async () => {
    setJobs((await listJobs()).jobs);
  }, []);
  const refreshNotifications = useCallback(async () => {
    setNotifications((await listNotifications()).notifications);
  }, []);
  const refreshAll = useCallback(async () => {
    const [me, jobData, notificationData] = await Promise.all([getMe(), listJobs(), listNotifications()]);
    setUser(me.user);
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
    });
  }, [refreshJobs, refreshNotifications]);

  async function handleComplete(jobID: number) {
    await completeJob(jobID);
    await Promise.all([refreshJobs(), refreshNotifications()]);
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
              Assigned jobs
            </Typography>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>Quote</TableCell>
                  <TableCell>Manager</TableCell>
                  <TableCell>Window</TableCell>
                  <TableCell>Status</TableCell>
                  <TableCell>Action</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {jobs.map((job) => (
                  <TableRow key={job.id}>
                    <TableCell>{job.quoteCustomerName}</TableCell>
                    <TableCell>{job.managerName}</TableCell>
                    <TableCell>{formatWindow(job)}</TableCell>
                    <TableCell><Chip size="small" label={job.status} /></TableCell>
                    <TableCell>
                      {job.status === "scheduled" ? (
                        <Button variant="contained" size="small" onClick={() => handleComplete(job.id)}>
                          Complete
                        </Button>
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
