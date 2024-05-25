package downloader

/*
 TODO features
 1. connect to DB and save download status
*/

type Params map[string]string

type ResourceType int

const (
	RT_File ResourceType = iota
	RT_StaticHtml
	RT_Text
	RT_Image
	RT_Audio
	RT_Video
	RT_List
)

type ResourceInfo struct {
	Site         string
	Name         string
	Size         int
	Url          string
	Type         ResourceType
	DownloadWith string
	Others       map[string]string
	Streams      map[string]StreamInfo
	DashStreams  map[string]StreamInfo
}

type StreamInfo struct {
	Id           string
	Codec        string
	Resolution   [2]int
	Container    string
	Size         int
	DownloadWith string
	Url          [][]string
	Others       map[string]string
}

type Progress struct {
	Status     string
	Percentage float32
	Err        error
}

// TODO better comment
// 1. downloaders are one-off
type Downloader interface {
	CanHandle(url string) bool
	GetResourceInfo() ([]ResourceInfo, error)
	Download(index int, path string) chan *Progress
	DownloadAll(path string) (chan byte, error)
}
