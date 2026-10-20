package models

import "time"

type Photo struct {
	ID                int64
	UserID            string
	TelegramUserID    int64
	TelegramMessageID int
	Sha256Hash        string
	Caption           string
	ReceivedAt        time.Time
	CreatedAt         time.Time
}
