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
	if quotes[0].OrganizationID != 1 {
		t.Fatalf("organization id = %d", quotes[0].OrganizationID)
	}
}

func TestQuoteService_ListQuotes_FiltersByOrganization(t *testing.T) {
	database := testutil.PrepareDB(t)
	testutil.InsertOtherOrganizationFixture(t, database)
	service := NewService(database)
	status := domain.QuoteUnscheduled

	manager1Quotes, err := service.ListQuotes(context.Background(), testutil.Manager1(), &status)
	if err != nil {
		t.Fatalf("list manager1 quotes: %v", err)
	}
	if len(manager1Quotes) != 5 {
		t.Fatalf("manager1 quotes = %#v", manager1Quotes)
	}
	manager3Quotes, err := service.ListQuotes(context.Background(), testutil.Manager3(), &status)
	if err != nil {
		t.Fatalf("list manager3 quotes: %v", err)
	}
	if len(manager3Quotes) != 1 || manager3Quotes[0].CustomerName != "Other Org HVAC" {
		t.Fatalf("manager3 quotes = %#v", manager3Quotes)
	}
}

func TestQuoteService_CreateQuote_ManagerCreatesOrganizationScopedQuote(t *testing.T) {
	database := testutil.PrepareDB(t)
	service := NewService(database)

	quote, err := service.CreateQuote(context.Background(), testutil.Manager1(), domain.CreateQuoteInput{
		CustomerName: "Sydney Bakery",
		Description:  "Install replacement oven circuit",
	})
	if err != nil {
		t.Fatalf("create quote: %v", err)
	}
	if quote.ID == 0 || quote.OrganizationID != 1 || quote.CustomerName != "Sydney Bakery" || quote.Status != domain.QuoteUnscheduled {
		t.Fatalf("quote = %#v", quote)
	}
	if got := testutil.CountRows(t, database, `SELECT COUNT(*) FROM quotes WHERE organization_id = 1 AND customer_name = 'Sydney Bakery'`); got != 1 {
		t.Fatalf("created quote count = %d", got)
	}
}

func TestQuoteService_CreateQuote_TechnicianForbidden(t *testing.T) {
	service := NewService(testutil.PrepareDB(t))

	_, err := service.CreateQuote(context.Background(), testutil.Technician1(), domain.CreateQuoteInput{
		CustomerName: "Sydney Bakery",
		Description:  "Install replacement oven circuit",
	})
	if code := domain.CodeOf(err); code != domain.ErrorForbidden {
		t.Fatalf("code = %v, err = %v", code, err)
	}
}

func TestQuoteService_CreateQuote_InvalidInput(t *testing.T) {
	service := NewService(testutil.PrepareDB(t))

	_, err := service.CreateQuote(context.Background(), testutil.Manager1(), domain.CreateQuoteInput{
		CustomerName: " ",
		Description:  "Install replacement oven circuit",
	})
	if code := domain.CodeOf(err); code != domain.ErrorInvalidInput {
		t.Fatalf("blank customer code = %v, err = %v", code, err)
	}

	_, err = service.CreateQuote(context.Background(), testutil.Manager1(), domain.CreateQuoteInput{
		CustomerName: "Sydney Bakery",
		Description:  "",
	})
	if code := domain.CodeOf(err); code != domain.ErrorInvalidInput {
		t.Fatalf("blank description code = %v, err = %v", code, err)
	}
}

func TestQuoteService_CreateQuote_ReturnsInsertError(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer database.Close()
	mock.ExpectExec("INSERT INTO quotes").WillReturnError(errors.New("insert failed"))
	service := NewService(database)

	_, err = service.CreateQuote(context.Background(), testutil.Manager1(), domain.CreateQuoteInput{
		CustomerName: "Sydney Bakery",
		Description:  "Install replacement oven circuit",
	})
	if err == nil {
		t.Fatal("expected insert error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestQuoteService_CreateQuote_ReturnsLastInsertIDError(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer database.Close()
	mock.ExpectExec("INSERT INTO quotes").WillReturnResult(sqlmock.NewErrorResult(errors.New("last id failed")))
	service := NewService(database)

	_, err = service.CreateQuote(context.Background(), testutil.Manager1(), domain.CreateQuoteInput{
		CustomerName: "Sydney Bakery",
		Description:  "Install replacement oven circuit",
	})
	if err == nil {
		t.Fatal("expected last insert id error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestQuoteService_CreateQuote_ReturnsSelectError(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer database.Close()
	mock.ExpectExec("INSERT INTO quotes").WillReturnResult(sqlmock.NewResult(12, 1))
	mock.ExpectQuery("SELECT id, organization_id").WillReturnError(errors.New("select failed"))
	service := NewService(database)

	_, err = service.CreateQuote(context.Background(), testutil.Manager1(), domain.CreateQuoteInput{
		CustomerName: "Sydney Bakery",
		Description:  "Install replacement oven circuit",
	})
	if err == nil {
		t.Fatal("expected select error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
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
	mock.ExpectQuery("SELECT id, organization_id").WillReturnError(errors.New("query failed"))
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
	rows := sqlmock.NewRows([]string{"id", "organization_id", "customer_name", "description", "status", "created_at", "updated_at"}).
		AddRow("bad-id", int64(1), "Acme", "desc", "unscheduled", time.Now(), time.Now())
	mock.ExpectQuery("SELECT id, organization_id").WillReturnRows(rows)
	service := NewService(database)

	_, err = service.ListQuotes(context.Background(), testutil.Manager1(), nil)
	if err == nil {
		t.Fatal("expected scan error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
