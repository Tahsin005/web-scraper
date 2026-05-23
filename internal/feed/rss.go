package feed

import (
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Language    string    `xml:"language"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

func URLToFeed(url string) (RSSFeed, error) {
	if url == "" {
		return RSSFeed{}, fmt.Errorf("URL cannot be empty")
	}

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := httpClient.Get(url)
	if err != nil {
		if netErr, ok := err.(net.Error); ok {
			if netErr.Timeout() {
				return RSSFeed{}, fmt.Errorf("request timeout: the RSS feed URL took too long to respond (>10 seconds)")
			}
		}
		if err.Error() == "EOF" {
			return RSSFeed{}, fmt.Errorf("connection closed unexpectedly: remote server closed connection without response")
		}
		return RSSFeed{}, fmt.Errorf("failed to fetch RSS feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes := make([]byte, 500)
		n, _ := resp.Body.Read(bodyBytes)
		bodyPreview := string(bodyBytes[:n])
		if len(bodyPreview) > 100 {
			bodyPreview = bodyPreview[:100] + "..."
		}
		return RSSFeed{}, fmt.Errorf("HTTP %d error: %s (response preview: %s)", resp.StatusCode, http.StatusText(resp.StatusCode), bodyPreview)
	}

	contentType := resp.Header.Get("Content-Type")
	if contentType != "" && !isValidFeedContentType(contentType) {
		fmt.Printf("Warning: unexpected content-type '%s' for RSS feed\n", contentType)
	}

	limitedBody := io.LimitReader(resp.Body, 50*1024*1024)

	data, err := io.ReadAll(limitedBody)
	if err != nil {
		return RSSFeed{}, fmt.Errorf("failed to read response body: %w", err)
	}

	if len(data) == 0 {
		return RSSFeed{}, fmt.Errorf("empty response body from RSS feed")
	}

	var feed RSSFeed
	err = xml.Unmarshal(data, &feed)
	if err != nil {
		return RSSFeed{}, fmt.Errorf("failed to parse XML: invalid or malformed RSS feed - %w", err)
	}

	return feed, nil
}

func isValidFeedContentType(contentType string) bool {
	validTypes := []string{
		"application/rss+xml",
		"application/atom+xml",
		"application/xml",
		"text/xml",
		"text/rss",
		"application/json",
	}

	for _, validType := range validTypes {
		if validType == contentType {
			return true
		}
	}

	return false
}
