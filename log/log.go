package log

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wueasy/wueasy-go-tools/config"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	sugarLogger      *zap.SugaredLogger
	lumberJackLogger *lumberjack.Logger
	levelFileLoggers []*lumberjack.Logger
	logConfig        config.LogConfig
	atomicLevel      zap.AtomicLevel
	serviceName      string
	hostname         string
)

type contextKey string

const TraceIdKey contextKey = "traceId"

// NewContext 注入 traceId 到 context
func NewContext(ctx context.Context, traceId string) context.Context {
	return context.WithValue(ctx, TraceIdKey, traceId)
}

// FromContext 从 context 获取 traceId
func FromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if traceId, ok := ctx.Value(TraceIdKey).(string); ok {
		return traceId
	}
	return ""
}

// Custom trace id Core implementation to format traceId like spring boot
type traceIdCore struct {
	zapcore.Core
	traceId string
}

func (c *traceIdCore) With(fields []zapcore.Field) zapcore.Core {
	return &traceIdCore{
		Core:    c.Core.With(fields),
		traceId: c.traceId,
	}
}

func (c *traceIdCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *traceIdCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	if c.traceId != "" {
		if ent.LoggerName == "" {
			ent.LoggerName = "[" + c.traceId + "]"
		} else {
			ent.LoggerName = ent.LoggerName + " [" + c.traceId + "]"
		}
	}

	return c.Core.Write(ent, fields)
}

type sensitiveWriteCore struct {
	zapcore.Core
}

func (c *sensitiveWriteCore) With(fields []zapcore.Field) zapcore.Core {
	return &sensitiveWriteCore{
		Core: c.Core.With(fields),
	}
}

func (c *sensitiveWriteCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *sensitiveWriteCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	ent.Message = DesensitizeText(ent.Message)
	for i := range fields {
		if fields[i].Type == zapcore.StringType {
			fields[i].String = desensitizeFieldValue(fields[i].Key, fields[i].String)
		}
	}
	return c.Core.Write(ent, fields)
}

// Ctx 返回一个带有 traceId 的 SugaredLogger
func Ctx(ctx context.Context) *zap.SugaredLogger {
	if ctx == nil || sugarLogger == nil {
		return sugarLogger
	}

	traceId := FromContext(ctx)
	enabled := IsSensitiveEnabled()

	if traceId == "" && !enabled {
		return sugarLogger
	}

	logger := sugarLogger.Desugar()
	core := logger.Core()

	if traceId != "" {
		core = &traceIdCore{
			Core:    core,
			traceId: traceId,
		}
	}

	if enabled {
		core = &sensitiveWriteCore{Core: core}
	}

	return zap.New(core,
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel)).Sugar()
}

// IsDebugEnabled 判断是否启用Debug级别日志
func IsDebugEnabled() bool {
	return atomicLevel.Level() <= zapcore.DebugLevel
}

// IsInfoEnabled 判断是否启用Info级别日志
func IsInfoEnabled() bool {
	return atomicLevel.Level() <= zapcore.InfoLevel
}

// IsWarnEnabled 判断是否启用Warn级别日志
func IsWarnEnabled() bool {
	return atomicLevel.Level() <= zapcore.WarnLevel
}

// IsErrorEnabled 判断是否启用Error级别日志
func IsErrorEnabled() bool {
	return atomicLevel.Level() <= zapcore.ErrorLevel
}

// GetLevel 获取当前日志级别
func GetLevel() zapcore.Level {
	return atomicLevel.Level()
}

// LumberJackLogger 获取全局的lumberJackLogger实例
func LumberJackLogger() *lumberjack.Logger {
	return lumberJackLogger
}

