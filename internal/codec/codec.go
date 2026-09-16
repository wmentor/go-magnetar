package codec

import (
	"errors"
	"strings"

	"github.com/wmentor/go-magnetar/internal/codec/csv"
	"github.com/wmentor/go-magnetar/internal/codec/docx"
	"github.com/wmentor/go-magnetar/internal/codec/excel"
	"github.com/wmentor/go-magnetar/internal/codec/odt"
	"github.com/wmentor/go-magnetar/internal/codec/pdf"
	"github.com/wmentor/go-magnetar/internal/codec/pptx"
	"github.com/wmentor/go-magnetar/internal/codec/tsv"
)

var (
	ErrCodecNotFound = errors.New("codec not found")

	codecs = map[string]Codec{
		".csv":  &csv.Codec{},
		".docx": &docx.Codec{},
		".odt":  &odt.Codec{},
		".pdf":  &pdf.Codec{},
		".pptx": &pptx.Codec{},
		".tsv":  &tsv.Codec{},
		".xlsx": &excel.Codec{},
		"csv":   &csv.Codec{},
		"docx":  &docx.Codec{},
		"odt":   &odt.Codec{},
		"pdf":   &pdf.Codec{},
		"pptx":  &pptx.Codec{},
		"tsv":   &tsv.Codec{},
		"xlsx":  &excel.Codec{},
	}
)

type Codec interface {
	ReadFile(filename string) (string, error)
}

func GetCodec(extention string) (Codec, error) {
	if c, has := codecs[strings.ToLower(extention)]; has {
		return c, nil
	}
	return nil, ErrCodecNotFound
}
