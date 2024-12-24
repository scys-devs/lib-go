package conn

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	jsoniter "github.com/json-iterator/go"
	"github.com/scys-devs/lib-go"
)

var redisClient *redis.Client

func NewRedis(host, port string) {
	if ENV == "local-docker" {
		host = HostDockerInternal
	}
	redisClient = redis.NewClient(&redis.Options{Addr: host + ":" + port})
	// 连通性测试
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second)
	defer cancel()
	if _, err := redisClient.Ping(ctx).Result(); err != nil {
		panic(err)
	}
}

func GetRedis() *redis.Client {
	return redisClient
}

// 精简版的接口
func K(_key ...string) string {
	return strings.Join(append([]string{PREFIX, "cache"}, _key...), ":")
}

func GetRedisKey(_key string) string {
	return K(_key)
}

// NOTE 应该上singleflight
func GetCacheFromRedis[T any](_key string, ttl int64, new func() (T, error), force ...bool) (ret T) {
	key := K(_key)

	if len(force) > 0 && force[0] {
		_ = redisClient.Del(context.TODO(), key).Err()
		ret, _ = new()
		return ret
	}

	v := redisClient.Get(context.TODO(), key).Val()
	if len(v) == 0 {
		var err error
		ret, err = new()

		if err != nil { // 报错了就不缓存了
			return ret
		}
		s, _ := jsoniter.MarshalToString(ret)
		if ttl == 86400 { // 尝试将缓存时间放到半夜
			ttl = lib.NextDayWithOffset(7200)
		}
		expire := time.Duration(ttl) * time.Second
		_ = redisClient.Set(context.TODO(), key, s, expire).Val()
	} else {
		_ = jsoniter.UnmarshalFromString(v, &ret)
	}
	return
}

// 每日巡检开关，有值就写入巡检，无值就返回是否巡检
// daemon状态不生效
func DayPatrol(key string, val ...interface{}) bool {
	if len(os.Getenv("DAEMON")) > 0 {
		return false
	}
	if len(val) == 0 {
		return len(GetRedis().Get(context.TODO(), GetRedisKey(key)).Val()) > 0
	} else {
		GetRedis().Set(context.TODO(), GetRedisKey(key), 1, time.Duration(lib.DaysAfter(1)-time.Now().Unix())*time.Second)
		return false
	}
}
