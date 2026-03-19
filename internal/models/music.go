package models

type Music struct {
	ID          string
	Name        string
	UploaderID  string
	CoverURL string
	SongURL string
	Likes       int
	DurationSec int
}
