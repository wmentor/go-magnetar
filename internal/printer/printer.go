package printer

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
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
	IconRead    = Icon("📚")
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

// printer is safe for concurrent use: writes are serialized, so the lines of
// concurrent callers don't interleave.
type printer struct {
	mu      sync.Mutex
	out     io.Writer
	verbose atomic.Bool
}

func New(out io.Writer, verbose bool) Printer {
	p := &printer{out: out}
	p.verbose.Store(verbose)
	return p
}

var (
	defaultMu      sync.RWMutex
	defaultPrinter Printer
)

func SetDefault(p Printer) {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	defaultPrinter = p
}

func GetDefault() Printer {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultPrinter
}

func init() {
	defaultPrinter = New(os.Stdout, true)
}

func Verbose(enable bool) {
	GetDefault().Verbose(enable)
}

func Debug(message string, params ...any) {
	GetDefault().Debug(message, params...)
}

func Info(message string, params ...any) {
	GetDefault().Info(message, params...)
}

func Warn(message string, params ...any) {
	GetDefault().Warn(message, params...)
}

func Error(message string, params ...any) {
	GetDefault().Error(message, params...)
}

func Print(typeIcon string, message string, params ...any) {
	GetDefault().Print(typeIcon, message, params...)
}

func ToolCall(typeIcon string, message string, params ...any) {
	GetDefault().ToolCall(typeIcon, message, params...)
}

func (p *printer) Debug(message string, params ...any) {
	if p.verbose.Load() {
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
	if p.verbose.Load() {
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

	sb.WriteString(time.Now().Format(" 15:04:05 > "))

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
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprintln(p.out, sb.String())
}

func ToolEmptyLine() {
	GetDefault().ToolEmptyLine()
}

func Ask(question string) string {
	return GetDefault().Ask(question)
}

func (p *printer) Ask(question string) string {
	p.Print(IconRobot, question)
	for {
		p.mu.Lock()
		fmt.Fprintf(p.out, " %s ", IconIam)
		p.mu.Unlock()

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
	if p.verbose.Load() {
		p.EmptyLine()
	}
}

func EmptyLine() {
	GetDefault().EmptyLine()
}

func (p *printer) EmptyLine() {
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprintln(p.out, "")
}

func (p *printer) Verbose(enable bool) {
	p.verbose.Store(enable)
}

func (p *printer) PrintRaw(data string) {
	if p.verbose.Load() {
		// one lock for the whole block, so another caller's line can't land inside it
		p.mu.Lock()
		defer p.mu.Unlock()
		fmt.Fprintln(p.out, "")
		fmt.Fprintln(p.out, " "+strings.ReplaceAll(data, "\n", "\n "))
		fmt.Fprintln(p.out, "")
	}
}

func PrintRaw(data string) {
	GetDefault().PrintRaw(data)
}
