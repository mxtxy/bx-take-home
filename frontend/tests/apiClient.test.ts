import { beforeEach, describe, expect, test, vi } from "vitest";

describe("api client", () => {
  beforeEach(() => {
    vi.resetModules();
    vi.unstubAllGlobals();
    process.env.NEXT_PUBLIC_API_URL = "http://api.test";
    delete process.env.NEXT_PUBLIC_WS_URL;
  });

  test("API client uses NEXT_PUBLIC_API_URL", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ ok: true }));
    vi.stubGlobal("fetch", fetchMock);
    const { getMe } = await import("../lib/api");

    await getMe();

    expect(fetchMock).toHaveBeenCalledWith(
      "http://api.test/api/me",
      expect.objectContaining({ credentials: "include" })
    );
  });

  test("API client sends credentials include", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ quotes: [] }));
    vi.stubGlobal("fetch", fetchMock);
    const { listQuotes } = await import("../lib/api");

    await listQuotes("unscheduled");

    expect(fetchMock.mock.calls[0][1]).toMatchObject({ credentials: "include" });
  });

  test("API client maps error response shape to readable error object", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        jsonResponse(
          { error: { code: "schedule_conflict", message: "Technician already has a job in that time window." } },
          409
        )
      )
    );
    const { assignJob, ApiError } = await import("../lib/api");

    await expect(assignJob({ quoteId: 1, technicianId: 1, startsAt: "2026-05-12T10:00:00Z" })).rejects.toMatchObject({
      code: "schedule_conflict",
      status: 409,
      message: "Technician already has a job in that time window."
    });
    expect(ApiError).toBeDefined();
  });

  test("API client uses localhost backend URL by default", async () => {
    delete process.env.NEXT_PUBLIC_API_URL;
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ ok: true }));
    vi.stubGlobal("fetch", fetchMock);
    const { getMe } = await import("../lib/api");

    await getMe();

    expect(fetchMock).toHaveBeenCalledWith(
      "http://localhost:8080/api/me",
      expect.objectContaining({ credentials: "include" })
    );
  });

  test("API client maps missing error details to fallback ApiError", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(jsonResponse({}, 500)));
    const { listJobs } = await import("../lib/api");

    await expect(listJobs()).rejects.toMatchObject({
      code: "internal_error",
      status: 500,
      message: "Request failed."
    });
  });

  test("API client serializes assignment request without endsAt", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ job: { id: 1 } }));
    vi.stubGlobal("fetch", fetchMock);
    const { assignJob } = await import("../lib/api");

    await assignJob({ quoteId: 1, technicianId: 2, startsAt: "2026-05-12T10:00:00Z" });

    const body = JSON.parse(fetchMock.mock.calls[0][1].body as string);
    expect(body).toEqual({ quoteId: 1, technicianId: 2, startsAt: "2026-05-12T10:00:00Z" });
    expect(body.endsAt).toBeUndefined();
  });

  test("API client serializes exported command methods", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ ok: true }));
    vi.stubGlobal("fetch", fetchMock);
    const {
      completeJob,
      listJobs,
      listNotifications,
      listQuotes,
      listTechnicians,
      login,
      logout,
      markNotificationRead,
      rescheduleJob
    } = await import("../lib/api");

    await login("manager1@brix.test", "password123");
    await logout();
    await listQuotes();
    await listTechnicians();
    await listJobs();
    await rescheduleJob(7, { technicianId: 2, startsAt: "2026-05-12T13:00:00Z" });
    await completeJob(7);
    await listNotifications();
    await markNotificationRead(9);

    expect(fetchMock.mock.calls.map(([url]) => url)).toEqual([
      "http://api.test/api/auth/login",
      "http://api.test/api/auth/logout",
      "http://api.test/api/quotes",
      "http://api.test/api/technicians",
      "http://api.test/api/jobs",
      "http://api.test/api/jobs/7/schedule",
      "http://api.test/api/jobs/7/complete",
      "http://api.test/api/notifications",
      "http://api.test/api/notifications/9/read"
    ]);
    expect(JSON.parse(fetchMock.mock.calls[0][1].body as string)).toEqual({
      email: "manager1@brix.test",
      password: "password123"
    });
    expect(fetchMock.mock.calls[5][1]).toMatchObject({ method: "PATCH" });
    expect(JSON.parse(fetchMock.mock.calls[5][1].body as string)).toEqual({
      technicianId: 2,
      startsAt: "2026-05-12T13:00:00Z"
    });
    expect(JSON.parse(fetchMock.mock.calls[6][1].body as string)).toEqual({});
    expect(JSON.parse(fetchMock.mock.calls[8][1].body as string)).toEqual({});
  });

  test("openEventSocket subscribes to WebSocket messages and closes the socket", async () => {
    process.env.NEXT_PUBLIC_WS_URL = "ws://socket.test/ws";
    const sockets: FakeWebSocket[] = [];

    class FakeWebSocket {
      onmessage: ((message: MessageEvent<string>) => void) | null = null;
      close = vi.fn();
      url: string;

      constructor(url: string) {
        this.url = url;
        sockets.push(this);
      }
    }

    vi.stubGlobal("WebSocket", FakeWebSocket);
    const onEvent = vi.fn();
    const { openEventSocket } = await import("../lib/api");

    const close = openEventSocket(onEvent);
    sockets[0].onmessage?.({ data: JSON.stringify({ type: "jobs.changed", jobId: 7 }) } as MessageEvent<string>);
    close();

    expect(sockets[0].url).toBe("ws://socket.test/ws");
    expect(onEvent).toHaveBeenCalledWith({ type: "jobs.changed", jobId: 7 });
    expect(sockets[0].close).toHaveBeenCalled();
  });
});

function jsonResponse(body: unknown, status = 200): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => body
  } as Response;
}
