package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"time"
)

func (h *pricesHandler) handleGetPrices(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	recs, err := h.loadPricesForExport(ctx)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	zipBytes, err := buildZipWithCSV(recs)
	if err != nil {
		writeAPIError(w, http.StatusInternalServerError, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=prices.zip")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(zipBytes)
}

type exportRecord struct {
	ID         int64
	Name       string
	Category   string
	PriceText  string
	CreateDate string
}

func (h *pricesHandler) loadPricesForExport(ctx context.Context) ([]exportRecord, error) {
	rows, err := h.db.QueryContext(ctx, `
SELECT
	id,
	name,
	category,
	price::text,
	to_char(create_date, 'YYYY-MM-DD')
FROM prices
ORDER BY id, create_date, name, category;
`)
	if err != nil {
		return nil, fmt.Errorf("select prices: %w", err)
	}
	defer rows.Close()

	out := make([]exportRecord, 0, 1024)
	for rows.Next() {
		var r exportRecord
		if err := rows.Scan(&r.ID, &r.Name, &r.Category, &r.PriceText, &r.CreateDate); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}
	return out, nil
}

func buildZipWithCSV(records []exportRecord) ([]byte, error) {
	var csvBuf bytes.Buffer
	w := csv.NewWriter(&csvBuf)

	if err := w.Write([]string{"id", "name", "category", "price", "create_date"}); err != nil {
		return nil, fmt.Errorf("write csv header: %w", err)
	}
	for _, r := range records {
		if err := w.Write([]string{
			fmt.Sprintf("%d", r.ID),
			r.Name,
			r.Category,
			r.PriceText,
			r.CreateDate,
		}); err != nil {
			return nil, fmt.Errorf("write csv row: %w", err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, fmt.Errorf("flush csv: %w", err)
	}

	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)
	f, err := zw.Create("data.csv")
	if err != nil {
		return nil, fmt.Errorf("create zip entry: %w", err)
	}
	if _, err := f.Write(csvBuf.Bytes()); err != nil {
		return nil, fmt.Errorf("write zip entry: %w", err)
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("close zip: %w", err)
	}
	return zipBuf.Bytes(), nil
}
