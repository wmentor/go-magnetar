package printer

import (
	"bytes"
	"fmt"
)

type Icon = string

const (
	IconAlert  = Icon("⚠️")
	IconDone   = Icon("💫")
	IconError  = Icon("❌")
	IconModule = Icon("🛰️")
	IconSave   = Icon("🚀")
	IconSearch = Icon("🔭")
	IconTool   = Icon("⚙️")
)

type Printer struct {
	verbose bool
}

func New(verbose bool) *Printer {
	return &Printer{verbose: verbose}
}

var (
	defaultPrinter *Printer
)

func SetDefault(p *Printer) {
	defaultPrinter = p
}

func init() {
	defaultPrinter = New(true)
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

func ToolCall(typeIcon string, message string, params ...any) {
	defaultPrinter.ToolCall(typeIcon, message, params...)
}
func (p *Printer) Debug(message string, params ...any) {
	if p.verbose {
		p.Print("", message, params...)
	}
}

func (p *Printer) Info(message string, params ...any) {
	p.Print("", message, params...)
}

func (p *Printer) Error(message string, params ...any) {
	p.Print(IconError, message, params...)
}

func (p *Printer) Warn(message string, params ...any) {
	p.Print(IconAlert, message, params...)
}

func (p *Printer) ToolCall(typeIcon string, message string, params ...any) {
	if p.verbose {
		p.Print(typeIcon, message, params...)
	}
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

func ToolEmptyLine() {
	defaultPrinter.ToolEmptyLine()
}

func (p *Printer) ToolEmptyLine() {
	if p.verbose {
		fmt.Println("")
	}
}
