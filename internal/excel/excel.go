package excel

import (
	"bytes"
	"fmt"

	"github.com/xuri/excelize/v2"

	"github.com/wmentor/go-magnetar/internal/markdown"
	"github.com/wmentor/go-magnetar/internal/printer"
)

func ReadFile(filename string) (string, error) {
	f, err := excelize.OpenFile(filename)
	if err != nil {
		return "", fmt.Errorf("open file %s error: %w", filename, err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return "", fmt.Errorf("no sheets found in file %s", filename)
	}

	result := bytes.NewBuffer(nil)

	for _, sheet := range sheets {
		if data, err := readSheet(filename, f, sheet); err == nil {
			result.WriteString(data)
		} else {
			printer.ToolCall(printer.IconError, "file_read: %v", err)
		}
	}

	return result.String(), nil
}

func readSheet(filename string, f *excelize.File, sheet string) (string, error) {
	rows, err := f.GetRows(sheet)
	if err != nil {
		return "", fmt.Errorf("unable to read rows from sheet %s in file %s: %w", sheet, filename, err)
	}

	md := markdown.RowsToTable(rows)

	return fmt.Sprintf("# %s\n\n%s\n", sheet, md), nil
}
