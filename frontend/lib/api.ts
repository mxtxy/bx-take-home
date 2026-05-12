export type Role = "manager" | "technician";
export type QuoteStatus = "unscheduled" | "scheduled";
export type JobStatus = "scheduled" | "completed";

export type User = {
  id: number;
  organizationId?: number;
  email?: string;
  displayName: string;
  role: Role;
  managerId?: number | null;
  technicianId?: number | null;
};

export type Quote = {
  id: number;
  organizationId?: number;
  customerName: string;
  description: string;
  status: QuoteStatus;
  createdAt?: string;
  updatedAt?: string;
};

export type Technician = {
  id: number;
  userId: number;
  displayName: string;
  email: string;
  availability?: TechnicianAvailabilityRule[];
};

export type TechnicianAvailabilityRule = {
  weekday: number;
  startsAt: string;
  endsAt: string;
};

export type Job = {
  id: number;
  organizationId?: number;
  quoteId: number;
  quoteCustomerName?: string;
  quoteDescription?: string;
  technicianId: number;
  technicianName?: string;
  managerId: number;
  managerName?: string;
  startsAt: string;
  endsAt: string;
  status: JobStatus;
  completedAt: string | null;
};

export type Notification = {
  id: number;
  organizationId?: number;
  type: "job_assigned" | "job_updated" | "job_completed";
  message: string;
  jobId: number | null;
  readAt: string | null;
  createdAt: string;
};

export type ServerEvent =
  | { type: "notification.created"; notificationId: number; jobId: number | null }
  | { type: "jobs.changed"; jobId: number }
  | { type: "quotes.changed"; quoteId: number };

export class ApiError extends Error {
  code: string;
  status: number;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
const WS_URL = process.env.NEXT_PUBLIC_WS_URL || "ws://localhost:8080/ws";

export async function getMe(): Promise<{ user: User }> {
  return apiFetch("/api/me");
}

export async function login(email: string, password: string): Promise<{ user: User }> {
  return apiFetch("/api/auth/login", {
    method: "POST",
    body: JSON.stringify({ email, password })
  });
}

export async function logout(): Promise<{ ok: boolean }> {
  return apiFetch("/api/auth/logout", {
    method: "POST",
    body: JSON.stringify({})
  });
}

export async function listQuotes(status?: QuoteStatus): Promise<{ quotes: Quote[] }> {
  const query = status ? `?status=${encodeURIComponent(status)}` : "";
  return apiFetch(`/api/quotes${query}`);
}

export async function createQuote(input: { customerName: string; description: string }): Promise<{ quote: Quote }> {
  return apiFetch("/api/quotes", {
    method: "POST",
    body: JSON.stringify(input)
  });
}

export async function listTechnicians(): Promise<{ technicians: Technician[] }> {
  return apiFetch("/api/technicians");
}

export async function listJobs(): Promise<{ jobs: Job[] }> {
  return apiFetch("/api/jobs");
}

export async function assignJob(input: { quoteId: number; technicianId: number; startsAt: string }): Promise<{ job: Job }> {
  return apiFetch("/api/jobs", {
    method: "POST",
    body: JSON.stringify({
      quoteId: input.quoteId,
      technicianId: input.technicianId,
      startsAt: input.startsAt
    })
  });
}

export async function rescheduleJob(jobId: number, input: { technicianId: number; startsAt: string }): Promise<{ job: Job }> {
  return apiFetch(`/api/jobs/${jobId}/schedule`, {
    method: "PATCH",
    body: JSON.stringify(input)
  });
}

export async function completeJob(jobId: number): Promise<{ job: Job }> {
  return apiFetch(`/api/jobs/${jobId}/complete`, {
    method: "PATCH",
    body: JSON.stringify({})
  });
}

export async function listNotifications(): Promise<{ notifications: Notification[]; unreadCount: number }> {
  return apiFetch("/api/notifications");
}

export async function markNotificationRead(notificationId: number): Promise<{ notification: Notification }> {
  return apiFetch(`/api/notifications/${notificationId}/read`, {
    method: "PATCH",
    body: JSON.stringify({})
  });
}

export function openEventSocket(onEvent: (event: ServerEvent) => void): () => void {
  let closed = false;
  let socket: WebSocket | null = null;
  let retry: ReturnType<typeof setTimeout> | null = null;

  const connect = () => {
    socket = new WebSocket(WS_URL);
    socket.onmessage = (message) => {
      onEvent(JSON.parse(message.data) as ServerEvent);
    };
    socket.onclose = () => {
      if (!closed) {
        retry = setTimeout(connect, 1000);
      }
    };
  };

  connect();

  return () => {
    closed = true;
    if (retry !== null) {
      clearTimeout(retry);
    }
    socket?.close();
  };
}

async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers);
  if (init.body) {
    headers.set("Content-Type", "application/json");
  }
  const response = await fetch(`${API_URL}${path}`, {
    ...init,
    headers,
    credentials: "include"
  });
  const data = await response.json();
  if (!response.ok) {
    const error = data?.error;
    throw new ApiError(response.status, error?.code || "internal_error", error?.message || "Request failed.");
  }
  return data as T;
}
