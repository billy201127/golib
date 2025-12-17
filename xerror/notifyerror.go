package xerror

import (
	"errors"
	"fmt"
)

func NewProviderError(provider int, err error) error {
	return &ProviderError{
		Provider: provider,
		Err:      err,
	}
}

type ProviderError struct {
	Provider int
	Err      error
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("provider=%d: %v", e.Provider, e.Err)
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}

func WrapProviderError(provider int, err error) error {
	if err == nil {
		return nil
	}
	return &ProviderError{
		Provider: provider,
		Err:      err,
	}
}

func GetProvider(err error) int {
	var pe *ProviderError
	if errors.As(err, &pe) {
		return pe.Provider
	}
	return 0
}
