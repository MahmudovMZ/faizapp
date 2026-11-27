package models

import "time"

type ReportRow struct {
	SRCode       string
	Territory    string
	LateAfter    int
	FullName     string
	PhotosNum    int
	PhotoPerUnit int
	FirstPhotoAt *time.Time
	Merch        int
}
type DayReport struct {
	NotSent []ReportRow
	Late    []ReportRow
	All     []ReportRow
}
