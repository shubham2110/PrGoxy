package log

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
)

const (
	debug   = "[DEBUG]"
	info    = "[INFO]"
	err     = "[ERROR]"
	warn    = "[WARN]"
	success = "[SUCCESS]"
	data    = "[DATA]"
)

var enabled = []string{
	info,
	// err,
	// warn,
	// debug,
	success,
	// data,
}

func printMessagePrefix(colorNumber color.Attribute, message string) {
	color.New(colorNumber).Printf(message + " ")
	color.New(color.FgHiBlack).Printf(formatTime() + " ")
}

func Data(format string, a ...interface{}) {
	for _, mode := range enabled {
		if mode == "[DATA]" {
			printMessagePrefix(color.FgMagenta, mode)
			fmt.Fprintln(os.Stderr, fmt.Sprintf(format, a...))
			return
		}
	}
}

func Debug(format string, a ...interface{}) {
	for _, mode := range enabled {
		if mode == "[DEBUG]" {
			printMessagePrefix(color.FgYellow, mode)
			fmt.Fprintln(os.Stderr, fmt.Sprintf(format, a...))
			return
		}
	}
}

func Info(format string, a ...interface{}) {
	for _, mode := range enabled {
		if mode == "[INFO]" {
			printMessagePrefix(color.FgBlue, mode)
			fmt.Fprintln(os.Stderr, fmt.Sprintf(format, a...))
			return
		}
	}
}

func Error(format string, a ...interface{}) {
	for _, mode := range enabled {
		if mode == "[ERROR]" {
			printMessagePrefix(color.FgRed, mode)
			fmt.Fprintln(os.Stderr, fmt.Sprintf(format, a...))
			return
		}
	}
}
func Warn(format string, a ...interface{}) {
	for _, mode := range enabled {
		if mode == "[WARN]" {
			printMessagePrefix(color.FgMagenta, mode)
			fmt.Fprintln(os.Stderr, fmt.Sprintf(format, a...))
			return
		}
	}
}

func Success(format string, a ...interface{}) {
	for _, mode := range enabled {
		if mode == "[SUCCESS]" {
			printMessagePrefix(color.FgGreen, mode)
			fmt.Fprintln(os.Stderr, fmt.Sprintf(format, a...))
			return
		}
	}
}

func CommandPrompt(commandPrompt string) {
	color.New(color.FgYellow).Print(commandPrompt)
}

func formatTime() string {
	return time.Now().Format("2006/01/02 15:04:05")
}
