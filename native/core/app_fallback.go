package core

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

var fallbackSearchSources = []string{sourceYaguo, sourceMaoguo, sourceFanguo, sourceHeguo, sourceXingguo}

var fallbackTitleNoise = regexp.MustCompile(`(?i)[\s\p{P}\p{S}]+`)

var fallbackTitleSeasons = regexp.MustCompile(`(?i)(第[一二三四五六七八九十0-9]+季|season\s*[0-9]+|s[0-9]{1,2}|短剧|全集|高清|完整版)`)

var fallbackCache = struct {
	sync.Mutex
	entries map[string]fallbackCacheEntry
}{entries: map[string]fallbackCacheEntry{}}

type fallbackCacheEntry struct {
	dramas  []Drama
	expires time.Time
}

func fallbackTitleKey(title string) string {
	text := fallbackTitleSeasons.ReplaceAllString(title, "")
	text = fallbackTitleNoise.ReplaceAllString(text, "")
	return strings.ToLower(text)
}

func fallbackEpisodeIndex(chapters []Chapter, episode int) int {
	if episode < 1 {
		episode = 1
	}
	for index, chapter := range chapters {
		if chapter.EpisodeString(index+1) == strconv.Itoa(episode) {
			return index
		}
	}
	if episode <= len(chapters) {
		return episode - 1
	}
	return -1
}

func (engine *nativeEngine) fallbackPlayback(ctx context.Context, input nativeInput) (nativePlan, string, bool) {
	source, _, valid := splitProviderDramaID(input.Drama.ID)
	if !valid || !fallbackEligible(source) {
		return nativePlan{}, "", false
	}
	key := fallbackTitleKey(input.Drama.Title)
	if len([]rune(key)) < 2 {
		return nativePlan{}, "", false
	}
	for _, candidateSource := range fallbackSearchSources {
		if candidateSource == source || !nativeSourceAvailable(candidateSource) || !duanjuSupportsSearch(candidateSource) {
			continue
		}
		items, err := engine.cachedFallbackSearch(ctx, candidateSource, input.Drama.Title)
		if err != nil || len(items) == 0 {
			continue
		}
		for _, candidate := range items {
			if fallbackTitleKey(candidate.Title) != key {
				continue
			}
			_, chapters, err := engine.downloader.fetchDuanjuDetail(ctx, candidateSource, candidate.SourceID)
			if err != nil || len(chapters) == 0 {
				break
			}
			index := fallbackEpisodeIndex(chapters, input.Index)
			if index < 0 {
				break
			}
			media, err := engine.downloader.resolveProviderMedia(ctx, Task{
				DramaID: candidate.ID, DramaTitle: candidate.Title, Chapter: chapters[index], Index: index + 1,
			})
			if err != nil {
				continue
			}
			plan, err := engine.nativeOpenPlayback(ctx, nativePlaybackChoices(media, input.Quality))
			if err != nil {
				continue
			}
			return plan, fmt.Sprintf("原站源第 %d 集不可播放，已改用%s播放同一集", input.Index, duanjuSourceName(candidateSource)), true
		}
	}
	return nativePlan{}, "", false
}

func fallbackEligible(source string) bool {
	switch canonicalProviderSource(source) {
	case sourceHongguo, sourceHuangguoAI, sourceHuangguoVideo, sourceCloudFront, sourceHuangju, sourceDSD:
		return true
	}
	return false
}

func (engine *nativeEngine) cachedFallbackSearch(ctx context.Context, source, title string) ([]Drama, error) {
	cacheKey := source + "|" + fallbackTitleKey(title)
	fallbackCache.Lock()
	entry, found := fallbackCache.entries[cacheKey]
	fallbackCache.Unlock()
	if found && time.Now().Before(entry.expires) {
		return entry.dramas, nil
	}
	items, err := engine.downloader.searchDuanju(ctx, source, title)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	fallbackCache.Lock()
	if len(fallbackCache.entries) >= 64 {
		fallbackCache.entries = map[string]fallbackCacheEntry{}
	}
	fallbackCache.entries[cacheKey] = fallbackCacheEntry{dramas: items, expires: now.Add(30 * time.Minute)}
	fallbackCache.Unlock()
	return items, nil
}
