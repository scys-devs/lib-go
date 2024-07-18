package server

import (
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"strings"

	"github.com/foolin/goview"
	"github.com/foolin/goview/supports/ginview"
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

func NewRender(dir string, funcMap template.FuncMap) gin.HandlerFunc {
	partials, _ := fs.Glob(os.DirFS(dir), "partial/*.gohtml")
	// 去除partial后缀
	for i := range partials {
		partials[i] = strings.TrimSuffix(partials[i], ".gohtml")
	}
	return ginview.NewMiddleware(goview.Config{
		Root:         dir,
		Master:       "layout/base",
		Extension:    ".gohtml",
		Partials:     partials,
		Funcs:        funcMap,
		DisableCache: len(conn.ENV) > 0,
		Delims:       goview.Delims{Left: "{{", Right: "}}"},
	})
}

func Run(port string) {
	Scheduler.Start()

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
