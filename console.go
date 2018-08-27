package blammo

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"

	"golang.org/x/crypto/ssh/terminal"
)

const bufferSize = 80

const timestampFormat = "2006-01-02 15:04:05 "

type Logger struct {
	ErrorWriter io.Writer
	InfoWriter  io.Writer
	DebugWriter io.Writer

	Timestamp string
	UTC       bool

	ErrorTag []byte
	WarnTag  []byte
	InfoTag  []byte
	DebugTag []byte
	KeyStart []byte
	KeyEnd   []byte

	Closer func()
}

// Event represents the text collected for output to a given log Writer
type Event struct {
	txt      []byte
	tag      []byte
	keyStart []byte
	keyEnd   []byte
	msgpos   int
	out      io.Writer
}

var eventPool = &sync.Pool{
	New: func() interface{} {
		return &Event{
			txt: make([]byte, 0, bufferSize),
		}
	},
}

// NewConsoleLogger creates a new logger with output to stdout and stderr,
// ANSI colored logging level tags, and timestamps to 1 second precision.
func NewConsoleLogger() *Logger {
	l := &Logger{
		ErrorWriter: os.Stderr,
		InfoWriter:  os.Stdout,
		DebugWriter: nil,
		Timestamp:   timestampFormat,
		ErrorTag:    []byte("[\x1b[91mERROR\x1b[0m] "),
		WarnTag:     []byte("[\x1b[93mWARN\x1b[0m ] "),
		InfoTag:     []byte("[\x1b[92mINFO\x1b[0m ] "),
		DebugTag:    []byte("[\x1b[37mDEBUG\x1b[0m] "),
		KeyStart:    []byte("\x1b[36m"),
		KeyEnd:      []byte("\x1b[0m"),
	}
	return l
}

// NewPipeLogger creates a new logger with output to stdout and stderr,
// no ANSI codes, and timestamps to 1 second precision.
func NewPipeLogger() *Logger {
	l := &Logger{
		ErrorWriter: os.Stderr,
		InfoWriter:  os.Stdout,
		DebugWriter: nil,
		Timestamp:   timestampFormat,
		ErrorTag:    []byte("[ERROR] "),
		WarnTag:     []byte("[WARN ] "),
		InfoTag:     []byte("[INFO ] "),
		DebugTag:    []byte("[DEBUG] "),
		KeyStart:    []byte(""),
		KeyEnd:      []byte(""),
	}
	return l
}

// NewCloudLogger creates a new logger with output to stdout and stderr,
// no ANSI codes or timestamps.
func NewCloudLogger() *Logger {
	l := &Logger{
		ErrorWriter: os.Stderr,
		InfoWriter:  os.Stdout,
		DebugWriter: nil,
		Timestamp:   "",
		ErrorTag:    []byte("[ERROR] "),
		WarnTag:     []byte("[WARN ] "),
		InfoTag:     []byte("[INFO ] "),
		DebugTag:    []byte("[DEBUG] "),
		KeyStart:    []byte(""),
		KeyEnd:      []byte(""),
	}
	return l
}

// NewFileLogger creates a new logger with output to the error and info log
// filenames provided, no ANSI codes, and timestamps to 1 second precision.
func NewFileLogger(errlog string, infolog string) (*Logger, error) {
	ferrlog, err := os.OpenFile(errlog, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return nil, fmt.Errorf("can't open error log: %v", err)
	}
	finfolog, err := os.OpenFile(infolog, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return nil, fmt.Errorf("can't open info log: %v", err)
	}
	l := &Logger{
		ErrorWriter: ferrlog,
		InfoWriter:  finfolog,
		DebugWriter: nil,
		Timestamp:   timestampFormat,
		ErrorTag:    []byte("[ERROR] "),
		WarnTag:     []byte("[WARN ] "),
		InfoTag:     []byte("[INFO ] "),
		DebugTag:    []byte("[DEBUG] "),
		KeyStart:    []byte(""),
		KeyEnd:      []byte(""),
		Closer: func() {
			ferrlog.Close()
			finfolog.Close()
		},
	}
	return l, nil
}

// NewLogger attempts to determine whether stdout is connected to the console. If so,
// it returns a ConsoleLogger; if not, it returns a PipeLogger.
func NewLogger() *Logger {
	if terminal.IsTerminal(int(os.Stdout.Fd())) {
		return NewConsoleLogger()
	}
	return NewPipeLogger()
}

// Close closes any open log files.
func (l *Logger) Close() {
	if l.Closer != nil {
		l.Closer()
	}
}

