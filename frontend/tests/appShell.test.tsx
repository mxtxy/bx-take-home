import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, test, vi } from "vitest";
import RootLayout, { metadata } from "../app/layout";
import { AppShell } from "../components/AppShell";
import { AppProviders } from "../components/AppProviders";

describe("app shell", () => {
  test("metadata describes the app", () => {
    expect(metadata).toEqual({
      title: "Brix Scheduler",
      description: "Service scheduling and notification system"
    });
  });

  test("AppProviders renders children inside the Joy UI theme", () => {
    render(
      <AppProviders>
        <span>Provided child</span>
      </AppProviders>
    );

    expect(screen.getByText("Provided child")).toBeInTheDocument();
  });

  test("RootLayout wraps children with html, color script, and Joy providers", () => {
    const element = RootLayout({ children: <span>Layout child</span> });
    const bodyChildren = element.props.children.props.children;

    expect(element.type).toBe("html");
    expect(element.props.lang).toBe("en");
    expect(element.props.suppressHydrationWarning).toBe(true);
    expect(element.props.children.type).toBe("body");
    expect(bodyChildren[1].type).toBe(AppProviders);
  });

  test("AppShell renders reference sidebar navigation and logout controls", async () => {
    const logout = vi.fn();

    render(
      <AppProviders>
        <AppShell userName="Sarah Manager" roleLabel="Manager" activePage="manager-quotes" notificationCount={2} onLogout={logout}>
          <span>Dashboard child</span>
        </AppShell>
      </AppProviders>
    );

    expect(screen.getByRole("banner")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Open navigation" })).toBeInTheDocument();
    expect(screen.getByRole("navigation", { name: "Primary navigation" })).toHaveTextContent("Quotes");
    expect(screen.getByRole("link", { name: "Quotes" })).toHaveAttribute("href", "/manager/quotes");
    expect(screen.getByRole("link", { name: "Assign quotes" })).toHaveAttribute("href", "/manager/assign");
    expect(screen.getByRole("link", { name: "Jobs" })).toHaveAttribute("href", "/manager/jobs");
    expect(screen.queryByRole("link", { name: "Technician workspace" })).not.toBeInTheDocument();
    expect(screen.getByText("2 unread")).toBeInTheDocument();
    expect(screen.queryByRole("radiogroup", { name: "Accent color" })).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Violet accent")).not.toBeInTheDocument();
    expect(screen.getByText("Dashboard child")).toBeInTheDocument();

    await userEvent.click(screen.getByRole("button", { name: "Logout" }));

    expect(logout).toHaveBeenCalledOnce();
  });

  test("Notifications action sits between the identity separator and identity controls", () => {
    render(
      <AppProviders>
        <AppShell userName="Sarah Manager" roleLabel="Manager" activePage="manager-quotes" notificationCount={2} onLogout={() => undefined}>
          <span>Dashboard child</span>
        </AppShell>
      </AppProviders>
    );

    const separator = screen.getByRole("separator", { name: "Identity separator" });
    const bottomControls = screen.getByRole("group", { name: "Sidebar account controls" });
    const notificationButton = screen.getByRole("button", { name: "Open notifications" });
    const identity = screen.getByText("Sarah");

    expect(notificationButton).toHaveTextContent("Notifications");
    expect(bottomControls).toContainElement(separator);
    expect(bottomControls).toContainElement(notificationButton);
    expect(bottomControls).toContainElement(identity);
    expect(separator.compareDocumentPosition(notificationButton)).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
    expect(notificationButton.compareDocumentPosition(identity)).toBe(Node.DOCUMENT_POSITION_FOLLOWING);
    expect(getComputedStyle(bottomControls).gap).toBe("8px");
  });

  test("AppShell lists workspace only below the product title and splits identity name", () => {
    render(
      <AppProviders>
        <AppShell userName="Sarah Manager" roleLabel="Manager workspace" activePage="manager-quotes" notificationCount={0} onLogout={() => undefined}>
          <span>Dashboard child</span>
        </AppShell>
      </AppProviders>
    );

    expect(screen.getAllByText("Manager workspace")).toHaveLength(1);
    expect(screen.queryByText("Sarah Manager")).not.toBeInTheDocument();
    expect(screen.getByText("Sarah")).toBeInTheDocument();
    expect(screen.getByText("Manager")).toBeInTheDocument();
  });

  test("AppShell renders technician navigation without manager pages", () => {
    render(
      <AppProviders>
        <AppShell userName="Tom Technician" roleLabel="Technician" activePage="technician-jobs" notificationCount={0} onLogout={() => undefined}>
          <span>Dashboard child</span>
        </AppShell>
      </AppProviders>
    );

    expect(screen.getByRole("navigation", { name: "Primary navigation" })).toHaveTextContent("Assigned jobs");
    expect(screen.getByRole("link", { name: "Assigned jobs" })).toHaveAttribute("href", "/technician");
    expect(screen.queryByRole("link", { name: "Assign quotes" })).not.toBeInTheDocument();
  });

  test("AppShell falls back to product initial when user name is empty", () => {
    render(
      <AppProviders>
        <AppShell userName="" roleLabel="Manager" activePage="manager-quotes" notificationCount={0} onLogout={() => undefined}>
          <span>Dashboard child</span>
        </AppShell>
      </AppProviders>
    );

    expect(screen.getAllByText("B").length).toBeGreaterThan(0);
  });

  test("Mobile navigation opens and closes from shell controls", async () => {
    render(
      <AppProviders>
        <AppShell userName="Sarah Manager" roleLabel="Manager" activePage="manager-quotes" notificationCount={0} onLogout={() => undefined}>
          <span>Dashboard child</span>
        </AppShell>
      </AppProviders>
    );

    await userEvent.click(screen.getByRole("button", { name: "Open navigation" }));
    await userEvent.click(screen.getByRole("button", { name: "Close navigation" }));

    expect(screen.getByRole("navigation", { name: "Primary navigation" })).toBeInTheDocument();
  });

  test("Mobile navigation closes from the backdrop", async () => {
    render(
      <AppProviders>
        <AppShell userName="Sarah Manager" roleLabel="Manager" activePage="manager-quotes" notificationCount={0} onLogout={() => undefined}>
          <span>Dashboard child</span>
        </AppShell>
      </AppProviders>
    );

    await userEvent.click(screen.getByRole("button", { name: "Open navigation" }));
    await userEvent.click(screen.getByTestId("sidebar-overlay"));

    expect(screen.getByRole("navigation", { name: "Primary navigation" })).toBeInTheDocument();
  });

  test("Notification drawer shows read notifications and closes from Escape", async () => {
    render(
      <AppProviders>
        <AppShell
          userName="Sarah Manager"
          roleLabel="Manager"
          activePage="manager"
          notifications={[{ id: 1, message: "Read notification", readAt: "2026-05-12T11:00:00Z", createdAt: "2026-05-12T10:00:00Z" }]}
          notificationCount={1}
          unreadCount={0}
          onLogout={() => undefined}
        >
          <span>Dashboard child</span>
        </AppShell>
      </AppProviders>
    );

    await userEvent.click(screen.getByRole("button", { name: "Open notifications" }));

    expect(screen.getByRole("dialog", { name: "Notifications" })).toHaveTextContent("Read notification");
    expect(screen.getByRole("dialog", { name: "Notifications" })).toHaveTextContent("12 May");

    await userEvent.keyboard("{Escape}");

    await waitFor(() => expect(screen.queryByRole("dialog", { name: "Notifications" })).not.toBeInTheDocument());
  });

  test("Notification drawer renders an empty state", async () => {
    render(
      <AppProviders>
        <AppShell userName="Sarah Manager" roleLabel="Manager" activePage="manager" notificationCount={0} onLogout={() => undefined}>
          <span>Dashboard child</span>
        </AppShell>
      </AppProviders>
    );

    await userEvent.click(screen.getByRole("button", { name: "Open notifications" }));

    expect(screen.getByRole("dialog", { name: "Notifications" })).toHaveTextContent("No notifications");
  });

  test("Notification drawer keeps a full-height right pane aligned to the notification panel width", async () => {
    render(
      <AppProviders>
        <AppShell userName="Sarah Manager" roleLabel="Manager" activePage="manager" notificationCount={0} onLogout={() => undefined}>
          <span>Dashboard child</span>
        </AppShell>
      </AppProviders>
    );

    await userEvent.click(screen.getByRole("button", { name: "Open notifications" }));

    const dialog = screen.getByRole("dialog", { name: "Notifications" });
    const drawerContent = dialog.parentElement;

    expect(drawerContent).toHaveClass("MuiDrawer-content");
    expect(getComputedStyle(drawerContent!).getPropertyValue("--Drawer-horizontalSize").trim()).toBe("380px");
    expect(drawerContent).toHaveStyle({ width: "min(100vw, var(--Drawer-horizontalSize))" });
    expect(drawerContent).toHaveStyle({ height: "100%" });
    expect(dialog).toHaveStyle({ height: "100%", width: "100%" });
    const drawerContentRule = document.head.textContent!.match(/\.css-[^{]+-JoyDrawer-content\{[^}]*\}/)?.[0] ?? "";
    expect(drawerContentRule).toMatch(/right:\s*0/);
    expect(drawerContentRule).toMatch(/top:\s*0/);
    expect(drawerContentRule).not.toMatch(/right:\s*24px/);
    expect(drawerContentRule).not.toMatch(/top:\s*24px/);
    expect(drawerContentRule).not.toMatch(/max-height:\s*calc\(100dvh - 48px\)/);
  });

  test("Notification drawer marks unread notifications as read", async () => {
    const markRead = vi.fn().mockResolvedValue(undefined);

    render(
      <AppProviders>
        <AppShell
          userName="Sarah Manager"
          roleLabel="Manager"
          activePage="manager-quotes"
          notifications={[
            { id: 1, message: "Unread notification", readAt: null, createdAt: "2026-05-12T10:00:00Z" },
            { id: 2, message: "Read notification", readAt: "2026-05-12T11:00:00Z", createdAt: "2026-05-12T10:00:00Z" }
          ]}
          notificationCount={1}
          unreadCount={1}
          onLogout={() => undefined}
          onMarkNotificationRead={markRead}
        >
          <span>Dashboard child</span>
        </AppShell>
      </AppProviders>
    );

    await userEvent.click(screen.getByRole("button", { name: "Open notifications" }));
    await userEvent.click(screen.getByRole("button", { name: "Mark notification 1 as read" }));

    expect(markRead).toHaveBeenCalledWith(1);
    expect(screen.queryByRole("button", { name: "Mark notification 2 as read" })).not.toBeInTheDocument();
  });

  test("Theme controls switch between light and dark modes", async () => {
    render(
      <AppProviders>
        <AppShell userName="Tom Technician" roleLabel="Technician" activePage="technician" notificationCount={0} onLogout={() => undefined}>
          <span>Dashboard child</span>
        </AppShell>
      </AppProviders>
    );

    const toggle = await screen.findByRole("button", { name: "Toggle dark mode" });
    await userEvent.click(toggle);

    await waitFor(() => expect(screen.getByRole("button", { name: "Toggle light mode" })).toBeInTheDocument());
    await userEvent.click(screen.getByRole("button", { name: "Toggle light mode" }));

    await waitFor(() => expect(screen.getByRole("button", { name: "Toggle dark mode" })).toBeInTheDocument());
  });

  test("AppShell keeps Joy default colors without custom accent state", () => {
    render(
      <AppProviders>
        <AppShell userName="Sarah Manager" roleLabel="Manager" activePage="manager" notificationCount={0} onLogout={() => undefined}>
          <span>Dashboard child</span>
        </AppShell>
      </AppProviders>
    );

    expect(screen.queryByRole("radiogroup", { name: "Accent color" })).not.toBeInTheDocument();
    expect(document.documentElement).not.toHaveAttribute("data-brix-accent");
  });
});
