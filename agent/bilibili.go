package agent

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"

	"internal/utils"

	"downloader"
)

/**
* 1. Download Single Video
    - Automatic select best quality
    - Manually select quality
    - Merge Video
  2. Download playlists
    - playlists
    - Default Favourite Folder
    - Get Favourite Folder list
    - Specific Favourite Folder
    - Watch later
    - Updates

*/

type Bilibili struct {
	Url      string
	SessData string

	httpClient *utils.CachedHttpClient
	vt         videoType
}

type videoType int

const (
	videoType_Not_Video   videoType = iota
	videoType_Video       videoType = iota
	videoType_Bangumi     videoType = iota
	videoType_VC_Video    videoType = iota
	videoType_Live        videoType = iota
	videoType_Interactive videoType = iota
)

type streamtype struct {
	Id              string
	Quality         int
	AudioQuality    int
	Container       string
	VideoResolution string
	Desc            string
}

var streamTypes = map[int]streamtype{
	127: {Id: "hdflv2_8k", Quality: 127, AudioQuality: 30280,
		Container: "FLV", VideoResolution: "4320p", Desc: "超高清 8K"},
	126: {Id: "hdflv2_dolby", Quality: 126, AudioQuality: 30280,
		Container: "FLV", VideoResolution: "3840p", Desc: "杜比视界"},
	125: {Id: "hdflv2_hdr", Quality: 125, AudioQuality: 30280,
		Container: "FLV", VideoResolution: "2160p", Desc: "真彩 HDR"},
	120: {Id: "hdflv2_4k", Quality: 120, AudioQuality: 30280,
		Container: "FLV", VideoResolution: "2160p", Desc: "超清 4K"},
	116: {Id: "flv_p60", Quality: 116, AudioQuality: 30280,
		Container: "FLV", VideoResolution: "1080p", Desc: "高清 1080P60"},
	112: {Id: "hdflv2", Quality: 112, AudioQuality: 30280,
		Container: "FLV", VideoResolution: "1080p", Desc: "高清 1080P+"},
	80: {Id: "flv", Quality: 80, AudioQuality: 30280,
		Container: "FLV", VideoResolution: "1080p", Desc: "高清 1080P"},
	74: {Id: "flv720_p60", Quality: 74, AudioQuality: 30280,
		Container: "FLV", VideoResolution: "720p", Desc: "高清 720P60"},
	64: {Id: "flv720", Quality: 64, AudioQuality: 30280,
		Container: "FLV", VideoResolution: "720p", Desc: "高清 720P"},
	48: {Id: "hdmp4", Quality: 48, AudioQuality: 30280,
		Container: "MP4", VideoResolution: "720p", Desc: "高清 720P (MP4)"},
	32: {Id: "flv480", Quality: 32, AudioQuality: 30280,
		Container: "FLV", VideoResolution: "480p", Desc: "清晰 480P"},
	16: {Id: "flv360", Quality: 16, AudioQuality: 30216,
		Container: "FLV", VideoResolution: "360p", Desc: "流畅 360P"},
	// Quality: 15?
	0: {Id: "mp4", Quality: 0},
	1: {Id: "jpg", Quality: 0},
}

func heightToQuality(height int, qn int) int {
	var quality int
	switch {
	case height <= 360 && qn <= 16:
		quality = 16
	case height <= 480 && qn <= 32:
		quality = 32
	case height <= 720 && qn <= 64:
		quality = 64
	case height <= 1080 && qn <= 80:
		quality = 80
	case height <= 1080 && qn <= 112:
		quality = 112
	default:
		quality = 120
	}
	return quality
}

func getHeader(referer string, cookie string) map[string]string {
	// a reasonable UA
	ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/63.0.3239.84 Safari/537.36"
	headers := map[string]string{"Accept": "*/*", "Accept-Language": "en-US,en;q=0.5", "User-Agent": ua}
	if referer != "" {
		headers["referer"] = referer
	}
	if cookie != "" {
		headers["cookie"] = cookie
	}
	return headers
}

func apiUrl(avid string, cid string, qn int) string {
	return fmt.Sprintf("https://api.bilibili.com/x/player/playurl?avid=%s&cid=%s&qn=%d&type=&otype=json&fnver=0&fnval=16&fourk=1", avid, cid, qn)
}

func audioApiUrl(sid string) string {
	return fmt.Sprintf("https://www.bilibili.com/audio/music-service-c/web/url?sid=%s", sid)
}

func audioInfoApiUrl(sid string) string {
	return fmt.Sprintf("https://www.bilibili.com/audio/music-service-c/web/song/info?sid=%s", sid)
}

