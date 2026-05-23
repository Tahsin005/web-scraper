package handlers

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/Tahsin005/web-scraper/internal/database"
	"github.com/google/uuid"
)

func (h *Handler) HandlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Name string `json:"name"`
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

	if r.Context().Err() != nil {
		RespondWithError(w, 408, "Request timeout: context cancelled or deadline exceeded")
		return
	}

	user, err := h.DB.CreateUser(r.Context(), database.CreateUserParams{
		ID:        uuid.New(),
		Name:      trimmedName,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	})

	if err != nil {
		if strings.Contains(err.Error(), "context deadline exceeded") {
			RespondWithError(w, 408, "Request timeout: database operation took too long")
			return
		}
		RespondWithError(w, 500, "Database error: failed to create user")
		return
	}

	RespondWithJSON(w, 201, DatabaseUserToUser(user))
}

func (h *Handler) HandlerGetUserByAPIKey(w http.ResponseWriter, r *http.Request, user database.User) {
	if r.Context().Err() != nil {
		RespondWithError(w, 408, "Request timeout: context cancelled or deadline exceeded")
		return
	}

	RespondWithJSON(w, 200, DatabaseUserToUser(user))
}

func (h *Handler) HandlerGetPostsForUser(w http.ResponseWriter, r *http.Request, user database.User) {
	if r.Context().Err() != nil {
		RespondWithError(w, 408, "Request timeout: context cancelled or deadline exceeded")
		return
	}

	posts, err := h.DB.GetPostsForUser(r.Context(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  100,
	})
	if err != nil {
		if strings.Contains(err.Error(), "context deadline exceeded") {
			RespondWithError(w, 408, "Request timeout: database operation took too long")
			return
		}
		RespondWithError(w, 500, "Database error: failed to retrieve posts")
		return
	}

	if posts == nil {
		posts = []database.Post{}
	}

	RespondWithJSON(w, 200, DatabasePostsToPostss(posts))
}
