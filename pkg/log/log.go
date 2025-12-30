package log

import (
	"fmt"
	"os"
	"runtime"
	"strconv"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const callerSkip = 8

var l *logger

func init() {
	l = newLogger(zapcore.InfoLevel, "console") // "json" or "console"

	zap.ReplaceGlobals(l.Logger)

	if _, err := zap.RedirectStdLogAt(l.Logger, zapcore.InfoLevel); err != nil {
		panic(err)
	}
}

type logger struct {
	logLevel    zapcore.Level
	logEncoding string

	*zap.Logger
}

func newLogger(logLevel zapcore.Level, encoding string) *logger {
	encoder, err := getEncoder(encoding)
	if err != nil {
		panic(fmt.Sprintf("failed to parse encoder: %v", err))
	}

	zapLogger := zap.New(zapcore.NewTee(
		zapcore.NewCore(
			encoder,
			zapcore.Lock(os.Stdout),
			zap.LevelEnablerFunc(func(level zapcore.Level) bool {
				return level >= logLevel && level < zapcore.ErrorLevel
			}),
		),
		zapcore.NewCore(
			encoder,
			zapcore.Lock(os.Stderr),
			zap.LevelEnablerFunc(func(level zapcore.Level) bool {
				return level >= zapcore.ErrorLevel
			}),
		),
	))

	zapLogger = zapLogger.WithOptions(zap.AddCaller())

	return &logger{
		logLevel:    logLevel,
		logEncoding: encoding,
		Logger:      zapLogger,
	}
}

func getEncoder(encoding string) (zapcore.Encoder, error) {
	encoderConfig := zapcore.EncoderConfig{
		MessageKey: "message",

		LevelKey:    "level",
		EncodeLevel: zapcore.CapitalLevelEncoder,

		TimeKey:    "time",
		EncodeTime: zapcore.ISO8601TimeEncoder,

		CallerKey:      "caller",
		EncodeCaller:   customEncodeCaller,
		EncodeDuration: zapcore.SecondsDurationEncoder,
	}

	switch encoding {
	case "json":
		return zapcore.NewJSONEncoder(encoderConfig), nil
	case "console":
		return zapcore.NewConsoleEncoder(encoderConfig), nil
	default:
		return nil, fmt.Errorf("failed to find encoder: %q", encoding)
	}
}

func Debug(msg string, fields ...zap.Field) { l.Debug(msg, fields...) }
func Info(msg string, fields ...zap.Field)  { l.Info(msg, fields...) }
func Warn(msg string, fields ...zap.Field)  { l.Warn(msg, fields...) }
func Error(msg string, fields ...zap.Field) { l.Error(msg, fields...) }
func Fatal(msg string, fields ...zap.Field) { l.Fatal(msg, fields...) }
func Panic(msg string, fields ...zap.Field) { l.Panic(msg, fields...) }
func Sync() error {
	return l.Sync()
}

func customEncodeCaller(_ zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
	file, _, line := findCaller()
	enc.AppendString(file + ":" + strconv.Itoa(line))
}

func findCaller() (string, string, int) {
	var (
		pc       uintptr
		file     string
		function string
		line     int
	)

	pc, file, line = getCaller(callerSkip)

	if pc != 0 {
		frames := runtime.CallersFrames([]uintptr{pc})
		frame, _ := frames.Next()
		function = frame.Function
	}

	return file, function, line
}

func getCaller(skip int) (uintptr, string, int) {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return 0, "", 0
	}

	n := 0
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			n++
			if n >= 2 {
				file = file[i+1:]
				break
			}
		}
	}

	return pc, file, line
}
