package models

// OTPRequest defines the structure for requesting an OTP
type OTPRequest struct {
	PhoneNumber string `json:"phone_number" validate:"required,e164"` // E.164 format
}

// OTPVerification defines the structure for verifying an OTP
type OTPVerification struct {
	PhoneNumber string `json:"phone_number" validate:"required,e164"` // E.164 format
	Code        string `json:"code" validate:"required,len=6"`         // Assuming 6-digit OTP
}

// VerifyEmailRequest defines the structure for verifying an email
type VerifyEmailRequest struct {
	Email string `json:"email" validate:"required,email"`
	Code  string `json:"code" validate:"required,len=6"` // Assuming 6-digit code
}