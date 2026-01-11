package main

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

func extractDataCSV(archiveType string, raw []byte) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(archiveType)) {
	case "", "zip":
		return extractDataCSVFromZip(raw)
	case "tar":
		return extractDataCSVFromTar(raw)
	default:
		return nil, fmt.Errorf("unsupported archive type: %s", archiveType)
	}
}

func extractDataCSVFromZip(raw []byte) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	for _, f := range r.File {
		if isDataCSVPath(f.Name) {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("open file in zip: %w", err)
			}
			defer rc.Close()

			b, err := io.ReadAll(rc)
			if err != nil {
				return nil, fmt.Errorf("read data.csv from zip: %w", err)
			}
			return b, nil
		}
	}
	return nil, fmt.Errorf("data.csv not found in archive")
}

func extractDataCSVFromTar(raw []byte) ([]byte, error) {
	var rdr io.Reader = bytes.NewReader(raw)
	if isGzip(raw) {
		gz, err := gzip.NewReader(bytes.NewReader(raw))
		if err != nil {
			return nil, fmt.Errorf("open gzip: %w", err)
		}
		defer gz.Close()
		rdr = gz
	}

	tr := tar.NewReader(rdr)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tar: %w", err)
		}
		if h == nil || h.FileInfo().IsDir() {
			continue
		}
		if isDataCSVPath(h.Name) {
			b, err := io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("read data.csv from tar: %w", err)
			}
			return b, nil
		}
	}
	return nil, fmt.Errorf("data.csv not found in archive")
}

func isDataCSVPath(name string) bool {
	base := strings.ToLower(filepath.Base(strings.TrimSpace(name)))
	return base == "data.csv"
}

func isGzip(b []byte) bool {
	return len(b) >= 2 && b[0] == 0x1f && b[1] == 0x8b
}
