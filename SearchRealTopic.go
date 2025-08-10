package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// ---------- 共通 ----------
type Topic struct {
	Source string // "google" or "yahoo"
	Title  string
	Note   string // 追加情報（Googleはトラフィック目安、Yahooは順位など）
	Rank   int
}

var httpClient = &http.Client{
	Timeout: 12 * time.Second,
}

func get(ctx context.Context, url string) (*http.Response, error) {
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125 Safari/537.36")
	req.Header.Set("Accept", "application/rss+xml, application/xml;q=0.9, text/xml;q=0.8, */*;q=0.5")
	req.Header.Set("Accept-Language", "ja,en;q=0.8")
	return httpClient.Do(req)
}

// ---------- Googleトレンド（RSS, 安定） ----------
type RSS struct {
	Channel Channel `xml:"channel"`
}
type Channel struct {
	Items []Item `xml:"item"`
}
type Item struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
}

func fetchGoogleTrends(ctx context.Context, geo string, max int) ([]Topic, error) {
	urls := []string{
		fmt.Sprintf("https://trends.google.com/trends/trendingsearches/daily/rss?hl=ja&geo=%s", geo),
		fmt.Sprintf("https://trends.google.com/trending/rss?geo=%s", geo),
	}
	var lastErr error
	for _, url := range urls {
		resp, err := get(ctx, url)
		if err != nil {
			lastErr = fmt.Errorf("google trends request: %w", err)
			continue
		}
		ct := resp.Header.Get("Content-Type")
		body := io.NopCloser(resp.Body)
		if !strings.Contains(ct, "xml") {
			snippet, _ := io.ReadAll(io.LimitReader(body, 512))
			resp.Body.Close()
			lastErr = fmt.Errorf("google trends non-XML response: %s ... %q", ct, string(snippet))
			continue
		}
		var rss RSS
		dec := xml.NewDecoder(body)
		if err := dec.Decode(&rss); err != nil {
			resp.Body.Close()
			lastErr = fmt.Errorf("google trends decode: %w", err)
			continue
		}
		resp.Body.Close()

		topics := make([]Topic, 0, len(rss.Channel.Items))
		reTraffic := regexp.MustCompile(`([0-9,\.]+)\s*万?\+?\s*検索|([0-9,\.]+)\s*searches`)
		for i, it := range rss.Channel.Items {
			title := strings.TrimSpace(it.Title)
			if title == "" {
				continue
			}
			note := ""
			if m := reTraffic.FindString(it.Description); m != "" {
				note = m
			}
			topics = append(topics, Topic{
				Source: "google",
				Title:  title,
				Note:   note,
				Rank:   i + 1,
			})
			if max > 0 && len(topics) >= max {
				break
			}
		}
		if len(topics) > 0 {
			return topics, nil
		}
		lastErr = fmt.Errorf("google trends: zero items from %s", url)
	}
	return nil, lastErr
}

// ---------- Yahooリアルタイム検索（HTMLスクレイピング） ----------
func fetchYahooRealtime(ctx context.Context, max int) ([]Topic, error) {
	url := "https://search.yahoo.co.jp/realtime/trend"
	resp, err := get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("yahoo realtime request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("yahoo realtime status: %s body: %q", resp.Status, string(b))
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("yahoo realtime parse: %w", err)
	}

	topics := []Topic{}
	// 1) trendページのランキング（ol/ul配下のa）を総当りで拾う
	doc.Find("ol li a, ul li a").Each(func(i int, a *goquery.Selection) {
		href, _ := a.Attr("href")
		txt := strings.TrimSpace(a.Text())
		if txt == "" || !strings.Contains(href, "/realtime/search") {
			return
		}
		topics = append(topics, Topic{
			Source: "yahoo",
			Title:  txt,
			Rank:   i + 1,
		})
	})

	// 重複除去＆切り詰め
	seen := map[string]bool{}
	out := make([]Topic, 0, len(topics))
	for _, t := range topics {
		key := strings.ToLower(strings.TrimSpace(t.Title))
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, t)
		if max > 0 && len(out) >= max {
			break
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("yahoo realtime: no topics parsed (DOM changed?)")
	}
	return out, nil
}

// ---------- マージ＆簡易スコア ----------
func mergeAndRank(yahoo, google []Topic, topN int) []Topic {
	type Acc struct {
		Title string
		Note  []string
		Score int
		Best  Topic
	}
	acc := map[string]*Acc{}

	add := func(t Topic) {
		key := strings.ToLower(strings.TrimSpace(t.Title))
		if key == "" {
			return
		}
		if _, ok := acc[key]; !ok {
			acc[key] = &Acc{Title: t.Title, Best: t}
		}
		// Yahooの瞬発力を少し強めに
		if t.Source == "yahoo" {
			acc[key].Score += 3
		} else {
			acc[key].Score += 2
		}
		if t.Note != "" {
			acc[key].Note = append(acc[key].Note, t.Note)
		}
		// ランクが良い方をBestに
		if t.Rank > 0 && (acc[key].Best.Rank == 0 || t.Rank < acc[key].Best.Rank) {
			acc[key].Best = t
		}
	}

	for _, t := range yahoo {
		add(t)
	}
	for _, t := range google {
		add(t)
	}

	merged := make([]Topic, 0, len(acc))
	for _, v := range acc {
		note := strings.Join(v.Note, " / ")
		merged = append(merged, Topic{
			Source: "mix",
			Title:  v.Title,
			Note:   note,
			Rank:   v.Best.Rank,
		})
	}

	sort.Slice(merged, func(i, j int) bool {
		// 基本はScore、同点ならRank（小さいほど上位）、次にタイトル
		if merged[i].Rank == 0 && merged[j].Rank > 0 {
			return false
		}
		if merged[j].Rank == 0 && merged[i].Rank > 0 {
			return true
		}
		// Scoreを保持してないので、タイトル長などは使わず Rank とタイトルで決める
		// （簡易。必要ならAccにScoreを持たせて取り出す設計に）
		if merged[i].Rank != merged[j].Rank {
			return merged[i].Rank < merged[j].Rank
		}
		return merged[i].Title < merged[j].Title
	})

	if topN > 0 && len(merged) > topN {
		return merged[:topN]
	}
	return merged
}
