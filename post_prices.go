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

	mediaType, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
    if mediaType == "multipart/form-data" {
		if err := r.ParseMultipartForm(maxBody); err != nil {
			return nil, fmt.Errorf("parse multipart: %w", err)
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			if r.MultipartForm != nil {
				for _, fhs := range r.MultipartForm.File {
					if len(fhs) == 0 {
						continue
					}
					file, err = fhs[0].Open()
					if err != nil {
						continue
					}
					break
				}
			}
			if file == nil {
				return nil, fmt.Errorf("multipart: file field not found")
			}
		}
		defer file.Close()
		return io.ReadAll(io.LimitReader(file, maxBody))
	}

	return io.ReadAll(io.LimitReader(r.Body, maxBody))
}

func (h *pricesHandler) insertAndStats(ctx context.Context, rows []priceRow) (postPricesResponse, error) {
	if len(rows) == 0 {
		return postPricesResponse{}, nil
	}

	tx, err := h.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return postPricesResponse{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `INSERT INTO prices (id, create_date, name, category, price) VALUES ($1,$2,$3,$4,$5)`)
	if err != nil {
		return postPricesResponse{}, fmt.Errorf("prepare insert: %w", err)
	}
	defer stmt.Close()

	for _, row := range rows {
		if _, err := stmt.ExecContext(ctx, row.id, row.createDate, row.name, row.category, row.priceText); err != nil {
			return postPricesResponse{}, fmt.Errorf("insert row: %w", err)
		}
	}

	resp := postPricesResponse{}
	q := `
SELECT
  COUNT(*) AS total_items,
  COUNT(DISTINCT category) AS total_categories,
  COALESCE(SUM(price), 0) AS total_price
FROM prices;

`
	if err := tx.QueryRowContext(ctx, q).Scan(&resp.TotalItems, &resp.TotalCategories, &resp.TotalPrice); err != nil {
		return postPricesResponse{}, fmt.Errorf("select stats: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return postPricesResponse{}, fmt.Errorf("commit tx: %w", err)
	}
	return resp, nil
}
