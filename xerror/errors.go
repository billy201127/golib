package xerror

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/metric"
)

type Error struct {
	code  int    // 错误码
	msg   string // 用户可读的错误消息
	cause error  // 原始错误（导致此错误的根本原因）
	stack string // 可选的调用栈信息
}

func (e *Error) SetCode(code int) *Error {
	e.code = code
	return e
}
func (e *Error) SetMsg(msg string) *Error {
	e.msg = msg
	return e
}

func (e *Error) SetCause(cause error) *Error {
	e.cause = cause
	return e
}

func (e *Error) SetStack(stack string) *Error {
	e.stack = stack
	return e
}

// Code 返回错误码
func (e *Error) Code() int {
	return e.code
}

// Message 返回错误消息
func (e *Error) Message() string {
	return e.msg
}

// Cause 返回原始错误
func (e *Error) Cause() error {
	return e.cause
}

// Stack 返回调用栈信息
func (e *Error) Stack() string {
	return e.stack
}

// Error 实现 error 接口
func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("code: %d, msg: %s, cause: %v", e.code, e.msg, e.cause)
	}
	return fmt.Sprintf("code: %d, msg: %s", e.code, e.msg)
}

// Unwrap 实现错误链支持
func (e *Error) Unwrap() error {
	return e.cause
}

var errorMetric = metric.NewCounterVec(&metric.CounterVecOpts{
	Namespace: "error",
	Subsystem: "code",
	Name:      "total",
	Help:      "How many error raised, partitioned by error code and critical flag.",
	Labels:    []string{"code", "msg", "critical"},
})

func New(code int, err error, useErrMsg ...bool) *Error {
	if err == nil {
		err = errors.New("error not set")
	}

	ce := &Error{code: code, cause: err}

	if len(useErrMsg) > 0 && useErrMsg[0] {
		ce.msg = err.Error()
		return ce
	}

	if v, ok := ErrMsgs[code]; ok {
		ce.msg = v
	} else {
		ce.msg = err.Error()
	}

	return ce
}

func RaiseCtx(ctx context.Context, code int, err error, args ...interface{}) *Error {
	ce := New(code, err)

	if err != nil {
		logx.WithContext(ctx).WithCallerSkip(1).Errorf("%s, args: %+v", ce, args)
	}

	return ce
}

func Raise(code int, err error, args ...interface{}) *Error {
	ce := New(code, err)

	if err != nil {
		logx.WithCallerSkip(1).Errorf("%s, args: %+v", ce, args)
	}

	return ce
}

func NewWithStack(code int, err error) *Error {
	ce := New(code, err)
	ce.stack = getStack(3)
	return ce
}

func getStack(offset int) string {
	const depth = 32
	var pcs [depth]uintptr
	// runtime.Callers used to get the current call stack information, +2 is to skip runtime.Callers and NewStackError itself
	n := runtime.Callers(offset, pcs[:])

	var str strings.Builder
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		// write file name and line number information
		str.WriteString(fmt.Sprintf("%s:%d %s\n", frame.File, frame.Line, frame.Function))
		if !more {
			break
		}
	}
	return str.String()
}
