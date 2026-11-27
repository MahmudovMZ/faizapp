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

const lateGrace = 10 * time.Minute

func dayBounds(moment time.Time) (time.Time, time.Time) {
	inDushanbe := moment.In(dushanbeTimeZone)
	year, month, day := inDushanbe.Date()
	from := time.Date(year, month, day, 0, 0, 0, 0, dushanbeTimeZone)
	to := from.AddDate(0, 0, 1)
	return from, to
}

func (s *ReportService) DailyReport(ctx context.Context, moment time.Time) (*models.DayReport, error) {
	from, to := dayBounds(moment)

	reports, err := s.store.PhotoReportByPeriod(ctx, from, to)
	if err != nil {
		return nil, fmt.Errorf("photo report by period: %w", err)
	}
	dayReports := models.DayReport{}
	for i := range reports {
		merch, err := CalcMerch(reports[i].PhotosNum, reports[i].PhotoPerUnit)
		if err != nil {
			return nil, fmt.Errorf("calculate merch for code %s: %w", reports[i].SRCode, err)
		}
		reports[i].Merch = merch

		if isLate(reports[i].FirstPhotoAt, reports[i].LateAfter) {
			dayReports.Late = append(dayReports.Late, reports[i])
		}
		if reports[i].PhotosNum == 0 && reports[i].FullName != "" {
			dayReports.NotSent = append(dayReports.NotSent, reports[i])
		}
	}
	dayReports.All = append(reports, dayReports.All...)
	return &dayReports, nil
}

func isLate(firstPhoto *time.Time, lateAfter int) bool {
	if firstPhoto == nil {
		return false
	}
	receivedAtInDushanbe := firstPhoto.In(dushanbeTimeZone)

	inDushanbeSec := receivedAtInDushanbe.Hour()*3600 + receivedAtInDushanbe.Minute()*60 + receivedAtInDushanbe.Second()
	graceSec := int(lateGrace.Seconds())

	return inDushanbeSec > lateAfter+graceSec
}
