package models

type Playlist struct {
	ID         string
	Name       string
	UploaderID string
	Cover      string
	Private    bool
}

type PlaylistInfo struct {
	Playlist
	Username string
}
