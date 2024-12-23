package lib

import (
	"fmt"
	"os"
	"path"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewRotateLog(name string, option ...rotatelogs.Option) *rotatelogs.RotateLogs {
	wd, _ := os.Getwd()
	filename := fmt.Sprintf("%s/log/%s.log", wd, name)
	base := path.Dir(filename) // 支持多级目录，建议将调试日志迁入子文件夹
	if !FileExist(base) {
		_ = os.MkdirAll(base, 0755)
	}

	option = append(option, rotatelogs.WithLinkName(filename))
	rotator, err := rotatelogs.New(filename+"-%Y-%m-%d", option...)
	if err != nil {
		panic(err)
	}

	return rotator
}

func GetLogger(name string) *zap.SugaredLogger {
	w := zapcore.AddSync(NewRotateLog(name, rotatelogs.WithRotationCount(30)))
	var enc = zap.NewProductionEncoderConfig()
	var level = zap.DebugLevel
	core := zapcore.NewCore(zapcore.NewJSONEncoder(enc), w, level)
	// 创建带有动态时间字段的 logger
	logger := zap.New(core, zap.AddCaller()).With(
		zap.String("formatted_time", time.Now().Format("2006-01-02 15:04:05")),
	)

	// 添加动态时间字段
	logger = logger.WithOptions(zap.Hooks(func(entry zapcore.Entry) error {
		logger.Core().With([]zapcore.Field{
			zap.String("formatted_time", entry.Time.Format("2006-01-02 15:04:05")),
		})
		return nil
	}))

	return logger.Sugar()
}

// 适用于可读日志
func GetConsoleLogger(name string) *zap.SugaredLogger {
	w := zapcore.AddSync(NewRotateLog(name, rotatelogs.WithRotationCount(90)))
	var enc = zap.NewDevelopmentEncoderConfig()
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(enc), w, zap.InfoLevel)
	return zap.New(core).Sugar()
}
