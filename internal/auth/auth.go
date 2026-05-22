package auth

import (
	"errors"
	"net/http"
	"strings"
)

// Authorization: ApiKey <api_key>
func GetAPIKey(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("no authentication info found")
	}

	vals := strings.Split(authHeader, " ")
	if len(vals) != 2 || vals[0] != "ApiKey" {
		return "", errors.New("invalid authentication header")
	}

	apiKey := vals[1]
	return apiKey, nil
}