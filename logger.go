package lib

import (
	"fmt"
	"github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"path"
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
	return zap.New(core, zap.AddCaller()).Sugar()
}

// 适用于可读日志
func GetConsoleLogger(name string) *zap.SugaredLogger {
	w := zapcore.AddSync(NewRotateLog(name, rotatelogs.WithRotationCount(90)))
	var enc = zap.NewDevelopmentEncoderConfig()
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(enc), w, zap.InfoLevel)
	return zap.New(core).Sugar()
}
