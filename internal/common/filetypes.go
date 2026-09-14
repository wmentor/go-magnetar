package common

import (
	"github.com/wmentor/go-magnetar/internal/csv"
	"github.com/wmentor/go-magnetar/internal/docx"
	"github.com/wmentor/go-magnetar/internal/excel"
	"github.com/wmentor/go-magnetar/internal/odt"
	"github.com/wmentor/go-magnetar/internal/pdf"
	"github.com/wmentor/go-magnetar/internal/pptx"
)

type fileReader func(filename string) (string, error)

var FileReaders map[string]fileReader = map[string]fileReader{
	".csv":  csv.ReadFile,
	".docx": docx.ReadFile,
	".odt":  odt.ReadFile,
	".pdf":  pdf.ReadFile,
	".pptx": pptx.ReadFile,
	".xlsx": excel.ReadFile,
}
