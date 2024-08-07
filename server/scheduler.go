package server

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/nacos-group/nacos-sdk-go/v2/inner/uuid"
	"go.uber.org/zap"

	"github.com/go-redis/redis/v8"
	"github.com/scys-devs/lib-go"
	"github.com/scys-devs/lib-go/conn"
)

var log = lib.GetLogger("scheduler")

type Context struct {
	Logger *zap.SugaredLogger
}

type Executor interface {
	Name() string                     // 名字
	Desc() string                     // 中文描述
	NextDuration() int64              // 获取距离下次执行的休眠时间
	Process(ctx *Context) (err error) // 具体的执行函数
	Processing() string               // 进度描述，由程序自己控制
}

type ExecutorStatus struct {
	Status    bool           `json:"status"`     // 是否开启
	NextGmt   int64          `json:"next_gmt"`   // 下次执行时间
	LastGmt   int64          `json:"last_gmt"`   // 上次执行开始时间
	LastSpent string         `json:"last_spent"` // 上次执行花费时间，单位毫秒
	LastUUID  string         `json:"last_uuid"`  // 上次执行批次
	Crashed   int            `json:"crashed"`    // 是否崩溃了
	Once      chan time.Time `json:"-"`          // 提前执行
	// 额外字段
	Processing string `json:"processing"` // 当前状态，获取时再计算
	Name       string `json:"name"`       // 任务名称
	Desc       string `json:"desc"`       // 任务描述
}

var Scheduler = &scheduler{
	prefix:         conn.PREFIX,
	executorList:   nil,
	executorStatus: make(map[string]*ExecutorStatus),
}

type scheduler struct {
	prefix         string
	queue          *redis.Client              // 依赖redis的队列实现
	executorList   []Executor                 // 执行器队列
	executorStatus map[string]*ExecutorStatus // 通过配置文件控制，简单方便
	current        int64                      // 现在的时间戳，仅用于补数据，暂时仅支持命令行形式
}

// GetAll 获取注册任务状态; 根据prefix排序一下
// Executor的运行状态；status=false 未开启；crashed>10 已崩溃；now >= next 正在执行; 等待下次执行
func (s *scheduler) GetAll() (ll []*ExecutorStatus) {
	for _, ex := range s.executorList {
		status := s.executorStatus[ex.Name()]
		status.Name = ex.Name()
		status.Desc = ex.Desc()
		status.Processing = ex.Processing()
		ll = append(ll, status)
	}
	sort.SliceStable(ll, func(i, j int) bool {
		return strings.Compare(ll[i].Name, ll[j].Name) < 0
	})
	return
}

func (s *scheduler) Now() time.Time {
	if s.current == 0 {
		return time.Now()
	} else {
		return time.Unix(s.current, 0)
	}
}

func (s *scheduler) DaysAfter(n int64) int64 {
	return (s.Now().Unix()+28800+n*86400)/86400*86400 - 28800
}

func (s *scheduler) NextDayWithOffset(offset int64) int64 {
	next := s.DaysAfter(1) + offset - s.Now().Unix()
	if next > 86400 {
		return s.DaysAfter(0) + offset - s.Now().Unix()
	}
	return next
}

// Register 添加待执行的任务, prod环境默认执行
func (s *scheduler) Register(ex Executor) {
	s.executorList = append(s.executorList, ex)
	s.executorStatus[ex.Name()] = &ExecutorStatus{
		Status: len(conn.ENV) == 0,
		Once:   make(chan time.Time),
	}
}

// RegisterStatus 覆盖执行状态
func (s *scheduler) RegisterStatus(status map[string]bool) {
	for name, val := range status {
		if _, ok := s.executorStatus[name]; ok {
			s.executorStatus[name].Status = val
		}
	}
}

// Once 手动执行一次；如果任务正在执行的话，那么提醒用户等会再尝试把
func (s *scheduler) Once(name string) bool {
	status, ok := s.executorStatus[name]
	if ok {
		select {
		case status.Once <- time.Now():
			return true
		default:
			return false
		}
	} else {
		return false
	}
}

// Start 开始创建后台执行的任务
func (s *scheduler) Start() {
	if s.queue == nil {
		s.queue = conn.GetRedis()
	}

	s.runForCMD()

	for _, ex := range s.executorList {
		status := s.executorStatus[ex.Name()]
		if status.Status {
			fmt.Printf("daemon register %s, status=%v\n", ex.Name(), status)
			go s.run(ex)
		}
	}
}

