package middleware

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

func TestChain(t *testing.T) {
	var sequence []string

	m1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sequence = append(sequence, "m1-start")
			next.ServeHTTP(w, r)
			sequence = append(sequence, "m1-end")
		})
	}

	m2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sequence = append(sequence, "m2-start")
			next.ServeHTTP(w, r)
			sequence = append(sequence, "m2-end")
		})
	}

	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sequence = append(sequence, "final")
	})

	chained := Chain(m1, m2)(finalHandler)

	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	chained.ServeHTTP(w, req)

	expected := []string{"m1-start", "m2-start", "final", "m2-end", "m1-end"}

	if len(sequence) != len(expected) {
		t.Fatalf("expected sequence length %d, got %d", len(expected), len(sequence))
	}

	if !slices.Equal(sequence, expected) {
		t.Fatalf("expected %v, got %v", expected, sequence)
	}
}
