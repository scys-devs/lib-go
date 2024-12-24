package cache

import (
	"context"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/scys-devs/lib-go/conn"
	"sync"
	"time"
)

type Key struct {
	Name   string
	Format string
}

// FormatKey 暂时只支持%v的格式化
func FormatKey(format string, a ...any) Key {
	return Key{
		Format: format,
		Name:   fmt.Sprintf(format, a...),
	}
}

type CacheItem struct {
	Last   int64               // 访问时间
	NewFun func() (any, error) // 待更新的闭包函数
}

type Cache struct {
	sync.RWMutex
	Keys     map[string]CacheItem // key名和访问时间
	Duration int64
}

// 缓存十倍的时间，但最多2天
func (cache *Cache) Expire() int64 {
	expire := cache.Duration * 10
	if expire > 86400*2 {
		expire = 86400 * 2
	}
	return expire
}

func (cache *Cache) Update() error {
	cache.RLock()
	defer cache.RUnlock()
	// 遍历key底下的值，然后判断是否要刷新
	now := time.Now().Unix()
	for name, item := range cache.Keys {
		//fmt.Println("开始检测缓存", name)
		if now-item.Last > cache.Duration {
			//fmt.Println(name, "太久不访问了")
			continue
		}
		// 判断下缓存剩余时间
		expire := conn.GetRedis().TTL(context.TODO(), conn.GetRedisKey(name)).Val()
		expireTS := cache.Expire() - int64(expire/time.Second) // 真正消耗的时间
		//fmt.Println(name, "已缓存时间", expireTS)
		if expireTS < cache.Duration {
			continue
		}
		// 准备更新
		d, err := item.NewFun()
		if err != nil {
			return err
		}
		s, _ := jsoniter.MarshalToString(d)
		// 缓存
		expire = time.Duration(cache.Expire()) * time.Second
		_ = conn.GetRedis().Set(context.TODO(), conn.GetRedisKey(name), s, expire).Val()
		//fmt.Println(name, "成功更新缓存")
	}
	return nil
}

func (cache *Cache) Write(key string, cacheItem CacheItem) {
	cache.Lock()
	cache.Keys[key] = cacheItem
	cache.Unlock()
}

type updaterMap struct {
	sync.RWMutex
	Map map[string]*Cache
}

func (m *updaterMap) Read(key string) (*Cache, bool) {
	m.RLock()
	cache, ok := m.Map[key]
	m.RUnlock()
	return cache, ok
}

func (m *updaterMap) Write(key string, cache *Cache) {
	m.Lock()
	m.Map[key] = cache
	m.Unlock()
}

func (m *updaterMap) Update() {
	m.RLock()
	for _, cache := range m.Map {
		if err := cache.Update(); err != nil {
			_ = cache.Update()
		}
	}
	m.RUnlock()
}

func (m *updaterMap) UpdateNow(name string) error {
	m.RLock()
	var err error
	for n, cache := range CacheUpdater.Map {
		if n == name {
			err = cache.Update()
		}
	}
	m.RUnlock()
	return err
}

var CacheUpdater = &updaterMap{
	Map: make(map[string]*Cache),
}

// 注册后，自动维护缓存；如果长期无人访问的话，也可以取消维护了
func ResignCacheFromRedis[T any](key Key, duration int64, newFun func() (T, error), force ...bool) (ret T) {
	if _, ok := CacheUpdater.Read(key.Format); !ok {
		CacheUpdater.Write(key.Format, &Cache{
			Keys:     make(map[string]CacheItem),
			Duration: duration,
		})
	}

	// 记录下缓存的访问时间
	cache, _ := CacheUpdater.Read(key.Format)
	cache.Write(key.Name, CacheItem{
		Last: time.Now().Unix(),
		NewFun: func() (any, error) {
			return newFun()
		},
	})

	expire := cache.Expire()
	ret = conn.GetCacheFromRedis(key.Name, expire, newFun, force...)
	return
}
