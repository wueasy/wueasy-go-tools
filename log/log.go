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
	// Add traceId to logger name so it appears right after level/hostname/servicename
	// The standard zapcore.ConsoleEncoder outputs logger name before caller/message
	// Format we want: TIME LEVEL [HOSTNAME] [SERVICENAME] [TRACEID]

	// Append traceId to the entry's logger name
	if c.traceId != "" {
		if ent.LoggerName == "" {
			ent.LoggerName = "[" + c.traceId + "]"
		} else {
			ent.LoggerName = ent.LoggerName + " [" + c.traceId + "]"
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
	if traceId != "" {
		// Wrap the core to customize the output format of traceId
		logger := sugarLogger.Desugar()
		wrappedCore := &traceIdCore{
			Core:    logger.Core(),
			traceId: traceId,
		}
		// Return a new logger with the wrapped core
		return zap.New(wrappedCore,
			zap.AddCaller(),
			zap.AddCallerSkip(1),
			zap.AddStacktrace(zapcore.ErrorLevel)).Sugar()
	}
	return sugarLogger
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

	// 根据日志级别设置不同的颜色
	// var color string
	// switch l {
	// case zapcore.DebugLevel:
	// 	color = "\x1b[36m" // 青色
	// case zapcore.InfoLevel:
	// 	color = "\x1b[32m" // 绿色
	// case zapcore.WarnLevel:
	// 	color = "\x1b[33m" // 黄色
	// case zapcore.ErrorLevel:
	// 	color = "\x1b[31m" // 红色
	// default:
	// 	color = "\x1b[0m" // 默认颜色
	// }

	// 添加颜色和重置颜色代码
	// enc.AppendString(color + l.CapitalString() + "\x1b[0m" +
	// 	"  [" + serviceName + "] " +
	// 	"  [" + strconv.Itoa(pid) + "]")
	enc.AppendString(l.CapitalString() +
		" [" + hostname + "]" +
		" [" + serviceName + "]")
}

// initLogger 初始化日志的通用函数
func initLogger(rootPath string) {
	// 设置日志级别
	var zapLevel zapcore.Level
	if logConfig.Level == "" {
		zapLevel = zapcore.InfoLevel // 默认级别
	} else {
		switch strings.ToLower(logConfig.Level) {
		case "debug":
			zapLevel = zapcore.DebugLevel
		case "info":
			zapLevel = zapcore.InfoLevel
		case "warn":
			zapLevel = zapcore.WarnLevel
		case "error":
			zapLevel = zapcore.ErrorLevel
		default:
			zapLevel = zapcore.InfoLevel
		}
	}

	// 创建日志目录
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
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      zapcore.OmitKey, // 禁用 caller 输出
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    springBootStyleLevelEncoder, // 使用自定义 LevelEncoder
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
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

	atomicLevel = zap.NewAtomicLevel()
	atomicLevel.SetLevel(zapLevel)

	// 创建日志核心
	core := zapcore.NewCore(
		encoder,
		func() zapcore.WriteSyncer {
			ws := zapcore.NewMultiWriteSyncer(
				zapcore.AddSync(lumberJackLogger),
				zapcore.AddSync(os.Stdout),
			)
			if logConfig.Async {
				// 启用异步缓冲写入，缓冲区 1MB
				// 注意：在异步模式下会有延迟，如果需要即时看到日志，建议关闭 Async
				return &zapcore.BufferedWriteSyncer{
					WS:            ws,
					Size:          1024 * 1024,
					FlushInterval: 1 * time.Second, // 增加定时刷新机制，避免日志长时间滞留在缓冲区
				}
			}
			return ws
		}(),
		atomicLevel,
	)

	// 创建日志记录器，添加服务名称
	logger := zap.New(core,
		zap.AddStacktrace(zapcore.ErrorLevel), // 只在错误级别添加堆栈信息
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
	// 将字符串日志级别转换为zapcore.Level
	var zapLevel zapcore.Level
	switch strings.ToLower(level) {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	// 更新日志级别
	atomicLevel.SetLevel(zapLevel)
}

func UpdateServiceName(name string) {
	serviceName = name
}

// UpdateLogRotation 动态更新日志轮转配置
func UpdateLogRotation(maxSize, maxBackups, maxAge int) {
	if lumberJackLogger == nil {
		return
	}

	// 更新全局配置
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

	// 记录配置更新日志
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

// GetLogConfig 获取当前日志配置
func GetLogConfig() config.LogConfig {
	return logConfig
}
