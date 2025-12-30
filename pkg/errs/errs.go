package errs

import (
	"errors"
	"log"
	"os"
	"runtime"
	"time"
)

const (
	traceSkip     = 3
	trackPrealloc = 50
)

type sFrame struct {
	filename string
	method   string
	line     int
}

type stack []sFrame

type errorWithTrace struct {
	error

	trace stack
}

func NewStack(err error) error {
	if err == nil {
		return nil
	}

	var errWT errorWithTrace

	// Add trace only once
	if errors.As(err, &errWT) {
		return err
	}

	stack := stackTrace(traceSkip)

	log.SetOutput(os.Stderr)
	log.Print(time.Now().Format(time.RFC3339), " ERROR\t", err, "\t", stack)

	return &errorWithTrace{
		error: err,
		trace: stack,
	}
}

func stackTrace(skip int) stack {
	pc := make([]uintptr, trackPrealloc)
	n := runtime.Callers(skip, pc)
	pc = pc[:n]

	frames := runtime.CallersFrames(pc)
	stack := make(stack, 0, n)

	for {
		frame, more := frames.Next()

		stack = append(stack, sFrame{filename: frame.File, method: frame.Function, line: frame.Line})

		if !more {
			break
		}
	}

	return stack
}
