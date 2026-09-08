package common

import (
	"github.com/wmentor/go-magnetar/internal/docx"
	"github.com/wmentor/go-magnetar/internal/odt"
	"github.com/wmentor/go-magnetar/internal/pdf"
	"github.com/wmentor/go-magnetar/internal/pptx"
)

type fileReader func(filename string) (string, error)

var FileReaders map[string]fileReader = map[string]fileReader{
	".docx": docx.ReadFile,
	".pdf":  pdf.ReadFile,
	".odt":  odt.ReadFile,
	".pptx": pptx.ReadFile,
}
