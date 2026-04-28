package models

type Song struct {
	ID             string
	AuthorName     string
	Title          string
	AuthorImageUrl string
	SongImageUrl   string
	SongUrl        string
}

type SongData struct {
	Name          string
	DataSong      *DataMultimedia
	Data          *DataMultimedia
	DataSongImage *DataMultimedia
}

type DataMultimedia struct {
	Data        []byte
	ContentType string
}
