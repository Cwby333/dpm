package models

import "time"

type ListeningHistory struct {
	ID            string
	UserID        string
	MusicID       string
	ListeningDate time.Time
}

type ListeningHistoryResponse struct {
	MusicID string 	
	MusicName string 
	MusicCover string 
	MusicSongURL string 
	MusicUploaderID string 
	UserUsername string 
	MusicLikes int 
	MusicDurationSeconds int 
	ListeningDate time.Time 
}