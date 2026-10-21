package printer

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
)

type Icon = string

const (
	IconAlert   = Icon("⚠️")
	IconAsk     = Icon("❓")
	IconBlocked = Icon("🚫")
	IconClock   = Icon("⏰")
	IconDone    = Icon("💫")
	IconError   = Icon("❌")
	IconFile    = Icon("📄")
	IconGuard   = Icon("🔒")
	IconIam     = Icon("👨")
	IconModule  = Icon("🛰️")
	IconRobot   = Icon("🤖")
	IconSave    = Icon("📝")
	IconSearch  = Icon("🔭")
	IconTool    = Icon("⚙️")
	IconURL     = Icon("🌐")
)

type Printer interface {
	Verbose(enable bool)
	Debug(message string, params ...any)
	Info(message string, params ...any)
	Warn(message string, params ...any)
	Error(message string, params ...any)
	ToolCall(typeIcon string, message string, params ...any)
	Print(typeIcon string, message string, params ...any)
	PrintRaw(data string)
	Ask(question string) string
	ToolEmptyLine()
	EmptyLine()
}

type printer struct {
	out     io.Writer
	verbose bool
}

func New(out io.Writer, verbose bool) Printer {
	return &printer{verbose: verbose, out: out}
}

var defaultPrinter Printer

func SetDefault(p Printer) {
	defaultPrinter = p
}

func GetDefault() Printer {
	return defaultPrinter
}

func init() {
	defaultPrinter = New(os.Stdout, true)
}

func Verbose(enable bool) {
	defaultPrinter.Verbose(enable)
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

func Print(typeIcon string, message string, params ...any) {
	defaultPrinter.Print(typeIcon, message, params...)
}

func ToolCall(typeIcon string, message string, params ...any) {
	defaultPrinter.ToolCall(typeIcon, message, params...)
}

func (p *printer) Debug(message string, params ...any) {
	if p.verbose {
		p.Print("", message, params...)
	}
}

func (p *printer) Info(message string, params ...any) {
	p.Print("", message, params...)
}

func (p *printer) Error(message string, params ...any) {
	p.Print(IconError, message, params...)
}

func (p *printer) Warn(message string, params ...any) {
	p.Print(IconAlert, message, params...)
}

func (p *printer) ToolCall(typeIcon string, message string, params ...any) {
	if p.verbose {
		p.Print(typeIcon, message, params...)
	}
}

func (p *printer) Print(typeIcon string, message string, params ...any) {
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
	fmt.Fprintln(p.out, sb.String())
}

func ToolEmptyLine() {
	defaultPrinter.ToolEmptyLine()
}

func Ask(question string) string {
	return defaultPrinter.Ask(question)
}

func (p *printer) Ask(question string) string {
	p.Print(IconRobot, question)
	for {
		fmt.Fprintf(p.out, " %s ", IconIam)

		var answer string
		_, _ = fmt.Scanln(&answer)
		answer = strings.TrimSpace(answer)
		if answer == "" {
			continue
		}
		return answer
	}
}

func (p *printer) ToolEmptyLine() {
	if p.verbose {
		fmt.Fprintln(p.out, "")
	}
}

func EmptyLine() {
	defaultPrinter.EmptyLine()
}

func (p *printer) EmptyLine() {
	fmt.Fprintln(p.out, "")
}

func (p *printer) Verbose(enable bool) {
	p.verbose = enable
}

func (p *printer) PrintRaw(data string) {
	if p.verbose {
		p.EmptyLine()
		fmt.Fprintln(p.out, " "+strings.ReplaceAll(data, "\n", "\n "))
		p.EmptyLine()
	}
}

func PrintRaw(data string) {
	GetDefault().PrintRaw(data)
}
