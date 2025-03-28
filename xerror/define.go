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
	CodeDataNotFound     = 4004
)

var ErrMsgs = map[int]string{
	// 4xx Client Errors
	CodeInvalidParams:    "Bad Request - Invalid request parameters or syntax",
	CodeUnauthorized:     "Unauthorized - Authentication required",
	CodeForbidden:        "Forbidden - Access denied",
	CodeDataNotExist:     "Not Found - The requested resource does not exist",
	CodeDataAlreadyExist: "Conflict - Resource state conflict",
	CodeOperateTooFast:   "Too Many Requests - Request rate limit exceeded",
	CodeConvertFailed:    "Unprocessable Entity - Convert failed",

	// 5xx Server Errors
	CodeInternalError: "Internal Server Error - Something went wrong",
	CodeCallFailed:    "Bad Gateway - Invalid response from upstream server",
	CodeUnableConnect: "Service Unavailable - Server temporarily unavailable",
	CodeDisabled:      "Gone - The requested resource is no longer available",
	CodeDataNotFound:  "Data Not Found - The requested resource does not exist",
}
