package utils

import (
	"errors"
	"mime/multipart"
)

var allowedTypes = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/jpg":  true,
	"video/mp4":  true,
}

func ValidateFileHeader(h *multipart.FileHeader) error {
	if h.Size == 0 || h.Size > 50*1024*1024 {
		return errors.New("file size not allowed")
	}

	if !allowedTypes[h.Header.Get("Content-Type")] {
		return errors.New("invalid content type")
	}

	return nil
}
