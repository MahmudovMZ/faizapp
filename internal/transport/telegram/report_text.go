package telegram

import (
	"fmt"
	"strings"
	"time"

	"github.com/MahmudovMZ/faizapp/internal/models"
	"github.com/MahmudovMZ/faizapp/internal/service"
)

func noonReportText(dayReport *models.DayReport, moment time.Time) string {
	var b strings.Builder

	day := moment.In(service.DushanbeTimeZone).Format("02.01")

	// Block 1: Not sent
	fmt.Fprintf(&b, "Не отправили Мерч за %s\n", day)

	if len(dayReport.NotSent) == 0 {
		b.WriteString("Нарушений нет. Все отправили\n")
	}

	for _, row := range dayReport.NotSent {
		fmt.Fprintf(&b, "%s %s %s\n", row.SRCode, row.FullName, row.Territory)
	}

	b.WriteString("\n")

	// Block 2: late
	fmt.Fprintf(&b, "Опоздание на %s\n", day)
	if len(dayReport.Late) == 0 {
		b.WriteString("Опоздавших нет\n")
	}
	for _, row := range dayReport.Late {
		sentAt := row.FirstPhotoAt.In(service.DushanbeTimeZone).Format("15:04")
		fmt.Fprintf(&b, "%s %s %s %s\n", row.SRCode, row.FullName, row.Territory, sentAt)
	}

	return b.String()
}

func eveningReportText(dayReport *models.DayReport, moment time.Time) string {
	emptyName := "Вакантно"

	var b strings.Builder

	day := moment.In(service.DushanbeTimeZone).Format("02.01")

	// Block 1: Not sent
	fmt.Fprintf(&b, "Не отправили Мерч к 16:00 за %s\n", day)

	if len(dayReport.NotSent) == 0 {
		b.WriteString("Нарушений нет. Все отправили\n")
	}

	for _, row := range dayReport.NotSent {
		fmt.Fprintf(&b, "%s %s %s\n", row.SRCode, row.FullName, row.Territory)
	}

	b.WriteString("\n")

	// Block 2: Result of the day
	fmt.Fprintf(&b, "Итоговый отчет по Мерчу за %s\n", day)

	for _, row := range dayReport.All {
		if row.FullName == "" {
			row.FullName = emptyName
		}
		fmt.Fprintf(&b, "%s %s %s - Отправлено фото:%d; Итого Мерч: %d\n", row.SRCode, row.FullName, row.Territory, row.PhotosNum, row.Merch)
	}

	return b.String()
}

func monthReportText(report []models.ReportRow, moment time.Time) string {
	var b strings.Builder
	emptyName := "Вакантно"
	inDushanbe := moment.In(service.DushanbeTimeZone)
	to := inDushanbe.Format("02.01")
	from := "01." + inDushanbe.Format("01")

	fmt.Fprintf(&b, "Мерчендайзинг с %s по %s\n", from, to)
	for _, row := range report {
		if row.FullName == "" {
			row.FullName = emptyName
		}
		fmt.Fprintf(&b, "%s %s - Итого Мерч: %d\n", row.SRCode, row.FullName, row.Merch)
	}

	return b.String()
}
