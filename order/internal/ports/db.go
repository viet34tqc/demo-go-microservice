package ports

import (
	"github.com/viet34tqc/demo-go-microservice/order/internal/application/core/domain"
)

type DBPort interface {
	Get(id string) (domain.Order, error)
	Save(*domain.Order) error
}