func audioMenuInfoApiUrl(sid string) string {
	return fmt.Sprintf("https://www.bilibili.com/audio/music-service-c/web/menu/info?sid=%s", sid)
}

func audioMenuSongApiUrl(sid string, ps int) string {
	if ps == 0 {
		ps = 100
	}
	return fmt.Sprintf("https://www.bilibili.com/audio/music-service-c/web/song/of-menu?sid=%s&pn=1&ps=%d", sid, ps)
}

func bangumiApiUrl(avid string, cid string, epid string, qn int, fnval int) string {
	if fnval == 0 {
		fnval = 16
	}
	return fmt.Sprintf("https://api.bilibili.com/pgc/player/web/playurl?avid=%s&cid=%s&qn=%d&type=&otype=json&ep_id=%s&fnver=0&fnval=%d", avid, cid, qn, epid, fnval)
}

func interfaceApiUrl(cid string, qn int) string {
	/*
	  entropy = 'rbMCKn@KuamXWlPMoJGsKcbiJKUfkPF_8dABscJntvqhRSETg'
	  appkey, sec = ''.join([chr(ord(i) + 2) for i in entropy[::-1]]).split(':')
	  params = 'appkey=%s&cid=%s&otype=json&qn=%s&quality=%s&type=' % (appkey, cid, qn, qn)
	  chksum = hashlib.md5(bytes(params + sec, 'utf8')).hexdigest()
	  return 'https://api.bilibili.com/pgc/player/web/v2/playurl?%s&sign=%s' % (params, chksum)
	*/
	appkey, secret := "iVGUTjsxvpLeuDCf", "aHRmhWMLkdeMuILqORnYZocwMBpMEOdt"
	params := fmt.Sprintf("appkey=%s&cid=%s&otype=json&qn=%d&quality=%d&type=", appkey, cid, qn, qn)
	md5Arr := md5.Sum([]byte(params + secret))
	chksum := hex.EncodeToString(md5Arr[:])
	return fmt.Sprintf("https://api.bilibili.com/pgc/player/web/v2/playurl?%s&sign=%s", params, chksum)
}

func liveApiUrl(cid string) string {
	return fmt.Sprintf("https://api.live.bilibili.com/room/v1/Room/playUrl?cid=%s&quality=0&platform=web", cid)
}

func liveRoomInfoApiUrl(roomid string) string {
	return fmt.Sprintf("https://api.live.bilibili.com/room/v1/Room/get_info?room_id=%s", roomid)
}

func liveRoomInitApiUrl(roomid string) string {
	return fmt.Sprintf("https://api.live.bilibili.com/room/v1/Room/room_init?id=%s", roomid)
}

func spacechannelApiUrl(mid string, cid string, pn int, ps int) string {
	if ps == 0 {
		ps = 100
	}
	if pn == 0 {
		pn = 1
	}
	return fmt.Sprintf("https://api.bilibili.com/x/space/channel/video?mid=%s&cid=%s&pn=%d&ps=%d&order=0&jsonp=jsonp", mid, cid, pn, ps)
}

func spaceCollectionApiUrl(mid string, cid string, pn int, ps int) string {
	if pn == 0 {
		pn = 1
	}
	if ps == 0 {
		ps = 30
	}
	return fmt.Sprintf("https://api.bilibili.com/x/polymer/space/seasons_archives_list?mid=%s&season_id=%s&sort_reverse=false&page_num=%d&page_size=%d", mid, cid, pn, ps)
}

func seriesArchivesApiUrl(mid string, cid string, pn int, ps int) string {
	if pn == 0 {
		pn = 1
	}
	if ps == 0 {
		ps = 100
	}
	return fmt.Sprintf("https://api.bilibili.com/x/series/archives?mid=%s&series_id=%s&pn=%d&ps=%d&only_normal=true&sort=asc&jsonp=jsonp", mid, cid, pn, ps)
}

func spaceFavlistApiUrl(fid string, pn int, ps int) string {
	if pn == 0 {
		pn = 1
	}
	if ps == 0 {
		ps = 20
	}
	return fmt.Sprintf("https://api.bilibili.com/x/v3/fav/resource/list?media_id=%s&pn=%d&ps=%d&order=mtime&type=0&tid=0&jsonp=jsonp", fid, pn, ps)
}

func spaceVideoApi(mid string, pn int, ps int) string {
	if pn == 0 {
		pn = 1
	}
	if ps == 0 {
		ps = 50
	}
	return fmt.Sprintf("https://api.bilibili.com/x/space/arc/search?mid=%s&pn=%d&ps=%d&tid=0&keyword=&order=pubdate&jsonp=jsonp", mid, pn, ps)
}

