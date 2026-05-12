"use client";

import Box from "@mui/joy/Box";
import Button from "@mui/joy/Button";
import FormControl from "@mui/joy/FormControl";
import FormHelperText from "@mui/joy/FormHelperText";
import FormLabel from "@mui/joy/FormLabel";
import Input from "@mui/joy/Input";
import Sheet from "@mui/joy/Sheet";
import Stack from "@mui/joy/Stack";
import Typography from "@mui/joy/Typography";
import { SCHEDULE_SLOT_TIMES } from "../lib/schedulingAvailability";

type SchedulePickerProps = {
  calendarLabel?: string;
  initialDate?: string;
  isDateDisabled?: (date: string) => boolean;
  isTimeDisabled?: (date: string, time: string) => boolean;
  label: string;
  onChange: (value: string) => void;
  timeSlotsLabel?: string;
  value: string;
};

export function SchedulePicker({
  calendarLabel,
  initialDate = localDate(new Date()),
  isDateDisabled,
  isTimeDisabled,
  label,
  onChange,
  timeSlotsLabel,
  value
}: SchedulePickerProps) {
  const selectedDate = datePart(value) || initialDate;
  const selectedTime = timePart(value) || "10:00";
  const days = weekDays(selectedDate);

  function setDate(nextDate: string) {
    onChange(`${nextDate}T${selectedTime}`);
  }

  function setTime(nextTime: string) {
    onChange(`${selectedDate}T${nextTime}`);
  }

  return (
    <FormControl>
      <FormLabel>{label}</FormLabel>
      <Input
        type="datetime-local"
        value={value}
        onChange={(event) => onChange(event.target.value)}
        slotProps={{ input: { "aria-label": label } }}
      />
      <Sheet
        aria-label={calendarLabel || `${label} calendar`}
        role="group"
        variant="outlined"
        sx={{
          mt: 1,
          p: 1.25
        }}
      >
        <Stack spacing={1}>
          <Typography level="body-sm" fontWeight="lg">
            Calendar
          </Typography>
          <Box
            sx={{
              display: "grid",
              gap: 0.75,
              gridTemplateColumns: { xs: "repeat(2, minmax(0, 1fr))", sm: "repeat(4, minmax(0, 1fr))", md: "repeat(7, minmax(0, 1fr))" }
            }}
          >
            {days.map((day) => (
              <Button
                key={day.value}
                aria-label={`Select ${day.value}`}
                aria-pressed={selectedDate === day.value}
                color={selectedDate === day.value ? "primary" : "neutral"}
                disabled={isDateDisabled?.(day.value) ?? false}
                onClick={() => setDate(day.value)}
                size="sm"
                variant={selectedDate === day.value ? "solid" : "soft"}
              >
                <Stack spacing={0} sx={{ alignItems: "center", lineHeight: 1.1 }}>
                  <Typography level="body-xs" sx={{ color: "inherit" }}>
                    {day.weekday}
                  </Typography>
                  <Typography level="title-sm" sx={{ color: "inherit" }}>
                    {day.day}
                  </Typography>
                </Stack>
              </Button>
            ))}
          </Box>
          <Stack
            aria-label={timeSlotsLabel || `${label} time slots`}
            direction="row"
            role="group"
            spacing={0.75}
            sx={{ flexWrap: "wrap", rowGap: 0.75 }}
          >
            {SCHEDULE_SLOT_TIMES.map((time) => (
              <Button
                key={time}
                aria-pressed={selectedTime === time}
                color={selectedTime === time ? "primary" : "neutral"}
                disabled={isTimeDisabled?.(selectedDate, time) ?? false}
                onClick={() => setTime(time)}
                size="sm"
                variant={selectedTime === time ? "solid" : "outlined"}
              >
                {time}
              </Button>
            ))}
          </Stack>
        </Stack>
      </Sheet>
      <FormHelperText>Windows are fixed at two hours. Use the field for exact minute-aligned starts.</FormHelperText>
    </FormControl>
  );
}

function datePart(value: string): string {
  return value.slice(0, 10);
}

function timePart(value: string): string {
  return value.slice(11, 16);
}

function weekDays(anchorDate: string) {
  const anchor = parseLocalDate(anchorDate);
  return Array.from({ length: 7 }, (_, index) => {
    const date = new Date(anchor);
    date.setDate(anchor.getDate() + index);
    return {
      day: new Intl.DateTimeFormat(undefined, { day: "numeric" }).format(date),
      value: localDate(date),
      weekday: new Intl.DateTimeFormat(undefined, { weekday: "short" }).format(date)
    };
  });
}

function parseLocalDate(value: string): Date {
  const [year, month, day] = value.split("-").map(Number);
  return new Date(year, month - 1, day);
}

function localDate(date: Date): string {
  const year = date.getFullYear();
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}
