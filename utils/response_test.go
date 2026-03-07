package utils

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/qinyul/flash-sale/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSON(t *testing.T) {
	tests := []struct {
		name         string
		status       int
		payload      model.JSONRes
		expectedData any
	}{
		{
			name:   "Success with data",
			status: http.StatusOK,
			payload: model.JSONRes{
				Message: "Success",
				Data:    map[string]string{"foo": "bar"},
			},
			expectedData: map[string]any{"foo": "bar"},
		},
		{
			name:   "Success with no data",
			status: http.StatusCreated,
			payload: model.JSONRes{
				Message: "Resource Created",
			},
			expectedData: nil,
		},
		{
			name:   "Internal Error with message",
			status: http.StatusInternalServerError,
			payload: model.JSONRes{
				Error: "something went wrong",
			},
			expectedData: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			JSON(w, tt.status, tt.payload)

			assert.Equal(t, tt.status, w.Code)
			assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

			var res model.JSONRes
			err := json.NewDecoder(w.Body).Decode(&res)
			require.NoError(t, err)

			assert.Equal(t, tt.payload.Message, res.Message)
			assert.Equal(t, tt.payload.Error, res.Error)

			if tt.expectedData != nil {
				assert.Equal(t, tt.expectedData, res.Data)
			}
		})
	}
}

func TestValidationErrors(t *testing.T) {
	t.Run("Validation Error Helper", func(t *testing.T) {
		validate := validator.New()
		type TestReq struct {
			Name string `json:"name" validate:"required"`
			Age  int    `json:"age" validate:"min=18"`
		}
		validate.RegisterTagNameFunc(func(field reflect.StructField) string {
			name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			return name
		})

		req := TestReq{Name: "", Age: 10}
		err := validate.Struct(req)
		require.Error(t, err)

		validationErrors := err.(validator.ValidationErrors)

		w := httptest.NewRecorder()
		ValidationErrors(w, validationErrors)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var res model.JSONRes
		err = json.NewDecoder(w.Body).Decode(&res)
		require.NoError(t, err)

		assert.Equal(t, "Payload validation failed", res.Error)
		assert.NotNil(t, res.Data)

		details, ok := res.Data.(map[string]any)
		require.True(t, ok)
		assert.Contains(t, details, "name")
		assert.Contains(t, details, "age")
	})
}

func TestError(t *testing.T) {
	t.Run("Standard Error Helper", func(t *testing.T) {
		w := httptest.NewRecorder()
		errMsg := "unauthorzied access"
		status := http.StatusUnauthorized

		Error(w, status, errMsg)

		assert.Equal(t, status, w.Code)
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		var res model.JSONRes
		err := json.NewDecoder(w.Body).Decode(&res)
		require.NoError(t, err)

		assert.Equal(t, errMsg, res.Error)
		assert.Empty(t, res.Message)
		assert.Nil(t, res.Data)
	})
}
