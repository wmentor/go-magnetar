package eml

import (
	"fmt"
	"io"
	"net/mail"
	"os"
	"strings"

	"github.com/pkg/errors"
)

type Codec struct{}

func (c *Codec) ReadFile(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", errors.Wrapf(err, "open %q error", filename)
	}
	defer file.Close()

	msg, err := mail.ReadMessage(file)
	if err != nil {
		return "", errors.Wrapf(err, "parse file %q error", filename)
	}

	var buffer strings.Builder

	header := msg.Header

	fmt.Fprintf(&buffer, "Subject: %s\n", header.Get("Subject"))
	fmt.Fprintf(&buffer, "From:    %s\n", header.Get("From"))
	fmt.Fprintf(&buffer, "To:      %s\n", header.Get("To"))
	fmt.Fprintf(&buffer, "Date:    %s\n", header.Get("Date"))

	body, err := io.ReadAll(msg.Body)
	if err != nil {
		return "", errors.Wrap(err, "read message body")
	}
	fmt.Fprintf(&buffer, "\nRaw Body:\n\n%s\n", body)

	return buffer.String(), nil
}
