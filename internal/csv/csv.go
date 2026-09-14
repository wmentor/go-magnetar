package csv

import (
	"encoding/csv"
	"fmt"
	"os"

	"github.com/wmentor/go-magnetar/internal/markdown"
)

func ReadFile(filename string) (string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("unable to open file %s: %w", filename, err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	rows, err := reader.ReadAll()
	if err != nil {
		return "", fmt.Errorf("unable to parse file %s: %w", filename, err)
	}

	return markdown.RowsToTable(rows), nil
}
