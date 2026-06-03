package models

type Playlist struct {
	ID         string
	Name       string
	UploaderID string
	Cover      string
	Private    bool
	LikesCount int
}

type PlaylistInfo struct {
	Playlist
	Username string
}

type PlaylistUpdate struct {
	ID string
	Name *string
	Cover *string
	Private *bool
}

type PlaylistFilter struct {
	LikesMin *int
	LikesMax *int
}