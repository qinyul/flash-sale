package model

type ProductStock struct {
	ProductID int64
	BasePrice float64
	Quantity  int
}

type CreateOrder struct {
	ProductID    int64
	QtyBought    int
	PricePerUnit float64
}

type CreateProductReq struct {
	Name      string  `json:"name" validate:"required"`
	BasePrice float64 `json:"base_price" validate:"min=0"`
	Quantity  int     `json:"quantity" validate:"min=0"`
}

type RestockReq struct {
	ProductID string `json:"product_id" validate:"required,uuid"` // uuid
	Quantity  int    `json:"quantity" validate:"gt=0"`
}

type PurchaseReq struct {
	ProductID string `json:"product_id" validate:"required,uuid"`
	Quantity  int    `json:"quantity" validate:"gt=0"`
}


type JSONRes struct {
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   any    `json:"error,omitempty"`
}
