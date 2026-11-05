package core

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLiveAllSourcesProbe(t *testing.T) {
	if os.Getenv("CHECK_LIVE_ALL") != "true" {
		t.Skip("set CHECK_LIVE_ALL=true to probe every live source")
	}
	engine, err := newNativeEngine(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(engine.downloads.close)
	d := engine.downloader
	scenarios := []struct {
		source  string
		catalog func(context.Context) ([]Drama, bool, error)
		detail  func(context.Context, string) ([]Chapter, error)
	}{}
	only := strings.Split(os.Getenv("CHECK_LIVE_SOURCES"), ",")
	wanted := func(source string) bool {
		if os.Getenv("CHECK_LIVE_SOURCES") == "" {
			return true
		}
		for _, name := range only {
			if strings.TrimSpace(name) == source {
				return true
			}
		}
		return false
	}
	duanjuLive := func(source string) {
		scenarios = append(scenarios, struct {
			source  string
			catalog func(context.Context) ([]Drama, bool, error)
			detail  func(context.Context, string) ([]Chapter, error)
		}{source: source,
			catalog: func(ctx context.Context) ([]Drama, bool, error) {
				return d.fetchDuanjuCatalogPage(ctx, source, 1, duanjuFirstCategory(source))
			},
			detail: func(ctx context.Context, sourceID string) ([]Chapter, error) {
				_, chapters, err := d.fetchDuanjuDetail(ctx, source, sourceID)
				return chapters, err
			}})
	}
	for _, spec := range duanjuProviderCatalog {
		if wanted(spec.ID) {
			duanjuLive(spec.ID)
		}
	}
	scenarios = append(scenarios,
		struct {
			source  string
			catalog func(context.Context) ([]Drama, bool, error)
			detail  func(context.Context, string) ([]Chapter, error)
		}{source: sourceHongguo,
			catalog: func(ctx context.Context) ([]Drama, bool, error) {
				items, err := d.fetchHongguoAppCatalogCategory(ctx, "")
				if len(items) == 0 && err != nil {
					items, _, err = d.fetchHongguoCategoryPage(ctx, "real-drama?page=1", "真人剧")
				}
				return items, false, err
			},
			detail: func(ctx context.Context, sourceID string) ([]Chapter, error) {
				_, chapters, err := d.fetchHongguoChapters(ctx, sourceID)
				return chapters, err
			}},
		struct {
			source  string
			catalog func(context.Context) ([]Drama, bool, error)
			detail  func(context.Context, string) ([]Chapter, error)
		}{source: sourceHuangdou,
			catalog: func(ctx context.Context) ([]Drama, bool, error) {
				client := newHuangdouAPIClient(d)
				var decoded any
				if err := client.call(ctx, "/drama/list", map[string]any{"page": "1", "page_size": "30"}, &decoded); err != nil {
					return nil, false, err
				}
				var items []Drama
				for _, node := range huangdouList(decoded) {
					if drama := huangdouDramaFromMap(node); drama.ID != "" {
						items = append(items, drama)
					}
				}
				return items, len(items) >= 30, nil
			},
			detail: func(ctx context.Context, sourceID string) ([]Chapter, error) {
				_, chapters, err := d.fetchHuangdouChapters(ctx, sourceID)
				return chapters, err
			}},
		struct {
			source  string
			catalog func(context.Context) ([]Drama, bool, error)
			detail  func(context.Context, string) ([]Chapter, error)
		}{source: sourceHuangguoAI,
			catalog: func(ctx context.Context) ([]Drama, bool, error) {
				return d.fetchHuangguoAICatalogPage(ctx, 1, "")
			},
			detail: func(ctx context.Context, sourceID string) ([]Chapter, error) {
				_, chapters, err := d.fetchHuangguoAIDetail(ctx, sourceID)
				return chapters, err
			}},
		struct {
			source  string
			catalog func(context.Context) ([]Drama, bool, error)
			detail  func(context.Context, string) ([]Chapter, error)
		}{source: sourceHuangguoVideo,
			catalog: func(ctx context.Context) ([]Drama, bool, error) {
				address := fmt.Sprintf("%s/videos?page=1", d.providerBaseURL(sourceHuangguoVideo))
				body, err := d.fetchProviderText(ctx, address, d.providerBaseURL(sourceHuangguoVideo)+"/")
				if err != nil {
					return nil, false, err
				}
				return parseHuangguoVideoCards(body, address), true, nil
			},
			detail: func(ctx context.Context, sourceID string) ([]Chapter, error) {
				_, chapters, err := d.fetchHuangguoVideoDetail(ctx, sourceID)
				return chapters, err
			}},
		struct {
			source  string
			catalog func(context.Context) ([]Drama, bool, error)
			detail  func(context.Context, string) ([]Chapter, error)
		}{source: sourceHuangju,
			catalog: func(ctx context.Context) ([]Drama, bool, error) {
				return d.fetchHuangjuCatalogPage(ctx, 1, "", "")
			},
			detail: func(ctx context.Context, sourceID string) ([]Chapter, error) {
				_, chapters, err := d.fetchHuangjuDetail(ctx, sourceID)
				return chapters, err
			}},
		struct {
			source  string
			catalog func(context.Context) ([]Drama, bool, error)
			detail  func(context.Context, string) ([]Chapter, error)
		}{source: sourceYeguo,
			catalog: func(ctx context.Context) ([]Drama, bool, error) {
				return d.fetchYeguoCatalogPage(ctx, 1, "", "")
			},
			detail: func(ctx context.Context, sourceID string) ([]Chapter, error) {
				_, chapters, err := d.fetchYeguoDetail(ctx, sourceID)
				return chapters, err
			}},
		struct {
			source  string
			catalog func(context.Context) ([]Drama, bool, error)
			detail  func(context.Context, string) ([]Chapter, error)
		}{source: sourceDSD,
			catalog: func(ctx context.Context) ([]Drama, bool, error) {
				return d.fetchDSDCatalogPage(ctx, 1, "", "")
			},
			detail: func(ctx context.Context, sourceID string) ([]Chapter, error) {
				_, chapters, err := d.fetchDSDDetail(ctx, sourceID)
				return chapters, err
			}},
		struct {
			source  string
			catalog func(context.Context) ([]Drama, bool, error)
			detail  func(context.Context, string) ([]Chapter, error)
		}{source: sourceCloudFront,
			catalog: func(ctx context.Context) ([]Drama, bool, error) {
				return d.fetchLegacyCatalogPage(ctx, 1)
			},
			detail: func(ctx context.Context, sourceID string) ([]Chapter, error) {
				_, chapters, err := d.fetchLegacyDetail(ctx, sourceID)
				return chapters, err
			}})
	selected := scenarios[:0]
	for _, scenario := range scenarios {
		if wanted(scenario.source) {
			selected = append(selected, scenario)
		}
	}
	for _, scenario := range selected {
		t.Run(scenario.source, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
			defer cancel()
			var items []Drama
			var err error
			for attempt := 0; attempt < 2 && len(items) == 0; attempt++ {
				if attempt > 0 {
					time.Sleep(4 * time.Second)
				}
				items, _, err = scenario.catalog(ctx)
			}
			if err != nil || len(items) == 0 {
				t.Fatalf("RESULT %-14s catalog=FAIL items=0 err=%v", scenario.source, err)
			}
			drama := items[0]
			var chapters []Chapter
			for attempt := 0; attempt < 2 && len(chapters) == 0; attempt++ {
				if attempt > 0 {
					time.Sleep(4 * time.Second)
				}
				chapters, err = scenario.detail(ctx, drama.SourceID)
			}
			if err != nil || len(chapters) == 0 {
				t.Fatalf("RESULT %-14s catalog=OK(%d) detail=FAIL err=%v", scenario.source, len(items), err)
			}
			var media providerMedia
			resolved := -1
			limit := len(chapters)
			if limit > 6 {
				limit = 6
			}
			for index := 0; index < limit; index++ {
				for attempt := 0; attempt < 2; attempt++ {
					media, err = d.resolveProviderMedia(ctx, Task{DramaID: drama.ID, DramaTitle: drama.DisplayTitle(), Chapter: chapters[index], Index: index + 1})
					if err == nil {
						break
					}
					time.Sleep(2 * time.Second)
				}
				if err == nil {
					resolved = index
					break
				}
			}
			if resolved < 0 {
				t.Fatalf("RESULT %-14s catalog=OK(%d) detail=OK(%d) resolve=FAIL err=%v", scenario.source, len(items), len(chapters), err)
			}
			choice := nativePlaybackChoices(media, 0)
			plan, err := engine.nativeOpenPlayback(ctx, choice)
			if err != nil {
				t.Fatalf("RESULT %-14s catalog=OK(%d) detail=OK(%d) ep=%d plan=FAIL err=%v", scenario.source, len(items), len(chapters), resolved+1, err)
			}
			defer engine.nativeReleasePlayback(plan.Session)
			status, kind, size := liveProbeURL(plan.URL, "")
			routes := plan.RouteCount
			t.Logf("RESULT %-14s catalog=OK(%d) detail=OK(%d) ep=%d route=%d/%d probe=%d type=%s bytes=%d", scenario.source, len(items), len(chapters), resolved+1, plan.RouteIndex+1, routes, status, kind, size)
			if status != 200 || size == 0 {
				t.Fatalf("RESULT %-14s playback route unreachable via local stream", scenario.source)
			}
		})
	}
}

func liveProbeURL(address, referer string) (int, string, int) {
	request, err := http.NewRequest(http.MethodGet, address, nil)
	if err != nil {
		return 0, "", 0
	}
	request.Header.Set("User-Agent", duanjuUserAgent)
	if referer != "" {
		request.Header.Set("Referer", referer)
	}
	client := &http.Client{Timeout: 20 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return 0, "", 0
	}
	defer response.Body.Close()
	payload, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
	return response.StatusCode, response.Header.Get("Content-Type"), len(payload)
}
