package throttle

import (
	"fmt"
	"testing"
	"time"
)

type Item string

func TestNew(t *testing.T) {
	tmp := New[Item](10, 100, func(ll []Item) {
		for _, item := range ll {
			fmt.Println(item)
		}
	})
	tmp.Put("1")
	tmp.Put("2")
	time.Sleep(time.Second)
	tmp.Put("3")
	tmp.Put("4")
	time.Sleep(time.Minute)
}
