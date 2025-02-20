package cache

import (
	"fmt"
	"testing"
	"time"

	"github.com/scys-devs/lib-go/conn"
)

func get(i int) {
	ResignCacheFromRedis(FormatKey("test:%v", i), 2, func() (string, error) {
		return fmt.Sprint(i), nil
	})
}

func TestUpdater_Process(t *testing.T) {
	conn.NewRedis("127.0.0.1", "6379")

	go func() {
		get(1)
		get(2)
	}()

	go func() {
		for {
			get(1)
			time.Sleep(time.Second)
		}
	}()

	ex := new(Updater)
	go func() {
		for {
			ex.Process(nil)
			time.Sleep(time.Second)
		}
	}()

	time.Sleep(time.Hour)
}
