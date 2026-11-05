package core

import (
	"context"
	"encoding/hex"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/beilunyang/openhongguo/adapters"
	"github.com/beilunyang/openhongguo/sdk"
)

const hongguoOpenReferer = "https://novel.snssdk.com/"

var hongguoOpenCache struct {
	sync.Mutex
	client *sdk.Client
	path   string
}

func (d *Downloader) hongguoOpenClient() (*sdk.Client, error) {
	directory := strings.TrimSpace(d.cfg.dataDir)
	if directory == "" {
		return nil, errors.New("缺少应用数据目录")
	}
	profile := strings.TrimRight(directory, `/\`) + "/hongguo-anonymous.json"
	hongguoOpenCache.Lock()
	defer hongguoOpenCache.Unlock()
	if hongguoOpenCache.client != nil && hongguoOpenCache.path == profile {
		return hongguoOpenCache.client, nil
	}
	client, err := adapters.NewClient(profile, sdk.Options{RetryAttempts: 2})
	if err != nil {
		return nil, err
	}
	hongguoOpenCache.client, hongguoOpenCache.path = client, profile
	return client, nil
}

func (d *Downloader) resolveHongguoOpenMedia(ctx context.Context, videoID string) (providerMedia, error) {
	client, err := d.hongguoOpenClient()
	if err != nil {
		return providerMedia{}, err
	}
	playback, err := client.ResolvePlayback(ctx, videoID, sdk.PlaybackOptions{})
	if playback == nil {
		return providerMedia{}, err
	}
	var key []byte
	if playback.Protected {
		content, keyErr := sdk.ContentKey(playback)
		if keyErr != nil {
			return providerMedia{}, keyErr
		}
		decoded, decodeErr := hex.DecodeString(content.Hex)
		if decodeErr != nil || len(decoded) != 16 {
			return providerMedia{}, errors.New("红果内容密钥无效")
		}
		key = decoded
	}
	type candidate struct {
		media providerMedia
		score int
	}
	var choices []candidate
	seen := map[string]bool{}
	for _, stream := range playback.Streams {
		address := strings.TrimSpace(stream.URL)
		if address == "" || seen[address] || !isProviderHTTPMediaURL(address) {
			continue
		}
		seen[address] = true
		codec := strings.ToLower(stream.Codec)
		if strings.Contains(codec, "bytevc2") || strings.Contains(codec, "av1") {
			continue
		}
		quality := hongguoOpenQuality(stream.Definition)
		score := quality * 10
		if strings.Contains(codec, "h264") || strings.Contains(codec, "avc") {
			score++
		}
		choices = append(choices, candidate{media: providerMedia{
			URL: address, Referer: hongguoOpenReferer, UserAgent: duanjuUserAgent, Quality: quality, CENCKey: key,
		}, score: score})
	}
	if len(choices) == 0 {
		return providerMedia{}, errors.New("红果播放模型没有可用码流")
	}
	sort.SliceStable(choices, func(left, right int) bool { return choices[left].score > choices[right].score })
	selected := choices[0].media
	selected.Variants = make([]providerMedia, 0, len(choices))
	for _, choice := range choices {
		variant := choice.media
		variant.Variants = nil
		selected.Variants = append(selected.Variants, variant)
	}
	return selected, nil
}

func hongguoOpenQuality(definition string) int {
	digits := hongguoQualityNumber.FindString(definition)
	if digits == "" {
		return 0
	}
	if value, err := strconv.Atoi(digits); err == nil {
		return value
	}
	return 0
}
