package server

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

const maxUpload = 8 << 20 // 8 MiB

// readUpload accepts either a multipart form (field "file") or a raw request
// body, and returns the uploaded text.
func readUpload(req *http.Request) (string, error) {
	content, _, err := readUploadNamed(req)
	return content, err
}

// readUploadNamed is like readUpload but also returns the uploaded filename
// (empty for a raw body).
func readUploadNamed(req *http.Request) (content, name string, err error) {
	ct := req.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		if err := req.ParseMultipartForm(maxUpload); err != nil {
			return "", "", fmt.Errorf("parse upload: %w", err)
		}
		f, hdr, err := req.FormFile("file")
		if err != nil {
			return "", "", fmt.Errorf("missing file field: %w", err)
		}
		defer f.Close()
		b, err := io.ReadAll(io.LimitReader(f, maxUpload))
		if err != nil {
			return "", "", err
		}
		return string(b), hdr.Filename, nil
	}
	b, err := io.ReadAll(io.LimitReader(req.Body, maxUpload))
	if err != nil {
		return "", "", err
	}
	if len(b) == 0 {
		return "", "", fmt.Errorf("empty upload")
	}
	return string(b), "", nil
}

func newStringReader(s string) io.Reader { return strings.NewReader(s) }
