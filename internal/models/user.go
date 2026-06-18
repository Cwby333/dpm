package models

import "time"

type User struct {
	ID             string
	Username       string
	HashPsw        string
	RegisterAt     time.Time
	Likes          int
	ListeningCount int
	FavorCount     int
	PrivateProfile bool
}

type UserFilter struct {
	MinRegisterAt *time.Time
	MaxRegisterAt *time.Time
	MinLikes *int
	MaxLikes *int
	MinLisCount *int
	MaxLisCount *int
	MinFavorCount *int
	MaxFavorCount *int
}