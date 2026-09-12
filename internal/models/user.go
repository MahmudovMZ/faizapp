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
