package errs

import "errors"

var (
	ErrProductNotFound = errors.New("product not found")
	ErrSoldOut         = errors.New("sold out")
	ErrNotEnoughStock  = errors.New("not enough stock available")
	ErrHighTraffic     = errors.New("order failed due to high traffic")
)
