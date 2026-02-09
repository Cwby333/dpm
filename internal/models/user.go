package models

import "time"

type User struct {
	ID         string
	Username   string
	HashPsw    string
	RegisterAt time.Time
	Likes      int
}
