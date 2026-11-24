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
