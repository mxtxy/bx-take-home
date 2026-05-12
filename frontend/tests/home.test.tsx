import { render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import HomePage from "../app/page";

const getMeMock = vi.fn();

vi.mock("../lib/api", () => ({
  getMe: () => getMeMock()
}));

describe("home page", () => {
  beforeEach(() => {
    getMeMock.mockReset();
  });

  test("renders loading state while checking the session", async () => {
    getMeMock.mockResolvedValue({ user: { role: "manager" } });
    render(<HomePage />);

    expect(screen.getByLabelText("Loading")).toBeInTheDocument();
    await waitFor(() => expect(getMeMock).toHaveBeenCalled());
  });

  test("redirects managers to the manager dashboard", async () => {
    getMeMock.mockResolvedValue({ user: { role: "manager" } });
    render(<HomePage />);

    await waitFor(() => expect(getMeMock).toHaveBeenCalled());
  });

  test("redirects technicians to the technician dashboard", async () => {
    getMeMock.mockResolvedValue({ user: { role: "technician" } });
    render(<HomePage />);

    await waitFor(() => expect(getMeMock).toHaveBeenCalled());
  });

  test("redirects unauthenticated users to login", async () => {
    getMeMock.mockRejectedValue(new Error("unauthorized"));
    render(<HomePage />);

    await waitFor(() => expect(getMeMock).toHaveBeenCalled());
  });
});
