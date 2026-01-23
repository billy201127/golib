package kmscred

import (
	"errors"
	"fmt"
)

type Client interface {
	GetSecretValue(secretName string) (string, error)
}

type Factory func(cfg Config) (Client, error)

var registry = map[Vendor]Factory{}

func Register(v Vendor, f Factory) {
	if f == nil {
		panic("kmscred: Register factory is nil")
	}
	if _, ok := registry[v]; ok {
		panic(fmt.Sprintf("kmscred: Register called twice for vendor %q", v))
	}
	registry[v] = f
}

func New(cfg Config) (Client, error) {
	if cfg.Vendor == "" {
		return nil, errors.New("kmscred: vendor is required")
	}
	if cfg.Mode == "" {
		return nil, errors.New("kmscred: mode is required")
	}
	f, ok := registry[cfg.Vendor]
	if !ok {
		return nil, fmt.Errorf("kmscred: unsupported vendor %q", cfg.Vendor)
	}
	return f(cfg)
}
