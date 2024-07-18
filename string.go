package lib

import (
	"math/rand"
	"strconv"
	"time"
	"unsafe"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

func GetRndString(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

func IsEmptyStr(s *string) bool {
	return s == nil || *s == ""
}

func Int64ToStr(i int64) string {
	return strconv.FormatInt(i, 10)
}

func IntToStr(i int) string {
	return strconv.Itoa(i)
}

func StrToInt64(s string) int64 {
	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}

func StrToInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func StrToFloat64(s string) float64 {
	i, _ := strconv.ParseFloat(s, 64)
	return i
}

func StrToBool(s string) bool {
	b, _ := strconv.ParseBool(s)
	return b
}

func StrToBytes(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}

func BytesToStr(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func BoolToStr(b bool) string {
	if b {
		return "是"
	} else {
		return "-"
	}
}

func Eclipse(content string, length int) string {
	tmp := []rune((content))
	if len(tmp) > length {
		return string(tmp[0:length]) + "....."
	}
	return content
}
