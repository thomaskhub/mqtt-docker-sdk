package utils

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	LOGGER_MODE_DEBUG = "debug"
	LOGGER_MODE_PROD  = "prod"
)

type Logger struct {
	Instance *zap.Logger
}

func (log *Logger) Init(mode string) {
	var err error

	if mode == LOGGER_MODE_DEBUG {
		log.Instance, _ = zap.NewDevelopment()
		if err != nil {
			panic(err)
		}
	} else {
		log.Instance, err = zap.NewProduction()
		if err != nil {
			panic(err)
		}
	}
}

func (log *Logger) Debug(msg string, fields ...zapcore.Field) {
	log.Instance.Debug(msg, fields...)
}

func (log *Logger) Warn(msg string, fields ...zapcore.Field) {
	log.Instance.Warn(msg, fields...)
}

func (log *Logger) Error(msg string, fields ...zapcore.Field) {
	log.Instance.Error(msg, fields...)
}

func (log *Logger) Fatal(msg string, fields ...zapcore.Field) {
	log.Instance.Fatal(msg, fields...)
}
