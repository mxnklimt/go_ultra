package domain

import (
	"encoding/csv"
	"os"
	"strconv"
	"testing"
)

func TestDanFromCSVFixture(t *testing.T) {
	const path = "testdata/rank_cases.csv"
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open fixture %s: %v", path, err)
	}
	defer f.Close()

	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatalf("parse csv %s: %v", path, err)
	}
	if len(rows) < 2 {
		t.Fatalf("fixture %s has no data rows", path)
	}

	header := rows[0]
	if len(header) != 2 || header[0] != "rating" || header[1] != "expected_dan" {
		t.Fatalf("unexpected header %v, want [rating expected_dan]", header)
	}

	for i, row := range rows[1:] {
		lineNo := i + 2
		if len(row) != 2 {
			t.Fatalf("line %d: expected 2 columns, got %d (%v)", lineNo, len(row), row)
		}
		rating, err := strconv.ParseFloat(row[0], 64)
		if err != nil {
			t.Fatalf("line %d: bad rating %q: %v", lineNo, row[0], err)
		}
		expected, err := strconv.Atoi(row[1])
		if err != nil {
			t.Fatalf("line %d: bad expected_dan %q: %v", lineNo, row[1], err)
		}
		if got := Dan(rating); got != expected {
			t.Fatalf("line %d: Dan(%v) = %d, want %d", lineNo, rating, got, expected)
		}
	}
}
