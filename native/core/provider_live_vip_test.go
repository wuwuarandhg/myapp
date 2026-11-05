package core

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func TestLiveVIPProbe(t *testing.T) {
	if os.Getenv("CHECK_LIVE_VIP") != "true" {
		t.Skip("set CHECK_LIVE_VIP=true to probe vip episode availability")
	}
	engine, err := newNativeEngine(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(engine.downloads.close)
	d := engine.downloader
	type probe struct {
		source string
		items  []Drama
	}
	var probes []probe
	hongguo, err := d.fetchHongguoAppCatalogCategory(context.Background(), "")
	if err != nil || len(hongguo) == 0 {
		t.Logf("hongguo catalog unavailable: %v", err)
	} else {
		probes = append(probes, probe{sourceHongguo, hongguo})
	}
	if items, _, err := d.fetchHuangguoAICatalogPage(context.Background(), 1, ""); err == nil && len(items) > 0 {
		probes = append(probes, probe{sourceHuangguoAI, items})
	}
	if items, _, err := d.fetchDSDCatalogPage(context.Background(), 1, "", ""); err == nil && len(items) > 0 {
		probes = append(probes, probe{sourceDSD, items})
	}
	for _, current := range probes {
		current := current
		t.Run(current.source, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
			defer cancel()
			limit := len(current.items)
			if limit > 3 {
				limit = 3
			}
			for _, drama := range current.items[:limit] {
				var chapters []Chapter
				switch current.source {
				case sourceHongguo:
					_, chapters, err = d.fetchHongguoChapters(ctx, drama.SourceID)
				case sourceHuangguoAI:
					_, chapters, err = d.fetchHuangguoAIDetail(ctx, drama.SourceID)
				case sourceDSD:
					_, chapters, err = d.fetchDSDDetail(ctx, drama.SourceID)
				}
				if err != nil || len(chapters) == 0 {
					t.Logf("%s %q detail failed: %v", current.source, drama.Title, err)
					continue
				}
				vipMarked := 0
				for _, chapter := range chapters {
					if chapter.VIP {
						vipMarked++
					}
				}
				picks := []int{0, len(chapters) / 2, len(chapters) - 1}
				for _, index := range picks {
					if index < 0 || index >= len(chapters) {
						continue
					}
					task := Task{DramaID: drama.ID, DramaTitle: drama.Title, Chapter: chapters[index], Index: index + 1}
					media, err := d.resolveProviderMedia(ctx, task)
					marker := ""
					if chapters[index].VIP {
						marker = "[VIP]"
					}
					if err != nil {
						t.Logf("%s %q ep%d/%d%s: FAIL %v", current.source, drama.Title, index+1, len(chapters), marker, err)
						continue
					}
					address := media.URL
					if address == "" && len(media.Variants) > 0 {
						address = media.Variants[0].URL
					}
					status, kind, size := liveProbeURL(address, media.Referer)
					t.Logf("%s %q ep%d/%d%s: OK dur=%s probe=%d type=%s bytes=%d", current.source, drama.Title, index+1, len(chapters), marker, media.Duration, status, kind, size)
				}
				if current.source == sourceDSD {
					t.Logf("%s %q vipChapters=%d total=%d", current.source, drama.Title, vipMarked, len(chapters))
				}
			}
		})
	}
	fmt.Println("done")
}
