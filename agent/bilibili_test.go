package agent

import (
	"downloader"
	"testing"
)

func TestGetVideoInfo(t *testing.T) {
	url := "https://www.bilibili.com/video/BV18J4m1n7To/?spm_id_from=333.999.list.card_archive.click&vd_source=37a28a610e3356e763e08ec5ebb1d310"
	bilibili := NewBilibili(url, "")
	infos, err := bilibili.GetResourceInfo()
	if err != nil {
		t.Fatalf("bilibili.GetVideoInfo() returned error: %v", err)
	}
	if len(infos) != 1 {
		t.Errorf("expect 1 info, got %d", len(infos))
	}
	info := infos[0]
	if info.Name != "被克格勃策反的理由可以有多离谱？【硬核狠人66】" {
		t.Errorf("expect title, got %s", info.Name)
	}
	if info.Type != downloader.RT_Video {
		t.Errorf("expect RT_Video, got %v", info.Type)
	}
	if len(info.Streams) != 2 {
		t.Errorf("expect 2 streams, got %d", len(info.Streams))
	}
}
