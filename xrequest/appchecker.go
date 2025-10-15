package xrequest

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// getContextValueCaseInsensitive tries to get a value from context with different case variations
func getContextValueCaseInsensitive(ctx context.Context, key string) interface{} {
	// Generate common case variations
	caser := cases.Title(language.English)
	variations := []string{
		key,                  // original
		strings.ToUpper(key), // UPPER
		strings.ToLower(key), // lower
		caser.String(key),    // Title Case
	}

	// Try each variation
	for _, k := range variations {
		if v := ctx.Value(k); v != nil {
			return v
		}
	}

	return nil
}

func GetApp(ctx context.Context, req interface{}) (string, error) {
	if v := getContextValueCaseInsensitive(ctx, "APP-ID"); v != nil {
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

func GetCountry(ctx context.Context, req interface{}) (string, error) {
	if v := getContextValueCaseInsensitive(ctx, "COUNTRY"); v != nil {
		if str, ok := v.(fmt.Stringer); ok {
			return str.String(), nil
		}
		return fmt.Sprint(v), nil
	}
	return "", errors.New("COUNTRY field not exists in request struct")
}
