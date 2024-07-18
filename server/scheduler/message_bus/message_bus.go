package message_bus

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	jsoniter "github.com/json-iterator/go"
	"github.com/scys-devs/lib-go"
	"github.com/scys-devs/lib-go/server"
)

var Logger = lib.GetLogger("message_bus_data")

type Exec struct {
	BusName   string  // 队列名
	Max       int     // 并发最大尺度
	WhiteList []int64 // 通过user_id来判断
	Send      func(m DO) error
	OnSend    func(m DO)
	dao       MessageDAO
}

func (e *Exec) Desc() string {
	return "消息发送队列"
}

func (e *Exec) Processing() string {
	return fmt.Sprintf("剩余%v条消息", server.Scheduler.Len(e.BusName))
}

func (e *Exec) Name() string {
	return e.BusName
}

func (*Exec) NextDuration() int64 {
	return 0
}

func (e *Exec) Process(ctx *server.Context) (err error) {
	var curr = fmt.Sprint(time.Now().Unix())
	c := make(chan DO, 10)
	wg := new(sync.WaitGroup)
	defer func() {
		close(c)
	}()

	e.dao = mysqlDao{}
	for i := 0; i < e.Max; i++ {
		go e.send(c, wg)
	}

	// 需要外部定义好任务的可重入性
	server.Scheduler.GetBatch(e.BusName, curr, func(item redis.Z) {
		var m DO // 解析消息
		if err := jsoniter.UnmarshalFromString(item.Member.(string), &m); err != nil {
			Logger.Errorw("parse message failed", "raw", item.Member, "err", err)
			return
		}
		m.CanSent = e.canSend(m) // 判断是否可以发送
		wg.Add(1)
		c <- m
	})
	wg.Wait()

	return nil
}

func (e *Exec) send(c chan DO, wg *sync.WaitGroup) {
	for m := range c {
		if m.UserId == 0 {
			wg.Done()
			continue // 其实感觉是需要warn一下的
		}

		if m.CanSent {
			if err := e.Send(m); err != nil {
				Logger.Errorw("send message failed", "raw", m, "err", err)
			} else {
				m.Sent = 1
				e.OnSend(m)
			}
		}
		// 记录发送信息
		id := e.dao.Put(m)
		wg.Done()

		Logger.Infow("sent message", "id", id, "sent", m.Sent)
	}
}

func (e *Exec) canSend(m DO) bool {
	// 白名单限制
	if m.WhiteListLimit && lib.Index(e.WhiteList, m.UserId) < 0 {
		return false
	}
	return !IsLimit(m, e.dao)
}