// Init 初始化日志
func Init(rootPath string, conf config.LogConfig) {

	if sugarLogger != nil {
		UpdateLogLevel(conf.Level)
		UpdateLogRotation(conf.MaxSize, conf.MaxBackups, conf.MaxAge)
		return
	}

	// 设置默认配置
	logConfig = config.LogConfig{
		Level:      "info",
		MaxSize:    100,
		MaxBackups: 100,
		MaxAge:     100,
		Async:      false,
	}
	// 如果传入了配置，则使用传入的配置覆盖默认配置
	if conf.Level != "" {
		logConfig.Level = conf.Level
	}
	if conf.MaxSize > 0 {
		logConfig.MaxSize = conf.MaxSize
	}
	if conf.MaxBackups > 0 {
		logConfig.MaxBackups = conf.MaxBackups
	}
	if conf.MaxAge > 0 {
		logConfig.MaxAge = conf.MaxAge
	}
	if conf.Async {
		logConfig.Async = conf.Async
	}
	if len(conf.LevelFiles) > 0 {
		logConfig.LevelFiles = conf.LevelFiles
	}

	// 获取主机名
	var err error
	hostname, err = os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	initLogger(rootPath)

	UpdateSensitiveConfig(conf.Sensitive)
}

// 自定义 LevelEncoder 实现 Spring Boot 风格格式
func springBootStyleLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	var sb strings.Builder
	sb.WriteString(l.CapitalString())
	sb.WriteString(" [")
	sb.WriteString(hostname)
	sb.WriteByte(']')
	if serviceName != "" {
		sb.WriteString(" [")
		sb.WriteString(serviceName)
		sb.WriteByte(']')
	}
	enc.AppendString(sb.String())
}

// initLogger 初始化日志的通用函数
func initLogger(rootPath string) {
	atomicLevel = zap.NewAtomicLevel()
	atomicLevel.SetLevel(parseLogLevel(logConfig.Level))
	logDir := filepath.Join(rootPath, "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		fmt.Printf("创建日志目录失败: %v\n", err)
	}

	// 创建日志文件
	logFile := filepath.Join(logDir, "app.log")
	lumberJackLogger = &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    logConfig.MaxSize,    // 日志文件最大大小（MB）
		MaxBackups: logConfig.MaxBackups, // 最大保留的旧日志文件数量
		MaxAge:     logConfig.MaxAge,     // 最大保留天数
		Compress:   true,                 // 是否压缩/归档旧文件
	}

	// 配置 EncoderConfig
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:          "time",
		LevelKey:         "level",
		NameKey:          "logger",
		CallerKey:        zapcore.OmitKey,
		FunctionKey:      zapcore.OmitKey,
		MessageKey:       "msg",
		StacktraceKey:    "stacktrace",
		LineEnding:       zapcore.DefaultLineEnding,
		EncodeLevel:      springBootStyleLevelEncoder,
		EncodeTime:       zapcore.ISO8601TimeEncoder,
		EncodeDuration:   zapcore.StringDurationEncoder,
		EncodeCaller:     zapcore.ShortCallerEncoder,
		ConsoleSeparator: " ",
	}

	encoderConfig.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
	}

	// 自定义 LoggerName 编码器（不带冒号），以便像 spring boot 那样展示 traceId
	encoderConfig.EncodeName = func(s string, enc zapcore.PrimitiveArrayEncoder) {
		if s != "" {
			enc.AppendString(s)
		}
	}

	// 创建日志编码器
	encoder := zapcore.NewConsoleEncoder(encoderConfig)

	// 创建按级别输出的独立文件 logger
	levelFileLoggers = nil
	var levelFileCores []zapcore.Core
	for _, lfc := range logConfig.LevelFiles {
		if lfc.Level == "" {
			continue
		}
		threshold := parseLogLevel(lfc.Level)
		filename := lfc.Filename
		if filename == "" {
			filename = "app-" + strings.ToLower(lfc.Level) + ".log"
		}
		lfLogger := &lumberjack.Logger{
			Filename:   filepath.Join(logDir, filename),
			MaxSize:    pickInt(lfc.MaxSize, logConfig.MaxSize),
			MaxBackups: pickInt(lfc.MaxBackups, logConfig.MaxBackups),
			MaxAge:     pickInt(lfc.MaxAge, logConfig.MaxAge),
			Compress:   true,
		}
		levelFileLoggers = append(levelFileLoggers, lfLogger)

		levelFileCores = append(levelFileCores, zapcore.NewCore(
			encoder,
			zapcore.AddSync(lfLogger),
			zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
				return lvl >= threshold
			}),
		))
	}

	// 创建主 Core：全级别 → app.log + stdout
	mainCore := zapcore.NewCore(
		encoder,
		func() zapcore.WriteSyncer {
			ws := zapcore.NewMultiWriteSyncer(
				zapcore.AddSync(lumberJackLogger),
				zapcore.AddSync(os.Stdout),
			)
			if logConfig.Async {
				return &zapcore.BufferedWriteSyncer{
					WS:            ws,
					Size:          1024 * 1024,
					FlushInterval: 1 * time.Second,
				}
			}
			return ws
		}(),
		atomicLevel,
	)

	// 合并所有 Core
	cores := append([]zapcore.Core{mainCore}, levelFileCores...)
	var core zapcore.Core
	if len(cores) == 1 {
		core = cores[0]
	} else {
		core = zapcore.NewTee(cores...)
	}

	// 创建日志记录器，添加服务名称
	logger := zap.New(core,
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	sugarLogger = logger.Sugar()
}

