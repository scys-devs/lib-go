package message_bus

import (
	"fmt"
	"testing"
	"time"

	"github.com/scys-devs/lib-go/conn"
	"github.com/scys-devs/lib-go/server"
)

func TestNewPeriodLimit(t *testing.T) {
	day := NewPeriodLimit(86400, 1)
	start := (server.Scheduler.Now().Unix()+day.Phase)/day.Period*day.Period - day.Phase
	end := start + day.Period
	fmt.Println("today", start, end)

	day = NewPeriodLimit(3600, 1)
	start = (server.Scheduler.Now().Unix()+day.Phase)/day.Period*day.Period - day.Phase
	end = start + day.Period
	fmt.Println("hour", start, end)
}

func TestExecAsynq_Start(t *testing.T) {
	asynq := ExecAsynq{
		Send: func(m DO) error {
			fmt.Println("发送消息", m.UserId, m.Group)
			return nil
		},
	}
	asynq.Start()

	for i := 0; i < 10; i++ {
		Emit(0, DO{
			UserId: int64(i),
			Group:  "message_bus",
		})
	}

	time.Sleep(time.Hour)
}

func TestEsDao_Put(t *testing.T) {
	var address []string
	if conn.ENV == "local" || conn.ENV == "local-docker" {
		address = []string{"http://es-cn-tl32m4b92000arbl2.public.elasticsearch.aliyuncs.com:9200"}
	} else {
		address = []string{"http://es-cn-tl32m4b92000arbl2.elasticsearch.aliyuncs.com:9200"}
	}

	conn.NewES(address, "elastic", "kMoXM8LDW!m37MOO")
	dao := GetESDao()

	for i := 12926; i < 100000; i++ {
		dao.Put(DO{
			UserId:    int64(i),
			Group:     fmt.Sprintf("dkakwkd%v", i),
			GmtCreate: time.Now().Unix() - int64(86400*i),
		})
	}
}

func TestESDao_CountInPeriod(t *testing.T) {
	var address []string
	if conn.ENV == "local" || conn.ENV == "local-docker" {
		address = []string{"http://es-cn-tl32m4b92000arbl2.public.elasticsearch.aliyuncs.com:9200"}
	} else {
		address = []string{"http://es-cn-tl32m4b92000arbl2.elasticsearch.aliyuncs.com:9200"}
	}

	conn.NewES(address, "elastic", "kMoXM8LDW!m37MOO")
	dao := GetESDao()

	now := time.Now()
	//a := dao.CountInPeriod(0, time.Now().Unix(), 12938, "dkakwkd12938")
	//fmt.Println(a)
	b := dao.CountAll(-999, "party_check1357")
	fmt.Println(b)

	fmt.Println(time.Since(now))
}
