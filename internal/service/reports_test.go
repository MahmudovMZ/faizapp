package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/MahmudovMZ/faizapp/internal/models"
	"github.com/MahmudovMZ/faizapp/internal/repository/postgres"
)

func TestDayBounds(t *testing.T) {
	utc := time.UTC

	tests := []struct {
		name     string
		moment   time.Time
		wantFrom time.Time
		wantTo   time.Time
	}{
		{
			name:     "afternoon in Dushanbe",
			moment:   time.Date(2026, time.November, 24, 16, 0, 0, 0, dushanbeTimeZone),
			wantFrom: time.Date(2026, time.November, 24, 0, 0, 0, 0, dushanbeTimeZone),
			wantTo:   time.Date(2026, time.November, 25, 0, 0, 0, 0, dushanbeTimeZone),
		},
		{
			name:     "late evening UTC is already the next day in Dushanbe",
			moment:   time.Date(2026, time.November, 24, 23, 30, 0, 0, utc),
			wantFrom: time.Date(2026, time.November, 25, 0, 0, 0, 0, dushanbeTimeZone),
			wantTo:   time.Date(2026, time.November, 26, 0, 0, 0, 0, dushanbeTimeZone),
		},
		{
			name:     "one second before midnight in Dushanbe is still the same day",
			moment:   time.Date(2026, time.November, 24, 18, 59, 59, 0, utc),
			wantFrom: time.Date(2026, time.November, 24, 0, 0, 0, 0, dushanbeTimeZone),
			wantTo:   time.Date(2026, time.November, 25, 0, 0, 0, 0, dushanbeTimeZone),
		},
		{
			name:     "exactly midnight in Dushanbe starts the day",
			moment:   time.Date(2026, time.November, 24, 0, 0, 0, 0, dushanbeTimeZone),
			wantFrom: time.Date(2026, time.November, 24, 0, 0, 0, 0, dushanbeTimeZone),
			wantTo:   time.Date(2026, time.November, 25, 0, 0, 0, 0, dushanbeTimeZone),
		},
		{
			name:     "last day of the month",
			moment:   time.Date(2026, time.November, 30, 12, 0, 0, 0, dushanbeTimeZone),
			wantFrom: time.Date(2026, time.November, 30, 0, 0, 0, 0, dushanbeTimeZone),
			wantTo:   time.Date(2026, time.December, 1, 0, 0, 0, 0, dushanbeTimeZone),
		},
		{
			name:     "last day of the year",
			moment:   time.Date(2026, time.December, 31, 20, 0, 0, 0, utc),
			wantFrom: time.Date(2027, time.January, 1, 0, 0, 0, 0, dushanbeTimeZone),
			wantTo:   time.Date(2027, time.January, 2, 0, 0, 0, 0, dushanbeTimeZone),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, to := dayBounds(tt.moment)

			if !from.Equal(tt.wantFrom) {
				t.Fatalf("from = %v, want %v", from, tt.wantFrom)
			}
			if !to.Equal(tt.wantTo) {
				t.Fatalf("to = %v, want %v", to, tt.wantTo)
			}
		})
	}
}

func TestDayBoundsCoverExactly24Hours(t *testing.T) {
	from, to := dayBounds(time.Date(2026, time.November, 24, 9, 0, 0, 0, dushanbeTimeZone))

	if got := to.Sub(from); got != 24*time.Hour {
		t.Fatalf("day length = %v, want 24h", got)
	}
}

func TestDushanbeTimeZoneIsUTCPlusFive(t *testing.T) {
	_, offset := time.Date(2026, time.July, 1, 0, 0, 0, 0, dushanbeTimeZone).Zone()

	if offset != 5*60*60 {
		t.Fatalf("offset = %d seconds, want %d (UTC+5)", offset, 5*60*60)
	}
}

// The real repository must fit the store the report service asks for.
var _ ReportStore = (*postgres.FaizAppRepo)(nil)

// reportStoreMock returns prepared rows and records the period it was asked for.
type reportStoreMock struct {
	rows []models.ReportRow
	err  error

	called bool
	from   time.Time
	to     time.Time
}

func (m *reportStoreMock) PhotoReportByPeriod(_ context.Context, from time.Time, to time.Time) ([]models.ReportRow, error) {
	m.called = true
	m.from = from
	m.to = to

	return m.rows, m.err
}

