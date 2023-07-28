package log

import (
	"os"
	"strings"

	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	conf                 zap.Config
	glevel               = zap.NewAtomicLevel() // global level
	defaultEncoderConfig zapcore.EncoderConfig  // default EncoderConfig
)

var (
	logger *zap.Logger
	sugar  *zap.SugaredLogger
)

const (
	timeLayout     = "2006-01-02T15:04:05.000"
	defaultLogName = "server.log"
)

// init default logger and sugar
func init() {
	conf = zap.NewProductionConfig()
	conf.Level = glevel
	conf.DisableStacktrace = true
	conf.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(timeLayout)
	// conf.OutputPaths = append(conf.OutputPaths, "server.log")

	logger, _ = conf.Build(zap.AddCaller(), zap.AddCallerSkip(1))
	sugar = logger.Sugar()
}

// init package variable
func init() {
	defaultEncoderConfig = zap.NewProductionEncoderConfig()
	defaultEncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(timeLayout)
}

// no export zap package

// // Logger return global logger
// func Logger() *zap.Logger {
// 	return logger.WithOptions(zap.AddCallerSkip(-1))
// }

// // Sugar return global sugarlogger
// func Sugar() *zap.SugaredLogger {
// 	return Logger().Sugar()
// }

// Config logger config
type Config struct {
	Level            string
	File             *lumberjack.Logger
	EnabledErrorFile bool // will create file-error if File is not nil
}

// Build new logger
func (c *Config) Build() (err error) {
	if c.Level != "" {
		SetLevelString(c.Level)
	}

	if c.File == nil {
		return newLogger()
	}
	if !c.EnabledErrorFile {
		return newLoggerWithFile(c.File)
	}
	return newLoggerWithErrorFile(c.File)
}

var DefaultLogFileCfg = &lumberjack.Logger{Filename: defaultLogName}

// newLogger log to console
func newLogger() (err error) {
	writesyncer := zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout))

	core := zapcore.NewCore(zapcore.NewJSONEncoder(defaultEncoderConfig), writesyncer, glevel)
	logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	sugar = logger.Sugar()
	return nil
}

// newLoggerWithFile and newLoggerWithErrorFile implemented in 2 different ways
// newLoggerWithFile implemented by multi write syncer
// newLoggerWithErrorFile implemented by core tee

// newLoggerWithFile log to console and file if filecfg is not nil
func newLoggerWithFile(filecfg *lumberjack.Logger) (err error) {
	if filecfg == nil || filecfg.Filename == "" {
		panic("log filecfg is nil or Filename field is empty")
	}

	writesyncer := zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(filecfg))
	core := zapcore.NewCore(zapcore.NewJSONEncoder(defaultEncoderConfig), writesyncer, glevel)
	logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	sugar = logger.Sugar()
	return nil
}

// newLoggerWithErrorFile log to console, file and file-error if filecfg is not nil
func newLoggerWithErrorFile(filecfg *lumberjack.Logger) (err error) {
	if filecfg == nil || filecfg.Filename == "" {
		panic("log filecfg is nil or Filename field is empty")
	}

	var cores = []zapcore.Core{}
	cores = append(cores, newCoreToConsole())
	cores = append(cores, newCoreToFile(filecfg))
	cores = append(cores, newCoreToFileErrorLevel(filecfg))

	logger = zap.New(zapcore.NewTee(cores...), zap.AddCaller(), zap.AddCallerSkip(1))
	sugar = logger.Sugar()
	return nil
}

// newCoreToConsole write to console
func newCoreToConsole() zapcore.Core {
	writesyncer := zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout))
	return zapcore.NewCore(zapcore.NewJSONEncoder(defaultEncoderConfig), writesyncer, glevel)
}

// newCoreToFile write to file
func newCoreToFile(filecfg *lumberjack.Logger) zapcore.Core {
	if filecfg.Filename == "" {
		filecfg.Filename = defaultLogName
	}

	writesyncer := zapcore.NewMultiWriteSyncer(zapcore.AddSync(filecfg))
	return zapcore.NewCore(zapcore.NewJSONEncoder(defaultEncoderConfig), writesyncer, glevel)

}

// newCoreToFileErrorLevel write to file-error
func newCoreToFileErrorLevel(filecfg *lumberjack.Logger) zapcore.Core {
	errcfg := &lumberjack.Logger{
		Filename:   wrapFileNameWithError(filecfg.Filename),
		MaxSize:    filecfg.MaxSize,
		MaxAge:     filecfg.MaxAge,
		MaxBackups: filecfg.MaxBackups,
		Compress:   filecfg.Compress,
	}

	writesyncer := zapcore.NewMultiWriteSyncer(zapcore.AddSync(errcfg))

	highPriority := zap.LevelEnablerFunc(func(lev zapcore.Level) bool { //error级别
		return lev >= zap.ErrorLevel
	})

	return zapcore.NewCore(zapcore.NewJSONEncoder(defaultEncoderConfig), writesyncer, highPriority)
}

// Level wrap internal/pkg/log Level
type Level zapcore.Level

