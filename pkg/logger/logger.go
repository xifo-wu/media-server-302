package logger

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Log *zap.Logger

func Init() *zap.Logger {
	logFilePath := fmt.Sprintf("%s/%s.log", "./logs", time.Now().Format("2006-01-02"))

	// 配置日志文件滚动
	logWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logFilePath, // 指定日志文件路径
		MaxSize:    100,         // 日志文件的最大大小（MB）
		MaxBackups: 3,           // 最多保留的旧日志文件数
		MaxAge:     2,           // 保留旧日志文件的最长天数
		LocalTime:  true,        // 使用本地时间
		// Compress:   true,        // 是否压缩旧日志文件
	})

	// 设置中国时区
	chnLoc, _ := time.LoadLocation("Asia/Shanghai")

	// 自定义时间格式化函数
	customTimeEncoder := func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.In(chnLoc).Format("2006-01-02 15:04:05"))
	}

	// 自定义编码器配置
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = customTimeEncoder

	// 设置日志级别和编码器
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		logWriter,
		zap.NewAtomicLevelAt(zap.InfoLevel), // 设置日志级别
	)

	// 创建 Logger
	Log = zap.New(core, zap.AddCaller())
	return Log
}
