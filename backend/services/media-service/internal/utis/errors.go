package utils

import "errors"

var (
	ErrInvalidFile    = errors.New("invalid file")
	ErrUploadFailed   = errors.New("file upload failed")
	ErrFileNotFound   = errors.New("file not found")
	ErrStorageFailure = errors.New("storage backend failure")
)
