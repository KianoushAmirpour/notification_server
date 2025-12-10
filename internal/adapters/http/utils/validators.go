package utils

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

func PasswordValidator(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	if !regexp.MustCompile(`[a-z]`).MatchString(password) {
		return false
	}

	if !regexp.MustCompile(`[A-Z]`).MatchString(password) {
		return false
	}

	if !regexp.MustCompile(`\d`).MatchString(password) {
		return false
	}

	if !regexp.MustCompile(`[@$!%*?&]`).MatchString(password) {
		return false
	}

	return true
}

func UserPreferencesValidation(fl validator.FieldLevel) bool {
	field := fl.Field()

	if field.Kind() != reflect.Slice {
		return false
	}

	if field.Len() > 5 {
		return false
	}

	for i := 0; i < field.Len(); i++ {
		item := field.Index(i)

		if item.Kind() != reflect.String {
			return false
		}

		preferences := item.String()
		if len(preferences) > 20 {
			return false
		}

		if len(strings.TrimSpace(preferences)) == 0 {
			return false
		}

	}

	return true
}
