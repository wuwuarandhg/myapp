package core

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestLiveHongguoOpenPlayback(t *testing.T) {
	if os.Getenv("CHECK_LIVE_VIP") != "true" {
		t.Skip("set CHECK_LIVE_VIP=true to probe anonymous playback")
	}
	title := os.Getenv("CHECK_LIVE_TITLE")
	if title == "" {
		title = "人前小保姆"
	}
	engine, err := newNativeEngine(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(engine.downloads.close)
	d := engine.downloader
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()
	entry, err := d.searchHongguoDramas(ctx, title)
	if err != nil || len(entry.Dramas) == 0 {
		t.Fatalf("search failed: %v", err)
	}
	drama := entry.Dramas[0]
	_, chapters, err := d.fetchHongguoChapters(ctx, drama.SourceID)
	if err != nil || len(chapters) < 2 {
		t.Fatalf("chapters failed: %v", err)
	}
	picks := []int{0, len(chapters) / 2, len(chapters) - 1}
	for _, index := range picks {
		media, err := d.resolveHongguoMedia(ctx, Task{DramaID: drama.ID, DramaTitle: drama.Title, Chapter: chapters[index], Index: index + 1})
		if err != nil {
			t.Fatalf("ep%d resolve failed: %v", index+1, err)
		}
		plan, err := engine.nativeOpenPlayback(ctx, nativePlaybackChoices(media, 0))
		if err != nil {
			t.Fatalf("ep%d plan failed: %v", index+1, err)
		}
		status, kind, size := liveProbeURL(plan.URL, "")
		engine.nativeReleasePlayback(plan.Session)
		t.Logf("%s ep%d/%d quality=%d key=%d routes=%d probe=%d type=%s bytes=%d", drama.Title, index+1, len(chapters), media.Quality, len(media.CENCKey), plan.RouteCount, status, kind, size)
		if status != 200 || size == 0 {
			t.Fatalf("ep%d route not playable", index+1)
		}
	}
}
