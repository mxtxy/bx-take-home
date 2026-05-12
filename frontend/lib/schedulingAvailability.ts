import type { Job, Technician, TechnicianAvailabilityRule } from "./api";

export const SCHEDULE_SLOT_TIMES = ["08:00", "10:00", "12:00", "14:00", "16:00", "18:00"];

export function formatAvailabilityRule(rule: TechnicianAvailabilityRule): string {
  return `${weekdayNames[rule.weekday] ?? "Day"} ${rule.startsAt}-${rule.endsAt} Sydney`;
}

export function formatScheduleWindow(job: Pick<Job, "startsAt" | "endsAt">): string {
  return `${formatScheduleDateTime(job.startsAt)} - ${formatScheduleDateTime(job.endsAt)} Sydney`;
}

export function bookedJobsForTechnician(jobs: Job[], technicianID: number): Job[] {
  return jobs
    .filter((job) => job.technicianId === technicianID)
    .sort((left, right) => new Date(left.startsAt).getTime() - new Date(right.startsAt).getTime());
}

export function canScheduleAt(technician: Technician, jobs: Job[], value: string): boolean {
  const date = value.slice(0, 10);
  const time = value.slice(11, 16);
  if (!date || !time) return false;
  return isSlotAvailable(technician, jobs, date, time);
}

export function scheduleInputToISOString(value: string): string {
  return parseScheduleDateTime(value.slice(0, 10), value.slice(11, 16))?.toISOString() ?? "";
}

export function dateHasAvailableSlot(technician: Technician, jobs: Job[], date: string): boolean {
  return SCHEDULE_SLOT_TIMES.some((time) => isSlotAvailable(technician, jobs, date, time));
}

export function isSlotAvailable(technician: Technician, jobs: Job[], date: string, time: string): boolean {
  const start = parseScheduleDateTime(date, time);
  if (!start) return false;
  const end = new Date(start.getTime() + twoHourWindowMS);
  return isInsideAvailability(technician.availability ?? [], start, end) && !overlapsBookedJob(jobs, technician.id, start, end);
}

function isInsideAvailability(rules: TechnicianAvailabilityRule[], startsAt: Date, endsAt: Date): boolean {
  if (rules.length === 0) return false;
  const startParts = scheduleZonedParts(startsAt);
  const endParts = scheduleZonedParts(endsAt);
  if (startParts.date !== endParts.date) return false;
  const weekday = scheduleWeekday(startParts);
  const startTime = scheduleTime(startParts);
  const endTime = scheduleTime(endParts);
  return rules.some((rule) => rule.weekday === weekday && rule.startsAt <= startTime && rule.endsAt >= endTime);
}

function overlapsBookedJob(jobs: Job[], technicianID: number, startsAt: Date, endsAt: Date): boolean {
  const startMS = startsAt.getTime();
  const endMS = endsAt.getTime();
  return jobs.some((job) => {
    if (job.technicianId !== technicianID) return false;
    return new Date(job.startsAt).getTime() < endMS && new Date(job.endsAt).getTime() > startMS;
  });
}

function parseScheduleDateTime(date: string, time: string): Date | null {
  const [year, month, day] = date.split("-").map(Number);
  const [hour, minute] = time.split(":").map(Number);
  if ([year, month, day, hour, minute].some((part) => !Number.isFinite(part))) return null;
  return zonedDateTimeToUTC({ day, hour, minute, month, year });
}

function zonedDateTimeToUTC(parts: ScheduleDateTimeParts): Date {
  const utcGuess = Date.UTC(parts.year, parts.month - 1, parts.day, parts.hour, parts.minute);
  const firstPass = new Date(utcGuess - scheduleOffsetMS(new Date(utcGuess)));
  return new Date(utcGuess - scheduleOffsetMS(firstPass));
}

function scheduleOffsetMS(value: Date): number {
  const parts = scheduleZonedParts(value);
  return Date.UTC(parts.year, parts.month - 1, parts.day, parts.hour, parts.minute) - value.getTime();
}

function formatScheduleDateTime(value: string): string {
  const parts = scheduleZonedParts(new Date(value));
  return `${parts.date} ${scheduleTime(parts)}`;
}

function scheduleZonedParts(value: Date): ScheduleDateTimeParts & { date: string } {
  const parts = Object.fromEntries(scheduleFormatter.formatToParts(value).map((part) => [part.type, part.value]));
  const year = Number(parts.year);
  const month = Number(parts.month);
  const day = Number(parts.day);
  const hour = Number(parts.hour);
  const minute = Number(parts.minute);
  return {
    date: `${parts.year}-${parts.month}-${parts.day}`,
    day,
    hour,
    minute,
    month,
    year
  };
}

function scheduleWeekday(parts: ScheduleDateTimeParts): number {
  return (new Date(Date.UTC(parts.year, parts.month - 1, parts.day)).getUTCDay() + 6) % 7;
}

function scheduleTime(parts: Pick<ScheduleDateTimeParts, "hour" | "minute">): string {
  return `${String(parts.hour).padStart(2, "0")}:${String(parts.minute).padStart(2, "0")}`;
}

const weekdayNames = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];
const scheduleFormatter = new Intl.DateTimeFormat("en-CA", {
  day: "2-digit",
  hour: "2-digit",
  hour12: false,
  hourCycle: "h23",
  minute: "2-digit",
  month: "2-digit",
  timeZone: "Australia/Sydney",
  year: "numeric"
});
const twoHourWindowMS = 2 * 60 * 60 * 1000;

type ScheduleDateTimeParts = {
  day: number;
  hour: number;
  minute: number;
  month: number;
  year: number;
};
