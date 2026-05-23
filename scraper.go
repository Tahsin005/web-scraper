package main

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/Tahsin005/web-scraper/internal/database"
)

func startScraping(
	db *database.Queries,
	concurrency int,
	timeBetweenRequests time.Duration,
) {
	log.Printf("Scarping on %v goroutines every %s duration", concurrency, timeBetweenRequests)

	ticker := time.NewTicker(timeBetweenRequests)
	for ; ; <-ticker.C {
		feeds, err := db.GetNextFeedsToFetch(
			context.Background(),
			int32(concurrency),
		)
		if err != nil {
			log.Printf("Error fetching next feeds: %v", err)
			continue
		}

		wg := &sync.WaitGroup{}
		for _, feed := range feeds {
			wg.Add(1)
			go scrapeFeed(db, wg, feed)
		}
		wg.Wait()
	}
}

func scrapeFeed(db *database.Queries, wg *sync.WaitGroup, feed database.Feed) {
	defer wg.Done()

	log.Printf("Scraping feed %s", feed.Name)
	_, err := db.MarkFeedAsFetched(
		context.Background(),
		feed.ID,
	)
	if err != nil {
		log.Printf("Error marking feed fetched: %v", err)
		return
	}

	rssFeed, err := urlToFeed(feed.Url)
	if err != nil {
		log.Printf("Error fetching RSS feed: %v", err)
		return
	}

	for _, item := range rssFeed.Channel.Item {
		log.Println("Found item", item.Title, "on feed", feed.Name)
	}

	log.Printf("Feed %s collected, %v posts found", rssFeed.Channel.Title, len(rssFeed.Channel.Item))
}
