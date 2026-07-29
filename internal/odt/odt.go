package odt

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"

	"github.com/pkg/errors"
)

var (
	ErrInvalidFileFormat = errors.New("invalid file format")
)

func ReadFile(filename string) (string, error) {
	zipReader, err := zip.OpenReader(filename)
	if err != nil {
		return "", err
	}
	defer zipReader.Close()

	var contentFile *zip.File
	for _, f := range zipReader.File {
		if f.Name == "content.xml" {
			contentFile = f
			break
		}
	}
	if contentFile == nil {
		return "", ErrInvalidFileFormat
	}

	// Открываем content.xml
	rc, err := contentFile.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	xmlData, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}

	// Парсим XML и собираем текст из тегов
	var textBuffer bytes.Buffer
	decoder := xml.NewDecoder(bytes.NewReader(xmlData))

	openP := false

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", ErrInvalidFileFormat
		}

		// Извлекаем текстовые данные
		switch se := token.(type) {
		case xml.CharData:
			if openP {
				textBuffer.Write(se)
			}
		case xml.StartElement:
			if se.Name.Local == "p" {
				openP = true
			}
			if se.Name.Local == "list-item" {
				textBuffer.WriteString("\n* ")
			}
		case xml.EndElement:
			if se.Name.Local == "p" {
				openP = false
				textBuffer.WriteString("\n\n")
			}
		}
	}

	return textBuffer.String(), nil
}
