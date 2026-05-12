"use client";

import Alert from "@mui/joy/Alert";
import Button from "@mui/joy/Button";
import Chip from "@mui/joy/Chip";
import Divider from "@mui/joy/Divider";
import Sheet from "@mui/joy/Sheet";
import Snackbar from "@mui/joy/Snackbar";
import Stack from "@mui/joy/Stack";
import Table from "@mui/joy/Table";
import Typography from "@mui/joy/Typography";
import { useCallback, useEffect, useState } from "react";
import { AppShell } from "../../components/AppShell";
import {
  Job,
  Notification,
  User,
  completeJob,
  getMe,
  listJobs,
  listNotifications,
  logout,
  markNotificationRead,
  openEventSocket
} from "../../lib/api";
import { formatScheduleWindow } from "../../lib/schedulingAvailability";

type ToastState = {
  color: "success" | "danger" | "primary";
  message: string;
};

export default function TechnicianPage() {
  const [user, setUser] = useState<User | null>(null);
  const [jobs, setJobs] = useState<Job[]>([]);
  const [notifications, setNotifications] = useState<Notification[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const [toast, setToast] = useState<ToastState | null>(null);

  const showToast = useCallback((message: string, color: ToastState["color"] = "primary") => {
    setToast({ message, color });
  }, []);

  const applyNotifications = useCallback((data: { notifications: Notification[]; unreadCount?: number }) => {
    setNotifications(data.notifications);
    setUnreadCount(data.unreadCount ?? data.notifications.filter((notification) => notification.readAt === null).length);
    return data.notifications;
  }, []);

  const refreshJobs = useCallback(async () => {
    setJobs((await listJobs()).jobs);
  }, []);
  const refreshNotifications = useCallback(async () => {
    return applyNotifications(await listNotifications());
  }, [applyNotifications]);
  const refreshAll = useCallback(async () => {
    const [me, jobData, notificationData] = await Promise.all([getMe(), listJobs(), listNotifications()]);
    setUser(me.user);
    setJobs(jobData.jobs);
    applyNotifications(notificationData);
  }, [applyNotifications]);

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
    });
  }, [refreshJobs, refreshNotifications, showToast]);

  useEffect(() => {
    const refreshStaleData = () => {
      void refreshAll();
    };
    window.addEventListener("focus", refreshStaleData);
    return () => window.removeEventListener("focus", refreshStaleData);
  }, [refreshAll]);

  async function handleComplete(jobID: number) {
    try {
      await completeJob(jobID);
      showToast("Job completed.", "success");
      await Promise.all([refreshJobs(), refreshNotifications()]);
    } catch (err) {
      showToast(errorMessage(err, "Completion failed."), "danger");
    }
  }

  async function handleMarkNotificationRead(notificationID: number) {
    await markNotificationRead(notificationID);
    await refreshNotifications();
  }

  return (
    <AppShell
      activePage="technician-jobs"
      notificationCount={unreadCount}
      notifications={notifications}
      onLogout={() => logout().finally(() => (window.location.href = "/login"))}
      onMarkNotificationRead={handleMarkNotificationRead}
      roleLabel="Technician workspace"
      unreadCount={unreadCount}
      userName={user?.displayName || "Loading"}
    >
      <Stack spacing={3}>
        <Stack spacing={1}>
          <Typography component="h1" level="h2">
            Assigned jobs
          </Typography>
          <Divider />
        </Stack>
        <Sheet variant="outlined" sx={{ overflow: "auto" }}>
          <Table aria-label="Assigned jobs" data-table-interface="manager" stickyHeader hoverRow sx={tableSx}>
            <thead>
              <tr>
                <th>Quote</th>
                <th>Manager</th>
                <th>Window</th>
                <th>Status</th>
                <th>Action</th>
              </tr>
            </thead>
            <tbody>
              {jobs.map((job) => (
                <tr key={job.id}>
                  <td>{job.quoteCustomerName}</td>
                  <td>{job.managerName}</td>
                  <td>{formatScheduleWindow(job)}</td>
                  <td>
                    <Chip size="sm" color={job.status === "scheduled" ? "primary" : "success"} variant="soft">
                      {job.status}
                    </Chip>
                  </td>
                  <td>
                    {job.status === "scheduled" ? (
                      <Button onClick={() => handleComplete(job.id)}>
                        Complete
                      </Button>
                    ) : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </Table>
        </Sheet>
      </Stack>
      <Snackbar autoHideDuration={4000} color={toast?.color || "primary"} onClose={setToast.bind(null, null)} open={toast !== null} variant="soft">
        <Alert aria-label="Notification popup" color={toast?.color || "primary"} role="alert" variant="soft">
          {toast?.message}
        </Alert>
      </Snackbar>
    </AppShell>
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
