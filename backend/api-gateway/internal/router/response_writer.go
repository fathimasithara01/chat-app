package router

import (
	"bytes"
	"io"
	"net/http"
)

// FakeWriter implements http.ResponseWriter
type FakeWriter struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func NewFakeWriter() *FakeWriter {
	return &FakeWriter{
		header: http.Header{},
		status: http.StatusOK,
	}
}

func (f *FakeWriter) Header() http.Header {
	return f.header
}

func (f *FakeWriter) Write(b []byte) (int, error) {
	return f.body.Write(b)
}

func (f *FakeWriter) WriteHeader(statusCode int) {
	f.status = statusCode
}

func (f *FakeWriter) Response() *http.Response {
	return &http.Response{
		StatusCode: f.status,
		Header:     f.header,
		Body:       io.NopCloser(bytes.NewReader(f.body.Bytes())),
	}
}
