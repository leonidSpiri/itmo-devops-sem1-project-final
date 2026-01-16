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
	ctx := r.Context()

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
	ProductID   int64
	Name        string
	Category    string
	Price       string
	CreateDate  time.Time
}


func (h *pricesHandler) loadPricesForExport(ctx context.Context) ([]exportRecord, error) {
	rows, err := h.db.QueryContext(ctx, `
SELECT
  product_id,
  name,
  category,
  price,
  create_date
FROM prices
ORDER BY product_id, create_date, name, category;

`)
	if err != nil {
		return nil, fmt.Errorf("select prices: %w", err)
	}
	defer rows.Close()

	var out []exportRecord
	for rows.Next() {
		var r exportRecord
        if err := rows.Scan(&r.ProductID, &r.Name, &r.Category, &r.Price, &r.CreateDate); err != nil {
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
	cw := csv.NewWriter(&csvBuf)

	if err := cw.Write([]string{"id", "name", "category", "price", "create_date"}); err != nil {
		return nil, fmt.Errorf("write csv header: %w", err)
	}
	for _, r := range records {
		if err := cw.Write([]string{
			strconv.FormatInt(r.ProductID, 10),
			r.Name,
			r.Category,
			r.Price,
			r.CreateDate.Format("2006-01-02"),
		}); err != nil {
			return nil, fmt.Errorf("write csv row: %w", err)
		}
	}

	cw.Flush()
	if err := cw.Error(); err != nil {
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
