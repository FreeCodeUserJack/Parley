package logger

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	envLogLevel  = "LOG_LEVEL"
	envLogOutput = "LOG_OUTPUT"
)

var (
	log logger
)

type logger struct {
	log *zap.Logger
}

type appLogger interface {
	// add mongoDB logger func signatures
}

func init() {
	logConfig := zap.Config{
		OutputPaths: []string{getOutput()},
		Level: zap.NewAtomicLevelAt(getLevel()),
		Encoding: "json",
		EncoderConfig: zapcore.EncoderConfig{
			LevelKey: "lvl",
			TimeKey: "time",
			MessageKey: "msg",
			EncodeTime: zapcore.ISO8601TimeEncoder,
			EncodeLevel: zapcore.LowercaseColorLevelEncoder,
			EncodeCaller: zapcore.ShortCallerEncoder,
		},
	}

	var err error
	if log.log, err = logConfig.Build(); err != nil {
		panic(err)
	}
}

func getLevel() zapcore.Level {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(envLogLevel))) {
	case "debug":
		return zap.DebugLevel
	case "info":
		return zap.InfoLevel
	case "error":
		return zap.ErrorLevel
	default:
		return zap.InfoLevel
	}
}

func getOutput() string {
	output := strings.TrimSpace(os.Getenv(envLogOutput))
	if output == "" {
		return "stdout"
	}
	return output
}

func GetLogger() appLogger {
	return log
}

func Info(msg string, tags ...string) {
	fieldTags := getFields(tags)

	log.log.Info(msg, fieldTags...)
	log.log.Sync()
}

func Error(msg string, err error, tags ...string) {
	fieldTags := getFields(tags)
	fieldTags = append(fieldTags, zap.NamedError("error", err))
	
	log.log.Error(msg, fieldTags...)
	log.log.Sync()
}

func getFields(tags []string) []zapcore.Field {
	var fieldTags []zap.Field

	for _, tag := range tags {
		input := strings.Split(tag, ":")
		fieldTags = append(fieldTags, zap.String(input[0], input[1]))
	}

	return fieldTags
}