package utils

import (
	"errors"
	"fmt"
	"strings" // Ensure this is imported if FormatPhoneNumberToE164 is here

	"github.com/go-playground/validator/v10"
)

// ValidationError represents a single validation error
type ValidationError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Value   string `json:"value,omitempty"`
	Message string `json:"message,omitempty"`
}

// FormatValidationErrors converts validator.ValidationErrors into a slice of ValidationError
func FormatValidationErrors(err error) []ValidationError {
	if err == nil {
		return nil
	}

	var ve validator.ValidationErrors
	if errors.As(err, &ve) {
		out := make([]ValidationError, len(ve))
		for i, fe := range ve {
			out[i] = ValidationError{
				Field: fe.Field(),
				Tag:   fe.Tag(),
				// Corrected: Use fe.Value() to get the actual field value that failed validation
				// Use fmt.Sprintf("%v", ...) to handle different types of values
				Value: fmt.Sprintf("%v", fe.Value()),
			}
			// Customize error message for common tags
			switch fe.Tag() {
			case "required":
				out[i].Message = fmt.Sprintf("%s is required", fe.Field())
			case "email":
				out[i].Message = fmt.Sprintf("%s must be a valid email address", fe.Field())
			case "min":
				out[i].Message = fmt.Sprintf("%s must be at least %s characters long", fe.Field(), fe.Param())
			case "len":
				out[i].Message = fmt.Sprintf("%s must be exactly %s characters long", fe.Field(), fe.Param())
			// Add more custom messages for other tags as needed
			default:
				out[i].Message = fmt.Sprintf("Validation failed on field '%s' for tag '%s'", fe.Field(), fe.Tag())
			}
		}
		return out
	}
	return nil // Not a validation error
}

// Ensure the FormatPhoneNumberToE164 function is still present if you put it in this file
// if not, then remove it or put it in a separate utils file as you prefer.
// For now, I'm keeping it here for completeness of previous context.

// FormatPhoneNumberToE164 ensures the phone number is in E.164 format.
// This is a basic example and might need a more robust library like libphonenumber for real-world use.
func FormatPhoneNumberToE164(phoneNumber string) string {
	// Remove non-digit characters
	var sb strings.Builder
	for _, r := range phoneNumber {
		if r >= '0' && r <= '9' {
			sb.WriteRune(r)
		}
	}
	cleaned := sb.String()

	if !strings.HasPrefix(cleaned, "+") {

		return "+91" + cleaned
	}
	return cleaned
}
