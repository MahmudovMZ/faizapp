package models

import "time"

type User struct {
	ID        string
	TgID      int64
	FullName  string
	Phone     *string
	Role      *string
	Status    string
	CreatedAt time.Time
}
type SRAssignment struct {
	ID         int
	UserID     string
	SRCodeID   int
	AssignedAt time.Time
}
