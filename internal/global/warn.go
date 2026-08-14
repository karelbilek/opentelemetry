package global

import (
	"fmt"
	"log"
	"os"
)

// FIXME: this is not good :)
var globalLog = log.New(os.Stderr, "", log.LstdFlags|log.Lshortfile)

func Warn(msg string, args ...any) {
	a := make([]any, 0, len(args)+1)
	a = append(a, msg)
	a = append(a, args...)
	allMsg := fmt.Sprint(a...)
	globalLog.Println(allMsg)
}

func Info(msg string, args ...any) {
	Warn(msg, args...)
}

func Debug(msg string, args ...any) {
	Warn(msg, args...)
}

func Error(err error, msg string, args ...any) {
	// FIXME none of this is nice
	Warn(err.Error())
	Warn(msg, args...)
}
