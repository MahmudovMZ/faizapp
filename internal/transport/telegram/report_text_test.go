package telegram

import (
	"strings"
	"testing"
	"time"

	"github.com/MahmudovMZ/faizapp/internal/models"
	"github.com/MahmudovMZ/faizapp/internal/service"
)

func TestNoonReportText(t *testing.T) {
	// 10:47 in Dushanbe is 05:47 UTC: the text must show Dushanbe time.
	lateAt := time.Date(2026, time.November, 28, 5, 47, 0, 0, time.UTC)

	report := &models.DayReport{
		NotSent: []models.ReportRow{
			{SRCode: "SAM-3", FullName: "Сидоров Сидор", Territory: "Розница 1"},
		},
		Late: []models.ReportRow{
			{SRCode: "SAE-4", FullName: "Петров Пётр", Territory: "Розница 2", FirstPhotoAt: &lateAt},
		},
	}

	// 23:30 UTC on the 27th is already the 28th in Dushanbe.
	moment := time.Date(2026, time.November, 27, 23, 30, 0, 0, time.UTC)

	text := noonReportText(report, moment)

	for _, want := range []string{
		"Не отправили Мерч за 28.11",
		"SAM-3 Сидоров Сидор Розница 1",
		"Опоздание на 28.11",
		"SAE-4 Петров Пётр Розница 2 10:47",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("text does not contain %q:\n%s", want, text)
		}
	}

	for _, unwanted := range []string{"Все отправили", "Опоздавших нет", "05:47", "%!"} {
		if strings.Contains(text, unwanted) {
			t.Fatalf("text must not contain %q:\n%s", unwanted, text)
		}
	}

	if strings.Index(text, "Не отправили") > strings.Index(text, "Опоздание") {
		t.Fatalf("not-sent section must come before the late section:\n%s", text)
	}
}

func TestNoonReportTextEmptyLists(t *testing.T) {
	moment := time.Date(2026, time.November, 5, 12, 30, 0, 0, service.DushanbeTimeZone)

	text := noonReportText(&models.DayReport{}, moment)

	for _, want := range []string{
		"Не отправили Мерч за 05.11",
		"Нарушений нет. Все отправили",
		"Опоздание на 05.11",
		"Опоздавших нет",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("text does not contain %q:\n%s", want, text)
		}
	}
}

func TestEveningReportText(t *testing.T) {
	report := &models.DayReport{
		NotSent: []models.ReportRow{
			{SRCode: "SAM-3", FullName: "Сидоров Сидор", Territory: "Розница 1"},
		},
		All: []models.ReportRow{
			{SRCode: "SAM-1", FullName: "Иванов Иван", Territory: "Розница 1", PhotosNum: 9, Merch: 2},
			{SRCode: "SAM-3", FullName: "Сидоров Сидор", Territory: "Розница 1", PhotosNum: 0, Merch: 0},
			{SRCode: "SAM-5", FullName: "", Territory: "Розница 2"},
		},
	}
	moment := time.Date(2026, time.November, 28, 16, 0, 0, 0, service.DushanbeTimeZone)

	text := eveningReportText(report, moment)

	for _, want := range []string{
		"Не отправили Мерч к 16:00 за 28.11",
		"SAM-3 Сидоров Сидор Розница 1",
		"Итоговый отчет по Мерчу за 28.11",
		"SAM-1 Иванов Иван Розница 1 - Отправлено фото:9; Итого Мерч: 2",
		"SAM-5 Вакантно Розница 2 - Отправлено фото:0; Итого Мерч: 0",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("text does not contain %q:\n%s", want, text)
		}
	}

	if strings.Contains(text, "%!") || strings.Contains(text, "Все отправили") {
		t.Fatalf("unexpected text:\n%s", text)
	}

	if strings.Index(text, "Не отправили") > strings.Index(text, "Итоговый") {
		t.Fatalf("not-sent section must come before the totals:\n%s", text)
	}

	if report.All[2].FullName != "" {
		t.Fatalf("the text must not change the report data, FullName = %q", report.All[2].FullName)
	}
}

func TestEveningReportTextEveryoneSent(t *testing.T) {
	moment := time.Date(2026, time.November, 28, 16, 0, 0, 0, service.DushanbeTimeZone)

	text := eveningReportText(&models.DayReport{}, moment)

	if !strings.Contains(text, "Нарушений нет. Все отправили") {
		t.Fatalf("text does not say everyone sent:\n%s", text)
	}
}

func TestMonthReportText(t *testing.T) {
	rows := []models.ReportRow{
		{SRCode: "SAM-1", FullName: "Иванов Иван", Merch: 41},
		{SRCode: "SAM-5", FullName: "", Merch: 0},
	}
	// 20:00 UTC on 30 November is already 1 December in Dushanbe.
	moment := time.Date(2026, time.November, 30, 20, 0, 0, 0, time.UTC)

	text := monthReportText(rows, moment)

	for _, want := range []string{
		"Мерчендайзинг с 01.12 по 01.12",
		"SAM-1 Иванов Иван - Итого Мерч: 41",
		"SAM-5 Вакантно - Итого Мерч: 0",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("text does not contain %q:\n%s", want, text)
		}
	}

	if strings.Index(text, "SAM-1") > strings.Index(text, "SAM-5") {
		t.Fatalf("codes must keep their order:\n%s", text)
	}
}

func TestMonthReportTextMiddleOfMonth(t *testing.T) {
	moment := time.Date(2026, time.November, 28, 16, 0, 0, 0, service.DushanbeTimeZone)

	text := monthReportText(nil, moment)

	if !strings.HasPrefix(text, "Мерчендайзинг с 01.11 по 28.11\n") {
		t.Fatalf("header = %q, want it to start with the period 01.11 - 28.11", text)
	}
}
