package models

import "time"

type PhotoJob struct {
	ID                int64
	ChatID            int64
	TelegramMessageID int
	TelegramUserID    int64
	SenderName        string
	FileID            string
	Caption           string
	MediaGroupID      string
	ReceivedAt        time.Time
	Status            string
	Attempts          int
	NextAttemptAt     time.Time
	LockedUntil       *time.Time
	LastError         string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

const (
	PhotoJobPending    = "pending"
	PhotoJobProcessing = "processing"
	PhotoJobDone       = "done"
	PhotoJobDead       = "dead"
)
