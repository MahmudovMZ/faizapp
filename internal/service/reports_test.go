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
			moment:   time.Date(2026, time.November, 24, 16, 0, 0, 0, DushanbeTimeZone),
			wantFrom: time.Date(2026, time.November, 24, 0, 0, 0, 0, DushanbeTimeZone),
			wantTo:   time.Date(2026, time.November, 25, 0, 0, 0, 0, DushanbeTimeZone),
		},
		{
			name:     "late evening UTC is already the next day in Dushanbe",
			moment:   time.Date(2026, time.November, 24, 23, 30, 0, 0, utc),
			wantFrom: time.Date(2026, time.November, 25, 0, 0, 0, 0, DushanbeTimeZone),
			wantTo:   time.Date(2026, time.November, 26, 0, 0, 0, 0, DushanbeTimeZone),
		},
		{
			name:     "one second before midnight in Dushanbe is still the same day",
			moment:   time.Date(2026, time.November, 24, 18, 59, 59, 0, utc),
			wantFrom: time.Date(2026, time.November, 24, 0, 0, 0, 0, DushanbeTimeZone),
			wantTo:   time.Date(2026, time.November, 25, 0, 0, 0, 0, DushanbeTimeZone),
		},
		{
			name:     "exactly midnight in Dushanbe starts the day",
			moment:   time.Date(2026, time.November, 24, 0, 0, 0, 0, DushanbeTimeZone),
			wantFrom: time.Date(2026, time.November, 24, 0, 0, 0, 0, DushanbeTimeZone),
			wantTo:   time.Date(2026, time.November, 25, 0, 0, 0, 0, DushanbeTimeZone),
		},
		{
			name:     "last day of the month",
			moment:   time.Date(2026, time.November, 30, 12, 0, 0, 0, DushanbeTimeZone),
			wantFrom: time.Date(2026, time.November, 30, 0, 0, 0, 0, DushanbeTimeZone),
			wantTo:   time.Date(2026, time.December, 1, 0, 0, 0, 0, DushanbeTimeZone),
		},
		{
			name:     "last day of the year",
			moment:   time.Date(2026, time.December, 31, 20, 0, 0, 0, utc),
			wantFrom: time.Date(2027, time.January, 1, 0, 0, 0, 0, DushanbeTimeZone),
			wantTo:   time.Date(2027, time.January, 2, 0, 0, 0, 0, DushanbeTimeZone),
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
	from, to := dayBounds(time.Date(2026, time.November, 24, 9, 0, 0, 0, DushanbeTimeZone))

	if got := to.Sub(from); got != 24*time.Hour {
		t.Fatalf("day length = %v, want 24h", got)
	}
}

