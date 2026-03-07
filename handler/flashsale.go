package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/qinyul/flash-sale/errs"
	"github.com/qinyul/flash-sale/middleware"
	"github.com/qinyul/flash-sale/model"
	"github.com/qinyul/flash-sale/service"
	"github.com/qinyul/flash-sale/utils"
)

type FlashSaleHandler struct {
	svc              service.FlashSaleService
	limitRequestBody int64
	validate         *validator.Validate
}

func NewFlashSaleHandler(svc service.FlashSaleService, limitRequestBody int64) *FlashSaleHandler {
	v := validator.New()
	v.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := strings.SplitN(field.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	return &FlashSaleHandler{
		svc:              svc,
		limitRequestBody: limitRequestBody,
		validate:         v,
	}
}

func (h *FlashSaleHandler) RegisterRoutes(mux *http.ServeMux) {
	standarChain := middleware.Chain(middleware.BodyLimit(h.limitRequestBody))

	mux.Handle("POST /products", standarChain(http.HandlerFunc(h.CreateProduct)))
	mux.Handle("POST /products/restock", standarChain(http.HandlerFunc(h.Restock)))
	mux.Handle("POST /orders", standarChain(http.HandlerFunc(h.Purchase)))
}

func (h *FlashSaleHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	req, err := utils.Decode[model.CreateProductReq](r)
	if err != nil {
		slog.Error("CreateProduct:: invalid payload")
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {

		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			utils.ValidationErrors(w, validationErrors)
			return
		}
		utils.Error(w, http.StatusBadRequest, "invalid request")
		return
	}
	productUUID, err := h.svc.CreateNewProduct(r.Context(), &req)
	if err != nil {
		slog.Error("CreateProduct Handler:: error", "err", err)
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.JSON(w, http.StatusCreated, model.JSONRes{
		Message: "Product created successfully",
		Data:    map[string]string{"product_id": productUUID.String()},
	})
}

func (h *FlashSaleHandler) Restock(w http.ResponseWriter, r *http.Request) {
	req, err := utils.Decode[model.RestockReq](r)
	if err != nil {
		slog.Error("Restock:: invalid payload")
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			utils.ValidationErrors(w, validationErrors)
			return
		}
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.svc.RestockProduct(r.Context(), &req); err != nil {
		slog.Error("Restock Handler:: error", "err", err)
		if errors.Is(err, errs.ErrProductNotFound) {
			utils.Error(w, http.StatusNotFound, err.Error())
			return
		}
		utils.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	utils.JSON(w, http.StatusOK, model.JSONRes{
		Message: "product restocked succesfully",
	})
}

func (h *FlashSaleHandler) Purchase(w http.ResponseWriter, r *http.Request) {
	req, err := utils.Decode[model.PurchaseReq](r)
	if err != nil {
		slog.Error("Purchase:: invalid payload")
		utils.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validate.Struct(req); err != nil {
		var validationErrors validator.ValidationErrors
		if errors.As(err, &validationErrors) {
			utils.ValidationErrors(w, validationErrors)
			return
		}
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	productUUID, err := uuid.Parse(req.ProductID)
	if err != nil {
		utils.Error(w, http.StatusBadRequest, "invalid product id format")
		return
	}

	orderUUID, err := h.svc.ProcessPurchase(r.Context(), productUUID, req.Quantity)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, errs.ErrSoldOut), errors.Is(err, errs.ErrNotEnoughStock):
			status = http.StatusBadRequest
		case errors.Is(err, errs.ErrHighTraffic):
			status = http.StatusTooManyRequests
		}
		utils.Error(w, status, err.Error())
		return
	}

	utils.JSON(w, http.StatusOK, model.JSONRes{
		Message: "purchase successful",
		Data:    map[string]string{"order_id": orderUUID.String()},
	})
}
