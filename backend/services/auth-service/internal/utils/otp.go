package utils

import (
	"math/rand" // Added for string manipulation
	"time"
)

// seed rand for OTP generation
func init() {
	rand.Seed(time.Now().UnixNano())
}

// GenerateOTP generates a random numeric OTP of the given length.
func GenerateOTP(length int) string {
	if length <= 0 {
		return ""
	}
	const charset = "0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
