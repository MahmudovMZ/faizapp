//go:build integration

package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/MahmudovMZ/faizapp/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPhotoReportByPeriod(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	defer pool.Close()

	repo := NewFaizAppRepo(pool)

	// Two codes nobody holds: one gets a test sales rep, the other stays vacant.
	freeCodes := findFreeSRCodes(t, ctx, pool, 2)
	assignedCode, vacantCode := freeCodes[0], freeCodes[1]

	tgID := randomTestTgID()

	var userID string
	err = pool.QueryRow(
		ctx,
		`
		INSERT INTO users (tg_id, full_name, status, role)
		VALUES ($1, $2, 'approved', $3)
		RETURNING id
		`,
		tgID,
		"Report Test User",
		models.RoleSalesRep,
	).Scan(&userID)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}

	defer func() {
		cleanup := context.Background()
		_, _ = pool.Exec(cleanup, `DELETE FROM photos WHERE user_id = $1`, userID)
		_, _ = pool.Exec(cleanup, `DELETE FROM sr_assignments WHERE user_id = $1`, userID)
		_, _ = pool.Exec(cleanup, `DELETE FROM users WHERE id = $1`, userID)
	}()

	_, err = pool.Exec(
		ctx,
		`
		INSERT INTO sr_assignments (user_id, sr_code_id)
		SELECT $1, id FROM sr_codes WHERE code = $2
		`,
		userID,
		assignedCode,
	)
	if err != nil {
		t.Fatalf("assign code: %v", err)
	}

	// A fixed day far in the past keeps the window free of other photos.
	from := time.Date(2001, time.March, 10, 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, 1)

	photoTimes := []struct {
		at     time.Time
		inside bool
	}{
		{at: from.Add(-time.Second), inside: false},  // the evening before
		{at: from, inside: true},                     // exactly at the start: included
		{at: from.Add(9 * time.Hour), inside: true},  // during the day
		{at: from.Add(15 * time.Hour), inside: true}, // during the day
		{at: to, inside: false},                      // exactly at the end: excluded
		{at: to.Add(2 * time.Hour), inside: false},   // the next day
	}

	for i, photo := range photoTimes {
		err := repo.CreatePhoto(ctx, &models.Photo{
			UserID:            userID,
			TelegramUserID:    tgID,
			TelegramMessageID: 7001 + i,
			Sha256Hash:        fmt.Sprintf("report-test-%d-%d", tgID, i),
			ReceivedAt:        photo.at,
		})
		if err != nil {
			t.Fatalf("create photo %d: %v", i, err)
		}
	}

	rows, err := repo.PhotoReportByPeriod(ctx, from, to)
	if err != nil {
		t.Fatalf("photo report: %v", err)
	}

	var codeCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM sr_codes`).Scan(&codeCount); err != nil {
		t.Fatalf("count sr codes: %v", err)
	}
	if len(rows) != codeCount {
		t.Fatalf("report has %d rows, want one per SR code (%d)", len(rows), codeCount)
	}

	byCode := map[string]models.ReportRow{}
	for _, row := range rows {
		byCode[row.SRCode] = row
	}

	t.Run("assigned code counts only photos inside the period", func(t *testing.T) {
		row, ok := byCode[assignedCode]
		if !ok {
			t.Fatalf("code %s is missing from the report", assignedCode)
		}

		if row.FullName != "Report Test User" {
			t.Fatalf("full name = %q, want %q", row.FullName, "Report Test User")
		}
		if row.PhotosNum != 3 {
			t.Fatalf("photos = %d, want 3", row.PhotosNum)
		}
		if row.PhotoPerUnit <= 0 {
			t.Fatalf("photos per unit = %d, want the brand value", row.PhotoPerUnit)
		}
		if row.FirstPhotoAt == nil || !row.FirstPhotoAt.Equal(from) {
			t.Fatalf("first photo at = %v, want %v", row.FirstPhotoAt, from)
		}
		if row.Merch != 0 {
			t.Fatalf("merch = %d, want 0: the repository must not fill it", row.Merch)
		}
	})

	t.Run("vacant code is listed with no name and no photos", func(t *testing.T) {
		row, ok := byCode[vacantCode]
		if !ok {
			t.Fatalf("code %s is missing from the report", vacantCode)
		}

		if row.FullName != "" {
			t.Fatalf("full name = %q, want empty for a vacant code", row.FullName)
		}
		if row.PhotosNum != 0 {
			t.Fatalf("photos = %d, want 0", row.PhotosNum)
		}
		if row.FirstPhotoAt != nil {
			t.Fatalf("first photo at = %v, want nil", row.FirstPhotoAt)
		}
	})

	t.Run("every row has its territory and late time", func(t *testing.T) {
		type place struct {
			territory string
			lateAfter int
		}

		dbRows, err := pool.Query(
			ctx,
			`
			SELECT s.code, t.name, EXTRACT(EPOCH FROM t.late_after)::int
			FROM sr_codes s
			JOIN territories t ON t.id = s.territory_id
			`,
		)
		if err != nil {
			t.Fatalf("query territories of codes: %v", err)
		}
		defer dbRows.Close()

		want := map[string]place{}
		for dbRows.Next() {
			var code string
			var p place
			if err := dbRows.Scan(&code, &p.territory, &p.lateAfter); err != nil {
				t.Fatalf("scan territory of code: %v", err)
			}
			want[code] = p
		}
		if err := dbRows.Err(); err != nil {
			t.Fatalf("read territories of codes: %v", err)
		}

		for _, row := range rows {
			w := want[row.SRCode]
			if row.Territory != w.territory || row.LateAfter != w.lateAfter {
				t.Fatalf("%s: territory/late = %q/%d, want %q/%d",
					row.SRCode, row.Territory, row.LateAfter, w.territory, w.lateAfter)
			}

			wantLate := 37800 // 10:30
			if row.Territory == "Опт Фаровон" {
				wantLate = 36000 // 10:00
			}
			if row.LateAfter != wantLate {
				t.Fatalf("%s (%s): late after = %d seconds, want %d", row.SRCode, row.Territory, row.LateAfter, wantLate)
			}
		}
	})

	t.Run("rows follow the report order", func(t *testing.T) {
		var wantFirst, wantLast string
		err := pool.QueryRow(
			ctx,
			`
			SELECT
				(SELECT code FROM sr_codes ORDER BY position_in_report LIMIT 1),
				(SELECT code FROM sr_codes ORDER BY position_in_report DESC LIMIT 1)
			`,
		).Scan(&wantFirst, &wantLast)
		if err != nil {
			t.Fatalf("read first and last code: %v", err)
		}

		if rows[0].SRCode != wantFirst || rows[len(rows)-1].SRCode != wantLast {
			t.Fatalf("first/last = %s/%s, want %s/%s",
				rows[0].SRCode, rows[len(rows)-1].SRCode, wantFirst, wantLast)
		}
	})
}

// findFreeSRCodes returns codes that no user holds yet.
func findFreeSRCodes(t *testing.T, ctx context.Context, pool *pgxpool.Pool, n int) []string {
	t.Helper()

	rows, err := pool.Query(
		ctx,
		`
		SELECT s.code
		FROM sr_codes s
		LEFT JOIN sr_assignments a ON a.sr_code_id = s.id
		WHERE a.sr_code_id IS NULL
		ORDER BY s.position_in_report DESC
		LIMIT $1
		`,
		n,
	)
	if err != nil {
		t.Fatalf("find free sr codes: %v", err)
	}
	defer rows.Close()

	var codes []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			t.Fatalf("scan free sr code: %v", err)
		}
		codes = append(codes, code)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read free sr codes: %v", err)
	}

	if len(codes) < n {
		t.Skipf("need %d free SR codes, found %d", n, len(codes))
	}

	return codes
}

func TestMonthlyReportByPeriodGroupsPhotosByDushanbeDay(t *testing.T) {
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatalf("create pool: %v", err)
	}
	defer pool.Close()

	repo := NewFaizAppRepo(pool)

	freeCodes := findFreeSRCodes(t, ctx, pool, 2)
	assignedCode, vacantCode := freeCodes[0], freeCodes[1]

	tgID := randomTestTgID()

	var userID string
	err = pool.QueryRow(
		ctx,
		`
		INSERT INTO users (tg_id, full_name, status, role)
		VALUES ($1, $2, 'approved', $3)
		RETURNING id
		`,
		tgID,
		"Monthly Report Test User",
		models.RoleSalesRep,
	).Scan(&userID)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}

	defer func() {
		cleanup := context.Background()
		_, _ = pool.Exec(cleanup, `DELETE FROM photos WHERE user_id = $1`, userID)
		_, _ = pool.Exec(cleanup, `DELETE FROM sr_assignments WHERE user_id = $1`, userID)
		_, _ = pool.Exec(cleanup, `DELETE FROM users WHERE id = $1`, userID)
	}()

	_, err = pool.Exec(
		ctx,
		`INSERT INTO sr_assignments (user_id, sr_code_id) SELECT $1, id FROM sr_codes WHERE code = $2`,
		userID,
		assignedCode,
	)
	if err != nil {
		t.Fatalf("assign code: %v", err)
	}

	// A month far in the past keeps the window free of other photos.
	dushanbe := time.FixedZone("Dushanbe", 5*60*60)
	from := time.Date(2001, time.April, 1, 0, 0, 0, 0, dushanbe)
	to := time.Date(2001, time.April, 4, 0, 0, 0, 0, dushanbe)

	photoTimes := []time.Time{
		// 1 April: two photos, one of them 23:30 in Dushanbe (18:30 UTC).
		time.Date(2001, time.April, 1, 9, 0, 0, 0, dushanbe),
		time.Date(2001, time.April, 1, 23, 30, 0, 0, dushanbe),
		// 2 April 00:30 in Dushanbe is still 1 April in UTC; it must count for 2 April.
		time.Date(2001, time.April, 2, 0, 30, 0, 0, dushanbe),
		// 3 April: three photos.
		time.Date(2001, time.April, 3, 10, 0, 0, 0, dushanbe),
		time.Date(2001, time.April, 3, 11, 0, 0, 0, dushanbe),
		time.Date(2001, time.April, 3, 12, 0, 0, 0, dushanbe),
		// Outside the period: the day before and the day after.
		time.Date(2001, time.March, 31, 23, 0, 0, 0, dushanbe),
		time.Date(2001, time.April, 4, 0, 0, 0, 0, dushanbe),
	}

	for i, at := range photoTimes {
		err := repo.CreatePhoto(ctx, &models.Photo{
			UserID:            userID,
			TelegramUserID:    tgID,
			TelegramMessageID: 8001 + i,
			Sha256Hash:        fmt.Sprintf("monthly-report-test-%d-%d", tgID, i),
			ReceivedAt:        at.UTC(),
		})
		if err != nil {
			t.Fatalf("create photo %d: %v", i, err)
		}
	}

	rows, err := repo.MonthlyReportByPeriod(ctx, from, to)
	if err != nil {
		t.Fatalf("monthly report: %v", err)
	}

	var assigned, vacant []models.ReportRow
	for _, row := range rows {
		switch row.SRCode {
		case assignedCode:
			assigned = append(assigned, row)
		case vacantCode:
			vacant = append(vacant, row)
		}
	}

	// One row per Dushanbe day with photos, in day order: 2, 1, 3.
	wantPhotos := []int{2, 1, 3}
	if len(assigned) != len(wantPhotos) {
		t.Fatalf("assigned code has %d rows, want %d (one per day): %+v", len(assigned), len(wantPhotos), assigned)
	}
	for i, want := range wantPhotos {
		row := assigned[i]
		if row.PhotosNum != want {
			t.Fatalf("day %d photos = %d, want %d", i+1, row.PhotosNum, want)
		}
		if row.FullName != "Monthly Report Test User" || row.Territory == "" || row.PhotoPerUnit <= 0 {
			t.Fatalf("day %d row is incomplete: %+v", i+1, row)
		}
	}

	if len(vacant) != 1 || vacant[0].PhotosNum != 0 || vacant[0].FullName != "" {
		t.Fatalf("vacant code rows = %+v, want one row with no name and 0 photos", vacant)
	}

	var codeCount int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM sr_codes`).Scan(&codeCount); err != nil {
		t.Fatalf("count sr codes: %v", err)
	}
	seen := map[string]bool{}
	for _, row := range rows {
		seen[row.SRCode] = true
	}
	if len(seen) != codeCount {
		t.Fatalf("report covers %d codes, want all %d", len(seen), codeCount)
	}
}
