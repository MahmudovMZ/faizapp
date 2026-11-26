//go:build integration

package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Merch rule: in the СМ and Хорека groups every SR code counts 2 photos per merch.
// Migration 23 moves the СМ mix codes to the «Микс СМ» brand to make this true.
func TestSMAndHorecaCodesUseTwoPhotosPerMerch(t *testing.T) {
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

	rows, err := pool.Query(
		ctx,
		`
		SELECT wg.name, s.code, bc.name, bc.photos_per_unit
		FROM sr_codes s
		JOIN territories t ON t.id = s.territory_id
		JOIN work_groups wg ON wg.id = t.work_group_id
		JOIN brand_categories bc ON bc.id = s.category_id
		WHERE wg.name IN ('СМ', 'Хорека')
		ORDER BY s.code
		`,
	)
	if err != nil {
		t.Fatalf("query sr codes: %v", err)
	}
	defer rows.Close()

	checked := map[string]int{}

	for rows.Next() {
		var group, code, brand string
		var photosPerUnit int

		if err := rows.Scan(&group, &code, &brand, &photosPerUnit); err != nil {
			t.Fatalf("scan sr code: %v", err)
		}

		checked[group]++

		if photosPerUnit != 2 {
			t.Errorf("%s %s (%s): photos per merch = %d, want 2", group, code, brand, photosPerUnit)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read sr codes: %v", err)
	}

	for _, group := range []string{"СМ", "Хорека"} {
		if checked[group] == 0 {
			t.Errorf("no SR codes found for group %s", group)
		}
	}
}

func TestSMMixBrandAssignedToMixCodesOnly(t *testing.T) {
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

	rows, err := pool.Query(
		ctx,
		`
		SELECT s.code
		FROM sr_codes s
		JOIN brand_categories bc ON bc.id = s.category_id
		WHERE bc.name = 'Микс СМ'
		ORDER BY s.code
		`,
	)
	if err != nil {
		t.Fatalf("query codes of Микс СМ: %v", err)
	}
	defer rows.Close()

	var codes []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			t.Fatalf("scan code: %v", err)
		}
		codes = append(codes, code)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read codes: %v", err)
	}

	want := []string{"SASM-1", "SASM-2", "SASM-3"}
	if len(codes) != len(want) {
		t.Fatalf("codes on Микс СМ = %v, want %v", codes, want)
	}
	for i := range want {
		if codes[i] != want[i] {
			t.Fatalf("codes on Микс СМ = %v, want %v", codes, want)
		}
	}
}

// The daily report lists SR codes in this order (migration 24).
func TestSRCodesReportOrder(t *testing.T) {
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

	rows, err := pool.Query(ctx, `SELECT code FROM sr_codes ORDER BY position_in_report`)
	if err != nil {
		t.Fatalf("query sr codes: %v", err)
	}
	defer rows.Close()

	var got []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			t.Fatalf("scan code: %v", err)
		}
		got = append(got, code)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("read codes: %v", err)
	}

	want := []string{
		"SAM-1", "SAM-2", "SAM-3", "SAM-4", "SAM-5", "SAM-6", "SAM-7", "SAM-8",
		"SAE-1", "SAE-2", "SAE-3", "SAE-4", "SAE-5", "SAE-6", "SAE-7", "SAE-8",
		"SAMah-1", "SAMah-2", "SAMah-3", "SAMah-4", "SAMah-5", "SAMah-6", "SAMah-7", "SAMah-8",
		"SAHO-1", "SAHO-2", "SAHO-3", "SAHO-4",
		"SASM-1", "SASM-2", "SASM-3", "SASM-4", "SASM-5", "SASM-6",
		"SADV1", "SADV2", "SADV3",
		"SADT1", "SADT2", "SADT3",
		"SADR1", "SADR2", "SADR3",
		"SADG1", "SADG2", "SADG3",
		"SADF1", "SADF2",
		"SADFT1", "SADFT2", "SADFT3",
	}

	if len(got) != len(want) {
		t.Fatalf("got %d codes, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("position %d: got %s, want %s\nfull order: %v", i+1, got[i], want[i], got)
		}
	}
}

func TestSRCodesReportPositionRequiredAndUnique(t *testing.T) {
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

	// Run each attempt in a transaction that is always rolled back,
	// so the test never leaves a code behind.
	tryInsert := func(position any) error {
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin: %v", err)
		}
		defer func() { _ = tx.Rollback(ctx) }()

		_, err = tx.Exec(
			ctx,
			`
			INSERT INTO sr_codes (territory_id, category_id, code, position_in_report)
			SELECT territory_id, category_id, 'TEST-POSITION-CODE', $1
			FROM sr_codes
			WHERE code = 'SAM-1'
			`,
			position,
		)
		return err
	}

	if err := tryInsert(nil); err == nil {
		t.Fatal("insert without a position succeeded, want NOT NULL error")
	}

	if err := tryInsert(1); err == nil {
		t.Fatal("insert with a taken position succeeded, want UNIQUE error")
	}

	if err := tryInsert(1000); err != nil {
		t.Fatalf("insert with a free position failed: %v", err)
	}
}
