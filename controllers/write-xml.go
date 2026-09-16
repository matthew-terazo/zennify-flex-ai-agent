package controllers

import (
	"net/http"
	"strconv"
)

// writeJSONResponse writes a JSON response with the given status code and data.
// For 204 No Content, no body is written (as per HTTP spec).
// For other statuses, data is written as-is (caller should provide marshaled JSON).
func writeXMLResponse(w http.ResponseWriter, status int, data string) {
	w.Header().Set("Content-Type", "application/xml")

	// 204 No Content must not have a response body
	if status == http.StatusNoContent {
		w.WriteHeader(status)
		return
	}

	// For other statuses, write the XML data as-is
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.WriteHeader(status)
	w.Write([]byte(data))
}
