"use client";

import Avatar from "@mui/joy/Avatar";
import Box from "@mui/joy/Box";
import Button from "@mui/joy/Button";
import Drawer from "@mui/joy/Drawer";
import Divider from "@mui/joy/Divider";
import GlobalStyles from "@mui/joy/GlobalStyles";
import IconButton from "@mui/joy/IconButton";
import List from "@mui/joy/List";
import ListItem from "@mui/joy/ListItem";
import ListItemButton from "@mui/joy/ListItemButton";
import ListItemContent from "@mui/joy/ListItemContent";
import Sheet from "@mui/joy/Sheet";
import Stack from "@mui/joy/Stack";
import Tooltip from "@mui/joy/Tooltip";
import Typography from "@mui/joy/Typography";
import { useColorScheme } from "@mui/joy/styles";
import { Bell, Briefcase, CalendarPlus, ClipboardList, LogOut, Menu, Moon, Sun, X } from "lucide-react";
import { useState, type ReactNode } from "react";

type ShellNotification = {
  id: number;
  message: string;
  readAt: string | null;
  createdAt: string;
};

type AppShellProps = {
  activePage: "manager" | "manager-quotes" | "manager-assign" | "manager-jobs" | "technician" | "technician-jobs";
  children: ReactNode;
  notifications?: ShellNotification[];
  notificationCount: number;
  unreadCount?: number;
  onLogout: () => void;
  onMarkNotificationRead?: (notificationID: number) => Promise<void> | void;
  roleLabel: string;
  userName: string;
};