func TestDushanbeTimeZoneIsUTCPlusFive(t *testing.T) {
	_, offset := time.Date(2026, time.July, 1, 0, 0, 0, 0, DushanbeTimeZone).Zone()

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

	dayRows   []models.ReportRow
	dayErr    error
	dayCalled bool
	dayFrom   time.Time
	dayTo     time.Time

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

func (m *reportStoreMock) MonthlyReportByPeriod(_ context.Context, from time.Time, to time.Time) ([]models.ReportRow, error) {
	m.dayCalled = true
	m.dayFrom = from
	m.dayTo = to

	return m.dayRows, m.dayErr
}

func TestDailyReportAsksForTheDushanbeDay(t *testing.T) {
	store := &reportStoreMock{}
	service := NewReportService(store)

	// 23:30 UTC on the 24th is already 04:30 on the 25th in Dushanbe.
	moment := time.Date(2026, time.November, 24, 23, 30, 0, 0, time.UTC)

	if _, err := service.DailyReport(context.Background(), moment); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantFrom := time.Date(2026, time.November, 25, 0, 0, 0, 0, DushanbeTimeZone)
	wantTo := time.Date(2026, time.November, 26, 0, 0, 0, 0, DushanbeTimeZone)

	if !store.called {
		t.Fatal("the store was not called")
	}
	if !store.from.Equal(wantFrom) || !store.to.Equal(wantTo) {
		t.Fatalf("period = [%v, %v), want [%v, %v)", store.from, store.to, wantFrom, wantTo)
	}
}

func TestDailyReportFillsMerchForEveryRow(t *testing.T) {
	onTime := time.Date(2026, time.November, 24, 9, 15, 0, 0, DushanbeTimeZone)
	const at1030 = 10*3600 + 30*60

	store := &reportStoreMock{
		rows: []models.ReportRow{
			{SRCode: "SAM-1", FullName: "Иванов Иван", PhotosNum: 10, PhotoPerUnit: 4, FirstPhotoAt: &onTime, LateAfter: at1030},
			{SRCode: "SAM-2", FullName: "Петров Пётр", PhotosNum: 1, PhotoPerUnit: 4, FirstPhotoAt: &onTime, LateAfter: at1030},
			{SRCode: "SAE-1", FullName: "Сидоров Сидор", PhotosNum: 7, PhotoPerUnit: 2, FirstPhotoAt: &onTime, LateAfter: at1030},
			{SRCode: "SAM-3", FullName: "Без Фото", PhotosNum: 0, PhotoPerUnit: 4, LateAfter: at1030},
			{SRCode: "SAM-4", FullName: "", PhotosNum: 0, PhotoPerUnit: 4, LateAfter: at1030},
		},
	}
	service := NewReportService(store)

	report, err := service.DailyReport(context.Background(), onTime)
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

	if len(report.All) != len(want) {
		t.Fatalf("All has %d rows, want %d", len(report.All), len(want))
	}

	for i, w := range want {
		if report.All[i].SRCode != w.code {
			t.Fatalf("All row %d code = %s, want %s: order must be kept", i, report.All[i].SRCode, w.code)
		}
		if report.All[i].Merch != w.merch {
			t.Fatalf("%s merch = %d, want %d", w.code, report.All[i].Merch, w.merch)
		}
	}

	if report.All[0].FullName != "Иванов Иван" || report.All[0].PhotosNum != 10 || report.All[0].FirstPhotoAt != &onTime {
		t.Fatalf("other fields of the row were changed: %+v", report.All[0])
	}
}

func TestDailyReportSplitsNotSentAndLate(t *testing.T) {
	const at1030 = 10*3600 + 30*60
	const at1000 = 10 * 3600

	at := func(h, m int) *time.Time {
		v := time.Date(2026, time.November, 24, h, m, 0, 0, DushanbeTimeZone)
		return &v
	}
	afterGrace := func(h, m int) *time.Time {
		v := at(h, m).Add(lateGrace + time.Minute)
		return &v
	}

	store := &reportStoreMock{
		rows: []models.ReportRow{
			{SRCode: "SAM-1", FullName: "Вовремя", PhotosNum: 4, PhotoPerUnit: 4, FirstPhotoAt: at(9, 50), LateAfter: at1030},
			{SRCode: "SAM-2", FullName: "Опоздал", PhotosNum: 4, PhotoPerUnit: 4, FirstPhotoAt: afterGrace(10, 30), LateAfter: at1030},
			{SRCode: "SAM-3", FullName: "Не прислал", PhotosNum: 0, PhotoPerUnit: 4, LateAfter: at1030},
			{SRCode: "SAM-4", FullName: "", PhotosNum: 0, PhotoPerUnit: 4, LateAfter: at1030},
			{SRCode: "SADF1", FullName: "Опт опоздал", PhotosNum: 2, PhotoPerUnit: 4, FirstPhotoAt: afterGrace(10, 0), LateAfter: at1000},
			{SRCode: "SADF2", FullName: "Опт вовремя", PhotosNum: 2, PhotoPerUnit: 4, FirstPhotoAt: at(10, 0), LateAfter: at1000},
		},
	}
	service := NewReportService(store)

	report, err := service.DailyReport(context.Background(), *at(16, 0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	codes := func(rows []models.ReportRow) []string {
		var out []string
		for _, row := range rows {
			out = append(out, row.SRCode)
		}
		return out
	}

	if got := codes(report.NotSent); len(got) != 1 || got[0] != "SAM-3" {
		t.Fatalf("NotSent = %v, want [SAM-3]: vacant SAM-4 must not be listed", got)
	}

	if got := codes(report.Late); len(got) != 2 || got[0] != "SAM-2" || got[1] != "SADF1" {
		t.Fatalf("Late = %v, want [SAM-2 SADF1]", got)
	}

	if len(report.All) != 6 {
		t.Fatalf("All has %d rows, want all 6", len(report.All))
	}

	if report.Late[0].Merch != 1 {
		t.Fatalf("late row merch = %d, want 1: rows must be copied after merch is set", report.Late[0].Merch)
	}
}

func TestDailyReportStoreError(t *testing.T) {
	dbErr := errors.New("database is down")
	store := &reportStoreMock{err: dbErr}
	service := NewReportService(store)

	report, err := service.DailyReport(context.Background(), time.Now())
	if !errors.Is(err, dbErr) {
		t.Fatalf("error = %v, want %v", err, dbErr)
	}
	if report != nil {
		t.Fatalf("report = %+v, want nil on error", report)
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

	report, err := service.DailyReport(context.Background(), time.Now())
	if !errors.Is(err, ErrInvalidCoefficient) {
		t.Fatalf("error = %v, want ErrInvalidCoefficient", err)
	}
	if !strings.Contains(err.Error(), "SAE-7") {
		t.Fatalf("error %q does not name the broken code SAE-7", err)
	}
	if report != nil {
		t.Fatalf("report = %+v, want nil on error", report)
	}
}

func TestDailyReportEmpty(t *testing.T) {
	service := NewReportService(&reportStoreMock{rows: nil})

	report, err := service.DailyReport(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if report == nil {
		t.Fatal("report = nil, want an empty report")
	}
	if len(report.All) != 0 || len(report.NotSent) != 0 || len(report.Late) != 0 {
		t.Fatalf("report = %+v, want empty lists", report)
	}
}

func TestIsLate(t *testing.T) {
	const at1030 = 10*3600 + 30*60
	const at1000 = 10 * 3600

	// at returns a Dushanbe time on a fixed day, shifted by extra.
	at := func(h, m int, extra time.Duration) *time.Time {
		v := time.Date(2026, time.November, 27, h, m, 0, 0, DushanbeTimeZone).Add(extra)
		return &v
	}

	tests := []struct {
		name      string
		photo     *time.Time
		lateAfter int
		want      bool
	}{
		{name: "no photo is not late", photo: nil, lateAfter: at1030, want: false},
		{name: "before the cutoff", photo: at(10, 25, 0), lateAfter: at1030, want: false},
		{name: "exactly at the cutoff", photo: at(10, 30, 0), lateAfter: at1030, want: false},
		{name: "inside the grace time", photo: at(10, 30, lateGrace/2), lateAfter: at1030, want: false},
		{name: "exactly at the end of the grace time", photo: at(10, 30, lateGrace), lateAfter: at1030, want: false},
		{name: "one second after the grace time", photo: at(10, 30, lateGrace+time.Second), lateAfter: at1030, want: true},
		{name: "much later", photo: at(13, 0, 0), lateAfter: at1030, want: true},
		{name: "ОПТ cutoff 10:00, after its grace", photo: at(10, 0, lateGrace+time.Minute), lateAfter: at1000, want: true},
		{name: "ОПТ cutoff 10:00, inside its grace", photo: at(10, 0, lateGrace-time.Minute), lateAfter: at1000, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isLate(tt.photo, tt.lateAfter); got != tt.want {
				t.Fatalf("isLate(%v, %d) = %v, want %v", tt.photo, tt.lateAfter, got, tt.want)
			}
		})
	}
}

func TestIsLateUsesDushanbeTime(t *testing.T) {
	// 05:45 UTC is 10:45 in Dushanbe.
	photo := time.Date(2026, time.November, 27, 5, 45, 0, 0, time.UTC)
	lateAfter := 10*3600 + 30*60

	want := 10*3600+45*60 > lateAfter+int(lateGrace.Seconds())
	if got := isLate(&photo, lateAfter); got != want {
		t.Fatalf("isLate(05:45 UTC) = %v, want %v (10:45 in Dushanbe)", got, want)
	}
}

func TestMonthToDateBounds(t *testing.T) {
	tests := []struct {
		name     string
		moment   time.Time
		wantFrom time.Time
		wantTo   time.Time
	}{
		{
			name:     "middle of the month",
			moment:   time.Date(2026, time.November, 28, 16, 0, 0, 0, DushanbeTimeZone),
			wantFrom: time.Date(2026, time.November, 1, 0, 0, 0, 0, DushanbeTimeZone),
			wantTo:   time.Date(2026, time.November, 29, 0, 0, 0, 0, DushanbeTimeZone),
		},
		{
			name:     "first day of the month covers only that day",
			moment:   time.Date(2026, time.December, 1, 16, 0, 0, 0, DushanbeTimeZone),
			wantFrom: time.Date(2026, time.December, 1, 0, 0, 0, 0, DushanbeTimeZone),
			wantTo:   time.Date(2026, time.December, 2, 0, 0, 0, 0, DushanbeTimeZone),
		},
		{
			name:     "last day of the year",
			moment:   time.Date(2026, time.December, 31, 16, 0, 0, 0, DushanbeTimeZone),
			wantFrom: time.Date(2026, time.December, 1, 0, 0, 0, 0, DushanbeTimeZone),
			wantTo:   time.Date(2027, time.January, 1, 0, 0, 0, 0, DushanbeTimeZone),
		},
		{
			name:     "late evening UTC on the 30th is already the 1st in Dushanbe",
			moment:   time.Date(2026, time.November, 30, 21, 0, 0, 0, time.UTC),
			wantFrom: time.Date(2026, time.December, 1, 0, 0, 0, 0, DushanbeTimeZone),
			wantTo:   time.Date(2026, time.December, 2, 0, 0, 0, 0, DushanbeTimeZone),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, to := monthToDateBounds(tt.moment)

			if !from.Equal(tt.wantFrom) {
				t.Fatalf("from = %v, want %v", from, tt.wantFrom)
			}
			if !to.Equal(tt.wantTo) {
				t.Fatalf("to = %v, want %v", to, tt.wantTo)
			}
		})
	}
}

func TestMonthToDateMerchSumsDailyMerchPerCode(t *testing.T) {
	store := &reportStoreMock{
		dayRows: []models.ReportRow{
			// SAM-1: three days with 2, 1 and 3 photos at 4 per merch -> 1 + 0 + 1 = 2.
			{SRCode: "SAM-1", FullName: "Иванов Иван", Territory: "Розница 1", PhotosNum: 2, PhotoPerUnit: 4},
			{SRCode: "SAM-1", FullName: "Иванов Иван", Territory: "Розница 1", PhotosNum: 1, PhotoPerUnit: 4},
			{SRCode: "SAM-1", FullName: "Иванов Иван", Territory: "Розница 1", PhotosNum: 3, PhotoPerUnit: 4},
			// SAM-2: one row without photos.
			{SRCode: "SAM-2", FullName: "Петров Пётр", Territory: "Розница 1", PhotosNum: 0, PhotoPerUnit: 4},
			// SAE-1: two days with 3 and 1 photos at 2 per merch -> 2 + 1 = 3.
			{SRCode: "SAE-1", FullName: "Сидоров Сидор", Territory: "Розница 2", PhotosNum: 3, PhotoPerUnit: 2},
			{SRCode: "SAE-1", FullName: "Сидоров Сидор", Territory: "Розница 2", PhotosNum: 1, PhotoPerUnit: 2},
			// Vacant code: one row, no name, no photos.
			{SRCode: "SAE-2", FullName: "", Territory: "Розница 2", PhotosNum: 0, PhotoPerUnit: 2},
		},
	}
	service := NewReportService(store)

	rows, err := service.MonthToDateMerch(context.Background(), time.Date(2026, time.November, 28, 16, 0, 0, 0, DushanbeTimeZone))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []struct {
		code   string
		name   string
		merch  int
		photos int
	}{
		{"SAM-1", "Иванов Иван", 2, 6},
		{"SAM-2", "Петров Пётр", 0, 0},
		{"SAE-1", "Сидоров Сидор", 3, 4},
		{"SAE-2", "", 0, 0},
	}

	if len(rows) != len(want) {
		t.Fatalf("got %d rows, want one per code (%d): %+v", len(rows), len(want), rows)
	}

	for i, w := range want {
		row := rows[i]
		if row.SRCode != w.code {
			t.Fatalf("row %d code = %s, want %s: order must be kept", i, row.SRCode, w.code)
		}
		if row.FullName != w.name {
			t.Fatalf("%s name = %q, want %q", w.code, row.FullName, w.name)
		}
		if row.Merch != w.merch {
			t.Fatalf("%s month merch = %d, want %d (sum of daily merch)", w.code, row.Merch, w.merch)
		}
		if row.PhotosNum != w.photos {
			t.Fatalf("%s month photos = %d, want %d", w.code, row.PhotosNum, w.photos)
		}
	}
}

func TestMonthToDateMerchIsNotMerchOfTheMonthTotal(t *testing.T) {
	// Two days with 2 photos at 4 per merch: 1 + 1 = 2, not 4 / 4 = 1.
	store := &reportStoreMock{
		dayRows: []models.ReportRow{
			{SRCode: "SAM-1", PhotosNum: 2, PhotoPerUnit: 4},
			{SRCode: "SAM-1", PhotosNum: 2, PhotoPerUnit: 4},
		},
	}
	service := NewReportService(store)

	rows, err := service.MonthToDateMerch(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rows) != 1 || rows[0].Merch != 2 {
		t.Fatalf("rows = %+v, want one row with merch 2", rows)
	}
}

func TestMonthToDateMerchAsksForTheMonth(t *testing.T) {
	store := &reportStoreMock{}
	service := NewReportService(store)

	moment := time.Date(2026, time.November, 28, 16, 0, 0, 0, DushanbeTimeZone)
	if _, err := service.MonthToDateMerch(context.Background(), moment); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantFrom := time.Date(2026, time.November, 1, 0, 0, 0, 0, DushanbeTimeZone)
	wantTo := time.Date(2026, time.November, 29, 0, 0, 0, 0, DushanbeTimeZone)

	if !store.dayCalled {
		t.Fatal("the store was not called")
	}
	if !store.dayFrom.Equal(wantFrom) || !store.dayTo.Equal(wantTo) {
		t.Fatalf("period = [%v, %v), want [%v, %v)", store.dayFrom, store.dayTo, wantFrom, wantTo)
	}
}

func TestMonthToDateMerchErrors(t *testing.T) {
	dbErr := errors.New("database is down")

	t.Run("store error", func(t *testing.T) {
		service := NewReportService(&reportStoreMock{dayErr: dbErr})

		rows, err := service.MonthToDateMerch(context.Background(), time.Now())
		if !errors.Is(err, dbErr) {
			t.Fatalf("error = %v, want %v", err, dbErr)
		}
		if rows != nil {
			t.Fatalf("rows = %v, want nil on error", rows)
		}
	})

	t.Run("invalid coefficient names the code", func(t *testing.T) {
		service := NewReportService(&reportStoreMock{
			dayRows: []models.ReportRow{{SRCode: "SAE-7", PhotosNum: 3, PhotoPerUnit: 0}},
		})

		rows, err := service.MonthToDateMerch(context.Background(), time.Now())
		if !errors.Is(err, ErrInvalidCoefficient) || !strings.Contains(err.Error(), "SAE-7") {
			t.Fatalf("error = %v, want ErrInvalidCoefficient naming SAE-7", err)
		}
		if rows != nil {
			t.Fatalf("rows = %v, want nil on error", rows)
		}
	})
}
