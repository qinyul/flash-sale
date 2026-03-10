//go:build integration

package tests

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/qinyul/flash-sale/config"
	"github.com/qinyul/flash-sale/handler"
	"github.com/qinyul/flash-sale/infrastructure"
	"github.com/qinyul/flash-sale/model"
	"github.com/qinyul/flash-sale/repository"
	"github.com/qinyul/flash-sale/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testDB      *sql.DB
	testHandler *handler.FlashSaleHandler
)

func TestMain(m *testing.M) {
	cfg, err := config.LoadConfig(os.LookupEnv)
	if err != nil {
		log.Fatalf("FATAL: config error: %v", err)
	}

	db, err := infrastructure.NewPostgresDB(cfg.Database)
	if err != nil {
		panic(err)
	}
	testDB = db.DB

	repo := repository.NewFlashSaleRepository(testDB)
	svc := service.NewFlashSaleService(repo)
	testHandler = handler.NewFlashSaleHandler(svc, cfg.App.MaxBodyBytes)

	code := m.Run()

	testDB.Close()
	os.Exit(code)
}

func setupTestDB(t *testing.T) {
	_, err := testDB.Exec("TRUNCATE TABLE orders, inventory, products RESTART IDENTITY CASCADE")
	require.NoError(t, err)
}

func TestProductIntegration(t *testing.T) {
	setupTestDB(t)

	mux := http.NewServeMux()
	testHandler.RegisterRoutes(mux)

	var productID string

	t.Run("Create Product Successfully", func(t *testing.T) {
		reqBody := model.CreateProductReq{
			Name:      "Test Product",
			BasePrice: 100.0,
			Quantity:  10,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewReader(body))
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)
		assert.Equal(t, http.StatusCreated, w.Code)

		var res model.JSONRes
		err := json.Unmarshal(w.Body.Bytes(), &res)
		assert.NoError(t, err)
		assert.Equal(t, "Product created successfully", res.Message)

		data := res.Data.(map[string]interface{})
		productID = data["product_id"].(string)
		_, err = uuid.Parse(productID)
		assert.NoError(t, err)
	})

	t.Run("Restock Product Successfully", func(t *testing.T) {
		require.NotEmpty(t, productID)

		reqBody := model.RestockReq{
			ProductID: productID,
			Quantity:  5,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/products/restock", bytes.NewReader(body))
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		var res model.JSONRes
		err := json.Unmarshal(w.Body.Bytes(), &res)
		assert.NoError(t, err)
		assert.Equal(t, "product restocked succesfully", res.Message)
	})

	t.Run("Purchase More Than Availabvle Stock", func(t *testing.T) {
		require.NotEmpty(t, productID)

		// Remaining stock should be 10 (initial) + 5 (restock) - 2 (purchase) = 13
		reqBody := model.PurchaseReq{
			ProductID: productID,
			Quantity:  2,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var res model.JSONRes
		err := json.Unmarshal(w.Body.Bytes(), &res)
		assert.NoError(t, err)
		assert.Equal(t, "purchase successful", res.Message)
	})

	t.Run("Purchase More Than Availabvle Stock", func(t *testing.T) {
		require.NotEmpty(t, productID)

		// Remaining stock should be 10 (initial) + 5 (restock) - 2 (purchase) = 13
		reqBody := model.PurchaseReq{
			ProductID: productID,
			Quantity:  20,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var res model.JSONRes
		err := json.Unmarshal(w.Body.Bytes(), &res)
		assert.NoError(t, err)
		assert.Contains(t, res.Error, "not enough stock")
	})

	t.Run("Purchase with Invalid Product ID", func(t *testing.T) {
		require.NotEmpty(t, productID)

		reqBody := model.PurchaseReq{
			ProductID: uuid.New().String(),
			Quantity:  1,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/orders", bytes.NewBuffer(body))
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var res model.JSONRes
		err := json.Unmarshal(w.Body.Bytes(), &res)
		assert.NoError(t, err)
		assert.Contains(t, res.Error, "not enough stock")
	})
}
