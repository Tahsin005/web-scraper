package handlers

import (
	"net/http"

	"github.com/Tahsin005/web-scraper/internal/auth"
	"github.com/Tahsin005/web-scraper/internal/database"
)

func (h *Handler) AuthedHandler(handler func(http.ResponseWriter, *http.Request, database.User)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey, err := auth.GetAPIKey(r.Header)
		if err != nil {
			RespondWithError(w, 403, "Validation error: authorization header is missing or invalid")
			return
		}

		user, err := h.DB.GetUserByAPIKey(r.Context(), apiKey)
		if err != nil {
			RespondWithError(w, 403, "Validation error: invalid API key")
			return
		}

		handler(w, r, user)
	}
}
