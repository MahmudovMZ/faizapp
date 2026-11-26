package models

import "time"

type ReportRow struct {
	SRCode       string
	FullName     string
	PhotosNum    int
	PhotoPerUnit int
	FirstPhotoAt *time.Time
	Merch        int
}