func (l *Logger) newEvent(w io.Writer, tag []byte) *Event {
	if w == nil {
		return nil
	}
	e := eventPool.Get().(*Event)
	e.keyStart = l.KeyStart
	e.keyEnd = l.KeyEnd
	e.out = w
	e.txt = e.txt[:0]
	if l.Timestamp != "" {
		if l.UTC {
			e.txt = time.Now().UTC().AppendFormat(e.txt, l.Timestamp)
		} else {
			e.txt = time.Now().AppendFormat(e.txt, l.Timestamp)
		}
	}
	e.txt = append(e.txt, tag...)
	e.msgpos = len(e.txt)
	return e
}

// Debug returns a debug level logging event you can add values and messages to
func (l *Logger) Debug() *Event {
	return l.newEvent(l.DebugWriter, l.DebugTag)
}

// Info returns an info level logging event you can add values and messages to
func (l *Logger) Info() *Event {
	return l.newEvent(l.InfoWriter, l.InfoTag)
}

// Warn returns a warning level logging event you can add values and messages to
func (l *Logger) Warn() *Event {
	return l.newEvent(l.ErrorWriter, l.WarnTag)
}

// Error returns an error level logging event you can add values and messages to
func (l *Logger) Error() *Event {
	return l.newEvent(l.ErrorWriter, l.ErrorTag)
}

// Splice inserts a string (as byte slice) into an existing string (as byte slice),
// starting at the specified insertion point. It then appends a newline to the result.
func splice(txt []byte, ins []byte, inspos int) []byte {
	// Calculate where to move the string currently at the insertion point
	ns := inspos + len(ins)
	// Increase length of slice by the length of the insertion by simply appending it
	txt = append(txt, ins...)
	// Copy the text at the insertion point to its new location
	copy(txt[ns:], txt[inspos:len(txt)-1])
	// Copy the insertion text into place
	copy(txt[inspos:ns], ins)
	return txt
}

func (e *Event) appendKey(key string) {
	e.txt = append(e.txt, e.keyStart...)
	e.txt = append(e.txt, []byte(key)...)
	e.txt = append(e.txt, e.keyEnd...)
	e.txt = append(e.txt, '=')
}

// Str adds a key (variable name) and string to the logging event.
func (e *Event) Str(key string, value string) *Event {
	if e == nil {
		return e
	}
	e.appendKey(key)
	e.txt = append(e.txt, []byte(value)...)
	e.txt = append(e.txt, ' ')
	return e
}

// Bool adds a key (variable name) and boolean to the logging event.
func (e *Event) Bool(key string, value bool) *Event {
	if e == nil {
		return e
	}
	if value {
		return e.Str(key, "true")
	}
	return e.Str(key, "false")
}

// Bytes adds a key (variable name) and slice of bytes to the logging event in hex.
func (e *Event) Bytes(key string, value []byte) *Event {
	if e == nil {
		return e
	}
	return e.Str(key, hex.EncodeToString(value))
}

// Err adds an error message as the @error key
func (e *Event) Err(err error) *Event {
	if e == nil {
		return e
	}
	if err == nil {
		return e.Str("@error", "nil");
	}
	return e.Str("@error", err.Error())
}

// Float32 adds a key (variable name) and float32 to the logging event.
func (e *Event) Float32(key string, f float32) *Event {
	if e == nil {
		return e
	}
	e.appendKey(key)
	e.txt = strconv.AppendFloat(e.txt, float64(f), 'G', -1, 32)
	e.txt = append(e.txt, ' ')
	return e
}

// Float64 adds a key (variable name) and float64 to the logging event.
func (e *Event) Float64(key string, f float64) *Event {
	if e == nil {
		return e
	}
	e.appendKey(key)
	e.txt = strconv.AppendFloat(e.txt, float64(f), 'G', -1, 32)
	e.txt = append(e.txt, ' ')
	return e
}

// Int adds a key (variable name) and integer to the logging event.
func (e *Event) Int(key string, value int) *Event {
	if e == nil {
		return e
	}
	return e.Str(key, strconv.Itoa(value))
}

// Uint8 adds a key (variable name) and integer to the logging event.
func (e *Event) Uint8(key string, value uint8) *Event {
	if e == nil {
		return e
	}
	e.appendKey(key)
	e.txt = strconv.AppendUint(e.txt, uint64(value), 10)
	e.txt = append(e.txt, ' ')
	return e
}

