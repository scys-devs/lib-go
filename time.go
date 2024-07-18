package lib

import (
	"strings"
	"time"
)

const FormatDay = "2006-01-02"
const FormatTime = "2006-01-02 15:04:05"
const FormatISOTime = "2006-01-02T15:04:05+0800"
const FormatISOMilliTime = "2006-01-02T15:04:05.000Z07:00"
const FormatTimeCN = "01月02日15时04分"

const MinTime = "946656000" // 2000年，防止解析时间不正常

// DaysAfter n天后0点
// n=0 今天0点
// n=1 明天0点
func DaysAfter(n int64) int64 {
	return DaysAfterWith(time.Now().Unix(), n)
}

func DaysAfterWith(t, n int64) int64 {
	return (t+28800+n*86400)/86400*86400 - 28800
}

// NextDayWithOffset 明天几点，当然今天也可以
func NextDayWithOffset(offset int64) int64 {
	next := DaysAfter(1) + offset - time.Now().Unix()
	if next > 86400 {
		return DaysAfter(0) + offset - time.Now().Unix()
	}
	return next
}

func ParseTime(layout, value string) int64 {
	t, _ := time.ParseInLocation(layout, value, time.Local)
	if ts := t.Unix(); ts > 0 {
		return ts
	} else {
		return 0 // 非法时间，就认为是没有了
	}
}

// 格式化时间戳为指定格式
func FormatUnix(unix int64, layout string) string {
	if unix <= 0 {
		return ""
	}
	return time.Unix(unix, 0).Format(layout)
}

// 是否在工作日内
// TODO 增加节假日判断 https://www.iamwawa.cn/workingday.html
func InWorkWeek(t time.Time) bool {
	return !(t.Weekday() == time.Saturday || t.Weekday() == time.Sunday)
}

// 是否在工作时间
func InWorkHour(t time.Time) bool {
	h := t.Hour()
	return InWorkWeek(t) && h >= 9 && h < 20
}

type DateRange string

func (item DateRange) Start() int64 {
	if len(item) == 0 {
		return DaysAfter(0)
	}
	return StrToInt64(strings.Split(string(item), ",")[0])
}

func (item DateRange) End() int64 {
	if len(item) == 0 {
		return DaysAfter(0) + 86400 - 1
	}
	return StrToInt64(strings.Split(string(item), ",")[1])
}
