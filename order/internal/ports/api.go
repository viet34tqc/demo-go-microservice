package ports

import "github.com/viet34tqc/demo-go-microservice/order/internal/application/core/domain"

type APIPort interface {
	PlaceOrder(order domain.Order) (domain.Order, error)
}
