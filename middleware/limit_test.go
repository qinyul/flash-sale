package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/qinyul/flash-sale/model"
)

func TestBodyLimit(t *testing.T) {

	tests := []struct {
		name          string
		limit         int64
		body          []byte
		expectedError bool
	}{
		{
			name:          "Under limit",
			limit:         10,
			body:          []byte("hello"),
			expectedError: false,
		},
		{
			name:          "Over limit",
			limit:         5,
			body:          []byte("hello world"),
			expectedError: true,
		},
		{
			name:          "Exactly at limit",
			limit:         5,
			body:          []byte("hello"),
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, err := io.ReadAll(r.Body)

				if (err != nil) != tt.expectedError {
					t.Fatalf("expected error state: %v, got error: %v", tt.expectedError, err)
				}

				w.WriteHeader(http.StatusOK)
			})

			middleware := BodyLimit(tt.limit)
			server := middleware(handler)

			req := httptest.NewRequest("POST", "/", bytes.NewReader(tt.body))
			w := httptest.NewRecorder()

			server.ServeHTTP(w, req)

			if tt.expectedError {
				if w.Code != http.StatusRequestEntityTooLarge {
					t.Errorf("expected status 413,got %d", w.Code)
				}

				var res model.JSONRes
				if err := json.NewDecoder(w.Body).Decode(&res); err != nil {
					t.Fatalf("failed to decode body %v", err)
				}

				if res.Error != "Request body too large" {
					t.Errorf("expected error message 'Request body too large' got %s", res.Error)
				}
			} else {
				if w.Code != http.StatusOK {
					t.Errorf("expected status 200, got %d", w.Code)
				}
			}
		})
	}
}
