package service

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func init() {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	cfg.EncoderConfig.EncodeCaller = nil
	cfg.EncoderConfig.TimeKey = ""
	Logger, _ = cfg.Build()
	defer Logger.Sync()
}
