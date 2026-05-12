import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, test, vi } from "vitest";
import LoginPage from "../app/login/page";

const loginMock = vi.fn();

vi.mock("../lib/api", () => ({
  login: (...args: unknown[]) => loginMock(...args)
}));

describe("login page", () => {
  beforeEach(() => {
    loginMock.mockReset();
  });

  test("Login page renders email input", () => {
    render(<LoginPage />);
    expect(screen.getByLabelText(/email/i)).toBeInTheDocument();
  });

  test("Login page renders password input", () => {
    render(<LoginPage />);
    expect(screen.getByLabelText(/password/i)).toBeInTheDocument();
  });

  test("Login page renders login button", () => {
    render(<LoginPage />);
    expect(screen.getByRole("button", { name: /^login$/i })).toBeInTheDocument();
  });

  test("Seeded manager login button exists", () => {
    render(<LoginPage />);
    expect(screen.getByRole("button", { name: /manager demo/i })).toBeInTheDocument();
  });

  test("Seeded technician login button exists", () => {
    render(<LoginPage />);
    expect(screen.getByRole("button", { name: /technician demo/i })).toBeInTheDocument();
  });

  test("Error alert renders when login fails", async () => {
    loginMock.mockRejectedValue(new Error("Invalid email or password."));
    render(<LoginPage />);
    await userEvent.type(screen.getByLabelText(/email/i), "manager1@brix.test");
    await userEvent.type(screen.getByLabelText(/password/i), "wrong");
    await userEvent.click(screen.getByRole("button", { name: /^login$/i }));

    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("Invalid email or password."));
  });

  test("Unknown login failure renders fallback message", async () => {
    loginMock.mockRejectedValue("network down");
    render(<LoginPage />);
    await userEvent.click(screen.getByRole("button", { name: /^login$/i }));

    await waitFor(() => expect(screen.getByRole("alert")).toHaveTextContent("Login failed."));
  });

  test("Manager demo submits seeded manager credentials", async () => {
    loginMock.mockResolvedValue({ user: { role: "manager" } });
    render(<LoginPage />);
    await userEvent.click(screen.getByRole("button", { name: /manager demo/i }));

    await waitFor(() => expect(loginMock).toHaveBeenCalledWith("manager1@brix.test", "password123"));
  });

  test("Technician demo submits seeded technician credentials", async () => {
    loginMock.mockResolvedValue({ user: { role: "technician" } });
    render(<LoginPage />);
    await userEvent.click(screen.getByRole("button", { name: /technician demo/i }));

    await waitFor(() => expect(loginMock).toHaveBeenCalledWith("technician1@brix.test", "password123"));
  });
});
