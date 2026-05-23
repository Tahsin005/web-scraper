package scraper

import (
	"context"
	"database/sql"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/Tahsin005/web-scraper/internal/database"
	"github.com/Tahsin005/web-scraper/internal/feed"
	"github.com/google/uuid"
)

func StartScraping(
	db *database.Queries,
	concurrency int,
	timeBetweenRequests time.Duration,
) {
	if concurrency <= 0 {
		log.Println("Warning: concurrency must be greater than 0, using default of 10")
		concurrency = 10
	}

	if timeBetweenRequests <= 0 {
		log.Println("Warning: timeBetweenRequests must be greater than 0, using default of 1 minute")
		timeBetweenRequests = time.Minute
	}

	log.Printf("Scraping on %v goroutines every %s duration", concurrency, timeBetweenRequests)

	ticker := time.NewTicker(timeBetweenRequests)
	for ; ; <-ticker.C {
		feeds, err := db.GetNextFeedsToFetch(
			context.Background(),
			int32(concurrency),
		)
		if err != nil {
			if strings.Contains(err.Error(), "connection refused") {
				log.Printf("Error: database connection refused - is the database running? Error: %v", err)
			} else {
				log.Printf("Error fetching next feeds: %v", err)
			}
			continue
		}

		if len(feeds) == 0 {
			log.Printf("No feeds to fetch at this time")
			continue
		}

		log.Printf("Fetching %d feeds...", len(feeds))

		wg := &sync.WaitGroup{}
		for _, f := range feeds {
			wg.Add(1)
			go ScrapeFeed(db, wg, f)
		}
		wg.Wait()
	}
}

func ScrapeFeed(db *database.Queries, wg *sync.WaitGroup, f database.Feed) {
	defer wg.Done()

	if f.ID == uuid.Nil {
		log.Printf("Error: invalid feed ID (nil UUID)")
		return
	}

	if f.Name == "" {
		log.Printf("Error: invalid feed with nil name, ID: %s", f.ID)
		return
	}

	if f.Url == "" {
		log.Printf("Error: feed '%s' has no URL, skipping", f.Name)
		return
	}

	if _, err := url.Parse(f.Url); err != nil {
		log.Printf("Error: feed '%s' has invalid URL format '%s': %v", f.Name, f.Url, err)
		MarkFeedFetchError(db, f.ID, "invalid URL format")
		return
	}

	log.Printf("Scraping feed '%s' from URL: %s", f.Name, f.Url)

	_, err := db.MarkFeedAsFetched(
		context.Background(),
		f.ID,
	)
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			log.Printf("Error marking feed '%s' as fetched: database connection refused", f.Name)
		} else {
			log.Printf("Error marking feed '%s' as fetched: %v", f.Name, err)
		}
		return
	}

	rssFeed, err := feed.URLToFeed(f.Url)
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			log.Printf("Error fetching RSS feed '%s': connection refused to remote URL", f.Name)
		} else if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "i/o timeout") {
			log.Printf("Error fetching RSS feed '%s': request timeout (URL took too long to respond)", f.Name)
		} else if strings.Contains(err.Error(), "no such host") {
			log.Printf("Error fetching RSS feed '%s': hostname '%s' not found (DNS resolution failed)", f.Name, f.Url)
		} else if strings.Contains(err.Error(), "XML syntax error") || strings.Contains(err.Error(), "syntax error") {
			log.Printf("Error fetching RSS feed '%s': invalid XML format at URL - may not be a valid RSS feed", f.Name)
		} else {
			log.Printf("Error fetching RSS feed '%s': %v", f.Name, err)
		}
		MarkFeedFetchError(db, f.ID, "failed to fetch feed")
		return
	}

	if rssFeed.Channel.Title == "" {
		log.Printf("Warning: feed '%s' has no channel title in RSS", f.Name)
	}

	if len(rssFeed.Channel.Item) == 0 {
		log.Printf("Warning: feed '%s' has no items (empty RSS feed)", f.Name)
	}

	itemsCreated := 0
	itemsSkipped := 0
	itemsErrored := 0

	for _, item := range rssFeed.Channel.Item {
		if item.Title == "" && item.Description == "" && item.Link == "" {
			log.Printf("Warning: skipping RSS item in feed '%s' - item has no title, description, or link", f.Name)
			itemsSkipped++
			continue
		}

		if item.Link == "" {
			log.Printf("Warning: skipping RSS item in feed '%s' - item has no URL/link", f.Name)
			itemsSkipped++
			continue
		}

		if _, err := url.Parse(item.Link); err != nil {
			log.Printf("Warning: skipping RSS item in feed '%s' - invalid item URL format: %s", f.Name, item.Link)
			itemsSkipped++
			continue
		}

		if item.Title == "" {
			log.Printf("Warning: RSS item in feed '%s' has no title, using placeholder", f.Name)
			item.Title = "(No title)"
		}

		description := sql.NullString{}
		if item.Description != "" {
			description.String = strings.TrimSpace(item.Description)
			description.Valid = true
		}

		var pubAt time.Time
		var dateParseErr error

		pubAt, dateParseErr = time.Parse(time.RFC1123Z, item.PubDate)
		if dateParseErr != nil {
			pubAt, dateParseErr = time.Parse(time.RFC1123, item.PubDate)
			if dateParseErr != nil {
				pubAt, dateParseErr = time.Parse(time.RFC3339, item.PubDate)
				if dateParseErr != nil {
					pubAt, dateParseErr = time.Parse("Mon, 02 Jan 2006 15:04:05 MST", item.PubDate)
					if dateParseErr != nil {
						if item.PubDate == "" {
							log.Printf("Warning: RSS item in feed '%s' has no publication date, using current time", f.Name)
							pubAt = time.Now().UTC()
						} else {
							log.Printf("Warning: could not parse publication date '%s' in feed '%s', using current time", item.PubDate, f.Name)
							pubAt = time.Now().UTC()
						}
					}
				}
			}
		}

		_, err = db.CreatePost(context.Background(), database.CreatePostParams{
			ID:          uuid.New(),
			CreatedAt:   time.Now().UTC(),
			UpdatedAt:   time.Now().UTC(),
			Title:       item.Title,
			Description: description,
			PublishedAt: pubAt,
			Url:         item.Link,
			FeedID:      f.ID,
		})
		if err != nil {
			if strings.Contains(err.Error(), "duplicate key") {
				continue
			}
			if strings.Contains(err.Error(), "connection refused") {
				log.Printf("Error: database connection refused while creating post for feed '%s'", f.Name)
				break
			}
			log.Printf("Error creating post for feed '%s' (title: '%s'): %v", f.Name, item.Title, err)
			itemsErrored++
			continue
		}

		itemsCreated++
	}

	log.Printf("Feed '%s' (RSS title: '%s') completed: %d posts created, %d skipped, %d errors from %d items total",
		f.Name, rssFeed.Channel.Title, itemsCreated, itemsSkipped, itemsErrored, len(rssFeed.Channel.Item))
}

func MarkFeedFetchError(db *database.Queries, feedID uuid.UUID, reason string) {
	log.Printf("Feed %s marked with fetch error: %s", feedID, reason)
}
