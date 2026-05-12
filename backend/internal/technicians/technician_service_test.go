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
		names[technician.DisplayName] = true
	}
	if !names["Tom Technician"] || !names["Priya Technician"] {
		t.Fatalf("names = %#v", names)
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
