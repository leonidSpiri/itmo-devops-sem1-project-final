package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"
	"strconv"
)

type postPricesResponse struct {
	TotalItems      int64   `json:"total_items"`
	TotalCategories int64   `json:"total_categories"`
	TotalPrice      float64 `json:"total_price"`
}

func (h *pricesHandler) handlePostPrices(w http.ResponseWriter, r *http.Request) {
	archiveType := strings.TrimSpace(r.URL.Query().Get("type"))
	if archiveType == "" {
		archiveType = "zip"
	}

	rawArchive, err := readArchiveFromRequest(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	csvBytes, err := extractDataCSV(archiveType, rawArchive)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	rows, err := parsePricesCSV(csvBytes)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx := r.Context()

	resp, err := h.insertAndStats(ctx, rows)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func readArchiveFromRequest(r *http.Request) ([]byte, error) {
	if isMultipartForm(r) {
		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			return nil, fmt.Errorf("parse multipart: %w", err)
		}

		file, _, err := r.FormFile("file")
		if err == nil {
			defer file.Close()
			return io.ReadAll(io.LimitReader(file, maxUploadSize))
		}

		if r.MultipartForm != nil {
			for _, fhs := range r.MultipartForm.File {
				if len(fhs) == 0 {
					continue
				}
				f, openErr := fhs[0].Open()
				if openErr != nil {
					continue
				}
				defer f.Close()
				return io.ReadAll(io.LimitReader(f, maxUploadSize))
			}
		}

		return nil, fmt.Errorf("multipart: file field not found")
	}

	return io.ReadAll(io.LimitReader(r.Body, maxUploadSize))
}

func (h *pricesHandler) insertAndStats(ctx context.Context, rows []priceRow) (postPricesResponse, error) {
	resp := postPricesResponse{
		TotalItems: int64(len(rows)),
	}

	tx, err := h.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return postPricesResponse{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO prices (product_id, create_date, name, category, price) VALUES ($1,$2,$3,$4,$5)`)
	if err != nil {
		return postPricesResponse{}, fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	for _, row := range rows {
		if _, err := stmt.ExecContext(ctx, row.id, row.createDate, row.name, row.category, row.priceText); err != nil {
			return postPricesResponse{}, fmt.Errorf("insert row: %w", err)
		}
	}

	var totalPriceStr string
	q := `
SELECT
  COUNT(DISTINCT category) AS total_categories,
  COALESCE(SUM(price), 0) AS total_price
FROM prices;
`
	if err := tx.QueryRowContext(ctx, q).Scan(&resp.TotalCategories, &totalPriceStr); err != nil {
		return postPricesResponse{}, fmt.Errorf("select stats: %w", err)
	}

	totalPrice, err := strconv.ParseFloat(totalPriceStr, 64)
	if err != nil {
		return postPricesResponse{}, fmt.Errorf("parse total_price %q: %w", totalPriceStr, err)
	}
	resp.TotalPrice = totalPrice

	if err := tx.Commit(); err != nil {
		return postPricesResponse{}, fmt.Errorf("commit tx: %w", err)
	}
	return resp, nil
}

func isMultipartForm(r *http.Request) bool {
	for _, ct := range r.Header.Values("Content-Type") {
		mediaType, _, err := mime.ParseMediaType(ct)
		if err == nil && mediaType == "multipart/form-data" {
			return true
		}
	}

	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	return err == nil && mediaType == "multipart/form-data"
}

