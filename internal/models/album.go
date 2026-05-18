package models

type Album struct {
	ID string
	Name string
	UploaderID string
}

type AlbumInfo struct {
	Album
	Username string
}