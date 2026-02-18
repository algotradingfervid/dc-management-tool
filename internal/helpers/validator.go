package helpers

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// SharedValidator is a singleton validator instance.
var SharedValidator *validator.Validate

func init() {
	SharedValidator = validator.New()

	// Use json tag names as field names in error messages and map keys.
	SharedValidator.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" || name == "" {
			return fld.Name
		}
		return name
	})
}

// ValidateStruct validates a struct and returns errors in the legacy map[string]string format.
// Field names in the returned map use the json tag name if available.
func ValidateStruct(s interface{}) map[string]string {
	err := SharedValidator.Struct(s)
	if err == nil {
		return make(map[string]string)
	}

	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return map[string]string{"_error": err.Error()}
	}

	return convertValidationErrors(validationErrors)
}

// convertValidationErrors converts validator.ValidationErrors to map[string]string.
func convertValidationErrors(errors validator.ValidationErrors) map[string]string {
	result := make(map[string]string)

	for _, err := range errors {
		fieldName := err.Field()
		tag := err.Tag()
		param := err.Param()

		message := buildErrorMessage(fieldName, tag, param)
		result[fieldName] = message
	}

	return result
}

func buildErrorMessage(fieldName, tag, param string) string {
	friendlyName := fieldToFriendly(fieldName)

	switch tag {
	case "required":
		return fmt.Sprintf("%s is required", friendlyName)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", friendlyName)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", friendlyName, param)
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", friendlyName, param)
	case "len":
		return fmt.Sprintf("%s must be exactly %s characters", friendlyName, param)
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", friendlyName, param)
	case "gte":
		return fmt.Sprintf("%s must be at least %s", friendlyName, param)
	case "lt":
		return fmt.Sprintf("%s must be less than %s", friendlyName, param)
	case "lte":
		return fmt.Sprintf("%s must be at most %s", friendlyName, param)
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", friendlyName, strings.ReplaceAll(param, " ", ", "))
	case "numeric":
		return fmt.Sprintf("%s must be a number", friendlyName)
	default:
		return fmt.Sprintf("%s is invalid (%s)", friendlyName, tag)
	}
}

// fieldToFriendly converts a field name (snake_case or CamelCase) to title-case words.
// e.g., "item_name" -> "Item name", "ItemName" -> "Item Name"
func fieldToFriendly(s string) string {
	if strings.Contains(s, "_") {
		// snake_case: replace underscores with spaces, capitalize first letter
		words := strings.ReplaceAll(s, "_", " ")
		if words != "" {
			return strings.ToUpper(words[:1]) + words[1:]
		}
		return words
	}
	// CamelCase
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune(' ')
		}
		result.WriteRune(r)
	}
	return result.String()
}
