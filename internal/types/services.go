// /services/internal/types/service.go
package types

// Service interface that all services must implement.
type Service interface {
	Name() string
	Initialize() error
	Start() error
	Stop() error
}
