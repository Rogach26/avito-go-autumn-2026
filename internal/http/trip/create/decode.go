package create

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"

	api "github.com/Rogach26/avito-go-autumn-2026/internal/generated"
)

const maxRequestBodyBytes = 1 << 20

func decodeTripData(w http.ResponseWriter, r *http.Request) (api.TripData, error) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return api.TripData{}, fmt.Errorf("content type must be application/json")
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return api.TripData{}, fmt.Errorf("read request body: %w", err)
	}
	if err := validateRequiredJSONFields(body); err != nil {
		return api.TripData{}, err
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var request api.TripData
	if err := decoder.Decode(&request); err != nil {
		return api.TripData{}, fmt.Errorf("decode request body: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return api.TripData{}, fmt.Errorf("request body must contain one JSON object")
	}
	return request, nil
}

func validateRequiredJSONFields(body []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return fmt.Errorf("decode required fields: %w", err)
	}
	for _, name := range []string{"user_id", "driver_id", "start_point", "end_point", "price"} {
		value, ok := fields[name]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("field %s is required", name)
		}
	}
	for _, name := range []string{"start_point", "end_point"} {
		var point map[string]json.RawMessage
		if err := json.Unmarshal(fields[name], &point); err != nil {
			return fmt.Errorf("decode %s: %w", name, err)
		}
		for _, coordinate := range []string{"latitude", "longitude"} {
			value, ok := point[coordinate]
			if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
				return fmt.Errorf("field %s.%s is required", name, coordinate)
			}
		}
	}
	return nil
}
