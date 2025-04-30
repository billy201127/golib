package xrequest

import (
	"context"
	"errors"
	"fmt"
	"reflect"
)

func GetApp(ctx context.Context, req interface{}) (string, error) {
	if v := ctx.Value("APP-ID"); v != nil {
		if str, ok := v.(fmt.Stringer); ok {
			return str.String(), nil
		}
		return fmt.Sprint(v), nil
	}

	// Use reflection to check if req has App field
	v := reflect.ValueOf(req)

	// Handle pointer type
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Must be a struct
	if v.Kind() != reflect.Struct {
		return "", errors.New("request struct is not a struct")
	}

	// Try to get App field first
	f := v.FieldByName("App")
	if f.IsValid() {
		return fmt.Sprint(f.Interface()), nil
	}

	// If App field doesn't exist, try to get AppId field
	f = v.FieldByName("AppId")
	if f.IsValid() {
		return fmt.Sprint(f.Interface()), nil
	}

	return "", errors.New("neither App nor AppId field exists in request struct")
}
