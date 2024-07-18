package throttle

import (
	"sync"
	"time"
)

type Throttle[T any] struct {
	mutex   *sync.Mutex
	cache   []T           // 缓存
	max     int           // 限制最大数量
	timeout int           // 限制最大缓存时间，单位ms
	notify  chan struct{} // 用于内部通知消费数据
	flush   func([]T)     // 实际消费行为由外部定义
}

func New[T any](max, timeout int, flush func([]T)) *Throttle[T] {
	t := &Throttle[T]{
		mutex:   new(sync.Mutex),
		cache:   make([]T, 0, max),
		max:     max - 1, // 留一位直接刷新
		timeout: timeout,
		notify:  make(chan struct{}),
		flush:   flush,
	}
	go t.wait()
	return t
}

func (t *Throttle[T]) wait() {
	for _ = range t.notify {
		time.Sleep(time.Duration(t.timeout) * time.Millisecond)
		t.Flush()
	}
}

func (t *Throttle[T]) Flush() {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if len(t.cache) == 0 {
		return
	}
	t.flush(t.cache)
	t.cache = make([]T, 0, t.max)
}

func (t *Throttle[T]) Put(item T) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	if len(t.cache) >= t.max {
		t.cache = append(t.cache, item)
		t.flush(t.cache)
		t.cache = make([]T, 0, t.max)
		return
	}
	t.cache = append(t.cache, item)

	select { // 尝试通知去消费，阻塞中就算了
	case t.notify <- struct{}{}:
	default:
	}
}
