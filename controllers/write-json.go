package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"
)

// writeJSONResponse writes a JSON response with the given status code and data.
// For 204 No Content, no body is written (as per HTTP spec).
// For other statuses, data is written as-is (caller should provide marshaled JSON).
func writeJSONResponse(w http.ResponseWriter, status int, response any) error {
	w.Header().Set("Content-Type", "application/json")

	// 204 No Content must not have a response body
	switch status {
	case http.StatusNoContent:
		w.WriteHeader(status)
		return nil
	}

	data, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}

	// For other statuses, write the JSON data as-is
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(status)
	w.Write(data)

	return nil
}
