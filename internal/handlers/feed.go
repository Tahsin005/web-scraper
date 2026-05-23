package handlers

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Tahsin005/web-scraper/internal/database"
	"github.com/google/uuid"
)

func (h *Handler) HandlerCreateFeed(w http.ResponseWriter, r *http.Request, user database.User) {
	type parameters struct {
		Name string `json:"name"`
		Url  string `json:"url"`
	}

	params := parameters{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		RespondWithError(w, 400, "Invalid JSON: unable to parse request body")
		return
	}

	if params.Name == "" {
		RespondWithError(w, 400, "Validation error: name is required and cannot be empty")
		return
	}

	trimmedName := strings.TrimSpace(params.Name)
	if trimmedName == "" {
		RespondWithError(w, 400, "Validation error: name cannot be only whitespace")
		return
	}

	if len(trimmedName) > 255 {
		RespondWithError(w, 400, "Validation error: name must be 255 characters or less")
		return
	}

	if params.Url == "" {
		RespondWithError(w, 400, "Validation error: url is required and cannot be empty")
		return
	}

	trimmedUrl := strings.TrimSpace(params.Url)
	if trimmedUrl == "" {
		RespondWithError(w, 400, "Validation error: url cannot be only whitespace")
		return
	}

	parsedUrl, err := url.Parse(trimmedUrl)
	if err != nil {
		RespondWithError(w, 400, "Validation error: url must be a valid URL")
		return
	}

	if parsedUrl.Scheme == "" {
		RespondWithError(w, 400, "Validation error: url must include a scheme (http:// or https://)")
		return
	}

	if parsedUrl.Host == "" {
		RespondWithError(w, 400, "Validation error: url must include a valid hostname")
		return
	}

	if len(trimmedUrl) > 2048 {
		RespondWithError(w, 400, "Validation error: url must be 2048 characters or less")
		return
	}

	feed, err := h.DB.CreateFeed(r.Context(), database.CreateFeedParams{
		ID:        uuid.New(),
		Name:      trimmedName,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Url:       trimmedUrl,
		UserID:    user.ID,
	})

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			RespondWithError(w, 409, "Feed URL already exists - this feed has already been added")
			return
		}
		if strings.Contains(err.Error(), "foreign key violation") {
			RespondWithError(w, 400, "Invalid user: user does not exist")
			return
		}
		if strings.Contains(err.Error(), "context deadline exceeded") {
			RespondWithError(w, 408, "Request timeout: database operation took too long")
			return
		}
		RespondWithError(w, 500, "Database error: failed to create feed")
		return
	}

	RespondWithJSON(w, 201, DatabaseFeedToFeed(feed))
}

func (h *Handler) HandlerGetFeeds(w http.ResponseWriter, r *http.Request) {
	if r.Context().Err() != nil {
		RespondWithError(w, 408, "Request timeout: context cancelled or deadline exceeded")
		return
	}

	feeds, err := h.DB.GetFeeds(r.Context())
	if err != nil {
		if strings.Contains(err.Error(), "context deadline exceeded") {
			RespondWithError(w, 408, "Request timeout: database operation took too long")
			return
		}
		RespondWithError(w, 500, "Database error: failed to retrieve feeds")
		return
	}

	if feeds == nil {
		feeds = []database.Feed{}
	}

	RespondWithJSON(w, 200, DatabaseFeedsToFeeds(feeds))
}
