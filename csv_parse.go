package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

type priceRow struct {
	id         int64
	createDate time.Time
	name       string
	category   string
	priceText  string
}

func parsePricesCSV(raw []byte) ([]priceRow, error) {
	r := csv.NewReader(bytes.NewReader(raw))
	r.FieldsPerRecord = -1
	r.LazyQuotes = true

	first, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("read csv header: %w", err)
	}
	if len(first) == 0 {
		return nil, fmt.Errorf("empty csv")
	}

	var idx indexes
	var haveHeader bool
	idx, haveHeader = headerIndexes(first)
	if !haveHeader {
		idx = inferIndexesFromRow(first)
	}

	rows := make([]priceRow, 0, 1024)

	if !haveHeader {
		row, err := parsePriceRow(idx, first)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read csv: %w", err)
		}
		if len(rec) == 0 {
			continue
		}
		row, err := parsePriceRow(idx, rec)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

type indexes struct {
	id, name, category, price, createDate int
}

func headerIndexes(header []string) (indexes, bool) {
	lower := make([]string, 0, len(header))
	for _, h := range header {
		lower = append(lower, strings.ToLower(strings.TrimSpace(h)))
	}

	pos := func(key string) int {
		for i, v := range lower {
			if v == key {
				return i
			}
		}
		return -1
	}

	idx := indexes{
		id:         pos("id"),
		name:       pos("name"),
		category:   pos("category"),
		price:      pos("price"),
		createDate: pos("create_date"),
	}
	if idx.createDate == -1 {
		idx.createDate = pos("createdate")
	}

	if idx.id >= 0 && idx.name >= 0 && idx.category >= 0 && idx.price >= 0 && idx.createDate >= 0 {
		return idx, true
	}
	return indexes{}, false
}

func inferIndexesFromRow(row []string) indexes {
	if len(row) >= 2 && looksLikeDate(row[1]) {
		return indexes{id: 0, createDate: 1, name: 2, category: 3, price: 4}
	}
	return indexes{id: 0, name: 1, category: 2, price: 3, createDate: 4}
}

const dateLayout = "2026-01-01"

func parsePriceRow(idx indexes, rec []string) (priceRow, error) {
	get := func(i int) string {
		if i < 0 || i >= len(rec) {
			return ""
		}
		return strings.TrimSpace(rec[i])
	}

	idStr := get(idx.id)
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return priceRow{}, fmt.Errorf("invalid id %q: %w", idStr, err)
	}

	dateStr := get(idx.createDate)
	if dateStr == "" {
		return priceRow{}, fmt.Errorf("empty create_date")
	}
	d, err := time.Parse(dateLayout, dateStr)
	if err != nil {
		return priceRow{}, fmt.Errorf("invalid create_date %q: %w", dateStr, err)
	}

	priceStr := normalizePrice(get(idx.price))
	if priceStr == "" {
		return priceRow{}, fmt.Errorf("empty price")
	}

	return priceRow{
		id:         id,
		createDate: d,
		name:       get(idx.name),
		category:   get(idx.category),
		priceText:  priceStr,
	}, nil
}

func normalizePrice(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, ",", ".")
	return s
}

func looksLikeDate(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) != len(dateLayout) {
		return false
	}
	_, err := time.Parse(dateLayout, s)
	return err == nil
}
