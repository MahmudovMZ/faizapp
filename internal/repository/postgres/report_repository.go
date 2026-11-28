package postgres

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/MahmudovMZ/faizapp/internal/models"
)

func (r *FaizAppRepo) PhotoReportByPeriod(ctx context.Context, startDate time.Time, endDate time.Time) ([]models.ReportRow, error) {
	log.Println("[REPOSITORY] Getting photo report by period")
	var reports []models.ReportRow

	query := `SELECT s.code,
       t.name,
       EXTRACT(EPOCH FROM t.late_after)::int AS late_after_seconds,
       COALESCE(u.full_name, '') AS full_name,
       COUNT(p.id) AS photos,
       bc.photos_per_unit,
       MIN(p.received_at) AS first_photo_at
FROM sr_codes s
JOIN territories t ON t.id = s.territory_id
JOIN brand_categories bc ON bc.id = s.category_id
LEFT JOIN sr_assignments a ON a.sr_code_id = s.id
LEFT JOIN users u ON u.id = a.user_id
LEFT JOIN photos p ON p.user_id = u.id
                  AND p.received_at >= $1
                  AND p.received_at < $2
GROUP BY s.id, u.full_name, bc.photos_per_unit, t.name, t.late_after
ORDER BY s.position_in_report`

	rows, err := r.Pool.Query(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("error while getting report data: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		report := models.ReportRow{}
		if err := rows.Scan(
			&report.SRCode,
			&report.Territory,
			&report.LateAfter,
			&report.FullName,
			&report.PhotosNum,
			&report.PhotoPerUnit,
			&report.FirstPhotoAt); err != nil {
			return nil, fmt.Errorf("error while scanning report data: %w", err)
		}

		reports = append(reports, report)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error while getting report data: %w", err)
	}

	return reports, nil
}

func (r *FaizAppRepo) MonthlyReportByPeriod(ctx context.Context, startDate time.Time, endDate time.Time) ([]models.ReportRow, error) {
	log.Println("[REPOSITORY] Getting monthly report by period")
	var reports []models.ReportRow

	query := `SELECT s.code, COALESCE(u.full_name, '') AS full_name, t.name, bc.photos_per_unit,COUNT(p.id) AS photos
			  FROM sr_codes s JOIN territories t ON t.id = s.territory_id
			  JOIN brand_categories bc ON bc.id = s.category_id
			  LEFT JOIN sr_assignments a ON a.sr_code_id = s.id
			  LEFT JOIN users u ON u.id = a.user_id
			  LEFT JOIN photos p ON p.user_id = u.id AND p.received_at >= $1
			  AND p.received_at < $2
			  GROUP BY s.id, u.full_name,  t.name, bc.photos_per_unit,
					   (p.received_at AT TIME ZONE 'Asia/Dushanbe')::date
              ORDER BY s.position_in_report,
					   (p.received_at AT TIME ZONE 'Asia/Dushanbe')::date`

	rows, err := r.Pool.Query(ctx, query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("error while getting report data: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		report := models.ReportRow{}
		if err := rows.Scan(
			&report.SRCode,
			&report.FullName,
			&report.Territory,
			&report.PhotoPerUnit,
			&report.PhotosNum); err != nil {
			return nil, fmt.Errorf("error while scanning report data: %w", err)
		}

		reports = append(reports, report)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error while getting report data: %w", err)
	}

	return reports, nil
}
