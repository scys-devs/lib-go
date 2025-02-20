package cache

import (
	"github.com/scys-devs/lib-go/server"
)

type Updater struct{}

func (e *Updater) Desc() string {
	return "缓存更新器"
}

func (e *Updater) Processing() string {
	return ""
}

func (*Updater) Name() string {
	return "cache_update"
}

func (*Updater) NextDuration() int64 {
	return 0
}

func (e *Updater) Process(ctx *server.Context) (err error) {
	CacheUpdater.Update()
	return
}