// Int8 adds a key (variable name) and integer to the logging event.
func (e *Event) Int8(key string, value int8) *Event {
	if e == nil {
		return e
	}
	e.appendKey(key)
	e.txt = strconv.AppendInt(e.txt, int64(value), 10)
	e.txt = append(e.txt, ' ')
	return e
}

// Uint16 adds a key (variable name) and integer to the logging event.
func (e *Event) Uint16(key string, value uint16) *Event {
	if e == nil {
		return e
	}
	e.appendKey(key)
	e.txt = strconv.AppendUint(e.txt, uint64(value), 10)
	e.txt = append(e.txt, ' ')
	return e
}

// Int16 adds a key (variable name) and integer to the logging event.
func (e *Event) Int16(key string, value int16) *Event {
	if e == nil {
		return e
	}
	e.appendKey(key)
	e.txt = strconv.AppendInt(e.txt, int64(value), 10)
	e.txt = append(e.txt, ' ')
	return e
}

// Uint32 adds a key (variable name) and integer to the logging event.
func (e *Event) Uint32(key string, value uint32) *Event {
	if e == nil {
		return e
	}
	e.appendKey(key)
	e.txt = strconv.AppendUint(e.txt, uint64(value), 10)
	e.txt = append(e.txt, ' ')
	return e
}

// Int32 adds a key (variable name) and integer to the logging event.
func (e *Event) Int32(key string, value int32) *Event {
	if e == nil {
		return e
	}
	e.appendKey(key)
	e.txt = strconv.AppendInt(e.txt, int64(value), 10)
	e.txt = append(e.txt, ' ')
	return e
}

// Uint64 adds a key (variable name) and integer to the logging event.
func (e *Event) Uint64(key string, value uint64) *Event {
	if e == nil {
		return e
	}
	e.appendKey(key)
	e.txt = strconv.AppendUint(e.txt, value, 10)
	e.txt = append(e.txt, ' ')
	return e
}

// Int64 adds a key (variable name) and integer to the logging event.
func (e *Event) Int64(key string, value int64) *Event {
	if e == nil {
		return e
	}
	e.appendKey(key)
	e.txt = strconv.AppendInt(e.txt, value, 10)
	e.txt = append(e.txt, ' ')
	return e
}

// Time adds a key (variable name) and time to the logging event.
func (e *Event) Time(key string, value time.Time) *Event {
	if e == nil {
		return e
	}
	e.appendKey(key)
	tv, err := value.MarshalText()
	if err != nil {
		tv = []byte(fmt.Sprintf("error marshaling time: %v", err))
	}
	e.txt = append(e.txt, tv...)
	e.txt = append(e.txt, ' ')
	return e
}

// Abbreviate chops off all but the last two pieces of a file path.
// e.g. /home/user/go/src/github.com/username/project/model/foo.go becomes model/foo.go
func abbreviate(path string) string {
	var ls, ps int
	for i, c := range path {
		if c == '/' {
			ps = ls
			ls = i
		}
	}
	if path[ps] == '/' {
		ps++
	}
	return path[ps:]
}

func (e *Event) writeCaller(n int) *Event {
	_, fn, line, _ := runtime.Caller(n)
	if n == 3 {
		e.Int("@line", line)
		e.Str("@file", abbreviate(fn))
	} else {
		e.Int("@c_line", line)
		e.Str("@c_file", abbreviate(fn))
	}
	return e
}

// Line writes the current line number and file of the source code as the
// @line and @file keys.
func (e *Event) Line() *Event {
	if e == nil {
		return e
	}
	return e.writeCaller(3)
}

// Caller writes the line number and file of the source code that the current
// function was called from, as the @c_line and @c_file keys.
func (e *Event) Caller() *Event {
	if e == nil {
		return e
	}
	return e.writeCaller(4)
}

// Msg writes the accumulated log entry to the log, along with the
// message provided.
func (e *Event) Msg(msg string) {
	if e == nil {
		return
	}
	bsx := []byte(msg + " ")
	e.txt = splice(e.txt, bsx, e.msgpos)
	e.txt[len(e.txt)-1] = '\n'
	e.out.Write(e.txt)
	eventPool.Put(e)
}

// Msgf writes a message formatted as per fmt.Sprintf. It's likely to be slower
// than any other log event method.
func (e *Event) Msgf(fmtstr string, vals ...interface{}) {
	if e == nil {
		return
	}
	msg := fmt.Sprintf(fmtstr, vals...)
	e.Msg(msg)
}
