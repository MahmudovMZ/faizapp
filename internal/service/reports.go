package service

import (
	"context"
	"fmt"
	"time"

	"github.com/MahmudovMZ/faizapp/internal/models"
)

type ReportStore interface {
	PhotoReportByPeriod(ctx context.Context, startDate time.Time, endDate time.Time) ([]models.ReportRow, error)
}

type ReportService struct {
	store ReportStore
}

func NewReportService(store ReportStore) *ReportService {
	return &ReportService{store: store}
}

var dushanbeTimeZone = time.FixedZone("Dushanbe", 5*60*60)

func dayBounds(moment time.Time) (time.Time, time.Time) {
	inDushanbe := moment.In(dushanbeTimeZone)
	year, month, day := inDushanbe.Date()
	from := time.Date(year, month, day, 0, 0, 0, 0, dushanbeTimeZone)
	to := from.AddDate(0, 0, 1)
	return from, to
}

func (s *ReportService) DailyReport(ctx context.Context, moment time.Time) ([]models.ReportRow, error) {
	from, to := dayBounds(moment)

	reports, err := s.store.PhotoReportByPeriod(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("photo report by period: %w", err)
	}

	for i := range reports {
		merch, err := CalcMerch(reports[i].PhotosNum, reports[i].PhotoPerUnit)
		if err != nil {
			return nil, fmt.Errorf("error while calculating merch: %w for the user %s: ", err, reports[i].SRCode)
		}
		reports[i].Merch = merch
	}
	return reports, nil
}