func vcApiUrl(videoid string) string {
	return fmt.Sprintf("https://api.vc.bilibili.com/clip/v1/video/detail?video_id=%s", videoid)
}

func hApiUrl(docid string) string {
	return fmt.Sprintf("https://api.vc.bilibili.com/link_draw/v1/doc/detail?doc_id=%s", docid)
}

// getContent send http GET request to URL and returns the replied content

// The http request is appended with bilibili headers
func (b *Bilibili) getContent(url string, referer string, cookie string) ([]byte, error) {
	req, _ := http.NewRequest("GET", url, nil)
	header := getHeader(referer, cookie)
	for k, v := range header {
		req.Header.Add(k, v)
	}

	content, err := b.httpClient.Get(req)
	if err != nil {
		return nil, err
	}
	return content, err
}

// convert url of some specific format into regular video url
func (b *Bilibili) prepare() ([]byte, error) {
	// TODO: add user SESSDATA to cookies
	htmlContent, err := b.getContent(b.Url, "", "")
	if err != nil {
		htmlContent = nil
	}

	watchlaterRegex := regexp.MustCompile(`https?://(www\.)?bilibili\.com/watchlater/#/(av(\d+)|BV(\S+)/?)`)
	bangumiRegex1 := regexp.MustCompile(`https?://(www\.)?bilibili\.com/bangumi/play/ss(\d+)`)
	bangumiRegex2 := regexp.MustCompile(`https?://bangumi\.bilibili\.com/anime/(\d+)/play`)
	sRegex := regexp.MustCompile(`https?://(www\.)?bilibili\.com/s/([!/]+)`)
	festivalRegex := regexp.MustCompile(`https?://(www\.)?bilibili\.com/festival/([!/]+)`)
	var referer string

	// convert watchlater url to video url
	if watchlaterRegex.MatchString(b.Url) {
		re := regexp.MustCompile(`/((av\d+)|BV(\w+))\/?`)
		vid := re.FindStringSubmatch(b.Url)[1]
		re = regexp.MustCompile(`/p(\d+)`)
		p := "1"
		pMatch := re.FindStringSubmatch(b.Url)
		if pMatch != nil {
			p = pMatch[1]
		}
		b.Url = fmt.Sprintf("https://www.bilibili.com/video/%s?p=%s", vid, p)

		// redirect: bangumi/play/ss -> bangumi/play/ep
		// redirect: bangumi.bilibili.com/anime -> bangumi/play/ep
	} else if bangumiRegex1.MatchString(b.Url) || bangumiRegex2.MatchString(b.Url) {
		regex := regexp.MustCompile(`__INITIAL_STATE__=(.*?);\(function\(\)`)
		initialStateText := regex.FindSubmatch(htmlContent)[1]
		var j interface{}
		err := json.Unmarshal(initialStateText, &j)
		if err != nil {
			return nil, fmt.Errorf("invalid json format when parsing bangumi content: %v", err)
		}
		epID, err := utils.JsonGetString(j, "epList.[0].id")
		if err != nil {
			return nil, fmt.Errorf("invalid json data when handling bangumi content: %v", err)
		}
		b.Url = fmt.Sprintf("https://www.bilibili.com/bangumi/play/ep%s", epID)
		referer = b.Url

		// redirect: s
	} else if match := sRegex.FindStringSubmatch(b.Url); match != nil {
		suffix := match[2]
		b.Url = fmt.Sprintf("https://www.bilibili.com/%s", suffix)

		// redirect: festival
	} else if festivalRegex.MatchString(b.Url) {
		regex := regexp.MustCompile(`bvid=([^&]+)`)
		match := regex.FindStringSubmatch(b.Url)
		if match == nil {
			return nil, fmt.Errorf("format of the festival url is unexpected")
		}
		b.Url = fmt.Sprintf("https://www.bilibili.com/video/%s", match[1])
	}

	if err != nil {
		return nil, err
	}
	htmlContent, err = b.getContent(b.Url, referer, "")
	if err != nil {
		return nil, err
	}

	return htmlContent, nil
}

