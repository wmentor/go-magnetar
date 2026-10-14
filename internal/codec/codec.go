package codec

import (
	"errors"
	"strings"

	"github.com/wmentor/go-magnetar/internal/codec/csv"
	"github.com/wmentor/go-magnetar/internal/codec/docx"
	"github.com/wmentor/go-magnetar/internal/codec/excel"
	"github.com/wmentor/go-magnetar/internal/codec/html"
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
		".htm":  &html.Codec{},
		".html": &html.Codec{},
		".odt":  &odt.Codec{},
		".pdf":  &pdf.Codec{},
		".pptx": &pptx.Codec{},
		".tsv":  &tsv.Codec{},
		".xlsx": &excel.Codec{},
		"csv":   &csv.Codec{},
		"docx":  &docx.Codec{},
		"htm":   &html.Codec{},
		"html":  &html.Codec{},
		"odt":   &odt.Codec{},
		"pdf":   &pdf.Codec{},
		"pptx":  &pptx.Codec{},
		"tsv":   &tsv.Codec{},
		"xlsx":  &excel.Codec{},
	}

	// mimeCodecs maps MIME types to codecs.
	mimeCodecs = map[string]Codec{
		"text/csv":                  &csv.Codec{},
		"text/tab-separated-values": &tsv.Codec{},
		"application/pdf":           &pdf.Codec{},
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   &docx.Codec{},
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         &excel.Codec{},
		"application/vnd.oasis.opendocument.text":                                   &odt.Codec{},
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": &pptx.Codec{},
	}

	mimeBinary = map[string]struct{}{
		"application/pdf": {},
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   {},
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         {},
		"application/vnd.oasis.opendocument.text":                                   {},
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": {},
	}
)

type Codec interface {
	ReadFile(filename string) (string, error)
}

func GetCodec(extentionName string) (Codec, error) {
	if c, has := codecs[strings.ToLower(extentionName)]; has {
		return c, nil
	}
	return nil, ErrCodecNotFound
}

func GetCodecByMIME(mime string) (Codec, error) {
	if c, has := mimeCodecs[strings.ToLower(mime)]; has {
		return c, nil
	}
	return nil, ErrCodecNotFound
}

func IsBinaryFormat(mimeType string) bool {
	_, has := mimeBinary[mimeType]
	return has
}
