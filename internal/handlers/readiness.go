package handlers

import (
	"net/http"
)

func (h *Handler) HandlerReadiness(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Status string `json:"status"`
	}
	RespondWithJSON(w, 200, response{
		Status: "ok",
	})
}
