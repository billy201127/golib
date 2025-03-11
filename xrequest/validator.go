package xrequest

import (
	"errors"
	"reflect"

	enLocal "github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTrans "github.com/go-playground/validator/v10/translations/en"
)

func Validate(v any) error {
	err := validate.Struct(v)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			return errors.New(err.Translate(trans))
		}
	}
	return nil
}

var (
	validate *validator.Validate
	trans    ut.Translator
)

func init() {
	validate = validator.New()
	validate.RegisterTagNameFunc(func(field reflect.StructField) string {
		return field.Tag.Get("label")
	})
	initCustomValidator(validate)

	local := enLocal.New()
	trans, _ = ut.New(local).GetTranslator(local.Locale())
	_ = enTrans.RegisterDefaultTranslations(validate, trans)
}

func initCustomValidator(validate *validator.Validate) {
	// add custom validators here

}
