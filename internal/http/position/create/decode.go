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

func decodePositionData(w http.ResponseWriter, r *http.Request) (api.PositionData, error) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return api.PositionData{}, fmt.Errorf("content type must be application/json")
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return api.PositionData{}, fmt.Errorf("read request body: %w", err)
	}
	if err := validateRequiredJSONFields(body); err != nil {
		return api.PositionData{}, err
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var request api.PositionData
	if err := decoder.Decode(&request); err != nil {
		return api.PositionData{}, fmt.Errorf("decode request body: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return api.PositionData{}, fmt.Errorf("request body must contain one JSON object")
	}
	return request, nil
}

func validateRequiredJSONFields(body []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return fmt.Errorf("decode required fields: %w", err)
	}
	for _, name := range []string{"coordinates", "recorded_at"} {
		value, ok := fields[name]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("field %s is required", name)
		}
	}

	var coordinates map[string]json.RawMessage
	if err := json.Unmarshal(fields["coordinates"], &coordinates); err != nil {
		return fmt.Errorf("decode coordinates: %w", err)
	}
	for _, name := range []string{"latitude", "longitude"} {
		value, ok := coordinates[name]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return fmt.Errorf("field coordinates.%s is required", name)
		}
	}
	return nil
}
