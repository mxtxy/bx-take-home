package technicians

import (
	"context"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/mxtxy/bx-take-home/backend/internal/domain"
	"github.com/mxtxy/bx-take-home/backend/internal/testutil"
)

func TestTechnicianService_ListTechnicians_ManagerSuccess(t *testing.T) {
	service := NewService(testutil.PrepareDB(t))

	technicians, err := service.ListTechnicians(context.Background(), testutil.Manager1())
	if err != nil {
		t.Fatalf("list technicians: %v", err)
	}
	if len(technicians) != 2 {
		t.Fatalf("len = %d", len(technicians))
	}
	names := map[string]bool{}
	for _, technician := range technicians {
		if technician.ID == 0 || technician.UserID == 0 || technician.Email == "" || technician.DisplayName == "" {
			t.Fatalf("technician missing fields: %#v", technician)
		}
		if len(technician.Availability) == 0 {
			t.Fatalf("technician missing availability: %#v", technician)
		}
		names[technician.DisplayName] = true
	}
	if !names["Tom Technician"] || !names["Priya Technician"] {
		t.Fatalf("names = %#v", names)
	}
}

func TestTechnicianService_ListTechnicians_FiltersByOrganization(t *testing.T) {
	database := testutil.PrepareDB(t)
	testutil.InsertOtherOrganizationFixture(t, database)
	service := NewService(database)

	technicians, err := service.ListTechnicians(context.Background(), testutil.Manager1())
	if err != nil {
		t.Fatalf("list manager1 technicians: %v", err)
	}
	if len(technicians) != 2 {
		t.Fatalf("manager1 technicians = %#v", technicians)
	}
	otherTechnicians, err := service.ListTechnicians(context.Background(), testutil.Manager3())
	if err != nil {
		t.Fatalf("list manager3 technicians: %v", err)
	}
	if len(otherTechnicians) != 1 || otherTechnicians[0].DisplayName != "Other Org Tech" {
		t.Fatalf("manager3 technicians = %#v", otherTechnicians)
	}
}

func TestTechnicianService_ListTechnicians_SortedByDisplayNameThenID(t *testing.T) {
	service := NewService(testutil.PrepareDB(t))

	technicians, err := service.ListTechnicians(context.Background(), testutil.Manager1())
	if err != nil {
		t.Fatalf("list technicians: %v", err)
	}
	if technicians[0].DisplayName != "Priya Technician" || technicians[1].DisplayName != "Tom Technician" {
		t.Fatalf("order = %#v", technicians)
	}
}

func TestTechnicianService_ListTechnicians_TechnicianForbidden(t *testing.T) {
	service := NewService(testutil.PrepareDB(t))

	_, err := service.ListTechnicians(context.Background(), testutil.Technician1())
	if code := domain.CodeOf(err); code != domain.ErrorForbidden {
		t.Fatalf("code = %v, err = %v", code, err)
	}
}

func TestTechnicianService_ListTechnicians_ReturnsQueryError(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer database.Close()
	mock.ExpectQuery("SELECT t.id, u.id, u.display_name, u.email").WillReturnError(errors.New("query failed"))
	service := NewService(database)

	_, err = service.ListTechnicians(context.Background(), testutil.Manager1())
	if err == nil {
		t.Fatal("expected query error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestTechnicianService_ListTechnicians_ReturnsScanError(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer database.Close()
	rows := sqlmock.NewRows([]string{"id", "user_id", "display_name", "email"}).
		AddRow("bad-id", int64(3), "Tom Technician", "technician1@brix.test")
	mock.ExpectQuery("SELECT t.id, u.id, u.display_name, u.email").WillReturnRows(rows)
	service := NewService(database)

	_, err = service.ListTechnicians(context.Background(), testutil.Manager1())
	if err == nil {
		t.Fatal("expected scan error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestTechnicianService_ListTechnicians_ReturnsRowsError(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer database.Close()
	rows := sqlmock.NewRows([]string{"id", "user_id", "display_name", "email"}).
		AddRow(int64(1), int64(3), "Tom Technician", "technician1@brix.test").
		RowError(0, errors.New("rows failed"))
	mock.ExpectQuery("SELECT t.id, u.id, u.display_name, u.email").WillReturnRows(rows)
	service := NewService(database)

	_, err = service.ListTechnicians(context.Background(), testutil.Manager1())
	if err == nil {
		t.Fatal("expected rows error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestTechnicianService_ListTechnicians_ReturnsAvailabilityQueryError(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer database.Close()
	technicianRows := sqlmock.NewRows([]string{"id", "user_id", "display_name", "email"}).
		AddRow(int64(1), int64(3), "Tom Technician", "technician1@brix.test")
	mock.ExpectQuery("SELECT t.id, u.id, u.display_name, u.email").WillReturnRows(technicianRows)
	mock.ExpectQuery("SELECT ar.technician_id").WillReturnError(errors.New("availability query failed"))
	service := NewService(database)

	_, err = service.ListTechnicians(context.Background(), testutil.Manager1())
	if err == nil {
		t.Fatal("expected availability query error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}

func TestTechnicianService_ListTechnicians_ReturnsAvailabilityScanError(t *testing.T) {
	database, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer database.Close()
	technicianRows := sqlmock.NewRows([]string{"id", "user_id", "display_name", "email"}).
		AddRow(int64(1), int64(3), "Tom Technician", "technician1@brix.test")
	availabilityRows := sqlmock.NewRows([]string{"technician_id", "weekday", "starts_at", "ends_at"}).
		AddRow(int64(1), "bad-weekday", "08:00", "18:00")
	mock.ExpectQuery("SELECT t.id, u.id, u.display_name, u.email").WillReturnRows(technicianRows)
	mock.ExpectQuery("SELECT ar.technician_id").WillReturnRows(availabilityRows)
	service := NewService(database)

	_, err = service.ListTechnicians(context.Background(), testutil.Manager1())
	if err == nil {
		t.Fatal("expected availability scan error")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("expectations: %v", err)
	}
}
