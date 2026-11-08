package response

import (
	"encoding/json"
	"net/http"
)

func JSON(w http.ResponseWriter, status int, contentType string, value any) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