// 从命令行临时启动，检查是否存在环境变量
// 格式 DAEMON=executor [START=20060102] [END=20060102] [SLEEP=10]
func (s *scheduler) runForCMD() {
	daemon := os.Getenv("DAEMON")
	if len(daemon) == 0 {
		return
	}
	sleep := lib.StrToInt64(os.Getenv("SLEEP"))
	time.Sleep(time.Duration(sleep) * time.Second)

	start := lib.ParseTime("20060102", os.Getenv("START"))
	end := lib.ParseTime("20060102", os.Getenv("END"))
	if end == 0 {
		end = start
	}
	if start == 0 {
		start = lib.DaysAfter(0)
		end = start
	}
	name := os.Getenv("DAEMON")
	for ; start <= end; start += 86400 {
		s.current = start
		fmt.Println(s.Now().Format("20060102"), "start")
		var err = errors.New("not found")
		for _, ex := range s.executorList {
			if ex.Name() == name {
				c := NewSchedulerContext(ex.Name())
				err = ex.Process(c)
			}
		}
		fmt.Println(s.Now().Format("20060102"), "complete", err)
	}
	os.Exit(0)
}

func NewSchedulerContext(name string) *Context {
	uid, _ := uuid.NewV4()
	return &Context{Logger: log.With("name", name, "uuid", uid.String())}
}

func (s *scheduler) run(ex Executor) {
	for {
		uid, _ := uuid.NewV4()
		c := &Context{Logger: log.With("name", ex.Name(), "uuid", uid.String())}
		status := s.executorStatus[ex.Name()]
		// 判断什么时候执行
		next := time.Duration(ex.NextDuration())
		showStart := true
		if next < 0 {
			break
		}
		if next == 0 {
			next = 400 * time.Millisecond
			showStart = false
		} else {
			status.NextGmt = time.Now().Unix() + int64(next)
			next = next * time.Second
			c.Logger.Infow("daemon waiting", "at", time.Now().Add(next).Format("2006-01-02 15:04:05"))
		}
		// 允许提前执行
		select {
		case <-status.Once:
			break
		case <-time.After(next):
			break
		}
		if showStart {
			c.Logger.Infow("daemon start")
		}
		status.LastUUID = uid.String() // 开始执行才修改，日志可以查看上一次的

		gmtStart := time.Now().UnixNano()
		if err := func() (err error) { // 使用闭包开始执行
			defer func() {
				if r := recover(); r != nil {
					fmt.Println("daemon run panic, name", ex.Name(), r)
					debug.PrintStack()

					err = errors.New("panic")
				}
			}()
			return ex.Process(c)
		}(); err != nil {
			status.Crashed += 1
			if status.Crashed > 10 {
				c.Logger.Errorw("daemon stop")
				return
			}
		}

		if next > 0 {
			used := float64(time.Now().UnixNano()-gmtStart) / 1e9
			status.LastGmt = gmtStart / 1e9
			status.LastSpent = fmt.Sprintf("%.4f", used)
			if showStart {
				c.Logger.Infow("daemon complete", "used", status.LastSpent)
			}
		}
	}
}

func (s *scheduler) key(name string) string {
	return fmt.Sprintf("%v:%v", s.prefix, name)
}

// Add 添加一个待执行的任务数据包
func (s *scheduler) Add(name, member string, after int64) {
	gmtExpire := float64(time.Now().Unix() + after)
	//s.Logger.Infow("add delay", "name", name, "member", member, "expire", gmtExpire)
	if err := s.queue.ZAdd(context.Background(), s.key(name), &redis.Z{Score: gmtExpire, Member: member}).Err(); err != nil {
		log.Errorw("put member failed", "name", name, "member", member, "after", after)
	}
}

const min = "-inf"

// Len 估算消息队列长度
func (s *scheduler) Len(name string) (cnt int64) {
	cnt, _ = s.queue.ZCard(context.Background(), s.key(name)).Result()
	return
}

// GetBatch 获取到执行任务时所有的任务，通过queue传出去
func (s *scheduler) GetBatch(name, max string, handler func(item redis.Z)) {
	var count int64 = 2000
	var offset int64 = 0
	defer func() {
		if offset > 0 {
			s.clearBatch(name, max)
		}
	}()
	for {
		batch, err := s.queue.ZRangeByScoreWithScores(context.Background(), s.key(name), &redis.ZRangeBy{
			Min:    min,
			Max:    max,
			Offset: offset,
			Count:  count,
		}).Result()
		if err != nil {
			log.Errorw("get batch failed", "name", name, "err", err)
			return
		}
		batchLen := int64(len(batch))
		for _, item := range batch {
			handler(item)
		}
		// next
		offset += batchLen
		if batchLen < count {
			return
		}
	}
}

// clearBatch 清除已经获取的批次
func (s *scheduler) clearBatch(name, max string) {
	cnt, _ := s.queue.ZRemRangeByScore(context.Background(), s.key(name), min, max).Result()
	log.Infow("clear batch success", "name", name, "cnt", cnt)
}
