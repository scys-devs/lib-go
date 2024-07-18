package message_bus

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hibiken/asynq"
	jsoniter "github.com/json-iterator/go"
	"github.com/scys-devs/lib-go"
	"github.com/scys-devs/lib-go/conn"
)

var client *asynq.Client

type ExecAsynq struct {
	Addr      []string // redis服务
	WhiteList []int64  // 通过user_id来判断
	Send      func(m DO) error
	OnMessage func(m DO)
	OnSend    func(m DO)
	DAO       MessageDAO
}

func (e *ExecAsynq) Start() {
	var addr = "localhost:6379"
	if conn.ENV == "local-docker" {
		addr = fmt.Sprintf("%v:6379", conn.HostDockerInternal)
	}
	if len(e.Addr) > 0 {
		addr = e.Addr[0]
	}
	if e.DAO == nil {
		//e.DAO = mysqlDao{}
		e.DAO = GetESDao()
	}

	db := 4 // 不用默认数据库
	if len(conn.ENV) > 0 {
		db = 5
	}
	// 初始化客户端
	client = asynq.NewClient(asynq.RedisClientOpt{Addr: addr, DB: db})
	// 初始化服务端
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: addr, DB: db},
		asynq.Config{
			Concurrency: 10,
			Queues: map[string]int{
				"critical": 6,
				"default":  3,
				"low":      1,
			}},
	)

	handler := func(ctx context.Context, t *asynq.Task) error {
		var m DO
		if err := jsoniter.Unmarshal(t.Payload(), &m); err != nil {
			Logger.Errorw("parse message failed", "raw", string(t.Payload()), "err", err)
			return fmt.Errorf("parse message failed: %w", asynq.SkipRetry)
		}
		if e.OnMessage != nil {
			e.OnMessage(m)
		}

		if m.UserId == 0 {
			//Logger.Errorw("not found user", "raw", string(t.Payload()))
			//return fmt.Errorf("not found user: %w", asynq.SkipRetry)
			m.SentErr = "not found user"
		} else if e.canSend(m) {
			if err := e.Send(m); err != nil {
				Logger.Errorw("send message failed", "raw", string(t.Payload()), "err", err)
				m.SentErr = err.Error()
			} else {
				m.Sent = 1
				e.DAO.Put(m) // 没发送成功的就不写入了，未发送成功走日志吧
			}
		}
		if e.OnSend != nil {
			e.OnSend(m)
		}

		return nil
	}

	if err := srv.Start(asynq.HandlerFunc(handler)); err != nil {
		log.Fatal(err)
	}
}

func (e *ExecAsynq) canSend(m DO) bool {
	// 白名单限制
	if m.WhiteListLimit && lib.Index(e.WhiteList, m.UserId) < 0 {
		return false
	}
	return !IsLimit(m, e.DAO)
}

// 默认，立即发送消息
// 防止缺少 user_id
func Emit(id int64, m DO) {
	m.UserId = id
	payload, _ := jsoniter.Marshal(m)
	if _, err := client.Enqueue(
		asynq.NewTask(m.GroupID(), payload),
		asynq.Queue("critical"),
	); err != nil {
		Logger.Errorw("emit", "message", m, "err", err)
	}
	return
}

func EmitAt(id int64, m DO, after int64) {
	m.UserId = id
	payload, _ := jsoniter.Marshal(m)
	if _, err := client.Enqueue(
		asynq.NewTask(m.GroupID(), payload),
		asynq.Queue("critical"),
		asynq.ProcessAt(time.Now().Add(time.Duration(after)*time.Second)),
	); err != nil {
		Logger.Errorw("emit", "message", m, "err", err)
	}
	return
}

// 默认延迟的批量消息都是低优先级
func EmitLow(id int64, m DO, after int64) {
	m.UserId = id
	payload, _ := jsoniter.Marshal(m)

	if _, err := client.Enqueue(
		asynq.NewTask(m.GroupID(), payload),
		asynq.ProcessAt(time.Now().Add(time.Duration(after)*time.Second)),
	); err != nil {
		Logger.Errorw("emit low", "message", m, "err", err)
	}
	return
}
