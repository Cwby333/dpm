package models

type Like struct {
	UserID string
	MusicID string
}

type LikedTrack struct {
	MusicID string 	
	MusicName string 
	MusicCover string 
	MusicSongURL string 
	MusicUploaderID string 
	UserUsername string 
	MusicLikes int 
	MusicDurationSeconds int 
}