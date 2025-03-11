package xerror

import "net/http"

const (
	CodeInternalError    = http.StatusInternalServerError
	CodeUnableConnect    = http.StatusServiceUnavailable
	CodeForbidden        = http.StatusForbidden
	CodeUnauthorized     = http.StatusUnauthorized
	CodeDisabled         = http.StatusGone
	CodeInvalidParams    = http.StatusBadRequest
	CodeConvertFailed    = http.StatusUnprocessableEntity
	CodeDataNotExist     = http.StatusNotFound
	CodeDataAlreadyExist = http.StatusConflict
	CodeOperateTooFast   = http.StatusTooManyRequests
	CodeCallFailed       = http.StatusBadGateway
)

var ErrMsgs = map[int]string{
	CodeInternalError:    "service internal error",
	CodeInvalidParams:    "invalid request params",
	CodeOperateTooFast:   "operation too fast",
	CodeDataNotExist:     "data not exist",
	CodeDataAlreadyExist: "data already exists",
}
