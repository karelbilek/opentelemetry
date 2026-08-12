package global

import (
	"fmt"
	"log"
	"os"
)

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
