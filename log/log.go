package log

import "github.com/lpar/blammo"

// Logger is the global logger
var Logger = blammo.NewConsoleLogger()

// Debug returns a debug level logging event you can add values and messages to
func Debug() *blammo.Event {
	return Logger.Debug()
}

// Info returns an info level logging event you can add values and messages to
func Info() *blammo.Event {
	return Logger.Info()
}

// Warn returns a warning level logging event you can add values and messages to
func Warn() *blammo.Event {
	return Logger.Warn()
}

// Error returns an error level logging event you can add values and messages to
func Error() *blammo.Event {
	return Logger.Error()
}
