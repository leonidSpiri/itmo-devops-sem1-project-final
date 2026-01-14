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
		return extractCSVFromZip(raw)
	case "tar":
		return extractCSVFromTar(raw)
	default:
		return nil, fmt.Errorf("unsupported archive type: %s", archiveType)
	}
}

func extractCSVFromZip(raw []byte) ([]byte, error) {
	r, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}

	if f := findZipFileByBase(r.File, "data.csv"); f != nil {
		return readZipFile(*f)
	}

	if f := findFirstZipCSV(r.File); f != nil {
		return readZipFile(*f)
	}

	return nil, fmt.Errorf("csv file not found in archive")
}

func extractCSVFromTar(raw []byte) ([]byte, error) {
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

	var firstCSV []byte
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

		base := strings.ToLower(filepath.Base(strings.TrimSpace(h.Name)))

		if base == "data.csv" {
			b, err := io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("read data.csv from tar: %w", err)
			}
			return b, nil
		}

		if firstCSV == nil && strings.HasSuffix(base, ".csv") {
			b, err := io.ReadAll(tr)
			if err != nil {
				return nil, fmt.Errorf("read csv from tar: %w", err)
			}
			firstCSV = b
		}
	}

	if firstCSV != nil {
		return firstCSV, nil
	}
	return nil, fmt.Errorf("csv file not found in archive")
}

func findZipFileByBase(files []*zip.File, wantBase string) *zip.File {
	wantBase = strings.ToLower(wantBase)
	for _, f := range files {
		base := strings.ToLower(filepath.Base(strings.TrimSpace(f.Name)))
		if base == wantBase {
			return f
		}
	}
	return nil
}

func findFirstZipCSV(files []*zip.File) *zip.File {
	for _, f := range files {
		base := strings.ToLower(filepath.Base(strings.TrimSpace(f.Name)))
		if strings.HasSuffix(base, ".csv") {
			return f
		}
	}
	return nil
}

func readZipFile(f zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("open file in zip: %w", err)
	}
	defer rc.Close()

	b, err := io.ReadAll(rc)
	if err != nil {
		return nil, fmt.Errorf("read csv from zip: %w", err)
	}
	return b, nil
}

func isGzip(b []byte) bool {
	return len(b) >= 2 && b[0] == 0x1f && b[1] == 0x8b
}