const (
	// DebugLevel logs are typically voluminous, and are usually disabled in
	// production.
	DebugLevel Level = iota - 1
	// InfoLevel is the default logging priority.
	InfoLevel
	// WarnLevel logs are more important than Info, but don't need individual
	// human review.
	WarnLevel
	// ErrorLevel logs are high-priority. If an application is running smoothly,
	// it shouldn't generate any error-level logs.
	ErrorLevel
	// DPanicLevel logs are particularly important errors. In development the
	// logger panics after writing the message.
	DPanicLevel
	// PanicLevel logs a message, then panics.
	PanicLevel
	// FatalLevel logs a message, then calls os.Exit(1).
	FatalLevel

	_minLevel = DebugLevel
	_maxLevel = FatalLevel

	// InvalidLevel is an invalid value for Level.
	//
	// Core implementations may panic if they see messages of this level.
	InvalidLevel = _maxLevel + 1
)

// SetLevel set global level
func SetLevel(level Level) {
	glevel.SetLevel(zapcore.Level(level))
}

// SetLevelString set global level by string
func SetLevelString(levelstr string) (err error) {
	var level zapcore.Level
	if err = level.Set(levelstr); err != nil {
		return err
	}
	glevel.SetLevel(level)
	return nil
}

// Debug debug level message
func Debug(args ...interface{}) {
	sugar.Debug(args...)
}

// Info info level message
func Info(args ...interface{}) {
	sugar.Info(args...)
}

// Warn warn level message
func Warn(args ...interface{}) {
	sugar.Warn(args...)
}

// Error error level message
func Error(args ...interface{}) {
	sugar.Error(args...)
}

// Panic panic level message
func Panic(args string) {
	sugar.Panic(args)
}

// Fatal fatal level message
func Fatal(args ...interface{}) {
	sugar.Fatal(args...)
}

// Debugf debug level message by template
func Debugf(template string, args ...interface{}) {
	sugar.Debugf(template, args...)
}

// Infof info level message by template
func Infof(template string, args ...interface{}) {
	sugar.Infof(template, args...)
}

// Warnf warn level message by template
func Warnf(template string, args ...interface{}) {
	sugar.Warnf(template, args...)
}

// Errorf error level message by template
func Errorf(template string, args ...interface{}) {
	sugar.Errorf(template, args...)
}

// Panicf panic level message by template
func Panicf(template string, args ...interface{}) {
	sugar.Panicf(template, args...)
}

// Fatalf fatal level message by template
func Fatalf(template string, args ...interface{}) {
	sugar.Fatalf(template, args...)
}

// Sync flushes any buffered log entries.
func Sync() (err error) {
	return multierr.Append(logger.Sync(), sugar.Sync())
}

// wrapFileNameWithError wrap filename with '-error' suffix
// example 'server.log' return 'server-error.log'
func wrapFileNameWithError(file string) string {
	index := strings.LastIndex(file, ".")
	if index < 0 {
		return file + "-error"
	}

	return file[:index] + "-error" + file[index:]
}

// With return *Logger with fields
// Logger is wrap for some scenarios that we need reused some fields
// But it can lead to performance degradation, so try not to call on the http entrance as much as possible
func With(args ...interface{}) *Logger {
	return &Logger{base: sugar.With(args...)}
}

// Logger wrap logger
type Logger struct {
	base *zap.SugaredLogger
}

// With return *Logger with fields
func (l *Logger) With(args ...interface{}) *Logger {
	return &Logger{base: l.base.With(args...)}
}

// Debug debug level message
func (l *Logger) Debug(args ...interface{}) {
	sugar.Debug(args...)
}

// Info info level message
func (l *Logger) Info(args ...interface{}) {
	l.base.Info(args...)
}

// Warn warn level message
func (l *Logger) Warn(args ...interface{}) {
	l.base.Warn(args...)
}

// Error error level message
func (l *Logger) Error(args ...interface{}) {
	l.base.Error(args...)
}

// Panic panic level message
func (l *Logger) Panic(args string) {
	l.base.Panic(args)
}

// Fatal fatal level message
func (l *Logger) Fatal(args ...interface{}) {
	l.base.Fatal(args...)
}

// Debugf debug level message by template
func (l *Logger) Debugf(template string, args ...interface{}) {
	l.base.Debugf(template, args...)
}

// Infof info level message by template
func (l *Logger) Infof(template string, args ...interface{}) {
	l.base.Infof(template, args...)
}

// Warnf warn level message by template
func (l *Logger) Warnf(template string, args ...interface{}) {
	l.base.Warnf(template, args...)
}

// Errorf error level message by template
func (l *Logger) Errorf(template string, args ...interface{}) {
	l.base.Errorf(template, args...)
}

// Panicf panic level message by template
func (l *Logger) Panicf(template string, args ...interface{}) {
	l.base.Panicf(template, args...)
}

// Fatalf fatal level message by template
func (l *Logger) Fatalf(template string, args ...interface{}) {
	l.base.Fatalf(template, args...)
}
