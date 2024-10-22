package server

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/scys-devs/lib-go"
	"github.com/scys-devs/lib-go/conn"
)

var (
	Engine       = gin.New()
	Host         = ""
	RouterPrefix = ""
	RouterStatic = "/static"

	DaoLogger     = lib.GetLogger("dao")
	CtrlLogger    = lib.GetLogger("ctrl")
	ServiceLogger = lib.GetLogger("service")

	AccessLogger = lib.GetConsoleLogger("access")
)

// 用户拼接这个站的对外链接
func URL(path string) string {
	return Host + RouterPrefix + path
}

// Router 加载路由
func Router(cc ...Controller) {
	r := Engine.Group(RouterPrefix)
	for _, c := range cc {
		c.Register(r)
	}
}

func Run(port string) {
	Scheduler.Start()

	Engine.ForwardedByClientIP = true
	Engine.Use(gin.Recovery())
	Engine.Static(RouterPrefix+RouterStatic, "./static")
	gin.SetMode(gin.ReleaseMode)

	// 开发环境特殊配置
	if len(conn.ENV) > 0 {
		gin.SetMode(gin.DebugMode)
	}

	// TODO 测试下pm2和平滑重启的兼容性
	if err := Engine.Run(port); err != nil {
		fmt.Println(err)
	}
}
