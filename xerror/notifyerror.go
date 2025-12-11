package xerror

import (
	"errors"
	"fmt"
)

func NewNotifyError(provider int, err error) error {
	return &NotifyError{
		Provider: provider,
		ErrMsg:   err.Error(),
	}
}

type NotifyError struct {
	Provider int
	ErrMsg   string
}

func (e *NotifyError) Error() string {
	return fmt.Sprintf("provider: %d, err_msg: %s", e.Provider, e.ErrMsg)
}

func (e *NotifyError) Wrap(err error) error {
	e.ErrMsg = err.Error()
	return e
}

func GetNotifyProvider(err error) int {
	var ne *NotifyError
	if errors.As(err, &ne) {
		return ne.Provider
	}
	return 0
}