func (b *Bilibili) GetVideoInfo() (*downloader.VideoInfo, error) {

	// regulate url and get page content.
	htmlContent, err := b.prepare()
	if err != nil {
		return nil, err
	}

	// get video type and fetch video information
	bangumiRegex1 := regexp.MustCompile(`https?://(www\.)?bilibili\.com/bangumi/play/ep(\d+)`)
	bangumiRegex2 := regexp.MustCompile(`<meta property="og:url" content="(https://www.bilibili.com/bangumi/play/[^"]+)"`)
	liveRegex := regexp.MustCompile(`https?://live\.bilibili\.com/`)
	vcRegex := regexp.MustCompile(`https?://vc\.bilibili\.com/video/(\d+)`)
	videoRegex := regexp.MustCompile(`https?://(www\.)?bilibili\.com/video/(av(\d+)|(bv(\S+))|(BV(\S+)))`)

	switch {
	case bangumiRegex1.MatchString(b.Url):
		b.vt = videoType_Bangumi
		return b.getVideoInfoBangumi(htmlContent)

	case bangumiRegex2.Match(htmlContent):
		b.vt = videoType_Bangumi
		return b.getVideoInfoBangumi(htmlContent)

	case liveRegex.MatchString(b.Url):
		b.vt = videoType_Live
		return b.getVideoInfoLive(htmlContent)

	case vcRegex.MatchString(b.Url):
		b.vt = videoType_VC_Video
		return b.getVideoInfoVC(htmlContent)

	case videoRegex.MatchString(b.Url):
		b.vt = videoType_Video
		return b.getVideoInfo(htmlContent)
	}

	return nil, errors.ErrUnsupported
}

