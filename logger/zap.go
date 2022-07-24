package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var zapLog *zap.SugaredLogger

func init() {
	config := zap.NewProductionConfig()
	config.Encoding = "console"
	// prodConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder
	config.EncoderConfig.TimeKey = ""
	config.EncoderConfig.NameKey = ""
	// config.EncoderConfig.StacktraceKey = "" // to hide stacktrace info
	logger, err := config.Build()

	if err != nil {
		panic(err)
	}
	defer func() { // flushes buffer, if any
		err = logger.Sync()
	}()
	if err != nil {
		panic(err)
	}

	zapLog = logger.Sugar()

}
func Info(args ...interface{}) {
	zapLog.Info(args)
}

func Debug(args ...interface{}) {
	zapLog.Debug(args)
}
func Debugf(template string, args ...interface{}) {
	zapLog.Debugf(template, args)
}

func Error(args ...interface{}) {
	zapLog.Error(args)
}
func Errorf(template string, args ...interface{}) {
	zapLog.Errorf(template, args)
}

func Fatal(args ...interface{}) {
	zapLog.Fatal(args)
}
func Fatalf(template string, args ...interface{}) {
	zapLog.Fatalf(template, args)
}
