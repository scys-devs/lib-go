package conn

import (
	"os"
	"path"
)

var (
	ENV                = os.Getenv("ENV") // 使用小写env和pm2冲突
	PREFIX             string
	AliyunID           = ""
	AliyunSecret       = ""
	HostDockerInternal = "host.docker.internal" // docker容器访问宿主机host
)

func init() {
	dir, _ := os.Getwd()
	PREFIX = path.Base(dir)
}
