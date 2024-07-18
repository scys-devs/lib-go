package nacos

import (
	"errors"
	"fmt"
	"strings"

	"github.com/scys-devs/lib-go/conn"

	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"gopkg.in/yaml.v3"
)

var (
	nacos      config_client.IConfigClient
	nacosGroup string

	Port = ":8080"
)

func New(group, endpoint, namespace string) {
	nacosGroup = group

	// 由于免费的acm服务没了，所以去掉吧
	//if ENV == "local" {
	//	endpoint = "acm.aliyun.com"
	//	namespace = "00566c00-b74e-42ec-a922-35e4abb3980f"
	//}
	nacos, _ = clients.CreateConfigClient(map[string]interface{}{
		"clientConfig": constant.ClientConfig{
			Endpoint:    endpoint + Port,
			NamespaceId: namespace,
			AccessKey:   conn.AliyunID,
			SecretKey:   conn.AliyunSecret,
			LogDir:      "/tmp/nacos/log",
			CacheDir:    "/tmp/nacos/cache",
			TimeoutMs:   10000,
		},
	})
	if nacos == nil {
		panic("new nacos error")
	}
	return
}

// ParseConfigFromNacos
// 先从default中去读取基础配置，然后在读取特定配置进行覆盖
// FIXME 可能会出现默认值，忘记关的情况，所以最好是改为将椰子节点整个替换
func ParseConfigFromNacos(dataId string, listen bool, decode func(string) error) (err error) {
	content, _ := nacos.GetConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  nacosGroup,
	})
	if len(content) == 0 {
		return errors.New("common config not found")
	}
	if err = decode(content); err != nil {
		return err
	}

	group := nacosGroup
	// 获取开发环境进行覆盖
	if len(conn.ENV) > 0 {
		group = nacosGroup + "_" + conn.ENV

		content, _ = nacos.GetConfig(vo.ConfigParam{
			DataId: dataId,
			Group:  group,
		})
		if len(content) > 0 {
			if err = decode(content); err != nil {
				return err
			}
		} else { // 由于没有替换配置，所以后面尝试监听源文件
			group = nacosGroup
		}
	}

	if listen {
		err = nacos.ListenConfig(vo.ConfigParam{
			DataId: dataId,
			Group:  group,
			OnChange: func(_, _, _, content string) {
				if len(content) > 0 {
					if err2 := decode(content); err2 != nil {
						fmt.Println("parse config failed", err2)
					}
				}
			},
		})
	}
	return err
}

// ConfigComputed 设计一个配置自动判断器
// 强制设定，配置文件下后缀_computed下字段都为计算字段
// judge 为规则名，可以通过_进行多重规则的拼接
// value 为可选值
//
// --- example
//
//	 welcome:
//		  judge: lang
//	   value:
//	     en: i am good boy
//	     cn: 我是一个好人
type ConfigComputed struct {
	Context *gin.Context
	Rule    map[string]ConfigComputedRule
}

type ConfigComputedRule struct {
	Judge    string
	Value    map[string]interface{}
	GetValue func(*gin.Context) interface{}
}

func NewConfigComputed(context *gin.Context) *ConfigComputed {
	return &ConfigComputed{
		Context: context,
		Rule:    map[string]ConfigComputedRule{},
	}
}

func (config *ConfigComputed) Set(rule ConfigComputedRule) {
	config.Rule[rule.Judge] = rule
}

// ConfigComputeValue 计算配置
func (config *ConfigComputed) ConfigComputeValue(content []byte, json bool) (tmpValue map[string]interface{}, err error) {
	tmp := make(map[string]ConfigComputedRule)
	tmpValue = make(map[string]interface{})
	if json {
		if err = jsoniter.Unmarshal(content, &tmp); err != nil {
			return
		}
	} else {
		if err = yaml.Unmarshal(content, &tmp); err != nil {
			return
		}
	}

	for name, item := range tmp {
		if len(item.Judge) == 0 {
			tmpValue[name] = item.Value[""]
			continue
		}
		nameLL := make([]string, 0)
		for _, j := range strings.Split(item.Judge, "_") {
			if rule, ok := config.Rule[j]; ok {
				if val := rule.GetValue(config.Context); val != nil {
					nameLL = append(nameLL, fmt.Sprint(val))
					continue
				}
			}
			// FIXME 这边理论上还有点问题。
			nameLL = append(nameLL, "")
		}
		tmpValue[name] = item.Value[strings.Join(nameLL, "_")]
	}
	return
}
