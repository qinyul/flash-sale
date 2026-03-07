package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/qinyul/flash-sale/errs"
	"github.com/qinyul/flash-sale/model"
	"github.com/qinyul/flash-sale/repository"
)

type FlashSaleService interface {
	ProcessPurchase(ctx context.Context, productUUID uuid.UUID, requestedQty int) (uuid.UUID, error)
	CreateNewProduct(ctx context.Context, req *model.CreateProductReq) (uuid.UUID, error)
	RestockProduct(ctx context.Context, req *model.RestockReq) error
}

type flashSaleService struct {
	repo repository.FlashSaleRepository
}

func NewFlashSaleService(repo repository.FlashSaleRepository) FlashSaleService {
	return &flashSaleService{repo: repo}
}

func (s *flashSaleService) ProcessPurchase(ctx context.Context, productUUID uuid.UUID, requestedQty int) (uuid.UUID, error) {
	var finalOrderUUID uuid.UUID

	err := s.repo.ExecuteTx(ctx, func(tx *sql.Tx) error {

		// read with EXCLUSIVE LOCK
		stockData, err := s.repo.GetAvailableProductAndStockForUpdate(ctx, tx, productUUID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return errs.ErrNotEnoughStock
			}
			return err
		}

		// already handle the quantity check in db, this for extra check
		if stockData.Quantity <= 0 {
			return errs.ErrSoldOut
		}

		if stockData.Quantity < requestedQty {
			return errs.ErrNotEnoughStock
		}

		if err := s.repo.DecrementStock(ctx, tx, stockData.ProductID, requestedQty); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return errs.ErrNotEnoughStock
			}
			slog.Error("DB Constraint Error (Inventory)", "err", err)
			return errs.ErrHighTraffic
		}

		order := model.CreateOrder{
			ProductID:    stockData.ProductID,
			QtyBought:    requestedQty,
			PricePerUnit: stockData.BasePrice,
		}
		orderUUID, err := s.repo.CreateOrder(ctx, tx, &order)
		if err != nil {
			slog.Info("failed to create order snapshot", "Err", err)
			return fmt.Errorf("Failed to create order: %w", err)
		}
		finalOrderUUID = orderUUID
		return nil
	})

	if err != nil {
		return uuid.Nil, err
	}

	return finalOrderUUID, nil
}

func (s *flashSaleService) CreateNewProduct(ctx context.Context, req *model.CreateProductReq) (uuid.UUID, error) {
	var newProductUUID uuid.UUID

	err := s.repo.ExecuteTx(ctx, func(tx *sql.Tx) error {
		productUUID, err := s.repo.CreateProductWithStock(ctx, tx, req)
		if err != nil {
			return err
		}

		newProductUUID = productUUID
		return nil
	})

	if err != nil {
		return uuid.Nil, fmt.Errorf("create product with stock failed with: %w", err)
	}
	return newProductUUID, nil
}

func (s *flashSaleService) RestockProduct(ctx context.Context, req *model.RestockReq) error {
	return s.repo.ExecuteTx(ctx, func(tx *sql.Tx) error {
		err := s.repo.AddStock(ctx, tx, req)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return errs.ErrProductNotFound
			}
			return fmt.Errorf("add stock failed: %w", err)
		}
		return nil
	})
}
