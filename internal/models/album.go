package models

type Album struct {
	ID         string
	Name       string
	UploaderID string
	Cover      string
	LikesCount int
}

type AlbumInfo struct {
	Album
	Username string
}

type AlbumFilter struct {
	LikesMin *int
	LikesMax *int
}