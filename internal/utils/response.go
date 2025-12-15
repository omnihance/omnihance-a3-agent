package utils

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func WriteJSONResponseWithStatus(w http.ResponseWriter, status int, data interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

func WriteJSONResponse(w http.ResponseWriter, data interface{}) error {
	return WriteJSONResponseWithStatus(w, http.StatusOK, data)
}

func WriteJSONResponseCached(w http.ResponseWriter, data interface{}, cacheDuration int) error {
	return WriteJSONResponseCachedWithStatus(w, http.StatusOK, data, cacheDuration)
}

func WriteJSONResponseCachedWithStatus(w http.ResponseWriter, status int, data interface{}, cacheDuration int) error {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(cacheDuration)+", s-maxage="+strconv.Itoa(cacheDuration))
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}
