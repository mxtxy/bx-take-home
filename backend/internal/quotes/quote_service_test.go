package quotes

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mxtxy/bx-take-home/backend/internal/domain"
	"github.com/mxtxy/bx-take-home/backend/internal/testutil"
)

func TestQuoteService_ListQuotes_ManagerCanListUnscheduled(t *testing.T) {
	service := NewService(testutil.PrepareDB(t))
	status := domain.QuoteUnscheduled

	quotes, err := service.ListQuotes(context.Background(), testutil.Manager1(), &status)
	if err != nil {
		t.Fatalf("list quotes: %v", err)
	}
	if len(quotes) != 5 {
		t.Fatalf("len = %d", len(quotes))
	}
	for _, quote := range quotes {
		if quote.Status != domain.QuoteUnscheduled {
			t.Fatalf("quote status = %s", quote.Status)
		}
	}
	if quotes[0].ID != 1 || quotes[4].ID != 5 {
		t.Fatalf("quotes not sorted by createdAt/id: %#v", quotes)
	}
}

func TestQuoteService_ListQuotes_ExcludesScheduledQuotes(t *testing.T) {
	database := testutil.PrepareDB(t)
	testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z")
	service := NewService(database)
	status := domain.QuoteUnscheduled

	quotes, err := service.ListQuotes(context.Background(), testutil.Manager1(), &status)
	if err != nil {
		t.Fatalf("list quotes: %v", err)
	}
	if len(quotes) != 4 || quotes[0].ID != 2 {
		t.Fatalf("quotes = %#v", quotes)
	}
}

func TestQuoteService_ListQuotes_CanListScheduled(t *testing.T) {
	database := testutil.PrepareDB(t)
	testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z")
	service := NewService(database)
	status := domain.QuoteScheduled

	quotes, err := service.ListQuotes(context.Background(), testutil.Manager1(), &status)
	if err != nil {
		t.Fatalf("list quotes: %v", err)
	}
	if len(quotes) != 1 || quotes[0].ID != 1 {
		t.Fatalf("quotes = %#v", quotes)
	}
}

func TestQuoteService_ListQuotes_CanListAllWhenStatusNil(t *testing.T) {
	database := testutil.PrepareDB(t)
	testutil.InsertScheduledJob(t, database, 1, 1, 1, "2026-05-12T10:00:00Z")
	service := NewService(database)

	quotes, err := service.ListQuotes(context.Background(), testutil.Manager1(), nil)
	if err != nil {
		t.Fatalf("list quotes: %v", err)
	}
	if len(quotes) != 5 {
		t.Fatalf("len = %d", len(quotes))
	}
}

func TestQuoteService_ListQuotes_TechnicianForbidden(t *testing.T) {
	service := NewService(testutil.PrepareDB(t))
	status := domain.QuoteUnscheduled

	_, err := service.ListQuotes(context.Background(), testutil.Technician1(), &status)
	if code := domain.CodeOf(err); code != domain.ErrorForbidden {
		t.Fatalf("code = %v, err = %v", code, err)
	}
}

func TestQuoteService_ListQuotes_InvalidStatus(t *testing.T) {
	service := NewService(testutil.PrepareDB(t))
	status := domain.QuoteStatus("archived")

	_, err := service.ListQuotes(context.Background(), testutil.Manager1(), &status)
	if code := domain.CodeOf(err); code != domain.ErrorInvalidInput {
		t.Fatalf("code = %v, err = %v", code, err)
	}
}

func TestQuoteService_ListQuotes_ReturnsQueryError(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer database.Close()
	mock.ExpectQuery("SELECT id, customer_name").WillReturnError(errors.New("query failed"))
	service := NewService(database)

	_, err = service.ListQuotes(context.Background(), testutil.Manager1(), nil)
	if err == nil {
		t.Fatal("expected query error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestQuoteService_ListQuotes_ReturnsScanError(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer database.Close()
	rows := sqlmock.NewRows([]string{"id", "customer_name", "description", "status", "created_at", "updated_at"}).
		AddRow("bad-id", "Acme", "desc", "unscheduled", time.Now(), time.Now())
	mock.ExpectQuery("SELECT id, customer_name").WillReturnRows(rows)
	service := NewService(database)

	_, err = service.ListQuotes(context.Background(), testutil.Manager1(), nil)
	if err == nil {
		t.Fatal("expected scan error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