// Sync 同步缓存日志到底层写入器
func Sync() {
	if sugarLogger != nil {
		_ = sugarLogger.Sync()
	}
}

func UpdateLogLevel(level string) {
	atomicLevel.SetLevel(parseLogLevel(level))
}

func UpdateServiceName(name string) {
	serviceName = name
}

// UpdateLogRotation 动态更新日志轮转配置
func UpdateLogRotation(maxSize, maxBackups, maxAge int) {
	if lumberJackLogger == nil {
		return
	}

	if maxSize > 0 {
		logConfig.MaxSize = maxSize
		lumberJackLogger.MaxSize = maxSize
	}
	if maxBackups > 0 {
		logConfig.MaxBackups = maxBackups
		lumberJackLogger.MaxBackups = maxBackups
	}
	if maxAge > 0 {
		logConfig.MaxAge = maxAge
		lumberJackLogger.MaxAge = maxAge
	}

	// 同步更新按级别输出文件
	for _, lf := range levelFileLoggers {
		if lf == nil {
			continue
		}
		if maxSize > 0 {
			lf.MaxSize = maxSize
		}
		if maxBackups > 0 {
			lf.MaxBackups = maxBackups
		}
		if maxAge > 0 {
			lf.MaxAge = maxAge
		}
	}

	if IsDebugEnabled() {
		sugarLogger.Debugf("日志轮转配置已更新 - MaxSize: %dMB, MaxBackups: %d, MaxAge: %d天",
			lumberJackLogger.MaxSize, lumberJackLogger.MaxBackups, lumberJackLogger.MaxAge)
	}
}

// UpdateMaxSize 动态更新日志文件最大大小
func UpdateMaxSize(maxSize int) {
	if lumberJackLogger == nil || maxSize <= 0 {
		return
	}

	logConfig.MaxSize = maxSize
	lumberJackLogger.MaxSize = maxSize
	sugarLogger.Infof("日志文件最大大小已更新为: %dMB", maxSize)
}

// UpdateMaxBackups 动态更新最大保留的旧日志文件数量
func UpdateMaxBackups(maxBackups int) {
	if lumberJackLogger == nil || maxBackups <= 0 {
		return
	}

	logConfig.MaxBackups = maxBackups
	lumberJackLogger.MaxBackups = maxBackups
	sugarLogger.Infof("最大保留旧日志文件数量已更新为: %d", maxBackups)
}

// UpdateMaxAge 动态更新最大保留天数
func UpdateMaxAge(maxAge int) {
	if lumberJackLogger == nil || maxAge <= 0 {
		return
	}

	logConfig.MaxAge = maxAge
	lumberJackLogger.MaxAge = maxAge
	sugarLogger.Infof("日志最大保留天数已更新为: %d天", maxAge)
}

// parseLogLevel 将字符串日志级别转换为zapcore.Level
func parseLogLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// pickInt 优先使用自定义值，为0则回退到默认值
func pickInt(custom, defaultVal int) int {
	if custom > 0 {
		return custom
	}
	return defaultVal
}

// GetLogConfig 获取当前日志配置
func GetLogConfig() config.LogConfig {
	return logConfig
}
