import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState } from "react";
import { describe, expect, test } from "vitest";
import { AppProviders } from "../components/AppProviders";
import { SchedulePicker } from "../components/SchedulePicker";

describe("schedule picker", () => {
  test("renders a calendar interface with the existing start-time input", () => {
    renderSchedulePicker();

    expect(screen.getByLabelText("Start time")).toBeInTheDocument();
    expect(screen.getByRole("group", { name: "Start time calendar" })).toBeInTheDocument();
    expect(screen.getByRole("group", { name: "Start time time slots" })).toBeInTheDocument();
  });

  test("Selecting a calendar day and time slot writes a local datetime value", async () => {
    renderSchedulePicker();

    await userEvent.click(screen.getByRole("button", { name: "Select 2026-05-12" }));
    await userEvent.click(screen.getByRole("button", { name: "10:00" }));

    expect(screen.getByLabelText("Start time")).toHaveValue("2026-05-12T10:00");
  });

  test("Direct datetime entry remains supported for exact scheduling", async () => {
    renderSchedulePicker("Reschedule start 1");

    await userEvent.type(screen.getByLabelText("Reschedule start 1"), "2026-05-12T13:30");

    expect(screen.getByLabelText("Reschedule start 1")).toHaveValue("2026-05-12T13:30");
  });
});

function renderSchedulePicker(label = "Start time") {
  render(
    <AppProviders>
      <SchedulePickerHarness label={label} />
    </AppProviders>
  );
}

function SchedulePickerHarness({ label }: { label: string }) {
  const [value, setValue] = useState("");

  return <SchedulePicker label={label} value={value} onChange={setValue} initialDate="2026-05-12" />;
}
