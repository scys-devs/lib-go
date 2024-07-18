package conn

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/gin-gonic/gin"
	jsoniter "github.com/json-iterator/go"
	"github.com/scys-devs/lib-go"
)

var (
	esLogger = lib.GetLogger("es")
	esClient *elasticsearch.Client
)

func NewES(address []string, name string, pass string) {
	if client, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: address,
		Username:  name,
		Password:  pass,
	}); err == nil {
		esClient = client
	} else {
		panic(err)
	}
}

func GetES() *elasticsearch.Client {
	return esClient
}

type ESResult struct {
	ID        string              `json:"_id"`
	Source    jsoniter.RawMessage `json:"_source"`
	Highlight Highlight           `json:"highlight"` // es返回的属性
}

func (raw *ESResult) ToItem(item interface{}) {
	_ = jsoniter.Unmarshal(raw.Source, item)
}

type Highlight map[string][]string

func ESHandlerErr(res *esapi.Response, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	if res.IsError() {
		var e map[string]interface{}
		raw, _ := ioutil.ReadAll(res.Body)
		if err := jsoniter.Unmarshal(raw, &e); err != nil {
			esLogger.Errorf("productRepo.BulkInsert: Error decoding error response: %s\n", err)
		} else {
			esLogger.Errorf("productRepo.BulkInsert: Error response [%s] %s: %s",
				res.Status(),
				e["error"].(map[string]interface{})["type"],
				e["error"].(map[string]interface{})["reason"],
			)
		}
		return nil, err
	}

	defer res.Body.Close()
	return ioutil.ReadAll(res.Body)
}

var json = jsoniter.ConfigFastest

func ESSearch(index string, req map[string]interface{}) ([]byte, error) {
	es := GetES()
	body, _ := json.Marshal(req)
	return ESHandlerErr(es.Search(
		es.Search.WithContext(context.Background()),
		es.Search.WithIndex(index),
		es.Search.WithBody(bytes.NewReader(body)),
		es.Search.WithTrackTotalHits(true),
		//es.Search.WithPretty(),
	))
}

func ESDelete(index string, id string) ([]byte, error) {
	es := GetES()
	return ESHandlerErr(es.Delete(index, id))
}

func ESSearchToVal(index string, req map[string]interface{}) (ll []ESResult, total int) {
	raw, err := ESSearch(index, req)
	if err != nil {
		return
	} else if raw == nil {
		return
	}

	jsoniter.Get(raw, "hits", "hits").ToVal(&ll)
	total = jsoniter.Get(raw, "hits", "total", "value").ToInt()
	return
}

func ESCount(index string, req map[string]interface{}) ([]byte, error) {
	es := GetES()
	body, _ := json.Marshal(req)
	return ESHandlerErr(es.Count(
		es.Count.WithContext(context.Background()),
		es.Count.WithIndex(index),
		es.Count.WithBody(bytes.NewReader(body)),
	))
}

// 写入ES
func ESPut(index string, buf *bytes.Buffer) error {
	es := GetES()
	req := esapi.BulkRequest{
		Index:   index,
		Body:    buf,
		Refresh: "true",
	}
	_, err := ESHandlerErr(req.Do(context.Background(), es))
	return err
}

type ESBody struct {
	Body *bytes.Buffer
}

// 批量更新 https://www.elastic.co/guide/en/elasticsearch/reference/master/docs-bulk.html#bulk-update
func (item *ESBody) Write(id interface{}, data interface{}) {
	if id == nil {
		item.Body.WriteString("{\"index\":{}}\n")
		tmp, _ := jsoniter.MarshalToString(data)
		item.Body.WriteString(tmp + "\n")
	} else {
		item.Body.WriteString(fmt.Sprintf("{\"update\":{\"_id\":\"%v\"}}\n", fmt.Sprint(id)))
		tmp, _ := jsoniter.MarshalToString(map[string]interface{}{
			"doc":           data,
			"doc_as_upsert": true,
		})
		item.Body.WriteString(tmp + "\n")
	}
}

func NewESBody() *ESBody {
	return &ESBody{Body: bytes.NewBufferString("")}
}

// 分词
type ESToken struct {
	Token       string `json:"token"`
	StartOffset int    `json:"start_offset"`
	EndOffset   int    `json:"end_offset"`
	Type        string `json:"type"`
	Position    int    `json:"position"`
}

type ESAnalyzeResp struct {
	Tokens []ESToken `json:"tokens"`
}

func ESAnalyze(text string) ([]ESToken, error) {
	es := GetES()
	body, _ := json.Marshal(gin.H{
		"text":     text,
		"analyzer": "ik_smart",
	})
	resp, err := ESHandlerErr(es.Indices.Analyze(
		es.Indices.Analyze.WithBody(bytes.NewBuffer(body)),
	))
	if err != nil {
		return nil, err
	}
	token := new(ESAnalyzeResp)
	_ = jsoniter.Unmarshal(resp, token)
	return token.Tokens, err
}
