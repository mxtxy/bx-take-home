import { describe, expect, test } from "vitest";
import type { Job, Technician } from "../lib/api";
import {
  bookedJobsForTechnician,
  canScheduleAt,
  dateHasAvailableSlot,
  formatScheduleWindow,
  formatAvailabilityRule,
  isSlotAvailable,
  scheduleInputToISOString
} from "../lib/schedulingAvailability";

describe("scheduling availability helpers", () => {
  test("formats known and unknown availability weekdays", () => {
    expect(formatAvailabilityRule({ weekday: 1, startsAt: "08:00", endsAt: "18:00" })).toBe("Tue 08:00-18:00 Sydney");
    expect(formatAvailabilityRule({ weekday: 99, startsAt: "08:00", endsAt: "18:00" })).toBe("Day 08:00-18:00 Sydney");
  });

  test("filters and sorts booked jobs for the selected technician", () => {
    expect(bookedJobsForTechnician([laterJob, otherTechnicianJob, earlierJob], 1).map((job) => job.quoteCustomerName)).toEqual([
      "Northside Dental",
      "Acme Plumbing"
    ]);
  });

  test("checks schedule availability against rules and booked jobs", () => {
    const freeStart = "2026-05-12T12:00";
    const bookedStart = "2026-05-12T10:00";

    expect(canScheduleAt(technician, [earlierJob, otherTechnicianJob], freeStart)).toBe(true);
    expect(canScheduleAt(technician, [laterJob], bookedStart)).toBe(false);
    expect(canScheduleAt(technician, [], "")).toBe(false);
    expect(canScheduleAt(technician, [], "2026-05-12")).toBe(false);

    expect(isSlotAvailable(technician, [], "2026-05-12", "08:00")).toBe(true);
    expect(isSlotAvailable(technician, [], "2026-05-12", "12:00")).toBe(true);
    expect(isSlotAvailable(technician, [], "2026-05-12", "16:00")).toBe(true);
    expect(isSlotAvailable(technician, [], "2026-05-12", "18:00")).toBe(false);
    expect(isSlotAvailable({ ...technician, availability: [] }, [], "2026-05-12", "12:00")).toBe(false);
    expect(isSlotAvailable({ ...technician, availability: undefined }, [], "2026-05-12", "12:00")).toBe(false);
    expect(isSlotAvailable(technician, [], "2026-05-12", "23:00")).toBe(false);
    expect(isSlotAvailable(technician, [], "bad", "bad")).toBe(false);
  });

  test("reports dates with no available slot", () => {
    expect(dateHasAvailableSlot(technician, [], "2026-05-12")).toBe(true);
    expect(dateHasAvailableSlot(technician, [], "2026-05-16")).toBe(false);
  });

  test("serializes schedule input without applying browser timezone offsets", () => {
    expect(scheduleInputToISOString("2026-05-12T08:00")).toBe("2026-05-11T22:00:00.000Z");
    expect(scheduleInputToISOString("2026-01-12T08:00")).toBe("2026-01-11T21:00:00.000Z");
    expect(scheduleInputToISOString("")).toBe("");
  });

  test("formats schedule windows without applying browser timezone offsets", () => {
    expect(formatScheduleWindow(earlierJob)).toBe("2026-05-12 08:00 - 2026-05-12 10:00 Sydney");
  });
});

const weekdayAvailability = [
  { weekday: 0, startsAt: "08:00", endsAt: "18:00" },
  { weekday: 1, startsAt: "08:00", endsAt: "18:00" },
  { weekday: 2, startsAt: "08:00", endsAt: "18:00" },
  { weekday: 3, startsAt: "08:00", endsAt: "18:00" },
  { weekday: 4, startsAt: "08:00", endsAt: "18:00" }
];

const technician: Technician = {
  id: 1,
  userId: 3,
  displayName: "Tom Technician",
  email: "technician1@brix.test",
  availability: weekdayAvailability
};

const earlierJob: Job = {
  id: 2,
  quoteId: 2,
  quoteCustomerName: "Northside Dental",
  technicianId: 1,
  managerId: 1,
  startsAt: "2026-05-11T22:00:00Z",
  endsAt: "2026-05-12T00:00:00Z",
  status: "scheduled",
  completedAt: null
};

const laterJob: Job = {
  id: 1,
  quoteId: 1,
  quoteCustomerName: "Acme Plumbing",
  technicianId: 1,
  managerId: 1,
  startsAt: "2026-05-12T00:00:00Z",
  endsAt: "2026-05-12T02:00:00Z",
  status: "scheduled",
  completedAt: null
};

const otherTechnicianJob: Job = {
  id: 3,
  quoteId: 3,
  quoteCustomerName: "Green Grocer",
  technicianId: 2,
  managerId: 1,
  startsAt: "2026-05-12T02:00:00Z",
  endsAt: "2026-05-12T04:00:00Z",
  status: "scheduled",
  completedAt: null
};
