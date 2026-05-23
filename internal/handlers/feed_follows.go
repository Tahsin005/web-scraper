package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/Tahsin005/web-scraper/internal/database"
	"github.com/go-chi/chi"
	"github.com/google/uuid"
)

func (h *Handler) HandlerCreateFeedFollows(w http.ResponseWriter, r *http.Request, user database.User) {
	type parameters struct {
		FeedID uuid.UUID `json:"feed_id"`
	}

	params := parameters{}
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		RespondWithError(w, 400, "Invalid JSON: unable to parse request body")
		return
	}

	if params.FeedID == uuid.Nil {
		RespondWithError(w, 400, "Validation error: feed_id is required and must be a valid UUID")
		return
	}

	feedFollow, err := h.DB.CreateFeedFollow(r.Context(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		FeedID:    params.FeedID,
		UserID:    user.ID,
	})

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			RespondWithError(w, 409, "You are already following this feed")
			return
		}
		if strings.Contains(err.Error(), "foreign key violation") {
			RespondWithError(w, 404, "Feed not found: the specified feed does not exist")
			return
		}
		if strings.Contains(err.Error(), "context deadline exceeded") {
			RespondWithError(w, 408, "Request timeout: database operation took too long")
			return
		}
		RespondWithError(w, 500, "Database error: failed to create feed follow")
		return
	}

	RespondWithJSON(w, 201, DatabaseFeedFollowToFeedFollow(feedFollow))
}

func (h *Handler) HandlerGetFeedFollows(w http.ResponseWriter, r *http.Request, user database.User) {
	if r.Context().Err() != nil {
		RespondWithError(w, 408, "Request timeout: context cancelled or deadline exceeded")
		return
	}

	feedFollows, err := h.DB.GetFeedFollows(r.Context(), user.ID)
	if err != nil {
		if strings.Contains(err.Error(), "context deadline exceeded") {
			RespondWithError(w, 408, "Request timeout: database operation took too long")
			return
		}
		RespondWithError(w, 500, "Database error: failed to retrieve feed follows")
		return
	}

	if feedFollows == nil {
		feedFollows = []database.FeedFollow{}
	}

	RespondWithJSON(w, 200, DatabaseFeedFollowsToFeedFollows(feedFollows))
}

func (h *Handler) HandlerDeleteFeedFollow(w http.ResponseWriter, r *http.Request, user database.User) {
	feedFollowIDStr := chi.URLParam(r, "feedFollowID")

	if feedFollowIDStr == "" {
		RespondWithError(w, 400, "Validation error: feedFollowID URL parameter is required")
		return
	}

	feedFollowID, err := uuid.Parse(feedFollowIDStr)
	if err != nil {
		RespondWithError(w, 400, "Validation error: feedFollowID must be a valid UUID")
		return
	}

	if feedFollowID == uuid.Nil {
		RespondWithError(w, 400, "Validation error: feedFollowID cannot be empty")
		return
	}

	if r.Context().Err() != nil {
		RespondWithError(w, 408, "Request timeout: context cancelled or deadline exceeded")
		return
	}

	err = h.DB.DeleteFeedFollow(r.Context(), database.DeleteFeedFollowParams{
		ID:     feedFollowID,
		UserID: user.ID,
	})
	if err != nil {
		if strings.Contains(err.Error(), "no rows in result set") {
			RespondWithError(w, 404, "Feed follow not found: you are not following this feed or it has already been deleted")
			return
		}
		if strings.Contains(err.Error(), "context deadline exceeded") {
			RespondWithError(w, 408, "Request timeout: database operation took too long")
			return
		}
		RespondWithError(w, 500, "Database error: failed to delete feed follow")
		return
	}

	RespondWithJSON(w, 200, struct{}{})
}