func (b *Bilibili) getVideoInfo(htmlContent []byte) (*downloader.VideoInfo, error) {
	initialStateRegex := regexp.MustCompile(`__INITIAL_STATE__=(.*?);\(function\(\)`)
	initialStateByte := initialStateRegex.FindSubmatch(htmlContent)[1]
	var initialStateJson interface{}
	err := json.Unmarshal(initialStateByte, &initialStateJson)
	if err != nil {
		return nil, fmt.Errorf("failed to parse initial state as json: %v", err)
	}

	videoInfo := &downloader.VideoInfo{}
	var avid, cid int
	if utils.JsonHasKey(initialStateJson, "videoData") {
		// This is a regular video

		// TODO: show warning if this is a multi-part video
		nParts, _ := utils.JsonGetInt(initialStateJson, "videoData.videos")
		isMultiPart := false
		if nParts > 1 {
			isMultiPart = true
		}

		videoInfo.Title, _ = utils.JsonGetString(initialStateJson, "videoData.title")
		pRegex1 := regexp.MustCompile(`[\?&]p=(\d+)`)
		pRegex2 := regexp.MustCompile(`/index_(\d+)`)
		p1 := pRegex1.FindStringSubmatch(b.Url)
		p2 := pRegex2.FindStringSubmatch(b.Url)
		p := 1
		if p1 != nil {
			p, _ = strconv.Atoi(p1[1])
		} else if p2 != nil {
			p, _ = strconv.Atoi(p2[1])
		}

		// refine title for multi-part video
		if isMultiPart {
			part, err := utils.JsonGetInt(initialStateJson, fmt.Sprintf("videoData.pages.[%d].part", p-1))
			if err != nil {
				// log warning
			}
			videoInfo.Title = fmt.Sprintf("%s (P%d. %d)", videoInfo.Title, p, part)
		}

		avid, err = utils.JsonGetInt(initialStateJson, "aid")
		if err != nil {
			// log
		}
		cid, err = utils.JsonGetInt(initialStateJson, fmt.Sprintf("videoData.pages.[%d].cid", p-1))
		if err != nil {
			// log
		}

		// initial state does not contain key "videoData"
		// meaning it's a festival video
	} else {
		videoInfo.Title, err = utils.JsonGetString(initialStateJson, "videoInfo.title")
		if err != nil {
			// log
		}
		avid, err = utils.JsonGetInt(initialStateJson, "videoInfo.aid")
		if err != nil {
			// log
		}
		cid, err = utils.JsonGetInt(initialStateJson, "videoInfo.cid")
		if err != nil {
			// log
		}
	}

	// Video Quality varations
	playInfoRegex := regexp.MustCompile(`__playinfo__=(.*?)</script><script>`)
	playInfoByte1 := playInfoRegex.FindSubmatch(htmlContent)[1]
	var playInfoJson1 interface{}
	if playInfoByte1 != nil {
		err = json.Unmarshal(playInfoByte1, &playInfoJson1)
		if err != nil {
			return nil, fmt.Errorf("failed to parse first playinfo data as json: %v", err)
		}
		code, err := utils.JsonGetInt(playInfoJson1, "code")
		if err != nil || code != 0 {
			playInfoJson1 = nil
		}
	}
	htmlContent2, err := b.getContent(b.Url, "", "CURRENT_FNVAL=16")
	if err != nil {
		return nil, fmt.Errorf("failed to get html content: %v", err)
	}
	var playInfoJson2 interface{}
	playInfoByte2 := playInfoRegex.FindSubmatch(htmlContent2)[1]
	if playInfoByte2 != nil {
		err = json.Unmarshal(playInfoByte2, &playInfoJson2)
		if err != nil {
			return nil, fmt.Errorf("failed to parse second playinfo data as json: %v", err)
		}
		code, err := utils.JsonGetInt(playInfoJson2, "code")
		if err != nil || code != 0 {
			playInfoJson2 = nil
		}
	}

	currentQuality, bestQuality := -1, -1
	if playInfoJson1 != nil {
		currentQuality, err = utils.JsonGetInt(playInfoJson1, "data.quality")
		if err != nil {
			currentQuality = -1
			// log
		}
		if utils.JsonHasKey(playInfoJson1, "data.accept_quality") && utils.JsonArraySize(playInfoJson1, "data.accept_quality") > 0 {
			bestQuality, err = utils.JsonGetInt(playInfoJson1, "data.accept_quality.[0]")
			if err != nil {
				bestQuality = -1
				// log
			}
		}
	}
	playInfos := make([]interface{}, 0)
	if playInfoJson1 != nil {
		playInfos = append(playInfos, playInfoJson1)
	}
	if playInfoJson2 != nil {
		playInfos = append(playInfos, playInfoJson2)
	}

	// get alternative formats from API
	qns := []int{120, 112, 80, 64, 32, 16}
	var errMessage string
	for _, qn := range qns {
		// automatic format for durl: qn=0
		// for dash, qn does not matter
		if currentQuality == -1 || qn < currentQuality {
			apiUrlStr := apiUrl(strconv.Itoa(avid), strconv.Itoa(cid), qn)
			apiContent, err := b.getContent(apiUrlStr, b.Url, "")
			if err != nil {
				return nil, fmt.Errorf("failed to get response from api url: %v", err)
			}
			var apiPlayInfoJson interface{}
			err = json.Unmarshal(apiContent, &apiPlayInfoJson)
			if err != nil {
				return nil, fmt.Errorf("failed to parse response from api url as json data: %v", err)
			}
			code, err := utils.JsonGetInt(apiPlayInfoJson, "code")
			if code == 0 {
				// success
				playInfos = append(playInfos, apiPlayInfoJson)
			} else {
				errMessage, _ = utils.JsonGetString(apiPlayInfoJson, "data.message")
			}
		}
		if bestQuality != -1 || qn < bestQuality {
			interfaceApiUrlString := interfaceApiUrl(strconv.Itoa(cid), qn)
			interfaceApiContent, err := b.getContent(interfaceApiUrlString, b.Url, "")
			if err != nil {
				return nil, fmt.Errorf("failed to get response from interface url: %v", err)
			}
			var interfaceApiJson interface{}
			err = json.Unmarshal(interfaceApiContent, &interfaceApiJson)
			if err != nil {
				return nil, fmt.Errorf("failed to parse response from interface url as json data: %v", err)
			}
			quality, err := utils.JsonGetInt(interfaceApiJson, "quality")
			if err != nil {
				// log
			}
			if quality > 0 {
				playInfos = append(playInfos, map[string]interface{}{"code": 0, "message": "0", "ttl": 1, "data": interfaceApiJson})
			}
		}
		if len(playInfos) == 0 {
			// TODO: research and replicate (if needed) the python behavior.
			return nil, fmt.Errorf("got 0 video info.")
		}

		for _, playinfo := range playInfos {
			quality, err := utils.JsonGetInt(playinfo, "data.quality")
			if err != nil {
				return nil, fmt.Errorf("ill-formated playinfo json data: %v", err)
			}
			st := streamTypes[quality]
			formatId := st.Id
			container := st.Container
			desc := st.Desc

		}
	}

	return nil, downloader.ErrUnimplemented
}

func (b *Bilibili) getVideoInfoBangumi(htmlContent []byte) (*downloader.VideoInfo, error) {
	return nil, downloader.ErrUnimplemented
}

func (b *Bilibili) getVideoInfoLive(htmlContent []byte) (*downloader.VideoInfo, error) {
	return nil, downloader.ErrUnimplemented
}

func (b *Bilibili) getVideoInfoVC(htmlContent []byte) (*downloader.VideoInfo, error) {
	return nil, downloader.ErrUnimplemented
}

/*
 *
 *
 */
func (*Bilibili) GetInfo() (*downloader.FileInfo, error) {
	return nil, errors.New("not implemented")
}
