package downloader

/*
 TODO features
 1. connect to DB and save download status
*/

import (
	"io"
)

type Params map[string]string

type FileInfo struct {
}

// TODO better comment
// 1. downloaders are one-off
type FileDownloader interface {
	GetInfo() (FileInfo, error)
	Download() (chan byte, error)
}

type VideoInfo struct {
	Title string
}

type VideoDownloader interface {
	FileDownloader
	GetVideoInfo() (*VideoInfo, error)
	// DownloadAndMerge() (chan byte, error)
	// DownloadByVideoId( id string) (chan byte, error)
}

type BatchDownloader interface {
	DownloadNthFileFromUrl(url string, params Params) (io.Reader, error)
	DownloadByUrl(url string, params Params, dest_dir string) (chan<- byte, error)
	DownloadByListId(id string, params Params, dest_dir string) (chan<- byte, error)
}

type BatchVideoDownloader interface {
	BatchDownloader
	GetInfos(url string, params Params) ([]*VideoInfo, error)
}