func TestDailyReportAsksForTheDushanbeDay(t *testing.T) {
	store := &reportStoreMock{}
	service := NewReportService(store)

	// 23:30 UTC on the 24th is already 04:30 on the 25th in Dushanbe.
	moment := time.Date(2026, time.November, 24, 23, 30, 0, 0, time.UTC)

	if _, err := service.DailyReport(context.Background(), moment); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantFrom := time.Date(2026, time.November, 25, 0, 0, 0, 0, dushanbeTimeZone)
	wantTo := time.Date(2026, time.November, 26, 0, 0, 0, 0, dushanbeTimeZone)

	if !store.called {
		t.Fatal("the store was not called")
	}
	if !store.from.Equal(wantFrom) || !store.to.Equal(wantTo) {
		t.Fatalf("period = [%v, %v), want [%v, %v)", store.from, store.to, wantFrom, wantTo)
	}
}

func TestDailyReportFillsMerchForEveryRow(t *testing.T) {
	first := time.Date(2026, time.November, 24, 9, 15, 0, 0, dushanbeTimeZone)

	store := &reportStoreMock{
		rows: []models.ReportRow{
			{SRCode: "SAM-1", FullName: "Иванов Иван", PhotosNum: 10, PhotoPerUnit: 4, FirstPhotoAt: &first},
			{SRCode: "SAM-2", FullName: "Петров Пётр", PhotosNum: 1, PhotoPerUnit: 4, FirstPhotoAt: &first},
			{SRCode: "SAE-1", FullName: "Сидоров Сидор", PhotosNum: 7, PhotoPerUnit: 2, FirstPhotoAt: &first},
			{SRCode: "SAM-3", FullName: "Без Фото", PhotosNum: 0, PhotoPerUnit: 4},
			{SRCode: "SAM-4", FullName: "", PhotosNum: 0, PhotoPerUnit: 4},
		},
	}
	service := NewReportService(store)

	rows, err := service.DailyReport(context.Background(), first)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []struct {
		code  string
		merch int
	}{
		{"SAM-1", 3}, // 10 / 4 = 2.5 -> 3
		{"SAM-2", 0}, // 1 / 4 = 0.25 -> 0
		{"SAE-1", 4}, // 7 / 2 = 3.5 -> 4
		{"SAM-3", 0}, // no photos
		{"SAM-4", 0}, // vacant code
	}

	if len(rows) != len(want) {
		t.Fatalf("got %d rows, want %d", len(rows), len(want))
	}

	for i, w := range want {
		if rows[i].SRCode != w.code {
			t.Fatalf("row %d code = %s, want %s: order must be kept", i, rows[i].SRCode, w.code)
		}
		if rows[i].Merch != w.merch {
			t.Fatalf("%s merch = %d, want %d", w.code, rows[i].Merch, w.merch)
		}
	}

	if rows[0].FullName != "Иванов Иван" || rows[0].PhotosNum != 10 || rows[0].FirstPhotoAt != &first {
		t.Fatalf("other fields of the row were changed: %+v", rows[0])
	}
}

func TestDailyReportStoreError(t *testing.T) {
	dbErr := errors.New("database is down")
	store := &reportStoreMock{err: dbErr}
	service := NewReportService(store)

	rows, err := service.DailyReport(context.Background(), time.Now())
	if !errors.Is(err, dbErr) {
		t.Fatalf("error = %v, want %v", err, dbErr)
	}
	if rows != nil {
		t.Fatalf("rows = %v, want nil on error", rows)
	}
}

func TestDailyReportInvalidCoefficient(t *testing.T) {
	store := &reportStoreMock{
		rows: []models.ReportRow{
			{SRCode: "SAM-1", PhotosNum: 4, PhotoPerUnit: 4},
			{SRCode: "SAE-7", PhotosNum: 4, PhotoPerUnit: 0},
		},
	}
	service := NewReportService(store)

	rows, err := service.DailyReport(context.Background(), time.Now())
	if !errors.Is(err, ErrInvalidCoefficient) {
		t.Fatalf("error = %v, want ErrInvalidCoefficient", err)
	}
	if !strings.Contains(err.Error(), "SAE-7") {
		t.Fatalf("error %q does not name the broken code SAE-7", err)
	}
	if rows != nil {
		t.Fatalf("rows = %v, want nil on error", rows)
	}
}

func TestDailyReportEmpty(t *testing.T) {
	service := NewReportService(&reportStoreMock{rows: nil})

	rows, err := service.DailyReport(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("rows = %v, want none", rows)
	}
}
