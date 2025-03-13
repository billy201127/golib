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

type CodeError struct {
	Code      int
	Msg       string
	Err       error
	CallStack string
}

func (c *CodeError) Error() string {
	return fmt.Sprintf("code: %d, msg: %s, error: %v", c.Code, c.Msg, c.Err)
}

func (c *CodeError) GetCallStack() string {
	var errorStr string
	if c.CallStack != "" {
		errorStr += fmt.Sprintf("\nerror stack: \n%s", c.CallStack)
	}
	return errorStr
}

var errorMetric = metric.NewCounterVec(&metric.CounterVecOpts{
	Namespace: "error",
	Subsystem: "code",
	Name:      "total",
	Help:      "How many error raised, partitioned by error code and critical flag.",
	Labels:    []string{"code", "msg", "critical"},
})

func New(code int, err error, useErrMsg ...bool) *CodeError {
	if err == nil {
		err = errors.New("error not set")
	}

	ce := &CodeError{Code: code, Err: err}

	if len(useErrMsg) > 0 && useErrMsg[0] {
		ce.Msg = err.Error()
		return ce
	}

	if v, ok := ErrMsgs[code]; ok {
		ce.Msg = v
	} else {
		ce.Msg = err.Error()
	}

	return ce
}

func RaiseCtx(ctx context.Context, code int, err error, args ...interface{}) *CodeError {
	ce := New(code, err)

	if err != nil {
		logx.WithContext(ctx).WithCallerSkip(1).Errorf("%s, args: %+v", ce, args)
	}

	return ce
}

func Raise(code int, err error, args ...interface{}) *CodeError {
	ce := New(code, err)

	if err != nil {
		logx.WithCallerSkip(1).Errorf("%s, args: %+v", ce, args)
	}

	return ce
}

func NewWithStack(code int, err error) *CodeError {
	ce := New(code, err)
	ce.CallStack = getStack(3)
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
