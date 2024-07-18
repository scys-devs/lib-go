package lib

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigFastest

func DoRequest(req *http.Request) (rb []byte, err error) {
	if req == nil {
		return nil, errors.New("no request")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		r, _ := gzip.NewReader(resp.Body)
		defer r.Close()

		return io.ReadAll(r)
	default:
		return io.ReadAll(resp.Body)
	}
}

func DoRequestJson(req *http.Request, v interface{}) (err error) {
	rb, err := DoRequest(req)
	if err != nil {
		return
	}
	if v == nil {
		return
	}
	err = json.Unmarshal(rb, v)
	if err != nil {
		err = fmt.Errorf("parse json err=%s, body=%s", err.Error(), string(rb))
	}
	return
}

func QueryValues(query url.Values) []string {
	val := make([]string, len(query))
	for key := range query {
		val = append(val, query[key][0])
	}
	return val
}

// 使用场景：需要重构url 加参数
func QueryAppend(raw string, extra map[string]string) string {
	u, _ := url.Parse(raw)
	q := u.Query()
	for key, value := range extra {
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func ParseCookieString(s string) []*http.Cookie {
	header := http.Header{}
	header.Add("Cookie", s)
	request := http.Request{Header: header}
	return request.Cookies()
}

// 使用于url中最后是参数ID的链接
func ParseParamURL(raw string, pos ...int) string {
	offset := 1
	if len(pos) > 0 {
		offset = pos[0]
	}
	u, _ := url.Parse(raw)
	if u == nil {
		return ""
	}
	pathLL := strings.Split(u.Path, "/")
	if len(pathLL) >= offset {
		return pathLL[len(pathLL)-offset]
	}
	return ""
}

// 获取域名第一部分
func ParseFirstHostUrl(raw string) string {
	u, _ := url.Parse(raw)
	if u == nil {
		return ""
	}
	hostLL := strings.Split(u.Host, ".")
	return hostLL[0]
}

// 获取302重定向的地址; 如果不是302的话，就返回空
func GetRedirectURL(raw string) string {
	var redirectCount = 0
	myRedirect := func(req *http.Request, via []*http.Request) (e error) {
		redirectCount++
		if redirectCount == 1 {
			redirectCount = 0
			return errors.New(req.URL.String())
		}
		return
	}

	client := &http.Client{CheckRedirect: myRedirect}
	_, err := client.Get(raw)
	if err != nil { // 报错了就是需要重定向了
		if e, ok := err.(*url.Error); ok && e.Err != nil {
			return e.URL
		}
	}
	return ""
}
