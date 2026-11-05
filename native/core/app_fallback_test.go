package core

import "testing"

func TestFallbackTitleKeyNormalizesMarketingSuffixes(t *testing.T) {
	cases := map[string]string{
		"十八岁太奶奶驾到，重整家族荣耀第五部": "十八岁太奶奶驾到重整家族荣耀第五部",
		"总裁的替身新娘（第二季）":       "总裁的替身新娘",
		"豪门千金 Season 2":      "豪门千金",
		" 短剧：闪婚老公是首富 ":       "闪婚老公是首富",
	}
	for input, expected := range cases {
		if actual := fallbackTitleKey(input); actual != expected {
			t.Fatalf("fallbackTitleKey(%q)=%q, want %q", input, actual, expected)
		}
	}
}

func TestFallbackEpisodeIndexUsesRealEpisodeNumbers(t *testing.T) {
	chapters := []Chapter{{CurrentEpisode: rawEpisode(3)}, {CurrentEpisode: rawEpisode(7)}, {CurrentEpisode: rawEpisode(9)}}
	if index := fallbackEpisodeIndex(chapters, 7); index != 1 {
		t.Fatalf("episode 7 resolved to %d", index)
	}
	if index := fallbackEpisodeIndex(chapters, 4); index != -1 {
		t.Fatalf("missing episode 4 resolved to %d", index)
	}
	if index := fallbackEpisodeIndex([]Chapter{{}, {}, {}}, 2); index != 1 {
		t.Fatalf("plain episode list resolved to %d", index)
	}
}

func TestHongguoPublicMediaAddressRequiresAvailableSeo(t *testing.T) {
	base := "https://hongguoduanju.com"
	available := map[string]any{"available": true, "itemId": "7701937936731737112"}
	if address := hongguoPublicMediaAddress(base, "7701937936731737112", available); address != "https://hongguoduanju.com/video/public/7701937936731737112/360p.mp4" {
		t.Fatalf("unexpected public address %q", address)
	}
	unavailable := map[string]any{"available": false, "itemId": "7701937936731737112"}
	if address := hongguoPublicMediaAddress(base, "7701937936731737112", unavailable); address != "" {
		t.Fatalf("unavailable episode produced %q", address)
	}
	if address := hongguoPublicMediaAddress(base, "", available); address != "" {
		t.Fatalf("missing video id produced %q", address)
	}
}
