package domain

import (
	"errors"
	"testing"
	"time"
)

func mustTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		t.Fatalf("parse time %q: %v", value, err)
	}
	return parsed
}

func TestCalculateWindow_ReturnsExactlyTwoHours(t *testing.T) {
	startsAt := mustTime(t, "2026-05-12T10:00:00Z")

	window := CalculateWindow(startsAt)

	if !window.EndsAt.Equal(mustTime(t, "2026-05-12T12:00:00Z")) {
		t.Fatalf("endsAt = %s", window.EndsAt.Format(time.RFC3339Nano))
	}
	if window.EndsAt.Sub(window.StartsAt) != 120*time.Minute {
		t.Fatalf("duration = %s", window.EndsAt.Sub(window.StartsAt))
	}
}

func TestValidateStartsAt_RejectsSecondPrecision(t *testing.T) {
	err := ValidateStartsAt(mustTime(t, "2026-05-12T10:00:01Z"))
	if CodeOf(err) != ErrorInvalidInput {
		t.Fatalf("code = %v, err = %v", CodeOf(err), err)
	}
}

func TestValidateStartsAt_RejectsNanosecondPrecision(t *testing.T) {
	err := ValidateStartsAt(mustTime(t, "2026-05-12T10:00:00.000000001Z"))
	if CodeOf(err) != ErrorInvalidInput {
		t.Fatalf("code = %v, err = %v", CodeOf(err), err)
	}
}

func TestValidateStartsAt_AllowsMinutePrecision(t *testing.T) {
	if err := ValidateStartsAt(mustTime(t, "2026-05-12T10:30:00Z")); err != nil {
		t.Fatalf("expected valid startsAt, got %v", err)
	}
}

func TestOverlaps_BoundaryCases(t *testing.T) {
	existing := TimeWindow{
		StartsAt: mustTime(t, "2026-05-12T10:00:00Z"),
		EndsAt:   mustTime(t, "2026-05-12T12:00:00Z"),
	}
	tests := []struct {
		name      string
		requested TimeWindow
		want      bool
	}{
		{"before boundary", TimeWindow{mustTime(t, "2026-05-12T08:00:00Z"), mustTime(t, "2026-05-12T10:00:00Z")}, false},
		{"left overlap", TimeWindow{mustTime(t, "2026-05-12T09:00:00Z"), mustTime(t, "2026-05-12T11:00:00Z")}, true},
		{"exact overlap", TimeWindow{mustTime(t, "2026-05-12T10:00:00Z"), mustTime(t, "2026-05-12T12:00:00Z")}, true},
		{"right overlap", TimeWindow{mustTime(t, "2026-05-12T11:00:00Z"), mustTime(t, "2026-05-12T13:00:00Z")}, true},
		{"after boundary", TimeWindow{mustTime(t, "2026-05-12T12:00:00Z"), mustTime(t, "2026-05-12T14:00:00Z")}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Overlaps(existing, tt.requested); got != tt.want {
				t.Fatalf("Overlaps() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateAssignJobInput_RejectsInvalidIDs(t *testing.T) {
	tests := []AssignJobInput{
		{QuoteID: 0, TechnicianID: 1, StartsAt: mustTime(t, "2026-05-12T10:00:00Z")},
		{QuoteID: -1, TechnicianID: 1, StartsAt: mustTime(t, "2026-05-12T10:00:00Z")},
		{QuoteID: 1, TechnicianID: 0, StartsAt: mustTime(t, "2026-05-12T10:00:00Z")},
		{QuoteID: 1, TechnicianID: -1, StartsAt: mustTime(t, "2026-05-12T10:00:00Z")},
	}
	for _, input := range tests {
		if code := CodeOf(ValidateAssignJobInput(input)); code != ErrorInvalidInput {
			t.Fatalf("input %#v code = %v", input, code)
		}
	}
}

func TestValidateAssignJobInput_AllowsValidInput(t *testing.T) {
	err := ValidateAssignJobInput(AssignJobInput{
		QuoteID:      1,
		TechnicianID: 1,
		StartsAt:     mustTime(t, "2026-05-12T10:00:00Z"),
	})
	if err != nil {
		t.Fatalf("expected valid assign input, got %v", err)
	}
}

func TestValidateRescheduleJobInput_RejectsInvalidIDs(t *testing.T) {
	tests := []RescheduleJobInput{
		{JobID: 0, TechnicianID: 1, StartsAt: mustTime(t, "2026-05-12T10:00:00Z")},
		{JobID: -1, TechnicianID: 1, StartsAt: mustTime(t, "2026-05-12T10:00:00Z")},
		{JobID: 1, TechnicianID: 0, StartsAt: mustTime(t, "2026-05-12T10:00:00Z")},
		{JobID: 1, TechnicianID: -1, StartsAt: mustTime(t, "2026-05-12T10:00:00Z")},
	}
	for _, input := range tests {
		if code := CodeOf(ValidateRescheduleJobInput(input)); code != ErrorInvalidInput {
			t.Fatalf("input %#v code = %v", input, code)
		}
	}
}

func TestValidateRescheduleJobInput_AllowsValidInput(t *testing.T) {
	err := ValidateRescheduleJobInput(RescheduleJobInput{
		JobID:        1,
		TechnicianID: 1,
		StartsAt:     mustTime(t, "2026-05-12T10:00:00Z"),
	})
	if err != nil {
		t.Fatalf("expected valid reschedule input, got %v", err)
	}
}

func TestValidateCompleteJobInput_RejectsInvalidIDs(t *testing.T) {
	for _, input := range []CompleteJobInput{{JobID: 0}, {JobID: -1}} {
		if code := CodeOf(ValidateCompleteJobInput(input)); code != ErrorInvalidInput {
			t.Fatalf("input %#v code = %v", input, code)
		}
	}
}

func TestValidateCompleteJobInput_AllowsValidInput(t *testing.T) {
	if err := ValidateCompleteJobInput(CompleteJobInput{JobID: 1}); err != nil {
		t.Fatalf("expected valid complete input, got %v", err)
	}
}

func TestServiceError_Error(t *testing.T) {
	err := NewError(ErrorForbidden, "custom message")
	if got := err.Error(); got != "forbidden: custom message" {
		t.Fatalf("Error() = %q", got)
	}
}

func TestErrorMessage_AllCodes(t *testing.T) {
	tests := map[ErrorCode]string{
		ErrorInvalidInput:          "Invalid input.",
		ErrorUnauthorized:          "Authentication is required.",
		ErrorForbidden:             "You are not allowed to perform this action.",
		ErrorNotFound:              "Resource not found.",
		ErrorInvalidCredentials:    "Invalid email or password.",
		ErrorQuoteAlreadyScheduled: "Quote has already been scheduled.",
		ErrorScheduleConflict:      "Technician already has a job in that time window.",
		ErrorCompletedJobImmutable: "Completed jobs cannot be rescheduled.",
		ErrorJobAlreadyCompleted:   "Job has already been completed.",
		ErrorInternal:              "Internal server error.",
	}
	for code, want := range tests {
		t.Run(string(code), func(t *testing.T) {
			if got := ErrorMessage(code); got != want {
				t.Fatalf("ErrorMessage(%q) = %q, want %q", code, got, want)
			}
		})
	}
}

func TestCodeOf_NilAndUnknownErrors(t *testing.T) {
	if code := CodeOf(nil); code != "" {
		t.Fatalf("nil code = %q", code)
	}
	if code := CodeOf(errors.New("plain error")); code != ErrorInternal {
		t.Fatalf("plain error code = %q", code)
	}
}
