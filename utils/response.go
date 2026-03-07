package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/qinyul/flash-sale/model"
)

// JSON writes a successful payload to the response writer
func JSON(w http.ResponseWriter, status int, payload model.JSONRes) {
	w.Header().Set("Content-Type", "application/json")

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(payload); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(status)
	w.Write(buf.Bytes())
}

// Error is a semantic helper for quickly returning error payloads
func Error(w http.ResponseWriter, status int, msg any) {
	JSON(w, status, model.JSONRes{Error: msg})
}

// ValidationErrors is a helper to return structured validation errors
func ValidationErrors(w http.ResponseWriter, errs validator.ValidationErrors) {
	errorDetails := make(map[string]string)
	for _, e := range errs {
		errorDetails[e.Field()] = fmt.Sprintf("failed validation on '%s' tag", e.Tag())
	}

	JSON(w, http.StatusBadRequest, model.JSONRes{
		Error: "Payload validation failed",
		Data:  errorDetails,
	})
}

// Decode is a helper to decode JSON request body and handle error response
func Decode[T any](r *http.Request) (T, error) {
	var payload T

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return payload, err
	}

	return payload, nil
}
