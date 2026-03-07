package service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/qinyul/flash-sale/errs"
	"github.com/qinyul/flash-sale/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock Reposistory
type mockFlashSaleRepository struct {
	mock.Mock
}

func (m *mockFlashSaleRepository) ExecuteTx(ctx context.Context, fn func(*sql.Tx) error) error {
	args := m.Called(ctx, mock.Anything)
	err := fn(nil)

	if err != nil {
		return err
	}

	return args.Error(0)
}

func (m *mockFlashSaleRepository) GetAvailableProductAndStockForUpdate(
	ctx context.Context,
	tx *sql.Tx,
	publicID uuid.UUID,
) (*model.ProductStock, error) {

	args := m.Called(ctx, tx, publicID)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*model.ProductStock), args.Error(1)
}

func (m *mockFlashSaleRepository) DecrementStock(
	ctx context.Context,
	tx *sql.Tx,
	productID int64,
	qty int,
) error {
	args := m.Called(ctx, tx, productID, qty)
	return args.Error(0)
}

func (m *mockFlashSaleRepository) CreateOrder(
	ctx context.Context,
	tx *sql.Tx,
	order *model.CreateOrder,
) (uuid.UUID, error) {
	args := m.Called(ctx, tx, order)

	if args.Get(0) == nil {
		return uuid.Nil, args.Error(1)
	}
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *mockFlashSaleRepository) CreateProductWithStock(
	ctx context.Context,
	tx *sql.Tx,
	product *model.CreateProductReq,
) (uuid.UUID, error) {
	args := m.Called(ctx, tx, product)

	if args.Get(0) == nil {
		return uuid.Nil, args.Error(1)
	}
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func (m *mockFlashSaleRepository) AddStock(
	ctx context.Context,
	tx *sql.Tx,
	product *model.RestockReq,
) error {
	args := m.Called(ctx, tx, product)
	return args.Error(0)
}

// End Mock Repository

// Happy Path
func TestProcessPurchase_Success(t *testing.T) {
	mockRepo := new(mockFlashSaleRepository)
	service := NewFlashSaleService(mockRepo)

	productUUID := uuid.New()
	orderUUID := uuid.New()

	mockRepo.On("ExecuteTx", mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("GetAvailableProductAndStockForUpdate", mock.Anything, mock.Anything, productUUID).Return(&model.ProductStock{
		ProductID: 1,
		BasePrice: 100,
		Quantity:  10,
	}, nil)

	mockRepo.On("DecrementStock", mock.Anything, mock.Anything, int64(1), 2).Return(nil)

	mockRepo.On("CreateOrder", mock.Anything, mock.Anything, mock.Anything).Return(orderUUID, nil)

	result, err := service.ProcessPurchase(context.Background(), productUUID, 2)

	assert.NoError(t, err)
	assert.Equal(t, orderUUID, result)
}

// High Traffic (SKIP LOCKED → sql.ErrNoRows) Path
func TestProcessPurchase_HighTraffic(t *testing.T) {
	mockRepo := new(mockFlashSaleRepository)
	service := NewFlashSaleService(mockRepo)

	productUUID := uuid.New()

	mockRepo.On("ExecuteTx", mock.Anything, mock.Anything).Return(nil)

	mockRepo.On("GetAvailableProductAndStockForUpdate", mock.Anything, mock.Anything, productUUID).Return(nil, sql.ErrNoRows)

	result, err := service.ProcessPurchase(context.Background(), productUUID, 1)

	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, result)
	assert.Equal(t, errs.ErrNotEnoughStock, err)
}

// Sold out path
func TestProcessPurchase_SoldOut(t *testing.T) {
	mockRepo := new(mockFlashSaleRepository)
	service := NewFlashSaleService(mockRepo)

	productUUID := uuid.New()

	mockRepo.On("ExecuteTx", mock.Anything, mock.Anything).Return(nil)

	mockRepo.On("GetAvailableProductAndStockForUpdate", mock.Anything, mock.Anything, productUUID).Return(&model.ProductStock{
		ProductID: 1,
		BasePrice: 100,
		Quantity:  0,
	}, nil)

	result, err := service.ProcessPurchase(context.Background(), productUUID, 1)

	assert.Error(t, err)
	assert.Equal(t, errs.ErrSoldOut, err)
	assert.Equal(t, uuid.Nil, result)
}

// Not enough stock path
func TestProcessPurchase_NotEnoughStock(t *testing.T) {
	mockRepo := new(mockFlashSaleRepository)
	service := NewFlashSaleService(mockRepo)

	productUUID := uuid.New()

	mockRepo.On("ExecuteTx", mock.Anything, mock.Anything).Return(nil)

	mockRepo.On("GetAvailableProductAndStockForUpdate", mock.Anything, mock.Anything, productUUID).Return(&model.ProductStock{
		ProductID: 1,
		BasePrice: 100,
		Quantity:  1,
	}, nil)

	result, err := service.ProcessPurchase(context.Background(), productUUID, 5)

	assert.Error(t, err)
	assert.Equal(t, errs.ErrNotEnoughStock, err)
	assert.Equal(t, uuid.Nil, result)
}

// decrement fails path
func TestProcessPurchase_DecrementFails(t *testing.T) {
	mockRepo := new(mockFlashSaleRepository)
	service := NewFlashSaleService(mockRepo)
	requestedQty := 5
	productUUID := uuid.New()

	mockRepo.On("ExecuteTx", mock.Anything, mock.Anything).Return(nil)

	mockRepo.On("GetAvailableProductAndStockForUpdate", mock.Anything, mock.Anything, productUUID).Return(&model.ProductStock{
		ProductID: 1,
		BasePrice: 100,
		Quantity:  10,
	}, nil)

	mockRepo.
		On("DecrementStock", mock.Anything, mock.Anything, int64(1), requestedQty).
		Return(errors.New("db constraint"))

	result, err := service.ProcessPurchase(context.Background(), productUUID, requestedQty)

	assert.Error(t, err)
	assert.Equal(t, errs.ErrHighTraffic, err)
	assert.Equal(t, uuid.Nil, result)
}

// Create order fails

func TestProcessPurchase_CreateOrderFails(t *testing.T) {
	mockRepo := new(mockFlashSaleRepository)
	service := NewFlashSaleService(mockRepo)

	productUUID := uuid.New()

	mockRepo.On("ExecuteTx", mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("GetAvailableProductAndStockForUpdate", mock.Anything, mock.Anything, productUUID).Return(&model.ProductStock{
		ProductID: 1,
		BasePrice: 100,
		Quantity:  10,
	}, nil)

	mockRepo.On("DecrementStock", mock.Anything, mock.Anything, int64(1), 2).Return(nil)

	mockRepo.On("CreateOrder", mock.Anything, mock.Anything, mock.Anything).Return(uuid.Nil, errors.New("insert fail"))

	result, err := service.ProcessPurchase(context.Background(), productUUID, 2)

	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, result)
}

// success create product
func TestCreateNewProduct_Success(t *testing.T) {
	mockRepo := new(mockFlashSaleRepository)
	service := NewFlashSaleService(mockRepo)
	req := &model.CreateProductReq{Name: "PS5"}

	expectedUUID := uuid.New()

	mockRepo.On("ExecuteTx", mock.Anything, mock.Anything).Return(nil)
	mockRepo.On("CreateProductWithStock", mock.Anything, mock.Anything, req).Return(expectedUUID, nil)

	result, err := service.CreateNewProduct(context.Background(), req)

	assert.NoError(t, err)
	assert.Equal(t, expectedUUID, result)

	mockRepo.AssertExpectations(t)
}

// failed create product
func TestCreateNewProduct_CreateFails(t *testing.T) {
	mockRepo := new(mockFlashSaleRepository)
	service := NewFlashSaleService(mockRepo)
	req := &model.CreateProductReq{Name: "PS5"}

	expectedErr := errors.New("db error")

	mockRepo.On("ExecuteTx", mock.Anything, mock.Anything).Return(expectedErr)
	mockRepo.On("CreateProductWithStock", mock.Anything, mock.Anything, req).Return(uuid.Nil, expectedErr)

	result, err := service.CreateNewProduct(context.Background(), req)

	assert.Error(t, err)
	assert.Equal(t, uuid.Nil, result)
	assert.Contains(t, err.Error(), "create product with stock failed")

	mockRepo.AssertExpectations(t)
}
