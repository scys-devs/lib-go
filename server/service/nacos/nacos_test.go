package nacos

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
	"testing"
)

var s = `
key:
  sub_key_0:
    value0: 1
    value1: 0
    value2: 0
`

var st = `
key:
  sub_key_0:
    value1: 1
    value2: 1
`

type TestS struct {
	Key struct {
		SubKey0 struct {
			Value0 int `yaml:"value0"`
			Value1 int `yaml:"value1"`
			Value2 int `yaml:"value2"`
		} `yaml:"sub_key_0"`
	} `yaml:"key"`
}

func TestParseConfigFromNacos(t *testing.T) {
	var target = new(map[string]interface{})
	yaml.NewDecoder(bytes.NewBufferString(s)).Decode(target)
	yaml.NewDecoder(bytes.NewBufferString(st)).Decode(target)
	fmt.Println(target)
}

var hermit = `
welcome:
  judge: lang
  value:
    en: i am good boy
    cn: 我是一个好人
welcome2:
  judge: user_lang
  value:
    en: i am good boy
    cn: 我是一个好人
`

func TestConfigRule(t *testing.T) {
	// 设置configRule
	c := &gin.Context{}
	config := NewConfigComputed(c)
	config.Set(ConfigComputedRule{
		Judge: "lang",
		GetValue: func(_ *gin.Context) interface{} {
			return nil
		},
	})
	fmt.Println(config.ConfigComputeValue([]byte(hermit), false))
}
