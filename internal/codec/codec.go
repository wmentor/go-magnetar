package codec

import (
	"errors"
	"strings"

	"github.com/wmentor/go-magnetar/internal/codec/csv"
	"github.com/wmentor/go-magnetar/internal/codec/docx"
	"github.com/wmentor/go-magnetar/internal/codec/eml"
	"github.com/wmentor/go-magnetar/internal/codec/excel"
	"github.com/wmentor/go-magnetar/internal/codec/html"
	"github.com/wmentor/go-magnetar/internal/codec/odt"
	"github.com/wmentor/go-magnetar/internal/codec/pages"
	"github.com/wmentor/go-magnetar/internal/codec/pdf"
	"github.com/wmentor/go-magnetar/internal/codec/pptx"
	"github.com/wmentor/go-magnetar/internal/codec/tsv"
)

var (
	ErrCodecNotFound = errors.New("codec not found")

	readCodecs = map[string]ReadCodec{
		".csv":   &csv.Codec{},
		".docx":  &docx.Codec{},
		".eml":   &eml.Codec{},
		".htm":   &html.Codec{},
		".html":  &html.Codec{},
		".odt":   &odt.Codec{},
		".pages": &pages.Codec{},
		".pdf":   &pdf.Codec{},
		".pptx":  &pptx.Codec{},
		".tsv":   &tsv.Codec{},
		".xlsx":  &excel.Codec{},
		"csv":    &csv.Codec{},
		"docx":   &docx.Codec{},
		"eml":    &eml.Codec{},
		"htm":    &html.Codec{},
		"html":   &html.Codec{},
		"odt":    &odt.Codec{},
		"pages":  &pages.Codec{},
		"pdf":    &pdf.Codec{},
		"pptx":   &pptx.Codec{},
		"tsv":    &tsv.Codec{},
		"xlsx":   &excel.Codec{},
	}

	writeCodecs = map[string]WriteCodec{
		".docx": &docx.Codec{},
		"docx":  &docx.Codec{},
	}

	// mimeCodecs maps MIME types to codecs.
	mimeReadCodecs = map[string]ReadCodec{
		"application/pdf":             &pdf.Codec{},
		"application/vnd.apple.pages": &pages.Codec{},
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   &docx.Codec{},
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         &excel.Codec{},
		"application/vnd.oasis.opendocument.text":                                   &odt.Codec{},
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": &pptx.Codec{},
		"application/x-iwork-pages-sffpages":                                        &pages.Codec{},
		"message/rfc822":                                                            &eml.Codec{},
		"text/csv":                                                                  &csv.Codec{},
		"text/tab-separated-values":                                                 &tsv.Codec{},
	}

	mimeBinary = map[string]struct{}{
		"application/pdf": {},
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   {},
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         {},
		"application/vnd.oasis.opendocument.text":                                   {},
		"application/vnd.openxmlformats-officedocument.presentationml.presentation": {},
		"message/rfc822": {},
	}
)

type ReadCodec interface {
	ReadFile(filename string) (string, error)
}

type WriteCodec interface {
	WriteFile(filename string, content string) error
}

func GetCodec(extentionName string) (ReadCodec, error) {
	if c, has := readCodecs[strings.ToLower(extentionName)]; has {
		return c, nil
	}
	return nil, ErrCodecNotFound
}

func GetWriteCodec(extentionName string) (WriteCodec, error) {
	if c, has := writeCodecs[strings.ToLower(extentionName)]; has {
		return c, nil
	}
	return nil, ErrCodecNotFound
}

func GetCodecByMIME(mime string) (ReadCodec, error) {
	if c, has := mimeReadCodecs[strings.ToLower(mime)]; has {
		return c, nil
	}
	return nil, ErrCodecNotFound
}

func IsBinaryFormat(mimeType string) bool {
	_, has := mimeBinary[mimeType]
	return has
}
