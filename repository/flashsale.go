package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"github.com/qinyul/flash-sale/errs"
	"github.com/qinyul/flash-sale/model"
)

type FlashSaleRepository interface {
	ExecuteTx(ctx context.Context, fh func(*sql.Tx) error) error
	GetAvailableProductAndStockForUpdate(ctx context.Context, tx *sql.Tx, publicID uuid.UUID) (*model.ProductStock, error)
	DecrementStock(ctx context.Context, tx *sql.Tx, productID int64, qtyToDeduct int) error
	CreateOrder(ctx context.Context, tx *sql.Tx, order *model.CreateOrder) (uuid.UUID, error)
	CreateProductWithStock(ctx context.Context, tx *sql.Tx, req *model.CreateProductReq) (uuid.UUID, error)
	AddStock(ctx context.Context, tx *sql.Tx, req *model.RestockReq) error
}

type flashSaleRepo struct {
	db *sql.DB
}

func NewFlashSaleRepository(db *sql.DB) FlashSaleRepository {
	return &flashSaleRepo{db: db}
}

func (r *flashSaleRepo) ExecuteTx(ctx context.Context, fn func(*sql.Tx) error) (err error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	})
	if err != nil {
		slog.Error("repository::ExecuteTx failed to begin transaction", "error", err)
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			slog.Error("repository::ExecuteTx recovered from panic", "panic", p)
			tx.Rollback()
			panic(p)
		}
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				slog.Error("repository::ExecuteTx failed to rollback", "error", rbErr, "original_error", err)
			}
			return
		}
		if cmErr := tx.Commit(); cmErr != nil {
			slog.Error("repository::ExecuteTx failed to commit", "error", cmErr)
			err = cmErr
		}
	}()
	err = fn(tx)
	return
}

func (r *flashSaleRepo) GetAvailableProductAndStockForUpdate(ctx context.Context, tx *sql.Tx, publicID uuid.UUID) (*model.ProductStock, error) {
	var ps model.ProductStock

	query := `
		SELECT p.id,p.base_price,i.quantity
		FROM products p
		JOIN inventory i ON p.id = i.product_id
		WHERE p.public_id = $1 AND i.quantity > 0
		FOR UPDATE OF i SKIP LOCKED;
	`
	err := tx.QueryRowContext(ctx, query, publicID).Scan(&ps.ProductID, &ps.BasePrice, &ps.Quantity)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			slog.Debug("repository::GetAvailableProductAndStockForUpdate no stock or product not found", "public_id", publicID)
			return nil, sql.ErrNoRows
		}
		slog.Error("repository::GetAvailableProductAndStockForUpdate query failed", "error", err, "public_id", publicID)
		return nil, err
	}
	return &ps, nil
}

func (r *flashSaleRepo) DecrementStock(ctx context.Context, tx *sql.Tx, productID int64, qtyToDeduct int) error {
	query := `
		UPDATE inventory
		SET  quantity = quantity - $1
		WHERE product_id = $2
		AND quantity >= $1
	`
	res, err := tx.ExecContext(ctx, query, qtyToDeduct, productID)
	if err != nil {
		slog.Error("repository::DecrementStock update failed", "error", err, "product_id", productID, "qty", qtyToDeduct)
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		slog.Error("repository::DecrementStock fetch rows failed", "error", err, "product_id", productID)
		return err
	}

	if rows == 0 {
		slog.Warn("repository::DecrementStock no rows affected (possibly insufficient stock)", "product_id", productID, "qty", qtyToDeduct)
		return errs.ErrNotEnoughStock
	}
	return nil
}

func (r *flashSaleRepo) CreateOrder(ctx context.Context, tx *sql.Tx, order *model.CreateOrder) (uuid.UUID, error) {
	var orderUUID uuid.UUID

	// to make it atomic we calculate total amount, cast PricePerUnit
	// numeric in the query to match CHECK constraint
	query := `
		INSERT INTO orders (product_id, quantity_bought, price_per_unit, total_amount,status)
		VALUES ($1,$2::numeric,$3::numeric, ROUND(($2::numeric * $3::numeric),2),'PENDING')
		RETURNING public_id
	`
	err := tx.QueryRowContext(ctx, query, order.ProductID, order.QtyBought, order.PricePerUnit).Scan(&orderUUID)

	if err != nil {
		slog.Error("repository::CreateOrder insert failed", "error", err, "product_id", order.ProductID, "qty", order.QtyBought)
		return uuid.Nil, err
	}

	return orderUUID, nil
}

func (r *flashSaleRepo) CreateProductWithStock(ctx context.Context, tx *sql.Tx, req *model.CreateProductReq) (uuid.UUID, error) {
	var newPublicID uuid.UUID
	// atomic insert product and inventory
	query := `
		WITH new_product AS (
			INSERT INTO products (name,base_price)
			VALUES ($1,$2)
			RETURNING id, public_id
		)
		INSERT INTO inventory (product_id, quantity)
		SELECT id, $3 FROM new_product
		RETURNING (SELECT public_id FROM new_product);
	`

	err := tx.QueryRowContext(ctx, query, req.Name, req.BasePrice, req.Quantity).
		Scan(&newPublicID)

	if err != nil {
		slog.Error("repository::CreateProductWithStock failed", "error", err, "name", req.Name, "qty", req.Quantity)
		return uuid.Nil, err
	}

	return newPublicID, nil
}

func (r *flashSaleRepo) AddStock(ctx context.Context, tx *sql.Tx, req *model.RestockReq) error {
	query := `
		UPDATE inventory i
		SET quantity = quantity + $1
		FROM products p
		WHERE i.product_id = p.id
		AND p.public_id = $2;
	`

	res, err := tx.ExecContext(ctx, query, req.Quantity, req.ProductID)
	if err != nil {
		slog.Error("repository::AddStock failed", "error", err, "product_id", req.ProductID, "qty", req.Quantity)
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		slog.Error("repository::AddStock rows affected check failed", "error", err, "product_id", req.ProductID)
		return err
	}
	if rowsAffected == 0 {
		slog.Warn("repository::AddStock no product found for restock", "product_id", req.ProductID)
		return sql.ErrNoRows
	}

	return nil
}
