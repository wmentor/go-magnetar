package markdown

import (
	"bytes"
	"strings"
)

const (
	defaultAllocation = 32 * 1024
)

func RowsToTable(rows [][]string) string {
	if len(rows) == 0 || len(rows[0]) == 0 {
		return ""
	}

	buf := bytes.NewBuffer(make([]byte, 0, defaultAllocation))

	headerColCount := len(rows[0])

	writeHeader(buf, rows[0])

	for _, row := range rows[1:] {
		buf.WriteByte('|')
		for i := range headerColCount {
			if i < len(row) {
				writeString(buf, strings.TrimSpace(row[i]))
			} else {
				buf.WriteString("  ")
			}

			buf.WriteByte('|')
		}
		buf.WriteByte('\n')
	}

	return buf.String()
}

func writeHeader(out *bytes.Buffer, row []string) {
	out.WriteByte('|')
	for _, cell := range row {
		writeString(out, strings.TrimSpace(cell))
		out.WriteByte('|')
	}
	out.WriteByte('\n')

	out.WriteByte('|')

	for range len(row) {
		out.WriteString(" --- |")
	}
	out.WriteByte('\n')
}

func writeString(out *bytes.Buffer, str string) {
	out.WriteByte(' ')
	for _, r := range str {
		if r == '|' {
			out.WriteByte('\\')
		}
		out.WriteRune(r)
	}
	out.WriteByte(' ')
}