export function AppShell({
  activePage,
  children,
  notifications = [],
  notificationCount,
  onLogout,
  onMarkNotificationRead,
  roleLabel,
  unreadCount,
  userName
}: AppShellProps) {
  const [notificationsOpen, setNotificationsOpen] = useState(false);
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const count = unreadCount ?? notificationCount;
  const nameLines = splitIdentityName(userName);
  const isTechnician = activePage === "technician" || activePage === "technician-jobs";
  const navigation = isTechnician
    ? [{ href: "/technician", label: "Assigned jobs", page: "technician-jobs" as const, Icon: Briefcase }]
    : [
        { href: "/manager/quotes", label: "Quotes", page: "manager-quotes" as const, Icon: ClipboardList },
        { href: "/manager/assign", label: "Assign quotes", page: "manager-assign" as const, Icon: CalendarPlus },
        { href: "/manager/jobs", label: "Jobs", page: "manager-jobs" as const, Icon: Briefcase }
      ];

  return (
    <Box sx={{ minHeight: "100dvh", bgcolor: "background.body", display: "flex" }}>
      <GlobalStyles
        styles={(theme) => ({
          ":root": {
            "--Sidebar-width": "220px",
            "--Header-height": "52px",
            [theme.breakpoints.up("lg")]: {
              "--Sidebar-width": "240px"
            }
          }
        })}
      />
      <Sheet
        component="header"
        variant="outlined"
        sx={{
          alignItems: "center",
          display: { xs: "flex", md: "none" },
          gap: 1,
          height: "var(--Header-height)",
          justifyContent: "space-between",
          borderRadius: 0,
          borderLeft: 0,
          borderRight: 0,
          borderTop: 0,
          boxShadow: "sm",
          p: 2,
          position: "fixed",
          top: 0,
          width: "100vw",
          zIndex: 9995
        }}
      >
        <IconButton aria-label="Open navigation" color="neutral" onClick={() => setSidebarOpen(true)} size="sm" variant="outlined">
          <Menu size={18} />
        </IconButton>
        <Typography level="title-md">Brix Scheduler</Typography>
      </Sheet>

      <Sheet
        className="Sidebar"
        sx={{
          borderRadius: 0,
          borderRight: "1px solid",
          borderColor: "divider",
          display: "flex",
          flexDirection: "column",
          flexShrink: 0,
          gap: 2,
          height: "100dvh",
          p: 2,
          position: { xs: "fixed", md: "sticky" },
          top: 0,
          transform: { xs: sidebarOpen ? "translateX(0)" : "translateX(-100%)", md: "none" },
          transition: "transform 0.25s ease",
          width: "var(--Sidebar-width)",
          zIndex: 10000
        }}
        variant="outlined"
      >
        <Box
          aria-hidden="true"
          data-testid="sidebar-overlay"
          onClick={() => setSidebarOpen(false)}
          sx={{
            bgcolor: "background.backdrop",
            display: { xs: sidebarOpen ? "block" : "none", md: "none" },
            height: "100vh",
            left: "var(--Sidebar-width)",
            opacity: 0.45,
            position: "fixed",
            top: 0,
            width: "100vw",
            zIndex: -1
          }}
        />
        <Stack direction="row" spacing={1} sx={{ alignItems: "center" }}>
          <Avatar variant="soft" color="primary" size="sm">
            B
          </Avatar>
          <Box sx={{ minWidth: 0, flex: 1 }}>
            <Typography level="title-lg">Brix Scheduler</Typography>
            <Typography level="body-xs" color="neutral">
              {roleLabel}
            </Typography>
          </Box>
          <IconButton aria-label="Close navigation" color="neutral" onClick={() => setSidebarOpen(false)} size="sm" sx={{ display: { xs: "inline-flex", md: "none" } }} variant="plain">
            <X size={18} />
          </IconButton>
        </Stack>

        <Box component="nav" aria-label="Primary navigation" sx={{ minHeight: 0, overflow: "hidden auto" }}>
          <List size="sm" sx={{ gap: 1, "--ListItem-radius": "8px" }}>
            {navigation.map((item) => (
              <ListItem key={item.href}>
                <ListItemButton component="a" href={item.href} selected={activePage === item.page || (activePage === "manager" && item.page === "manager-quotes") || (activePage === "technician" && item.page === "technician-jobs")}>
                  <item.Icon size={18} />
                  <ListItemContent>
                    <Typography level="title-sm">{item.label}</Typography>
                  </ListItemContent>
                </ListItemButton>
              </ListItem>
            ))}
          </List>
        </Box>

        <Stack aria-label="Sidebar account controls" role="group" spacing={0} sx={{ gap: 1, mt: "auto" }}>
          <Divider aria-label="Identity separator" />

          <List aria-label="Account actions" size="sm" sx={{ "--ListItem-radius": "8px", py: 0 }}>
            <ListItem>
              <ListItemButton aria-label="Open notifications" onClick={() => setNotificationsOpen(true)}>
                <Bell size={18} />
                <ListItemContent>
                  <Typography level="title-sm">Notifications</Typography>
                  <Typography level="body-xs" color={count > 0 ? "primary" : "neutral"}>
                    {count} unread
                  </Typography>
                </ListItemContent>
              </ListItemButton>
            </ListItem>
          </List>

          <Stack direction="row" spacing={1} sx={{ alignItems: "center" }}>
            <Avatar size="sm" variant="outlined">
              {initials(userName)}
            </Avatar>
            <Box sx={{ minWidth: 0, flex: 1 }}>
              {nameLines.map((line, index) => (
                <Typography key={`${line}-${index}`} level="title-sm" noWrap sx={{ lineHeight: 1.15 }}>
                  {line}
                </Typography>
              ))}
            </Box>
            <ColorModeButton />
            <Tooltip title="Logout">
              <IconButton aria-label="Logout" color="neutral" onClick={onLogout} size="sm" variant="plain">
                <LogOut size={18} />
              </IconButton>
            </Tooltip>
          </Stack>
        </Stack>
      </Sheet>

      <Box
        component="main"
        sx={{
          flex: 1,
          minWidth: 0,
          height: "100dvh",
          overflow: "auto",
          px: { xs: 2, md: 6 },
          pt: { xs: "calc(12px + var(--Header-height))", md: 3 },
          pb: { xs: 2, md: 3 }
        }}
      >
        {children}
      </Box>

      {notificationsOpen ? (
        <Drawer anchor="right" open onClose={() => setNotificationsOpen(false)}>
          <Sheet
            aria-label="Notifications"
            role="dialog"
            sx={{ height: "100%", maxWidth: 380, p: 2, width: "min(100vw, 380px)" }}
            variant="outlined"
          >
            <Stack spacing={2}>
              <Box>
                <Typography level="h2" sx={{ fontSize: "1.25rem" }}>
                  Notifications
                </Typography>
                <Typography level="body-sm" color="neutral">
                  {count} unread
                </Typography>
              </Box>
              <List size="sm" sx={{ "--ListItem-paddingY": "0.65rem" }}>
                {notifications.map((notification) => (
                  <ListItem key={notification.id} sx={{ display: "block" }}>
                    <Stack spacing={0.75}>
                      <Typography level="body-sm">{notification.message}</Typography>
                      <Typography level="body-xs" color={notification.readAt ? "neutral" : "primary"}>
                        {formatNotificationTime(notification.createdAt)}
                      </Typography>
                      {notification.readAt === null && onMarkNotificationRead ? (
                        <Button
                          aria-label={`Mark notification ${notification.id} as read`}
                          color="neutral"
                          onClick={() => onMarkNotificationRead(notification.id)}
                          size="sm"
                          variant="outlined"
                        >
                          Mark as read
                        </Button>
                      ) : null}
                    </Stack>
                  </ListItem>
                ))}
                {notifications.length === 0 ? (
                  <ListItem>
                    <Typography level="body-sm">No notifications</Typography>
                  </ListItem>
                ) : null}
              </List>
            </Stack>
          </Sheet>
        </Drawer>
      ) : null}
    </Box>
  );
}

function initials(value: string): string {
  return value
    .split(" ")
    .filter(Boolean)
    .map((part) => part[0])
    .join("")
    .slice(0, 2)
    .toUpperCase() || "B";
}

function splitIdentityName(value: string): string[] {
  const parts = value.trim().split(/\s+/).filter(Boolean);
  if (parts.length <= 1) return parts;
  return [parts[0], parts[parts.length - 1]];
}

function ColorModeButton() {
  const { mode, setMode } = useColorScheme();
  const dark = mode === "dark";
  const label = dark ? "Toggle light mode" : "Toggle dark mode";
  const Icon = dark ? Sun : Moon;

  return (
    <Tooltip title={label}>
      <IconButton aria-label={label} color="neutral" onClick={() => setMode(dark ? "light" : "dark")} size="sm" variant="outlined">
        <Icon size={16} />
      </IconButton>
    </Tooltip>
  );
}

function formatNotificationTime(value: string): string {
  return new Date(value).toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit"
  });
}
