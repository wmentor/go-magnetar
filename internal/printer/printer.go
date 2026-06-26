package printer

import (
	"bytes"
	"fmt"
)

const (
	IconTool = "🚀"
)

type Printer struct{}

func New() *Printer {
	return &Printer{}
}

var (
	defaultPrinter *Printer
)

func SetDefault(p *Printer) {
	defaultPrinter = p
}

func init() {
	defaultPrinter = New()
}

func Debug(message string, params ...any) {
	defaultPrinter.Debug(message, params...)
}

func Info(message string, params ...any) {
	defaultPrinter.Info(message, params...)
}

func Warn(message string, params ...any) {
	defaultPrinter.Warn(message, params...)
}

func Error(message string, params ...any) {
	defaultPrinter.Error(message, params...)
}

func (p *Printer) Debug(message string, params ...any) {
	p.Print("", message, params...)
}

func (p *Printer) Info(message string, params ...any) {
	p.Print("", message, params...)
}

func (p *Printer) Error(message string, params ...any) {
	p.Print("", message, params...)
}

func (p *Printer) Warn(message string, params ...any) {
	p.Print("⚠️", message, params...)
}

func (p *Printer) Print(typeIcon string, message string, params ...any) {
	sb := bytes.NewBuffer(nil)

	if typeIcon != "" {
		sb.WriteRune(' ')
		sb.WriteString(typeIcon)
		sb.WriteRune(' ')
	}
	sb.WriteString(message)

	if len(params) > 1 {
		sb.WriteString(" [")
		for i := range len(params) / 2 {
			if i != 0 {
				sb.WriteRune(',')
			}
			fmt.Fprintf(sb, "%v=%v", params[i*2], params[i*2+1])
		}
		sb.WriteString("]")
	}
	fmt.Println(sb.String())
}
