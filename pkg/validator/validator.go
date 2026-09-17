package validator

import (
	"errors"
	"reflect"
	"strings"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
)

type Validator struct {
	v     *validator.Validate
	trans ut.Translator
}

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
} //@name FieldError

func New() (*Validator, error) {
	v := validator.New(validator.WithRequiredStructEnabled())

	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name, _, _ := strings.Cut(fld.Tag.Get("json"), ",")

		if name == "-" || name == "" {
			return fld.Name
		}

		return name
	})

	enLocale := en.New()
	uni := ut.New(enLocale, enLocale)
	trans, _ := uni.GetTranslator("en")

	if err := en_translations.RegisterDefaultTranslations(v, trans); err != nil {
		return nil, err
	}

	return &Validator{v: v, trans: trans}, nil
}

func (vd *Validator) Validate(s any) []FieldError {
	err := vd.v.Struct(s)

	if err == nil {
		return nil
	}

	var verrs validator.ValidationErrors

	if !errors.As(err, &verrs) {
		return []FieldError{{Message: err.Error()}}
	}

	out := make([]FieldError, len(verrs))
	for _, fe := range verrs {
		out = append(out, FieldError{Field: fe.Field(), Message: fe.Translate(vd.trans)})
	}

	return out
}
